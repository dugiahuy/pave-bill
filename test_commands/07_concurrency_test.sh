#!/bin/bash
# Test concurrent AddLineItem and CloseBill operations
# Creates fresh bills for each test case

BASE_URL="http://localhost:4000"

# Helper function to create an active bill
create_active_bill() {
  local test_name="$1"
  echo "📋 Creating new active bill for $test_name..." >&2

  local response=$(curl -s -X POST "$BASE_URL/v1/bills" \
    -H 'Content-Type: application/json' \
    -H "X-Idempotency-Key: concurrency-$test_name-$(date +%s)-$RANDOM" \
    -d '{
      "currency": "USD",
      "start_time": null,
      "end_time": "'$(date -v+1d -u +"%Y-%m-%dT%H:%M:%SZ")'"
    }')

  local bill_id=$(echo "$response" | jq -r '.bill.id // empty')
  local status=$(echo "$response" | jq -r '.bill.status // "unknown"')

  if [ -z "$bill_id" ] || [ "$bill_id" = "null" ]; then
    echo "❌ Failed to create bill for $test_name" >&2
    echo "Response: $response" >&2
    return 1
  fi

  echo "✅ Created bill ID: $bill_id (status: $status)" >&2

  # Wait for bill to become active if it's pending
  if [ "$status" = "pending" ]; then
    echo "⏳ Waiting for bill to become active..." >&2
    sleep 3

    # Check status again
    local status_response=$(curl -s -X GET "$BASE_URL/v1/bills/$bill_id" -H 'Content-Type: application/json')
    local new_status=$(echo "$status_response" | jq -r '.bill.status // "unknown"')
    echo "📊 Bill status after wait: $new_status" >&2
  fi

  echo "$bill_id"
}

# Cleanup function
cleanup_temp_files() {
  rm -f /tmp/concurrent_*.json /tmp/rapid_*.json 2>/dev/null
}

echo "==========================================="
echo "      CONCURRENCY TEST SUITE"
echo "==========================================="
echo "Testing concurrent operations with fresh bills"
echo "This tests timeout handling for row-level locks"
echo ""

# Test 1: Simple concurrent test
echo "=== Test 1: Concurrent AddLineItem vs CloseBill ==="
cleanup_temp_files

# Create fresh bill for test 1
BILL_ID_1=$(create_active_bill "Test1")
if [ $? -ne 0 ]; then
  echo "❌ Skipping Test 1 due to bill creation failure"
else
  echo "DEBUG: Captured Bill ID 1: '$BILL_ID_1'"
  echo "🎯 Testing concurrent operations on Bill ID: $BILL_ID_1"
  echo ""

  # Start both operations simultaneously
  (
    echo "🟢 Starting AddLineItem..."
    curl -X POST "$BASE_URL/v1/bills/$BILL_ID_1/line_items" \
      -H 'Content-Type: application/json' \
      -H "X-Idempotency-Key: concurrent-add-$(date +%s)-$RANDOM" \
      -d '{
        "currency": "USD",
        "amount_cents": 1000,
        "description": "Concurrent Test Item",
        "reference_id": "concurrent-ref"
      }' \
      -w "\nAdd Status: %{http_code}\nTime: %{time_total}s\n" \
      -o /tmp/concurrent_add_response.json
    echo "🟢 AddLineItem completed"
  ) &

  (
    echo "🔴 Starting CloseBill..."
    curl -X POST "$BASE_URL/v1/bills/$BILL_ID_1/close" \
      -H 'Content-Type: application/json' \
      -H "X-Idempotency-Key: concurrent-close-$(date +%s)-$RANDOM" \
      -d '{"reason": "concurrent_test"}' \
      -w "\nClose Status: %{http_code}\nTime: %{time_total}s\n" \
      -o /tmp/concurrent_close_response.json
    echo "🔴 CloseBill completed"
  ) &

  # Wait for both to complete
  wait

  echo ""
  echo "=== Test 1 Results ==="
  echo "AddLineItem Response:"
  cat /tmp/concurrent_add_response.json | jq '.' 2>/dev/null || cat /tmp/concurrent_add_response.json
  echo ""

  echo "CloseBill Response:"
  cat /tmp/concurrent_close_response.json | jq '.' 2>/dev/null || cat /tmp/concurrent_close_response.json
  echo ""

  echo "✅ Expected behavior:"
  echo "- One operation succeeds quickly"
  echo "- Other operation either:"
  echo "  a) Succeeds after waiting (if first operation was fast)"
  echo "  b) Times out after 5 seconds with timeout error"
  echo "  c) Gets logical error (e.g., 'bill not active')"
  echo ""
fi

# Test 2: Rapid fire test
echo "=== Test 2: Rapid Fire Test ==="
cleanup_temp_files

# Create fresh bill for test 2
BILL_ID_2=$(create_active_bill "Test2-RapidFire")
if [ $? -ne 0 ]; then
  echo "❌ Skipping Test 2 due to bill creation failure"
else
  echo "DEBUG: Captured Bill ID 2: '$BILL_ID_2'"
  echo "🎯 Testing rapid fire operations on Bill ID: $BILL_ID_2"
  echo "Sending 5 rapid requests to test lock contention..."
  echo ""

  for i in {1..5}; do
    if [ $((i % 2)) -eq 1 ]; then
      # Add line item
      (
        echo "🟢 Rapid AddLineItem $i"
        curl -s -X POST "$BASE_URL/v1/bills/$BILL_ID_2/line_items" \
          -H 'Content-Type: application/json' \
          -H "X-Idempotency-Key: rapid-add-$i-$(date +%s)-$RANDOM" \
          -d "{
            \"currency\": \"USD\",
            \"amount_cents\": $((100 * i)),
            \"description\": \"Rapid Item $i\",
            \"reference_id\": \"rapid-$i\"
          }" \
          -w "Add $i Status: %{http_code} Time: %{time_total}s\n" \
          -o "/tmp/rapid_add_${i}.json"
      ) &
    else
      # Close bill
      (
        echo "🔴 Rapid CloseBill $i"
        curl -s -X POST "$BASE_URL/v1/bills/$BILL_ID_2/close" \
          -H 'Content-Type: application/json' \
          -H "X-Idempotency-Key: rapid-close-$i-$(date +%s)-$RANDOM" \
          -d "{\"reason\": \"rapid_test_$i\"}" \
          -w "Close $i Status: %{http_code} Time: %{time_total}s\n" \
          -o "/tmp/rapid_close_${i}.json"
      ) &
    fi

    # Very small delay
    sleep 0.1
  done

  wait

  echo ""
  echo "=== Test 2 Rapid Fire Results ==="
  for i in {1..5}; do
    if [ $((i % 2)) -eq 1 ]; then
      echo "Rapid Add $i:"
      cat "/tmp/rapid_add_${i}.json" | jq '.line_item.id // .error // .' 2>/dev/null || echo "Failed to parse"
    else
      echo "Rapid Close $i:"
      cat "/tmp/rapid_close_${i}.json" | jq '.error // .' 2>/dev/null || echo "Failed to parse"
    fi
  done
fi

echo ""
echo "==========================================="
echo "           CONCURRENCY TEST COMPLETE"
echo "==========================================="
echo ""
echo "🔍 What to analyze in the results:"
echo "1. No requests should hang indefinitely (all should complete within 6s)"
echo "2. Should see timeout errors (ResourceExhausted) if lock contention occurs"
echo "3. Some operations should succeed, others should fail with clear errors"
echo "4. Response times should be reasonable (< 6 seconds max)"
echo ""
echo "💡 Good error examples:"
echo "- 'operation timed out - bill may be locked by another operation'"
echo "- 'bill is not in active state for adding line items'"
echo "- 'bill must be in active status to transition to closing'"
echo ""
echo "📋 Test Summary:"
echo "- Test 1 Bill ID: ${BILL_ID_1:-'creation failed'}"
echo "- Test 2 Bill ID: ${BILL_ID_2:-'creation failed'}"
echo ""

# Final cleanup
echo "🧹 Cleaning up temporary files..."
cleanup_temp_files
echo "✅ Concurrency tests completed!"