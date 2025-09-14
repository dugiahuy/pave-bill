package bill

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/repository/bill_repo"
	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

func TestCreateBill(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := bill_repo.NewMockQuerier(ctrl)
	business := &business{billRepo: mockRepo}

	testCases := []struct {
		name          string
		input         *model.Bill
		mockReturn    bills.Bill
		mockError     error
		expectedError string
		expectSuccess bool
	}{
		{
			name: "happy_case",
			input: &model.Bill{
				Currency:       "USD",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "test-key-123",
			},
			mockReturn: bills.Bill{
				ID:             1,
				Currency:       "USD",
				Status:         "pending",
				IdempotencyKey: "test-key-123",
			},
			mockError:     nil,
			expectSuccess: true,
		},
		{
			name: "duplicate_error",
			input: &model.Bill{
				Currency:       "USD",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "duplicate-key",
			},
			mockReturn:    bills.Bill{},
			mockError:     &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			expectedError: "bill is duplicated",
			expectSuccess: false,
		},
		{
			name: "general_error",
			input: &model.Bill{
				Currency:       "USD",
				StartTime:      time.Now(),
				EndTime:        time.Now().Add(time.Hour),
				IdempotencyKey: "test-key",
			},
			mockReturn:    bills.Bill{},
			mockError:     assert.AnError,
			expectedError: "failed to create bill",
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for each test case
			mockRepo.EXPECT().
				CreateBill(gomock.Any(), gomock.Any()).
				Return(tc.mockReturn, tc.mockError)

			result, err := business.CreateBill(context.Background(), tc.input)

			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.mockReturn.ID, result.ID)
				assert.Equal(t, tc.mockReturn.Currency, result.Currency)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}
