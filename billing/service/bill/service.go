package bill

import (
	"context"

	"encore.app/billing/model"
	"encore.app/billing/store/bills"
	"encore.dev/rlog"
)

type Service interface {
	Create(ctx context.Context, bill *model.Bill) (*model.Bill, error)
}

// BillService handles business logic for bills
type service struct {
	billRepo bills.Querier
}

// NewBillService creates a new bill service
func NewBillService(billRepo bills.Querier) Service {
	rlog.Info("Initializing Bill Service", "billRepo", billRepo)
	return &service{
		billRepo: billRepo,
	}
}
