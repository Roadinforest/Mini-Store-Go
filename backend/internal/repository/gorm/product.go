package gormrepo

import (
	"context"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) repository.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	var product model.Product
	if err := r.db.WithContext(ctx).First(&product, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) GetBySlug(ctx context.Context, slug string) (*model.Product, error) {
	var product model.Product
	if err := r.db.WithContext(ctx).First(&product, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) List(ctx context.Context, filter dto.ProductListFilter) ([]model.Product, int64, error) {
	filter = dto.ProductListFilter{
		PageParams: filter.PageParams.Normalize(20),
		Query:      filter.Query,
		Category:   filter.Category,
		Price:      filter.Price,
		Rating:     filter.Rating,
		Sort:       filter.Sort,
	}

	query := r.db.WithContext(ctx).Model(&model.Product{})
	query = applyProductFilter(query, filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []model.Product
	if err := applyProductSort(query, filter.Sort).
		Offset(filter.Offset()).
		Limit(filter.Limit).
		Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

func (r *productRepository) ListLatest(ctx context.Context, limit int) ([]model.Product, error) {
	var products []model.Product
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *productRepository) ListFeatured(ctx context.Context, limit int) ([]model.Product, error) {
	var products []model.Product
	if err := r.db.WithContext(ctx).
		Where("is_featured = ?", true).
		Order("created_at DESC").
		Limit(limit).
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *productRepository) ListCategories(ctx context.Context) ([]repository.CategoryCount, error) {
	var counts []repository.CategoryCount
	if err := r.db.WithContext(ctx).
		Model(&model.Product{}).
		Select("category, COUNT(*) AS count").
		Group("category").
		Order("category ASC").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	return counts, nil
}

func (r *productRepository) Create(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *productRepository) Update(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Product{}, "id = ?", id).Error
}

func applyProductFilter(query *gorm.DB, filter dto.ProductListFilter) *gorm.DB {
	if trimmed := strings.TrimSpace(filter.Query); trimmed != "" && trimmed != "all" {
		query = query.Where("name ILIKE ?", "%"+trimmed+"%")
	}
	if trimmed := strings.TrimSpace(filter.Category); trimmed != "" && trimmed != "all" {
		query = query.Where("category = ?", trimmed)
	}
	if trimmed := strings.TrimSpace(filter.Price); trimmed != "" && trimmed != "all" {
		parts := strings.Split(trimmed, "-")
		if len(parts) == 2 {
			minPrice, minErr := strconv.Atoi(parts[0])
			maxPrice, maxErr := strconv.Atoi(parts[1])
			if minErr == nil && maxErr == nil {
				query = query.Where("price BETWEEN ? AND ?", minPrice, maxPrice)
			}
		}
	}
	if trimmed := strings.TrimSpace(filter.Rating); trimmed != "" && trimmed != "all" {
		if minRating, err := strconv.Atoi(trimmed); err == nil {
			query = query.Where("rating >= ?", minRating)
		}
	}
	return query
}

func applyProductSort(query *gorm.DB, sort string) *gorm.DB {
	switch sort {
	case "lowest":
		return query.Order("price ASC")
	case "highest":
		return query.Order("price DESC")
	case "rating":
		return query.Order("rating DESC")
	default:
		return query.Order("created_at DESC")
	}
}
