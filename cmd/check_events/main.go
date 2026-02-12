package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()
	spannerDB := "projects/test-project/instances/dev-instance/databases/product-catalog-db"

	client, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	stmt := spanner.Statement{
		SQL: "SELECT event_id, event_type, aggregate_id, status, created_at FROM outbox_events ORDER BY created_at DESC LIMIT 10",
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	fmt.Println("Events in outbox_events table:")
	count := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}

		var eventID, eventType, aggregateID, status string
		var createdAt spanner.NullTime
		if err := row.Columns(&eventID, &eventType, &aggregateID, &status, &createdAt); err != nil {
			log.Fatalf("Failed to scan: %v", err)
		}

		fmt.Printf("%d. %s - %s (aggregate: %s, status: %s)\n", count+1, eventType, eventID, aggregateID, status)
		count++
	}

	if count == 0 {
		fmt.Println("No events found!")
	} else {
		fmt.Printf("\nTotal: %d events\n", count)
	}
}
