# Idempotency Middleware

This middleware provides idempotency protection for Encore API endpoints using the `X-Idempotency-Key` header with business logic integration.

## Architecture

The idempotency system has two layers:

1. **Middleware Layer**: Handles HTTP-level concerns (header extraction, cache lookup, concurrent request protection)
2. **Business Logic Integration**: Allows services to control cache behavior based on business outcomes

## Usage

### 1. Tag your API endpoint

```go
//encore:api public path=/v1/bills method=POST tag:idempotent
func (s *Service) CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
    // Your implementation
}
```

### 2. Use idempotency context in business logic

```go
import "encore.app/billing/middleware/idempotent"

func (s *service) Create(ctx context.Context, bill *model.Bill) (*model.Bill, error) {
    // Your business logic here
    result, err := s.billRepo.CreateBill(ctx, params)
    if err != nil {
        // Mark cache as failed to allow retry
        if cacheErr := idempotent.MarkCacheAsFailed(ctx); cacheErr != nil {
            log.Error("Failed to mark cache as failed", "error", cacheErr)
        }
        return nil, err
    }
    
    // Update cache with successful result
    if resultBytes, err := json.Marshal(result); err == nil {
        if cacheErr := idempotent.UpdateCacheOnSuccess(ctx, resultBytes); cacheErr != nil {
            log.Error("Failed to update cache", "error", cacheErr)
            // Don't fail the request if cache update fails
        }
    }
    
    return result, nil
}
```

### 3. Client Usage

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: unique-request-id-123" \
  -d '{"currency":"USD","end_time":"2024-12-01T00:00:00Z"}' \
  http://localhost:4000/v1/bills
```

## How it works

### Request Flow

1. **Middleware receives request** with `X-Idempotency-Key` header
2. **Cache lookup**: Check if key already exists
   - If processing: Return `409 Conflict`
   - If completed: Return cached response
   - If not found: Continue to step 3
3. **Mark as processing**: Set cache status to "processing"
4. **Add context**: Inject `IdempotencyContext` into request context
5. **Execute handler**: Call your API handler and business logic
6. **Business logic decides**:
   - Success: Calls `UpdateCacheOnSuccess()` to cache result
   - Failure: Calls `MarkCacheAsFailed()` to allow retry

### Context API

The middleware provides a context-based API for business logic:

```go
// Check if request has idempotency context
if idempCtx, ok := idempotent.GetIdempotencyContext(ctx); ok {
    fmt.Printf("Idempotency key: %s", idempCtx.Key)
    fmt.Printf("From cache: %v", idempCtx.IsFromCache)
}

// Update cache on successful business operation
idempotent.UpdateCacheOnSuccess(ctx, responseBytes)

// Mark cache as failed to allow retry
idempotent.MarkCacheAsFailed(ctx)
```

## Benefits of This Architecture

### ✅ **Separation of Concerns**
- Middleware handles HTTP/cache concerns
- Business logic controls cache lifecycle
- Clear boundaries between layers

### ✅ **Business Logic Control**
- Services decide when to cache vs. retry
- Handles partial failures gracefully
- Supports complex business rules

### ✅ **Reliability**
- Failed requests don't get cached as successful
- Database rollbacks properly clear cache
- Concurrent request protection

### ✅ **Testability**
- Business logic can be tested without middleware
- Context can be mocked for testing
- Clear interfaces between components

## Cache States

- **processing**: Request is currently being processed
- **completed**: Request completed successfully, response cached
- **failed/deleted**: Request failed, cache entry removed to allow retry

## Error Responses

- **Missing Header**: `400 Bad Request` - "X-Idempotency-Key header is required"
- **Concurrent Processing**: `409 Conflict` - "Request is already being processed"

## Production Considerations

1. **Cache TTL**: 24-hour default expiry for completed requests
2. **Monitoring**: Monitor cache hit rates and business logic outcomes
3. **Error Handling**: Always handle cache errors gracefully in business logic
4. **Rollback Strategy**: Use `MarkCacheAsFailed()` in transaction rollback handlers
