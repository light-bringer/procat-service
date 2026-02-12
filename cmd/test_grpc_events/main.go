package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/light-bringer/procat-service/proto/product/v1"
)

func main() {
	// Connect to gRPC server
	conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewProductServiceClient(conn)
	ctx := context.Background()

	// List events
	limit := int32(10)
	resp, err := client.ListEvents(ctx, &pb.ListEventsRequest{
		Limit: limit,
	})
	if err != nil {
		log.Fatalf("Failed to list events: %v", err)
	}

	fmt.Printf("Found %d events (total: %d):\n\n", len(resp.Events), resp.TotalCount)
	for i, event := range resp.Events {
		fmt.Printf("%d. %s\n", i+1, event.EventType)
		fmt.Printf("   Event ID: %s\n", event.EventId)
		fmt.Printf("   Aggregate ID: %s\n", event.AggregateId)
		fmt.Printf("   Status: %s\n", event.Status)
		fmt.Printf("   Created: %s\n", event.CreatedAt.AsTime().Format("2006-01-02 15:04:05"))
		fmt.Printf("   Payload: %s\n\n", event.Payload)
	}
}
