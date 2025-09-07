package currency

import (
	"context"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

	"encore.dev/beta/errs"

	"encore.app/billing/model"
	"encore.app/billing/repository/currencies"
)

type Service interface {
	GetCurrency(ctx context.Context, code string) (*model.CurrencyInfo, error)
	ConvertAmount(ctx context.Context, fromCurrency, toCurrency string, amountCents int64) (*model.ConversionResult, error)
}

type service struct {
	currencyRepo currencies.Querier
}

func NewCurrencyService(currencyRepo currencies.Querier) Service {
	return &service{
		currencyRepo: currencyRepo,
	}
}

func (s *service) GetCurrency(ctx context.Context, code string) (*model.CurrencyInfo, error) {
	dbCurrency, err := s.currencyRepo.GetCurrency(ctx, pgtype.Text{String: code, Valid: true})
	if err != nil {
		return nil, &errs.Error{Code: errs.NotFound, Message: "currency not supported"}
	}

	currency := &model.CurrencyInfo{
		ID:      dbCurrency.ID,
		Code:    dbCurrency.Code.String,
		Rate:    parseNumeric(dbCurrency.Rate),
		Enabled: dbCurrency.Enabled,
	}

	if dbCurrency.Symbol.Valid {
		currency.Symbol = &dbCurrency.Symbol.String
	}

	return currency, nil
}

func (s *service) ConvertAmount(ctx context.Context, fromCurrency, toCurrency string, amountCents int64) (*model.ConversionResult, error) {
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