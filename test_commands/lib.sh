#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:4000}"

_red()  { printf "\033[31m%s\033[0m\n" "$*"; }
_grn()  { printf "\033[32m%s\033[0m\n" "$*"; }
_blu()  { printf "\033[34m%s\033[0m\n" "$*"; }

info() { _blu "[INFO] $*"; }
pass() { _grn  "[PASS] $*"; }
fail() { _red  "[FAIL] $*"; exit 1; }

require() { command -v "$1" >/dev/null || fail "Missing dependency: $1"; }

# Usage: json_field <json> <jq expr>
json_field() { echo "$1" | jq -r "$2"; }

# assert_json <json> <jq expr> <expected>
assert_json() {
  local json="$1" expr="$2" expected="$3" got
  got=$(json_field "$json" "$expr") || fail "jq failed for $expr"
  if [[ "$got" == "$expected" ]]; then
    pass "$expr == $expected"
  else
    fail "$expr expected=$expected got=$got"
  fi
}

# assert_nonempty <value> <label>
assert_nonempty() { [[ -n "$1" && "$1" != "null" ]] || fail "Empty: $2"; pass "$2 present"; }

# api <method> <path> <json-body> [idempotency-key]
api() {
  local method="$1"; shift
  local path="$1"; shift
  local body="${1:-}"; shift || true
  local idem="${1:-}"; shift || true
  local hdrs=(-H 'Content-Type: application/json')
  [[ -n "$idem" ]] && hdrs+=( -H "X-Idempotency-Key: $idem" )
  curl -sS -X "$method" "${hdrs[@]}" "$BASE_URL$path" -d "$body"
}

# api_status returns body and writes status to global STATUS
api_status() {
  local method="$1"; shift
  local path="$1"; shift
  local body="${1:-}"; shift || true
  local idem="${1:-}"; shift || true
  local hdrs=(-H 'Content-Type: application/json')
  [[ -n "$idem" ]] && hdrs+=( -H "X-Idempotency-Key: $idem" )
  local resp
  resp=$(curl -sS -w '\n%{http_code}' -X "$method" "${hdrs[@]}" "$BASE_URL$path" -d "$body") || fail "curl failed"
  STATUS=$(echo "$resp" | tail -n1)
  echo "$resp" | sed '$d'
}

assert_status() { local expected="$1"; [[ "$STATUS" == "$expected" ]] || fail "status expected=$expected got=$STATUS"; pass "status $STATUS"; }
assert_status_in() { local range="$1"; [[ "$STATUS" =~ $range ]] || fail "status $STATUS not in $range"; pass "status $STATUS in $range"; }

SUMMARY_TOTAL=0
SUMMARY_FAIL=0
summary_inc() { SUMMARY_TOTAL=$((SUMMARY_TOTAL+1)); }
summary_fail() { SUMMARY_FAIL=$((SUMMARY_FAIL+1)); }
summary_report() { if [[ $SUMMARY_FAIL -eq 0 ]]; then pass "Summary: $SUMMARY_TOTAL checks passed"; else fail "Summary: $SUMMARY_FAIL failed / $SUMMARY_TOTAL total"; fi }

save_bill_id() { echo -n "$1" > "$(dirname "$0")/.last_bill_id"; }
load_bill_id() { cat "$(dirname "$0")/.last_bill_id" 2>/dev/null || true; }

# trap to show failing line
trap 's=$?; if [[ $s -ne 0 ]]; then _red "Script failed (exit $s) at line $BASH_LINENO"; fi' EXIT
