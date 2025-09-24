#!/bin/bash
# Test 5: Error handling tests

echo "=== Error Test 1: Invalid Bill ID ==="
curl -X POST 'http://localhost:4000/v1/bills/99999/line_items' \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: error-test-'$(date +%s)-$RANDOM'' \
  -d '{
    "currency": "USD",
    "amount_cents": 1000,
    "description": "Test",
    "reference_id": "test"
  }' \
  -w "Status: %{http_code}\n"

echo ""
echo "=== Error Test 2: Missing Idempotency Key ==="
curl -X POST 'http://localhost:4000/v1/bills' \
  -H 'Content-Type: application/json' \
  -d '{
    "currency": "USD",
    "start_time": null,
    "end_time": "'$(date -v+1d -u +"%Y-%m-%dT%H:%M:%SZ")'"
  }' \
  -w "Status: %{http_code}\n"

echo ""
echo "=== Error Test 3: Invalid Currency ==="
curl -X POST 'http://localhost:4000/v1/bills' \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: error-currency-'$(date +%s)-$RANDOM'' \
  -d '{
    "currency": "INVALID",
    "start_time": null,
    "end_time": "'$(date -v+1d -u +"%Y-%m-%dT%H:%M:%SZ")'"
  }' \
  -w "Status: %{http_code}\n"

echo ""
echo "=== Error Test 4: End Time Before Start Time ==="
FUTURE_START=$(date -v+2d -u +"%Y-%m-%dT%H:%M:%SZ")
PAST_END=$(date -v+1d -u +"%Y-%m-%dT%H:%M:%SZ")
curl -X POST 'http://localhost:4000/v1/bills' \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: error-time-'$(date +%s)-$RANDOM'' \
  -d '{
    "currency": "USD",
    "start_time": "'$FUTURE_START'",
    "end_time": "'$PAST_END'"
  }' \
  -w "Status: %{http_code}\n"

echo ""
echo "=== Error Test 5: Start Time in Past ==="
PAST_START=$(date -v-1d -u +"%Y-%m-%dT%H:%M:%SZ")
FUTURE_END=$(date -v+1d -u +"%Y-%m-%dT%H:%M:%SZ")
curl -X POST 'http://localhost:4000/v1/bills' \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: error-past-start-'$(date +%s)-$RANDOM'' \
  -d '{
    "currency": "USD",
    "start_time": "'$PAST_START'",
    "end_time": "'$FUTURE_END'"
  }' \
  -w "Status: %{http_code}\n"

echo ""
echo "All above should return 4xx status codes"
