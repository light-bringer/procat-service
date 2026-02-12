package e2e

import (
	"context"
	"testing"

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
	"github.com/light-bringer/procat-service/tests/testutil"
)

// Services holds all use cases and queries for E2E tests.
type Services struct {
	// Commands
	CreateProduct     *create_product.Interactor
	UpdateProduct     *update_product.Interactor
	ActivateProduct   *activate_product.Interactor
	DeactivateProduct *deactivate_product.Interactor
	ApplyDiscount     *apply_discount.Interactor
	RemoveDiscount    *remove_discount.Interactor
	ArchiveProduct    *archive_product.Interactor

	// Queries
	GetProduct   *get_product.Query
	ListProducts *list_products.Query

	// Infrastructure
	Clock  clock.Clock
	Client *spanner.Client
}

// setupTest initializes all dependencies for E2E testing.
func setupTest(t *testing.T) (*Services, func()) {
	t.Helper()

	// Setup Spanner client with clean database
	client, cleanup := testutil.SetupSpannerTest(t)

	// Create infrastructure components
	clk := clock.NewRealClock()
	comm := committer.NewCommitter(client)

	// Create repositories
	productRepo := repo.NewProductRepo(client)
	outboxRepo := repo.NewOutboxRepo(client)
	readModel := repo.NewReadModel(client, clk)

	// Create command use cases
	createProductUseCase := create_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	updateProductUseCase := update_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	activateProductUseCase := activate_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	deactivateProductUseCase := deactivate_product.NewInteractor(productRepo, outboxRepo, comm, clk)
	applyDiscountUseCase := apply_discount.NewInteractor(productRepo, outboxRepo, comm, clk)
	removeDiscountUseCase := remove_discount.NewInteractor(productRepo, outboxRepo, comm, clk)
	archiveProductUseCase := archive_product.NewInteractor(productRepo, outboxRepo, comm, clk)

	// Create query use cases
	getProductQuery := get_product.NewQuery(readModel)
	listProductsQuery := list_products.NewQuery(readModel)

	services := &Services{
		CreateProduct:     createProductUseCase,
		UpdateProduct:     updateProductUseCase,
		ActivateProduct:   activateProductUseCase,
		DeactivateProduct: deactivateProductUseCase,
		ApplyDiscount:     applyDiscountUseCase,
		RemoveDiscount:    removeDiscountUseCase,
		ArchiveProduct:    archiveProductUseCase,
		GetProduct:        getProductQuery,
		ListProducts:      listProductsQuery,
		Clock:             clk,
		Client:            client,
	}

	return services, cleanup
}

// setupTestWithMockClock initializes services with a controllable mock clock.
func setupTestWithMockClock(t *testing.T) (*Services, *clock.MockClock, func()) {
	t.Helper()

	// Setup Spanner client
	client, cleanup := testutil.SetupSpannerTest(t)

	// Create mock clock
	mockClock := testutil.NewMockClock()
	comm := committer.NewCommitter(client)

	// Create repositories
	productRepo := repo.NewProductRepo(client)
	outboxRepo := repo.NewOutboxRepo(client)
	readModel := repo.NewReadModel(client, mockClock)

	// Create command use cases with mock clock
	createProductUseCase := create_product.NewInteractor(productRepo, outboxRepo, comm, mockClock)
	updateProductUseCase := update_product.NewInteractor(productRepo, outboxRepo, comm, mockClock)
	activateProductUseCase := activate_product.NewInteractor(productRepo, outboxRepo, comm, mockClock)
	deactivateProductUseCase := deactivate_product.NewInteractor(productRepo, outboxRepo, comm, mockClock)
	applyDiscountUseCase := apply_discount.NewInteractor(productRepo, outboxRepo, comm, mockClock)
	removeDiscountUseCase := remove_discount.NewInteractor(productRepo, outboxRepo, comm, mockClock)
	archiveProductUseCase := archive_product.NewInteractor(productRepo, outboxRepo, comm, mockClock)

	// Create query use cases
	getProductQuery := get_product.NewQuery(readModel)
	listProductsQuery := list_products.NewQuery(readModel)

	services := &Services{
		CreateProduct:     createProductUseCase,
		UpdateProduct:     updateProductUseCase,
		ActivateProduct:   activateProductUseCase,
		DeactivateProduct: deactivateProductUseCase,
		ApplyDiscount:     applyDiscountUseCase,
		RemoveDiscount:    removeDiscountUseCase,
		ArchiveProduct:    archiveProductUseCase,
		GetProduct:        getProductQuery,
		ListProducts:      listProductsQuery,
		Clock:             mockClock,
		Client:            client,
	}

	return services, mockClock, cleanup
}

// ctx returns a context for testing.
func ctx() context.Context {
	return context.Background()
}
