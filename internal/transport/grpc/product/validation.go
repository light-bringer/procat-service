package product

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/light-bringer/procat-service/proto/product/v1"
)

// validateCreateProductRequest validates the CreateProduct request.
func validateCreateProductRequest(req *pb.CreateProductRequest) error {
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Category == "" {
		return status.Error(codes.InvalidArgument, "category is required")
	}
	if req.BasePrice == nil {
		return status.Error(codes.InvalidArgument, "base_price is required")
	}
	if req.BasePrice.Denominator == 0 {
		return status.Error(codes.InvalidArgument, "base_price denominator cannot be zero")
	}
	return nil
}

// validateUpdateProductRequest validates the UpdateProduct request.
func validateUpdateProductRequest(req *pb.UpdateProductRequest) error {
	if req.ProductId == "" {
		return status.Error(codes.InvalidArgument, "product_id is required")
	}
	// At least one field must be provided for update
	if req.Name == nil && req.Description == nil && req.Category == nil {
		return status.Error(codes.InvalidArgument, "at least one field must be provided for update")
	}
	return nil
}

// validateApplyDiscountRequest validates the ApplyDiscount request.
func validateApplyDiscountRequest(req *pb.ApplyDiscountRequest) error {
	if req.ProductId == "" {
		return status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.DiscountPercent < 0 || req.DiscountPercent > 100 {
		return status.Error(codes.InvalidArgument, "discount_percent must be between 0 and 100")
	}
	if req.StartDate == nil {
		return status.Error(codes.InvalidArgument, "start_date is required")
	}
	if req.EndDate == nil {
		return status.Error(codes.InvalidArgument, "end_date is required")
	}
	return nil
}
