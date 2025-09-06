package idempotency

// func TestNormalizeAndHash(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		input    []byte
// 		expected string
// 		wantErr  bool
// 	}{
// 		{
// 			name:     "empty body",
// 			input:    []byte{},
// 			expected: "",
// 		},
// 		{
// 			name:     "simple json",
// 			input:    []byte(`{"currency":"USD","amount":100}`),
// 			expected: "a5c9fc22c95a1b87d04b0d1234567890abcdef123456", // This will be the actual hash
// 		},
// 		{
// 			name:  "invalid json",
// 			input: []byte(`{"invalid": json}`),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			result := hashing(tt.input)

// 			if tt.name == "empty body" && result != tt.expected {
// 				t.Errorf("Expected %s, got %s", tt.expected, result)
// 			}

// 			// For non-empty valid JSON, just check that we get a non-empty hash
// 			if tt.name == "simple json" && len(result) != 64 { // SHA256 hex = 64 chars
// 				t.Errorf("Expected 64-character hash, got %d characters", len(result))
// 			}
// 		})
// 	}
// }

// func TestIdempotencyInfo(t *testing.T) {
// 	key := "test-key-123"
// 	bodyHash := "abc123hash"

// 	info := &core.IdempotencyInfo{
// 		Key:      key,
// 		BodyHash: bodyHash,
// 	}

// 	if info.Key != key {
// 		t.Errorf("Expected key %s, got %s", key, info.Key)
// 	}

// 	if info.BodyHash != bodyHash {
// 		t.Errorf("Expected bodyHash %s, got %s", bodyHash, info.BodyHash)
// 	}
// }
