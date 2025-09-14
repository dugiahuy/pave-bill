package bill

import (
	"context"
	"errors"
	"testing"

	"encore.app/billing/mocks/business/currency_business"
	"encore.app/billing/mocks/domain/state_machine"
	"encore.app/billing/mocks/repository/lineitem_repo"
	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
	"encore.app/billing/repository/lineitems"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAddLineItemToBill(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStateMachine := state_machine.NewMockStateMachine(ctrl)
	mockCurrencyService := currency_business.NewMockBusiness(ctrl)
	mockLineItemRepo := lineitem_repo.NewMockQuerier(ctrl)

	testCases := []struct {
		name              string
		billID            int32
		lineItem          *model.LineItem
		mockBillStatus    string
		mockConversion    *model.ConversionResult
		mockConversionErr error
		mockCreateReturn  lineitems.LineItem
		mockCreateError   error
		expectedError     string
		expectSuccess     bool
	}{
		{
			name:   "happy_case",
			billID: 1,
			lineItem: &model.LineItem{
				AmountCents:    1000,
				Currency:       "GEL",
				Description:    "Test line item",
				ReferenceID:    "ref-123",
				IdempotencyKey: "key-123",
			},
			mockBillStatus: string(model.BillStatusActive),
			mockConversion: &model.ConversionResult{
				ConvertedAmount: 377, // 1000 GEL converted to USD cents
			},
			mockConversionErr: nil,
			mockCreateReturn: lineitems.LineItem{
				ID:             1,
				BillID:         pgtype.Int4{Int32: 1, Valid: true},
				AmountCents:    377,
				Currency:       "USD",
				Description:    pgtype.Text{String: "Test line item", Valid: true},
				ReferenceID:    pgtype.Text{String: "ref-123", Valid: true},
				IdempotencyKey: "key-123",
			},
			mockCreateError: nil,
			expectSuccess:   true,
		},
		{
			name:   "bill_not_active",
			billID: 1,
			lineItem: &model.LineItem{
				AmountCents:    1000,
				Currency:       "USD",
				Description:    "Test line item",
				IdempotencyKey: "key-123",
			},
			mockBillStatus: string(model.BillStatusPending),
			expectedError:  "bill is not in active state",
			expectSuccess:  false,
		},
		{
			name:   "currency_conversion_error",
			billID: 1,
			lineItem: &model.LineItem{
				AmountCents:    1000,
				Currency:       "INVALID",
				Description:    "Test line item",
				IdempotencyKey: "key-123",
			},
			mockBillStatus:    string(model.BillStatusActive),
			mockConversionErr: errors.New("conversion error"),
			expectedError:     "conversion error",
			expectSuccess:     false,
		},
		{
			name:   "duplicate_line_item",
			billID: 1,
			lineItem: &model.LineItem{
				AmountCents:    1000,
				Currency:       "USD",
				Description:    "Test line item",
				IdempotencyKey: "duplicate-key",
			},
			mockBillStatus: string(model.BillStatusActive),
			mockConversion: &model.ConversionResult{
				ConvertedAmount: 1000,
			},
			mockConversionErr: nil,
			mockCreateError:   &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			expectedError:     "line item already exists",
			expectSuccess:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			business := &business{
				stateMachine:    mockStateMachine,
				currencyService: mockCurrencyService,
			}

			mockStateMachine.EXPECT().
				GetBillWithLock(gomock.Any(), tc.billID, gomock.Any()).
				DoAndReturn(func(ctx context.Context, billID int32, businessLogic func(bills.Bill) error) error {
					mockBill := bills.Bill{
						ID:         tc.billID,
						Status:     tc.mockBillStatus,
						Currency:   "USD",
						WorkflowID: pgtype.Text{String: "workflow-123", Valid: true},
					}

					return businessLogic(mockBill)
				})

			if tc.expectSuccess || tc.mockBillStatus == string(model.BillStatusActive) {
				mockCurrencyService.EXPECT().
					ConvertAmount(gomock.Any(), tc.lineItem.Currency, "USD", tc.lineItem.AmountCents).
					Return(tc.mockConversion, tc.mockConversionErr)

				if tc.mockConversionErr == nil {
					mockStateMachine.EXPECT().
						GetTxLineItemRepo().
						Return(mockLineItemRepo)

					mockLineItemRepo.EXPECT().
						CreateLineItem(gomock.Any(), gomock.Any()).
						Return(tc.mockCreateReturn, tc.mockCreateError)
				}
			}

			result, err := business.AddLineItemToBill(context.Background(), tc.billID, tc.lineItem)

			if tc.expectSuccess {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if result != nil {
					assert.Equal(t, tc.mockCreateReturn.ID, result.ID)
					assert.Equal(t, tc.mockCreateReturn.AmountCents, result.AmountCents)
					assert.Equal(t, tc.mockCreateReturn.Currency, result.Currency)
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
