package domain

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
)

// StateMachine defines the interface for bill state transitions and transaction management
type StateMachine interface {
	// GetBillWithLock performs any operation with proper row-level locking and transaction management
	GetBillWithLock(ctx context.Context, billID int32, businessLogic func(bills.Bill) error) error

	// GetCurrentTx returns the current transaction for use with other repositories
	GetCurrentTx() pgx.Tx

	// State transition methods
	TransitionToActive(ctx context.Context, id int32) error
	TransitionToClosingTx(ctx context.Context, id int32, reason string) error
	TransitionToClosedTx(ctx context.Context, id int32, reason string) error
	TransitionToFailureStateTx(ctx context.Context, id int32, errorMessage string) error

	// UpdateBillTotalTx recalculates bill total within transaction
	UpdateBillTotalTx(ctx context.Context, id int32) error

	// GetTxBillRepo returns transaction-aware bill repository
	GetTxBillRepo() bills.Querier

	// GetTxLineItemRepo returns transaction-aware line item repository
	GetTxLineItemRepo() lineitems.Querier
}

// BillStateMachine handles all bill state transitions and complex domain operations
// Following DDD principles: owns transaction boundaries and repository access
type BillStateMachine struct {
	db           *pgxpool.Pool
	billRepo     bills.Querier
	lineItemRepo lineitems.Querier

	// Transaction-aware repositories
	billTx     bills.Querier
	lineItemTx lineitems.Querier
	currentTx  pgx.Tx
}

// NewBillStateMachine creates a new bill state machine with database and repository access
func NewBillStateMachine(db *pgxpool.Pool, billRepo bills.Querier, lineItemRepo lineitems.Querier) *BillStateMachine {
	return &BillStateMachine{
		db:           db,
		billRepo:     billRepo,
		lineItemRepo: lineItemRepo,
	}
}

// GetTxBillRepo returns transaction-aware bill repository
func (sm *BillStateMachine) GetTxBillRepo() bills.Querier {
	return sm.billTx
}

// GetTxLineItemRepo returns transaction-aware line item repository
func (sm *BillStateMachine) GetTxLineItemRepo() lineitems.Querier {
	return sm.lineItemTx
}

// GetCurrentTx returns the current transaction for use with other repositories
func (sm *BillStateMachine) GetCurrentTx() pgx.Tx {
	return sm.currentTx
}

// transitionWithLock performs a state transition with proper row-level locking and transaction management
func (sm *BillStateMachine) transitionWithLock(ctx context.Context, id int32, transitionFunc func(bills.Bill) error) error {
	tx, err := sm.db.Begin(ctx)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to start transaction"}
	}
	defer tx.Rollback(ctx)

	sm.currentTx = tx
	sm.billTx = sm.billRepo.(*bills.Queries).WithTx(tx)
	sm.lineItemTx = sm.lineItemRepo.(*lineitems.Queries).WithTx(tx)

	currentBill, err := sm.billTx.GetBillForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &errs.Error{Code: errs.NotFound, Message: "bill not found"}
		}
		return &errs.Error{Code: errs.Internal, Message: "failed to lock bill for state transition"}
	}

	err = transitionFunc(currentBill)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to commit state transition"}
	}

	return nil
}

// TransitionToActive updates bill status to active with row locking
func (sm *BillStateMachine) TransitionToActive(ctx context.Context, id int32) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		if currentBill.Status != string(model.BillStatusPending) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill must be in pending status to transition to active",
			}
		}

		_, err := sm.billTx.UpdateBillStatus(ctx, bills.UpdateBillStatusParams{
			ID:     id,
			Status: string(model.BillStatusActive),
		})
		return err
	})
}

// TransitionToClosing updates bill status to closing with close reason and row locking
func (sm *BillStateMachine) TransitionToClosingTx(ctx context.Context, id int32, reason string) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		if currentBill.Status != string(model.BillStatusActive) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill must be in active status to transition to closing",
			}
		}

		_, err := sm.billTx.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
			ID:           id,
			Status:       string(model.BillStatusClosing),
			CloseReason:  pgtype.Text{String: reason, Valid: true},
			ErrorMessage: pgtype.Text{Valid: false},
		})
		return err
	})
}

// TransitionToClosed updates bill status to closed with close reason and row locking
func (sm *BillStateMachine) TransitionToClosedTx(ctx context.Context, id int32, reason string) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		// Validate current state - pending or closing can go to closed
		if currentBill.Status != string(model.BillStatusPending) && currentBill.Status != string(model.BillStatusClosing) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill must be in pending or closing status to transition to closed",
			}
		}

		_, err := sm.billTx.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
			ID:           id,
			Status:       string(model.BillStatusClosed),
			CloseReason:  pgtype.Text{String: reason, Valid: true},
			ErrorMessage: pgtype.Text{Valid: false},
		})
		return err
	})
}

// TransitionToFailureState updates bill to failed or attention_required with error details and row locking
func (sm *BillStateMachine) TransitionToFailureStateTx(ctx context.Context, id int32, errorMessage string) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		// Can transition to failure state from any non-terminal state
		if currentBill.Status != string(model.BillStatusClosed) ||
			currentBill.Status == string(model.BillStatusAttentionRequired) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is already in terminal status",
			}
		}

		_, err := sm.billTx.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
			ID:           id,
			Status:       string(model.BillStatusAttentionRequired),
			CloseReason:  pgtype.Text{Valid: false},
			ErrorMessage: pgtype.Text{String: errorMessage, Valid: true},
		})
		return err
	})
}

// GetBillWithLock performs any operation with proper row-level locking and transaction management
// This is the main callback pattern method that allows business logic to be executed within a transaction
func (sm *BillStateMachine) GetBillWithLock(ctx context.Context, id int32, businessLogic func(bills.Bill) error) error {
	return sm.transitionWithLock(ctx, id, businessLogic)
}

// UpdateBillTotalTx recalculates bill total within transaction
func (sm *BillStateMachine) UpdateBillTotalTx(ctx context.Context, id int32) error {
	_, err := sm.billTx.UpdateBillTotal(ctx, pgtype.Int4{Int32: id, Valid: true})
	return err
}
