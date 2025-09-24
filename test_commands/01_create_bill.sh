#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
require jq

info "Creating bill - Interactive Mode"

# Prompt for currency input
echo "Supported currencies: USD, GEL"
echo -n "Enter currency code (default: USD): "
read -r CURRENCY
CURRENCY="${CURRENCY:-USD}"
echo "Using currency: $CURRENCY"

# Prompt for start time
echo ""
echo "Start time options:"
echo "  1) Now"
echo "  2) Custom minutes in the future"
echo -n "Choose start time (1-2, default: 1): "
read -r start_choice
case "${start_choice:-1}" in
    1) START_MINUTES=0 ;;
    2)
        echo -n "Enter minutes in the future for start time: "
        read -r START_MINUTES
        if ! [[ "$START_MINUTES" =~ ^[0-9]+$ ]]; then
            echo "Invalid input, defaulting to 0 (now)"
            START_MINUTES=0
        fi
        ;;
    *) START_MINUTES=0; echo "Invalid choice, defaulting to now" ;;
esac

# Prompt for end time
echo ""
echo "End time options: How many minutes after start time"
echo -n "Choose end time in minutes (default: 60): "
read -r end_choice
case "${end_choice:-1}" in
    1) END_OFFSET=60 ;;
    2)
        echo -n "Enter minutes after start time for end time: "
        read -r END_OFFSET
        if ! [[ "$END_OFFSET" =~ ^[0-9]+$ ]] || [ "$END_OFFSET" -eq 0 ]; then
            echo "Invalid input, defaulting to 60 minutes"
            END_OFFSET=60
        fi
        ;;
    *) END_OFFSET=60; echo "Invalid choice, defaulting to 1 hour" ;;
esac

# Calculate timestamps
if [ "$START_MINUTES" -eq 0 ]; then
    START_TIME_JSON="null"
    START_TIME_DISPLAY="now"
    # For end time calculation when start is now, use current time
    END_TIME=$(date -u -v+"${END_OFFSET}"M +%Y-%m-%dT%H:%M:%SZ)
else
    START_TIME=$(date -u -v+"${START_MINUTES}"M +%Y-%m-%dT%H:%M:%SZ)
    START_TIME_JSON="\"$START_TIME\""
    START_TIME_DISPLAY="$START_TIME"
    END_MINUTES=$((START_MINUTES + END_OFFSET))
    END_TIME=$(date -u -v+"${END_MINUTES}"M +%Y-%m-%dT%H:%M:%SZ)
fi

echo ""
info "Creating bill with:"
echo "  Currency: $CURRENCY"
echo "  Start time: $START_TIME_DISPLAY"
echo "  End time: $END_TIME"

IDEMP="create-$(date +%s)-$RANDOM"
RESP=$(api POST /v1/bills '{
  "currency": "'$CURRENCY'",
  "start_time": '$START_TIME_JSON',
  "end_time": "'$END_TIME'"
}' "$IDEMP")

echo "$RESP" | jq '.'

assert_json "$RESP" '.bill.currency' "$CURRENCY"
assert_json "$RESP" '.bill.status' 'pending'
BILL_ID=$(json_field "$RESP" '.bill.id')
assert_nonempty "$BILL_ID" bill_id
save_bill_id "$BILL_ID"
pass "Bill created id=$BILL_ID (idempotency key $IDEMP)"
