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

	"encore.dev/beta/errs"

	"encore.app/billing/mocks/business/bill_business"
	"encore.app/billing/model"
)

func TestAddLineItem(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockBusiness := bill_business.NewMockBusiness(ctrl)
	mockTemporal := mocks.NewClient(t)

	service := &Service{
		business: mockBusiness,
		temporal: mockTemporal,
	}

	now := time.Now()

	testCases := []struct {
		name                     string
		billID                   int32
		request                  *CreateLineItemRequest
		mockAddLineItemReturn    *model.LineItem
		mockAddLineItemError     error
		mockTemporalSignalError  error
		expectedError            string
		expectSuccess            bool
		expectAddLineItemCall    bool
		expectTemporalSignalCall bool
	}{
		{
			name:   "successful_line_item_creation_with_workflow_signal",
			billID: 1,
			request: &CreateLineItemRequest{
				IdempotencyKey: "line-item-key-123",
				Currency:       "USD",
				AmountCents:    1000,
				Description:    "Test line item",
				ReferenceID:    "ref-123",
			},
			mockAddLineItemReturn: &model.LineItem{
				ID:             1,
				BillID:         1,
				Currency:       "USD",
				AmountCents:    1000,
				Description:    "Test line item",
				ReferenceID:    "ref-123",
				IncurredAt:     now,
				IdempotencyKey: "line-item-key-123",
				BillWorkflowID: "workflow-123",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			mockAddLineItemError:     nil,
			mockTemporalSignalError:  nil,
			expectSuccess:            true,
			expectAddLineItemCall:    true,
			expectTemporalSignalCall: true,
		},
		{
			name:   "successful_line_item_creation_signal_fails",
			billID: 2,
			request: &CreateLineItemRequest{
				IdempotencyKey: "line-item-key-456",
				Currency:       "EUR",
				AmountCents:    2500,
				Description:    "Another test line item",
				ReferenceID:    "ref-456",
			},
			mockAddLineItemReturn: &model.LineItem{
				ID:             2,
				BillID:         2,
				Currency:       "EUR",
				AmountCents:    2500,
				Description:    "Another test line item",
				ReferenceID:    "ref-456",
				IncurredAt:     now,
				IdempotencyKey: "line-item-key-456",
				BillWorkflowID: "workflow-456",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			mockAddLineItemError:     nil,
			mockTemporalSignalError:  errors.New("workflow signal failed"),
			expectSuccess:            true, // API still succeeds even if signal fails
			expectAddLineItemCall:    true,
			expectTemporalSignalCall: true,
		},
		{
			name:   "invalid_bill_id_zero",
			billID: 0,
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "Test item",
				ReferenceID: "ref-001",
			},
			expectedError:            "invalid bill ID",
			expectSuccess:            false,
			expectAddLineItemCall:    false,
			expectTemporalSignalCall: false,
		},
		{
			name:   "invalid_bill_id_negative",
			billID: -5,
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "Test item",
				ReferenceID: "ref-002",
			},
			expectedError:            "invalid bill ID",
			expectSuccess:            false,
			expectAddLineItemCall:    false,
			expectTemporalSignalCall: false,
		},
		{
			name:   "add_line_item_business_logic_fails",
			billID: 3,
			request: &CreateLineItemRequest{
				IdempotencyKey: "line-item-key-789",
				Currency:       "GBP",
				AmountCents:    5000,
				Description:    "Failed line item",
				ReferenceID:    "ref-789",
			},
			mockAddLineItemError:     &errs.Error{Code: errs.NotFound, Message: "bill not found"},
			expectedError:            "bill not found",
			expectSuccess:            false,
			expectAddLineItemCall:    true,
			expectTemporalSignalCall: false,
		},
		{
			name:   "bill_closed_cannot_add_line_item",
			billID: 4,
			request: &CreateLineItemRequest{
				IdempotencyKey: "line-item-key-closed",
				Currency:       "USD",
				AmountCents:    3000,
				Description:    "Line item for closed bill",
				ReferenceID:    "ref-closed",
			},
			mockAddLineItemError:     &errs.Error{Code: errs.FailedPrecondition, Message: "cannot add line item to closed bill"},
			expectedError:            "cannot add line item to closed bill",
			expectSuccess:            false,
			expectAddLineItemCall:    true,
			expectTemporalSignalCall: false,
		},
		{
			name:   "large_amount_line_item",
			billID: 5,
			request: &CreateLineItemRequest{
				IdempotencyKey: "line-item-key-large",
				Currency:       "USD",
				AmountCents:    999999999,
				Description:    "Large amount line item",
				ReferenceID:    "ref-large",
			},
			mockAddLineItemReturn: &model.LineItem{
				ID:             5,
				BillID:         5,
				Currency:       "USD",
				AmountCents:    999999999,
				Description:    "Large amount line item",
				ReferenceID:    "ref-large",
				IncurredAt:     now,
				IdempotencyKey: "line-item-key-large",
				BillWorkflowID: "workflow-large",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			mockAddLineItemError:     nil,
			mockTemporalSignalError:  nil,
			expectSuccess:            true,
			expectAddLineItemCall:    true,
			expectTemporalSignalCall: true,
		},
		{
			name:   "duplicate_idempotency_key",
			billID: 6,
			request: &CreateLineItemRequest{
				IdempotencyKey: "duplicate-key",
				Currency:       "USD",
				AmountCents:    1500,
				Description:    "Duplicate request",
				ReferenceID:    "ref-duplicate",
			},
			mockAddLineItemError:     &errs.Error{Code: errs.AlreadyExists, Message: "line item with this idempotency key already exists"},
			expectedError:            "line item with this idempotency key already exists",
			expectSuccess:            false,
			expectAddLineItemCall:    true,
			expectTemporalSignalCall: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up business mock expectations for AddLineItemToBill
			if tc.expectAddLineItemCall {
				mockBusiness.EXPECT().
					AddLineItemToBill(gomock.Any(), tc.billID, gomock.Any()).
					DoAndReturn(func(ctx context.Context, billID int32, lineItem *model.LineItem) (*model.LineItem, error) {
						// Verify the line item fields are set correctly
						assert.Equal(t, tc.billID, lineItem.BillID)
						assert.Equal(t, tc.request.Currency, lineItem.Currency)
						assert.Equal(t, tc.request.AmountCents, lineItem.AmountCents)
						assert.Equal(t, tc.request.Description, lineItem.Description)
						assert.Equal(t, tc.request.ReferenceID, lineItem.ReferenceID)
						assert.Equal(t, tc.request.IdempotencyKey, lineItem.IdempotencyKey)
						assert.False(t, lineItem.IncurredAt.IsZero())

						return tc.mockAddLineItemReturn, tc.mockAddLineItemError
					}).
					Times(1)
			}

			// Set up temporal mock expectations for SignalWorkflow
			if tc.expectTemporalSignalCall && tc.mockAddLineItemError == nil {
				mockTemporal.On("SignalWorkflow",
					mock.Anything, // context
					mock.Anything, // workflowID
					mock.Anything, // runID
					mock.Anything, // signalName
					mock.Anything, // signalArgs
				).Return(tc.mockTemporalSignalError)
			}

			// Execute the API call
			response, err := service.AddLineItem(context.Background(), tc.billID, tc.request)

			// Verify results
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				if tc.expectSuccess {
					assert.Equal(t, tc.mockAddLineItemReturn.ID, response.LineItem.ID)
					assert.Equal(t, tc.mockAddLineItemReturn.BillID, response.LineItem.BillID)
					assert.Equal(t, tc.mockAddLineItemReturn.Currency, response.LineItem.Currency)
					assert.Equal(t, tc.mockAddLineItemReturn.AmountCents, response.LineItem.AmountCents)
					assert.Equal(t, tc.mockAddLineItemReturn.Description, response.LineItem.Description)
					assert.Equal(t, tc.mockAddLineItemReturn.ReferenceID, response.LineItem.ReferenceID)
					assert.Equal(t, tc.mockAddLineItemReturn.IdempotencyKey, response.LineItem.IdempotencyKey)
				}
			}

			// Give async goroutine time to complete for signal calls
			if tc.expectTemporalSignalCall && tc.mockAddLineItemError == nil {
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
}

// TestCreateLineItemRequest_Validation tests the validation logic
func TestCreateLineItemRequest_Validation(t *testing.T) {
	testCases := []struct {
		name          string
		request       *CreateLineItemRequest
		expectedError string
	}{
		{
			name: "valid_request",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "Valid line item",
				ReferenceID: "ref-123",
			},
		},
		{
			name: "missing_currency",
			request: &CreateLineItemRequest{
				Currency:    "",
				AmountCents: 1000,
				Description: "Missing currency",
				ReferenceID: "ref-001",
			},
			expectedError: "required",
		},
		{
			name: "invalid_currency_too_short",
			request: &CreateLineItemRequest{
				Currency:    "US",
				AmountCents: 1000,
				Description: "Short currency",
				ReferenceID: "ref-002",
			},
			expectedError: "len",
		},
		{
			name: "invalid_currency_too_long",
			request: &CreateLineItemRequest{
				Currency:    "USDD",
				AmountCents: 1000,
				Description: "Long currency",
				ReferenceID: "ref-003",
			},
			expectedError: "len",
		},
		{
			name: "invalid_currency_numeric",
			request: &CreateLineItemRequest{
				Currency:    "123",
				AmountCents: 1000,
				Description: "Numeric currency",
				ReferenceID: "ref-004",
			},
			expectedError: "alpha",
		},
		{
			name: "missing_amount",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 0,
				Description: "Missing amount",
				ReferenceID: "ref-005",
			},
			expectedError: "required",
		},
		{
			name: "negative_amount",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: -100,
				Description: "Negative amount",
				ReferenceID: "ref-006",
			},
			expectedError: "min",
		},
		{
			name: "missing_description",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "",
				ReferenceID: "ref-007",
			},
			expectedError: "required",
		},
		{
			name: "description_too_long",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "This is an extremely long description that exceeds the maximum allowed length of 255 characters. It goes on and on with unnecessary details about the line item, providing way more information than needed for a simple description field which should be concise and to the point but this one clearly is not and will definitely exceed the character limit set for descriptions.",
				ReferenceID: "ref-008",
			},
			expectedError: "max",
		},
		{
			name: "missing_reference_id",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "Missing reference",
				ReferenceID: "",
			},
			expectedError: "required",
		},
		{
			name: "reference_id_too_long",
			request: &CreateLineItemRequest{
				Currency:    "USD",
				AmountCents: 1000,
				Description: "Long reference ID",
				ReferenceID: "this-is-a-very-long-reference-id-that-exceeds-the-maximum-allowed-length-of-100-characters-for-reference-ids",
			},
			expectedError: "max",
		},
		{
			name: "valid_request_with_special_characters",
			request: &CreateLineItemRequest{
				Currency:    "EUR",
				AmountCents: 2500,
				Description: "Line item with special chars: #123 - (urgent)",
				ReferenceID: "ref-special-#123",
			},
		},
		{
			name: "valid_request_with_unicode",
			request: &CreateLineItemRequest{
				Currency:    "JPY",
				AmountCents: 150000,
				Description: "支払い項目",
				ReferenceID: "ref-unicode-日本",
			},
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
