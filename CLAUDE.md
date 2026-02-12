# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Product Catalog Service - A Go microservice implementing product management and pricing using Domain-Driven Design (DDD), Clean Architecture, and the Golden Mutation Pattern with Google Cloud Spanner.

**Tech Stack**: Go 1.25+, gRPC, Google Cloud Spanner, github.com/Vektor-AI/commitplan

## Development Commands

```bash
# Start Spanner emulator
docker-compose up -d

# Initialize Go module (if needed)
go mod init github.com/light-bringer/procat-service
go mod tidy

# Run migrations
make migrate

# Run tests
make test
go test ./tests/e2e/... -v

# Run server
make run
go run cmd/server/main.go

# Generate protobuf
protoc --go_out=. --go-grpc_out=. proto/product/v1/*.proto

# Stop Spanner emulator
docker-compose down
```

## Critical Architecture Patterns

### 1. Domain Layer Purity (NON-NEGOTIABLE)

The domain layer (`internal/app/product/domain/`) must be pure Go business logic only:

**ALLOWED in domain:**
- Pure Go types and logic
- `math/big` for money calculations (use `*big.Rat`)
- `time.Time` for dates
- Domain errors as sentinel values (`var ErrProductNotActive = errors.New(...)`)
- Simple structs for domain events

**FORBIDDEN in domain:**
- `context.Context`
- Database imports (`cloud.google.com/go/spanner`, `database/sql`)
- Proto definitions
- Any external frameworks or infrastructure concerns

### 2. Golden Mutation Pattern

Every write operation MUST follow this exact sequence:

```go
func (it *Interactor) Execute(ctx context.Context, req Request) (string, error) {
    // 1. Load or create domain aggregate
    product := domain.NewProduct(...)

    // 2. Call domain methods (validation happens here)
    if err := product.ApplyDiscount(discount, it.clock.Now()); err != nil {
        return "", err
    }

    // 3. Create commit plan
    plan := commitplan.NewPlan()

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
- Usecases apply CommitPlans, handlers never do
- Repositories return mutations, never apply them
- All writes go through domain aggregate methods
- Domain events stored in outbox within same transaction

### 3. CQRS Separation

**Commands (Write Operations):**
- Live in `internal/app/product/usecases/*/interactor.go`
- MUST go through domain aggregate
- Use CommitPlan for atomic transactions
- Return error only (or minimal reply with IDs)

**Queries (Read Operations):**
- Live in `internal/app/product/queries/*/query.go`
- MAY bypass domain for performance
- Use DTOs defined in `dto.go`
- Direct database access via read model contracts
- NO mutations or side effects

### 4. Repository Pattern

Repositories use change tracking to build targeted updates:

```go
func (r *ProductRepo) UpdateMut(p *domain.Product) *spanner.Mutation {
    updates := make(map[string]interface{})

    // Only update dirty fields
    if p.Changes().Dirty(domain.FieldName) {
        updates[m_product.Name] = p.Name()
    }

    if len(updates) == 0 {
        return nil // No changes = no mutation
    }

    updates[m_product.UpdatedAt] = time.Now()
    return r.model.UpdateMut(p.ID(), updates)
}
```

**Repository responsibilities:**
- Map domain entities ↔ database models
- Use `m_product` and `m_outbox` model facades for type-safe field names
- Read change tracker to optimize updates
- Return `*spanner.Mutation`, never execute them

### 5. Money Calculations

Always use `math/big.Rat` for precision:

```go
// Store as numerator/denominator in DB
type Money struct {
    rat *big.Rat
}

// Database representation
base_price_numerator INT64
base_price_denominator INT64

// Discount calculation
discount := new(big.Rat).Mul(price, big.NewRat(20, 100)) // 20% off
finalPrice := new(big.Rat).Sub(price, discount)
```

### 6. Change Tracking

Domain aggregates track dirty fields to optimize updates:

```go
type ChangeTracker struct {
    dirtyFields map[string]bool
}

// In domain methods
func (p *Product) SetName(name string) error {
    p.name = name
    p.changes.MarkDirty(FieldName)
    return nil
}
```

### 7. Transactional Outbox

Domain events are intents captured during business operations:

```go
// Domain captures simple event
p.events = append(p.events, &ProductCreatedEvent{
    ProductID: p.id,
    Name:      p.name,
})

// Usecase enriches with metadata
func enrichEvent(event DomainEvent) *m_outbox.OutboxEvent {
    return &m_outbox.OutboxEvent{
        EventID:     uuid.New().String(),
        EventType:   event.Type(),
        AggregateID: event.AggregateID(),
        Payload:     marshalPayload(event),
        Status:      "pending",
        CreatedAt:   time.Now(),
    }
}
```

Events are stored in `outbox_events` table within the same transaction as the aggregate changes.

## Project Structure

```
internal/
├── app/product/
│   ├── domain/              # Pure business logic (no external deps!)
│   │   ├── product.go       # Product aggregate
│   │   ├── discount.go      # Discount value object
│   │   ├── money.go         # Money value object
│   │   ├── domain_events.go # Event intents
│   │   └── services/        # Domain services (e.g., PricingCalculator)
│   ├── usecases/            # Command handlers (write operations)
│   │   ├── create_product/interactor.go
│   │   ├── update_product/interactor.go
│   │   └── apply_discount/interactor.go
│   ├── queries/             # Query handlers (read operations)
│   │   ├── get_product/query.go
│   │   └── list_products/query.go
│   ├── contracts/           # Interfaces
│   │   ├── product_repo.go  # Repository interface
│   │   └── read_model.go    # Read model interface
│   └── repo/                # Spanner implementations
│       └── product_repo.go
├── models/                  # Database models
│   ├── m_product/
│   │   ├── data.go          # Product table model
│   │   └── fields.go        # Field name constants
│   └── m_outbox/
│       ├── data.go          # Outbox table model
│       └── fields.go
├── transport/grpc/product/  # gRPC handlers (thin layer)
│   ├── handler.go
│   ├── mappers.go           # Proto ↔ Domain mapping
│   └── errors.go            # Error code mapping
└── pkg/
    ├── committer/plan.go    # Typed CommitPlan wrapper
    └── clock/clock.go       # Time abstraction for testing
```

## Database Schema

**products table:**
- Stores price as `base_price_numerator`/`base_price_denominator` (INT64 pair)
- Discount fields: `discount_percent`, `discount_start_date`, `discount_end_date`
- Status field: `active`, `inactive`, `archived`
- Soft delete via `archived_at` timestamp

**outbox_events table:**
- `event_id`, `event_type`, `aggregate_id`
- `payload` as JSON
- `status` field for processing state
- Index on `(status, created_at)` for efficient polling

## gRPC Handler Pattern

Handlers are thin coordinators:

```go
func (h *ProductHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductReply, error) {
    // 1. Validate proto request
    if err := validateCreateRequest(req); err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }

    // 2. Map proto → application request
    appReq := mapToCreateProductRequest(req)

    // 3. Call usecase (usecase applies plan)
    productID, err := h.commands.CreateProduct.Execute(ctx, appReq)
    if err != nil {
        return nil, mapDomainErrorToGRPC(err)
    }

    // 4. Return response
    return &pb.CreateProductReply{ProductId: productID}, nil
}
```

**Handler responsibilities:**
- Proto validation
- Type mapping (proto ↔ application layer)
- Error code translation (domain errors → gRPC status codes)
- Usecase orchestration (usecases own transactions)

## Testing Strategy

**E2E Tests** (`tests/e2e/`):
- Use real Spanner connection (emulator)
- Test usecases directly (no gRPC layer needed)
- Verify business rules and error conditions
- Check side effects (outbox events created)
- Required scenarios: create, update, discount application, activation/deactivation, concurrent updates

**Unit Tests**:
- Domain logic in isolation
- Money calculations
- Discount validation
- State machine transitions

## Important Notes

- **Never bypass domain**: All writes must go through aggregate methods
- **Atomic transactions**: Use CommitPlan for all write operations that touch multiple tables
- **Context placement**: Pass `context.Context` to usecase.Execute(), never to domain methods
- **Error handling**: Domain errors are sentinel values; map them to gRPC status codes in handlers
- **Optimistic locking**: Use version fields or timestamps if implementing concurrent update protection
- **No over-engineering**: Don't add auth, background processors, actual Pub/Sub, or monitoring beyond basic logging
