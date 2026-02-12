package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/light-bringer/procat-service/internal/services"
	pb "github.com/light-bringer/procat-service/proto/product/v1"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func run() error {
	ctx := context.Background()

	// 1. Load configuration from environment variables
	config := loadConfig()

	log.Printf("Starting Product Catalog Service...")
	log.Printf("Spanner Database: %s", config.SpannerDB)
	log.Printf("gRPC Port: %s", config.GRPCPort)

	// 2. Initialize service dependencies (DI container)
	serviceOpts, err := services.NewServiceOptions(ctx, config.SpannerDB)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}
	defer serviceOpts.Close()

	// 3. Create gRPC server
	grpcServer := grpc.NewServer()

	// 4. Register services
	pb.RegisterProductServiceServer(grpcServer, serviceOpts.ProductHandler)

	// 5. Enable reflection (for grpcurl and debugging)
	reflection.Register(grpcServer)

	// 6. Start listening
	lis, err := net.Listen("tcp", ":"+config.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// 7. Graceful shutdown handling
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	// 8. Start serving
	log.Printf("gRPC server listening on :%s", config.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Config holds application configuration.
type Config struct {
	SpannerDB string
	GRPCPort  string
}

// loadConfig loads configuration from environment variables with defaults.
func loadConfig() Config {
	spannerDB := os.Getenv("SPANNER_DATABASE")
	if spannerDB == "" {
		// Default for local development with emulator
		spannerDB = "projects/test-project/instances/dev-instance/databases/product-catalog-db"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}

	return Config{
		SpannerDB: spannerDB,
		GRPCPort:  grpcPort,
	}
}
