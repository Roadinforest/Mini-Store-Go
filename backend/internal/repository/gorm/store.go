package gormrepo

import (
	"gorm.io/gorm"

	"mini-store-go/backend/internal/repository"
)

func NewStore(db *gorm.DB) *repository.Store {
	return &repository.Store{
		Products: NewProductRepository(db),
		Users:    NewUserRepository(db),
		Carts:    NewCartRepository(db),
		Orders:   NewOrderRepository(db),
		Reviews:  NewReviewRepository(db),
	}
}
