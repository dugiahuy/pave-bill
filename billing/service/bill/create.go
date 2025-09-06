package bill

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/store/bills"
)

// Create handles the business logic for creating a new bill with explicit idempotency
func (s *service) Create(ctx context.Context, bill *model.Bill) (*model.Bill, error) {
	dbBill, err := s.billRepo.CreateBill(ctx, bills.CreateBillParams{
		Status:         string(model.BillStatusPending),
		Currency:       bill.Currency,
		StartTime:      pgtype.Timestamptz{Time: bill.StartTime, Valid: true},
		EndTime:        pgtype.Timestamptz{Time: bill.EndTime, Valid: true},
		IdempotencyKey: bill.IdempotencyKey,
	})
	if err != nil {
		var e *pgconn.PgError
		if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
			return nil, &errs.Error{
				Code:    errs.AlreadyExists,
				Message: "bill is duplicated",
			}
		}

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
