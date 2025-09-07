package bill

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
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
			return nil, &errs.Error{Code: errs.AlreadyExists, Message: "bill is duplicated"}
		}

		return nil, &errs.Error{Code: errs.Internal, Message: "failed to create bill"}
	}

	return convertDBBillToModel(dbBill), nil
}

// convertDBBillToModel converts a database Bill to a domain model Bill
func convertDBBillToModel(dbBill bills.Bill) *model.Bill {
	bill := &model.Bill{
		ID:               dbBill.ID,
		Currency:         dbBill.Currency,
		Status:           model.BillStatus(dbBill.Status),
		TotalAmountCents: dbBill.TotalAmountCents.Int64,
		StartTime:        dbBill.StartTime.Time,
		EndTime:          dbBill.EndTime.Time,
		IdempotencyKey:   dbBill.IdempotencyKey,
		CreatedAt:        dbBill.CreatedAt.Time,
		UpdatedAt:        dbBill.UpdatedAt.Time,
	}

	if dbBill.CloseReason.Valid {
		bill.CloseReason = &dbBill.CloseReason.String
	}

	if dbBill.ErrorMessage.Valid {
		bill.ErrorMessage = &dbBill.ErrorMessage.String
	}

	if dbBill.BilledAt.Valid {
		bill.BilledAt = &dbBill.BilledAt.Time
	}

	return bill
}
