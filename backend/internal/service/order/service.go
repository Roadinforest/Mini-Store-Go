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
	"mini-store-go/backend/internal/repository"
)

type Service struct {
	db       *gorm.DB
	orders   repository.OrderRepository
	carts    repository.CartRepository
	users    repository.UserRepository
	products repository.ProductRepository
}

func NewService(db *gorm.DB, orders repository.OrderRepository, carts repository.CartRepository, users repository.UserRepository, products repository.ProductRepository) *Service {
	return &Service{
		db:       db,
		orders:   orders,
		carts:    carts,
		users:    users,
		products: products,
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
			return nil, appErr
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to mark order paid", err)
	}

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

func isCompleteAddress(address valueobject.ShippingAddress) bool {
	return address.FullName != "" &&
		address.StreetAddress != "" &&
		address.City != "" &&
		address.PostalCode != "" &&
		address.Country != ""
}
