package productservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

type Service struct {
	products repository.ProductRepository
}

func NewService(products repository.ProductRepository) *Service {
	return &Service{products: products}
}

func (s *Service) GetByID(ctx context.Context, productID string) (*model.Product, error) {
	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "product not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load product", err)
	}
	return product, nil
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*model.Product, error) {
	product, err := s.products.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "product not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load product", err)
	}
	return product, nil
}

func (s *Service) List(ctx context.Context, filter dto.ProductListFilter) ([]model.Product, dto.PageMeta, error) {
	filter.PageParams = filter.PageParams.Normalize(20)

	items, total, err := s.products.List(ctx, filter)
	if err != nil {
		return nil, dto.PageMeta{}, apperror.Wrap(apperror.CodeInternal, "failed to list products", err)
	}

	return items, dto.NewPageMeta(filter.Page, filter.Limit, total), nil
}

func (s *Service) ListLatest(ctx context.Context, limit int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 6
	}
	items, err := s.products.ListLatest(ctx, limit)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to list latest products", err)
	}
	return items, nil
}

func (s *Service) ListFeatured(ctx context.Context, limit int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 4
	}
	items, err := s.products.ListFeatured(ctx, limit)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to list featured products", err)
	}
	return items, nil
}

func (s *Service) ListCategories(ctx context.Context) ([]repository.CategoryCount, error) {
	items, err := s.products.ListCategories(ctx)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to list categories", err)
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, input dto.UpsertProductInput) (*model.Product, error) {
	product, err := buildProductModel("", input)
	if err != nil {
		return nil, err
	}
	product.ID = uuid.NewString()
	product.CreatedAt = time.Now().UTC()

	if err := s.products.Create(ctx, product); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create product", err)
	}

	return product, nil
}

func (s *Service) Update(ctx context.Context, productID string, input dto.UpsertProductInput) (*model.Product, error) {
	existing, err := s.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	product, err := buildProductModel(productID, input)
	if err != nil {
		return nil, err
	}
	product.CreatedAt = existing.CreatedAt

	if err := s.products.Update(ctx, product); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to update product", err)
	}

	return product, nil
}

func (s *Service) Delete(ctx context.Context, productID string) error {
	if _, err := s.GetByID(ctx, productID); err != nil {
		return err
	}

	if err := s.products.Delete(ctx, productID); err != nil {
		return apperror.Wrap(apperror.CodeInternal, "failed to delete product", err)
	}

	return nil
}

func buildProductModel(productID string, input dto.UpsertProductInput) (*model.Product, error) {
	price, err := decimal.NewFromString(strings.TrimSpace(input.Price))
	if err != nil {
		return nil, apperror.WithDetails(
			apperror.New(apperror.CodeValidation, "invalid price"),
			map[string]string{"field": "price"},
		)
	}

	rating := decimal.Zero
	if trimmed := strings.TrimSpace(input.Rating); trimmed != "" {
		rating, err = decimal.NewFromString(trimmed)
		if err != nil {
			return nil, apperror.WithDetails(
				apperror.New(apperror.CodeValidation, "invalid rating"),
				map[string]string{"field": "rating"},
			)
		}
	}

	var banner *string
	if input.Banner != nil {
		trimmed := strings.TrimSpace(*input.Banner)
		if trimmed != "" {
			banner = &trimmed
		}
	}

	return &model.Product{
		ID:          productID,
		Name:        strings.TrimSpace(input.Name),
		Slug:        strings.TrimSpace(input.Slug),
		Category:    strings.TrimSpace(input.Category),
		Images:      append([]string(nil), input.Images...),
		Brand:       strings.TrimSpace(input.Brand),
		Description: strings.TrimSpace(input.Description),
		Stock:       input.Stock,
		Price:       price,
		Rating:      rating,
		NumReviews:  input.NumReviews,
		IsFeatured:  input.IsFeatured,
		Banner:      banner,
	}, nil
}
