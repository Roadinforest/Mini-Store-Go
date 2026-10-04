package valueobject

import "github.com/shopspring/decimal"

type Amounts struct {
	ItemsPrice    decimal.Decimal
	ShippingPrice decimal.Decimal
	TaxPrice      decimal.Decimal
	TotalPrice    decimal.Decimal
}

// CartAmounts applies the current policy. Orders retain their historical charges.
func CartAmounts(subtotal decimal.Decimal) Amounts {
	subtotal = subtotal.Round(2)
	shipping := decimal.Zero
	if subtotal.IsPositive() && !subtotal.GreaterThan(decimal.NewFromInt(100)) {
		shipping = decimal.NewFromInt(10)
	}
	tax := subtotal.Mul(decimal.RequireFromString("0.15")).Round(2)
	return NewAmounts(subtotal, shipping, tax)
}

func NewAmounts(subtotal, shipping, tax decimal.Decimal) Amounts {
	return Amounts{ItemsPrice: subtotal, ShippingPrice: shipping, TaxPrice: tax, TotalPrice: subtotal.Add(shipping).Add(tax)}
}
