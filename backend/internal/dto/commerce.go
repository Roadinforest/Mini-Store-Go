package dto

import (
	"time"

	"mini-store-go/backend/internal/domain/valueobject"
)

type CartItemResponse struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Qty       int     `json:"qty"`
	Image     string  `json:"image"`
	Price     float64 `json:"price"`
}

type CartResponse struct {
	ID            string             `json:"id,omitempty"`
	UserID        *string            `json:"user_id,omitempty"`
	SessionCartID string             `json:"session_cart_id"`
	Items         []CartItemResponse `json:"items"`
	ItemsPrice    float64            `json:"items_price"`
	ShippingPrice float64            `json:"shipping_price"`
	TaxPrice      float64            `json:"tax_price"`
	TotalPrice    float64            `json:"total_price"`
	CreatedAt     *time.Time         `json:"created_at,omitempty"`
}

type OrderItemResponse struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Qty       int     `json:"qty"`
	Image     string  `json:"image"`
	Price     float64 `json:"price"`
}

type OrderResponse struct {
	ID              string                      `json:"id"`
	UserID          string                      `json:"user_id"`
	ShippingAddress valueobject.ShippingAddress `json:"shipping_address"`
	PaymentMethod   string                      `json:"payment_method"`
	ItemsPrice      float64                     `json:"items_price"`
	ShippingPrice   float64                     `json:"shipping_price"`
	TaxPrice        float64                     `json:"tax_price"`
	TotalPrice      float64                     `json:"total_price"`
	IsPaid          bool                        `json:"is_paid"`
	PaidAt          *time.Time                  `json:"paid_at,omitempty"`
	IsDelivered     bool                        `json:"is_delivered"`
	DeliveredAt     *time.Time                  `json:"delivered_at,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	OrderItems      []OrderItemResponse         `json:"order_items"`
	User            *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user,omitempty"`
}
