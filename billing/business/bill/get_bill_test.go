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
	"encore.app/billing/mocks/repository/lineitem_repo"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
)

func TestGetBill(t *testing.T) {
	testCases := []struct {
		name                string
		billID              int32
		mockGetBillReturn   bills.Bill
		mockGetBillError    error
		mockLineItemsReturn []lineitems.LineItem
		mockLineItemsError  error
		expectedError       string
		expectSuccess       bool
	}{
		{
			name:   "happy_case_with_line_items",
			billID: 1,
			mockGetBillReturn: bills.Bill{
				ID:         1,
				Currency:   "USD",
				Status:     "active",
				StartTime:  pgtype.Timestamptz{Valid: true},
				EndTime:    pgtype.Timestamptz{Valid: true},
				WorkflowID: pgtype.Text{String: "workflow-123", Valid: true},
			},
			mockGetBillError: nil,
			mockLineItemsReturn: []lineitems.LineItem{
				{
					ID:          1,
					AmountCents: 1000,
					Currency:    "USD",
					Description: pgtype.Text{String: "Test item 1", Valid: true},
				},
				{
					ID:          2,
					AmountCents: 500,
					Currency:    "USD",
					Description: pgtype.Text{String: "Test item 2", Valid: true},
				},
			},
			mockLineItemsError: nil,
			expectSuccess:      true,
		},
		{
			name:   "happy_case_no_line_items",
			billID: 2,
			mockGetBillReturn: bills.Bill{
				ID:         2,
				Currency:   "USD",
				Status:     "pending",
				StartTime:  pgtype.Timestamptz{Valid: true},
				EndTime:    pgtype.Timestamptz{Valid: true},
				WorkflowID: pgtype.Text{String: "workflow-456", Valid: true},
			},
			mockGetBillError:    nil,
			mockLineItemsReturn: []lineitems.LineItem{}, // Empty slice
			mockLineItemsError:  nil,
			expectSuccess:       true,
		},
		{
			name:              "bill_not_found",
			billID:            999,
			mockGetBillReturn: bills.Bill{},
			mockGetBillError:  pgx.ErrNoRows,
			expectedError:     "bill not found",
			expectSuccess:     false,
		},
		{
			name:              "database_error_on_get_bill",
			billID:            1,
			mockGetBillReturn: bills.Bill{},
			mockGetBillError:  errors.New("database connection error"),
			expectedError:     "failed to get bill",
			expectSuccess:     false,
		},
		{
			name:   "error_getting_line_items",
			billID: 1,
			mockGetBillReturn: bills.Bill{
				ID:         1,
				Currency:   "USD",
				Status:     "active",
				StartTime:  pgtype.Timestamptz{Valid: true},
				EndTime:    pgtype.Timestamptz{Valid: true},
				WorkflowID: pgtype.Text{String: "workflow-123", Valid: true},
			},
			mockGetBillError:   nil,
			mockLineItemsError: errors.New("line items database error"),
			expectedError:      "failed to get line items",
			expectSuccess:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockBillRepo := bill_repo.NewMockQuerier(ctrl)
			mockLineItemRepo := lineitem_repo.NewMockQuerier(ctrl)

			business := &business{
				billRepo:     mockBillRepo,
				lineItemRepo: mockLineItemRepo,
			}

			mockBillRepo.EXPECT().
				GetBill(gomock.Any(), tc.billID).
				Return(tc.mockGetBillReturn, tc.mockGetBillError)

			if tc.mockGetBillError == nil {
				mockLineItemRepo.EXPECT().
					GetLineItemsByBill(gomock.Any(), pgtype.Int4{Int32: tc.billID, Valid: true}).
					Return(tc.mockLineItemsReturn, tc.mockLineItemsError)
			}

			result, err := business.GetBill(context.Background(), tc.billID)

			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if result != nil {
					assert.Equal(t, tc.mockGetBillReturn.ID, result.ID)
					assert.Equal(t, tc.mockGetBillReturn.Currency, result.Currency)
					assert.Equal(t, tc.mockGetBillReturn.Status, string(result.Status))
					assert.Equal(t, len(tc.mockLineItemsReturn), len(result.LineItems))

					for i, expectedItem := range tc.mockLineItemsReturn {
						if i < len(result.LineItems) {
							assert.Equal(t, expectedItem.ID, result.LineItems[i].ID)
							assert.Equal(t, expectedItem.AmountCents, result.LineItems[i].AmountCents)
							assert.Equal(t, expectedItem.Currency, result.LineItems[i].Currency)
						}
					}
				}
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			}
		})
	}
}
