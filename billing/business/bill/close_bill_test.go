package bill

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"encore.app/billing/mocks/domain/state_machine"
	"encore.app/billing/model"
	"encore.app/billing/repository/bills"
)

func TestCloseBill(t *testing.T) {
	testCases := []struct {
		name                         string
		billID                       int32
		reason                       string
		mockBillStatus               string
		mockTransitionToClosedError  error
		mockTransitionToClosingError error
		mockUpdateBillTotalError     error
		mockFailureTransitionError   error
		expectedError                string
		expectSuccess                bool
		expectTransitionToClosing    bool
		expectUpdateBillTotal        bool
		expectTransitionToClosed     bool
		expectFailureTransition      bool
	}{
		{
			name:                        "pending_bill_direct_close",
			billID:                      1,
			reason:                      "Manual closure",
			mockBillStatus:              string(model.BillStatusPending),
			mockTransitionToClosedError: nil,
			expectSuccess:               true,
			expectTransitionToClosed:    true,
		},
		{
			name:                         "active_bill_full_process_success",
			billID:                       2,
			reason:                       "End of billing period",
			mockBillStatus:               string(model.BillStatusActive),
			mockTransitionToClosingError: nil,
			mockUpdateBillTotalError:     nil,
			mockTransitionToClosedError:  nil,
			expectSuccess:                true,
			expectTransitionToClosing:    true,
			expectUpdateBillTotal:        true,
			expectTransitionToClosed:     true,
		},
		{
			name:                         "active_bill_transition_to_closing_fails",
			billID:                       3,
			reason:                       "Manual closure",
			mockBillStatus:               string(model.BillStatusActive),
			mockTransitionToClosingError: errors.New("failed to transition to closing"),
			expectedError:                "failed to transition to closing",
			expectSuccess:                false,
			expectTransitionToClosing:    true,
		},
		{
			name:                         "active_bill_update_total_fails",
			billID:                       4,
			reason:                       "End of billing period",
			mockBillStatus:               string(model.BillStatusActive),
			mockTransitionToClosingError: nil,
			mockUpdateBillTotalError:     errors.New("database error calculating total"),
			mockFailureTransitionError:   nil,
			expectedError:                "failed to calculate final bill total",
			expectSuccess:                false,
			expectTransitionToClosing:    true,
			expectUpdateBillTotal:        true,
			expectFailureTransition:      true,
		},
		{
			name:                        "pending_bill_transition_fails",
			billID:                      5,
			reason:                      "Manual closure",
			mockBillStatus:              string(model.BillStatusPending),
			mockTransitionToClosedError: errors.New("database error on close transition"),
			expectedError:               "database error on close transition",
			expectSuccess:               false,
			expectTransitionToClosed:    true,
		},
		{
			name:           "already_closed_bill_idempotent",
			billID:         6,
			reason:         "Closure attempt",
			mockBillStatus: string(model.BillStatusClosed), // Already closed
			expectedError:  "",                             // Should be idempotent, no error
			expectSuccess:  true,                           // Should succeed idempotently
		},
		{
			name:           "attention_required_status",
			billID:         7,
			reason:         "Closure attempt",
			mockBillStatus: string(model.BillStatusAttentionRequired),
			expectedError:  "invalid bill status for closure",
			expectSuccess:  false,
		},
		{
			name:           "closing_status_should_fail",
			billID:         8,
			reason:         "Closure attempt",
			mockBillStatus: string(model.BillStatusClosing), // Should not be handled by CloseBill
			expectedError:  "invalid bill status for closure",
			expectSuccess:  false,
		},
		{
			name:                         "active_bill_failure_transition_also_fails",
			billID:                       9,
			reason:                       "End of billing period",
			mockBillStatus:               string(model.BillStatusActive),
			mockTransitionToClosingError: nil,
			mockUpdateBillTotalError:     errors.New("database error"),
			mockFailureTransitionError:   errors.New("failed to set failure state"),
			expectedError:                "failed to set failure state",
			expectSuccess:                false,
			expectTransitionToClosing:    true,
			expectUpdateBillTotal:        true,
			expectFailureTransition:      true,
		},
		{
			name:                         "active_bill_final_close_transition_fails",
			billID:                       10,
			reason:                       "End of billing period",
			mockBillStatus:               string(model.BillStatusActive),
			mockTransitionToClosingError: nil,
			mockUpdateBillTotalError:     nil,
			mockTransitionToClosedError:  errors.New("failed final close transition"),
			expectedError:                "failed final close transition",
			expectSuccess:                false,
			expectTransitionToClosing:    true,
			expectUpdateBillTotal:        true,
			expectTransitionToClosed:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStateMachine := state_machine.NewMockStateMachine(ctrl)
			business := &business{stateMachine: mockStateMachine}

			// Mock GetBillWithLock to simulate bill status and execute business logic
			mockStateMachine.EXPECT().
				GetBillWithLock(gomock.Any(), tc.billID, gomock.Any()).
				DoAndReturn(func(ctx context.Context, billID int32, businessLogic func(bills.Bill) error) error {
					// Simulate the bill with test case status
					mockBill := bills.Bill{
						ID:     tc.billID,
						Status: tc.mockBillStatus,
					}
					return businessLogic(mockBill)
				})

			// Setup expectations based on test case flow
			if tc.expectTransitionToClosing {
				mockStateMachine.EXPECT().
					TransitionToClosingTx(gomock.Any(), tc.billID, tc.reason).
					Return(tc.mockTransitionToClosingError)
			}

			if tc.expectUpdateBillTotal {
				mockStateMachine.EXPECT().
					UpdateBillTotalTx(gomock.Any(), tc.billID).
					Return(tc.mockUpdateBillTotalError)
			}

			if tc.expectFailureTransition {
				expectedErrorMsg := "failed to calculate final bill total: " + tc.mockUpdateBillTotalError.Error()
				mockStateMachine.EXPECT().
					TransitionToFailureStateTx(gomock.Any(), tc.billID, expectedErrorMsg).
					Return(tc.mockFailureTransitionError)
			}

			if tc.expectTransitionToClosed {
				mockStateMachine.EXPECT().
					TransitionToClosedTx(gomock.Any(), tc.billID, tc.reason).
					Return(tc.mockTransitionToClosedError)
			}

			// Execute the test
			err := business.CloseBill(context.Background(), tc.billID, tc.reason)

			// Assertions
			if tc.expectSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tc.expectedError != "" {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			}
		})
	}
}