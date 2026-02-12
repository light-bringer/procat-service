package testutil

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/require"
)

// SetupSpannerTest creates a test Spanner client and returns a cleanup function.
func SetupSpannerTest(t *testing.T) (*spanner.Client, func()) {
	t.Helper()

	ctx := context.Background()
	spannerDB := GetTestSpannerDB()

	client, err := spanner.NewClient(ctx, spannerDB)
	require.NoError(t, err, "failed to create Spanner client")

	// Clean database before test
	CleanDatabase(t, client)

	cleanup := func() {
		CleanDatabase(t, client)
		client.Close()
	}

	return client, cleanup
}

// GetTestSpannerDB returns the test Spanner database string.
func GetTestSpannerDB() string {
	// Use environment variable or default
	db := "projects/test-project/instances/test-instance/databases/product-catalog-test"
	return db
}

// CleanDatabase truncates all tables for test isolation.
func CleanDatabase(t *testing.T, client *spanner.Client) {
	t.Helper()

	ctx := context.Background()

	// Delete all data from tables (order matters due to potential foreign keys)
	mutations := []*spanner.Mutation{
		spanner.Delete("outbox_events", spanner.AllKeys()),
		spanner.Delete("products", spanner.AllKeys()),
	}

	_, err := client.Apply(ctx, mutations)
	require.NoError(t, err, "failed to clean database")
}

// ExecuteDDL executes DDL statements (for schema changes in tests).
func ExecuteDDL(t *testing.T, ctx context.Context, adminClient *spanner.DatabaseAdminClient, database string, statements []string) {
	t.Helper()

	op, err := adminClient.UpdateDatabaseDdl(ctx, &spanner.UpdateDatabaseDdlRequest{
		Database:   database,
		Statements: statements,
	})
	require.NoError(t, err, "failed to start DDL operation")

	err = op.Wait(ctx)
	require.NoError(t, err, "DDL operation failed")
}

// WaitForEmulator waits for the Spanner emulator to be ready.
func WaitForEmulator(t *testing.T, spannerDB string) {
	t.Helper()

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		t.Fatalf("Spanner emulator not ready: %v", err)
	}
	defer client.Close()

	// Try a simple query
	stmt := spanner.Statement{SQL: "SELECT 1"}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	_, err = iter.Next()
	if err != nil {
		t.Fatalf("Spanner emulator not responding: %v", err)
	}
}

// AssertRowCount asserts the number of rows in a table.
func AssertRowCount(t *testing.T, client *spanner.Client, table string, expectedCount int) {
	t.Helper()

	ctx := context.Background()
	stmt := spanner.Statement{
		SQL: fmt.Sprintf("SELECT COUNT(*) FROM %s", table),
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	require.NoError(t, err, "failed to query row count")

	var count int64
	err = row.Columns(&count)
	require.NoError(t, err, "failed to parse count")

	require.Equal(t, int64(expectedCount), count, "unexpected row count in table %s", table)
}
