package repository

import (
	"context"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
)

type ProductRepository interface {
	GetByID(ctx context.Context, id string) (*model.Product, error)
	GetBySlug(ctx context.Context, slug string) (*model.Product, error)
	List(ctx context.Context, filter dto.ProductListFilter) ([]model.Product, int64, error)
	ListLatest(ctx context.Context, limit int) ([]model.Product, error)
	ListFeatured(ctx context.Context, limit int) ([]model.Product, error)
	ListCategories(ctx context.Context) ([]CategoryCount, error)
	Create(ctx context.Context, product *model.Product) error
	Update(ctx context.Context, product *model.Product) error
	Delete(ctx context.Context, id string) error
}

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, filter dto.UserListFilter) ([]model.User, int64, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id string) error
}

type CartRepository interface {
	GetByID(ctx context.Context, id string) (*model.Cart, error)
	GetByUserID(ctx context.Context, userID string) (*model.Cart, error)
	GetBySessionCartID(ctx context.Context, sessionCartID string) (*model.Cart, error)
	Create(ctx context.Context, cart *model.Cart) error
	Update(ctx context.Context, cart *model.Cart) error
	Delete(ctx context.Context, id string) error
}

type OrderRepository interface {
	GetByID(ctx context.Context, id string) (*model.Order, error)
	ListByUserID(ctx context.Context, userID string, page dto.PageParams) ([]model.Order, int64, error)
	List(ctx context.Context, page dto.PageParams) ([]model.Order, int64, error)
	Create(ctx context.Context, order *model.Order) error
	Update(ctx context.Context, order *model.Order) error
}

type ReviewRepository interface {
	GetByID(ctx context.Context, id string) (*model.Review, error)
	GetByUserAndProduct(ctx context.Context, userID, productID string) (*model.Review, error)
	ListByProductID(ctx context.Context, productID string) ([]model.Review, error)
	Create(ctx context.Context, review *model.Review) error
	Update(ctx context.Context, review *model.Review) error
}

type CategoryCount struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}
