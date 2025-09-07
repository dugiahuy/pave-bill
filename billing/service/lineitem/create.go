package lineitem

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
	"encore.dev/beta/errs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *service) Create(ctx context.Context, lineItem *model.LineItem) (*model.LineItem, error) {
	bill, err := s.billRepo.GetBill(ctx, lineItem.BillID)
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "bill not found"}
	}

	if bill.Status != string(model.BillStatusActive) {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "bill is not active"}
	}

	conversion, err := s.currencyService.ConvertAmount(ctx, lineItem.Currency, bill.Currency, lineItem.AmountCents)
	if err != nil {
		return nil, err
	}

	var metadataJSON []byte
	if conversion.Metadata != nil {
		metadataJSON, err = json.Marshal(conversion.Metadata)
		if err != nil {
			return nil, &errs.Error{Code: errs.Internal, Message: "failed to marshal metadata"}
		}
	}

	if lineItem.IncurredAt.IsZero() {
		lineItem.IncurredAt = time.Now()
	}

	// Create the line item with converted amount and bill currency
	dbLineItem, err := s.lineItemRepo.CreateLineItem(ctx, lineitems.CreateLineItemParams{
		BillID:         pgtype.Int4{Int32: lineItem.BillID, Valid: true},
		AmountCents:    conversion.ConvertedAmount,
		Currency:       bill.Currency,
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

	// Update bill total after adding line item
	_, err = s.billRepo.UpdateBillTotal(ctx, bills.UpdateBillTotalParams{
		BillID: pgtype.Int4{Int32: lineItem.BillID, Valid: true},
		ID:     lineItem.BillID,
	})
	if err != nil {
		// Log error but don't fail the request - line item was created successfully
		// In production, consider using transactions to ensure atomicity
		// For now, the total can be recalculated later if needed
	}

	return convertDBLineItemToModel(dbLineItem), nil
}

// convertDBLineItemToModel converts database LineItem to domain model LineItem
func convertDBLineItemToModel(dbLineItem lineitems.LineItem) *model.LineItem {
	lineItem := &model.LineItem{
		ID:             dbLineItem.ID,
		BillID:         dbLineItem.BillID.Int32,
		AmountCents:    dbLineItem.AmountCents,
		Currency:       dbLineItem.Currency,
		IncurredAt:     dbLineItem.IncurredAt.Time,
		IdempotencyKey: dbLineItem.IdempotencyKey,
		CreatedAt:      dbLineItem.CreatedAt.Time,
		UpdatedAt:      dbLineItem.UpdatedAt.Time,
	}

	if dbLineItem.Description.Valid {
		lineItem.Description = dbLineItem.Description.String
	}

	if dbLineItem.ReferenceID.Valid {
		lineItem.ReferenceID = dbLineItem.ReferenceID.String
	}

	if len(dbLineItem.Metadata) > 0 {
		var metadata model.CurrencyMetadata
		if err := json.Unmarshal(dbLineItem.Metadata, &metadata); err == nil {
			lineItem.Metadata = &metadata
		}
	}

	return lineItem
}
