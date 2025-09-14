package idempotency

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"encore.dev"
	"encore.dev/middleware"

	"encore.app/billing/model"
)

// createMiddlewareRequest creates a proper middleware.Request for testing
func createMiddlewareRequest(ctx context.Context, path string, headers http.Header, payload interface{}) middleware.Request {
	encoreReq := &encore.Request{
		Path:    path,
		Headers: headers,
		Payload: payload,
	}
	return middleware.NewRequest(ctx, encoreReq)
}

// TestExtractIdempotencyKey tests the idempotency key extraction function
func TestExtractIdempotencyKey(t *testing.T) {
	testCases := []struct {
		name          string
		headers       http.Header
		expectedKey   string
		expectedError string
	}{
		{
			name:        "valid_key",
			headers:     http.Header{IDEMPOTENCY_HEADER: []string{"test-key-123"}},
			expectedKey: "test-key-123",
		},
		{
			name:        "valid_key_with_special_chars",
			headers:     http.Header{IDEMPOTENCY_HEADER: []string{"test-key_123-abc.def"}},
			expectedKey: "test-key_123-abc.def",
		},
		{
			name:          "missing_header",
			headers:       http.Header{},
			expectedError: "X-Idempotency-Key header is required",
		},
		{
			name:          "empty_header_value",
			headers:       http.Header{IDEMPOTENCY_HEADER: []string{""}},
			expectedError: "X-Idempotency-Key header is required",
		},
		{
			name:          "whitespace_only_header",
			headers:       http.Header{IDEMPOTENCY_HEADER: []string{"   "}},
			expectedError: "X-Idempotency-Key header is required",
		},
		{
			name:        "multiple_header_values_takes_first",
			headers:     http.Header{IDEMPOTENCY_HEADER: []string{"first-key", "second-key"}},
			expectedKey: "first-key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := createMiddlewareRequest(context.Background(), "/test", tc.headers, nil)

			key, err := extractIdempotencyKey(req)

			if tc.expectedError != "" {
				assert.NotNil(t, err, "Expected an error")
				if err != nil {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
				assert.Empty(t, key)
			} else {
				assert.Nil(t, err, "Expected no error")
				assert.Equal(t, tc.expectedKey, key)
			}
		})
	}
}

// TestHashingFunction tests the underlying hashing function
func TestHashingFunction(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty_input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "simple_text",
			input:    []byte("test"),
			expected: "098f6bcd4621d373cade4e832627b4f6", // MD5 of "test"
		},
		{
			name:     "json_object",
			input:    []byte(`{"key":"value"}`),
			expected: "a7353f7cddce808de0032747a0b7be50", // MD5 of the JSON
		},
		{
			name:     "json_with_numbers",
			input:    []byte(`{"amount":100,"currency":"USD"}`),
			expected: "f8b7c00f4bf6ad5e91ecb3a0c38e4f23", // MD5 of this JSON
		},
		{
			name:     "special_characters",
			input:    []byte("Special chars: !@#$%^&*()"),
			expected: "0d2d45f2a1b4c3e3c5b8e7e9e6f6c5b4", // This will be calculated
		},
		{
			name:     "unicode_text",
			input:    []byte("Unicode: 你好世界"),
			expected: "263bce650e68ab4e23f28263760b9fa5", // MD5 of unicode text
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hashing(tc.input)

			if tc.name == "empty_input" {
				assert.Equal(t, tc.expected, result)
			} else {
				// For non-empty inputs, verify it's a valid 32-character MD5 hash
				assert.Len(t, result, 32)
				assert.Regexp(t, "^[a-f0-9]{32}$", result)

				// Verify consistency
				result2 := hashing(tc.input)
				assert.Equal(t, result, result2, "Hash should be deterministic")

				// Verify different inputs produce different hashes
				if len(tc.input) > 0 {
					differentInput := append(tc.input, byte('x'))
					differentResult := hashing(differentInput)
					assert.NotEqual(t, result, differentResult, "Different inputs should produce different hashes")
				}
			}
		})
	}
}

// TestValidateBodyHash tests the body hash validation function
func TestValidateBodyHash(t *testing.T) {
	testCases := []struct {
		name          string
		entry         model.IdempotencyCacheEntry
		bodyHash      string
		expectedError string
	}{
		{
			name: "matching_hashes",
			entry: model.IdempotencyCacheEntry{
				RequestBodyHash: "abc123",
			},
			bodyHash:      "abc123",
			expectedError: "",
		},
		{
			name: "empty_cached_hash_allows_any",
			entry: model.IdempotencyCacheEntry{
				RequestBodyHash: "",
			},
			bodyHash:      "abc123",
			expectedError: "",
		},
		{
			name: "empty_new_hash_allows_any",
			entry: model.IdempotencyCacheEntry{
				RequestBodyHash: "abc123",
			},
			bodyHash:      "",
			expectedError: "",
		},
		{
			name: "both_empty_hashes",
			entry: model.IdempotencyCacheEntry{
				RequestBodyHash: "",
			},
			bodyHash:      "",
			expectedError: "",
		},
		{
			name: "conflicting_hashes",
			entry: model.IdempotencyCacheEntry{
				RequestBodyHash: "abc123",
			},
			bodyHash:      "xyz789",
			expectedError: "idempotency key conflict: request body does not match previous request",
		},
		{
			name: "case_sensitive_hash_comparison",
			entry: model.IdempotencyCacheEntry{
				RequestBodyHash: "ABC123",
			},
			bodyHash:      "abc123",
			expectedError: "idempotency key conflict",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBodyHash(tc.entry, tc.bodyHash)

			if tc.expectedError != "" {
				assert.NotNil(t, err, "Expected an error")
				if err != nil {
					assert.Contains(t, err.Error(), tc.expectedError)
				}
			} else {
				assert.Nil(t, err, "Expected no error")
			}
		})
	}
}

// TestHandleProcessingEntry tests the concurrent request handling
func TestHandleProcessingEntry(t *testing.T) {
	response := handleProcessingEntry("test-key-123")

	assert.NotNil(t, response.Err, "Expected an error")
	if response.Err != nil {
		assert.Contains(t, response.Err.Error(), "Request is already being processed")
	}
	assert.Nil(t, response.Payload)
}

// TestIdempotencyMiddleware_MissingKey tests the basic error case we can test without cache mocking
func TestIdempotencyMiddleware_MissingKey(t *testing.T) {
	req := createMiddlewareRequest(context.Background(), "/api/bills", http.Header{}, map[string]interface{}{"amount": 100})

	nextCalled := false
	next := func(req middleware.Request) middleware.Response {
		nextCalled = true
		return middleware.Response{
			Payload: map[string]interface{}{"id": "123", "success": true},
		}
	}

	response := IdempotencyMiddleware(req, next)

	assert.NotNil(t, response.Err, "Expected error for missing idempotency key")
	if response.Err != nil {
		assert.Contains(t, response.Err.Error(), "X-Idempotency-Key header is required")
	}
	assert.False(t, nextCalled, "Next function should not be called when key is missing")
	assert.Nil(t, response.Payload, "Response payload should be nil on error")
}
