# Idempotency Flow Diagram - Swimlanes.io Format

```
title: Billing Service - Idempotency Middleware Flow

Client -> Middleware: HTTP Request with X-Idempotency-Key header
Middleware -> Middleware: Construct idempotency key from header and Generate body hash (MD5)
Middleware -> Redis: Check cache for existing entry
group: Cache Miss (New Request)
  Middleware -> Redis: Mark idempotency request as **"processing"**
  Middleware -> API Layer: Forward request to next middleware/handler
  API Layer -> API Layer: Process business logic
  group: API Success
    API Layer -> Middleware: Return success response
    Middleware -> Redis: Store **"completed"** response with body hash
    Middleware -> Client: Return success response
  else: API Error
    API Layer -> Middleware: Return error response
    Middleware -> Redis: Delete cache entry
    Middleware -> Client: Return error response
  end
else: Cache Hit (Existing Entry)
  Redis -> Middleware: Return cached entry
  group: If Entry is **"processing"**
      Middleware -> Client: Return 409 Conflict "Request is being processed"
    else: If Entry is **"completed"**
      Middleware -> Middleware: Validate body hash matches
      group: Body hash matches
        Middleware -> Client: Return cached response
      else: Body hash differs
        Middleware -> Client: Return 409 Conflict "Idempotency key reused"
      end
    end
  end
```

## Key Components:

- **Client**: Sends HTTP requests with X-Idempotency-Key header
- **Middleware**: IdempotencyMiddleware that manages the idempotency logic
- **Redis**: IdempotencyCache that stores request state and responses
- **API Layer**: The actual business logic (CreateBill, AddLineItem, CloseBill APIs)

## Flow States:

1. **New Request**: Cache miss → Mark as processing → Execute API → Cache result
2. **Duplicate Request**: Cache hit with completed status → Return cached response
3. **Concurrent Request**: Cache hit with processing status → Wait and retry
4. **Invalid Reuse**: Same key with different request body → Return conflict error

## Error Scenarios:

- Missing idempotency key → 400 Invalid Argument
- Key reused with different body → 409 Conflict  
- Request still processing → 409 Conflict
- Cache failures → 500 Internal Error