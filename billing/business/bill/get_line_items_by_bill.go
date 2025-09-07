package bill

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/lineitems"
)

// GetLineItemsByBill retrieves all line items for a given bill ID
func (b *business) GetLineItemsByBill(ctx context.Context, billID int32) ([]model.LineItem, error) {
	dbLineItems, err := b.lineItemRepo.GetLineItemsByBill(ctx, pgtype.Int4{Int32: billID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []model.LineItem{}, nil
		}
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to get line items"}
	}

	lineItems := make([]model.LineItem, len(dbLineItems))
	for i, dbLineItem := range dbLineItems {
		lineItems[i] = *convertDBLineItemToModel(dbLineItem)
	}

	return lineItems, nil
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
