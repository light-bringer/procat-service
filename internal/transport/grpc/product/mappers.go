package product

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/light-bringer/procat-service/internal/app/product/contracts"
	"github.com/light-bringer/procat-service/internal/app/product/domain"
	pb "github.com/light-bringer/procat-service/proto/product/v1"
)

// protoMoneyToDomain converts proto Money to domain Money.
func protoMoneyToDomain(m *pb.Money) (*domain.Money, error) {
	if m == nil {
		return nil, nil
	}
	return domain.NewMoney(m.Numerator, m.Denominator)
}

// dtoToProtoProduct converts a ProductDTO to proto Product.
func dtoToProtoProduct(dto *contracts.ProductDTO) *pb.Product {
	p := &pb.Product{
		ProductId:      dto.ProductID,
		Name:           dto.Name,
		Description:    dto.Description,
		Category:       dto.Category,
		BasePrice:      dto.BasePrice,
		EffectivePrice: dto.EffectivePrice,
		DiscountActive: dto.DiscountActive,
		Status:         dto.Status,
		CreatedAt:      timestamppb.New(dto.CreatedAt),
		UpdatedAt:      timestamppb.New(dto.UpdatedAt),
	}

	if dto.DiscountPercent != nil {
		p.DiscountPercent = dto.DiscountPercent
	}

	return p
}
