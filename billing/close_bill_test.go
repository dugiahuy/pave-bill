package billing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/mocks"
	"go.uber.org/mock/gomock"

	"encore.dev/beta/errs"

	"encore.app/billing/mocks/business/bill_business"
	"encore.app/billing/model"
)

func TestCloseBill(t *testing.T) {

	testCases := []struct {
		name                string
		billID              int32
		request             *CloseBillRequest
		mockCloseBillError  error
		mockGetBillReturn   *model.Bill
		mockGetBillError    error
		expectedError       string
		expectSuccess       bool
		expectCloseBillCall bool
		expectGetBillCall   bool
	}{
		{
			name:   "successful_bill_closure",
			billID: 1,
			request: &CloseBillRequest{
				Reason: "Customer requested closure",
			},
			mockCloseBillError: nil,
			mockGetBillReturn: &model.Bill{
				ID:          1,
				Currency:    "USD",
				Status:      model.BillStatusClosed,
				CloseReason: stringPtr("Customer requested closure"),
				WorkflowID:  stringPtr("bill-test-workflow-1"),
			},
			mockGetBillError:    nil,
			expectSuccess:       true,
			expectCloseBillCall: true,
			expectGetBillCall:   true,
		},
		{
			name:   "invalid_bill_id_zero",
			billID: 0,
			request: &CloseBillRequest{
				Reason: "Some reason",
			},
			expectedError:       "invalid bill ID",
			expectSuccess:       false,
			expectCloseBillCall: false,
			expectGetBillCall:   false,
		},
		{
			name:   "invalid_bill_id_negative",
			billID: -5,
			request: &CloseBillRequest{
				Reason: "Some reason",
			},
			expectedError:       "invalid bill ID",
			expectSuccess:       false,
			expectCloseBillCall: false,
			expectGetBillCall:   false,
		},
		{
			name:   "close_bill_business_logic_fails",
			billID: 2,
			request: &CloseBillRequest{
				Reason: "Business closure",
			},
			mockCloseBillError:  &errs.Error{Code: errs.NotFound, Message: "bill not found"},
			expectedError:       "bill not found",
			expectSuccess:       false,
			expectCloseBillCall: true,
			expectGetBillCall:   false,
		},
		{
			name:   "close_bill_succeeds_but_get_bill_fails",
			billID: 3,
			request: &CloseBillRequest{
				Reason: "System maintenance",
			},
			mockCloseBillError:  nil,
			mockGetBillError:    &errs.Error{Code: errs.Internal, Message: "database error"},
			expectedError:       "database error",
			expectSuccess:       false,
			expectCloseBillCall: true,
			expectGetBillCall:   true,
		},
		{
			name:   "bill_already_closed",
			billID: 4,
			request: &CloseBillRequest{
				Reason: "Duplicate closure attempt",
			},
			mockCloseBillError:  &errs.Error{Code: errs.FailedPrecondition, Message: "bill already closed"},
			expectedError:       "bill already closed",
			expectSuccess:       false,
			expectCloseBillCall: true,
			expectGetBillCall:   false,
		},
		{
			name:   "bill_cannot_be_closed_due_to_state",
			billID: 5,
			request: &CloseBillRequest{
				Reason: "Early closure",
			},
			mockCloseBillError:  &errs.Error{Code: errs.FailedPrecondition, Message: "bill cannot be closed in current state"},
			expectedError:       "bill cannot be closed in current state",
			expectSuccess:       false,
			expectCloseBillCall: true,
			expectGetBillCall:   false,
		},
		{
			name:   "successful_closure_with_long_reason",
			billID: 6,
			request: &CloseBillRequest{
				Reason: "This is a very long reason for closing the bill that might test the validation and storage capabilities of our system",
			},
			mockCloseBillError: nil,
			mockGetBillReturn: &model.Bill{
				ID:          6,
				Currency:    "EUR",
				Status:      model.BillStatusClosed,
				CloseReason: stringPtr("This is a very long reason for closing the bill that might test the validation and storage capabilities of our system"),
				WorkflowID:  stringPtr("bill-test-workflow-6"),
			},
			mockGetBillError:    nil,
			expectSuccess:       true,
			expectCloseBillCall: true,
			expectGetBillCall:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Override async to run synchronously for deterministic test
			originalRunAsync := runAsync
			runAsync = func(op string, fn func(ctx context.Context) error) { _ = fn(context.Background()) }
			defer func() { runAsync = originalRunAsync }()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockBusiness := bill_business.NewMockBusiness(ctrl)
			mockTemporal := mocks.NewClient(t)

			service := &Service{business: mockBusiness, temporal: mockTemporal}

			if tc.expectCloseBillCall {
				mockBusiness.EXPECT().
					CloseBill(gomock.Any(), tc.billID, tc.request.Reason).
					Return(tc.mockCloseBillError).
					Times(1)
			}

			if tc.expectGetBillCall {
				mockBusiness.EXPECT().
					GetBill(gomock.Any(), tc.billID).
					Return(tc.mockGetBillReturn, tc.mockGetBillError).
					Times(1)
			}

			if tc.expectSuccess {
				// Only successful path triggers workflow termination
				mockTemporal.On("TerminateWorkflow", mock.Anything, *tc.mockGetBillReturn.WorkflowID, "", "manual_close_via_api").Return(nil).Once()
			}

			response, err := service.CloseBill(context.Background(), tc.billID, tc.request)

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
					if tc.mockGetBillReturn.CloseReason != nil {
						assert.Equal(t, *tc.mockGetBillReturn.CloseReason, *response.Bill.CloseReason)
					}
					mockTemporal.AssertExpectations(t)
				}
			}
		})
	}
}

// TestCloseBillRequest_Validation tests the validation logic
func TestCloseBillRequest_Validation(t *testing.T) {
	testCases := []struct {
		name          string
		request       *CloseBillRequest
		expectedError string
	}{
		{
			name: "valid_request",
			request: &CloseBillRequest{
				Reason: "Valid closure reason",
			},
		},
		{
			name: "missing_reason",
			request: &CloseBillRequest{
				Reason: "",
			},
			expectedError: "required",
		},
		{
			name: "reason_too_long",
			request: &CloseBillRequest{
				Reason: "This is an extremely long reason that exceeds the maximum allowed length of 255 characters. It goes on and on with unnecessary details about why the bill is being closed, providing way more information than needed for a simple closure reason field which should be concise and to the point but this one clearly is not and will definitely exceed the character limit.",
			},
			expectedError: "max",
		},
		{
			name: "valid_reason_at_max_length",
			request: &CloseBillRequest{
				Reason: "This reason is exactly at the maximum allowed length of 255 characters. It provides a good amount of detail about the closure without being excessive. This should pass validation since it meets the requirement precisely at boundary.",
			},
		},
		{
			name: "reason_with_special_characters",
			request: &CloseBillRequest{
				Reason: "Closure due to policy #123 - customer requested (urgent)",
			},
		},
		{
			name: "reason_with_unicode",
			request: &CloseBillRequest{
				Reason: "Closure requested by customer 顧客",
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

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
