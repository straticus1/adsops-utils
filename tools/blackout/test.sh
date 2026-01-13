#!/bin/bash
#
# Blackout Tool - Integration Test Suite
#
# This script tests all major functionality of the blackout tool.
# Run this after installation to verify everything works correctly.
#
# Usage:
#   ./test.sh
#

set -euo pipefail

# Configuration
BLACKOUT_TOOL="${BLACKOUT_TOOL:-./blackout}"
TEST_HOSTNAME="test-host-$$"  # Use PID to avoid conflicts
TEST_TICKET="CHG-TEST-$$"
VERBOSE="${VERBOSE:-0}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Functions
log() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $*"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

error() {
    echo -e "${RED}[FAIL]${NC} $*"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

warning() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

run_test() {
    local test_name="$1"
    shift
    local command=("$@")

    TESTS_RUN=$((TESTS_RUN + 1))

    log "Running: ${test_name}"

    if [[ $VERBOSE -eq 1 ]]; then
        echo "  Command: ${command[*]}"
    fi

    local output
    local exit_code=0

    output=$("${command[@]}" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        success "${test_name}"
        if [[ $VERBOSE -eq 1 ]]; then
            echo "$output" | sed 's/^/  /'
        fi
        return 0
    else
        error "${test_name}"
        echo "$output" | sed 's/^/  /'
        return 1
    fi
}

assert_contains() {
    local text="$1"
    local pattern="$2"
    local description="$3"

    TESTS_RUN=$((TESTS_RUN + 1))

    if echo "$text" | grep -q "$pattern"; then
        success "${description}"
        return 0
    else
        error "${description}"
        echo "  Expected to find: ${pattern}"
        echo "  In output:"
        echo "$text" | sed 's/^/    /'
        return 1
    fi
}

cleanup() {
    log "Cleaning up test data..."

    # Try to end any active blackouts for test host
    "${BLACKOUT_TOOL}" end "${TEST_HOSTNAME}" 2>/dev/null || true

    log "Cleanup complete"
}

# Main test suite
main() {
    echo ""
    echo "=========================================="
    echo "Blackout Tool - Integration Test Suite"
    echo "=========================================="
    echo ""

    # Check if binary exists
    if [[ ! -f "${BLACKOUT_TOOL}" ]]; then
        error "Blackout tool not found: ${BLACKOUT_TOOL}"
        echo ""
        echo "Please build the tool first:"
        echo "  make build"
        echo ""
        exit 1
    fi

    # Set up cleanup trap
    trap cleanup EXIT

    # Test 1: Version command
    run_test "Test version command" "${BLACKOUT_TOOL}" version

    # Test 2: Help command
    run_test "Test help command" "${BLACKOUT_TOOL}" help

    # Test 3: List (should work even if empty)
    run_test "Test list command" "${BLACKOUT_TOOL}" list

    # Test 4: Start a blackout
    log "Creating test blackout..."
    output=$("${BLACKOUT_TOOL}" start "${TEST_TICKET}" "${TEST_HOSTNAME}" "0:05" "Integration test" 2>&1)
    assert_contains "$output" "Blackout started" "Start blackout command"

    # Test 5: Show the blackout
    log "Checking blackout details..."
    output=$("${BLACKOUT_TOOL}" show "${TEST_HOSTNAME}" 2>&1)
    assert_contains "$output" "${TEST_HOSTNAME}" "Show blackout details"
    assert_contains "$output" "${TEST_TICKET}" "Verify ticket in output"

    # Test 6: List active blackouts
    log "Listing active blackouts..."
    output=$("${BLACKOUT_TOOL}" list --active 2>&1)
    assert_contains "$output" "${TEST_HOSTNAME}" "List active blackouts"

    # Test 7: Extend the blackout
    log "Extending blackout..."
    output=$("${BLACKOUT_TOOL}" extend "${TEST_HOSTNAME}" "0:05" 2>&1)
    assert_contains "$output" "extended" "Extend blackout"

    # Test 8: Export active blackouts
    run_test "Test export command" "${BLACKOUT_TOOL}" export

    # Test 9: Check JSON file was created
    TESTS_RUN=$((TESTS_RUN + 1))
    if [[ -f /var/lib/adsops/active-blackouts.json ]]; then
        success "JSON export file created"

        # Verify JSON is valid
        if command -v jq >/dev/null 2>&1; then
            if jq empty /var/lib/adsops/active-blackouts.json 2>/dev/null; then
                success "JSON export file is valid JSON"
                TESTS_RUN=$((TESTS_RUN + 1))
            else
                error "JSON export file is invalid"
                TESTS_RUN=$((TESTS_RUN + 1))
            fi
        fi

        # Check if our test host is in the export
        output=$(cat /var/lib/adsops/active-blackouts.json)
        assert_contains "$output" "${TEST_HOSTNAME}" "Test host in JSON export"
    else
        warning "JSON export file not found (may need sudo access to /var/lib/adsops)"
    fi

    # Test 10: End the blackout
    log "Ending blackout..."
    output=$("${BLACKOUT_TOOL}" end "${TEST_HOSTNAME}" 2>&1)
    assert_contains "$output" "Blackout ended" "End blackout command"

    # Test 11: Verify blackout is no longer active
    log "Verifying blackout ended..."
    output=$("${BLACKOUT_TOOL}" show "${TEST_HOSTNAME}" 2>&1)
    assert_contains "$output" "completed\|expired" "Blackout status is completed/expired"

    # Test 12: List should not show as active
    TESTS_RUN=$((TESTS_RUN + 1))
    output=$("${BLACKOUT_TOOL}" list --active 2>&1)
    if ! echo "$output" | grep -q "${TEST_HOSTNAME}"; then
        success "Host not in active list after ending"
    else
        error "Host still appears in active list"
    fi

    # Test 13: Cleanup command
    run_test "Test cleanup command" "${BLACKOUT_TOOL}" cleanup

    # Test 14: Try to start with shorthand syntax
    log "Testing shorthand syntax..."
    output=$("${BLACKOUT_TOOL}" "${TEST_TICKET}-2" "${TEST_HOSTNAME}-2" "0:05" "Shorthand test" 2>&1)
    assert_contains "$output" "Blackout started" "Shorthand syntax"

    # Clean up the second test host
    "${BLACKOUT_TOOL}" end "${TEST_HOSTNAME}-2" 2>/dev/null || true

    # Summary
    echo ""
    echo "=========================================="
    echo "Test Results"
    echo "=========================================="
    echo "Total tests:  ${TESTS_RUN}"
    echo "Passed:       ${GREEN}${TESTS_PASSED}${NC}"
    echo "Failed:       ${RED}${TESTS_FAILED}${NC}"
    echo "=========================================="
    echo ""

    if [[ $TESTS_FAILED -eq 0 ]]; then
        success "All tests passed! ✅"
        exit 0
    else
        error "Some tests failed ❌"
        exit 1
    fi
}

# Run tests
main "$@"
