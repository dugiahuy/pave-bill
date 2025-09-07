package lineitem

import (
	"context"

	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
	"encore.app/billing/service/currency"
)

type Service interface {
	Create(ctx context.Context, lineItem *model.LineItem) (*model.LineItem, error)
}

type service struct {
	lineItemRepo    lineitems.Querier
	billRepo        bills.Querier
	currencyService currency.Service
}

func NewLineItemService(lineItemRepo lineitems.Querier, billRepo bills.Querier, currencyService currency.Service) Service {
	return &service{
		lineItemRepo:    lineItemRepo,
		billRepo:        billRepo,
		currencyService: currencyService,
	}
}
