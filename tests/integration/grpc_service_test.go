//go:build integration

package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/light-bringer/procat-service/internal/app/product/queries/get_product"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_events"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_products"
	"github.com/light-bringer/procat-service/internal/app/product/repo"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/activate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/apply_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/archive_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/create_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/deactivate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/remove_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_price"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_product"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/internal/pkg/committer"
	"github.com/light-bringer/procat-service/internal/transport/grpc/product"
	pb "github.com/light-bringer/procat-service/proto/product/v1"
	"github.com/light-bringer/procat-service/tests/testutil"
)

const bufSize = 1024 * 1024

// setupGRPCTest creates an in-memory gRPC server for testing.
func setupGRPCTest(t *testing.T) (pb.ProductServiceClient, func()) {
	t.Helper()

	// Setup Spanner
	client, cleanupDB := testutil.SetupSpannerTest(t)

	// Create infrastructure
	clk := clock.NewRealClock()
	comm := committer.NewCommitter(client)

	// Create repositories
	productRepo := repo.NewProductRepo(client, clk)
	outboxRepo := repo.NewOutboxRepo(client)
	priceHistoryRepo := repo.NewPriceHistoryRepo(client)
	readModel := repo.NewReadModel(client, clk)

	// Create use cases
	createProductUC := create_product.NewInteractor(productRepo, outboxRepo, priceHistoryRepo, comm, clk)
	updateProductUC := update_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	updatePriceUC := update_price.NewInteractor(productRepo, outboxRepo, priceHistoryRepo, comm, clk)
	activateProductUC := activate_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	deactivateProductUC := deactivate_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	applyDiscountUC := apply_discount.NewInteractor(productRepo, outboxRepo, comm, clk)
	removeDiscountUC := remove_discount.NewInteractor(productRepo, outboxRepo, comm, clk)
	archiveProductUC := archive_product.NewInteractor(productRepo, outboxRepo, comm, clk)

	// Create queries
	getProductQ := get_product.NewQuery(readModel)
	listProductsQ := list_products.NewQuery(readModel)
	eventsReadModel := repo.NewEventsReadModel(client)
	listEventsQ := list_events.NewQuery(eventsReadModel)

	// Create handler
	handler := product.NewHandler(
		createProductUC,
		updateProductUC,
		updatePriceUC,
		activateProductUC,
		deactivateProductUC,
		applyDiscountUC,
		removeDiscountUC,
		archiveProductUC,
		getProductQ,
		listProductsQ,
		listEventsQ,
	)

	// Setup in-memory gRPC server
	lis := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	pb.RegisterProductServiceServer(server, handler)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Create client
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	grpcClient := pb.NewProductServiceClient(conn)

	cleanup := func() {
		conn.Close()
		server.Stop()
		cleanupDB()
	}

	return grpcClient, cleanup
}

func TestGRPC_CreateProduct(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := &pb.CreateProductRequest{
			Name:        "Test Product",
			Description: "A test product",
			Category:    "electronics",
			BasePrice: &pb.Money{
				Numerator:   249900,
				Denominator: 100,
			},
		}

		resp, err := client.CreateProduct(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.ProductId)
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		req := &pb.CreateProductRequest{
			Name:        "",
			Description: "Test",
			Category:    "electronics",
			BasePrice: &pb.Money{
				Numerator:   10000,
				Denominator: 100,
			},
		}

		_, err := client.CreateProduct(ctx, req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("validation error - missing base price", func(t *testing.T) {
		req := &pb.CreateProductRequest{
			Name:        "Test",
			Description: "Test",
			Category:    "electronics",
			BasePrice:   nil,
		}

		_, err := client.CreateProduct(ctx, req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("domain validation - negative price", func(t *testing.T) {
		req := &pb.CreateProductRequest{
			Name:        "Test",
			Description: "Test",
			Category:    "electronics",
			BasePrice: &pb.Money{
				Numerator:   -10000,
				Denominator: 100,
			},
		}

		_, err := client.CreateProduct(ctx, req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "price")
	})
}

func TestGRPC_GetProduct(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a product first
	createResp, err := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "MacBook Pro",
		Description: "16-inch laptop",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 249900, Denominator: 100},
	})
	require.NoError(t, err)

	t.Run("get existing product", func(t *testing.T) {
		resp, err := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		require.NoError(t, err)

		product := resp.Product
		assert.Equal(t, createResp.ProductId, product.ProductId)
		assert.Equal(t, "MacBook Pro", product.Name)
		assert.Equal(t, "16-inch laptop", product.Description)
		assert.Equal(t, "electronics", product.Category)
		assert.Equal(t, 2499.00, product.BasePrice)
		assert.Equal(t, 2499.00, product.EffectivePrice)
		assert.Equal(t, "inactive", product.Status)
		assert.False(t, product.DiscountActive)
	})

	t.Run("get non-existent product", func(t *testing.T) {
		_, err := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: "non-existent-id",
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})
}

func TestGRPC_UpdateProduct(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a product
	createResp, _ := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Original Name",
		Description: "Original Description",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 10000, Denominator: 100},
	})

	t.Run("update product fields", func(t *testing.T) {
		newName := "Updated Name"
		newCategory := "books"

		_, err := client.UpdateProduct(ctx, &pb.UpdateProductRequest{
			ProductId: createResp.ProductId,
			Name:      &newName,
			Category:  &newCategory,
		})
		require.NoError(t, err)

		// Verify updates
		getResp, _ := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		assert.Equal(t, "Updated Name", getResp.Product.Name)
		assert.Equal(t, "books", getResp.Product.Category)
		assert.Equal(t, "Original Description", getResp.Product.Description)
	})

	t.Run("validation error - no fields to update", func(t *testing.T) {
		_, err := client.UpdateProduct(ctx, &pb.UpdateProductRequest{
			ProductId: createResp.ProductId,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGRPC_ActivateDeactivateProduct(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a product
	createResp, _ := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Test Product",
		Description: "Test",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 10000, Denominator: 100},
	})

	t.Run("activate product", func(t *testing.T) {
		_, err := client.ActivateProduct(ctx, &pb.ActivateProductRequest{
			ProductId: createResp.ProductId,
		})
		require.NoError(t, err)

		// Verify status
		getResp, _ := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		assert.Equal(t, "active", getResp.Product.Status)
	})

	t.Run("deactivate product", func(t *testing.T) {
		_, err := client.DeactivateProduct(ctx, &pb.DeactivateProductRequest{
			ProductId: createResp.ProductId,
		})
		require.NoError(t, err)

		// Verify status
		getResp, _ := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		assert.Equal(t, "inactive", getResp.Product.Status)
	})

	t.Run("cannot activate already active product", func(t *testing.T) {
		// Activate once
		client.ActivateProduct(ctx, &pb.ActivateProductRequest{
			ProductId: createResp.ProductId,
		})

		// Try to activate again
		_, err := client.ActivateProduct(ctx, &pb.ActivateProductRequest{
			ProductId: createResp.ProductId,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "already active")
	})
}

func TestGRPC_ApplyDiscount(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create and activate a product
	createResp, _ := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Expensive Product",
		Description: "High-end",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 100000, Denominator: 100},
	})
	client.ActivateProduct(ctx, &pb.ActivateProductRequest{
		ProductId: createResp.ProductId,
	})

	t.Run("apply discount successfully", func(t *testing.T) {
		startDate := time.Now()
		endDate := startDate.Add(30 * 24 * time.Hour)

		_, err := client.ApplyDiscount(ctx, &pb.ApplyDiscountRequest{
			ProductId:       createResp.ProductId,
			DiscountPercent: 20,
			StartDate:       timestamppb.New(startDate),
			EndDate:         timestamppb.New(endDate),
		})
		require.NoError(t, err)

		// Verify discount applied
		getResp, _ := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		assert.Equal(t, 1000.00, getResp.Product.BasePrice)
		assert.Equal(t, 800.00, getResp.Product.EffectivePrice)
		assert.True(t, getResp.Product.DiscountActive)
		require.NotNil(t, getResp.Product.DiscountPercent)
		assert.Equal(t, int64(20), *getResp.Product.DiscountPercent)
	})

	t.Run("validation error - invalid discount percentage", func(t *testing.T) {
		_, err := client.ApplyDiscount(ctx, &pb.ApplyDiscountRequest{
			ProductId:       createResp.ProductId,
			DiscountPercent: 150,
			StartDate:       timestamppb.Now(),
			EndDate:         timestamppb.New(time.Now().Add(24 * time.Hour)),
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

func TestGRPC_RemoveDiscount(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create and activate product with discount
	createResp, _ := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Product",
		Description: "Test",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 10000, Denominator: 100},
	})
	client.ActivateProduct(ctx, &pb.ActivateProductRequest{ProductId: createResp.ProductId})
	client.ApplyDiscount(ctx, &pb.ApplyDiscountRequest{
		ProductId:       createResp.ProductId,
		DiscountPercent: 15,
		StartDate:       timestamppb.Now(),
		EndDate:         timestamppb.New(time.Now().Add(24 * time.Hour)),
	})

	t.Run("remove discount successfully", func(t *testing.T) {
		_, err := client.RemoveDiscount(ctx, &pb.RemoveDiscountRequest{
			ProductId: createResp.ProductId,
		})
		require.NoError(t, err)

		// Verify discount removed
		getResp, _ := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		assert.False(t, getResp.Product.DiscountActive)
		assert.Equal(t, 100.00, getResp.Product.EffectivePrice)
		assert.Nil(t, getResp.Product.DiscountPercent)
	})
}

func TestGRPC_ArchiveProduct(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a product
	createResp, _ := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Product to Archive",
		Description: "Test",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 10000, Denominator: 100},
	})

	t.Run("archive product successfully", func(t *testing.T) {
		_, err := client.ArchiveProduct(ctx, &pb.ArchiveProductRequest{
			ProductId: createResp.ProductId,
		})
		require.NoError(t, err)

		// Verify status
		getResp, _ := client.GetProduct(ctx, &pb.GetProductRequest{
			ProductId: createResp.ProductId,
		})
		assert.Equal(t, "archived", getResp.Product.Status)
	})

	t.Run("cannot modify archived product", func(t *testing.T) {
		newName := "Should Fail"
		_, err := client.UpdateProduct(ctx, &pb.UpdateProductRequest{
			ProductId: createResp.ProductId,
			Name:      &newName,
		})
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "archived")
	})
}

func TestGRPC_ListProducts(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create multiple products
	client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name: "Product 1", Description: "Test", Category: "electronics",
		BasePrice: &pb.Money{Numerator: 10000, Denominator: 100},
	})
	client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name: "Product 2", Description: "Test", Category: "books",
		BasePrice: &pb.Money{Numerator: 5000, Denominator: 100},
	})
	client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name: "Product 3", Description: "Test", Category: "electronics",
		BasePrice: &pb.Money{Numerator: 15000, Denominator: 100},
	})

	t.Run("list all products", func(t *testing.T) {
		resp, err := client.ListProducts(ctx, &pb.ListProductsRequest{
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Products), 3)
	})

	t.Run("filter by category", func(t *testing.T) {
		resp, err := client.ListProducts(ctx, &pb.ListProductsRequest{
			Category: "electronics",
			PageSize: 10,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Products), 2)
		for _, p := range resp.Products {
			assert.Equal(t, "electronics", p.Category)
		}
	})

	t.Run("pagination", func(t *testing.T) {
		resp, err := client.ListProducts(ctx, &pb.ListProductsRequest{
			PageSize: 2,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(resp.Products), 2)
	})
}

func TestGRPC_ConcurrentRequests(t *testing.T) {
	client, cleanup := setupGRPCTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a product
	createResp, _ := client.CreateProduct(ctx, &pb.CreateProductRequest{
		Name:        "Concurrent Test",
		Description: "Test",
		Category:    "electronics",
		BasePrice:   &pb.Money{Numerator: 10000, Denominator: 100},
	})

	// Activate it
	client.ActivateProduct(ctx, &pb.ActivateProductRequest{
		ProductId: createResp.ProductId,
	})

	t.Run("concurrent reads are safe", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				_, err := client.GetProduct(ctx, &pb.GetProductRequest{
					ProductId: createResp.ProductId,
				})
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
