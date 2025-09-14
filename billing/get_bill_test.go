package billing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/mocks"
	"go.uber.org/mock/gomock"

	"encore.dev/beta/errs"

	"encore.app/billing/mocks/business/bill_business"
	"encore.app/billing/model"
)

func TestGetBill(t *testing.T) {
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
		name              string
		billID            int
		mockGetBillReturn *model.Bill
		mockGetBillError  error
		expectedError     string
		expectSuccess     bool
		expectGetBillCall bool
	}{
		{
			name:   "successful_bill_retrieval",
			billID: 1,
			mockGetBillReturn: &model.Bill{
				ID:               1,
				Currency:         "USD",
				Status:           model.BillStatusActive,
				TotalAmountCents: 5000,
				StartTime:        now,
				EndTime:          futureTime,
				IdempotencyKey:   "test-key-123",
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			mockGetBillError:  nil,
			expectSuccess:     true,
			expectGetBillCall: true,
		},
		{
			name:              "invalid_bill_id_zero",
			billID:            0,
			expectedError:     "invalid bill ID",
			expectSuccess:     false,
			expectGetBillCall: false,
		},
		{
			name:              "invalid_bill_id_negative",
			billID:            -5,
			expectedError:     "invalid bill ID",
			expectSuccess:     false,
			expectGetBillCall: false,
		},
		{
			name:              "bill_not_found",
			billID:            999,
			mockGetBillError:  &errs.Error{Code: errs.NotFound, Message: "bill not found"},
			expectedError:     "bill not found",
			expectSuccess:     false,
			expectGetBillCall: true,
		},
		{
			name:   "bill_with_line_items",
			billID: 2,
			mockGetBillReturn: &model.Bill{
				ID:               2,
				Currency:         "EUR",
				Status:           model.BillStatusActive,
				TotalAmountCents: 10000,
				StartTime:        now,
				EndTime:          futureTime,
				IdempotencyKey:   "test-key-456",
				LineItems: []model.LineItem{
					{
						ID:          1,
						BillID:      2,
						Currency:    "EUR",
						AmountCents: 5000,
						Description: "First line item",
						ReferenceID: "ref-001",
						IncurredAt:  now,
					},
					{
						ID:          2,
						BillID:      2,
						Currency:    "EUR",
						AmountCents: 5000,
						Description: "Second line item",
						ReferenceID: "ref-002",
						IncurredAt:  now,
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			mockGetBillError:  nil,
			expectSuccess:     true,
			expectGetBillCall: true,
		},
		{
			name:   "closed_bill_with_reason",
			billID: 3,
			mockGetBillReturn: &model.Bill{
				ID:               3,
				Currency:         "GBP",
				Status:           model.BillStatusClosed,
				CloseReason:      stringPtr("Customer requested closure"),
				TotalAmountCents: 7500,
				StartTime:        now.Add(-48 * time.Hour),
				EndTime:          now.Add(-24 * time.Hour),
				BilledAt:         &now,
				IdempotencyKey:   "test-key-789",
				CreatedAt:        now.Add(-48 * time.Hour),
				UpdatedAt:        now,
			},
			mockGetBillError:  nil,
			expectSuccess:     true,
			expectGetBillCall: true,
		},
		{
			name:   "pending_bill",
			billID: 4,
			mockGetBillReturn: &model.Bill{
				ID:               4,
				Currency:         "JPY",
				Status:           model.BillStatusPending,
				TotalAmountCents: 0,
				StartTime:        futureTime,
				EndTime:          futureTime.Add(time.Hour),
				IdempotencyKey:   "test-key-pending",
				WorkflowID:       stringPtr("workflow-123"),
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			mockGetBillError:  nil,
			expectSuccess:     true,
			expectGetBillCall: true,
		},
		{
			name:              "database_error",
			billID:            5,
			mockGetBillError:  &errs.Error{Code: errs.Internal, Message: "database connection error"},
			expectedError:     "database connection error",
			expectSuccess:     false,
			expectGetBillCall: true,
		},
		{
			name:   "bill_with_error_message",
			billID: 7,
			mockGetBillReturn: &model.Bill{
				ID:               7,
				Currency:         "CAD",
				Status:           model.BillStatusClosed,
				ErrorMessage:     stringPtr("Processing failed"),
				TotalAmountCents: 0,
				StartTime:        now.Add(-24 * time.Hour),
				EndTime:          now,
				IdempotencyKey:   "test-key-error",
				CreatedAt:        now.Add(-24 * time.Hour),
				UpdatedAt:        now,
			},
			mockGetBillError:  nil,
			expectSuccess:     true,
			expectGetBillCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up business mock expectations for GetBill
			if tc.expectGetBillCall {
				mockBusiness.EXPECT().
					GetBill(gomock.Any(), int32(tc.billID)).
					Return(tc.mockGetBillReturn, tc.mockGetBillError).
					Times(1)
			}

			// Execute the API call
			response, err := service.GetBill(context.Background(), tc.billID)

			// Verify results
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				if tc.expectSuccess {
					assert.Equal(t, tc.mockGetBillReturn.ID, response.Bill.ID)
					assert.Equal(t, tc.mockGetBillReturn.Currency, response.Bill.Currency)
					assert.Equal(t, tc.mockGetBillReturn.Status, response.Bill.Status)
					assert.Equal(t, tc.mockGetBillReturn.TotalAmountCents, response.Bill.TotalAmountCents)
					assert.Equal(t, tc.mockGetBillReturn.IdempotencyKey, response.Bill.IdempotencyKey)
					assert.Equal(t, tc.mockGetBillReturn.StartTime, response.Bill.StartTime)
					assert.Equal(t, tc.mockGetBillReturn.EndTime, response.Bill.EndTime)

					// Check optional fields
					if tc.mockGetBillReturn.CloseReason != nil {
						assert.Equal(t, *tc.mockGetBillReturn.CloseReason, *response.Bill.CloseReason)
					}
					if tc.mockGetBillReturn.ErrorMessage != nil {
						assert.Equal(t, *tc.mockGetBillReturn.ErrorMessage, *response.Bill.ErrorMessage)
					}
					if tc.mockGetBillReturn.BilledAt != nil {
						assert.Equal(t, *tc.mockGetBillReturn.BilledAt, *response.Bill.BilledAt)
					}
					if tc.mockGetBillReturn.WorkflowID != nil {
						assert.Equal(t, *tc.mockGetBillReturn.WorkflowID, *response.Bill.WorkflowID)
					}

					// Check line items if present
					if len(tc.mockGetBillReturn.LineItems) > 0 {
						assert.Equal(t, len(tc.mockGetBillReturn.LineItems), len(response.Bill.LineItems))
						for i, expectedItem := range tc.mockGetBillReturn.LineItems {
							assert.Equal(t, expectedItem.ID, response.Bill.LineItems[i].ID)
							assert.Equal(t, expectedItem.BillID, response.Bill.LineItems[i].BillID)
							assert.Equal(t, expectedItem.Currency, response.Bill.LineItems[i].Currency)
							assert.Equal(t, expectedItem.AmountCents, response.Bill.LineItems[i].AmountCents)
							assert.Equal(t, expectedItem.Description, response.Bill.LineItems[i].Description)
							assert.Equal(t, expectedItem.ReferenceID, response.Bill.LineItems[i].ReferenceID)
						}
					}
				}
			}
		})
	}
}

// TestGetBill_ParameterValidation tests parameter validation
func TestGetBill_ParameterValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBusiness := bill_business.NewMockBusiness(ctrl)
	mockTemporal := mocks.NewClient(t)

	service := &Service{
		business: mockBusiness,
		temporal: mockTemporal,
	}

	invalidIDs := []struct {
		name   string
		id     int
		reason string
	}{
		{"zero_id", 0, "ID cannot be zero"},
		{"negative_small", -1, "ID cannot be negative"},
		{"negative_large", -999999, "ID cannot be large negative"},
	}

	for _, tc := range invalidIDs {
		t.Run(tc.name, func(t *testing.T) {
			response, err := service.GetBill(context.Background(), tc.id)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid bill ID")
			assert.Nil(t, response)
		})
	}
}
