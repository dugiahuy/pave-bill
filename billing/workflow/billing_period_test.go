package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/mock/gomock"

	billmock "encore.app/billing/mocks/business/bill_business"
)

// helper to register activities & set dependencies to mock
func setupMockDeps(ctrl *gomock.Controller, m *billmock.MockBusiness) {
	SetActivityDependencies(m)
}

func TestBillingPeriodWorkflow_ImmediateActivationAndAutoClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBiz := billmock.NewMockBusiness(ctrl)
	setupMockDeps(ctrl, mockBiz)

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(ActivateBillActivity)
	env.RegisterActivity(CloseBillActivity)
	env.RegisterActivity(UpdateBillTotalActivity)

	start := time.Now().Add(-1 * time.Second)
	end := time.Now().Add(1200 * time.Millisecond)

	// Expectations
	mockBiz.EXPECT().ActivateBill(gomock.Any(), int32(101)).Return(nil).Times(1)
	mockBiz.EXPECT().CloseBill(gomock.Any(), int32(101), "auto_close").Return(nil).Times(1)

	params := BillingPeriodWorkflowParams{BillID: 101, StartTime: start, EndTime: end}
	env.ExecuteWorkflow(BillingPeriod, params)
	require.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestBillingPeriodWorkflow_WaitsUntilStartThenManualClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBiz := billmock.NewMockBusiness(ctrl)
	setupMockDeps(ctrl, mockBiz)

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(ActivateBillActivity)
	env.RegisterActivity(CloseBillActivity)
	env.RegisterActivity(UpdateBillTotalActivity)

	futureStart := time.Now().Add(400 * time.Millisecond)
	end := futureStart.Add(1 * time.Second)
	billID := int32(202)

	mockBiz.EXPECT().ActivateBill(gomock.Any(), billID).Return(nil).Times(1)
	mockBiz.EXPECT().CloseBill(gomock.Any(), billID, "manual").Return(nil).Times(1)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(CloseBillSignalName, CloseBillSignal{Reason: "manual"})
	}, 800*time.Millisecond)

	params := BillingPeriodWorkflowParams{BillID: billID, StartTime: futureStart, EndTime: end}
	env.ExecuteWorkflow(BillingPeriod, params)
	require.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestBillingPeriodWorkflow_AddLineItemSignalUpdatesTotal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBiz := billmock.NewMockBusiness(ctrl)
	setupMockDeps(ctrl, mockBiz)

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(ActivateBillActivity)
	env.RegisterActivity(CloseBillActivity)
	env.RegisterActivity(UpdateBillTotalActivity)

	start := time.Now().Add(-150 * time.Millisecond)
	end := time.Now().Add(1100 * time.Millisecond)
	billID := int32(303)

	mockBiz.EXPECT().ActivateBill(gomock.Any(), billID).Return(nil).Times(1)
	mockBiz.EXPECT().UpdateBillTotal(gomock.Any(), billID).Return(nil).Times(2)
	mockBiz.EXPECT().CloseBill(gomock.Any(), billID, "auto_close").Return(nil).Times(1)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(AddLineItemSignalName, AddLineItemSignal{LineItemID: 1})
	}, 120*time.Millisecond)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(AddLineItemSignalName, AddLineItemSignal{LineItemID: 2})
	}, 250*time.Millisecond)

	params := BillingPeriodWorkflowParams{BillID: billID, StartTime: start, EndTime: end}
	env.ExecuteWorkflow(BillingPeriod, params)
	require.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestBillingPeriodWorkflow_InvalidPeriodImmediateClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBiz := billmock.NewMockBusiness(ctrl)
	setupMockDeps(ctrl, mockBiz)

	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(ActivateBillActivity)
	env.RegisterActivity(CloseBillActivity)
	env.RegisterActivity(UpdateBillTotalActivity)

	start := time.Now().Add(600 * time.Millisecond)
	end := start.Add(-400 * time.Millisecond) // invalid period
	billID := int32(404)

	// Expect only close with invalid_period reason
	mockBiz.EXPECT().CloseBill(gomock.Any(), billID, "invalid_period").Return(nil).Times(1)

	params := BillingPeriodWorkflowParams{BillID: billID, StartTime: start, EndTime: end}
	env.ExecuteWorkflow(BillingPeriod, params)
	require.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())
}

func TestActivities_FailurePaths(t *testing.T) {
	testErr := errors.New("boom")

	run := func(name string, expect func(m *billmock.MockBusiness), invoke func(env *testsuite.TestActivityEnvironment) error) {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockBiz := billmock.NewMockBusiness(ctrl)
			SetActivityDependencies(mockBiz)
			t.Cleanup(func() { SetActivityDependencies(nil) })

			var ts testsuite.WorkflowTestSuite
			env := ts.NewTestActivityEnvironment()
			env.RegisterActivity(ActivateBillActivity)
			env.RegisterActivity(CloseBillActivity)
			env.RegisterActivity(UpdateBillTotalActivity)

			expect(mockBiz)
			err := invoke(env)
			// We expect an error either from the ExecuteActivity scheduling (rare) or from the future.Get.
			if err == nil {
				t.Fatalf("expected error from activity but got nil")
			}
			assert.Contains(t, err.Error(), testErr.Error())
		})
	}

	run("ActivateBillActivity failure", func(m *billmock.MockBusiness) {
		m.EXPECT().ActivateBill(gomock.Any(), int32(1)).Return(testErr).Times(1)
	}, func(env *testsuite.TestActivityEnvironment) error {
		fut, err := env.ExecuteActivity(ActivateBillActivity, int32(1))
		if err != nil { // scheduling error already contains activity err in test env sometimes
			return err
		}
		var out interface{}
		return fut.Get(&out)
	})

	run("CloseBillActivity failure", func(m *billmock.MockBusiness) {
		m.EXPECT().CloseBill(gomock.Any(), int32(1), "reason").Return(testErr).Times(1)
	}, func(env *testsuite.TestActivityEnvironment) error {
		fut, err := env.ExecuteActivity(CloseBillActivity, int32(1), "reason")
		if err != nil {
			return err
		}
		var out interface{}
		return fut.Get(&out)
	})

	run("UpdateBillTotalActivity failure", func(m *billmock.MockBusiness) {
		m.EXPECT().UpdateBillTotal(gomock.Any(), int32(1)).Return(testErr).Times(1)
	}, func(env *testsuite.TestActivityEnvironment) error {
		fut, err := env.ExecuteActivity(UpdateBillTotalActivity, int32(1))
		if err != nil {
			return err
		}
		var out interface{}
		return fut.Get(&out)
	})
}
