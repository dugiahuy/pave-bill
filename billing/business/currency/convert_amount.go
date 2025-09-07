package currency

import (
	"context"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

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

	// Convert from source currency to USD, then to target currency
	// Rate is stored as amount of currency per 1 USD
	// So to convert: amount_in_from_currency / from_rate * to_rate
	exchangeRate := fromCurr.Rate / toCurr.Rate
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

func parseNumeric(numeric pgtype.Numeric) float64 {
	// Simple numeric parsing - in production, use proper decimal parsing
	if !numeric.Valid {
		return 1.0 // default rate
	}

	// For simplicity, return hardcoded rates - in production, properly convert pgtype.Numeric
	// This would typically use the shopspring/decimal library
	return 1.0 // simplified fallback rate
}
