package currency

import (
	"context"

	"github.com/shopspring/decimal"

	"encore.app/billing/model"
)

func (s *business) ConvertAmount(ctx context.Context, fromCurrency, toCurrency string, amountCents int64) (*model.ConversionResult, error) {
	if fromCurrency == toCurrency {
		return &model.ConversionResult{
			ConvertedAmount: amountCents,
			Metadata:        nil,
		}, nil
	}

	fromCurr, err := s.GetCurrency(ctx, fromCurrency)
	if err != nil {
		return nil, err
	}

	toCurr, err := s.GetCurrency(ctx, toCurrency)
	if err != nil {
		return nil, err
	}

	// Use decimal arithmetic for precise financial calculations
	// amount_in_from_currency / from_rate * to_rate
	amount := decimal.NewFromInt(amountCents)
	fromRate := decimal.NewFromFloat(fromCurr.Rate)
	toRate := decimal.NewFromFloat(toCurr.Rate)

	// Calculate exchange rate: to_rate / from_rate
	exchangeRate := toRate.Div(fromRate)

	// Convert amount with proper rounding
	convertedDecimal := amount.Mul(exchangeRate).Round(0)
	convertedAmount := convertedDecimal.IntPart()

	// Store exchange rate as float64 for compatibility with existing metadata structure
	exchangeRateFloat, _ := exchangeRate.Float64()

	return &model.ConversionResult{
		ConvertedAmount: convertedAmount,
		Metadata: &model.CurrencyMetadata{
			OriginalAmountCents: amountCents,
			OriginalCurrency:    fromCurrency,
			ExchangeRate:        exchangeRateFloat,
		},
	}, nil
}
