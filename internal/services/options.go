package services

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"

	"github.com/light-bringer/procat-service/internal/app/product/queries/get_product"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_products"
	"github.com/light-bringer/procat-service/internal/app/product/repo"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/activate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/apply_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/archive_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/create_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/deactivate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/remove_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_product"
	"github.com/light-bringer/procat-service/internal/pkg/clock"
	"github.com/light-bringer/procat-service/internal/pkg/committer"
	"github.com/light-bringer/procat-service/internal/transport/grpc/product"
)

// ServiceOptions holds all dependencies for the application.
type ServiceOptions struct {
	SpannerClient *spanner.Client
	ProductHandler *product.Handler
}

// NewServiceOptions creates and wires up all application dependencies.
func NewServiceOptions(ctx context.Context, spannerDB string) (*ServiceOptions, error) {
	// 1. Initialize Spanner client
	spannerClient, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %w", err)
	}

	// 2. Create infrastructure components
	clk := clock.NewRealClock()
	comm := committer.NewCommitter(spannerClient)

	// 3. Create repositories
	productRepo := repo.NewProductRepo(spannerClient)
	outboxRepo := repo.NewOutboxRepo(spannerClient)
	readModel := repo.NewReadModel(spannerClient)

	// 4. Create command use cases (write operations)
	createProductUseCase := create_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	updateProductUseCase := update_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	activateProductUseCase := activate_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	deactivateProductUseCase := deactivate_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	applyDiscountUseCase := apply_discount.NewInteractor(productRepo, outboxRepo, comm, clk)
	removeDiscountUseCase := remove_discount.NewInteractor(productRepo, outboxRepo, comm, clk)
	archiveProductUseCase := archive_product.NewInteractor(productRepo, outboxRepo, comm, clk)

	// 5. Create query use cases (read operations)
	getProductQuery := get_product.NewQuery(readModel)
	listProductsQuery := list_products.NewQuery(readModel)

	// 6. Create gRPC handler
	productHandler := product.NewHandler(
		createProductUseCase,
		updateProductUseCase,
		activateProductUseCase,
		deactivateProductUseCase,
		applyDiscountUseCase,
		removeDiscountUseCase,
		archiveProductUseCase,
		getProductQuery,
		listProductsQuery,
	)

	return &ServiceOptions{
		SpannerClient:  spannerClient,
		ProductHandler: productHandler,
	}, nil
}

// Close closes all resources.
func (s *ServiceOptions) Close() {
	if s.SpannerClient != nil {
		s.SpannerClient.Close()
	}
}
