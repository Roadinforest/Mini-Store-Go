package cartservice

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	"mini-store-go/backend/internal/repository"
)

const (
	taxRate          = 0.15
	shippingPrice    = 10
	freeShippingBar  = 100
	defaultZeroPrice = "0.00"
)

type Service struct {
	carts    repository.CartRepository
	products repository.ProductRepository
}

func NewService(carts repository.CartRepository, products repository.ProductRepository) *Service {
	return &Service{
		carts:    carts,
		products: products,
	}
}

func (s *Service) GetCurrentCart(ctx context.Context, sessionCartID string, userID *string) (*model.Cart, error) {
	cart, err := s.resolveCart(ctx, sessionCartID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.emptyCart(sessionCartID, userID), nil
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load cart", err)
	}
	return cart, nil
}

func (s *Service) AddItem(ctx context.Context, sessionCartID string, userID *string, productID string) (*model.Cart, error) {
	product, err := s.products.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "product not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load product", err)
	}

	cart, err := s.resolveCart(ctx, sessionCartID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load cart", err)
	}
	if cart == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		cart = s.emptyCart(sessionCartID, userID)
	}

	items := cloneItems(cart.Items.Data)
	index := -1
	for i := range items {
		if items[i].ProductID == productID {
			index = i
			break
		}
	}

	if index >= 0 {
		nextQty := items[index].Qty + 1
		if product.Stock < nextQty {
			return nil, apperror.New(apperror.CodeOutOfStock, "not enough stock")
		}
		items[index].Qty = nextQty
	} else {
		if product.Stock < 1 {
			return nil, apperror.New(apperror.CodeOutOfStock, "not enough stock")
		}
		items = append(items, valueobject.CartItem{
			ProductID: product.ID,
			Name:      product.Name,
			Slug:      product.Slug,
			Qty:       1,
			Image:     firstImage(product.Images),
			Price:     product.Price.StringFixed(2),
		})
	}

	s.applyCartItems(cart, items)
	if err := s.saveCart(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *Service) RemoveItem(ctx context.Context, sessionCartID string, userID *string, productID string) (*model.Cart, error) {
	cart, err := s.resolveCart(ctx, sessionCartID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.emptyCart(sessionCartID, userID), nil
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load cart", err)
	}

	items := cloneItems(cart.Items.Data)
	index := -1
	for i := range items {
		if items[i].ProductID == productID {
			index = i
			break
		}
	}
	if index < 0 {
		return cart, nil
	}

	if items[index].Qty <= 1 {
		items = append(items[:index], items[index+1:]...)
	} else {
		items[index].Qty--
	}

	s.applyCartItems(cart, items)
	if err := s.saveCart(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *Service) ClearCart(ctx context.Context, sessionCartID string, userID *string) (*model.Cart, error) {
	cart, err := s.resolveCart(ctx, sessionCartID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.emptyCart(sessionCartID, userID), nil
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load cart", err)
	}

	s.applyCartItems(cart, nil)
	if err := s.saveCart(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *Service) resolveCart(ctx context.Context, sessionCartID string, userID *string) (*model.Cart, error) {
	if userID != nil && *userID != "" {
		cart, err := s.carts.GetByUserID(ctx, *userID)
		if err == nil {
			if cart.SessionCartID != sessionCartID {
				cart.SessionCartID = sessionCartID
				_ = s.carts.Update(ctx, cart)
			}
			return cart, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		sessionCart, sessionErr := s.carts.GetBySessionCartID(ctx, sessionCartID)
		if sessionErr == nil {
			sessionCart.UserID = userID
			if err := s.carts.Update(ctx, sessionCart); err != nil {
				return nil, err
			}
			return sessionCart, nil
		}
		if !errors.Is(sessionErr, gorm.ErrRecordNotFound) {
			return nil, sessionErr
		}
	}

	return s.carts.GetBySessionCartID(ctx, sessionCartID)
}

func (s *Service) saveCart(ctx context.Context, cart *model.Cart) error {
	if cart.ID == "" {
		cart.ID = uuid.NewString()
		cart.CreatedAt = time.Now().UTC()
		if err := s.carts.Create(ctx, cart); err != nil {
			return apperror.Wrap(apperror.CodeInternal, "failed to create cart", err)
		}
		return nil
	}

	if err := s.carts.Update(ctx, cart); err != nil {
		return apperror.Wrap(apperror.CodeInternal, "failed to update cart", err)
	}
	return nil
}

func (s *Service) emptyCart(sessionCartID string, userID *string) *model.Cart {
	return &model.Cart{
		UserID:        userID,
		SessionCartID: sessionCartID,
		Items:         valueobject.NewJSONArray([]valueobject.CartItem{}),
		ItemsPrice:    decimal.RequireFromString(defaultZeroPrice),
		ShippingPrice: decimal.RequireFromString(defaultZeroPrice),
		TaxPrice:      decimal.RequireFromString(defaultZeroPrice),
		TotalPrice:    decimal.RequireFromString(defaultZeroPrice),
	}
}

func (s *Service) applyCartItems(cart *model.Cart, items []valueobject.CartItem) {
	if items == nil {
		items = []valueobject.CartItem{}
	}
	cart.Items = valueobject.NewJSONArray(items)

	itemsPrice := 0.0
	for _, item := range items {
		price, _ := decimal.NewFromString(item.Price)
		itemsPrice += price.InexactFloat64() * float64(item.Qty)
	}

	itemsPriceDec := round2Decimal(itemsPrice)
	shippingDec := decimal.Zero
	if itemsPriceDec.GreaterThan(decimal.NewFromFloat(freeShippingBar)) {
		shippingDec = decimal.Zero
	} else if itemsPriceDec.GreaterThan(decimal.Zero) {
		shippingDec = decimal.NewFromFloat(shippingPrice)
	}
	taxDec := round2Decimal(itemsPriceDec.InexactFloat64() * taxRate)
	totalDec := round2Decimal(itemsPriceDec.InexactFloat64() + shippingDec.InexactFloat64() + taxDec.InexactFloat64())

	cart.ItemsPrice = itemsPriceDec
	cart.ShippingPrice = shippingDec
	cart.TaxPrice = taxDec
	cart.TotalPrice = totalDec
}

func cloneItems(items []valueobject.CartItem) []valueobject.CartItem {
	if len(items) == 0 {
		return []valueobject.CartItem{}
	}
	out := make([]valueobject.CartItem, len(items))
	copy(out, items)
	return out
}

func firstImage(images []string) string {
	if len(images) == 0 {
		return ""
	}
	return images[0]
}

func round2Decimal(value float64) decimal.Decimal {
	return decimal.NewFromFloat(value).Round(2)
}
