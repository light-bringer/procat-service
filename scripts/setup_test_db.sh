#!/bin/bash
# Setup test database for integration and e2e tests
# This script is idempotent - safe to run multiple times

set -e

export SPANNER_EMULATOR_HOST=${SPANNER_EMULATOR_HOST:-localhost:19010}
export TEST_INSTANCE_ID=${TEST_INSTANCE_ID:-test-instance}
export TEST_DATABASE_ID=${TEST_DATABASE_ID:-product-catalog-test}
export SPANNER_PROJECT_ID=${SPANNER_PROJECT_ID:-test-project}

echo "Setting up test database..."
echo "Emulator: $SPANNER_EMULATOR_HOST"
echo "Instance: $TEST_INSTANCE_ID"
echo "Database: $TEST_DATABASE_ID"

# Wait for emulator to be ready
echo "Waiting for Spanner emulator..."
max_attempts=30
attempt=0
until curl -s http://localhost:19020/v1/projects/${SPANNER_PROJECT_ID}/instances > /dev/null 2>&1 || [ $attempt -eq $max_attempts ]; do
    attempt=$((attempt + 1))
    echo "Attempt $attempt/$max_attempts..."
    sleep 1
done

if [ $attempt -eq $max_attempts ]; then
    echo "ERROR: Spanner emulator did not become ready"
    exit 1
fi

echo "Emulator is ready!"

# Create instance
echo "Creating test instance..."
gcloud spanner instances create $TEST_INSTANCE_ID \
    --config=emulator-config \
    --description="Test instance" \
    --nodes=1 \
    --project=$SPANNER_PROJECT_ID 2>/dev/null || echo "Test instance already exists"

# Create database
echo "Creating test database..."
gcloud spanner databases create $TEST_DATABASE_ID \
    --instance=$TEST_INSTANCE_ID \
    --project=$SPANNER_PROJECT_ID 2>/dev/null || echo "Test database already exists"

# Migrations are handled by the application migration tool
echo "Database created successfully. Migrations will be applied by the application."

echo "Test database setup completed!"
