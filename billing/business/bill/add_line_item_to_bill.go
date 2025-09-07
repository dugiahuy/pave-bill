package bill

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"encore.dev/beta/errs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
)

// AddLineItemToBill adds a line item to a bill with proper row locking to prevent race conditions
// This method coordinates with CloseBill to ensure atomicity and consistency
func (b *business) AddLineItemToBill(ctx context.Context, billID int32, lineItem *model.LineItem) (*model.LineItem, error) {
	var result *model.LineItem
	var resultErr error

	err := b.stateMachine.ExecuteWithLock(ctx, billID, func(currentBill bills.Bill) error {
		switch currentBill.Status {
		case string(model.BillStatusActive):
			// Create line item within the same transaction that holds the bill lock
			txLineItemRepo := b.lineItemRepo.WithTx(b.stateMachine.GetCurrentTx())
			result, resultErr = b.createLineItemTx(ctx, billID, lineItem, currentBill, txLineItemRepo)
			return resultErr

		case string(model.BillStatusClosing):
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is being closed, cannot add line items",
			}

		case string(model.BillStatusClosed):
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is closed, cannot add line items",
			}

		case string(model.BillStatusPending):
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is not active yet, cannot add line items",
			}

		case string(model.BillStatusAttentionRequired):
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is in error state, cannot add line items",
			}

		default:
			return &errs.Error{
				Code:    errs.InvalidArgument,
				Message: "bill is not in valid state for line items",
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// createLineItem handles the complete line item creation process
// This runs within the bill's row lock transaction
func (b *business) createLineItem(ctx context.Context, billID int32, lineItem *model.LineItem, currentBill bills.Bill) (*model.LineItem, error) {
	// Validate bill currency and convert line item amount if needed
	conversion, err := b.currencyService.ConvertAmount(ctx, lineItem.Currency, currentBill.Currency, lineItem.AmountCents)
	if err != nil {
		return nil, err
	}

	// Prepare metadata for currency conversion
	var metadataJSON []byte
	if conversion.Metadata != nil {
		metadataJSON, err = json.Marshal(conversion.Metadata)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to marshal metadata"}
		}
	}

	// Set default incurred time if not provided
	if lineItem.IncurredAt.IsZero() {
		lineItem.IncurredAt = time.Now()
	}

	// Create the line item with converted amount and bill currency
	dbLineItem, err := b.lineItemRepo.CreateLineItem(ctx, lineitems.CreateLineItemParams{
		BillID:         pgtype.Int4{Int32: billID, Valid: true},
		AmountCents:    conversion.ConvertedAmount,
		Currency:       currentBill.Currency,
		Description:    pgtype.Text{String: lineItem.Description, Valid: true},
		IncurredAt:     pgtype.Timestamptz{Time: lineItem.IncurredAt, Valid: true},
		ReferenceID:    pgtype.Text{String: lineItem.ReferenceID, Valid: true},
		Metadata:       metadataJSON,
		IdempotencyKey: lineItem.IdempotencyKey,
	})
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			return nil, &errs.Error{Code: errs.AlreadyExists, Message: "line item already exists"}
		}
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to create line item"}
	}

	result := convertDBLineItemToModel(dbLineItem)
	result.SetBillWorkflowID(currentBill.WorkflowID.String)

	return result, nil
}
