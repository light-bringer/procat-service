# Test Suite Documentation

This directory contains the complete test suite for the Product Catalog Service.

## 📁 Directory Structure

```
tests/
├── testutil/           # Test utilities and helpers
│   ├── spanner.go     # Spanner test setup and cleanup
│   ├── fixtures.go    # Test data fixtures
│   └── clock.go       # Mock clock for time-based tests
├── integration/       # Integration tests (with Spanner)
│   ├── product_repo_test.go      # Repository CRUD tests
│   ├── outbox_repo_test.go       # Outbox event tests
│   ├── read_model_test.go        # Query tests
│   └── grpc_service_test.go      # gRPC endpoint tests
└── e2e/               # End-to-end tests (full vertical slice)
    ├── setup_test.go              # Test setup and DI
    ├── product_lifecycle_test.go  # Complete product flows
    └── discount_test.go           # Discount scenarios
```

## 🧪 Test Categories

### Unit Tests
**Location:** `internal/app/product/domain/*_test.go`
**Purpose:** Test pure domain logic
**Dependencies:** None (no DB, no external services)
**Execution Time:** < 2 seconds

```bash
# Run locally
make test-unit

# Run in Docker
make docker-test-unit
```

### Integration Tests
**Location:** `tests/integration/`
**Purpose:** Test repository layer and gRPC endpoints with real Spanner
**Dependencies:** Spanner emulator
**Execution Time:** ~10-15 seconds
**Container:** ✅ **Runs in Docker container**

```bash
# Run locally (starts Spanner emulator automatically)
make test-integration

# Run in Docker container (recommended for CI)
make docker-test-integration
```

**What's tested:**
- Repository CRUD operations
- Change tracking and dirty field optimization
- Domain ↔ Database mapping
- Outbox event persistence
- gRPC endpoint request/response cycles
- Error mapping (domain → gRPC status codes)
- Concurrent request handling

### E2E Tests
**Location:** `tests/e2e/`
**Purpose:** Test complete business scenarios
**Dependencies:** Spanner emulator
**Execution Time:** ~20-30 seconds
**Container:** ✅ **Runs in Docker container**

```bash
# Run locally
make test-e2e

# Run in Docker container (recommended for CI)
make docker-test-e2e
```

**What's tested:**
- Complete product lifecycle
- Discount application flows
- Business rule validations
- Golden Mutation Pattern
- Event persistence
- Multi-step operations

## 🐳 Docker-Based Testing

All integration and E2E tests run in **isolated Docker containers** with their own Spanner emulator.

### Architecture

```
┌─────────────────────┐
│  test-integration   │  ← Test runner container
│  or test-e2e        │
└──────────┬──────────┘
           │ network
┌──────────▼──────────┐
│   spanner-test      │  ← Isolated Spanner emulator
│   (port 19010)      │
└─────────────────────┘
```

### Docker Compose Services

```yaml
services:
  spanner-test:       # Spanner emulator on isolated network
  test-unit:          # Unit tests (no DB needed)
  test-integration:   # Integration tests WITH Spanner
  test-e2e:           # E2E tests WITH Spanner
  test-all:           # Complete suite with coverage
  test-coverage:      # Generate coverage reports
```

### Running Tests in Containers

```bash
# Individual test suites
docker-compose -f docker-compose.test.yml run --rm test-unit
docker-compose -f docker-compose.test.yml run --rm test-integration
docker-compose -f docker-compose.test.yml run --rm test-e2e

# Or use Make targets
make docker-test-unit          # Unit tests
make docker-test-integration   # Integration tests in container
make docker-test-e2e           # E2E tests in container
make docker-test-all           # Everything in container

# Generate coverage reports
make docker-test-coverage
```

## 📊 Test Reporting with gotestsum

All tests use `gotestsum` for enhanced output and CI/CD integration.

### Output Formats

```bash
# Pretty output during development
make test-unit-pretty
make test-integration-pretty
make test-e2e-pretty

# CI-friendly output with JUnit XML
make test-ci

# JSON output for processing
make test-json

# Watch mode (auto-rerun on changes)
make test-watch-pretty
```

### CI/CD Integration

```bash
# Complete suite with JUnit XML and JSON reports
make test-ci

# Output files generated in test-results/:
# - unit-junit.xml
# - integration-junit.xml
# - e2e-junit.xml
# - tests.json
# - coverage.out
# - coverage.html
```

## 🔍 Test Utilities

### Spanner Helpers (`testutil/spanner.go`)

```go
// Setup test with clean database
client, cleanup := testutil.SetupSpannerTest(t)
defer cleanup()

// Assert row count
testutil.AssertRowCount(t, client, "products", 5)

// Clean database manually
testutil.CleanDatabase(t, client)
```

### Test Fixtures (`testutil/fixtures.go`)

```go
// Create test product
productID := testutil.CreateTestProduct(t, client, "Test Product")

// Create product with discount
productID := testutil.CreateTestProductWithDiscount(t, client, "Product", 20)

// Verify outbox event
testutil.AssertOutboxEvent(t, client, "product.created")
```

### Mock Clock (`testutil/clock.go`)

```go
// Create mock clock for time-based tests
mockClock := testutil.NewMockClock()
mockClock.Set(time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))

// Advance time
mockClock.Advance(24 * time.Hour)
```

## 📈 Coverage Goals

| Layer | Target | Current |
|-------|--------|---------|
| Domain | > 90% | ✅ High |
| Repository | > 80% | ✅ High |
| Use Cases | > 85% | ✅ High |
| gRPC Handlers | > 75% | ✅ High |
| **Overall** | **> 80%** | ✅ **High** |

## 🚀 Quick Start

```bash
# Install gotestsum
make tools

# Run all tests locally
make test-all

# Run all tests in Docker (CI simulation)
make docker-test-all

# Generate coverage report
make docker-test-coverage
open test-results/coverage.html
```

## 📝 Writing New Tests

### Integration Test Template

```go
//go:build integration

package integration

import (
    "testing"
    "github.com/stretchr/testify/require"
    "github.com/light-bringer/procat-service/tests/testutil"
)

func TestMyFeature(t *testing.T) {
    // Setup with automatic cleanup
    client, cleanup := testutil.SetupSpannerTest(t)
    defer cleanup()

    // Your test code here
    // Database is clean and ready
}
```

### E2E Test Template

```go
package e2e

import (
    "testing"
    "github.com/stretchr/testify/require"
)

func TestMyScenario(t *testing.T) {
    // Setup all dependencies
    services, cleanup := setupTest(t)
    defer cleanup()

    // Test complete business flow
    productID, err := services.CreateProduct.Execute(ctx(), req)
    require.NoError(t, err)

    // Verify via query
    dto, _ := services.GetProduct.Execute(ctx(), &get_product.Request{
        ProductID: productID,
    })
    // assertions...
}
```

## 🔄 CI/CD Integration Example

```yaml
# GitHub Actions example
- name: Run Tests
  run: |
    make docker-test-all

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./test-results/coverage.out

- name: Publish Test Results
  uses: EnricoMi/publish-unit-test-result-action@v2
  with:
    files: test-results/*.xml
```

## 💡 Best Practices

1. **Isolation:** Each test cleans database before running
2. **Independence:** Tests don't depend on execution order
3. **Clarity:** Use descriptive test names and subtests
4. **Speed:** Unit tests < 2s, Integration < 15s, E2E < 30s
5. **Coverage:** Aim for high coverage but don't chase 100%
6. **Deterministic:** No flaky tests (use mock clock for time)
7. **Container-based:** Integration/E2E tests run in Docker

## 🐛 Debugging Tests

```bash
# Run single test
go test -v -run TestGRPC_CreateProduct ./tests/integration/

# Run with race detector
go test -race ./tests/integration/

# Run with coverage
go test -cover ./tests/integration/

# Verbose output
SPANNER_EMULATOR_HOST=localhost:19010 \
go test -v -tags=integration ./tests/integration/ -test.v
```

---

**All integration and E2E tests run in Docker containers with isolated Spanner emulators for consistent, reproducible results!** 🎉
