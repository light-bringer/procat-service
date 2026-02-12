package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/light-bringer/procat-service/internal/services"
	httphandler "github.com/light-bringer/procat-service/internal/transport/http"
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
	log.Printf("HTTP Port: %s", config.HTTPPort)

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

	// 6. Start gRPC server listening
	lis, err := net.Listen("tcp", ":"+config.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port: %w", err)
	}

	// 7. Start gRPC server in background
	go func() {
		log.Printf("gRPC server listening on :%s", config.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// 8. Create HTTP server with gRPC client (using direct handler reference)
	// Instead of creating a gRPC client, use the handler directly
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		// Create a gRPC client connection for each request (simpler approach)
		grpcConn, err := grpc.NewClient("localhost:"+config.GRPCPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			http.Error(w, "Failed to connect to gRPC: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer grpcConn.Close()

		grpcClient := pb.NewProductServiceClient(grpcConn)
		handler := httphandler.NewEventsHandler(grpcClient)
		handler.ServeHTTP(w, r)
	})

	httpServer := &http.Server{
		Addr:    ":" + config.HTTPPort,
		Handler: httpMux,
	}

	// 9. Start HTTP server in background
	go func() {
		log.Printf("HTTP server listening on :%s", config.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// 10. Graceful shutdown handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down gracefully...")

	// Shutdown HTTP server
	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop gRPC server
	grpcServer.GracefulStop()

	return nil
}

// Config holds application configuration.
type Config struct {
	SpannerDB string
	GRPCPort  string
	HTTPPort  string
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

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	return Config{
		SpannerDB: spannerDB,
		GRPCPort:  grpcPort,
		HTTPPort:  httpPort,
	}
}
