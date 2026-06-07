package dto

import "mini-store-go/backend/internal/domain/valueobject"

type CreateOrderInput struct {
	ShippingAddress *valueobject.ShippingAddress `json:"shipping_address,omitempty"`
	PaymentMethod   string                       `json:"payment_method,omitempty"`
}
