package cartservice

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/infra/rediscache"
	"mini-store-go/backend/internal/repository"
	inventoryservice "mini-store-go/backend/internal/service/inventory"
	"time"
)

type Service struct {
	carts      repository.CartRepository
	products   repository.ProductRepository
	stockStore *rediscache.StockStore
	inventory  *inventoryservice.Service
}

func NewService(carts repository.CartRepository, products repository.ProductRepository, stockStore *rediscache.StockStore, inventory ...*inventoryservice.Service) *Service {
	s := &Service{carts: carts, products: products, stockStore: stockStore}
	if len(inventory) > 0 {
		s.inventory = inventory[0]
	}
	return s
}

func (s *Service) GetCurrentCart(ctx context.Context, sessionCartID string, userID *string) (*model.Cart, error) {
	return s.current(ctx, sessionCartID, userID, false, nil)
}

func (s *Service) AddItem(ctx context.Context, sessionCartID string, userID *string, productID string) (*model.Cart, error) {
	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "product not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load product", err)
	}
	available := s.availableStock(ctx, product)
	return s.current(ctx, sessionCartID, userID, true, func(cart *model.Cart) error {
		for i := range cart.Items {
			if cart.Items[i].ProductID == productID {
				if cart.Items[i].Qty >= available {
					return apperror.New(apperror.CodeOutOfStock, "not enough stock")
				}
				cart.Items[i].Qty++
				return nil
			}
		}
		if available < 1 {
			return apperror.New(apperror.CodeOutOfStock, "not enough stock")
		}
		cart.Items = append(cart.Items, model.CartItem{CartID: cart.ID, ProductID: product.ID, Name: product.Name, Slug: product.Slug, Qty: 1, Image: product.FirstImage(), Price: product.Price, CreatedAt: time.Now().UTC()})
		return nil
	})
}

func (s *Service) availableStock(ctx context.Context, product *model.Product) int {
	if s.stockStore == nil || !s.stockStore.Enabled() {
		return product.Stock
	}

	stock, ok, err := s.stockStore.Available(ctx, product.ID)
	if !ok && err == nil && s.inventory != nil {
		if syncErr := s.inventory.Sync(ctx); syncErr == nil {
			stock, ok, err = s.stockStore.Available(ctx, product.ID)
		}
	}
	if err != nil || !ok {
		return product.Stock
	}
	return stock
}

func (s *Service) RemoveItem(ctx context.Context, sessionCartID string, userID *string, productID string) (*model.Cart, error) {
	return s.current(ctx, sessionCartID, userID, false, func(cart *model.Cart) error {
		for i := range cart.Items {
			if cart.Items[i].ProductID != productID {
				continue
			}
			if cart.Items[i].Qty <= 1 {
				cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			} else {
				cart.Items[i].Qty--
			}
			break
		}
		return nil
	})
}

func (s *Service) ClearCart(ctx context.Context, sessionCartID string, userID *string) (*model.Cart, error) {
	return s.current(ctx, sessionCartID, userID, false, func(cart *model.Cart) error { cart.Items = []model.CartItem{}; return nil })
}

func (s *Service) current(ctx context.Context, sessionCartID string, userID *string, create bool, change func(*model.Cart) error) (*model.Cart, error) {
	cart, err := s.carts.WithCurrent(ctx, sessionCartID, userID, create, change)
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to access cart", err)
	}
	return cart, nil
}
