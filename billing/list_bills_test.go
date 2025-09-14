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

func TestListBills(t *testing.T) {
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
		name                string
		request             *GetBillsRequest
		mockListBillsReturn []*model.Bill
		mockTotalCount      int64
		mockListBillsError  error
		expectedError       string
		expectSuccess       bool
		expectListBillsCall bool
		expectedLimit       int
		expectedOffset      int
	}{
		{
			name: "successful_bills_listing_default_limit",
			request: &GetBillsRequest{
				Limit:  0, // Should default to 10
				Offset: 0,
			},
			mockListBillsReturn: []*model.Bill{
				{
					ID:               1,
					Currency:         "USD",
					Status:           model.BillStatusActive,
					TotalAmountCents: 5000,
					StartTime:        now,
					EndTime:          futureTime,
					IdempotencyKey:   "test-key-1",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
				{
					ID:               2,
					Currency:         "EUR",
					Status:           model.BillStatusPending,
					TotalAmountCents: 3000,
					StartTime:        futureTime,
					EndTime:          futureTime.Add(time.Hour),
					IdempotencyKey:   "test-key-2",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			},
			mockTotalCount:      25,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       10,
			expectedOffset:      0,
		},
		{
			name: "successful_bills_listing_custom_limit",
			request: &GetBillsRequest{
				Limit:  5,
				Offset: 10,
			},
			mockListBillsReturn: []*model.Bill{
				{
					ID:               11,
					Currency:         "GBP",
					Status:           model.BillStatusClosed,
					CloseReason:      stringPtr("Completed"),
					TotalAmountCents: 7500,
					StartTime:        now.Add(-48 * time.Hour),
					EndTime:          now.Add(-24 * time.Hour),
					BilledAt:         &now,
					IdempotencyKey:   "test-key-11",
					CreatedAt:        now.Add(-48 * time.Hour),
					UpdatedAt:        now,
				},
			},
			mockTotalCount:      25,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       5,
			expectedOffset:      10,
		},
		{
			name: "limit_exceeds_maximum_capped_to_100",
			request: &GetBillsRequest{
				Limit:  150, // Should be capped to 100
				Offset: 0,
			},
			mockListBillsReturn: []*model.Bill{
				{
					ID:               1,
					Currency:         "USD",
					Status:           model.BillStatusActive,
					TotalAmountCents: 1000,
					StartTime:        now,
					EndTime:          futureTime,
					IdempotencyKey:   "test-key-capped",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			},
			mockTotalCount:      150,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       100,
			expectedOffset:      0,
		},
		{
			name: "negative_limit_defaults_to_10",
			request: &GetBillsRequest{
				Limit:  -5, // Should default to 10
				Offset: 0,
			},
			mockListBillsReturn: []*model.Bill{},
			mockTotalCount:      0,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       10,
			expectedOffset:      0,
		},
		{
			name: "empty_results",
			request: &GetBillsRequest{
				Limit:  10,
				Offset: 1000, // Far beyond available data
			},
			mockListBillsReturn: []*model.Bill{},
			mockTotalCount:      5,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       10,
			expectedOffset:      1000,
		},
		{
			name: "business_logic_error",
			request: &GetBillsRequest{
				Limit:  10,
				Offset: 0,
			},
			mockListBillsError:  &errs.Error{Code: errs.Internal, Message: "database connection error"},
			expectedError:       "database connection error",
			expectSuccess:       false,
			expectListBillsCall: true,
			expectedLimit:       10,
			expectedOffset:      0,
		},
		{
			name: "bills_with_line_items",
			request: &GetBillsRequest{
				Limit:  2,
				Offset: 0,
			},
			mockListBillsReturn: []*model.Bill{
				{
					ID:               1,
					Currency:         "USD",
					Status:           model.BillStatusActive,
					TotalAmountCents: 15000,
					StartTime:        now,
					EndTime:          futureTime,
					IdempotencyKey:   "test-key-with-items-1",
					LineItems: []model.LineItem{
						{
							ID:          1,
							BillID:      1,
							Currency:    "USD",
							AmountCents: 10000,
							Description: "First item",
							ReferenceID: "ref-001",
							IncurredAt:  now,
						},
						{
							ID:          2,
							BillID:      1,
							Currency:    "USD",
							AmountCents: 5000,
							Description: "Second item",
							ReferenceID: "ref-002",
							IncurredAt:  now,
						},
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:               2,
					Currency:         "EUR",
					Status:           model.BillStatusPending,
					TotalAmountCents: 0,
					StartTime:        futureTime,
					EndTime:          futureTime.Add(time.Hour),
					IdempotencyKey:   "test-key-with-items-2",
					LineItems:        []model.LineItem{},
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			},
			mockTotalCount:      2,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       2,
			expectedOffset:      0,
		},
		{
			name: "bills_with_different_statuses",
			request: &GetBillsRequest{
				Limit:  4,
				Offset: 0,
			},
			mockListBillsReturn: []*model.Bill{
				{
					ID:               1,
					Currency:         "USD",
					Status:           model.BillStatusPending,
					TotalAmountCents: 0,
					StartTime:        futureTime,
					EndTime:          futureTime.Add(time.Hour),
					IdempotencyKey:   "pending-bill",
					WorkflowID:       stringPtr("workflow-pending"),
					CreatedAt:        now,
					UpdatedAt:        now,
				},
				{
					ID:               2,
					Currency:         "EUR",
					Status:           model.BillStatusActive,
					TotalAmountCents: 5000,
					StartTime:        now.Add(-time.Hour),
					EndTime:          futureTime,
					IdempotencyKey:   "active-bill",
					CreatedAt:        now.Add(-time.Hour),
					UpdatedAt:        now,
				},
				{
					ID:               3,
					Currency:         "GBP",
					Status:           model.BillStatusClosing,
					TotalAmountCents: 7500,
					StartTime:        now.Add(-2 * time.Hour),
					EndTime:          now.Add(-time.Hour),
					IdempotencyKey:   "closing-bill",
					CreatedAt:        now.Add(-2 * time.Hour),
					UpdatedAt:        now,
				},
				{
					ID:               4,
					Currency:         "JPY",
					Status:           model.BillStatusClosed,
					CloseReason:      stringPtr("Manual closure"),
					TotalAmountCents: 10000,
					StartTime:        now.Add(-24 * time.Hour),
					EndTime:          now.Add(-12 * time.Hour),
					BilledAt:         &now,
					IdempotencyKey:   "closed-bill",
					CreatedAt:        now.Add(-24 * time.Hour),
					UpdatedAt:        now,
				},
			},
			mockTotalCount:      4,
			mockListBillsError:  nil,
			expectSuccess:       true,
			expectListBillsCall: true,
			expectedLimit:       4,
			expectedOffset:      0,
		},
		{
			name: "access_denied_error",
			request: &GetBillsRequest{
				Limit:  10,
				Offset: 0,
			},
			mockListBillsError:  &errs.Error{Code: errs.PermissionDenied, Message: "access denied"},
			expectedError:       "access denied",
			expectSuccess:       false,
			expectListBillsCall: true,
			expectedLimit:       10,
			expectedOffset:      0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up business mock expectations for ListBills
			if tc.expectListBillsCall {
				mockBusiness.EXPECT().
					ListBills(gomock.Any(), int32(tc.expectedLimit), int32(tc.expectedOffset)).
					Return(tc.mockListBillsReturn, tc.mockTotalCount, tc.mockListBillsError).
					Times(1)
			}

			// Execute the API call
			response, err := service.ListBills(context.Background(), tc.request)

			// Verify results
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				if tc.expectSuccess {
					// Check response metadata
					assert.Equal(t, tc.mockTotalCount, response.TotalCount)
					assert.Equal(t, tc.expectedLimit, response.Limit)
					assert.Equal(t, tc.expectedOffset, response.Offset)
					assert.Equal(t, len(tc.mockListBillsReturn), len(response.Bills))

					// Check bill data
					for i, expectedBill := range tc.mockListBillsReturn {
						actualBill := response.Bills[i]
						assert.Equal(t, expectedBill.ID, actualBill.ID)
						assert.Equal(t, expectedBill.Currency, actualBill.Currency)
						assert.Equal(t, expectedBill.Status, actualBill.Status)
						assert.Equal(t, expectedBill.TotalAmountCents, actualBill.TotalAmountCents)
						assert.Equal(t, expectedBill.IdempotencyKey, actualBill.IdempotencyKey)
						assert.Equal(t, expectedBill.StartTime, actualBill.StartTime)
						assert.Equal(t, expectedBill.EndTime, actualBill.EndTime)

						// Check optional fields
						if expectedBill.CloseReason != nil {
							assert.Equal(t, *expectedBill.CloseReason, *actualBill.CloseReason)
						}
						if expectedBill.ErrorMessage != nil {
							assert.Equal(t, *expectedBill.ErrorMessage, *actualBill.ErrorMessage)
						}
						if expectedBill.BilledAt != nil {
							assert.Equal(t, *expectedBill.BilledAt, *actualBill.BilledAt)
						}
						if expectedBill.WorkflowID != nil {
							assert.Equal(t, *expectedBill.WorkflowID, *actualBill.WorkflowID)
						}

						// Check line items
						assert.Equal(t, len(expectedBill.LineItems), len(actualBill.LineItems))
						for j, expectedItem := range expectedBill.LineItems {
							actualItem := actualBill.LineItems[j]
							assert.Equal(t, expectedItem.ID, actualItem.ID)
							assert.Equal(t, expectedItem.BillID, actualItem.BillID)
							assert.Equal(t, expectedItem.Currency, actualItem.Currency)
							assert.Equal(t, expectedItem.AmountCents, actualItem.AmountCents)
							assert.Equal(t, expectedItem.Description, actualItem.Description)
							assert.Equal(t, expectedItem.ReferenceID, actualItem.ReferenceID)
						}
					}
				}
			}
		})
	}
}

// TestListBills_EdgeCases tests edge cases and error conditions
func TestListBills_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBusiness := bill_business.NewMockBusiness(ctrl)
	mockTemporal := mocks.NewClient(t)

	service := &Service{
		business: mockBusiness,
		temporal: mockTemporal,
	}

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockBusiness.EXPECT().
			ListBills(gomock.Any(), int32(10), int32(0)).
			Return(nil, int64(0), context.Canceled).
			Times(1)

		request := &GetBillsRequest{Limit: 10, Offset: 0}
		response, err := service.ListBills(ctx, request)

		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("large_dataset", func(t *testing.T) {
		// Test with maximum limit and large offset
		now := time.Now()
		bills := make([]*model.Bill, 100)
		for i := 0; i < 100; i++ {
			bills[i] = &model.Bill{
				ID:               int32(i + 1000),
				Currency:         "USD",
				Status:           model.BillStatusActive,
				TotalAmountCents: int64((i + 1) * 100),
				StartTime:        now,
				EndTime:          now.Add(time.Hour),
				IdempotencyKey:   "large-dataset-" + string(rune(i+1000)),
				CreatedAt:        now,
				UpdatedAt:        now,
			}
		}

		mockBusiness.EXPECT().
			ListBills(gomock.Any(), int32(100), int32(5000)).
			Return(bills, int64(10000), nil).
			Times(1)

		request := &GetBillsRequest{Limit: 100, Offset: 5000}
		response, err := service.ListBills(context.Background(), request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 100, len(response.Bills))
		assert.Equal(t, int64(10000), response.TotalCount)
		assert.Equal(t, 100, response.Limit)
		assert.Equal(t, 5000, response.Offset)
	})

	t.Run("concurrent_requests", func(t *testing.T) {
		// Test concurrent requests to the same endpoint
		now := time.Now()
		mockBills := []*model.Bill{
			{
				ID:               1,
				Currency:         "USD",
				Status:           model.BillStatusActive,
				TotalAmountCents: 5000,
				StartTime:        now,
				EndTime:          now.Add(time.Hour),
				IdempotencyKey:   "concurrent-test",
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		}

		// Multiple calls should work
		mockBusiness.EXPECT().
			ListBills(gomock.Any(), int32(10), int32(0)).
			Return(mockBills, int64(1), nil).
			Times(3)

		// Make multiple concurrent calls
		done := make(chan bool, 3)
		for i := 0; i < 3; i++ {
			go func() {
				request := &GetBillsRequest{Limit: 10, Offset: 0}
				response, err := service.ListBills(context.Background(), request)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, 1, len(response.Bills))
				assert.Equal(t, int64(1), response.TotalCount)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			<-done
		}
	})
}

// TestListBills_ParameterValidation tests parameter validation and normalization
func TestListBills_ParameterValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBusiness := bill_business.NewMockBusiness(ctrl)
	mockTemporal := mocks.NewClient(t)

	service := &Service{
		business: mockBusiness,
		temporal: mockTemporal,
	}

	parameterTests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		expectedLimit  int32
		expectedOffset int32
		description    string
	}{
		{"zero_limit", 0, 0, 10, 0, "Zero limit should default to 10"},
		{"negative_limit", -10, 0, 10, 0, "Negative limit should default to 10"},
		{"max_limit", 100, 0, 100, 0, "Limit of 100 should be preserved"},
		{"over_max_limit", 200, 0, 100, 0, "Limit over 100 should be capped"},
		{"large_offset", 50, 999999, 50, 999999, "Large offset should be preserved"},
		{"negative_offset", 20, -50, 20, -50, "Negative offset should be preserved"},
	}

	for _, tc := range parameterTests {
		t.Run(tc.name, func(t *testing.T) {
			mockBusiness.EXPECT().
				ListBills(gomock.Any(), tc.expectedLimit, tc.expectedOffset).
				Return([]*model.Bill{}, int64(0), nil).
				Times(1)

			request := &GetBillsRequest{
				Limit:  tc.inputLimit,
				Offset: tc.inputOffset,
			}
			response, err := service.ListBills(context.Background(), request)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, int(tc.expectedLimit), response.Limit, tc.description)
			assert.Equal(t, int(tc.expectedOffset), response.Offset, tc.description)
		})
	}
}
