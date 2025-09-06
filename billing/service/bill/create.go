package bill

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"
	"encore.dev/rlog"

	"encore.app/billing/model"
	"encore.app/billing/store/bills"
)

// Create handles the business logic for creating a new bill with explicit idempotency
func (s *service) Create(ctx context.Context, bill *model.Bill) (*model.Bill, error) {
	rlog.Info("Creating bill in service1", "bill", bill)

	r := bills.CreateBillParams{
		Status:         string(model.BillStatusPending),
		Currency:       bill.Currency,
		StartTime:      pgtype.Timestamptz{Time: bill.StartTime, Valid: true},
		EndTime:        pgtype.Timestamptz{Time: bill.EndTime, Valid: true},
		IdempotencyKey: bill.IdempotencyKey,
	}
	rlog.Info("Creating bill in service2", "params", r)

	dbBill, err := s.billRepo.CreateBill(ctx, r)
	if err != nil {
		return nil, &errs.Error{
			Code:    errs.Internal,
			Message: "failed to create bill",
		}
	}

	return &model.Bill{
		ID:             dbBill.ID,
		Currency:       dbBill.Currency,
		Status:         model.BillStatus(dbBill.Status),
		StartTime:      dbBill.StartTime.Time,
		EndTime:        dbBill.EndTime.Time,
		IdempotencyKey: dbBill.IdempotencyKey,
		CreatedAt:      dbBill.CreatedAt.Time,
		UpdatedAt:      dbBill.UpdatedAt.Time,
	}, nil
}
