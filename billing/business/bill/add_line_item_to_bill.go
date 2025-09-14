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

	err := b.stateMachine.GetBillWithLock(ctx, billID, func(currentBill bills.Bill) error {
		if currentBill.Status != string(model.BillStatusActive) {
			return &errs.Error{Code: errs.InvalidArgument, Message: "bill is not in active state for adding line items"}
		}

		conversion, err := b.currencyService.ConvertAmount(ctx, lineItem.Currency, currentBill.Currency, lineItem.AmountCents)
		if err != nil {
			return err
		}

		var metadataJSON []byte
		if conversion.Metadata != nil {
			metadataJSON, err = json.Marshal(conversion.Metadata)
			if err != nil {
				return &errs.Error{Code: errs.Internal, Message: "failed to marshal metadata"}
			}
		}

		if lineItem.IncurredAt.IsZero() {
			lineItem.IncurredAt = time.Now()
		}

		dbLineItem, err := b.stateMachine.GetTxLineItemRepo().CreateLineItem(ctx, lineitems.CreateLineItemParams{
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
				return &errs.Error{Code: errs.AlreadyExists, Message: "line item already exists"}
			}
			return &errs.Error{Code: errs.Internal, Message: "failed to create line item"}
		}

		result = convertDBLineItemToModel(dbLineItem)
		result.SetBillWorkflowID(currentBill.WorkflowID.String)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
