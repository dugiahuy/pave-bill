package bill

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
)

// GetBill handles the business logic for retrieving a bill by ID with line items
func (b *business) GetBill(ctx context.Context, id int32) (*model.Bill, error) {
	dbBill, err := b.billRepo.GetBill(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "bill not found"}
		}
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to get bill"}
	}

	bill := convertDBBillToModel(dbBill)

	lineItems, err := b.GetLineItemsByBill(ctx, id)
	if err != nil {
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to get line items"}
	}
	bill.LineItems = lineItems

	return bill, nil
}
