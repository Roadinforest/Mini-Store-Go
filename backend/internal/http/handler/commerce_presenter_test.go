package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/domain/model"
)

func TestCommerceResponseContractAfterNormalization(t *testing.T) {
	cart := toCartResponse(&model.Cart{SessionCartID: "guest", Items: []model.CartItem{{ProductID: "p1", Qty: 2, Price: decimal.RequireFromString("0.10")}}})
	now := time.Now().UTC()
	order := toOrderResponse(&model.Order{PaidAt: &now, ShippingPrice: decimal.NewFromInt(7), TaxPrice: decimal.NewFromInt(1), OrderItems: []model.OrderItem{{ProductID: "p1", Qty: 2, Price: decimal.NewFromInt(10)}}})
	for _, tc := range []struct {
		response interface{}
		total    float64
		paid     bool
	}{{cart, 10.23, false}, {order, 28, true}} {
		encoded, err := json.Marshal(tc.response)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(encoded, &payload); err != nil {
			t.Fatal(err)
		}
		if payload["total_price"] != tc.total {
			t.Fatalf("numeric response total: %s", encoded)
		}
		for _, key := range []string{"items_price", "shipping_price", "tax_price", "total_price"} {
			if _, ok := payload[key].(float64); !ok {
				t.Fatalf("%s must remain numeric: %s", key, encoded)
			}
		}
		if _, exists := payload["is_paid"]; exists && payload["is_paid"] != tc.paid {
			t.Fatalf("computed payment status: %s", encoded)
		}
	}
	for _, response := range []interface{}{toCartResponse(&model.Cart{}), toOrderResponse(&model.Order{}), toProductResponse(&model.Product{})} {
		encoded, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(encoded, &payload); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"items", "order_items", "images"} {
			if value, exists := payload[key]; exists {
				if list, ok := value.([]interface{}); !ok || len(list) != 0 {
					t.Fatalf("empty %s must be []: %s", key, encoded)
				}
			}
		}
	}
}
