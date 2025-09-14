package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/mocks"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/business/bill_business"
	"encore.app/billing/model"
)

// Run tests using `encore test`, which compiles the Encore app and then runs `go test`.
// It supports all the same flags that the `go test` command does.
// You automatically get tracing for tests in the local dev dash: http://localhost:9400
// Learn more: https://encore.dev/docs/go/develop/testing

func TestCreateBill(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBusiness := bill_business.NewMockBusiness(ctrl)
	mockTemporal := mocks.NewClient(t)

	service := &Service{
		business: mockBusiness,
		temporal: mockTemporal,
	}

	now := time.Now()
	futureTime := now.Add(24 * time.Hour)

	testCases := []struct {
		name               string
		request            *CreateBillRequest
		mockBusinessReturn *model.Bill
		mockBusinessError  error
		mockTemporalError  error
		expectedError      string
		expectSuccess      bool
		expectWorkflow     bool
	}{
		{
			name: "successful_bill_creation_with_workflow",
			request: &CreateBillRequest{
				IdempotencyKey: "test-key-123",
				Currency:       "USD",
				StartTime:      futureTime,
				EndTime:        futureTime.Add(time.Hour),
			},
			mockBusinessReturn: &model.Bill{
				ID:             1,
				Currency:       "USD",
				Status:         model.BillStatusPending,
				StartTime:      futureTime,
				EndTime:        futureTime.Add(time.Hour),
				IdempotencyKey: "test-key-123",
			},
			mockBusinessError: nil,
			mockTemporalError: nil,
			expectSuccess:     true,
			expectWorkflow:    true,
		},
		{
			name: "successful_bill_creation_workflow_fails",
			request: &CreateBillRequest{
				IdempotencyKey: "test-key-456",
				Currency:       "EUR",
				StartTime:      futureTime,
				EndTime:        futureTime.Add(2 * time.Hour),
			},
			mockBusinessReturn: &model.Bill{
				ID:             2,
				Currency:       "EUR",
				Status:         model.BillStatusPending,
				StartTime:      futureTime,
				EndTime:        futureTime.Add(2 * time.Hour),
				IdempotencyKey: "test-key-456",
			},
			mockBusinessError: nil,
			mockTemporalError: errors.New("temporal workflow failed"),
			expectSuccess:     true, // API still succeeds even if workflow fails
			expectWorkflow:    true,
		},
		{
			name: "bill_creation_fails",
			request: &CreateBillRequest{
				IdempotencyKey: "test-key-789",
				Currency:       "USD",
				StartTime:      futureTime,
				EndTime:        futureTime.Add(time.Hour),
			},
			mockBusinessReturn: nil,
			mockBusinessError:  errors.New("database error"),
			expectedError:      "database error",
			expectSuccess:      false,
			expectWorkflow:     false,
		},
		{
			name: "zero_start_time_sets_to_now",
			request: &CreateBillRequest{
				IdempotencyKey: "test-key-now",
				Currency:       "GEL",
				StartTime:      time.Time{}, // Zero time
				EndTime:        futureTime,
			},
			mockBusinessReturn: &model.Bill{
				ID:             3,
				Currency:       "GEL",
				Status:         model.BillStatusPending,
				StartTime:      now, // Will be set to now
				EndTime:        futureTime,
				IdempotencyKey: "test-key-now",
			},
			mockBusinessError: nil,
			mockTemporalError: nil,
			expectSuccess:     true,
			expectWorkflow:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up business mock expectations
			mockBusiness.EXPECT().
				CreateBill(gomock.Any(), gomock.Any()).
				Return(tc.mockBusinessReturn, tc.mockBusinessError).
				Times(1)

			// Set up temporal mock expectations only if workflow should be called
			if tc.expectWorkflow && tc.mockBusinessError == nil {
				mockTemporal.On("ExecuteWorkflow",
					mock.Anything, // context
					mock.Anything, // StartWorkflowOptions
					mock.Anything, // workflow function
					mock.Anything, // workflow args
				).Return(nil, tc.mockTemporalError)
			}

			// Execute the API call
			response, err := service.CreateBill(context.Background(), tc.request)

			// Verify results
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				if tc.expectSuccess {
					assert.Equal(t, tc.mockBusinessReturn.ID, response.Bill.ID)
					assert.Equal(t, tc.mockBusinessReturn.Currency, response.Bill.Currency)
					assert.Equal(t, tc.mockBusinessReturn.Status, response.Bill.Status)
					assert.Equal(t, tc.mockBusinessReturn.IdempotencyKey, response.Bill.IdempotencyKey)
				}
			}
		})
	}
}

// TestCreateBillRequest_Validation tests the validation logic
func TestCreateBillRequest_Validation(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-time.Hour)
	futureTime := now.Add(time.Hour)

	testCases := []struct {
		name          string
		request       *CreateBillRequest
		expectedError string
	}{
		{
			name: "valid_request",
			request: &CreateBillRequest{
				Currency:  "USD",
				StartTime: futureTime,
				EndTime:   futureTime.Add(time.Hour),
			},
		},
		{
			name: "invalid_currency_too_short",
			request: &CreateBillRequest{
				Currency:  "US",
				StartTime: futureTime,
				EndTime:   futureTime.Add(time.Hour),
			},
			expectedError: "len",
		},
		{
			name: "invalid_currency_too_long",
			request: &CreateBillRequest{
				Currency:  "USDD",
				StartTime: futureTime,
				EndTime:   futureTime.Add(time.Hour),
			},
			expectedError: "len",
		},
		{
			name: "invalid_currency_numeric",
			request: &CreateBillRequest{
				Currency:  "123",
				StartTime: futureTime,
				EndTime:   futureTime.Add(time.Hour),
			},
			expectedError: "alpha",
		},
		{
			name: "missing_currency",
			request: &CreateBillRequest{
				Currency:  "",
				StartTime: futureTime,
				EndTime:   futureTime.Add(time.Hour),
			},
			expectedError: "required",
		},
		{
			name: "missing_end_time",
			request: &CreateBillRequest{
				Currency:  "USD",
				StartTime: futureTime,
				EndTime:   time.Time{},
			},
			expectedError: "required",
		},
		{
			name: "start_time_in_past",
			request: &CreateBillRequest{
				Currency:  "USD",
				StartTime: pastTime,
				EndTime:   futureTime,
			},
			expectedError: "start_time must be in the future",
		},
		{
			name: "end_time_in_past",
			request: &CreateBillRequest{
				Currency:  "USD",
				StartTime: futureTime,
				EndTime:   pastTime,
			},
			expectedError: "end_time must be in the future",
		},
		{
			name: "end_time_before_start_time",
			request: &CreateBillRequest{
				Currency:  "USD",
				StartTime: time.Time{}, // Zero time, will be set to now
				EndTime:   pastTime,
			},
			expectedError: "end_time must be after start_time",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
