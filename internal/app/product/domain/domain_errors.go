package domain

import "errors"

// Domain errors as sentinel values
var (
	// Product errors
	ErrProductNotFound        = errors.New("product not found")
	ErrProductNotActive       = errors.New("product is not active")
	ErrEmptyName              = errors.New("product name cannot be empty")
	ErrInvalidPrice           = errors.New("product price must be positive")
	ErrInvalidCategory        = errors.New("product category cannot be empty")
	ErrOptimisticLockConflict = errors.New("product was modified by another transaction")
	ErrMoneyOverflow          = errors.New("money value exceeds int64 bounds")

	// Discount errors
	ErrInvalidDiscountPeriod  = errors.New("discount end date must be after start date")
	ErrDiscountAlreadyActive  = errors.New("product already has an active discount")
	ErrInvalidDiscountPercent = errors.New("discount percentage must be between 0 and 100")
	ErrCannotApplyToInactive  = errors.New("cannot apply discount to inactive product")

	// Status errors
	ErrAlreadyActive        = errors.New("product is already active")
	ErrAlreadyInactive      = errors.New("product is already inactive")
	ErrAlreadyArchived      = errors.New("product is already archived")
	ErrCannotModifyArchived = errors.New("cannot modify archived product")
)
