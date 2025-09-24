package bill

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

// CreateBill handles the business logic for creating a new bill with explicit idempotency
func (b *business) CreateBill(ctx context.Context, bill *model.Bill) (*model.Bill, error) {
	currency, err := b.currencyService.GetCurrency(ctx, bill.Currency)
	if err != nil {
		return nil, err
	}
	if !currency.Enabled {
		return nil, &errs.Error{Code: errs.InvalidArgument, Message: "currency is not enabled"}
	}

	workflowID := fmt.Sprintf("bill-%s", bill.IdempotencyKey)

	dbBill, err := b.billRepo.CreateBill(ctx, bills.CreateBillParams{
		Status:         string(model.BillStatusPending),
		Currency:       bill.Currency,
		StartTime:      pgtype.Timestamptz{Time: bill.StartTime, Valid: true},
		EndTime:        pgtype.Timestamptz{Time: bill.EndTime, Valid: true},
		IdempotencyKey: bill.IdempotencyKey,
		WorkflowID:     pgtype.Text{String: workflowID, Valid: true},
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

	if dbBill.WorkflowID.Valid {
		bill.WorkflowID = &dbBill.WorkflowID.String
	}

	return bill
}
