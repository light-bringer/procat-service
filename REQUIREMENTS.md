# Requirements Document

## Overview

Product Catalog Service is a microservice for managing product information, pricing, and lifecycle with support for dynamic discounts and price history tracking.

## Table of Contents

- [Functional Requirements](#functional-requirements)
- [Non-Functional Requirements](#non-functional-requirements)
- [Technical Requirements](#technical-requirements)
- [Business Rules](#business-rules)
- [Data Requirements](#data-requirements)
- [API Requirements](#api-requirements)
- [Out of Scope](#out-of-scope)

## Functional Requirements

### FR-1: Product Management

#### FR-1.1: Create Product
- **Description:** System shall allow creation of new products
- **Priority:** HIGH
- **Requirements:**
  - Product must have name, description, category
  - Product must have base price (positive value)
  - New products created as "inactive" by default
  - System generates unique product ID (UUID)
  - System records creation timestamp

#### FR-1.2: Update Product Details
- **Description:** System shall allow updating product information
- **Priority:** HIGH
- **Requirements:**
  - Can update name, description, category
  - Cannot update archived products
  - System records update timestamp
  - System tracks which fields changed (change tracking)

#### FR-1.3: Activate Product
- **Description:** System shall allow products to be made available for sale
- **Priority:** HIGH
- **Requirements:**
  - Only inactive products can be activated
  - Cannot activate archived products
  - System publishes "product.activated" event

#### FR-1.4: Deactivate Product
- **Description:** System shall allow products to be made unavailable for sale
- **Priority:** HIGH
- **Requirements:**
  - Only active products can be deactivated
  - Cannot deactivate archived products
  - System publishes "product.deactivated" event

#### FR-1.5: Archive Product
- **Description:** System shall support soft-deletion of products
- **Priority:** MEDIUM
- **Requirements:**
  - Archive is irreversible (cannot unarchive)
  - Cannot modify archived products
  - Archived products remain queryable (for audit)
  - System removes any active discount when archiving
  - System records archived_at timestamp
  - System publishes "product.archived" event

### FR-2: Pricing

#### FR-2.1: Set Base Price
- **Description:** System shall store product base price
- **Priority:** HIGH
- **Requirements:**
  - Price stored as numerator/denominator fraction
  - Price must be positive (> 0)
  - Price precision up to 2 decimal places minimum
  - System supports fractional cents if needed

#### FR-2.2: Update Price
- **Description:** System shall allow changing product price
- **Priority:** HIGH
- **Requirements:**
  - New price must be positive
  - Cannot update price of archived products
  - System records price change in price_history table
  - System captures old price, new price, timestamp
  - System publishes "product.price.updated" event

#### FR-2.3: Calculate Effective Price
- **Description:** System shall calculate current selling price
- **Priority:** HIGH
- **Requirements:**
  - If no discount: effective_price = base_price
  - If discount active: effective_price = base_price × (1 - discount%)
  - Calculation performed at query time based on current date

### FR-3: Discounts

#### FR-3.1: Apply Discount
- **Description:** System shall support time-bound percentage discounts
- **Priority:** HIGH
- **Requirements:**
  - Discount percentage between 0-100
  - Discount has start_date and end_date
  - Dates must be in UTC timezone
  - End date must be after start date
  - Can only apply discount to active products
  - Only one discount per product at a time
  - System publishes "product.discount.applied" event

#### FR-3.2: Remove Discount
- **Description:** System shall allow removing active discounts
- **Priority:** MEDIUM
- **Requirements:**
  - Can only remove if discount exists
  - Effective price reverts to base price
  - System publishes "product.discount.removed" event

#### FR-3.3: Time-Based Discount Activation
- **Description:** System shall activate/deactivate discounts based on time
- **Priority:** HIGH
- **Requirements:**
  - Discount only active during [start_date, end_date] period
  - Before start_date: discount inactive
  - After end_date: discount inactive
  - Calculation performed at query time

### FR-4: Product Queries

#### FR-4.1: Get Product by ID
- **Description:** System shall retrieve product details
- **Priority:** HIGH
- **Requirements:**
  - Returns all product fields
  - Returns calculated effective_price
  - Returns discount_active flag (true if valid now)
  - Returns 404 if product not found

#### FR-4.2: List Products
- **Description:** System shall support product listing with filtering
- **Priority:** HIGH
- **Requirements:**
  - Filter by category (optional)
  - Filter by status (optional)
  - Paginated results (page_size, page_token)
  - Default sort: created_at DESC
  - Returns total_count
  - Max page_size: 100

### FR-5: Event Publishing

#### FR-5.1: Domain Events
- **Description:** System shall publish events for all state changes
- **Priority:** HIGH
- **Requirements:**
  - Events stored in outbox_events table
  - Events and state changes in same transaction (atomic)
  - Event types: created, updated, activated, deactivated, archived, discount.applied, discount.removed, price.updated
  - Events include aggregate_id, timestamp, payload

#### FR-5.2: Transactional Outbox
- **Description:** System shall ensure reliable event delivery
- **Priority:** HIGH
- **Requirements:**
  - Events stored in database (not published directly)
  - Events marked with status (pending, processing, completed, failed)
  - Background processor publishes events (out of scope)
  - Supports retry mechanism

### FR-6: Price History

#### FR-6.1: Track Price Changes
- **Description:** System shall maintain audit trail of price changes
- **Priority:** MEDIUM
- **Requirements:**
  - Records old_price and new_price
  - Records changed_at timestamp
  - Records changed_by identifier
  - Price history immutable (no updates/deletes)

## Non-Functional Requirements

### NFR-1: Performance

#### NFR-1.1: Response Time
- **Requirement:** 95th percentile response time < 100ms
- **Priority:** HIGH
- **Measurement:** API response time for GetProduct

#### NFR-1.2: Throughput
- **Requirement:** Support 1000 requests/second
- **Priority:** MEDIUM
- **Measurement:** Concurrent requests without errors

#### NFR-1.3: Database Performance
- **Requirement:** Query optimization with proper indexes
- **Priority:** HIGH
- **Implementation:**
  - Index on (category, status, created_at)
  - Index on (status, updated_at)

### NFR-2: Scalability

#### NFR-2.1: Horizontal Scaling
- **Requirement:** Service must be stateless
- **Priority:** HIGH
- **Implementation:** No in-memory state, all state in Spanner

#### NFR-2.2: Database Scaling
- **Requirement:** Support millions of products
- **Priority:** MEDIUM
- **Implementation:** Google Cloud Spanner (globally distributed)

### NFR-3: Reliability

#### NFR-3.1: Availability
- **Requirement:** 99.9% uptime (43 minutes downtime/month)
- **Priority:** HIGH
- **Measurement:** Service health checks

#### NFR-3.2: Data Durability
- **Requirement:** Zero data loss for committed transactions
- **Priority:** CRITICAL
- **Implementation:** Spanner ACID transactions

#### NFR-3.3: Fault Tolerance
- **Requirement:** Graceful degradation on failures
- **Priority:** HIGH
- **Implementation:** Circuit breakers, retry logic

### NFR-4: Security

#### NFR-4.1: Input Validation
- **Requirement:** Validate all inputs at API boundary
- **Priority:** HIGH
- **Implementation:** gRPC request validation

#### NFR-4.2: SQL Injection Prevention
- **Requirement:** Use parameterized queries
- **Priority:** CRITICAL
- **Implementation:** Spanner client library

#### NFR-4.3: Authentication/Authorization
- **Requirement:** Assume API gateway handles auth
- **Priority:** MEDIUM
- **Note:** Not implemented in service itself

### NFR-5: Maintainability

#### NFR-5.1: Code Quality
- **Requirement:** Pass linter checks
- **Priority:** MEDIUM
- **Implementation:** golangci-lint v2 with 27+ linters

#### NFR-5.2: Test Coverage
- **Requirement:**
  - Domain layer: 100%
  - Use cases: >90%
  - Overall: >80%
- **Priority:** HIGH
- **Measurement:** go test -cover

#### NFR-5.3: Documentation
- **Requirement:** Comprehensive API and architecture docs
- **Priority:** MEDIUM
- **Implementation:** README, DESIGN, USAGE docs

### NFR-6: Observability

#### NFR-6.1: Logging
- **Requirement:** Structured logging for all operations
- **Priority:** MEDIUM
- **Implementation:** Standard Go log package

#### NFR-6.2: Error Tracking
- **Requirement:** All errors logged with context
- **Priority:** HIGH
- **Implementation:** Error wrapping with context

#### NFR-6.3: Metrics (Future)
- **Requirement:** Expose Prometheus metrics
- **Priority:** LOW
- **Note:** Out of scope for initial version

## Technical Requirements

### TR-1: Technology Stack

#### TR-1.1: Programming Language
- **Requirement:** Go 1.25.7+
- **Rationale:** Performance, concurrency, strong typing

#### TR-1.2: Database
- **Requirement:** Google Cloud Spanner
- **Rationale:** Global consistency, unlimited scale, ACID transactions

#### TR-1.3: API Protocol
- **Requirement:** gRPC with Protocol Buffers
- **Rationale:** Performance, type safety, efficient serialization

#### TR-1.4: Testing Framework
- **Requirement:** testify for assertions
- **Rationale:** Readable tests, good assertions

### TR-2: Architecture Patterns

#### TR-2.1: Clean Architecture
- **Requirement:** Strict layer separation
- **Implementation:**
  - Domain layer: zero external dependencies
  - Application layer: use cases and queries
  - Infrastructure layer: repositories and gRPC

#### TR-2.2: Domain-Driven Design
- **Requirement:** Rich domain model with business rules
- **Implementation:**
  - Product aggregate
  - Money and Discount value objects
  - Domain events

#### TR-2.3: CQRS
- **Requirement:** Separate read and write models
- **Implementation:**
  - Commands: use cases with domain validation
  - Queries: direct database access with DTOs

#### TR-2.4: Golden Mutation Pattern
- **Requirement:** Consistent transaction management
- **Implementation:**
  - Repositories return mutations
  - Use cases apply commit plans
  - All changes atomic

### TR-3: Data Precision

#### TR-3.1: Money Calculations
- **Requirement:** No floating-point errors
- **Implementation:** math/big.Rat for all money operations
- **Storage:** Numerator/denominator (INT64 pair)

#### TR-3.2: Discount Precision
- **Requirement:** Support fractional percentages (e.g., 15.5%)
- **Implementation:** float64 in domain, NUMERIC in database

### TR-4: Concurrency

#### TR-4.1: Optimistic Locking
- **Requirement:** Prevent lost updates
- **Implementation:** Version field on Product
- **Behavior:** Return ABORTED error on conflict

#### TR-4.2: Race Condition Prevention
- **Requirement:** No race conditions in concurrent updates
- **Implementation:** Version checking + domain validation

## Business Rules

### BR-1: Product Lifecycle

1. New products start as "inactive"
2. Products must be activated before applying discounts
3. Archived products cannot be modified
4. Archiving is irreversible
5. Product status transitions:
   - inactive → active (activate)
   - active → inactive (deactivate)
   - any → archived (archive)

### BR-2: Pricing

1. Price must be positive (> 0)
2. Price stored with precision (no rounding errors)
3. Price changes recorded in history
4. Effective price calculated at query time

### BR-3: Discounts

1. Only one discount per product at a time
2. Discount percentage: 0-100
3. Discount dates must be in UTC
4. Discount end date must be after start date
5. Can only apply discount to active products
6. Discount automatically inactive outside valid period
7. Archiving product removes active discount

### BR-4: Validation

1. Product name cannot be empty
2. Product name max 255 characters
3. Description max 1000 characters
4. Category max 100 characters
5. Discount dates must be valid timestamps
6. All timestamps stored in UTC

### BR-5: Events

1. All state changes generate events
2. Events stored in same transaction as state
3. Events immutable (no updates/deletes)
4. Events include full context (aggregate_id, timestamp, payload)

## Data Requirements

### DR-1: Data Model

#### Products Table
- **Primary Key:** product_id (UUID)
- **Required Fields:** name, category, base_price, status, version, created_at, updated_at
- **Optional Fields:** description, discount fields, archived_at
- **Indexes:**
  - (category, status, created_at DESC)
  - (status, updated_at DESC)

#### Outbox Events Table
- **Primary Key:** event_id (UUID)
- **Required Fields:** event_type, aggregate_id, payload, status, created_at, retry_count
- **Optional Fields:** processed_at, error_message
- **Index:** (status, created_at)

#### Price History Table
- **Primary Key:** history_id (UUID)
- **Required Fields:** product_id, old_price, new_price, changed_at
- **Optional Fields:** changed_by
- **Index:** (product_id, changed_at DESC)

### DR-2: Data Integrity

1. All foreign keys enforced
2. All timestamps in UTC
3. All prices stored as fractions
4. No NULL in required fields
5. Enum values validated

### DR-3: Data Retention

1. Products: Retain indefinitely (soft delete via archive)
2. Events: Retain for 90 days (configurable)
3. Price history: Retain indefinitely

## API Requirements

### API-1: gRPC Service

#### Service Definition
- **Service:** product.v1.ProductService
- **Protocol:** gRPC
- **Port:** 9090 (configurable)
- **Serialization:** Protocol Buffers

#### Endpoints
1. CreateProduct
2. UpdateProduct
3. UpdatePrice
4. ActivateProduct
5. DeactivateProduct
6. ApplyDiscount
7. RemoveDiscount
8. ArchiveProduct
9. GetProduct
10. ListProducts

### API-2: Error Handling

#### gRPC Status Codes
- INVALID_ARGUMENT: Validation errors
- NOT_FOUND: Resource not found
- FAILED_PRECONDITION: Business rule violation
- ABORTED: Concurrent modification
- INTERNAL: Server errors

#### Error Messages
- Must be descriptive
- Must not leak sensitive data
- Must include context for debugging

### API-3: Versioning

#### Proto Versioning
- **Current:** v1
- **Strategy:** Maintain backward compatibility
- **Breaking Changes:** Increment major version (v2)

## Out of Scope

The following are explicitly **NOT** included in this version:

### Authentication & Authorization
- User management
- JWT/OAuth validation
- Role-based access control
- API keys

**Rationale:** Assume API gateway handles authentication

### Background Processing
- Event publisher
- Outbox processor
- Scheduled jobs
- Batch operations

**Rationale:** Focus on core domain, defer processing to separate service

### Advanced Features
- Multiple simultaneous discounts
- Tiered pricing (wholesale, retail)
- Inventory management
- Product variants (size, color)
- Product images/media
- Product reviews/ratings

**Rationale:** Keep initial version focused on core functionality

### Monitoring & Observability
- Metrics (Prometheus)
- Distributed tracing (Jaeger)
- APM integration
- Health checks beyond basic readiness

**Rationale:** Infrastructure concern, handled at deployment level

### Caching
- Redis integration
- In-memory caching
- CDN integration

**Rationale:** Optimization for future, not needed initially

### Multi-tenancy
- Tenant isolation
- Per-tenant configuration
- Tenant-specific pricing

**Rationale:** Single-tenant use case initially

### Internationalization
- Multi-currency support
- Locale-specific formatting
- Translation management

**Rationale:** Single currency/locale initially

### REST API
- REST endpoints
- gRPC-gateway integration
- OpenAPI/Swagger

**Rationale:** gRPC-first, REST can be added later via gateway

## Acceptance Criteria

### For Each Feature

A feature is considered complete when:

1. ✅ Domain logic implemented and validated
2. ✅ Use case/query handler implemented
3. ✅ gRPC endpoint implemented and tested
4. ✅ Unit tests written (>90% coverage)
5. ✅ Integration tests written
6. ✅ E2E tests written
7. ✅ Documentation updated
8. ✅ All CI checks pass

### For Overall Project

Project is ready for production when:

1. ✅ All functional requirements implemented
2. ✅ All critical non-functional requirements met
3. ✅ All tests passing (unit, integration, E2E)
4. ✅ Test coverage >80% overall
5. ✅ Documentation complete (README, DESIGN, USAGE)
6. ✅ CI/CD pipeline working
7. ✅ Load testing completed (1000 req/s)
8. ✅ Security review completed
9. ✅ Database migrations tested
10. ✅ Deployment runbook created

## Appendix

### Glossary

- **Aggregate:** Domain object that enforces invariants
- **CQRS:** Command Query Responsibility Segregation
- **DDD:** Domain-Driven Design
- **DTO:** Data Transfer Object
- **gRPC:** Google Remote Procedure Call
- **Outbox Pattern:** Reliable event publishing via database
- **Optimistic Locking:** Concurrency control using version numbers
- **Value Object:** Immutable domain object without identity

### References

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design](https://www.domainlanguage.com/ddd/)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [Transactional Outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [Google Cloud Spanner](https://cloud.google.com/spanner/docs)
- [gRPC](https://grpc.io/)
