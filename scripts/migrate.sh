#!/bin/bash
# Migration script for Cloud Spanner
# Usage: ./scripts/migrate.sh <instance-id> <database-id>

set -e

INSTANCE_ID=${1:-dev-instance}
DATABASE_ID=${2:-product-catalog-db}
PROJECT_ID=${SPANNER_PROJECT_ID:-test-project}

echo "Running migrations..."
echo "Project: $PROJECT_ID"
echo "Instance: $INSTANCE_ID"
echo "Database: $DATABASE_ID"

# Check if running against emulator
if [ -n "$SPANNER_EMULATOR_HOST" ]; then
    echo "Using Spanner emulator at $SPANNER_EMULATOR_HOST"
fi

# Create instance if it doesn't exist (emulator only)
if [ -n "$SPANNER_EMULATOR_HOST" ]; then
    echo "Creating instance (if not exists)..."
    gcloud spanner instances create $INSTANCE_ID \
        --config=emulator-config \
        --description="Dev instance" \
        --nodes=1 \
        --project=$PROJECT_ID 2>/dev/null || echo "Instance already exists"
fi

# Create database if it doesn't exist
echo "Creating database (if not exists)..."
gcloud spanner databases create $DATABASE_ID \
    --instance=$INSTANCE_ID \
    --project=$PROJECT_ID 2>/dev/null || echo "Database already exists"

# Apply migrations
echo "Applying migrations..."
for migration in migrations/*.sql; do
    echo "Applying $migration..."
    gcloud spanner databases ddl update $DATABASE_ID \
        --instance=$INSTANCE_ID \
        --project=$PROJECT_ID \
        --ddl-file=$migration
done

echo "Migrations completed successfully!"
