.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help

# ==================================================================================== #
# DEPENDENCY MANAGEMENT
# ==================================================================================== #

.PHONY: deps
deps: ## Install Go dependencies
	go mod download
	go mod tidy

.PHONY: tools
tools: ## Install development tools
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# ==================================================================================== #
# CODE GENERATION
# ==================================================================================== #

.PHONY: proto
proto: ## Generate protobuf code
	protoc --go_out=. --go-grpc_out=. proto/product/v1/*.proto

.PHONY: generate
generate: proto ## Run all code generation

# ==================================================================================== #
# DOCKER & INFRASTRUCTURE
# ==================================================================================== #

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

# ==================================================================================== #
# DATABASE MIGRATIONS
# ==================================================================================== #

.PHONY: migrate
migrate: ## Run migrations on dev database
	SPANNER_EMULATOR_HOST=localhost:9010 ./scripts/migrate.sh dev-instance product-catalog-db

.PHONY: migrate-test
migrate-test: ## Run migrations on test database
	SPANNER_EMULATOR_HOST=localhost:19010 ./scripts/setup_test_db.sh

.PHONY: migrate-clean
migrate-clean: ## Clean and recreate test database
	SPANNER_EMULATOR_HOST=localhost:19010 ./scripts/cleanup_test_db.sh
	SPANNER_EMULATOR_HOST=localhost:19010 ./scripts/setup_test_db.sh

# ==================================================================================== #
# TESTING
# ==================================================================================== #

.PHONY: test
test: test-unit ## Run all tests (unit + integration + e2e)

.PHONY: test-unit
test-unit: ## Run unit tests (domain layer only, no DB)
	go test -v -race -count=1 ./internal/app/product/domain/...

.PHONY: test-integration
test-integration: docker-test-up migrate-test ## Run integration tests (with real Spanner)
	SPANNER_EMULATOR_HOST=localhost:19010 \
	go test -v -race -count=1 -tags=integration ./tests/integration/... || ($(MAKE) docker-test-down && exit 1)
	$(MAKE) docker-test-down

.PHONY: test-e2e
test-e2e: docker-test-up migrate-test ## Run E2E tests
	SPANNER_EMULATOR_HOST=localhost:19010 \
	go test -v -race -count=1 -timeout=5m ./tests/e2e/... || ($(MAKE) docker-test-down && exit 1)
	$(MAKE) docker-test-down

.PHONY: test-all
test-all: test-unit test-integration test-e2e ## Run all test suites

.PHONY: test-coverage
test-coverage: docker-test-up migrate-test ## Run tests with coverage report
	SPANNER_EMULATOR_HOST=localhost:19010 \
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./... || ($(MAKE) docker-test-down && exit 1)
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out
	$(MAKE) docker-test-down

.PHONY: test-docker
test-docker: ## Run tests inside Docker container (CI simulation)
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down -v

.PHONY: docker-test-unit
docker-test-unit: ## Run unit tests in Docker
	docker-compose -f docker-compose.test.yml run --rm test-unit

.PHONY: docker-test-integration
docker-test-integration: ## Run integration tests in Docker with Spanner
	docker-compose -f docker-compose.test.yml up -d spanner-test
	docker-compose -f docker-compose.test.yml run --rm test-integration
	docker-compose -f docker-compose.test.yml down -v

.PHONY: docker-test-e2e
docker-test-e2e: ## Run E2E tests in Docker with Spanner
	docker-compose -f docker-compose.test.yml up -d spanner-test
	docker-compose -f docker-compose.test.yml run --rm test-e2e
	docker-compose -f docker-compose.test.yml down -v

.PHONY: docker-test-all
docker-test-all: ## Run complete test suite in Docker
	docker-compose -f docker-compose.test.yml up --build -d spanner-test
	@echo "Waiting for Spanner emulator..."
	@sleep 5
	docker-compose -f docker-compose.test.yml run --rm test-all
	docker-compose -f docker-compose.test.yml down -v

.PHONY: docker-test-coverage
docker-test-coverage: ## Generate coverage report in Docker
	mkdir -p coverage-reports
	docker-compose -f docker-compose.test.yml up -d spanner-test
	@echo "Waiting for Spanner emulator..."
	@sleep 5
	docker-compose -f docker-compose.test.yml run --rm test-coverage
	docker-compose -f docker-compose.test.yml down -v
	@echo ""
	@echo "Coverage report generated:"
	@echo "  - coverage-reports/coverage.out (raw)"
	@echo "  - coverage-reports/coverage.html (view in browser)"

.PHONY: docker-test-watch
docker-test-watch: ## Watch and re-run unit tests in Docker on file changes
	docker-compose -f docker-compose.test.yml run --rm test-unit
	@echo "Note: For true watch mode, use 'make test-watch' on host"

# ==================================================================================== #
# BUILD & RUN
# ==================================================================================== #

.PHONY: build
build: ## Build the service binary
	go build -o bin/server ./cmd/server/

.PHONY: run
run: ## Run the gRPC server locally
	go run ./cmd/server/

.PHONY: run-dev
run-dev: docker-up migrate ## Start dev environment and run server
	go run ./cmd/server/

# ==================================================================================== #
# CODE QUALITY
# ==================================================================================== #

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

# ==================================================================================== #
# CLEANUP
# ==================================================================================== #

.PHONY: clean
clean: ## Clean build artifacts and test data
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker-compose down -v
	docker-compose -f docker-compose.test.yml down -v

.PHONY: clean-all
clean-all: clean ## Deep clean including Go cache
	go clean -cache -testcache -modcache
