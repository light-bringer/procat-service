# Database Migration Tool

Go-based migration tool for Cloud Spanner that works with both the local emulator and production instances.

## Usage

### With Makefile
```bash
# Migrate dev database (emulator on localhost:9010)
make migrate

# Migrate test database (emulator on localhost:19010)
make migrate-test
```

### Direct Usage
```bash
# With emulator
SPANNER_EMULATOR_HOST=localhost:9010 \
  go run cmd/migrate/main.go \
  -instance=dev-instance \
  -database=product-catalog-db

# With production (requires authentication)
go run cmd/migrate/main.go \
  -project=your-gcp-project \
  -instance=prod-instance \
  -database=product-catalog-db
```

## Features

- ✅ Creates Spanner instance automatically (emulator only)
- ✅ Creates database if it doesn't exist
- ✅ Applies all SQL migrations from `migrations/` directory
- ✅ Idempotent: Safe to run multiple times
- ✅ Works with Spanner emulator (no gcloud required)
- ✅ Production-ready for real GCP Spanner

## Migration Files

Place SQL migration files in the `migrations/` directory:
- Files are applied in lexicographical order
- Use naming like `001_initial_schema.sql`, `002_add_index.sql`
- Each file can contain multiple DDL statements separated by semicolons

## Why Not Shell Scripts?

The previous shell-based migration used `gcloud` CLI, which doesn't work with the Spanner emulator. This Go tool uses the Spanner client library directly, making it compatible with both emulator and production.
