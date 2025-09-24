package bill

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.dev/beta/errs"

	"encore.app/billing/mocks/business/currency_business"
	"encore.app/billing/mocks/repository/bill_repo"
	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

func TestCreateBill(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := bill_repo.NewMockQuerier(ctrl)
	mockCurrencyService := currency_business.NewMockBusiness(ctrl)
	business := &business{
		billRepo:        mockRepo,
		currencyService: mockCurrencyService,
	}

	testCases := []struct {
		name                     string
		input                    *model.Bill
		mockCurrencyReturn       *model.CurrencyInfo
		mockCurrencyError        error
		mockBillReturn           bills.Bill
		mockBillError            error
		expectedError            string
		expectSuccess            bool
		expectCurrencyCall       bool
		expectBillRepoCall       bool
	}{
		{
			name: "happy_case",
			input: &model.Bill{
				Currency:       "USD",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "test-key-123",
			},
			mockCurrencyReturn: &model.CurrencyInfo{
				Code:    "USD",
				Enabled: true,
			},
			mockCurrencyError: nil,
			mockBillReturn: bills.Bill{
				ID:             1,
				Currency:       "USD",
				Status:         "pending",
				IdempotencyKey: "test-key-123",
			},
			mockBillError:      nil,
			expectSuccess:      true,
			expectCurrencyCall: true,
			expectBillRepoCall: true,
		},
		{
			name: "currency_not_found",
			input: &model.Bill{
				Currency:       "INVALID",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "test-key-invalid",
			},
			mockCurrencyReturn: nil,
			mockCurrencyError:  &errs.Error{Code: errs.NotFound, Message: "currency not found"},
			expectedError:      "currency not found",
			expectSuccess:      false,
			expectCurrencyCall: true,
			expectBillRepoCall: false,
		},
		{
			name: "currency_disabled",
			input: &model.Bill{
				Currency:       "EUR",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "test-key-disabled",
			},
			mockCurrencyReturn: &model.CurrencyInfo{
				Code:    "EUR",
				Enabled: false,
			},
			mockCurrencyError:  nil,
			expectedError:      "currency is not enabled",
			expectSuccess:      false,
			expectCurrencyCall: true,
			expectBillRepoCall: false,
		},
		{
			name: "duplicate_error",
			input: &model.Bill{
				Currency:       "USD",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "duplicate-key",
			},
			mockCurrencyReturn: &model.CurrencyInfo{
				Code:    "USD",
				Enabled: true,
			},
			mockCurrencyError:  nil,
			mockBillReturn:     bills.Bill{},
			mockBillError:      &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			expectedError:      "bill is duplicated",
			expectSuccess:      false,
			expectCurrencyCall: true,
			expectBillRepoCall: true,
		},
		{
			name: "general_error",
			input: &model.Bill{
				Currency:       "USD",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "test-key",
			},
			mockCurrencyReturn: &model.CurrencyInfo{
				Code:    "USD",
				Enabled: true,
			},
			mockCurrencyError:  nil,
			mockBillReturn:     bills.Bill{},
			mockBillError:      assert.AnError,
			expectedError:      "failed to create bill",
			expectSuccess:      false,
			expectCurrencyCall: true,
			expectBillRepoCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup currency service mock expectations
			if tc.expectCurrencyCall {
				mockCurrencyService.EXPECT().
					GetCurrency(gomock.Any(), tc.input.Currency).
					Return(tc.mockCurrencyReturn, tc.mockCurrencyError)
			}

			// Setup bill repository mock expectations
			if tc.expectBillRepoCall {
				mockRepo.EXPECT().
					CreateBill(gomock.Any(), gomock.Any()).
					Return(tc.mockBillReturn, tc.mockBillError)
			}

			result, err := business.CreateBill(context.Background(), tc.input)

			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.mockBillReturn.ID, result.ID)
				assert.Equal(t, tc.mockBillReturn.Currency, result.Currency)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}
