package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/spanner"
)

// Configuration for outbox cleanup job
type Config struct {
	SpannerDB              string
	CompletedRetentionDays int
	FailedRetentionDays    int
	DryRun                 bool
}

func main() {
	// Parse command-line flags
	config := Config{}
	flag.StringVar(&config.SpannerDB, "database", "", "Spanner database (required, format: projects/PROJECT/instances/INSTANCE/databases/DATABASE)")
	flag.IntVar(&config.CompletedRetentionDays, "completed-retention", 30, "Retention days for completed events")
	flag.IntVar(&config.FailedRetentionDays, "failed-retention", 90, "Retention days for failed events")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	flag.Parse()

	if config.SpannerDB == "" {
		log.Fatal("Error: -database flag is required")
	}

	ctx := context.Background()

	// Run cleanup
	if err := cleanupOutbox(ctx, config); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}

	log.Println("Cleanup completed successfully")
}

func cleanupOutbox(ctx context.Context, config Config) error {
	// Create Spanner client
	client, err := spanner.NewClient(ctx, config.SpannerDB)
	if err != nil {
		return fmt.Errorf("failed to create Spanner client: %w", err)
	}
	defer client.Close()

	// Calculate cutoff timestamps
	now := time.Now().UTC()
	completedCutoff := now.AddDate(0, 0, -config.CompletedRetentionDays)
	failedCutoff := now.AddDate(0, 0, -config.FailedRetentionDays)

	log.Printf("Starting outbox cleanup...")
	log.Printf("  Completed events cutoff: %s (retention: %d days)", completedCutoff.Format(time.RFC3339), config.CompletedRetentionDays)
	log.Printf("  Failed events cutoff: %s (retention: %d days)", failedCutoff.Format(time.RFC3339), config.FailedRetentionDays)
	log.Printf("  Dry run: %v", config.DryRun)

	if config.DryRun {
		return dryRunCleanup(ctx, client, completedCutoff, failedCutoff)
	}

	return performCleanup(ctx, client, completedCutoff, failedCutoff)
}

func dryRunCleanup(ctx context.Context, client *spanner.Client, completedCutoff, failedCutoff time.Time) error {
	// Count events that would be deleted
	countQuery := `
		SELECT status, COUNT(*) as count
		FROM outbox_events
		WHERE (status = 'completed' AND processed_at < @completedCutoff)
		   OR (status = 'failed' AND processed_at < @failedCutoff)
		GROUP BY status
	`

	stmt := spanner.Statement{
		SQL: countQuery,
		Params: map[string]interface{}{
			"completedCutoff": completedCutoff,
			"failedCutoff":    failedCutoff,
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	totalCount := int64(0)
	for {
		row, err := iter.Next()
		if err != nil {
			if err.Error() == "iterator done" {
				break
			}
			return fmt.Errorf("failed to query events: %w", err)
		}

		var status string
		var count int64
		if err := row.Columns(&status, &count); err != nil {
			return fmt.Errorf("failed to parse row: %w", err)
		}

		log.Printf("  Would delete %d %s events", count, status)
		totalCount += count
	}

	log.Printf("DRY RUN: Would delete %d total events", totalCount)
	log.Println("Run without --dry-run to actually delete events")

	return nil
}

func performCleanup(ctx context.Context, client *spanner.Client, completedCutoff, failedCutoff time.Time) error {
	// Delete old events in a transaction
	deleteQuery := `
		DELETE FROM outbox_events
		WHERE (status = 'completed' AND processed_at < @completedCutoff)
		   OR (status = 'failed' AND processed_at < @failedCutoff)
	`

	stmt := spanner.Statement{
		SQL: deleteQuery,
		Params: map[string]interface{}{
			"completedCutoff": completedCutoff,
			"failedCutoff":    failedCutoff,
		},
	}

	// Execute as a mutation
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// First count what we're about to delete
		countQuery := `
			SELECT COUNT(*) as count
			FROM outbox_events
			WHERE (status = 'completed' AND processed_at < @completedCutoff)
			   OR (status = 'failed' AND processed_at < @failedCutoff)
		`

		countStmt := spanner.Statement{
			SQL: countQuery,
			Params: map[string]interface{}{
				"completedCutoff": completedCutoff,
				"failedCutoff":    failedCutoff,
			},
		}

		iter := txn.Query(ctx, countStmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			return fmt.Errorf("failed to count events: %w", err)
		}

		var count int64
		if err := row.Columns(&count); err != nil {
			return fmt.Errorf("failed to parse count: %w", err)
		}

		if count == 0 {
			log.Println("No old events to delete")
			return nil
		}

		log.Printf("Deleting %d old events...", count)

		// Perform the delete
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to delete events: %w", err)
		}

		log.Printf("Successfully deleted %d events", rowCount)

		return nil
	})
	if err != nil {
		return fmt.Errorf("cleanup transaction failed: %w", err)
	}

	return nil
}
