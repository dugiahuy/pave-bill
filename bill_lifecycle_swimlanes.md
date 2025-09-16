# Bill Lifecycle - Swimlanes.io Syntax

```
title: Billing Service - Complete Lifecycle Flow

note over Client, Temporal Activities: Bill Creation Flow
Client -> API Layer: POST /v1/bills (CreateBill)
API Layer -> Database: Store bill (status: PENDING)
API Layer -> Temporal Workflow: Start BillingPeriodWorkflow
API Layer -> Client: Return bill info

note over Client, Temporal Activities: Workflow Activation
Temporal Workflow -> Temporal Workflow: Wait until start_time
Temporal Workflow -> Temporal Activities: Execute ActivateBillActivity
Temporal Activities -> Database: Update status to ACTIVE

note over Client, Temporal Activities: Line Item Addition Flow
Client -> API Layer: POST /v1/bills/:id/line_items
API Layer -> Database: Store line item with row locking
API Layer -> Temporal Workflow: Signal AddLineItem (async)
Temporal Workflow -> Temporal Activities: Execute UpdateBillTotalActivity
Temporal Activities -> Database: Recalculate bill total
API Layer -> Client: Return line item info

note over Client, Temporal Activities: Manual Close Flow
Client -> API Layer: POST /v1/bills/:id/close
API Layer -> Database: Update status to CLOSED (with locking)
API Layer -> Temporal Workflow: Terminate workflow (async)
API Layer -> Client: Return closed bill

note over Client, Temporal Activities: Auto Close Flow (Alternative)
Temporal Workflow -> Temporal Workflow: Timer reaches end_time
Temporal Workflow -> Temporal Activities: Execute CloseBillActivity
Temporal Activities -> Database: Transition ACTIVE → CLOSING → CLOSED

note over Client, Temporal Activities: Error Scenarios
alt: Close Bill Error During Total Calculation
  Temporal Activities -> Database: Update status to ATTENTION_REQUIRED
  Temporal Activities -> Temporal Activities: Log error details
else: Concurrent Lock Timeout
  API Layer -> Client: Return 409 ResourceExhausted "operation timed out"
else: Idempotency Key Reuse
  API Layer -> Client: Return 409 Conflict "key reused with different body"
end

note over Client, Temporal Activities: Get Bill Status (Anytime)
Client -> API Layer: GET /v1/bills/:id
API Layer -> Database: Query bill with current status
API Layer -> Client: Return bill info with status
```

## Key Components:

- **Client**: External systems making HTTP requests
- **API Layer**: Encore API endpoints with middleware (idempotency, validation)
- **Database**: PostgreSQL with row-level locking and state management
- **Temporal Workflow**: BillingPeriodWorkflow managing bill lifecycle
- **Temporal Activities**: Individual tasks (ActivateBill, CloseBill, UpdateBillTotal)

## Flow Highlights:

1. **Async Operations**: Workflow signals and termination don't block API responses
2. **Row-Level Locking**: Prevents race conditions during concurrent operations
3. **State Transitions**: PENDING → ACTIVE → CLOSING → CLOSED with proper validation
4. **Error Handling**: Timeout errors, conflict resolution, and attention-required states
5. **Idempotency**: All operations support idempotency keys with body hash validation