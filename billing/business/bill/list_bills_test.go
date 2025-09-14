package bill

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/repository/bill_repo"
	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

func TestListBills(t *testing.T) {
	testCases := []struct {
		name                string
		limit               int32
		offset              int32
		mockListBillsReturn []bills.Bill
		mockListBillsError  error
		mockCountReturn     int64
		mockCountError      error
		expectedError       string
		expectSuccess       bool
		expectedBillsCount  int
		expectedTotalCount  int64
	}{
		{
			name:   "happy_case_with_multiple_bills",
			limit:  10,
			offset: 0,
			mockListBillsReturn: []bills.Bill{
				{
					ID:             1,
					Currency:       "USD",
					Status:         string(model.BillStatusActive),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-123", Valid: true},
					IdempotencyKey: "bill-key-1",
				},
				{
					ID:             2,
					Currency:       "EUR",
					Status:         string(model.BillStatusPending),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-456", Valid: true},
					IdempotencyKey: "bill-key-2",
				},
				{
					ID:             3,
					Currency:       "GEL",
					Status:         string(model.BillStatusClosed),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-789", Valid: true},
					IdempotencyKey: "bill-key-3",
				},
			},
			mockListBillsError: nil,
			mockCountReturn:    25, // Total count in database
			mockCountError:     nil,
			expectSuccess:      true,
			expectedBillsCount: 3,
			expectedTotalCount: 25,
		},
		{
			name:                "happy_case_with_pagination",
			limit:               5,
			offset:              10,
			mockListBillsReturn: []bills.Bill{
				{
					ID:             11,
					Currency:       "USD",
					Status:         string(model.BillStatusActive),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-11", Valid: true},
					IdempotencyKey: "bill-key-11",
				},
				{
					ID:             12,
					Currency:       "EUR",
					Status:         string(model.BillStatusPending),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-12", Valid: true},
					IdempotencyKey: "bill-key-12",
				},
			},
			mockListBillsError: nil,
			mockCountReturn:    50,
			mockCountError:     nil,
			expectSuccess:      true,
			expectedBillsCount: 2,
			expectedTotalCount: 50,
		},
		{
			name:                "no_bills_found",
			limit:               10,
			offset:              0,
			mockListBillsReturn: []bills.Bill{},
			mockListBillsError:  pgx.ErrNoRows,
			expectedError:       "bills not found",
			expectSuccess:       false,
		},
		{
			name:                "database_error_on_list_bills",
			limit:               10,
			offset:              0,
			mockListBillsReturn: nil,
			mockListBillsError:  errors.New("database connection error"),
			expectedError:       "failed to list bills",
			expectSuccess:       false,
		},
		{
			name:   "database_error_on_count_bills",
			limit:  10,
			offset: 0,
			mockListBillsReturn: []bills.Bill{
				{
					ID:             1,
					Currency:       "USD",
					Status:         string(model.BillStatusActive),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-123", Valid: true},
					IdempotencyKey: "bill-key-1",
				},
			},
			mockListBillsError: nil,
			mockCountReturn:    0,
			mockCountError:     errors.New("count query failed"),
			expectedError:      "failed to count bills",
			expectSuccess:      false,
		},
		{
			name:                "empty_result_but_successful",
			limit:               10,
			offset:              100, // Offset beyond available data
			mockListBillsReturn: []bills.Bill{},
			mockListBillsError:  nil, // No error, just empty result
			mockCountReturn:     25,
			mockCountError:      nil,
			expectSuccess:       true,
			expectedBillsCount:  0,
			expectedTotalCount:  25,
		},
		{
			name:   "single_bill_result",
			limit:  1,
			offset: 0,
			mockListBillsReturn: []bills.Bill{
				{
					ID:             1,
					Currency:       "USD",
					Status:         string(model.BillStatusActive),
					StartTime:      pgtype.Timestamptz{Valid: true},
					EndTime:        pgtype.Timestamptz{Valid: true},
					WorkflowID:     pgtype.Text{String: "workflow-single", Valid: true},
					IdempotencyKey: "single-bill-key",
				},
			},
			mockListBillsError: nil,
			mockCountReturn:    1,
			mockCountError:     nil,
			expectSuccess:      true,
			expectedBillsCount: 1,
			expectedTotalCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockBillRepo := bill_repo.NewMockQuerier(ctrl)
			business := &business{billRepo: mockBillRepo}

			// Mock ListBills call
			mockBillRepo.EXPECT().
				ListBills(gomock.Any(), bills.ListBillsParams{
					Limit:  tc.limit,
					Offset: tc.offset,
				}).
				Return(tc.mockListBillsReturn, tc.mockListBillsError)

			// Mock CountBills call only if ListBills succeeds
			if tc.mockListBillsError == nil {
				mockBillRepo.EXPECT().
					CountBills(gomock.Any()).
					Return(tc.mockCountReturn, tc.mockCountError)
			}

			// Execute the test
			result, totalCount, err := business.ListBills(context.Background(), tc.limit, tc.offset)

			// Assertions
			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedBillsCount, len(result))
				assert.Equal(t, tc.expectedTotalCount, totalCount)

				// Verify each bill's conversion from database model to domain model
				for i, bill := range result {
					if i < len(tc.mockListBillsReturn) {
						expectedBill := tc.mockListBillsReturn[i]
						assert.Equal(t, expectedBill.ID, bill.ID)
						assert.Equal(t, expectedBill.Currency, bill.Currency)
						assert.Equal(t, expectedBill.Status, string(bill.Status))
						assert.Equal(t, expectedBill.IdempotencyKey, bill.IdempotencyKey)

						// Verify WorkflowID conversion
						if expectedBill.WorkflowID.Valid {
							assert.NotNil(t, bill.WorkflowID)
							if bill.WorkflowID != nil {
								assert.Equal(t, expectedBill.WorkflowID.String, *bill.WorkflowID)
							}
						} else {
							assert.Nil(t, bill.WorkflowID)
						}
					}
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Equal(t, int64(0), totalCount)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			}
		})
	}
}