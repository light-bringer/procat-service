# Product Catalog Service

A Go microservice implementing product management and pricing using **Domain-Driven Design (DDD)**, **Clean Architecture**, and the **Golden Mutation Pattern** with Google Cloud Spanner.

## 🎯 Features

- **Product Management**: Create, update, activate, deactivate, and archive products
- **Dynamic Pricing**: Apply time-bound percentage discounts with precise decimal arithmetic
- **CQRS Pattern**: Separate command (write) and query (read) models for optimal performance
- **Event Sourcing**: Transactional outbox pattern for reliable event publishing
- **Change Tracking**: Optimized database updates by tracking dirty fields
- **gRPC API**: High-performance protocol buffer-based API with 9 operations

## 🏗️ Architecture

### Tech Stack

- **Language**: Go 1.25+
- **Database**: Google Cloud Spanner (with emulator for local dev)
- **API**: gRPC with Protocol Buffers
- **Testing**: testify, Spanner emulator

### Key Patterns

#### 1. Domain-Driven Design (DDD)

The domain layer (`internal/app/product/domain/`) contains pure business logic with zero external dependencies:

- ✅ Pure Go types and logic
- ✅ `math/big.Rat` for precise money calculations
- ✅ Domain events as simple structs
- ❌ No `context.Context`
- ❌ No database imports
- ❌ No proto definitions

#### 2. Clean Architecture Layers

```
cmd/                    # Entry points
internal/
├── app/product/
│   ├── domain/         # Pure business logic (innermost layer)
│   ├── usecases/       # Command handlers (application layer)
│   ├── queries/        # Query handlers (application layer)
│   ├── contracts/      # Repository interfaces
│   └── repo/           # Spanner implementations (infrastructure layer)
├── models/             # Database models (m_product, m_outbox)
├── transport/grpc/     # gRPC handlers (outermost layer)
└── pkg/                # Shared utilities (clock, committer)
```

#### 3. Golden Mutation Pattern

Every write operation follows this exact sequence:

```go
func (it *Interactor) Execute(ctx context.Context, req Request) (string, error) {
    // 1. Load or create domain aggregate
    product := domain.NewProduct(...)

    // 2. Call domain methods (validation happens here)
    if err := product.ApplyDiscount(discount, it.clock.Now()); err != nil {
        return "", err
    }

    // 3. Create commit plan
    plan := committer.NewPlan()

    // 4. Repository RETURNS mutations (doesn't apply)
    if mut := it.repo.UpdateMut(product); mut != nil {
        plan.Add(mut)
    }

    // 5. Add outbox events
    for _, event := range product.DomainEvents() {
        plan.Add(it.outboxRepo.InsertMut(enrichEvent(event)))
    }

    // 6. Usecase APPLIES plan (not handler!)
    if err := it.committer.Apply(ctx, plan); err != nil {
        return "", err
    }

    return product.ID(), nil
}
```

**Key Rules:**
- Use cases apply CommitPlans, handlers never do
- Repositories return mutations, never apply them
- All writes go through domain aggregate methods
- Domain events stored in outbox within same transaction

#### 4. CQRS Separation

**Commands (Write Operations):**
- Live in `usecases/*/interactor.go`
- MUST go through domain aggregate
- Use CommitPlan for atomic transactions
- Return minimal data (ID or error)

**Queries (Read Operations):**
- Live in `queries/*/query.go`
- MAY bypass domain for performance
- Use DTOs for data transfer
- Direct database access via read model

#### 5. Precise Money Calculations

Uses `math/big.Rat` to avoid floating-point precision errors:

```go
// Store as numerator/denominator
basePrice, _ := domain.NewMoney(249900, 100) // $2499.00

// Apply 20% discount
discount, _ := domain.NewDiscount(20, startDate, endDate)
finalPrice := discount.Apply(basePrice) // $1999.20 (exact)
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25 or later
- Docker and Docker Compose
- protoc (Protocol Buffers compiler)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd procat-service

# Install dependencies
make deps

# Install development tools (protoc plugins, grpcurl)
make tools

# Generate protobuf code
make proto
```

### Running Locally

```bash
# Start Spanner emulator
make docker-up

# Run database migrations
make migrate

# Run the service
make run

# Service will be available at localhost:9090
```

### Testing

```bash
# Run unit tests (domain layer only, fast)
make test-unit

# Run all tests with Spanner emulator
make test-all

# Run with coverage report
make test-coverage

# Run tests in Docker (CI simulation)
make test-docker
```

## 📡 API

### gRPC Service: ProductService

**Commands:**
- `CreateProduct`: Create a new product
- `UpdateProduct`: Update product details
- `ActivateProduct`: Activate product for sale
- `DeactivateProduct`: Deactivate product
- `ApplyDiscount`: Apply time-bound percentage discount
- `RemoveDiscount`: Remove active discount
- `ArchiveProduct`: Soft delete product

**Queries:**
- `GetProduct`: Retrieve product by ID
- `ListProducts`: List products with filtering and pagination

### Example: Create Product

```bash
grpcurl -plaintext -d '{
  "name": "MacBook Pro",
  "description": "16-inch laptop",
  "category": "electronics",
  "base_price": {"numerator": 249900, "denominator": 100}
}' localhost:9090 product.v1.ProductService/CreateProduct
```

### Example: Apply Discount

```bash
grpcurl -plaintext -d '{
  "product_id": "<product-id>",
  "discount_percent": 20,
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-12-31T23:59:59Z"
}' localhost:9090 product.v1.ProductService/ApplyDiscount
```

### Example: Get Product

```bash
grpcurl -plaintext -d '{
  "product_id": "<product-id>"
}' localhost:9090 product.v1.ProductService/GetProduct
```

## 🗄️ Database Schema

### products table

```sql
- product_id (STRING)
- name (STRING)
- description (STRING)
- category (STRING)
- base_price_numerator (INT64)
- base_price_denominator (INT64)
- discount_percent (INT64, nullable)
- discount_start_date (TIMESTAMP, nullable)
- discount_end_date (TIMESTAMP, nullable)
- status (STRING) - "active", "inactive", "archived"
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
- archived_at (TIMESTAMP, nullable)
```

### outbox_events table

```sql
- event_id (STRING)
- event_type (STRING)
- aggregate_id (STRING)
- payload (JSON)
- status (STRING) - "pending", "processing", "completed", "failed"
- created_at (TIMESTAMP)
- processed_at (TIMESTAMP, nullable)
- retry_count (INT64)
- error_message (STRING, nullable)
```

## 🧪 Testing Strategy

### Unit Tests
- Domain layer only (pure Go, no DB)
- Fast execution (< 1 second)
- Test business logic, value objects, aggregates

```bash
make test-unit
```

### Integration Tests
- Repository layer with real Spanner emulator
- Test CRUD operations, change tracking, mutations

```bash
make test-integration
```

### E2E Tests
- Full vertical slice (usecase → repo → DB)
- Test complete business scenarios
- Verify Golden Mutation Pattern implementation

```bash
make test-e2e
```

## 📦 Makefile Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make deps` | Install Go dependencies |
| `make tools` | Install dev tools (protoc, grpcurl) |
| `make proto` | Generate protobuf code |
| `make docker-up` | Start Spanner emulator |
| `make docker-down` | Stop Spanner emulator |
| `make migrate` | Run database migrations |
| `make build` | Build service binary |
| `make run` | Run gRPC server |
| `make run-dev` | Start dev environment and run |
| `make test-unit` | Run unit tests |
| `make test-all` | Run all tests |
| `make test-coverage` | Run tests with coverage report |
| `make fmt` | Format code |
| `make lint` | Run linters |
| `make clean` | Clean build artifacts |

## ⚙️ Configuration

Environment variables (see `.env.example`):

```bash
# Spanner Configuration
SPANNER_EMULATOR_HOST=localhost:9010
SPANNER_DATABASE=projects/test-project/instances/dev-instance/databases/product-catalog-db

# gRPC Configuration
GRPC_PORT=9090
```

## 🏛️ Design Decisions

### Why Clean Architecture?
- **Testability**: Domain logic can be tested in isolation
- **Independence**: Business rules don't depend on frameworks or databases
- **Flexibility**: Easy to swap infrastructure components

### Why Golden Mutation Pattern?
- **Atomic Transactions**: All changes committed together or rolled back
- **Event Consistency**: Events and state changes in same transaction
- **Testability**: Repositories return mutations for easy testing

### Why CQRS?
- **Performance**: Queries optimized for reads, commands for consistency
- **Scalability**: Read and write models can scale independently
- **Simplicity**: Clear separation of concerns

### Why big.Rat for Money?
- **Precision**: No floating-point rounding errors
- **Correctness**: Financial calculations always exact
- **Auditability**: Stored as numerator/denominator for transparency

## 🚧 Trade-offs

- **No Optimistic Locking**: Can add version field if needed for concurrent updates
- **Simple Outbox**: Events stored but not processed (background processor out of scope)
- **In-memory Config**: Could add config file or vault integration
- **Single Discount**: One active discount per product (could extend to multiple)
- **gRPC Only**: No REST endpoints (could add gRPC-gateway if needed)

## 📚 Further Reading

- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design by Eric Evans](https://www.domainlanguage.com/ddd/)
- [CQRS Pattern by Martin Fowler](https://martinfowler.com/bliki/CQRS.html)
- [Transactional Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)

## 📄 License

MIT License - see LICENSE file for details
