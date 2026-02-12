package product

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/light-bringer/procat-service/internal/app/product/queries/get_product"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_events"
	"github.com/light-bringer/procat-service/internal/app/product/queries/list_products"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/activate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/apply_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/archive_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/create_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/deactivate_product"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/remove_discount"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_price"
	"github.com/light-bringer/procat-service/internal/app/product/usecases/update_product"
	pb "github.com/light-bringer/procat-service/proto/product/v1"
)

// Handler implements the gRPC ProductService interface.
// It's a thin coordinator that delegates to use cases and queries.
type Handler struct {
	pb.UnimplementedProductServiceServer

	// Commands
	createProduct     *create_product.Interactor
	updateProduct     *update_product.Interactor
	updatePrice       *update_price.Interactor
	activateProduct   *activate_product.Interactor
	deactivateProduct *deactivate_product.Interactor
	applyDiscount     *apply_discount.Interactor
	removeDiscount    *remove_discount.Interactor
	archiveProduct    *archive_product.Interactor

	// Queries
	getProduct   *get_product.Query
	listProducts *list_products.Query
	listEvents   *list_events.Query
}

// NewHandler creates a new gRPC product handler.
func NewHandler(
	createProduct *create_product.Interactor,
	updateProduct *update_product.Interactor,
	updatePrice *update_price.Interactor,
	activateProduct *activate_product.Interactor,
	deactivateProduct *deactivate_product.Interactor,
	applyDiscount *apply_discount.Interactor,
	removeDiscount *remove_discount.Interactor,
	archiveProduct *archive_product.Interactor,
	getProduct *get_product.Query,
	listProducts *list_products.Query,
	listEvents *list_events.Query,
) *Handler {
	return &Handler{
		createProduct:     createProduct,
		updateProduct:     updateProduct,
		updatePrice:       updatePrice,
		activateProduct:   activateProduct,
		deactivateProduct: deactivateProduct,
		applyDiscount:     applyDiscount,
		removeDiscount:    removeDiscount,
		archiveProduct:    archiveProduct,
		getProduct:        getProduct,
		listProducts:      listProducts,
		listEvents:        listEvents,
	}
}

// CreateProduct creates a new product.
func (h *Handler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductReply, error) {
	// 1. Validate proto request
	if err := validateCreateProductRequest(req); err != nil {
		return nil, err
	}

	// 2. Map proto → application request
	basePrice, err := protoMoneyToDomain(req.BasePrice)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid base_price")
	}

	appReq := &create_product.Request{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		BasePrice:   basePrice,
	}

	// 3. Call usecase (usecase applies plan)
	productID, err := h.createProduct.Execute(ctx, appReq)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// 4. Return response
	return &pb.CreateProductReply{ProductId: productID}, nil
}

// UpdateProduct updates product details.
func (h *Handler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductReply, error) {
	// 1. Validate proto request
	if err := validateUpdateProductRequest(req); err != nil {
		return nil, err
	}

	// 2. Map proto → application request
	appReq := &update_product.Request{
		ProductID:   req.ProductId,
		Version:     req.GetVersion(), // Optional version for optimistic locking
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
	}

	// 3. Call usecase
	if err := h.updateProduct.Execute(ctx, appReq); err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// 4. Return response
	return &pb.UpdateProductReply{}, nil
}

// UpdatePrice updates a product's price.
func (h *Handler) UpdatePrice(ctx context.Context, req *pb.UpdatePriceRequest) (*pb.UpdatePriceReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	// Map proto money to domain money
	newPrice, err := protoMoneyToDomain(req.NewPrice)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid new_price")
	}

	appReq := &update_price.Request{
		ProductID:     req.ProductId,
		Version:       req.GetVersion(), // Optional version for optimistic locking
		NewPrice:      newPrice,
		ChangedBy:     req.ChangedBy,
		ChangedReason: req.ChangedReason,
	}

	if err := h.updatePrice.Execute(ctx, appReq); err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.UpdatePriceReply{}, nil
}

// ActivateProduct activates a product.
func (h *Handler) ActivateProduct(ctx context.Context, req *pb.ActivateProductRequest) (*pb.ActivateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	appReq := &activate_product.Request{
		ProductID: req.ProductId,
		Version:   req.GetVersion(), // Optional version for optimistic locking
	}
	if err := h.activateProduct.Execute(ctx, appReq); err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.ActivateProductReply{}, nil
}

// DeactivateProduct deactivates a product.
func (h *Handler) DeactivateProduct(ctx context.Context, req *pb.DeactivateProductRequest) (*pb.DeactivateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	appReq := &deactivate_product.Request{
		ProductID: req.ProductId,
		Version:   req.GetVersion(), // Optional version for optimistic locking
	}
	if err := h.deactivateProduct.Execute(ctx, appReq); err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.DeactivateProductReply{}, nil
}

// ApplyDiscount applies a discount to a product.
func (h *Handler) ApplyDiscount(ctx context.Context, req *pb.ApplyDiscountRequest) (*pb.ApplyDiscountReply, error) {
	// 1. Validate proto request
	if err := validateApplyDiscountRequest(req); err != nil {
		return nil, err
	}

	// 2. Map proto → application request
	appReq := &apply_discount.Request{
		ProductID:       req.ProductId,
		Version:         req.GetVersion(),    // Optional version for optimistic locking
		DiscountPercent: req.DiscountPercent, // Now float64
		StartDate:       req.StartDate.AsTime(),
		EndDate:         req.EndDate.AsTime(),
	}

	// 3. Call usecase
	if err := h.applyDiscount.Execute(ctx, appReq); err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	// 4. Return response
	return &pb.ApplyDiscountReply{}, nil
}

// RemoveDiscount removes a discount from a product.
func (h *Handler) RemoveDiscount(ctx context.Context, req *pb.RemoveDiscountRequest) (*pb.RemoveDiscountReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	appReq := &remove_discount.Request{
		ProductID: req.ProductId,
		Version:   req.GetVersion(), // Optional version for optimistic locking
	}
	if err := h.removeDiscount.Execute(ctx, appReq); err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.RemoveDiscountReply{}, nil
}

// ArchiveProduct archives a product (soft delete).
func (h *Handler) ArchiveProduct(ctx context.Context, req *pb.ArchiveProductRequest) (*pb.ArchiveProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	appReq := &archive_product.Request{
		ProductID: req.ProductId,
		Version:   req.GetVersion(), // Optional version for optimistic locking
	}
	archivedAt, err := h.archiveProduct.Execute(ctx, appReq)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.ArchiveProductReply{
		ArchivedAt: timestamppb.New(archivedAt),
	}, nil
}

// GetProduct retrieves a product by ID.
func (h *Handler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	queryReq := &get_product.Request{ProductID: req.ProductId}
	dto, err := h.getProduct.Execute(ctx, queryReq)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	return &pb.GetProductReply{
		Product: dtoToProtoProduct(dto),
	}, nil
}

// ListProducts retrieves a paginated list of products.
func (h *Handler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsReply, error) {
	queryReq := &list_products.Request{
		Category:  req.Category,
		Status:    req.Status,
		PageSize:  int(req.PageSize),
		PageToken: req.PageToken,
	}

	result, err := h.listProducts.Execute(ctx, queryReq)
	if err != nil {
		return nil, mapDomainErrorToGRPC(err)
	}

	products := make([]*pb.Product, 0, len(result.Products))
	for _, dto := range result.Products {
		products = append(products, dtoToProtoProduct(dto))
	}

	return &pb.ListProductsReply{
		Products:      products,
		NextPageToken: result.NextPageToken,
		TotalCount:    result.TotalCount,
	}, nil
}

// ListEvents retrieves a list of domain events from the outbox.
func (h *Handler) ListEvents(ctx context.Context, req *pb.ListEventsRequest) (*pb.ListEventsReply, error) {
	queryReq := &list_events.Request{
		Limit: int(req.Limit),
	}

	// Set optional filters
	if req.EventType != nil {
		queryReq.EventType = req.EventType
	}
	if req.AggregateId != nil {
		queryReq.AggregateID = req.AggregateId
	}
	if req.Status != nil {
		queryReq.Status = req.Status
	}

	events, totalCount, err := h.listEvents.Execute(ctx, queryReq)
	if err != nil {
		// Temporarily return the actual error for debugging
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list events: %v", err))
	}

	// Convert events to proto
	protoEvents := make([]*pb.Event, 0, len(events))
	for _, event := range events {
		// Convert NullJSON to string
		payload := ""
		if event.Payload.Valid {
			// NullJSON.Value is interface{}, need to marshal to JSON string
			payloadBytes, err := json.Marshal(event.Payload.Value)
			if err == nil {
				payload = string(payloadBytes)
			}
		}

		protoEvent := &pb.Event{
			EventId:     event.EventID,
			EventType:   event.EventType,
			AggregateId: event.AggregateID,
			Payload:     payload,
			Status:      event.Status,
			CreatedAt:   timestamppb.New(event.CreatedAt),
		}
		if event.ProcessedAt.Valid {
			protoEvent.ProcessedAt = timestamppb.New(event.ProcessedAt.Time)
		}
		protoEvents = append(protoEvents, protoEvent)
	}

	return &pb.ListEventsReply{
		Events:     protoEvents,
		TotalCount: totalCount,
	}, nil
}
