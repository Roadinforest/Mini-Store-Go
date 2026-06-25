package orderservice

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
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/infra/rediscache"
	"mini-store-go/backend/internal/repository"
)

const (
	reservationTTL          = 15 * time.Minute
	reservationCleanupLimit = 100
)

type Service struct {
	db         *gorm.DB
	orders     repository.OrderRepository
	carts      repository.CartRepository
	users      repository.UserRepository
	products   repository.ProductRepository
	stockStore *rediscache.StockStore
}

func NewService(db *gorm.DB, orders repository.OrderRepository, carts repository.CartRepository, users repository.UserRepository, products repository.ProductRepository, stockStore *rediscache.StockStore) *Service {
	return &Service{
		db:         db,
		orders:     orders,
		carts:      carts,
		users:      users,
		products:   products,
		stockStore: stockStore,
	}
}

func (s *Service) Create(ctx context.Context, userID string, sessionCartID string) (*model.Order, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeUnauthorized, "user not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load user", err)
	}

	cart, err := s.loadCheckoutCart(ctx, userID, sessionCartID)
	if err != nil {
		return nil, err
	}
	if len(cart.Items.Data) == 0 {
		return nil, apperror.New(apperror.CodeBadRequest, "cart is empty")
	}
	if !user.Address.Valid || !isCompleteAddress(user.Address.Data) {
		return nil, apperror.New(apperror.CodeBadRequest, "shipping address is required")
	}
	if user.PaymentMethod == nil || *user.PaymentMethod == "" {
		return nil, apperror.New(apperror.CodeBadRequest, "payment method is required")
	}

	order := &model.Order{
		ID:              uuid.NewString(),
		UserID:          user.ID,
		ShippingAddress: user.Address,
		PaymentMethod:   *user.PaymentMethod,
		PaymentResult:   valueobject.JSON[valueobject.PaymentResult]{},
		ItemsPrice:      cart.ItemsPrice,
		ShippingPrice:   cart.ShippingPrice,
		TaxPrice:        cart.TaxPrice,
		TotalPrice:      cart.TotalPrice,
		IsPaid:          false,
		IsDelivered:     false,
		CreatedAt:       time.Now().UTC(),
		OrderItems:      nil,
	}
	order.OrderItems = toOrderItems(order.ID, cart.Items.Data)

	reserved, err := s.reserveStock(ctx, order.ID, cart.Items.Data)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		cart.Items = valueobject.NewJSONArray([]valueobject.CartItem{})
		cart.ItemsPrice = decimal.Zero
		cart.ShippingPrice = decimal.Zero
		cart.TaxPrice = decimal.Zero
		cart.TotalPrice = decimal.Zero

		if err := tx.Save(cart).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if reserved {
			_ = s.stockStore.Release(ctx, order.ID)
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create order", err)
	}

	return s.GetByID(ctx, order.ID)
}

func (s *Service) GetByID(ctx context.Context, orderID string) (*model.Order, error) {
	order, err := s.orders.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "order not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load order", err)
	}
	return order, nil
}

func (s *Service) ListMine(ctx context.Context, userID string, page dto.PageParams) ([]model.Order, dto.PageMeta, error) {
	page = page.Normalize(20)
	items, total, err := s.orders.ListByUserID(ctx, userID, page)
	if err != nil {
		return nil, dto.PageMeta{}, apperror.Wrap(apperror.CodeInternal, "failed to list orders", err)
	}
	return items, dto.NewPageMeta(page.Page, page.Limit, total), nil
}

func (s *Service) List(ctx context.Context, page dto.PageParams) ([]model.Order, dto.PageMeta, error) {
	page = page.Normalize(20)
	items, total, err := s.orders.List(ctx, page)
	if err != nil {
		return nil, dto.PageMeta{}, apperror.Wrap(apperror.CodeInternal, "failed to list orders", err)
	}
	return items, dto.NewPageMeta(page.Page, page.Limit, total), nil
}

func (s *Service) MarkPaid(ctx context.Context, orderID string) (*model.Order, error) {
	order, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.IsPaid {
		return order, nil
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range order.OrderItems {
			result := tx.Model(&model.Product{}).
				Where("id = ? AND stock >= ?", item.ProductID, item.Qty).
				Update("stock", gorm.Expr("stock - ?", item.Qty))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return apperror.New(apperror.CodeOutOfStock, "not enough stock")
			}
		}

		now := time.Now().UTC()
		order.IsPaid = true
		order.PaidAt = &now
		return tx.Save(order).Error
	})
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			_ = s.releaseStockReservation(ctx, orderID)
			return nil, appErr
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to mark order paid", err)
	}

	_ = s.confirmStockReservation(ctx, orderID)

	return s.GetByID(ctx, orderID)
}

func (s *Service) MarkDelivered(ctx context.Context, orderID string) (*model.Order, error) {
	order, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !order.IsPaid {
		return nil, apperror.New(apperror.CodeBadRequest, "order is not paid")
	}
	if order.IsDelivered {
		return order, nil
	}

	now := time.Now().UTC()
	order.IsDelivered = true
	order.DeliveredAt = &now
	if err := s.orders.Update(ctx, order); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to mark order delivered", err)
	}
	return s.GetByID(ctx, orderID)
}

func (s *Service) loadCheckoutCart(ctx context.Context, userID, sessionCartID string) (*model.Cart, error) {
	cart, err := s.carts.GetByUserID(ctx, userID)
	if err == nil {
		if cart.SessionCartID == "" {
			cart.SessionCartID = sessionCartID
			_ = s.carts.Update(ctx, cart)
		}
		return cart, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load cart", err)
	}

	sessionCart, sessionErr := s.carts.GetBySessionCartID(ctx, sessionCartID)
	if sessionErr != nil {
		if errors.Is(sessionErr, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeBadRequest, "cart is empty")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load cart", sessionErr)
	}
	sessionCart.UserID = &userID
	if saveErr := s.carts.Update(ctx, sessionCart); saveErr != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to assign cart", saveErr)
	}
	return sessionCart, nil
}

func toOrderItems(orderID string, items []valueobject.CartItem) []model.OrderItem {
	orderItems := make([]model.OrderItem, 0, len(items))
	for _, item := range items {
		price, _ := decimal.NewFromString(item.Price)
		orderItems = append(orderItems, model.OrderItem{
			OrderID:   orderID,
			ProductID: item.ProductID,
			Qty:       item.Qty,
			Price:     price,
			Name:      item.Name,
			Slug:      item.Slug,
			Image:     item.Image,
		})
	}
	return orderItems
}

func (s *Service) reserveStock(ctx context.Context, orderID string, items []valueobject.CartItem) (bool, error) {
	if s.stockStore == nil || !s.stockStore.Enabled() {
		return false, nil
	}

	stockItems := toStockItems(items)
	if len(stockItems) == 0 {
		return false, nil
	}

	_ = s.releaseExpiredReservations(ctx)

	if err := s.primeStockCache(ctx, stockItems); err != nil {
		return false, nil
	}

	err := s.stockStore.Reserve(ctx, orderID, stockItems, reservationTTL)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, rediscache.ErrStockCacheMiss) {
		if primeErr := s.primeStockCache(ctx, stockItems); primeErr != nil {
			return false, nil
		}
		if retryErr := s.stockStore.Reserve(ctx, orderID, stockItems, reservationTTL); retryErr == nil {
			return true, nil
		} else {
			err = retryErr
		}
	}
	if errors.Is(err, rediscache.ErrInsufficient) {
		return false, apperror.New(apperror.CodeOutOfStock, "not enough stock")
	}

	return false, nil
}

func (s *Service) primeStockCache(ctx context.Context, items []rediscache.StockItem) error {
	stocks := make(map[string]int, len(items))
	for _, item := range items {
		if _, exists := stocks[item.ProductID]; exists {
			continue
		}
		product, err := s.products.GetByID(ctx, item.ProductID)
		if err != nil {
			return err
		}
		stocks[item.ProductID] = product.Stock
	}
	return s.stockStore.PrimeStocks(ctx, stocks)
}

func (s *Service) releaseStockReservation(ctx context.Context, orderID string) error {
	if s.stockStore == nil || !s.stockStore.Enabled() {
		return nil
	}
	return s.stockStore.Release(ctx, orderID)
}

func (s *Service) confirmStockReservation(ctx context.Context, orderID string) error {
	if s.stockStore == nil || !s.stockStore.Enabled() {
		return nil
	}
	return s.stockStore.Confirm(ctx, orderID)
}

func (s *Service) releaseExpiredReservations(ctx context.Context) error {
	if s.stockStore == nil || !s.stockStore.Enabled() {
		return nil
	}

	orderIDs, err := s.stockStore.ExpiredReservations(ctx, time.Now().UTC(), reservationCleanupLimit)
	if err != nil {
		return err
	}

	for _, orderID := range orderIDs {
		order, err := s.orders.GetByID(ctx, orderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = s.stockStore.Release(ctx, orderID)
			}
			continue
		}
		if order.IsPaid {
			_ = s.stockStore.Confirm(ctx, orderID)
			continue
		}
		_ = s.stockStore.Release(ctx, orderID)
	}
	return nil
}

func toStockItems(items []valueobject.CartItem) []rediscache.StockItem {
	merged := make(map[string]int, len(items))
	for _, item := range items {
		if item.Qty <= 0 {
			continue
		}
		merged[item.ProductID] += item.Qty
	}

	stockItems := make([]rediscache.StockItem, 0, len(merged))
	for productID, qty := range merged {
		stockItems = append(stockItems, rediscache.StockItem{
			ProductID: productID,
			Qty:       qty,
		})
	}
	return stockItems
}

func isCompleteAddress(address valueobject.ShippingAddress) bool {
	return address.FullName != "" &&
		address.StreetAddress != "" &&
		address.City != "" &&
		address.PostalCode != "" &&
		address.Country != ""
}
