package service

import (
	"encore.app/billing/repository"
	"encore.app/billing/service/bill"
	"encore.app/billing/service/currency"
	"encore.app/billing/service/lineitem"
)

// Services holds all business services
type Services struct {
	Bill     bill.Service
	LineItem lineitem.Service
	Currency currency.Service
}

// NewServices creates a new services container
func NewServices(repo *repository.Repository) Services {
	currencyService := currency.NewCurrencyService(repo.Currencies)
	billService := bill.NewBillService(repo.Bills)
	lineItemService := lineitem.NewLineItemService(repo.LineItems, repo.Bills, currencyService)

	return Services{
		Bill:     billService,
		LineItem: lineItemService,
		Currency: currencyService,
	}
}
