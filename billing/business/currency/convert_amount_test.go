package currency

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/repository/currency_repo"
	"encore.app/billing/model"
	"encore.app/billing/repository/currencies"
)

// Helper function to create pgtype.Numeric from float64
func createNumericFromFloat(f float64) pgtype.Numeric {
	return pgtype.Numeric{
		Int:              big.NewInt(int64(f * 1000000)), // Store with 6 decimal precision
		Exp:              -6,                             // 6 decimal places
		NaN:              false,
		InfinityModifier: pgtype.Finite,
		Valid:            true,
	}
}

func TestConvertAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCurrencyRepo := currency_repo.NewMockQuerier(ctrl)
	business := &business{currencyRepo: mockCurrencyRepo}

	testCases := []struct {
		name                  string
		fromCurrency          string
		toCurrency            string
		amountCents           int64
		fromCurrencyDBReturn  currencies.Currency
		toCurrencyDBReturn    currencies.Currency
		fromCurrencyError     error
		toCurrencyError       error
		expectedResult        *model.ConversionResult
		expectedError         string
		expectGetFromCurrency bool
		expectGetToCurrency   bool
	}{
		{
			name:         "same_currency_no_conversion",
			fromCurrency: "USD",
			toCurrency:   "USD",
			amountCents:  10000,
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 10000,
				Metadata:        nil,
			},
			expectGetFromCurrency: false,
			expectGetToCurrency:   false,
		},
		{
			name:         "usd_to_gel_conversion",
			fromCurrency: "USD",
			toCurrency:   "GEL",
			amountCents:  10000,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      2,
				Code:    pgtype.Text{String: "GEL", Valid: true},
				Rate:    createNumericFromFloat(2.7),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 27000,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: 10000,
					OriginalCurrency:    "USD",
					ExchangeRate:        2.7,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:         "gel_to_usd_conversion",
			fromCurrency: "GEL",
			toCurrency:   "USD",
			amountCents:  27000,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      2,
				Code:    pgtype.Text{String: "GEL", Valid: true},
				Rate:    createNumericFromFloat(2.7),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 10000,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: 27000,
					OriginalCurrency:    "GEL",
					ExchangeRate:        0.37037037037037035,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:         "fractional_conversion_with_rounding",
			fromCurrency: "USD",
			toCurrency:   "EUR",
			amountCents:  10033,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      3,
				Code:    pgtype.Text{String: "EUR", Valid: true},
				Rate:    createNumericFromFloat(0.85),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 8528,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: 10033,
					OriginalCurrency:    "USD",
					ExchangeRate:        0.85,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:         "zero_amount_conversion",
			fromCurrency: "USD",
			toCurrency:   "GEL",
			amountCents:  0,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      2,
				Code:    pgtype.Text{String: "GEL", Valid: true},
				Rate:    createNumericFromFloat(2.7),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 0,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: 0,
					OriginalCurrency:    "USD",
					ExchangeRate:        2.7,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:         "negative_amount_conversion",
			fromCurrency: "USD",
			toCurrency:   "GEL",
			amountCents:  -5000,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      2,
				Code:    pgtype.Text{String: "GEL", Valid: true},
				Rate:    createNumericFromFloat(2.7),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: -13500,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: -5000,
					OriginalCurrency:    "USD",
					ExchangeRate:        2.7,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:                  "from_currency_not_found",
			fromCurrency:          "INVALID",
			toCurrency:            "USD",
			amountCents:           10000,
			fromCurrencyError:     errors.New("currency not found"),
			expectedError:         "currency not supported",
			expectGetFromCurrency: true,
			expectGetToCurrency:   false,
		},
		{
			name:         "to_currency_not_found",
			fromCurrency: "USD",
			toCurrency:   "INVALID",
			amountCents:  10000,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			toCurrencyError:       errors.New("currency not found"),
			expectedError:         "currency not supported",
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:         "cross_currency_conversion_eur_to_jpy",
			fromCurrency: "EUR",
			toCurrency:   "JPY",
			amountCents:  8500,
			fromCurrencyDBReturn: currencies.Currency{
				ID:      3,
				Code:    pgtype.Text{String: "EUR", Valid: true},
				Rate:    createNumericFromFloat(0.85),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      4,
				Code:    pgtype.Text{String: "JPY", Valid: true},
				Rate:    createNumericFromFloat(150.0),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 1500000,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: 8500,
					OriginalCurrency:    "EUR",
					ExchangeRate:        176.47058823529412,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
		{
			name:         "small_fractional_amount_rounds_to_zero",
			fromCurrency: "USD",
			toCurrency:   "BTC",
			amountCents:  1, // $0.01
			fromCurrencyDBReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Rate:    createNumericFromFloat(1.0),
				Enabled: true,
			},
			toCurrencyDBReturn: currencies.Currency{
				ID:      5,
				Code:    pgtype.Text{String: "BTC", Valid: true},
				Rate:    createNumericFromFloat(0.000025),
				Enabled: true,
			},
			expectedResult: &model.ConversionResult{
				ConvertedAmount: 0,
				Metadata: &model.CurrencyMetadata{
					OriginalAmountCents: 1,
					OriginalCurrency:    "USD",
					ExchangeRate:        0.000025,
				},
			},
			expectGetFromCurrency: true,
			expectGetToCurrency:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectGetFromCurrency {
				mockCurrencyRepo.EXPECT().
					GetCurrency(gomock.Any(), pgtype.Text{String: tc.fromCurrency, Valid: true}).
					Return(tc.fromCurrencyDBReturn, tc.fromCurrencyError)
			}

			if tc.expectGetToCurrency && tc.fromCurrencyError == nil {
				mockCurrencyRepo.EXPECT().
					GetCurrency(gomock.Any(), pgtype.Text{String: tc.toCurrency, Valid: true}).
					Return(tc.toCurrencyDBReturn, tc.toCurrencyError)
			}

			result, err := business.ConvertAmount(context.Background(), tc.fromCurrency, tc.toCurrency, tc.amountCents)

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedResult.ConvertedAmount, result.ConvertedAmount)

				if tc.expectedResult.Metadata == nil {
					assert.Nil(t, result.Metadata)
				} else {
					assert.NotNil(t, result.Metadata)
					assert.Equal(t, tc.expectedResult.Metadata.OriginalAmountCents, result.Metadata.OriginalAmountCents)
					assert.Equal(t, tc.expectedResult.Metadata.OriginalCurrency, result.Metadata.OriginalCurrency)
					assert.InDelta(t, tc.expectedResult.Metadata.ExchangeRate, result.Metadata.ExchangeRate, 0.0001)
				}
			}
		})
	}
}
