package reviewservice

import (
	"context"
	"errors"
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
	db       *gorm.DB
	reviews  repository.ReviewRepository
	products repository.ProductRepository
}

func NewService(db *gorm.DB, reviews repository.ReviewRepository, products repository.ProductRepository) *Service {
	return &Service{
		db:       db,
		reviews:  reviews,
		products: products,
	}
}

func (s *Service) ListByProductID(ctx context.Context, productID string) ([]model.Review, error) {
	if _, err := s.products.GetByID(ctx, productID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "product not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load product", err)
	}

	items, err := s.reviews.ListByProductID(ctx, productID)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to list reviews", err)
	}
	return items, nil
}

func (s *Service) ListByUserID(ctx context.Context, userID string, page dto.PageParams) ([]model.Review, dto.PageMeta, error) {
	page = page.Normalize(20)

	items, total, err := s.reviews.ListByUserID(ctx, userID, page)
	if err != nil {
		return nil, dto.PageMeta{}, apperror.Wrap(apperror.CodeInternal, "failed to list reviews", err)
	}

	return items, dto.NewPageMeta(page.Page, page.Limit, total), nil
}

func (s *Service) GetByUserAndProduct(ctx context.Context, userID, productID string) (*model.Review, error) {
	review, err := s.reviews.GetByUserAndProduct(ctx, userID, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "review not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load review", err)
	}
	return review, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, input dto.UpsertReviewInput) (*model.Review, error) {
	product, err := s.products.GetByID(ctx, input.ProductID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "product not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load product", err)
	}

	var result model.Review
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var review model.Review
		loadErr := tx.Where(`"userId" = ? AND "productId" = ?`, userID, input.ProductID).First(&review).Error
		now := time.Now().UTC()

		if loadErr != nil {
			if !errors.Is(loadErr, gorm.ErrRecordNotFound) {
				return loadErr
			}

			review = model.Review{
				ID:                 uuid.NewString(),
				UserID:             userID,
				ProductID:          input.ProductID,
				Rating:             input.Rating,
				Title:              input.Title,
				Description:        input.Description,
				IsVerifiedPurchase: true,
				CreatedAt:          now,
			}

			if err := tx.Create(&review).Error; err != nil {
				return err
			}
		} else {
			review.Rating = input.Rating
			review.Title = input.Title
			review.Description = input.Description

			if err := tx.Save(&review).Error; err != nil {
				return err
			}
		}

		var stats struct {
			AverageRating decimal.Decimal
			NumReviews    int64
		}
		if err := tx.Model(&model.Review{}).
			Select(`COALESCE(AVG(rating), 0) AS average_rating, COUNT(*) AS num_reviews`).
			Where(`"productId" = ?`, input.ProductID).
			Scan(&stats).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Product{}).
			Where("id = ?", input.ProductID).
			Updates(map[string]interface{}{
				"rating":     stats.AverageRating,
				"numReviews": int(stats.NumReviews),
			}).Error; err != nil {
			return err
		}

		result = review
		return nil
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to save review", err)
	}

	if refreshed, err := s.products.GetByID(ctx, product.ID); err == nil {
		product = refreshed
	}
	_ = product

	return &result, nil
}
