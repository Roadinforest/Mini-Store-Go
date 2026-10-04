package valueobject

import (
	"github.com/shopspring/decimal"
	"testing"
)

func TestCartAmountsPolicyBoundaries(t *testing.T) {
	for _, tc := range []struct{ subtotal, shipping, tax, total string }{
		{"0", "0", "0", "0"},
		{"0.10", "10", "0.02", "10.12"},
		{"100", "10", "15", "125"},
		{"100.01", "0", "15", "115.01"},
		{"9999999999.99", "0", "1500000000", "11499999999.99"},
	} {
		t.Run(tc.subtotal, func(t *testing.T) {
			amounts := CartAmounts(decimal.RequireFromString(tc.subtotal))
			if !amounts.ShippingPrice.Equal(decimal.RequireFromString(tc.shipping)) || !amounts.TaxPrice.Equal(decimal.RequireFromString(tc.tax)) || !amounts.TotalPrice.Equal(decimal.RequireFromString(tc.total)) {
				t.Fatalf("amounts=%+v", amounts)
			}
		})
	}
}
