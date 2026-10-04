package handler

import (
	"time"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	"mini-store-go/backend/internal/dto"
)

func toCartResponse(cart *model.Cart) dto.CartResponse {
	amounts := cart.Amounts()
	items := make([]dto.CartItemResponse, 0, len(cart.Items))
	for _, item := range cart.Items {
		price := item.Price.InexactFloat64()
		items = append(items, dto.CartItemResponse{
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

	return dto.CartResponse{
		ID:            cart.ID,
		UserID:        cart.UserID,
		SessionCartID: cart.SessionCartID,
		Items:         items,
		ItemsPrice:    amounts.ItemsPrice.InexactFloat64(),
		ShippingPrice: amounts.ShippingPrice.InexactFloat64(),
		TaxPrice:      amounts.TaxPrice.InexactFloat64(),
		TotalPrice:    amounts.TotalPrice.InexactFloat64(),
		CreatedAt:     createdAt,
	}
}

func toOrderResponse(order *model.Order) dto.OrderResponse {
	amounts := order.Amounts()
	address := valueobject.ShippingAddress{}
	if order.ShippingAddress.Valid {
		address = order.ShippingAddress.Data
	}

	items := make([]dto.OrderItemResponse, 0, len(order.OrderItems))
	for _, item := range order.OrderItems {
		items = append(items, dto.OrderItemResponse{
			ProductID: item.ProductID,
			Name:      item.Name,
			Slug:      item.Slug,
			Qty:       item.Qty,
			Image:     item.Image,
			Price:     item.Price.InexactFloat64(),
		})
	}

	resp := dto.OrderResponse{
		ID:              order.ID,
		UserID:          order.UserID,
		ShippingAddress: address,
		PaymentMethod:   order.PaymentMethod,
		ItemsPrice:      amounts.ItemsPrice.InexactFloat64(),
		ShippingPrice:   amounts.ShippingPrice.InexactFloat64(),
		TaxPrice:        amounts.TaxPrice.InexactFloat64(),
		TotalPrice:      amounts.TotalPrice.InexactFloat64(),
		IsPaid:          order.IsPaid(),
		PaidAt:          order.PaidAt,
		IsDelivered:     order.IsDelivered(),
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

func toOrderResponses(orders []model.Order) []dto.OrderResponse {
	items := make([]dto.OrderResponse, 0, len(orders))
	for i := range orders {
		items = append(items, toOrderResponse(&orders[i]))
	}
	return items
}

func toPagedOrders(orders []model.Order, meta dto.PageMeta) dto.Paged[dto.OrderResponse] {
	return dto.Paged[dto.OrderResponse]{
		Items: toOrderResponses(orders),
		Meta:  meta,
	}
}
