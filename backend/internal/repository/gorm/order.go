package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) repository.OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	var order model.Order
	if err := r.db.WithContext(ctx).
		Preload("OrderItems").
		Preload("User").
		First(&order, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) ListByUserID(ctx context.Context, userID string, page dto.PageParams) ([]model.Order, int64, error) {
	page = page.Normalize(20)

	query := r.db.WithContext(ctx).Model(&model.Order{}).Where(`"userId" = ?`, userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []model.Order
	if err := query.
		Preload("OrderItems").
		Order(`"createdAt" DESC`).
		Offset(page.Offset()).
		Limit(page.Limit).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *orderRepository) List(ctx context.Context, page dto.PageParams) ([]model.Order, int64, error) {
	page = page.Normalize(20)

	query := r.db.WithContext(ctx).Model(&model.Order{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []model.Order
	if err := query.
		Preload("User").
		Preload("OrderItems").
		Order(`"createdAt" DESC`).
		Offset(page.Offset()).
		Limit(page.Limit).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func (r *orderRepository) Create(ctx context.Context, order *model.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *orderRepository) Update(ctx context.Context, order *model.Order) error {
	return r.db.WithContext(ctx).Save(order).Error
}
