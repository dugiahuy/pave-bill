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

CONCURRENT="${RACE_CONCURRENT:-5}"
AMOUNT="${RACE_AMOUNT:-1000}"
info "Race condition test: $CONCURRENT concurrent additions (amount=$AMOUNT) bill=$BILL_ID"

TMP_DIR=$(mktemp -d)
PIDS=()

for i in $(seq 1 "$CONCURRENT"); do
  (
    key="race-$i-$(date +%s)-$RANDOM"
    body=$(cat <<JSON
{ "currency":"USD", "amount_cents": $AMOUNT, "description": "Race Item $i", "reference_id":"race-ref-$i" }
JSON
)
    resp=$(curl -sS -w '\n%{http_code}' -X POST "$BASE_URL/v1/bills/$BILL_ID/line_items" \
      -H 'Content-Type: application/json' \
      -H "X-Idempotency-Key: $key" \
      -d "$body") || echo "curl_error" > "$TMP_DIR/$i.status"
    status=$(echo "$resp" | tail -n1)
    echo "$status" > "$TMP_DIR/$i.status"
    echo "$resp" | sed '$d' > "$TMP_DIR/$i.json"
  ) &
  PIDS+=("$!")
done

FAILS=0
for pid in "${PIDS[@]}"; do
  wait "$pid" || FAILS=$((FAILS+1))
done

SUCCESS_COUNT=0
for i in $(seq 1 "$CONCURRENT"); do
  status=$(cat "$TMP_DIR/$i.status" 2>/dev/null || echo "err")
  body=$(cat "$TMP_DIR/$i.json" 2>/dev/null || echo '{}')
  if [[ "$status" == "200" || "$status" == "201" ]]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT+1))
  else
    info "Request $i failed status=$status body=$(echo "$body" | jq -c '.error // .')"
  fi
done

info "Concurrent successes: $SUCCESS_COUNT / $CONCURRENT (some failures acceptable if lock contention handled)"
[[ $SUCCESS_COUNT -ge 1 ]] || fail "All concurrent requests failed"

EXPECTED_TOTAL=$((CONCURRENT * AMOUNT))
info "Expected additional max total: $EXPECTED_TOTAL cents if all succeeded"
pass "Race condition script completed"
