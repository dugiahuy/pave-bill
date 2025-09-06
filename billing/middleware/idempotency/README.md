# Simplified Idempotency Middleware

This middleware provides basic idempotency validation by extracting the `X-Idempotency-Key` header and generating a hash of the request body.

## Architecture

This middleware follows the **Option 2: Domain-Driven Idempotency** approach:

- **Middleware**: Only handles HTTP concerns (header validation, body hashing)
- **Service Layer**: Handles all business idempotency logic and caching
- **Clean Separation**: No import cycles, clear responsibilities

## What This Middleware Does

1. ✅ **Extracts** `X-Idempotency-Key` from HTTP headers
2. ✅ **Validates** that the header is present
3. ✅ **Generates** a normalized hash of the request body
4. ✅ **Passes** idempotency info to service layer via context
5. ✅ **Logs** for debugging and monitoring

## What This Middleware Does NOT Do

- ❌ Cache management (handled by service layer)
- ❌ Business logic decisions (handled by service layer)
- ❌ Database operations (handled by service layer)

## Usage

### 1. Tag your API endpoint

```go
//encore:api public path=/v1/bills method=POST tag:idempotent
func (s *Service) CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
    // Extract idempotency info from context
    if idempInfo, ok := idempotent.GetIdempotencyInfo(ctx); ok {
        // Pass to service layer for business idempotency handling
        return s.billService.CreateWithIdempotency(ctx, req, idempInfo.Key, idempInfo.BodyHash)
    }
    
    // Fallback to regular creation (shouldn't happen with middleware)
    return s.billService.Create(ctx, req)
}
```

### 2. Client Usage

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: unique-request-id-123" \
  -d '{"currency":"USD","end_time":"2024-12-01T00:00:00Z"}' \
  http://localhost:4000/v1/bills
```

## Context API

The middleware provides a simple context API:

```go
// Get idempotency information
if info, ok := idempotent.GetIdempotencyInfo(ctx); ok {
    fmt.Printf("Key: %s\n", info.Key)
    fmt.Printf("Body Hash: %s\n", info.BodyHash)
}
```

## Request Body Hashing

The middleware automatically:
- Normalizes JSON by re-marshaling (consistent key ordering)
- Generates SHA256 hash for conflict detection
- Handles empty request bodies gracefully

## Error Responses

- **Missing Header**: `400 Bad Request` - "X-Idempotency-Key header is required"

## Benefits

### ✅ **Simple & Focused**
- Single responsibility: HTTP validation only
- No complex caching logic in middleware
- Easy to understand and maintain

### ✅ **No Import Cycles**
- Clean dependency flow: HTTP → Service → Repository
- Middleware doesn't import business packages
- Service layer controls all caching decisions

### ✅ **Testable**
- Middleware can be tested independently
- Simple mocking for context data
- Clear interfaces between layers

### ✅ **Flexible**
- Service layer has full control over idempotency behavior
- Can implement different caching strategies
- Easy to add features like TTL, conflict detection

## Integration with Service Layer

The service layer receives the idempotency information and handles all business logic:

```go
func (s *BillService) CreateWithIdempotency(ctx context.Context, req *CreateBillRequest, idempotencyKey, bodyHash string) (*CreateBillResponse, error) {
    // Check cache
    if cached, found := s.idempotency.Check(ctx, idempotencyKey); found {
        return cached, nil
    }
    
    // Mark as processing
    s.idempotency.MarkProcessing(ctx, idempotencyKey, bodyHash)
    
    // Do business logic
    result, err := s.createBill(ctx, req)
    if err != nil {
        s.idempotency.MarkFailed(ctx, idempotencyKey)
        return nil, err
    }
    
    // Cache successful result
    s.idempotency.MarkCompleted(ctx, idempotencyKey, result)
    return result, nil
}
```

This approach gives you complete control over idempotency behavior while keeping the middleware simple and focused.
