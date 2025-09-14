package currency

import (
	"context"
	"math"

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

	// amount_in_from_currency / from_rate * to_rate
	exchangeRate := toCurr.Rate / fromCurr.Rate
	convertedAmount := int64(math.Round(float64(amountCents) * exchangeRate))

	return &model.ConversionResult{
		ConvertedAmount: convertedAmount,
		Metadata: &model.CurrencyMetadata{
			OriginalAmountCents: amountCents,
			OriginalCurrency:    fromCurrency,
			ExchangeRate:        exchangeRate,
		},
	}, nil
}
