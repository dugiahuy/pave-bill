# Bill Lifecycle Diagram - README Documentation

## Bill State Transitions and API Flow

```mermaid
graph TD
    A[Client] --> B[POST /v1/bills<br/>CreateBill API]
    B --> C{Bill Created}
    C --> D[Bill Status: PENDING<br/>Workflow: Started]
    
    D --> E[Temporal Workflow<br/>Waits for start_time]
    E --> F[ActivateBillActivity<br/>Triggered at start_time]
    F --> G[Bill Status: ACTIVE<br/>Ready for line items]
    
    G --> H[POST /v1/bills/:id/line_items<br/>AddLineItem API]
    H --> I[Line Item Added<br/>Signal Workflow]
    I --> J[UpdateBillTotalActivity<br/>Recalculate totals]
    J --> G
    
    G --> K{End Time Reached?}
    K -->|Yes| L[CloseBillActivity<br/>Auto-close]
    K -->|No| G
    
    G --> M[POST /v1/bills/:id/close<br/>Manual Close API]
    M --> N[Bill Status: CLOSING<br/>Calculate final total]
    N --> O[Bill Status: CLOSED<br/>Terminate Workflow]
    
    L --> P[Bill Status: CLOSING<br/>Calculate final total]
    P --> Q[Bill Status: CLOSED<br/>Workflow Complete]
    
    O --> R[GET /v1/bills/:id<br/>Get Bill Status]
    Q --> R
    R --> S[Return Bill Info<br/>with Final Status]
    
    %% Error States
    N -->|Error| T[Bill Status: ATTENTION_REQUIRED<br/>Manual Review Needed]
    P -->|Error| T
    
    %% Styling
    classDef pending fill:#fff2cc,stroke:#d6b656
    classDef active fill:#d5e8d4,stroke:#82b366
    classDef closing fill:#ffe6cc,stroke:#d79b00
    classDef closed fill:#f8cecc,stroke:#b85450
    classDef error fill:#ffcccc,stroke:#ff0000
    classDef api fill:#e1d5e7,stroke:#9673a6
    classDef workflow fill:#dae8fc,stroke:#6c8ebf
    
    class D pending
    class G active
    class N,P closing
    class O,Q closed
    class T error
    class B,H,M,R api
    class E,F,I,J,L workflow
```

## Bill States and Transitions

| State | Description | Possible Transitions | Trigger |
|-------|-------------|---------------------|---------|
| **PENDING** | Bill created, waiting for start_time | → ACTIVE<br/>→ CLOSED | Workflow timer<br/>Manual close |
| **ACTIVE** | Bill is active, accepting line items | → CLOSING<br/>→ CLOSED | Auto-close timer<br/>Manual close |
| **CLOSING** | Bill being finalized, calculating totals | → CLOSED<br/>→ ATTENTION_REQUIRED | Success<br/>Error in processing |
| **CLOSED** | Bill finalized, no more changes | _(Terminal state)_ | N/A |
| **ATTENTION_REQUIRED** | Error state, needs manual review | → CLOSED | Manual intervention |

## API Endpoints and Their Role

### 1. **CreateBill API** - `POST /v1/bills`
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │───▶│ Create Bill  │───▶│   Temporal  │
│             │    │   Service    │    │  Workflow   │
└─────────────┘    └──────────────┘    └─────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │ Bill Status: │
                   │   PENDING    │
                   └──────────────┘
```

### 2. **AddLineItem API** - `POST /v1/bills/:id/line_items`
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │───▶│ Add Line Item│───▶│   Signal    │
│             │    │   Service    │    │  Workflow   │
└─────────────┘    └──────────────┘    └─────────────┘
                           │                    │
                           ▼                    ▼
                   ┌──────────────┐    ┌─────────────┐
                   │  Line Item   │    │Update Total │
                   │    Stored    │    │  Activity   │
                   └──────────────┘    └─────────────┘
```

### 3. **CloseBill API** - `POST /v1/bills/:id/close`
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │───▶│  Close Bill  │───▶│ Terminate   │
│             │    │   Service    │    │  Workflow   │
└─────────────┘    └──────────────┘    └─────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │ Bill Status: │
                   │   CLOSED     │
                   └──────────────┘
```

### 4. **GetBill API** - `GET /v1/bills/:id`
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │───▶│   Get Bill   │───▶│  Database   │
│             │    │   Service    │    │    Query    │
└─────────────┘    └──────────────┘    └─────────────┘
                           │
                           ▼
                   ┌──────────────┐
                   │ Return Bill  │
                   │ with Status  │
                   └──────────────┘
```

## Workflow Integration

```mermaid
sequenceDiagram
    participant C as Client
    participant API as API Layer
    participant DB as Database
    participant TW as Temporal Workflow
    participant TA as Temporal Activities
    
    Note over C,TA: Bill Creation Flow
    C->>API: POST /v1/bills (CreateBill)
    API->>DB: Store bill (status: PENDING)
    API->>TW: Start BillingPeriodWorkflow
    API->>C: Return bill info
    
    Note over C,TA: Workflow Activation
    TW->>TW: Wait until start_time
    TW->>TA: Execute ActivateBillActivity
    TA->>DB: Update status to ACTIVE
    
    Note over C,TA: Line Item Addition
    C->>API: POST /v1/bills/:id/line_items
    API->>DB: Store line item
    API->>TW: Signal AddLineItem
    TW->>TA: Execute UpdateBillTotalActivity
    TA->>DB: Recalculate bill total
    API->>C: Return line item info
    
    Note over C,TA: Manual Close
    C->>API: POST /v1/bills/:id/close
    API->>DB: Update status to CLOSED
    API->>TW: Terminate workflow
    API->>C: Return closed bill
    
    Note over C,TA: Auto Close (Alternative)
    TW->>TW: Timer reaches end_time
    TW->>TA: Execute CloseBillActivity
    TA->>DB: Update status to CLOSED
```

## Key Features

### 🔒 **Concurrency Control**
- Row-level locking prevents race conditions
- 5-second timeout prevents hanging requests
- Proper error handling for lock conflicts

### 🔄 **Idempotency**
- All APIs support idempotency keys
- Duplicate requests return cached responses
- Body hash validation prevents key reuse

### ⚡ **Async Processing**
- Workflow signals don't block API responses
- Background total recalculations
- Non-blocking workflow termination

### 🛡️ **Error Handling**
- Timeout errors with clear messages
- Graceful degradation on failures
- ATTENTION_REQUIRED state for manual review

## Testing the Lifecycle

1. **Create a bill**: `./test_commands/01_create_bill.sh`
2. **Add line items**: `./test_commands/02_add_line_items.sh <bill_id>`
3. **Check bill status**: `curl GET /v1/bills/<bill_id>`
4. **Close manually**: `./test_commands/05_close_bill.sh <bill_id>`
5. **Test concurrency**: `./test_commands/07_concurrency_test.sh <bill_id>`