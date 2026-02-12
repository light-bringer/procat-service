# Product Catalog Service Implementation Plan

## Context

This is a greenfield implementation of a Product Catalog Service following strict Domain-Driven Design (DDD) and Clean Architecture principles. The service implements product management with pricing rules using the Golden Mutation Pattern, CQRS, and transactional outbox for event publishing.

**Technology Stack:**
- Go 1.25
- Google Cloud Spanner (with emulator for local dev)
- gRPC with Protocol Buffers
- github.com/Vektor-AI/commitplan for transaction management
- math/big for precise decimal arithmetic

**Key Requirements:**
- Domain layer must be pure (no context, DB, or external deps)
- All writes follow Golden Mutation Pattern (load → domain logic → build plan → apply)
- CQRS separation (commands vs queries)
- Transactional outbox for reliable event publishing
- Repository returns mutations, usecases apply them
- Change tracking for optimized updates

## Implementation Plan

### Phase 1: Project Foundation & Infrastructure Setup

**1.1 Initialize Go Module and Dependencies**
- Create `go.mod` with Go 1.25
- Add core dependencies:
  - cloud.google.com/go/spanner
  - github.com/Vektor-AI/commitplan
  - google.golang.org/grpc
  - google.golang.org/protobuf
  - github.com/google/uuid
  - github.com/stretchr/testify (for tests)

**1.2 Docker Compose Setup (Multi-Environment)**

Create three Docker Compose configurations:

**A. `docker-compose.yml` - Development Environment**
```yaml
services:
  spanner-emulator:
    image: gcr.io/cloud-spanner-emulator/emulator:latest
    ports:
      - "9010:9010"  # gRPC
      - "9020:9020"  # HTTP
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9020"]
      interval: 5s
      timeout: 3s
      retries: 5
```

**B. `docker-compose.test.yml` - Testing Environment**
```yaml
services:
  spanner-test:
    image: gcr.io/cloud-spanner-emulator/emulator:latest
    ports:
      - "19010:9010"  # Different ports to avoid conflicts
      - "19020:9020"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9020"]
      interval: 2s
      timeout: 2s
      retries: 10

  test-runner:
    build:
      context: .
      dockerfile: Dockerfile.test
    depends_on:
      spanner-test:
        condition: service_healthy
    environment:
      - SPANNER_EMULATOR_HOST=spanner-test:9010
      - TEST_DB_INSTANCE=test-instance
      - TEST_DB_NAME=test-db
    volumes:
      - .:/app
      - /app/vendor  # Cache dependencies
    command: make test-all
```

**C. `Dockerfile.test` - Test Container**
```dockerfile
FROM golang:1.25-alpine

RUN apk add --no-cache make git curl

WORKDIR /app

# Cache dependencies layer
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Install test tools
RUN go install github.com/google/go-cloud/cmd/spanner-cli@latest

CMD ["make", "test-all"]
```

**1.3 Database Migrations**

**A. Migration Files**
- `migrations/001_initial_schema.sql`: DDL for products and outbox_events tables
- `scripts/migrate.sh`: Migration runner script
- `scripts/migrate_test.sh`: Test database migration runner
- `scripts/cleanup_test_db.sh`: Test database cleanup script

**B. Migration Script (`scripts/migrate.sh`)**
```bash
#!/bin/bash
# Applies migrations to Spanner instance
# Usage: ./scripts/migrate.sh <instance> <database>
```

**C. Test Database Setup (`scripts/setup_test_db.sh`)**
```bash
#!/bin/bash
# Creates test instance/database and applies migrations
# Idempotent - safe to run multiple times
```

**1.4 Comprehensive Makefile**

Create `Makefile` with the following targets organized by category:

**A. Dependency Management**
```makefile
.PHONY: deps
deps: ## Install Go dependencies
	go mod download
	go mod tidy

.PHONY: tools
tools: ## Install development tools
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

**B. Code Generation**
```makefile
.PHONY: proto
proto: ## Generate protobuf code
	protoc --go_out=. --go-grpc_out=. proto/product/v1/*.proto

.PHONY: generate
generate: proto ## Run all code generation
```

**C. Docker & Infrastructure**
```makefile
.PHONY: docker-up
docker-up: ## Start development Spanner emulator
	docker-compose up -d
	@echo "Waiting for Spanner emulator to be ready..."
	@sleep 3

.PHONY: docker-down
docker-down: ## Stop development Spanner emulator
	docker-compose down -v

.PHONY: docker-test-up
docker-test-up: ## Start test Spanner emulator
	docker-compose -f docker-compose.test.yml up -d spanner-test
	@echo "Waiting for test Spanner emulator to be ready..."
	@sleep 3

.PHONY: docker-test-down
docker-test-down: ## Stop test environment
	docker-compose -f docker-compose.test.yml down -v
```

**D. Database Migrations**
```makefile
.PHONY: migrate
migrate: ## Run migrations on dev database
	./scripts/migrate.sh dev-instance product-catalog-db

.PHONY: migrate-test
migrate-test: ## Run migrations on test database
	./scripts/setup_test_db.sh

.PHONY: migrate-clean
migrate-clean: ## Clean and recreate test database
	./scripts/cleanup_test_db.sh
	./scripts/setup_test_db.sh
```

**E. Testing Targets (Enhanced)**
```makefile
.PHONY: test
test: test-unit ## Run all tests (unit + integration + e2e)

.PHONY: test-unit
test-unit: ## Run unit tests (domain layer only, no DB)
	go test -v -race -count=1 ./internal/app/product/domain/...

.PHONY: test-integration
test-integration: docker-test-up migrate-test ## Run integration tests (with real Spanner)
	SPANNER_EMULATOR_HOST=localhost:19010 \
	go test -v -race -count=1 -tags=integration ./tests/integration/...
	$(MAKE) docker-test-down

.PHONY: test-e2e
test-e2e: docker-test-up migrate-test ## Run E2E tests
	SPANNER_EMULATOR_HOST=localhost:19010 \
	go test -v -race -count=1 -timeout=5m ./tests/e2e/...
	$(MAKE) docker-test-down

.PHONY: test-all
test-all: test-unit test-integration test-e2e ## Run all test suites

.PHONY: test-coverage
test-coverage: docker-test-up migrate-test ## Run tests with coverage report
	SPANNER_EMULATOR_HOST=localhost:19010 \
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out
	$(MAKE) docker-test-down

.PHONY: test-docker
test-docker: ## Run tests inside Docker container (CI simulation)
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down -v
```

**F. Build & Run**
```makefile
.PHONY: build
build: ## Build the service binary
	go build -o bin/server ./cmd/server/

.PHONY: run
run: ## Run the gRPC server locally
	go run ./cmd/server/

.PHONY: run-dev
run-dev: docker-up migrate ## Start dev environment and run server
	go run ./cmd/server/
```

**G. Code Quality**
```makefile
.PHONY: lint
lint: ## Run linters
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: check
check: fmt vet lint ## Run all code quality checks
```

**H. Cleanup**
```makefile
.PHONY: clean
clean: ## Clean build artifacts and test data
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker-compose down -v
	docker-compose -f docker-compose.test.yml down -v

.PHONY: clean-all
clean-all: clean ## Deep clean including Go cache
	go clean -cache -testcache -modcache
```

**I. Help**
```makefile
.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
```

### Phase 2: Domain Layer (Pure Business Logic)

**Location:** `internal/app/product/domain/`

**2.1 Value Objects**
- `money.go`: Money value object using `*big.Rat`
  - Constructor: `NewMoney(numerator, denominator int64)`
  - Methods: `Add()`, `Subtract()`, `Multiply()`, `Divide()`, `Numerator()`, `Denominator()`
  - Validation for non-zero denominator

- `discount.go`: Discount value object
  - Fields: percentage (0-100), startDate, endDate
  - Methods: `IsValidAt(time.Time) bool`, `Apply(*Money) *Money`
  - Validation: percentage range, date ordering

**2.2 Domain Errors**
- `domain_errors.go`: Sentinel errors
  - `ErrProductNotActive`
  - `ErrProductNotFound`
  - `ErrInvalidDiscountPeriod`
  - `ErrDiscountAlreadyActive`
  - `ErrInvalidPrice`
  - `ErrEmptyName`

**2.3 Product Aggregate**
- `product.go`: Core aggregate with encapsulated state
  - Fields: id, name, description, category, basePrice, discount, status, createdAt, updatedAt, archivedAt
  - Field constants for change tracking
  - Constructor: `NewProduct(...)`
  - Reconstitution: `ReconstructProduct(...)` for loading from DB
  - Business methods:
    - `SetName(string) error`
    - `SetDescription(string)`
    - `SetCategory(string) error`
    - `ApplyDiscount(*Discount, time.Time) error`
    - `RemoveDiscount()`
    - `Activate() error`
    - `Deactivate() error`
    - `Archive()`
    - `CalculateEffectivePrice(time.Time) *Money`
  - Change tracking and event capturing

**2.4 Change Tracker**
- `change_tracker.go`: Track dirty fields
  - `MarkDirty(field string)`
  - `Dirty(field string) bool`
  - `Clear()`

**2.5 Domain Events**
- `domain_events.go`: Event structs (simple data holders)
  - `ProductCreatedEvent`
  - `ProductUpdatedEvent`
  - `ProductActivatedEvent`
  - `ProductDeactivatedEvent`
  - `DiscountAppliedEvent`
  - `DiscountRemovedEvent`
  - `ProductArchivedEvent`

**2.6 Domain Services**
- `services/pricing_calculator.go`: Calculate effective prices with discount logic

### Phase 3: Database Models

**Location:** `internal/models/`

**3.1 Product Model**
- `m_product/data.go`: Product database model struct
- `m_product/fields.go`: Field name constants
- `m_product/model.go`: Model facade with type-safe CRUD operations
  - `InsertMut(data) *spanner.Mutation`
  - `UpdateMut(id, updates) *spanner.Mutation`
  - `DeleteMut(id) *spanner.Mutation`

**3.2 Outbox Model**
- `m_outbox/data.go`: Outbox event database model struct
- `m_outbox/fields.go`: Field name constants
- `m_outbox/model.go`: Model facade for outbox operations

### Phase 4: Infrastructure - Repository Layer

**Location:** `internal/app/product/`

**4.1 Contracts (Interfaces)**
- `contracts/product_repo.go`: Repository interface
  - `InsertMut(*domain.Product) *spanner.Mutation`
  - `UpdateMut(*domain.Product) *spanner.Mutation`
  - `GetByID(ctx, id) (*domain.Product, error)`

- `contracts/read_model.go`: Read model interface for queries
  - `GetProductByID(ctx, id) (*ProductDTO, error)`
  - `ListProducts(ctx, filter) ([]*ProductDTO, error)`

- `contracts/outbox_repo.go`: Outbox repository interface
  - `InsertMut(*OutboxEvent) *spanner.Mutation`

**4.2 Repository Implementation**
- `repo/product_repo.go`: Spanner implementation
  - Map domain.Product ↔ m_product.Data
  - Use change tracker to build optimized updates
  - Only return mutations, never apply them

- `repo/outbox_repo.go`: Spanner outbox implementation

- `repo/read_model.go`: Query implementation with DTOs

### Phase 5: Application Layer - Use Cases

**Location:** `internal/app/product/usecases/`

**5.1 Command Use Cases (Write Operations)**
Each in its own subdirectory with `interactor.go`:

- `create_product/`: Create new product
  - Request struct with validation
  - Interactor with Execute(ctx, req) (string, error)
  - Follow Golden Mutation Pattern

- `update_product/`: Update product details
- `activate_product/`: Activate product
- `deactivate_product/`: Deactivate product
- `apply_discount/`: Apply discount to product
- `remove_discount/`: Remove discount from product
- `archive_product/`: Archive product (soft delete)

All follow same pattern:
1. Load aggregate (or create new)
2. Call domain methods
3. Build CommitPlan
4. Add repository mutations
5. Add outbox events
6. Apply plan
7. Return result

**5.2 Query Use Cases (Read Operations)**

**Location:** `internal/app/product/queries/`

- `get_product/`:
  - `query.go`: GetProductQuery with Execute(ctx, id) (*ProductDTO, error)
  - `dto.go`: ProductDTO with calculated effective price

- `list_products/`:
  - `query.go`: ListProductsQuery with pagination and filtering
  - `dto.go`: ProductListDTO with items and pagination info

### Phase 6: Infrastructure - CommitPlan & Clock

**Location:** `internal/pkg/`

**6.1 Committer Wrapper**
- `committer/plan.go`: Typed wrapper around CommitPlan
  - `Apply(ctx, plan) error`
  - Handle Spanner-specific transaction logic

**6.2 Clock Abstraction**
- `clock/clock.go`: Time interface for testability
  - `Now() time.Time`
  - Real implementation and mock for testing

### Phase 7: gRPC Transport Layer

**Location:** `proto/` and `internal/transport/grpc/product/`

**7.1 Protocol Buffers**
- `proto/product/v1/product_service.proto`: gRPC service definition
  - Request/Response messages for all operations
  - ProductService with 8 RPCs (6 commands + 2 queries)
  - Money message type (numerator, denominator)
  - Product message with all fields

**7.2 gRPC Handlers**
- `handler.go`: Main handler struct with dependencies
- `create.go`: CreateProduct RPC handler
- `update.go`: UpdateProduct RPC handler
- `activate.go`: Activate/Deactivate handlers
- `discount.go`: ApplyDiscount/RemoveDiscount handlers
- `get.go`: GetProduct query handler
- `list.go`: ListProducts query handler
- `mappers.go`: Bidirectional mapping between proto ↔ domain/DTOs
- `errors.go`: Map domain errors → gRPC status codes
- `validation.go`: Proto request validation

### Phase 8: Dependency Injection & Service Setup

**Location:** `internal/services/`

**8.1 Service Options**
- `options.go`: DI container struct
  - Initialize Spanner client
  - Create repository instances
  - Create use case instances
  - Wire dependencies
  - Return configured service

**8.2 Main Entry Point**
- `cmd/server/main.go`:
  - Load configuration (Spanner connection string, gRPC port)
  - Initialize Spanner client
  - Create service options (DI)
  - Start gRPC server
  - Graceful shutdown handling

### Phase 9: Comprehensive Testing Infrastructure

**9.1 Test Organization**

Create three test categories with proper isolation:

**A. Unit Tests** - `internal/app/product/domain/*_test.go`
- Pure Go tests, no external dependencies
- Test files alongside implementation
- Fast execution (< 1 second total)
- Tests:
  - `money_test.go`: Money value object calculations, precision, edge cases
  - `discount_test.go`: Discount validation, date ranges, percentage bounds
  - `product_test.go`: Aggregate business logic, state transitions
  - `change_tracker_test.go`: Dirty field tracking
  - `services/pricing_calculator_test.go`: Price calculation with discounts

**B. Integration Tests** - `tests/integration/`
- Test repository layer with real Spanner
- Uses Docker Compose test environment
- Build tag: `//go:build integration`
- Tests:
  - `product_repo_test.go`: CRUD operations, change tracking, mutations
  - `outbox_repo_test.go`: Event storage
  - `read_model_test.go`: Query operations, DTOs

**C. E2E Tests** - `tests/e2e/`
- Full vertical slice testing (usecase → repo → DB)
- Uses Docker Compose test environment
- Tests complete business scenarios
- Each test is self-contained with setup/teardown

**9.2 Test Infrastructure & Helpers**

**Location:** `tests/testutil/`

**A. `testutil/spanner.go` - Spanner Test Utilities**
```go
// SetupSpannerTest: Creates test Spanner client and ensures clean state
func SetupSpannerTest(t *testing.T) (*spanner.Client, func())

// CreateTestInstance: Creates isolated test instance/database
func CreateTestInstance(t *testing.T, instanceID, dbID string) error

// CleanDatabase: Truncates all tables for test isolation
func CleanDatabase(t *testing.T, client *spanner.Client)
```

**B. `testutil/fixtures.go` - Test Data Fixtures**
```go
// CreateTestProduct: Helper to create product for testing
func CreateTestProduct(t *testing.T, client *spanner.Client, name string) string

// CreateTestProductWithDiscount: Create product with discount
func CreateTestProductWithDiscount(t *testing.T, client *spanner.Client) string

// AssertOutboxEvent: Verify outbox event exists
func AssertOutboxEvent(t *testing.T, client *spanner.Client, eventType string)
```

**C. `testutil/clock.go` - Mock Clock for Testing**
```go
type MockClock struct {
    current time.Time
}

func (m *MockClock) Now() time.Time
func (m *MockClock) Set(t time.Time)
func (m *MockClock) Advance(d time.Duration)
```

**9.3 E2E Test Suite**

**Location:** `tests/e2e/product_test.go`

Comprehensive test scenarios with proper setup/teardown:

```go
func TestMain(m *testing.M) {
    // Setup: Ensure Spanner emulator is running
    // Run migrations
    // Exit
}

func setupTest(t *testing.T) (*Services, func()) {
    // Create Spanner client
    // Clean database
    // Wire up all dependencies (repos, usecases, queries)
    // Return services + cleanup function
}
```

**Test Scenarios:**
1. **TestProductCreationFlow**
   - Create product via usecase
   - Verify product exists via query
   - Assert outbox event created
   - Validate all fields persisted correctly

2. **TestProductUpdateFlow**
   - Create product
   - Update name, description, category
   - Verify only dirty fields updated
   - Check UpdatedAt timestamp changed

3. **TestDiscountApplicationFlow**
   - Create active product
   - Apply 20% discount
   - Verify effective price calculation
   - Check discount dates stored
   - Verify DiscountAppliedEvent in outbox

4. **TestDiscountRemovalFlow**
   - Create product with discount
   - Remove discount
   - Verify base price returned
   - Check DiscountRemovedEvent

5. **TestProductActivationDeactivation**
   - Create inactive product
   - Activate product
   - Verify status changed
   - Deactivate product
   - Verify status changed back

6. **TestBusinessRuleValidations**
   - Cannot apply discount to inactive product
   - Cannot apply invalid discount (> 100%)
   - Cannot apply discount with end date before start date
   - Cannot create product with negative price

7. **TestProductArchiving**
   - Archive product (soft delete)
   - Verify archived_at set
   - Verify not returned in list queries
   - Can still retrieve by ID

8. **TestListProductsWithPagination**
   - Create 25 products
   - List with page size 10
   - Verify pagination works
   - Test cursor-based pagination

9. **TestListProductsWithCategoryFilter**
   - Create products in multiple categories
   - Filter by category
   - Verify only matching products returned

10. **TestConcurrentUpdates** (if optimistic locking implemented)
    - Load same product in two goroutines
    - Update concurrently
    - Verify proper conflict detection

**9.4 Integration Test Suite**

**Location:** `tests/integration/repository_test.go`

Tests repository layer in isolation:

1. **TestProductRepository_Insert**
   - InsertMut returns valid mutation
   - Apply mutation and verify in DB
   - Check all fields persisted

2. **TestProductRepository_Update**
   - Only dirty fields included in mutation
   - UpdatedAt always updated
   - Clean change tracker after read

3. **TestProductRepository_GetByID**
   - Reconstruct domain aggregate from DB
   - All fields mapped correctly
   - Change tracker initialized as clean

4. **TestOutboxRepository_Insert**
   - Event stored with correct payload
   - Status set to pending
   - JSON serialization correct

**9.5 Test Configuration**

**A. `.env.test` - Test Environment Variables**
```bash
SPANNER_EMULATOR_HOST=localhost:19010
TEST_INSTANCE_ID=test-instance
TEST_DATABASE_ID=product-catalog-test
TEST_PROJECT_ID=test-project
```

**B. `scripts/run_tests.sh` - CI Test Runner**
```bash
#!/bin/bash
# Complete test suite for CI/CD
set -e

echo "Starting test environment..."
make docker-test-up
make migrate-test

echo "Running unit tests..."
make test-unit

echo "Running integration tests..."
make test-integration

echo "Running E2E tests..."
make test-e2e

echo "Generating coverage report..."
make test-coverage

echo "Cleaning up..."
make docker-test-down

echo "All tests passed!"
```

**9.6 GitHub Actions Workflow** (Optional)

**Location:** `.github/workflows/test.yml`

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - name: Run tests
        run: make test-docker
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

### Phase 10: Documentation & Final Touches

**10.1 Update README.md**
- Project overview
- Prerequisites (Go 1.25, Docker)
- Quick start guide
- Running the service
- Running tests
- API documentation reference
- Design decisions and trade-offs

**10.2 Configuration**
- `.env.example`: Environment variable template
- Configuration loading in main.go

## Critical Files to Create

### Domain Layer (Pure Go, no external deps)
1. `internal/app/product/domain/money.go`
2. `internal/app/product/domain/discount.go`
3. `internal/app/product/domain/product.go`
4. `internal/app/product/domain/change_tracker.go`
5. `internal/app/product/domain/domain_events.go`
6. `internal/app/product/domain/domain_errors.go`
7. `internal/app/product/domain/services/pricing_calculator.go`

### Infrastructure
8. `internal/models/m_product/data.go`
9. `internal/models/m_product/fields.go`
10. `internal/models/m_product/model.go`
11. `internal/models/m_outbox/data.go`
12. `internal/models/m_outbox/fields.go`
13. `internal/models/m_outbox/model.go`

### Application Layer
14. `internal/app/product/contracts/product_repo.go`
15. `internal/app/product/contracts/read_model.go`
16. `internal/app/product/contracts/outbox_repo.go`
17. `internal/app/product/repo/product_repo.go`
18. `internal/app/product/repo/outbox_repo.go`
19. `internal/app/product/repo/read_model.go`

### Use Cases (7 interactors)
20-26. `internal/app/product/usecases/{create,update,activate,deactivate,apply_discount,remove_discount,archive}_product/interactor.go`

### Queries (2 query handlers)
27-28. `internal/app/product/queries/{get,list}_product/query.go` + `dto.go`

### gRPC Transport
29. `proto/product/v1/product_service.proto`
30. `internal/transport/grpc/product/handler.go`
31-38. `internal/transport/grpc/product/{create,update,activate,discount,get,list,mappers,errors,validation}.go`

### Infrastructure Support
39. `internal/pkg/committer/plan.go`
40. `internal/pkg/clock/clock.go`
41. `internal/services/options.go`
42. `cmd/server/main.go`

### Configuration & Build
43. `go.mod`
44. `Makefile` (comprehensive with all test targets)
45. `docker-compose.yml` (dev environment)
46. `docker-compose.test.yml` (test environment)
47. `Dockerfile.test` (test container)
48. `.env.example`
49. `.env.test`

### Database & Migrations
50. `migrations/001_initial_schema.sql`
51. `scripts/migrate.sh` (migration runner)
52. `scripts/migrate_test.sh` (test DB migration)
53. `scripts/setup_test_db.sh` (test DB setup)
54. `scripts/cleanup_test_db.sh` (test DB cleanup)
55. `scripts/run_tests.sh` (CI test runner)

### Tests - Unit Tests
56. `internal/app/product/domain/money_test.go`
57. `internal/app/product/domain/discount_test.go`
58. `internal/app/product/domain/product_test.go`
59. `internal/app/product/domain/change_tracker_test.go`
60. `internal/app/product/domain/services/pricing_calculator_test.go`

### Tests - Integration Tests
61. `tests/integration/product_repo_test.go`
62. `tests/integration/outbox_repo_test.go`
63. `tests/integration/read_model_test.go`

### Tests - E2E Tests
64. `tests/e2e/product_test.go`
65. `tests/e2e/product_creation_test.go`
66. `tests/e2e/product_update_test.go`
67. `tests/e2e/discount_test.go`
68. `tests/e2e/activation_test.go`
69. `tests/e2e/list_test.go`

### Tests - Test Utilities
70. `tests/testutil/spanner.go` (Spanner test helpers)
71. `tests/testutil/fixtures.go` (test data fixtures)
72. `tests/testutil/clock.go` (mock clock)
73. `tests/testutil/assertions.go` (custom assertions)

### CI/CD (Optional)
74. `.github/workflows/test.yml` (GitHub Actions)
75. `.github/workflows/lint.yml` (Linting workflow)

## Comprehensive Verification Plan

### Step 1: Environment Setup Verification
```bash
# Verify Go version
go version  # Should show 1.25

# Install all dependencies
make deps
make tools

# Verify tools installed
protoc --version
grpcurl --version
```

### Step 2: Docker Infrastructure Verification
```bash
# Start development Spanner emulator
make docker-up

# Verify emulator is running and healthy
docker ps | grep spanner
curl http://localhost:9020  # Should return emulator info

# Start test environment
make docker-test-up

# Verify test emulator on different ports
curl http://localhost:19020

# Clean up
make docker-down
make docker-test-down
```

### Step 3: Code Generation Verification
```bash
# Generate protobuf code
make proto

# Verify generated files exist
ls -la proto/product/v1/*.pb.go

# Check for compilation errors
go build ./...
```

### Step 4: Database Migration Verification
```bash
# Start dev emulator
make docker-up

# Run migrations
make migrate

# Verify migration script success
echo $?  # Should be 0

# Clean up
make docker-down
```

### Step 5: Unit Test Verification (No DB Required)
```bash
# Run unit tests only (fast, isolated)
make test-unit

# Expected results:
# - All domain tests pass
# - Money calculations correct (precision testing)
# - Discount validation works
# - Product state transitions valid
# - Change tracker functionality verified
# - No external dependencies used

# Verify test output shows PASS for all packages
# Tests should complete in < 1 second
```

### Step 6: Integration Test Verification (With Spanner)
```bash
# Run integration tests (automatically starts/stops test DB)
make test-integration

# This will:
# 1. Start Docker Compose test environment
# 2. Wait for Spanner emulator health check
# 3. Run migrations on test DB
# 4. Execute integration tests
# 5. Clean up test environment

# Expected results:
# - Repository insert/update mutations work correctly
# - Change tracking optimizes updates
# - Domain aggregates reconstruct from DB
# - Outbox events persist
# - All DB operations succeed
```

### Step 7: E2E Test Verification (Full Stack)
```bash
# Run end-to-end tests
make test-e2e

# Expected test results:
# ✓ TestProductCreationFlow - product created, event in outbox
# ✓ TestProductUpdateFlow - only dirty fields updated
# ✓ TestDiscountApplicationFlow - price calculated correctly
# ✓ TestDiscountRemovalFlow - reverts to base price
# ✓ TestProductActivationDeactivation - status transitions work
# ✓ TestBusinessRuleValidations - domain errors raised correctly
# ✓ TestProductArchiving - soft delete works
# ✓ TestListProductsWithPagination - pagination correct
# ✓ TestListProductsWithCategoryFilter - filtering works
# ✓ TestConcurrentUpdates - no race conditions

# All tests should pass with race detector enabled (-race flag)
```

### Step 8: Full Test Suite with Coverage
```bash
# Run all tests with coverage report
make test-coverage

# Verify coverage report generated
open coverage.html  # View in browser

# Expected coverage targets:
# - Domain layer: > 90%
# - Repository layer: > 80%
# - Use cases: > 85%
# - Overall: > 80%

# Check coverage summary
make test-coverage | grep "total:"
```

### Step 9: Docker-Based Testing (CI Simulation)
```bash
# Run tests exactly as CI would run them
make test-docker

# This runs tests inside Docker container
# Verifies:
# - Clean environment testing
# - No reliance on local setup
# - Reproducible results
# - All dependencies available

# Container should exit with code 0 (success)
echo $?
```

### Step 10: Build and Service Startup Verification
```bash
# Build the service binary
make build

# Verify binary created
ls -lh bin/server

# Start development environment
make docker-up
make migrate

# Start the gRPC server
make run

# In another terminal, verify server is running
grpcurl -plaintext localhost:9090 list

# Should show: product.v1.ProductService
```

### Step 11: Manual API Testing with grpcurl
```bash
# Ensure server is running from Step 10

# 1. Create a product
grpcurl -plaintext -d '{
  "name": "MacBook Pro",
  "description": "16-inch laptop",
  "category": "electronics",
  "base_price": {"numerator": 249900, "denominator": 100}
}' localhost:9090 product.v1.ProductService/CreateProduct

# Save the returned product_id

# 2. Get the product
grpcurl -plaintext -d '{
  "product_id": "<product-id-from-step-1>"
}' localhost:9090 product.v1.ProductService/GetProduct

# 3. Apply a discount
grpcurl -plaintext -d '{
  "product_id": "<product-id>",
  "discount_percent": 20,
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-12-31T23:59:59Z"
}' localhost:9090 product.v1.ProductService/ApplyDiscount

# 4. Verify effective price (should be $1999.20)
grpcurl -plaintext -d '{
  "product_id": "<product-id>"
}' localhost:9090 product.v1.ProductService/GetProduct

# 5. List products
grpcurl -plaintext -d '{
  "page_size": 10,
  "category": "electronics"
}' localhost:9090 product.v1.ProductService/ListProducts

# 6. Activate product
grpcurl -plaintext -d '{
  "product_id": "<product-id>"
}' localhost:9090 product.v1.ProductService/ActivateProduct

# All requests should return success (no gRPC error codes)
```

### Step 12: Architecture Pattern Verification
```bash
# Verify domain layer purity (no forbidden imports)
grep -r "context\." internal/app/product/domain/ && echo "FAIL: Context found in domain" || echo "PASS: Domain is pure"
grep -r "spanner" internal/app/product/domain/ && echo "FAIL: Spanner in domain" || echo "PASS: No DB in domain"
grep -r "proto" internal/app/product/domain/ && echo "FAIL: Proto in domain" || echo "PASS: No proto in domain"

# Verify Golden Mutation Pattern in use cases
grep -r "plan.Add" internal/app/product/usecases/ || echo "FAIL: Not using CommitPlan"

# Verify repositories return mutations
grep -r "func.*Mut.*spanner.Mutation" internal/app/product/repo/ || echo "FAIL: Repos not returning mutations"

# Verify CQRS separation
ls internal/app/product/usecases/  # Should show command directories
ls internal/app/product/queries/   # Should show query directories

# Verify all files compile without errors
go build ./...
echo "PASS: All files compile successfully"
```

### Step 13: Code Quality Verification
```bash
# Format check
make fmt
git diff --exit-code  # Should show no changes

# Run go vet
make vet

# Run linter (if golangci-lint installed)
make lint

# All should pass with no errors
```

### Step 14: Cleanup Verification
```bash
# Stop all services
make docker-down
make docker-test-down

# Verify no containers running
docker ps | grep spanner  # Should be empty

# Clean all artifacts
make clean-all

# Verify cleanup
ls bin/  # Should not exist or be empty
ls coverage.*  # Should not exist
```

### Step 15: Full Workflow End-to-End
```bash
# This simulates a complete development workflow
make clean-all           # Start fresh
make deps                # Install dependencies
make tools               # Install dev tools
make proto               # Generate code
make docker-up           # Start dev environment
make migrate             # Setup database
make test-all            # Run all tests (takes 30-60 seconds)
make build               # Build binary
make run                 # Start server

# In another terminal:
# Run manual API tests from Step 11

# Cleanup
make docker-down
make clean
```

### Success Criteria

All verification steps should complete successfully with:
- ✅ All unit tests passing (< 1 second)
- ✅ All integration tests passing (< 10 seconds)
- ✅ All E2E tests passing (< 30 seconds)
- ✅ Code coverage > 80%
- ✅ No race conditions detected
- ✅ Domain layer has zero external dependencies
- ✅ Golden Mutation Pattern used in all usecases
- ✅ Repositories return mutations only
- ✅ gRPC server starts successfully
- ✅ Manual API calls work correctly
- ✅ Docker-based tests pass (CI simulation)
- ✅ All code quality checks pass (fmt, vet, lint)

## Trade-offs and Decisions

1. **Go 1.25 vs 1.21**: Using latest Go 1.25 for improved performance and language features
2. **Spanner Emulator**: Local development uses emulator; production would use real Spanner
3. **No Optimistic Locking Initially**: Can add version field later if concurrent updates become an issue
4. **Simple Outbox**: Events stored but not processed; background processor out of scope
5. **In-memory config**: Simple configuration; could add config file or env loading as needed
6. **Single Discount Rule**: One active discount per product; could extend to multiple overlapping discounts
7. **gRPC Only**: No REST endpoints; could add gRPC-gateway later if needed
8. **Manual Migration**: Simple SQL file; could add migration tool (golang-migrate) for production

## Implementation Order Rationale

1. **Foundation First**: Setup project structure, dependencies, and database
2. **Inside-Out**: Domain layer first (pure logic), then infrastructure, then transport
3. **Vertical Slices**: Complete one feature end-to-end before moving to next
4. **Test-Driven**: Write tests alongside implementation for immediate feedback
5. **Layer by Layer**: Complete each architectural layer before moving up the stack
