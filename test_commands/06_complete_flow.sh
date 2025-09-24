#!/bin/bash
# Complete test flow - run everything in sequence

echo "=========================================="
echo "      COMPLETE BILLING SERVICE TEST"
echo "=========================================="

# Step 1: Create bill
echo "STEP 1: Creating bill..."
RESPONSE=$(curl -s -X POST 'http://localhost:4000/v1/bills' \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: complete-flow-'$(date +%s)-$RANDOM'' \
  -d '{
    "currency": "USD",
    "start_time": null,
    "end_time": "'$(date -v+2d -u +"%Y-%m-%dT%H:%M:%SZ")'"
  }')

echo "$RESPONSE" | jq '.'
BILL_ID=$(echo "$RESPONSE" | jq -r '.bill.id // empty')

if [ -z "$BILL_ID" ] || [ "$BILL_ID" = "null" ]; then
  echo "❌ Failed to create bill"
  exit 1
fi

echo "✅ Bill created with ID: $BILL_ID"
echo ""

# Wait for workflow to start
echo "Waiting 3 seconds for workflow to start..."
sleep 3

# Step 2: Add line items
echo "STEP 2: Adding line items..."
declare -a amounts=(5000 3000 2000 1500 1000)
total_expected=0

for i in ${!amounts[@]}; do
  amount=${amounts[$i]}
  item_num=$((i + 1))
  
  echo "Adding item $item_num: $amount cents"
  curl -s -X POST 'http://localhost:4000/v1/bills/'$BILL_ID'/line_items' \
    -H 'Content-Type: application/json' \
    -H 'X-Idempotency-Key: flow-item-$item_num-'$(date +%s)-$RANDOM'' \
    -d '{
      "currency": "USD",
      "amount_cents": '$amount',
      "description": "Flow Test Item '$item_num'",
      "reference_id": "flow-ref-'$item_num'"
    }' | jq '.line_item | {id, amount_cents, description}'
  
  total_expected=$(($total_expected + $amount))
  
  # Small delay between items
  sleep 1
done

echo ""
echo "✅ All line items added"
echo "💰 Total expected: $total_expected cents (\$$(echo "scale=2; $total_expected/100" | bc))"

echo ""
echo "Waiting 5 seconds for workflow to process all items..."
sleep 5

# Step 3: Fetch bill status
echo ""
echo "STEP 3: Fetching bill status..."
BILL_STATUS_RESPONSE=$(curl -s -X GET "http://localhost:4000/v1/bills/$BILL_ID" \
  -H 'Content-Type: application/json')

echo "$BILL_STATUS_RESPONSE" | jq '.'

# Extract bill details
BILL_STATUS=$(echo "$BILL_STATUS_RESPONSE" | jq -r '.bill.status // "unknown"')
TOTAL_AMOUNT=$(echo "$BILL_STATUS_RESPONSE" | jq -r '.bill.total_amount_cents // 0')
LINE_ITEMS_COUNT=$(echo "$BILL_STATUS_RESPONSE" | jq -r '.bill.line_items | length // 0')

echo ""
echo "📊 Bill Status Summary:"
echo "   Status: $BILL_STATUS"
echo "   Total Amount: $TOTAL_AMOUNT cents (\$$(echo "scale=2; $TOTAL_AMOUNT/100" | bc))"
echo "   Line Items: $LINE_ITEMS_COUNT"
echo "   Expected: $total_expected cents"

# Verify totals match
if [ "$TOTAL_AMOUNT" -eq "$total_expected" ]; then
  echo "✅ Total amount matches expected!"
else
  echo "⚠️  Total mismatch: got $TOTAL_AMOUNT, expected $total_expected"
fi

# Step 4: Close bill
echo ""
echo "STEP 4: Closing bill..."
CLOSE_RESPONSE=$(curl -s -X POST "http://localhost:4000/v1/bills/$BILL_ID/close" \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: complete-flow-close-'$(date +%s)-$RANDOM'' \
  -d '{
    "reason": "complete_flow_test_finished"
  }')

echo "$CLOSE_RESPONSE" | jq '.'

# Extract close response details
CLOSE_STATUS=$(echo "$CLOSE_RESPONSE" | jq -r '.bill.status // .error // "unknown"')
CLOSE_SUCCESS=$(echo "$CLOSE_RESPONSE" | jq -r 'has("bill")')

echo ""
echo "🔒 Bill Closure Summary:"
if [ "$CLOSE_SUCCESS" = "true" ]; then
  echo "   ✅ Bill closed successfully"
  echo "   Final Status: $CLOSE_STATUS"
  FINAL_TOTAL=$(echo "$CLOSE_RESPONSE" | jq -r '.bill.total_amount_cents // 0')
  echo "   Final Total: $FINAL_TOTAL cents"
else
  echo "   ❌ Failed to close bill"
  echo "   Error: $CLOSE_STATUS"
fi

echo ""
echo "=========================================="
echo "✅ Complete flow test finished!"
echo "📋 Bill ID: $BILL_ID"
echo "📊 Final Status: $CLOSE_STATUS"
echo "💰 Final Total: $TOTAL_AMOUNT cents"
echo "🔍 Check your logs for workflow processing"
echo "=========================================="
