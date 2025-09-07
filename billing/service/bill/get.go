package bill

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
)

// Get handles the business logic for retrieving a bill by ID
func (s *service) Get(ctx context.Context, id int32) (*model.Bill, error) {
	dbBill, err := s.billRepo.GetBill(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errs.Error{Code: errs.NotFound, Message: "bill not found"}
		}
		return nil, &errs.Error{Code: errs.Internal, Message: "failed to get bill"}
	}

	return convertDBBillToModel(dbBill), nil
}
