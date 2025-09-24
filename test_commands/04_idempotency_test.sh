#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
require jq

DUPLICATE_KEY="dup-$(date +%s)"
START_TIME=$(date -v+1d -u +"%Y-%m-%dT%H:%M:%SZ")
END_TIME=$(date -v+2d -u +"%Y-%m-%dT%H:%M:%SZ")
BODY="{\"currency\":\"USD\",\"start_time\":\"$START_TIME\",\"end_time\":\"$END_TIME\"}"

info "Idempotent create bill (key=$DUPLICATE_KEY) first call"
R1=$(api POST /v1/bills "$BODY" "$DUPLICATE_KEY")
ID1=$(json_field "$R1" '.bill.id')
assert_nonempty "$ID1" bill_id_first

sleep 1
info "Second call with same key"
R2=$(api POST /v1/bills "$BODY" "$DUPLICATE_KEY")
ID2=$(json_field "$R2" '.bill.id')
assert_json "$R2" '.bill.id' "$ID1"

pass "Idempotency verified (bill id $ID1 reused)"
