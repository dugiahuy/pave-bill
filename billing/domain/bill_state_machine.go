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
)

// BillStateMachine handles all bill state transitions and complex domain operations
// Following DDD principles: owns transaction boundaries and repository access
type BillStateMachine struct {
	db        *pgxpool.Pool
	billRepo  *bills.Queries
	txQueries *bills.Queries // Transaction-aware queries, set during transitions
}

// NewBillStateMachine creates a new bill state machine with database and repository access
func NewBillStateMachine(db *pgxpool.Pool, billRepo *bills.Queries) *BillStateMachine {
	return &BillStateMachine{
		db:       db,
		billRepo: billRepo,
	}
}

// transitionWithLock performs a state transition with proper row-level locking and transaction management
func (sm *BillStateMachine) transitionWithLock(ctx context.Context, id int32, transitionFunc func(bills.Bill) error) error {
	// Start transaction - Domain service owns transaction boundary
	tx, err := sm.db.Begin(ctx)
	if err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to start transaction"}
	}
	defer tx.Rollback(ctx)

	// Create transaction-aware queries - store temporarily for use in transition
	sm.txQueries = sm.billRepo.WithTx(tx)

	// Use sqlc-generated GetBillForUpdate which uses SELECT FOR UPDATE
	currentBill, err := sm.txQueries.GetBillForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &errs.Error{Code: errs.NotFound, Message: "bill not found"}
		}
		return &errs.Error{Code: errs.Internal, Message: "failed to lock bill for state transition"}
	}

	// Execute the state transition with validation
	// The row is now locked until transaction commits/rollbacks
	err = transitionFunc(currentBill)
	if err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return &errs.Error{Code: errs.Internal, Message: "failed to commit state transition"}
	}

	return nil
}

// TransitionToActive updates bill status to active with row locking
func (sm *BillStateMachine) TransitionToActive(ctx context.Context, id int32) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		// Validate current state - only pending can go to active
		if currentBill.Status != string(model.BillStatusPending) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill must be in pending status to transition to active",
			}
		}

		_, err := sm.txQueries.UpdateBillStatus(ctx, bills.UpdateBillStatusParams{
			ID:     id,
			Status: string(model.BillStatusActive),
		})
		return err
	})
}

// TransitionToClosing updates bill status to closing with close reason and row locking
func (sm *BillStateMachine) TransitionToClosing(ctx context.Context, id int32, reason string) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		// Validate current state - only active can go to closing
		if currentBill.Status != string(model.BillStatusActive) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill must be in active status to transition to closing",
			}
		}

		_, err := sm.txQueries.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
			ID:           id,
			Status:       string(model.BillStatusClosing),
			CloseReason:  pgtype.Text{String: reason, Valid: true},
			ErrorMessage: pgtype.Text{Valid: false},
		})
		return err
	})
}

// TransitionToClosed updates bill status to closed with close reason and row locking
func (sm *BillStateMachine) TransitionToClosed(ctx context.Context, id int32, reason string) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		// Validate current state - only closing can go to closed
		if currentBill.Status != string(model.BillStatusClosing) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill must be in closing status to transition to closed",
			}
		}

		_, err := sm.txQueries.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
			ID:           id,
			Status:       string(model.BillStatusClosed),
			CloseReason:  pgtype.Text{String: reason, Valid: true},
			ErrorMessage: pgtype.Text{Valid: false},
		})
		return err
	})
}

// TransitionToFailureState updates bill to failed or attention_required with error details and row locking
func (sm *BillStateMachine) TransitionToFailureState(ctx context.Context, id int32, errorMessage string) error {
	return sm.transitionWithLock(ctx, id, func(currentBill bills.Bill) error {
		// Can transition to failure state from any non-terminal state
		if currentBill.Status != string(model.BillStatusClosed) ||
			currentBill.Status == string(model.BillStatusAttentionRequired) {
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is already in terminal status",
			}
		}

		_, err := sm.txQueries.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
			ID:           id,
			Status:       string(model.BillStatusAttentionRequired),
			CloseReason:  pgtype.Text{Valid: false},
			ErrorMessage: pgtype.Text{String: errorMessage, Valid: true},
		})
		return err
	})
}

// ExecuteWithLock performs any operation with proper row-level locking and transaction management
// This is the main callback pattern method that allows business logic to be executed within a transaction
func (sm *BillStateMachine) ExecuteWithLock(ctx context.Context, id int32, businessLogic func(bills.Bill) error) error {
	return sm.transitionWithLock(ctx, id, businessLogic)
}

// Helper methods for use within ExecuteWithLock callbacks
// These methods use the current transaction context (sm.txQueries)

// TransitionToClosedTx transitions bill to closed status (for use within callbacks)
func (sm *BillStateMachine) TransitionToClosedTx(ctx context.Context, id int32, reason string) error {
	_, err := sm.txQueries.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
		ID:           id,
		Status:       string(model.BillStatusClosed),
		CloseReason:  pgtype.Text{String: reason, Valid: true},
		ErrorMessage: pgtype.Text{Valid: false},
	})
	return err
}

// TransitionToClosingTx transitions bill to closing status (for use within callbacks)
func (sm *BillStateMachine) TransitionToClosingTx(ctx context.Context, id int32, reason string) error {
	_, err := sm.txQueries.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
		ID:           id,
		Status:       string(model.BillStatusClosing),
		CloseReason:  pgtype.Text{String: reason, Valid: true},
		ErrorMessage: pgtype.Text{Valid: false},
	})
	return err
}

// TransitionToFailureStateTx transitions bill to failure state (for use within callbacks)
func (sm *BillStateMachine) TransitionToFailureStateTx(ctx context.Context, id int32, errorMessage string) error {
	_, err := sm.txQueries.UpdateBillClosure(ctx, bills.UpdateBillClosureParams{
		ID:           id,
		Status:       string(model.BillStatusAttentionRequired),
		CloseReason:  pgtype.Text{Valid: false},
		ErrorMessage: pgtype.Text{String: errorMessage, Valid: true},
	})
	return err
}

// UpdateBillTotalTx recalculates bill total within transaction (for use within callbacks)
func (sm *BillStateMachine) UpdateBillTotalTx(ctx context.Context, id int32) error {
	_, err := sm.txQueries.UpdateBillTotal(ctx, pgtype.Int4{Int32: id, Valid: true})
	return err
}

// GetTransactionQueries returns the current transaction-aware queries
// This allows business logic to use the same transaction for related operations
func (sm *BillStateMachine) GetTransactionQueries() *bills.Queries {
	return sm.txQueries
}

// GetCurrentTx returns the current transaction for use with other repositories
func (sm *BillStateMachine) GetCurrentTx() pgx.Tx {
	return sm.currentTx
}
