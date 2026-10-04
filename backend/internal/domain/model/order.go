package model

import (
	"time"

	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/valueobject"
)

// Legacy adjustments preserve unexplained differences in historical invoices;
// new orders leave them zero. They are not a claim that an extra fee was charged.
type Order struct {
	ID                    string                                        `gorm:"column:id;primaryKey;type:uuid"`
	UserID                string                                        `gorm:"column:userId;type:uuid;index;not null"`
	ShippingAddress       valueobject.JSON[valueobject.ShippingAddress] `gorm:"column:shippingAddress;type:jsonb;not null"`
	PaymentMethod         string                                        `gorm:"column:paymentMethod;type:text;not null"`
	PaymentResult         valueobject.JSON[valueobject.PaymentResult]   `gorm:"column:paymentResult;type:jsonb"`
	ShippingPrice         decimal.Decimal                               `gorm:"column:shippingPrice;type:numeric(12,2);not null"`
	TaxPrice              decimal.Decimal                               `gorm:"column:taxPrice;type:numeric(12,2);not null"`
	LegacyItemsAdjustment decimal.Decimal                               `gorm:"column:legacyItemsAdjustment;type:numeric(12,2);not null;default:0"`
	LegacyTotalAdjustment decimal.Decimal                               `gorm:"column:legacyTotalAdjustment;type:numeric(12,2);not null;default:0"`
	PaidAt                *time.Time                                    `gorm:"column:paidAt"`
	DeliveredAt           *time.Time                                    `gorm:"column:deliveredAt"`
	CreatedAt             time.Time                                     `gorm:"column:createdAt;autoCreateTime"`
	ExpiresAt             *time.Time                                    `gorm:"column:expiresAt;index"`
	ExpiredAt             *time.Time                                    `gorm:"column:expiredAt"`

	User       User        `gorm:"foreignKey:UserID;references:ID"`
	OrderItems []OrderItem `gorm:"foreignKey:OrderID;references:ID"`
}

func (Order) TableName() string {
	return "Order"
}

type OrderItem struct {
	OrderID   string          `gorm:"column:orderId;primaryKey;type:uuid"`
	ProductID string          `gorm:"column:productId;primaryKey;type:text"`
	Qty       int             `gorm:"column:qty;not null"`
	Price     decimal.Decimal `gorm:"column:price;type:numeric(12,2);not null"`
	Name      string          `gorm:"column:name;type:text;not null"`
	Slug      string          `gorm:"column:slug;type:text;not null"`
	Image     string          `gorm:"column:image;type:text;not null"`

	Order   Order   `gorm:"foreignKey:OrderID;references:ID"`
	Product Product `gorm:"foreignKey:ProductID;references:ID"`
}

func (OrderItem) TableName() string {
	return "OrderItem"
}

func (o Order) IsPaid() bool      { return o.PaidAt != nil }
func (o Order) IsDelivered() bool { return o.DeliveredAt != nil }

const OrderReservationTTL = 15 * time.Minute

func (o Order) Deadline() time.Time {
	if o.ExpiresAt != nil {
		return *o.ExpiresAt
	}
	return o.CreatedAt.Add(OrderReservationTTL)
}

func (o Order) Status(now time.Time) string {
	if o.IsPaid() {
		return "PAID"
	}
	if o.ExpiredAt != nil || !now.Before(o.Deadline()) {
		return "EXPIRED"
	}
	return "UNPAID"
}

func (o Order) Amounts() valueobject.Amounts {
	subtotal := decimal.Zero
	for _, item := range o.OrderItems {
		subtotal = subtotal.Add(item.Price.Mul(decimal.NewFromInt(int64(item.Qty))))
	}
	amounts := valueobject.NewAmounts(subtotal.Add(o.LegacyItemsAdjustment), o.ShippingPrice, o.TaxPrice)
	amounts.TotalPrice = amounts.TotalPrice.Add(o.LegacyTotalAdjustment)
	return amounts
}
