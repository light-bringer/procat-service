package product

import (
	"errors"

	"github.com/light-bringer/procat-service/internal/app/product/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapDomainErrorToGRPC converts domain errors to gRPC status codes.
func mapDomainErrorToGRPC(err error) error {
	if err == nil {
		return nil
	}

	// Map specific domain errors to gRPC codes
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		return status.Error(codes.NotFound, "product not found")

	case errors.Is(err, domain.ErrProductNotActive):
		return status.Error(codes.FailedPrecondition, "product is not active")

	case errors.Is(err, domain.ErrEmptyName):
		return status.Error(codes.InvalidArgument, "product name cannot be empty")

	case errors.Is(err, domain.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, "product price must be positive")

	case errors.Is(err, domain.ErrInvalidCategory):
		return status.Error(codes.InvalidArgument, "product category cannot be empty")

	case errors.Is(err, domain.ErrInvalidDiscountPeriod):
		return status.Error(codes.InvalidArgument, "discount end date must be after start date")

	case errors.Is(err, domain.ErrDiscountAlreadyActive):
		return status.Error(codes.FailedPrecondition, "product already has an active discount")

	case errors.Is(err, domain.ErrInvalidDiscountPercent):
		return status.Error(codes.InvalidArgument, "discount percentage must be between 0 and 100")

	case errors.Is(err, domain.ErrCannotApplyToInactive):
		return status.Error(codes.FailedPrecondition, "cannot apply discount to inactive product")

	case errors.Is(err, domain.ErrAlreadyActive):
		return status.Error(codes.FailedPrecondition, "product is already active")

	case errors.Is(err, domain.ErrAlreadyInactive):
		return status.Error(codes.FailedPrecondition, "product is already inactive")

	case errors.Is(err, domain.ErrAlreadyArchived):
		return status.Error(codes.FailedPrecondition, "product is already archived")

	case errors.Is(err, domain.ErrCannotModifyArchived):
		return status.Error(codes.FailedPrecondition, "cannot modify archived product")

	default:
		// Unknown error - return Internal
		return status.Error(codes.Internal, "internal server error")
	}
}
