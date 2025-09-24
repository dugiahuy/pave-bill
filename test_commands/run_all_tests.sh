#!/bin/bash
# Comprehensive automated test suite for billing service
# Runs all tests without manual intervention

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
START_TIME=$(date +%s)

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $*"; }
log_error() { echo -e "${RED}[FAIL]${NC} $*"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $*"; }

run_test() {
  local test_name="$1"
  local test_script="$2"
  local description="$3"

  TESTS_RUN=$((TESTS_RUN + 1))

  echo ""
  echo "========================================"
  echo "TEST $TESTS_RUN: $test_name"
  echo "========================================"
  log_info "$description"
  echo ""

  if [ ! -f "$test_script" ]; then
    log_error "Test script not found: $test_script"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi

  if [ ! -x "$test_script" ]; then
    log_error "Test script not executable: $test_script"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi

  local test_start=$(date +%s)

  if ./"$test_script"; then
    local test_end=$(date +%s)
    local duration=$((test_end - test_start))
    log_success "$test_name completed in ${duration}s"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    local test_end=$(date +%s)
    local duration=$((test_end - test_start))
    log_error "$test_name failed after ${duration}s"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

print_header() {
  echo ""
  echo "================================================"
  echo "           BILLING SERVICE TEST SUITE"
  echo "================================================"
  echo "🚀 Running comprehensive automated tests"
  echo "📅 Started at: $(date)"
  echo "📁 Test directory: $SCRIPT_DIR"
  echo ""
  log_info "All tests are now fully automated - no manual input required!"
  echo ""
}

print_summary() {
  local end_time=$(date +%s)
  local total_duration=$((end_time - START_TIME))

  echo ""
  echo "================================================"
  echo "              TEST SUITE SUMMARY"
  echo "================================================"
  echo "📊 Tests run: $TESTS_RUN"
  echo "✅ Tests passed: $TESTS_PASSED"
  echo "❌ Tests failed: $TESTS_FAILED"
  echo "⏱️  Total duration: ${total_duration}s"
  echo "📅 Completed at: $(date)"
  echo ""

  if [ $TESTS_FAILED -eq 0 ]; then
    log_success "🎉 All tests passed!"
    echo ""
    echo "🔍 What was tested:"
    echo "• Bill creation and idempotency"
    echo "• Line item addition with currency conversion"
    echo "• Complete end-to-end billing flow"
    echo "• Concurrent operations and race conditions"
    echo "• Error handling and validation"
    echo ""
  else
    log_error "💥 $TESTS_FAILED test(s) failed"
    echo ""
    log_info "Check the output above for detailed error information"
    echo ""
  fi
}

# Main execution
print_header

# Test 1: Idempotency Test (quick, self-contained)
run_test "Idempotency Test" "04_idempotency_test.sh" \
  "Tests that duplicate requests with same idempotency key return same bill"

# Test 2: Error Handling Tests (validation tests)
run_test "Error Handling Tests" "05_error_tests.sh" \
  "Tests various error conditions and API validation"

# Test 3: Complete Flow Test (comprehensive end-to-end)
run_test "Complete End-to-End Flow" "06_complete_flow.sh" \
  "Creates bill, adds line items, checks status, and closes bill"

# Test 4: Concurrency Tests (stress testing)
run_test "Concurrency Tests" "07_concurrency_test.sh" \
  "Tests concurrent operations and database locking behavior"

# Optional: Interactive tests (skip if non-interactive)
if [ -t 0 ] && [ -t 1 ]; then
  echo ""
  log_info "Interactive tests available (require manual input):"
  log_warning "Skipping 01_create_bill.sh - requires user input"
  log_warning "Skipping 02_add_line_items.sh - requires bill ID"
  log_warning "Skipping 03_race_condition_test.sh - requires bill ID"
  echo ""
  log_info "💡 To run interactive tests manually:"
  echo "   ./01_create_bill.sh"
  echo "   ./02_add_line_items.sh <bill_id>"
  echo "   ./03_race_condition_test.sh <bill_id>"
else
  log_info "Non-interactive environment detected - skipping manual tests"
fi

print_summary

# Exit with appropriate code
if [ $TESTS_FAILED -eq 0 ]; then
  exit 0
else
  exit 1
fi