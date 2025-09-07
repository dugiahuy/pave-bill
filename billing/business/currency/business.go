package currency

import (
	"context"

	"encore.app/billing/model"
	"encore.app/billing/repository/currencies"
)

type Business interface {
	GetCurrency(ctx context.Context, code string) (*model.CurrencyInfo, error)
	ConvertAmount(ctx context.Context, fromCurrency, toCurrency string, amountCents int64) (*model.ConversionResult, error)
}

type business struct {
	currencyRepo currencies.Querier
}

func NewCurrencyBusiness(currencyRepo currencies.Querier) Business {
	return &business{
		currencyRepo: currencyRepo,
	}
}
