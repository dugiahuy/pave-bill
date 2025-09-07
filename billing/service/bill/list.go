package bill

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

// List handles the business logic for retrieving bills with pagination
func (s *service) List(ctx context.Context, limit, offset int32) ([]*model.Bill, int64, error) {
	dbBills, err := s.billRepo.ListBills(ctx, bills.ListBillsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, &errs.Error{Code: errs.NotFound, Message: "bills not found"}
		}
		return nil, 0, &errs.Error{Code: errs.Internal, Message: "failed to list bills"}
	}

	totalCount, err := s.billRepo.CountBills(ctx)
	if err != nil {
		return nil, 0, &errs.Error{Code: errs.Internal, Message: "failed to count bills"}
	}

	billList := make([]*model.Bill, len(dbBills))
	for i, dbBill := range dbBills {
		billList[i] = convertDBBillToModel(dbBill)
	}

	return billList, totalCount, nil
}
