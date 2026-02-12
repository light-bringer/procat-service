# Usage Guide

Complete guide to using the Product Catalog Service API.

## Table of Contents

- [Quick Start](#quick-start)
- [API Overview](#api-overview)
- [Common Workflows](#common-workflows)
- [API Reference](#api-reference)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Quick Start

### Prerequisites

- gRPC service running at `localhost:9090`
- `grpcurl` installed for testing

```bash
# Install grpcurl (macOS)
brew install grpcurl

# Install grpcurl (Linux)
curl -sSL "https://github.com/fullstorydev/grpcurl/releases/download/v1.8.9/grpcurl_1.8.9_linux_x86_64.tar.gz" | tar -xz -C /usr/local/bin
```

### List Available Services

```bash
grpcurl -plaintext localhost:9090 list
```

### Describe a Service

```bash
grpcurl -plaintext localhost:9090 describe product.v1.ProductService
```

## API Overview

### gRPC Service: `product.v1.ProductService`

**Endpoints:**
- 8 Commands (Write Operations)
- 2 Queries (Read Operations)

**Port:** 9090 (default)

**Protocol:** gRPC with Protocol Buffers

## Common Workflows

### Workflow 1: Create and Activate Product

```bash
# Step 1: Create Product
grpcurl -plaintext -d '{
  "name": "iPhone 15 Pro",
  "description": "Latest flagship smartphone",
  "category": "electronics",
  "base_price": {"numerator": 99900, "denominator": 100}
}' localhost:9090 product.v1.ProductService/CreateProduct

# Response:
# {
#   "product_id": "550e8400-e29b-41d4-a716-446655440000"
# }

# Step 2: Get Product Details
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/GetProduct

# Response shows status="inactive"

# Step 3: Activate Product
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/ActivateProduct

# Product is now available for sale
```

### Workflow 2: Update Product Price

```bash
# Get current version
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/GetProduct

# Note the version field (e.g., version=1)

# Update price with optimistic locking
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 1,
  "new_price": {"numerator": 89900, "denominator": 100}
}' localhost:9090 product.v1.ProductService/UpdatePrice

# Price updated: $999.00 → $899.00
```

### Workflow 3: Apply Time-Bound Discount

```bash
# Get current version first
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/GetProduct

# Apply 20% discount for holiday season
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 2,
  "discount_percent": 20.0,
  "start_date": "2025-12-01T00:00:00Z",
  "end_date": "2025-12-31T23:59:59Z"
}' localhost:9090 product.v1.ProductService/ApplyDiscount

# Effective price during discount: $899 * 0.8 = $719.20
```

### Workflow 4: Search and Filter Products

```bash
# List all products in electronics category
grpcurl -plaintext -d '{
  "category": "electronics",
  "page_size": 10
}' localhost:9090 product.v1.ProductService/ListProducts

# List only active products
grpcurl -plaintext -d '{
  "status": "active",
  "page_size": 20
}' localhost:9090 product.v1.ProductService/ListProducts

# Paginate through results
grpcurl -plaintext -d '{
  "category": "electronics",
  "page_size": 10,
  "page_token": "<token-from-previous-response>"
}' localhost:9090 product.v1.ProductService/ListProducts
```

### Workflow 5: Archive Product

```bash
# Get current version
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/GetProduct

# Archive product (soft delete)
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 3
}' localhost:9090 product.v1.ProductService/ArchiveProduct

# Response includes archived_at timestamp
# Archived products cannot be modified
```

## API Reference

### CreateProduct

Create a new product (status=inactive by default).

**Request:**
```json
{
  "name": "string (required, max 255)",
  "description": "string (optional, max 1000)",
  "category": "string (required, max 100)",
  "base_price": {
    "numerator": "int64 (required)",
    "denominator": "int64 (required, usually 100)"
  }
}
```

**Response:**
```json
{
  "product_id": "string (UUID)"
}
```

**Validations:**
- Name cannot be empty
- Price must be positive (numerator > 0)
- Price denominator must be positive

**Example:**
```bash
grpcurl -plaintext -d '{
  "name": "MacBook Pro",
  "description": "16-inch laptop with M3 chip",
  "category": "computers",
  "base_price": {"numerator": 249900, "denominator": 100}
}' localhost:9090 product.v1.ProductService/CreateProduct
```

### UpdateProduct

Update product details (name, description, category).

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional, for optimistic locking)",
  "name": "string (optional)",
  "description": "string (optional)",
  "category": "string (optional)"
}
```

**Response:**
```json
{}
```

**Notes:**
- At least one field (name, description, or category) must be provided
- Cannot update archived products
- Version field enables optimistic locking

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "MacBook Pro 16-inch M3",
  "description": "Updated description"
}' localhost:9090 product.v1.ProductService/UpdateProduct
```

### UpdatePrice

Change product's base price.

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional)",
  "new_price": {
    "numerator": "int64 (required)",
    "denominator": "int64 (required)"
  }
}
```

**Response:**
```json
{}
```

**Validations:**
- New price must be positive
- Cannot update archived products
- Creates entry in price_history table

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 1,
  "new_price": {"numerator": 229900, "denominator": 100}
}' localhost:9090 product.v1.ProductService/UpdatePrice
```

### ActivateProduct

Make product available for sale.

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional)"
}
```

**Response:**
```json
{}
```

**Rules:**
- Product must be inactive
- Cannot activate archived products
- Cannot activate already active products

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/ActivateProduct
```

### DeactivateProduct

Make product unavailable for sale.

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional)"
}
```

**Response:**
```json
{}
```

**Rules:**
- Product must be active
- Cannot deactivate archived products
- Cannot deactivate already inactive products

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 2
}' localhost:9090 product.v1.ProductService/DeactivateProduct
```

### ApplyDiscount

Add a time-bound percentage discount.

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional)",
  "discount_percent": "double (required, 0-100)",
  "start_date": "timestamp (required, UTC)",
  "end_date": "timestamp (required, UTC)"
}
```

**Response:**
```json
{}
```

**Validations:**
- Product must be active
- Discount percentage: 0-100
- Start and end dates must be in UTC
- End date must be after start date
- Cannot apply if discount already exists

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 2,
  "discount_percent": 15.5,
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-01-31T23:59:59Z"
}' localhost:9090 product.v1.ProductService/ApplyDiscount
```

### RemoveDiscount

Remove active discount from product.

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional)"
}
```

**Response:**
```json
{}
```

**Rules:**
- Product must have an active discount
- Reverts effective_price to base_price

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 3
}' localhost:9090 product.v1.ProductService/RemoveDiscount
```

### ArchiveProduct

Soft-delete a product (irreversible).

**Request:**
```json
{
  "product_id": "string (required)",
  "version": "int64 (optional)"
}
```

**Response:**
```json
{
  "archived_at": "timestamp"
}
```

**Rules:**
- Cannot modify archived products
- Removes any active discount
- Sets archived_at timestamp
- Product still visible in queries (for audit)

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 4
}' localhost:9090 product.v1.ProductService/ArchiveProduct
```

### GetProduct

Retrieve product details by ID.

**Request:**
```json
{
  "product_id": "string (required)"
}
```

**Response:**
```json
{
  "product": {
    "product_id": "string",
    "name": "string",
    "description": "string",
    "category": "string",
    "base_price": "double",
    "effective_price": "double",
    "discount_percent": "double (nullable)",
    "discount_active": "bool",
    "status": "string (inactive|active|archived)",
    "version": "int64",
    "created_at": "timestamp",
    "updated_at": "timestamp",
    "archived_at": "timestamp (nullable)"
  }
}
```

**Notes:**
- `effective_price`: Calculated at query time based on current date
- `discount_active`: True if discount exists and valid now
- Returns error if product not found

**Example:**
```bash
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/GetProduct
```

### ListProducts

List products with filtering and pagination.

**Request:**
```json
{
  "category": "string (optional)",
  "status": "string (optional, inactive|active|archived)",
  "page_size": "int32 (required, max 100)",
  "page_token": "string (optional, for pagination)"
}
```

**Response:**
```json
{
  "products": [
    {
      "product_id": "string",
      "name": "string",
      // ... same fields as GetProduct
    }
  ],
  "next_page_token": "string (empty if last page)",
  "total_count": "int64"
}
```

**Notes:**
- Default page_size: 20
- Max page_size: 100
- Sorted by created_at DESC
- Use next_page_token for pagination

**Example:**
```bash
# First page
grpcurl -plaintext -d '{
  "category": "electronics",
  "status": "active",
  "page_size": 10
}' localhost:9090 product.v1.ProductService/ListProducts

# Next page
grpcurl -plaintext -d '{
  "category": "electronics",
  "status": "active",
  "page_size": 10,
  "page_token": "<token-from-previous-response>"
}' localhost:9090 product.v1.ProductService/ListProducts
```

## Error Handling

### gRPC Status Codes

| Code | Description | Common Causes |
|------|-------------|---------------|
| `INVALID_ARGUMENT` | Validation failed | Empty name, negative price, invalid dates |
| `NOT_FOUND` | Resource not found | Product ID doesn't exist |
| `FAILED_PRECONDITION` | Business rule violated | Cannot activate archived product |
| `ABORTED` | Concurrent modification | Version mismatch (optimistic locking) |
| `INTERNAL` | Server error | Database error, unexpected failure |

### Example Error Responses

**Invalid Argument:**
```json
{
  "error": "rpc error: code = InvalidArgument desc = product name cannot be empty"
}
```

**Not Found:**
```json
{
  "error": "rpc error: code = NotFound desc = product not found"
}
```

**Failed Precondition:**
```json
{
  "error": "rpc error: code = FailedPrecondition desc = product is already active"
}
```

**Aborted (Version Conflict):**
```json
{
  "error": "rpc error: code = Aborted desc = version mismatch: expected 1, got 2 (concurrent modification detected)"
}
```

### Handling Version Conflicts

When you get an `ABORTED` error due to version mismatch:

```bash
# 1. Re-fetch the product to get latest version
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 product.v1.ProductService/GetProduct

# 2. Check if your operation is still valid
#    (e.g., another user might have already applied a discount)

# 3. Retry with new version if appropriate
grpcurl -plaintext -d '{
  "product_id": "550e8400-e29b-41d4-a716-446655440000",
  "version": 3,  # Use latest version
  "discount_percent": 20.0,
  "start_date": "2025-01-01T00:00:00Z",
  "end_date": "2025-01-31T23:59:59Z"
}' localhost:9090 product.v1.ProductService/ApplyDiscount
```

## Best Practices

### 1. Always Use UTC for Timestamps

```bash
# Good: UTC timezone
"start_date": "2025-01-01T00:00:00Z"

# Bad: Local timezone (will be rejected)
"start_date": "2025-01-01T00:00:00-05:00"
```

### 2. Store Prices as Fractions

```bash
# Good: Numerator/denominator
"base_price": {"numerator": 249900, "denominator": 100}  # $2499.00

# The denominator is usually 100 for cents
# For more precision, use 1000: {"numerator": 2499000, "denominator": 1000}
```

### 3. Use Optimistic Locking for Concurrent Updates

```bash
# Always include version for update operations
{
  "product_id": "...",
  "version": 2,  # Include version
  "new_price": {"numerator": 199900, "denominator": 100}
}
```

### 4. Pagination for Large Result Sets

```bash
# Don't fetch all products at once
# Use pagination with reasonable page_size

page_size=50  # Start with modest page size
while [ -n "$next_token" ]; do
  grpcurl -plaintext -d "{
    \"page_size\": $page_size,
    \"page_token\": \"$next_token\"
  }" localhost:9090 product.v1.ProductService/ListProducts
done
```

### 5. Check Effective Price, Not Just Base Price

```json
// The effective_price includes discounts
{
  "base_price": 100.00,
  "effective_price": 80.00,  // 20% discount active
  "discount_active": true
}
```

### 6. Handle Archived Products

```bash
# Archived products are immutable
# Any modification attempt will return FailedPrecondition

# Check status before update
status=$(grpcurl -plaintext -d '{"product_id":"..."}' \
  localhost:9090 product.v1.ProductService/GetProduct | jq -r '.product.status')

if [ "$status" == "archived" ]; then
  echo "Cannot modify archived product"
  exit 1
fi
```

### 7. Validate Before Calling API

Client-side validation prevents unnecessary API calls:

```bash
# Validate discount percentage (0-100)
if (( $(echo "$discount < 0 || $discount > 100" | bc -l) )); then
  echo "Invalid discount percentage"
  exit 1
fi

# Validate dates are in future
start_timestamp=$(date -d "$start_date" +%s)
now=$(date +%s)
if (( start_timestamp < now )); then
  echo "Start date must be in the future"
  exit 1
fi
```

## Troubleshooting

### Connection Refused

```bash
# Check if service is running
grpcurl -plaintext localhost:9090 list

# If fails, start the service
docker compose up -d
go run cmd/server/main.go
```

### Method Not Found

```bash
# Verify service is registered
grpcurl -plaintext localhost:9090 list

# Should show: product.v1.ProductService
```

### Permission Denied

```bash
# API doesn't have auth by default
# If you get permission errors, check your API gateway configuration
```

### Invalid Timestamp Format

```bash
# Use RFC 3339 format with 'Z' suffix for UTC
"2025-01-01T00:00:00Z"  # Correct
"2025-01-01 00:00:00"    # Wrong
```

## Additional Resources

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)
- [grpcurl GitHub](https://github.com/fullstorydev/grpcurl)
- [Project README](README.md)
- [Design Document](DESIGN.md)
