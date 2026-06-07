package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/repository"
)

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) repository.CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) GetByID(ctx context.Context, id string) (*model.Cart, error) {
	var cart model.Cart
	if err := r.db.WithContext(ctx).First(&cart, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) GetByUserID(ctx context.Context, userID string) (*model.Cart, error) {
	var cart model.Cart
	if err := r.db.WithContext(ctx).First(&cart, `"userId" = ?`, userID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) GetBySessionCartID(ctx context.Context, sessionCartID string) (*model.Cart, error) {
	var cart model.Cart
	if err := r.db.WithContext(ctx).First(&cart, `"sessionCartId" = ?`, sessionCartID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) Create(ctx context.Context, cart *model.Cart) error {
	return r.db.WithContext(ctx).Create(cart).Error
}

func (r *cartRepository) Update(ctx context.Context, cart *model.Cart) error {
	return r.db.WithContext(ctx).Save(cart).Error
}

func (r *cartRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Cart{}, "id = ?", id).Error
}
