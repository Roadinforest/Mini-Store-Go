package orderservice

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

	var order *model.Order
	reserved := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedUser model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&lockedUser, "id = ?", userID).Error; err != nil {
			return err
		}
		cart, err := s.loadCheckoutCart(tx, userID, sessionCartID)
		if err != nil {
			return err
		}
		if len(cart.Items) == 0 {
			return apperror.New(apperror.CodeBadRequest, "cart is empty")
		}
		if !user.Address.Valid || !isCompleteAddress(user.Address.Data) {
			return apperror.New(apperror.CodeBadRequest, "shipping address is required")
		}
		if user.PaymentMethod == nil || *user.PaymentMethod == "" {
			return apperror.New(apperror.CodeBadRequest, "payment method is required")
		}
		amounts := cart.Amounts()
		order = &model.Order{
			ID: uuid.NewString(), UserID: user.ID, ShippingAddress: user.Address,
			PaymentMethod: *user.PaymentMethod,
			ShippingPrice: amounts.ShippingPrice, TaxPrice: amounts.TaxPrice,
			CreatedAt: time.Now().UTC(),
		}
		order.OrderItems = toOrderItems(order.ID, cart.Items)
		reserved, err = s.reserveStock(ctx, order.ID, cart.Items)
		if err != nil {
			return err
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		return tx.Where(`"cartId" = ?`, cart.ID).Delete(&model.CartItem{}).Error
	})
	if err != nil {
		if reserved {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = s.stockStore.Release(cleanupCtx, order.ID)
		}
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return nil, appErr
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, "id = ?", orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.New(apperror.CodeNotFound, "order not found")
			}
			return err
		}
		if order.IsPaid() {
			return nil
		}
		if err := tx.Where(`"orderId" = ?`, orderID).Order(`"productId"`).Find(&order.OrderItems).Error; err != nil {
			return err
		}

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
		order.PaidAt = &now
		return tx.Model(&order).Updates(map[string]interface{}{"paidAt": now}).Error
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
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, "id = ?", orderID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.New(apperror.CodeNotFound, "order not found")
			}
			return err
		}
		if !order.IsPaid() {
			return apperror.New(apperror.CodeBadRequest, "order is not paid")
		}
		if order.IsDelivered() {
			return nil
		}
		return tx.Model(&order).Update("deliveredAt", time.Now().UTC()).Error
	})
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to mark order delivered", err)
	}
	return s.GetByID(ctx, orderID)
}

// The cart is consumed under the same row lock and transaction as order creation.
func (s *Service) loadCheckoutCart(tx *gorm.DB, userID, sessionCartID string) (*model.Cart, error) {
	var cart model.Cart
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(`"userId" = ?`, userID).First(&cart).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(`"sessionCartId" = ? AND ("userId" IS NULL OR "userId" = ?)`, sessionCartID, userID).First(&cart).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.New(apperror.CodeBadRequest, "cart is empty")
	}
	if err != nil {
		return nil, err
	}
	if cart.UserID == nil {
		cart.UserID = &userID
		if err := tx.Model(&cart).Update("userId", userID).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Where(`"cartId" = ?`, cart.ID).Order(`"createdAt" ASC`).Order(`"productId" ASC`).Find(&cart.Items).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func toOrderItems(orderID string, items []model.CartItem) []model.OrderItem {
	orderItems := make([]model.OrderItem, 0, len(items))
	for _, item := range items {
		orderItems = append(orderItems, model.OrderItem{
			OrderID:   orderID,
			ProductID: item.ProductID,
			Qty:       item.Qty,
			Price:     item.Price,
			Name:      item.Name,
			Slug:      item.Slug,
			Image:     item.Image,
		})
	}
	sort.Slice(orderItems, func(i, j int) bool { return orderItems[i].ProductID < orderItems[j].ProductID })
	return orderItems
}

func (s *Service) reserveStock(ctx context.Context, orderID string, items []model.CartItem) (bool, error) {
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
		if order.IsPaid() {
			_ = s.stockStore.Confirm(ctx, orderID)
			continue
		}
		_ = s.stockStore.Release(ctx, orderID)
	}
	return nil
}

func toStockItems(items []model.CartItem) []rediscache.StockItem {
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
