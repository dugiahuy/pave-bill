package bill

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/repository/lineitem_repo"
	"encore.app/billing/model"
	"encore.app/billing/repository/lineitems"
)

func TestGetLineItemsByBill(t *testing.T) {
	testCases := []struct {
		name              string
		billID            int32
		mockReturn        []lineitems.LineItem
		mockError         error
		expectedError     string
		expectSuccess     bool
		expectedLineItems []model.LineItem
	}{
		{
			name:   "happy_case_with_multiple_line_items",
			billID: 1,
			mockReturn: []lineitems.LineItem{
				{
					ID:             1,
					BillID:         pgtype.Int4{Int32: 1, Valid: true},
					AmountCents:    1000,
					Currency:       "USD",
					Description:    pgtype.Text{String: "Test item 1", Valid: true},
					ReferenceID:    pgtype.Text{String: "ref-123", Valid: true},
					IncurredAt:     pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
					IdempotencyKey: "key-123",
					CreatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
					UpdatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
					Metadata:       []byte(`{"original_amount_cents":2650,"original_currency":"GEL","exchange_rate":2.65}`),
				},
				{
					ID:             2,
					BillID:         pgtype.Int4{Int32: 1, Valid: true},
					AmountCents:    500,
					Currency:       "USD",
					Description:    pgtype.Text{String: "Test item 2", Valid: true},
					ReferenceID:    pgtype.Text{String: "ref-456", Valid: true},
					IncurredAt:     pgtype.Timestamptz{Time: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Valid: true},
					IdempotencyKey: "key-456",
					CreatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Valid: true},
					UpdatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Valid: true},
					Metadata:       []byte{},
				},
			},
			mockError:     nil,
			expectSuccess: true,
			expectedLineItems: []model.LineItem{
				{
					ID:             1,
					BillID:         1,
					AmountCents:    1000,
					Currency:       "USD",
					Description:    "Test item 1",
					ReferenceID:    "ref-123",
					IncurredAt:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
					IdempotencyKey: "key-123",
					CreatedAt:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
					Metadata: &model.CurrencyMetadata{
						OriginalAmountCents: 2650,
						OriginalCurrency:    "GEL",
						ExchangeRate:        2.65,
					},
				},
				{
					ID:             2,
					BillID:         1,
					AmountCents:    500,
					Currency:       "USD",
					Description:    "Test item 2",
					ReferenceID:    "ref-456",
					IncurredAt:     time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC),
					IdempotencyKey: "key-456",
					CreatedAt:      time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC),
					Metadata:       nil,
				},
			},
		},
		{
			name:              "happy_case_no_line_items",
			billID:            2,
			mockReturn:        []lineitems.LineItem{},
			mockError:         pgx.ErrNoRows,
			expectSuccess:     true,
			expectedLineItems: []model.LineItem{},
		},
		{
			name:              "database_error",
			billID:            3,
			mockReturn:        nil,
			mockError:         errors.New("database connection error"),
			expectedError:     "failed to get line items",
			expectSuccess:     false,
			expectedLineItems: nil,
		},
		{
			name:   "line_item_with_invalid_metadata",
			billID: 4,
			mockReturn: []lineitems.LineItem{
				{
					ID:             1,
					BillID:         pgtype.Int4{Int32: 4, Valid: true},
					AmountCents:    1000,
					Currency:       "USD",
					Description:    pgtype.Text{String: "Test item", Valid: true},
					IncurredAt:     pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
					IdempotencyKey: "key-123",
					CreatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
					UpdatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
					Metadata:       []byte(`invalid json`), // Invalid JSON should be handled gracefully
				},
			},
			mockError:     nil,
			expectSuccess: true,
			expectedLineItems: []model.LineItem{
				{
					ID:             1,
					BillID:         4,
					AmountCents:    1000,
					Currency:       "USD",
					Description:    "Test item",
					IncurredAt:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
					IdempotencyKey: "key-123",
					CreatedAt:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
					Metadata:       nil, // Should be nil when JSON is invalid
				},
			},
		},
		{
			name:   "line_item_with_null_optional_fields",
			billID: 5,
			mockReturn: []lineitems.LineItem{
				{
					ID:             1,
					BillID:         pgtype.Int4{Int32: 5, Valid: true},
					AmountCents:    1000,
					Currency:       "USD",
					Description:    pgtype.Text{Valid: false}, // Null description
					ReferenceID:    pgtype.Text{Valid: false}, // Null reference ID
					IncurredAt:     pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
					IdempotencyKey: "key-123",
					CreatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
					UpdatedAt:      pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Valid: true},
					Metadata:       []byte{},
				},
			},
			mockError:     nil,
			expectSuccess: true,
			expectedLineItems: []model.LineItem{
				{
					ID:             1,
					BillID:         5,
					AmountCents:    1000,
					Currency:       "USD",
					Description:    "", // Should be empty string for null description
					ReferenceID:    "", // Should be empty string for null reference ID
					IncurredAt:     time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
					IdempotencyKey: "key-123",
					CreatedAt:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
					Metadata:       nil,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLineItemRepo := lineitem_repo.NewMockQuerier(ctrl)
			business := &business{lineItemRepo: mockLineItemRepo}

			// Mock the repository call
			mockLineItemRepo.EXPECT().
				GetLineItemsByBill(gomock.Any(), pgtype.Int4{Int32: tc.billID, Valid: true}).
				Return(tc.mockReturn, tc.mockError)

			// Execute the test
			result, err := business.GetLineItemsByBill(context.Background(), tc.billID)

			// Assertions
			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tc.expectedLineItems), len(result))

				// Verify each line item
				for i, expectedItem := range tc.expectedLineItems {
					if i < len(result) {
						assert.Equal(t, expectedItem.ID, result[i].ID)
						assert.Equal(t, expectedItem.BillID, result[i].BillID)
						assert.Equal(t, expectedItem.AmountCents, result[i].AmountCents)
						assert.Equal(t, expectedItem.Currency, result[i].Currency)
						assert.Equal(t, expectedItem.Description, result[i].Description)
						assert.Equal(t, expectedItem.ReferenceID, result[i].ReferenceID)
						assert.Equal(t, expectedItem.IncurredAt, result[i].IncurredAt)
						assert.Equal(t, expectedItem.IdempotencyKey, result[i].IdempotencyKey)
						assert.Equal(t, expectedItem.CreatedAt, result[i].CreatedAt)
						assert.Equal(t, expectedItem.UpdatedAt, result[i].UpdatedAt)

						// Special handling for metadata comparison
						if expectedItem.Metadata == nil {
							assert.Nil(t, result[i].Metadata)
						} else {
							assert.NotNil(t, result[i].Metadata)
							if result[i].Metadata != nil {
								assert.Equal(t, expectedItem.Metadata.OriginalAmountCents, result[i].Metadata.OriginalAmountCents)
								assert.Equal(t, expectedItem.Metadata.OriginalCurrency, result[i].Metadata.OriginalCurrency)
								assert.Equal(t, expectedItem.Metadata.ExchangeRate, result[i].Metadata.ExchangeRate)
							}
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
