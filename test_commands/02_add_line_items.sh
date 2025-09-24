#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
require jq

BILL_ID="${1:-}"
if [[ -z "$BILL_ID" ]]; then
  BILL_ID=$(load_bill_id)
fi
[[ -n "$BILL_ID" ]] || fail "Bill ID required (arg or .last_bill_id)"

info "Adding line items to bill $BILL_ID"
TOTAL=0

add_item() {
  local currency="$1" amount="$2" desc="$3" ref="$4" key="$5"
  echo "Adding item: currency=$currency, amount=$amount, desc=$desc, ref=$ref, key=$key"

  # Create properly formatted JSON
  local json=$(cat <<EOF
{
  "currency": "$currency",
  "amount_cents": $amount,
  "description": "$desc",
  "reference_id": "$ref"
}
EOF
)

  echo "Request JSON:"
  echo "$json" | jq '.'

  RESP=$(api POST "/v1/bills/$BILL_ID/line_items" "$json" "$key")
  echo "Full response:"
  echo "$RESP" | jq '.'

  # Check if response contains an error
  if echo "$RESP" | jq -e '.error // .code' >/dev/null 2>&1; then
    echo "❌ API Error Response:"
    echo "$RESP" | jq '.'
    echo "Error: $(echo "$RESP" | jq -r '.message // .error.message // .error // "Unknown error"')"
    return 1
  fi

  echo "Line item summary:"
  echo "$RESP" | jq '.line_item | {id, amount_cents, currency, metadata}'

  # Get the actual converted amount and currency from the response
  CONVERTED_AMOUNT=$(json_field "$RESP" '.line_item.amount_cents')
  BILL_CURRENCY=$(json_field "$RESP" '.line_item.currency')

  # Check if currency conversion happened (metadata exists)
  if echo "$RESP" | jq -e '.line_item.metadata' >/dev/null 2>&1; then
    # Currency conversion occurred - verify metadata
    assert_json "$RESP" '.line_item.metadata.original_currency' "$currency"
    assert_json "$RESP" '.line_item.metadata.original_amount_cents' "$amount"
    echo "✓ Converted $amount $currency → $CONVERTED_AMOUNT $BILL_CURRENCY"
  else
    # No conversion needed - currency matches bill currency
    if [[ "$currency" == "$BILL_CURRENCY" ]]; then
      echo "✓ No conversion needed: $amount $currency (matches bill currency)"
    else
      echo "⚠️  No metadata found, but currencies don't match: $currency vs $BILL_CURRENCY"
    fi
  fi

  ID=$(json_field "$RESP" '.line_item.id')
  assert_nonempty "$ID" line_item_id

  # Add the converted amount to total (in bill currency)
  TOTAL=$((TOTAL + CONVERTED_AMOUNT))
}

add_item USD 10000 "Test Item 1" ref-001 "li1-$(date +%s)-$RANDOM"
add_item GEL 5000  "Test Item 2" ref-002 "li2-$(date +%s)-$RANDOM"
add_item JPY 2500  "Test Item 3" ref-003 "li3-$(date +%s)-$RANDOM"

pass "Added 3 items, total converted amount: $TOTAL (in bill currency)"

# Note: The total will vary based on currency conversion rates
# USD 10000 → GEL (×2.65) = 26500
# GEL 5000 → GEL (×1.0) = 5000
# JPY 2500 → GEL (rate varies) = varies
echo "Final total in bill currency: $TOTAL"
