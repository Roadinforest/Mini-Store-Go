package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/repository"
)

type reviewRepository struct {
	db *gorm.DB
}

func NewReviewRepository(db *gorm.DB) repository.ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) GetByID(ctx context.Context, id string) (*model.Review, error) {
	var review model.Review
	if err := r.db.WithContext(ctx).First(&review, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) GetByUserAndProduct(ctx context.Context, userID, productID string) (*model.Review, error) {
	var review model.Review
	if err := r.db.WithContext(ctx).
		First(&review, "user_id = ? AND product_id = ?", userID, productID).Error; err != nil {
		return nil, err
	}
	return &review, nil
}

func (r *reviewRepository) ListByProductID(ctx context.Context, productID string) ([]model.Review, error) {
	var reviews []model.Review
	if err := r.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("created_at DESC").
		Find(&reviews).Error; err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *reviewRepository) Create(ctx context.Context, review *model.Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *reviewRepository) Update(ctx context.Context, review *model.Review) error {
	return r.db.WithContext(ctx).Save(review).Error
}
