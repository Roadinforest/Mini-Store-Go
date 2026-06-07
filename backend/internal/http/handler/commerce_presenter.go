package handler

import (
	"strconv"
	"time"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	"mini-store-go/backend/internal/dto"
)

type cartItemResponse struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Qty       int     `json:"qty"`
	Image     string  `json:"image"`
	Price     float64 `json:"price"`
}

type cartResponse struct {
	ID            string             `json:"id,omitempty"`
	UserID        *string            `json:"user_id,omitempty"`
	SessionCartID string             `json:"session_cart_id"`
	Items         []cartItemResponse `json:"items"`
	ItemsPrice    float64            `json:"items_price"`
	ShippingPrice float64            `json:"shipping_price"`
	TaxPrice      float64            `json:"tax_price"`
	TotalPrice    float64            `json:"total_price"`
	CreatedAt     *time.Time         `json:"created_at,omitempty"`
}

type orderItemResponse struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Qty       int     `json:"qty"`
	Image     string  `json:"image"`
	Price     float64 `json:"price"`
}

type orderResponse struct {
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
	OrderItems      []orderItemResponse         `json:"order_items"`
	User            *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user,omitempty"`
}

func toCartResponse(cart *model.Cart) cartResponse {
	items := make([]cartItemResponse, 0, len(cart.Items.Data))
	for _, item := range cart.Items.Data {
		price, _ := parseFloat(item.Price)
		items = append(items, cartItemResponse{
			ProductID: item.ProductID,
			Name:      item.Name,
			Slug:      item.Slug,
			Qty:       item.Qty,
			Image:     item.Image,
			Price:     price,
		})
	}

	var createdAt *time.Time
	if !cart.CreatedAt.IsZero() {
		createdAt = &cart.CreatedAt
	}

	return cartResponse{
		ID:            cart.ID,
		UserID:        cart.UserID,
		SessionCartID: cart.SessionCartID,
		Items:         items,
		ItemsPrice:    cart.ItemsPrice.InexactFloat64(),
		ShippingPrice: cart.ShippingPrice.InexactFloat64(),
		TaxPrice:      cart.TaxPrice.InexactFloat64(),
		TotalPrice:    cart.TotalPrice.InexactFloat64(),
		CreatedAt:     createdAt,
	}
}

func toOrderResponse(order *model.Order) orderResponse {
	address := valueobject.ShippingAddress{}
	if order.ShippingAddress.Valid {
		address = order.ShippingAddress.Data
	}

	items := make([]orderItemResponse, 0, len(order.OrderItems))
	for _, item := range order.OrderItems {
		items = append(items, orderItemResponse{
			ProductID: item.ProductID,
			Name:      item.Name,
			Slug:      item.Slug,
			Qty:       item.Qty,
			Image:     item.Image,
			Price:     item.Price.InexactFloat64(),
		})
	}

	resp := orderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		ShippingAddress: address,
		PaymentMethod:   order.PaymentMethod,
		ItemsPrice:      order.ItemsPrice.InexactFloat64(),
		ShippingPrice:   order.ShippingPrice.InexactFloat64(),
		TaxPrice:        order.TaxPrice.InexactFloat64(),
		TotalPrice:      order.TotalPrice.InexactFloat64(),
		IsPaid:          order.IsPaid,
		PaidAt:          order.PaidAt,
		IsDelivered:     order.IsDelivered,
		DeliveredAt:     order.DeliveredAt,
		CreatedAt:       order.CreatedAt,
		OrderItems:      items,
	}

	if order.User.ID != "" {
		resp.User = &struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			ID:    order.User.ID,
			Name:  order.User.Name,
			Email: order.User.Email,
		}
	}

	return resp
}

func toOrderResponses(orders []model.Order) []orderResponse {
	items := make([]orderResponse, 0, len(orders))
	for i := range orders {
		items = append(items, toOrderResponse(&orders[i]))
	}
	return items
}

func toPagedOrders(orders []model.Order, meta dto.PageMeta) dto.Paged[orderResponse] {
	return dto.Paged[orderResponse]{
		Items: toOrderResponses(orders),
		Meta:  meta,
	}
}

func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
