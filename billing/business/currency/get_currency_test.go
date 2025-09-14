package currency

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/repository/currency_repo"
	"encore.app/billing/repository/currencies"
)

func TestGetCurrency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := currency_repo.NewMockQuerier(ctrl)
	business := &business{currencyRepo: mockRepo}

	testCases := []struct {
		name          string
		inputCode     string
		mockReturn    currencies.Currency
		mockError     error
		expectedError string
		expectSuccess bool
	}{
		{
			name:      "happy_case",
			inputCode: "USD",
			mockReturn: currencies.Currency{
				ID:      1,
				Code:    pgtype.Text{String: "USD", Valid: true},
				Symbol:  pgtype.Text{String: "$", Valid: true},
				Rate:    pgtype.Numeric{Int: nil, Exp: 0, NaN: false, InfinityModifier: 0, Valid: true},
				Enabled: true,
			},
			mockError:     nil,
			expectSuccess: true,
		},
		{
			name:          "currency_not_found",
			inputCode:     "XYZ",
			mockReturn:    currencies.Currency{},
			mockError:     errors.New("not found"),
			expectedError: "currency not supported",
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo.EXPECT().GetCurrency(
				context.Background(),
				pgtype.Text{String: tc.inputCode, Valid: true},
			).Return(tc.mockReturn, tc.mockError)

			result, err := business.GetCurrency(context.Background(), tc.inputCode)

			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.inputCode, result.Code)
				if tc.mockReturn.Symbol.Valid {
					assert.Equal(t, tc.mockReturn.Symbol.String, *result.Symbol)
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}
