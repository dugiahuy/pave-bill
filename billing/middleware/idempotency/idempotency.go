package idempotency

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/middleware"
	"encore.dev/rlog"
	"encore.dev/storage/cache"

	"encore.app/billing/model"
)

var (
	IDEMPOTENCY_HEADER = "X-Idempotency-Key"
)

//encore:middleware target=tag:idempotency
func IdempotencyMiddleware(req middleware.Request, next middleware.Next) middleware.Response {
	idempotencyKey, err := extractIdempotencyKey(req)
	if err != nil {
		return middleware.Response{Err: err}
	}

	bodyHash := generateBodyHash(req)

	// Create cache key
	cacheKey := model.IdempotencyKey{
		Resource: req.Data().Path,
		Key:      idempotencyKey,
	}

	// Check existing cache entry
	entry, cacheErr := IdempotencyCache.Get(req.Context(), cacheKey)
	if cacheErr != nil {
		// Handle cache miss - process new request
		if errors.Is(cacheErr, cache.Miss) {
			if err := markAsProcessing(req.Context(), cacheKey); err != nil {
				return middleware.Response{Err: err}
			}

			response := next(req)

			if response.Err != nil {
				deleteCacheEntry(req.Context(), cacheKey)
			} else {
				markAsCompleted(req.Context(), cacheKey, bodyHash, idempotencyKey, response)
			}

			return response
		}

		return middleware.Response{
			Err: &errs.Error{Code: errs.Internal, Message: "Failed to check idempotency"},
		}
	}

	// Handle existing cache entry
	return handleExistingEntry(req, next, entry, bodyHash, idempotencyKey)
}

// extractIdempotencyKey extracts and validates the idempotency key from headers
func extractIdempotencyKey(req middleware.Request) (string, *errs.Error) {
	var idempotencyKey string
	if headers := req.Data().Headers; headers != nil {
		if headerVal := headers.Get(IDEMPOTENCY_HEADER); headerVal != "" {
			idempotencyKey = headerVal
		}
	}

	if len(idempotencyKey) == 0 {
		return "", &errs.Error{Code: errs.InvalidArgument, Message: "X-Idempotency-Key header is required"}
	}

	return idempotencyKey, nil
}

// generateBodyHash creates a hash of the request body for conflict detection
func generateBodyHash(req middleware.Request) string {
	var bodyHash string
	if payload := req.Data().Payload; payload != nil {
		if bodyBytes, err := json.Marshal(payload); err != nil {
			rlog.Error("Failed to marshal request body", "error", err)
		} else {
			bodyHash = hashing(bodyBytes)
		}
	}
	return bodyHash
}

// handleExistingEntry handles cases where a cache entry already exists
func handleExistingEntry(req middleware.Request, next middleware.Next, entry model.IdempotencyCacheEntry, bodyHash, idempotencyKey string) middleware.Response {
	// Validate body hash for conflict detection
	if err := validateBodyHash(entry, bodyHash); err != nil {
		return middleware.Response{Err: err}
	}

	// Handle entry based on status
	switch entry.Status {
	case "processing":
		return handleProcessingEntry(idempotencyKey)
	case "completed":
		return handleCompletedEntry(req, next, entry, idempotencyKey)
	default:
		rlog.Warn("Unknown cache entry status, processing as new request", "key", idempotencyKey, "status", entry.Status)
		return next(req)
	}
}

// validateBodyHash checks for conflicts in request body hash
func validateBodyHash(entry model.IdempotencyCacheEntry, bodyHash string) *errs.Error {
	if bodyHash != "" && entry.RequestBodyHash != "" && bodyHash != entry.RequestBodyHash {
		return &errs.Error{Code: errs.InvalidArgument, Message: "idempotency key conflict: request body does not match previous request"}
	}
	return nil
}

// handleProcessingEntry handles concurrent request detection
func handleProcessingEntry(idempotencyKey string) middleware.Response {
	rlog.Info("Concurrent request detected", "key", idempotencyKey)
	return middleware.Response{
		Err: &errs.Error{Code: errs.Aborted, Message: "Request is already being processed."},
	}
}

// handleCompletedEntry handles returning cached responses
func handleCompletedEntry(req middleware.Request, next middleware.Next, entry model.IdempotencyCacheEntry, idempotencyKey string) middleware.Response {
	if len(entry.Response) > 0 {
		rlog.Info("Returning cached response", "key", idempotencyKey)

		// Get the response type from the API metadata
		responseType := req.Data().API.ResponseType
		if responseType != nil {
			// Create a new instance of the response type
			responseValue := reflect.New(responseType.Elem()).Interface()

			// Unmarshal the cached JSON into the correct type
			err := json.Unmarshal(entry.Response, responseValue)
			if err == nil {
				return middleware.Response{Payload: responseValue}
			}
			rlog.Error("Failed to unmarshal cached response into correct type", "error", err, "key", idempotencyKey)
		}
	}

	// Fallback: if cached response is corrupted, treat as new request
	return next(req)
}

// markAsProcessing marks a request as currently being processed
func markAsProcessing(ctx context.Context, cacheKey model.IdempotencyKey) *errs.Error {
	if err := IdempotencyCache.Set(ctx, cacheKey, model.IdempotencyCacheEntry{
		Status:    "processing",
		CreatedAt: time.Now(),
	}); err != nil {
		rlog.Error("Failed to mark request as processing", "error", err)
		return &errs.Error{Code: errs.Internal, Message: "Failed to mark request as processing"}
	}
	return nil
}

// deleteCacheEntry removes processing entry to allow retry
func deleteCacheEntry(ctx context.Context, cacheKey model.IdempotencyKey) {
	if _, deleteErr := IdempotencyCache.Delete(ctx, cacheKey); deleteErr != nil {
		rlog.Error("Failed to clear failed request from cache", "error", deleteErr)
	}
}

// markAsCompleted caches the successful response
func markAsCompleted(ctx context.Context, cacheKey model.IdempotencyKey, bodyHash, idempotencyKey string, response middleware.Response) {
	completedEntry := model.IdempotencyCacheEntry{
		Status:          "completed",
		RequestBodyHash: bodyHash,
		UpdatedAt:       time.Now(),
	}

	// Only cache the payload as JSON, not the entire middleware.Response
	if response.Payload != nil {
		payloadBytes, err := json.Marshal(response.Payload)
		if err != nil {
			rlog.Error("Failed to marshal response payload for caching", "error", err)
			return
		}
		completedEntry.Response = payloadBytes
	}

	if setErr := IdempotencyCache.Set(ctx, cacheKey, completedEntry); setErr != nil {
		rlog.Error("Failed to cache successful response", "error", setErr)
	}

	rlog.Debug("Request completed and response cached", "key", idempotencyKey)
}

// hashing creates a stable hash of the JSON request body
func hashing(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	hash := md5.New()
	hash.Write(body)
	return hex.EncodeToString(hash.Sum(nil))
}
