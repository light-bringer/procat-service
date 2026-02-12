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

	// Create a product
	createResp, err := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Test Product",
		Description: "A test product for events demo",
		Category:    "electronics",
		BasePrice: &pb.Money{
			Numerator:   10000,
			Denominator: 100,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create product: %v", err)
	}
	productID := createResp.ProductId
	fmt.Printf("Created product: %s\n", productID)

	// Update the product
	_, err = client.UpdateProduct(ctx, &pb.UpdateProductRequest{
		ProductId: productID,
		Name:      strPtr("Updated Product Name"),
	})
	if err != nil {
		log.Fatalf("Failed to update product: %v", err)
	}
	fmt.Println("Updated product")

	// Activate the product
	_, err = client.ActivateProduct(ctx, &pb.ActivateProductRequest{
		ProductId: productID,
	})
	if err != nil {
		log.Fatalf("Failed to activate product: %v", err)
	}
	fmt.Println("Activated product")

	fmt.Println("\n✅ Test data created successfully!")
	fmt.Println("Now test the endpoints:")
	fmt.Println("  gRPC: grpcurl -plaintext localhost:9090 product.v1.ProductService/ListEvents")
	fmt.Println("  HTTP: curl 'http://localhost:8080/api/v1/events'")
}

func strPtr(s string) *string {
	return &s
}
