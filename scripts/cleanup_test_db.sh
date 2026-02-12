#!/bin/bash
# Cleanup test database
# Drops and recreates the test database for a clean state

set -e

export SPANNER_EMULATOR_HOST=${SPANNER_EMULATOR_HOST:-localhost:19010}
export TEST_INSTANCE_ID=${TEST_INSTANCE_ID:-test-instance}
export TEST_DATABASE_ID=${TEST_DATABASE_ID:-product-catalog-test}
export SPANNER_PROJECT_ID=${SPANNER_PROJECT_ID:-test-project}

echo "Cleaning up test database..."
echo "Instance: $TEST_INSTANCE_ID"
echo "Database: $TEST_DATABASE_ID"

# Drop database if it exists
echo "Dropping test database (if exists)..."
gcloud spanner databases delete $TEST_DATABASE_ID \
    --instance=$TEST_INSTANCE_ID \
    --project=$SPANNER_PROJECT_ID \
    --quiet 2>/dev/null || echo "Database doesn't exist or already dropped"

echo "Test database cleanup completed!"
