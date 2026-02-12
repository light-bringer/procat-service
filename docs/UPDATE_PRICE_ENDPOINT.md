# UpdatePrice Endpoint

The UpdatePrice endpoint is fully implemented and allows updating a product's base price with price history tracking and optimistic locking.

## gRPC Endpoint

```protobuf
rpc UpdatePrice(UpdatePriceRequest) returns (UpdatePriceReply);
```

## Request

```protobuf
message UpdatePriceRequest {
  string product_id = 1;
  optional int64 version = 2;      // For optimistic locking
  Money new_price = 3;              // Numerator/denominator representation
  string changed_by = 4;            // User/system identifier
  string changed_reason = 5;        // Optional explanation
}
```

## Response

```protobuf
message UpdatePriceReply {
  // Empty - success indicated by no error
}
```

## Implementation Details

### Use Case
**Location:** `internal/app/product/usecases/update_price/interactor.go`

**Features:**
- ✅ Optimistic locking via version field (always enforced)
- ✅ Price validation (must be > 0)
- ✅ Price history tracking (old price → new price)
- ✅ Domain event emission (`BasePriceChangedEvent`)
- ✅ Transactional outbox pattern
- ✅ Golden Mutation Pattern

**Flow:**
1. Validate request (product_id, new_price > 0, changed_by required)
2. Load product aggregate
3. Call `product.SetBasePrice(newPrice)` - domain validation
4. Create commit plan with:
   - Product update mutation
   - Price history record
   - Outbox events
5. Execute atomic transaction with version check
6. Clear domain events

### Price History

All price changes are tracked in the `price_history` table with:
- `history_id` - Unique record ID
- `product_id` - Product being updated
- `old_price_numerator` / `old_price_denominator` - Previous price
- `new_price_numerator` / `new_price_denominator` - New price
- `changed_by` - Who made the change
- `changed_reason` - Why the change was made
- `changed_at` - When the change occurred

### Optimistic Locking

UpdatePrice **always enforces** optimistic locking via the version field:

```go
err = i.committer.ApplyWithVersionCheck(ctx, req.ProductID, req.Version, plan)
```

This prevents concurrent price updates from causing race conditions.

### Domain Events

Emits `BasePriceChangedEvent`:
```go
type BasePriceChangedEvent struct {
    ProductID string
    OldPrice  *Money
    NewPrice  *Money
    ChangedAt time.Time
}
```

### Error Handling

Returns gRPC errors for:
- `InvalidArgument` - Missing/invalid fields
- `NotFound` - Product doesn't exist
- `FailedPrecondition` - Product is archived
- `Aborted` - Version mismatch (concurrent update)
- `OutOfRange` - Price exceeds int64 storage capacity

## Example Usage

### Go Client

```go
import pb "github.com/light-bringer/procat-service/proto/product/v1"

// Get current product to obtain version
product, _ := client.GetProduct(ctx, &pb.GetProductRequest{
    ProductId: "prod-123",
})

// Update price with optimistic locking
_, err := client.UpdatePrice(ctx, &pb.UpdatePriceRequest{
    ProductId: "prod-123",
    Version:   &product.Version,
    NewPrice: &pb.Money{
        Numerator:   19999,
        Denominator: 100,
    },
    ChangedBy:     "admin@example.com",
    ChangedReason: "Seasonal promotion",
})
```

### grpcurl

```bash
grpcurl -plaintext \
  -d '{
    "product_id": "prod-123",
    "version": 5,
    "new_price": {
      "numerator": 19999,
      "denominator": 100
    },
    "changed_by": "admin@example.com",
    "changed_reason": "Seasonal promotion"
  }' \
  localhost:50051 \
  product.v1.ProductService/UpdatePrice
```

## Testing

### E2E Tests
**Location:** `tests/e2e/update_price_test.go`

Tests cover:
- ✅ Successful price update
- ✅ Optimistic locking (concurrent updates)
- ✅ Invalid price validation (zero, negative)
- ✅ Price history creation
- ✅ Event emission

Run tests:
```bash
# Requires Spanner emulator running
docker-compose up -d
go test ./tests/e2e -run TestUpdatePrice -v
```

## Architecture

```
┌─────────────────┐
│  gRPC Handler   │
│  UpdatePrice()  │
└────────┬────────┘
         │
         ├─ Validate proto request
         ├─ Map proto → domain Money
         │
         ▼
┌─────────────────────────┐
│  update_price.Interactor│
│  Execute()              │
└────────┬────────────────┘
         │
         ├─ Load Product aggregate
         ├─ product.SetBasePrice()
         ├─ Create CommitPlan
         │    ├─ UpdateMut (product)
         │    ├─ InsertMut (price_history)
         │    └─ InsertMut (outbox_events)
         ├─ ApplyWithVersionCheck()
         │
         ▼
┌──────────────────┐
│  Domain Layer    │
│  Product.        │
│  SetBasePrice()  │
└──────────────────┘
         │
         ├─ Validate price > 0
         ├─ Check not archived
         ├─ Update basePrice
         ├─ Mark field dirty
         └─ Emit BasePriceChangedEvent
```

## Key Design Decisions

1. **Separate from UpdateProduct:** Price changes are tracked separately from general product updates for audit/compliance reasons

2. **Always Use Version Check:** Price updates always enforce optimistic locking to prevent race conditions in high-concurrency scenarios

3. **Price History Table:** All price changes are recorded for audit trails and analytics

4. **Money as Numerator/Denominator:** Precise rational number representation prevents floating-point precision loss

5. **changed_by & changed_reason:** Required fields for compliance and auditing (who changed what and why)

## Related Endpoints

- `CreateProduct` - Initial product creation with base price
- `UpdateProduct` - Update name/description/category (not price)
- `ApplyDiscount` - Apply time-bound percentage discount
- `GetProduct` - Retrieve product with effective price

## Status

✅ **Fully Implemented and Tested**
- Proto definition complete
- Use case implementation complete
- gRPC handler complete
- DI container wired
- E2E tests passing
- Price history tracking working
- Optimistic locking enforced
