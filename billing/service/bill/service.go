package bill

import (
	"context"

	"encore.app/billing/model"
	"encore.app/billing/store/bills"
)

type Service interface {
	Create(ctx context.Context, bill *model.Bill) (*model.Bill, error)
	Get(ctx context.Context, id int32) (*model.Bill, error)
	List(ctx context.Context, limit, offset int32) ([]*model.Bill, int64, error)
}

// BillService handles business logic for bills
type service struct {
	billRepo bills.Querier
}

// NewBillService creates a new bill service
func NewBillService(billRepo bills.Querier) Service {
	return &service{
		billRepo: billRepo,
	}
}
