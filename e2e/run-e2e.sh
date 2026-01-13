#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

export NEXUS_URL="${NEXUS_URL:-http://localhost:8081}"
export NEXUS_USER="${NEXUS_USER:-admin}"
export NEXUS_PASS="${NEXUS_PASS:-admin123}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Run all test scripts
run_tests() {
    local failed=0
    local passed=0

    for test_script in "${SCRIPT_DIR}/tests/"*.sh; do
        test_name=$(basename "$test_script")
        log_info "Running test: $test_name"
        echo "========================================"

        if bash "$test_script"; then
            log_info "PASSED: $test_name"
            ((passed++))
        else
            log_error "FAILED: $test_name"
            ((failed++))
        fi

        echo ""
    done

    echo "========================================"
    echo "Test Summary"
    echo "========================================"
    log_info "Passed: $passed"
    if [ $failed -gt 0 ]; then
        log_error "Failed: $failed"
        return 1
    fi
    log_info "All tests passed!"
    return 0
}

# Main
case "${1:-run}" in
    run)
        run_tests
        ;;
    *)
        echo "Usage: $0 [run]"
        exit 1
        ;;
esac
