package bill

import (
	"context"

	"encore.app/billing/business/currency"
	"encore.app/billing/domain"
	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
)

type Business interface {
	CreateBill(ctx context.Context, bill *model.Bill) (*model.Bill, error)
	GetBill(ctx context.Context, id int32) (*model.Bill, error)
	ListBills(ctx context.Context, limit, offset int32) ([]*model.Bill, int64, error)
	ActivateBill(ctx context.Context, billID int32) error
	CloseBill(ctx context.Context, id int32, reason string) error
	UpdateBillTotal(ctx context.Context, billID int32) error

	AddLineItemToBill(ctx context.Context, billID int32, lineItem *model.LineItem) (*model.LineItem, error)
	GetLineItemsByBill(ctx context.Context, billID int32) ([]model.LineItem, error)
}

// BillBusiness handles business logic for bills and line items
type business struct {
	billRepo        bills.Querier
	lineItemRepo    lineitems.Querier
	currencyService currency.Business
	stateMachine    *domain.BillStateMachine
}

// NewBillBusiness creates a new unified bill business layer
func NewBillBusiness(
	billRepo bills.Querier,
	lineItemRepo lineitems.Querier,
	currencyService currency.Business,
	stateMachine *domain.BillStateMachine,
) Business {
	return &business{
		billRepo:        billRepo,
		lineItemRepo:    lineItemRepo,
		currencyService: currencyService,
		stateMachine:    stateMachine,
	}
}
