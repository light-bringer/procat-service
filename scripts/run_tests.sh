#!/bin/bash
# Complete test suite runner for CI/CD
# Runs all test categories with proper setup and teardown

set -e

echo "============================================"
echo "Product Catalog Service - Test Suite"
echo "============================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track overall status
FAILED=0

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2 passed${NC}"
    else
        echo -e "${RED}✗ $2 failed${NC}"
        FAILED=1
    fi
}

# 1. Unit Tests (no DB required)
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. Running Unit Tests (Domain Layer)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
make test-unit
print_status $? "Unit tests"
echo ""

# 2. Start Test Environment
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. Setting Up Test Environment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
make docker-test-up
print_status $? "Test environment startup"

echo "Waiting for Spanner emulator to be ready..."
sleep 5

# 3. Database Migration
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. Running Database Migrations"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
make migrate-test
print_status $? "Database migrations"

# 4. Integration Tests
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "4. Running Integration Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
SPANNER_EMULATOR_HOST=localhost:19010 go test -v -race -count=1 -tags=integration ./tests/integration/...
INTEGRATION_STATUS=$?
print_status $INTEGRATION_STATUS "Integration tests"

# 5. E2E Tests
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "5. Running E2E Tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
SPANNER_EMULATOR_HOST=localhost:19010 go test -v -race -count=1 -timeout=5m ./tests/e2e/...
E2E_STATUS=$?
print_status $E2E_STATUS "E2E tests"

# 6. Coverage Report
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "6. Generating Coverage Report"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
SPANNER_EMULATOR_HOST=localhost:19010 go test -coverprofile=coverage.out -covermode=atomic ./...
COVERAGE_STATUS=$?
if [ $COVERAGE_STATUS -eq 0 ]; then
    go tool cover -func=coverage.out | tail -1
    print_status 0 "Coverage report generated"
else
    print_status 1 "Coverage report generation"
fi

# 7. Cleanup
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "7. Cleaning Up Test Environment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
make docker-test-down
print_status $? "Test environment cleanup"

# Summary
echo ""
echo "============================================"
echo "Test Suite Summary"
echo "============================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
