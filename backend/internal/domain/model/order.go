package model

import (
	"time"

	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/valueobject"
)

type Order struct {
	ID              string                                        `gorm:"column:id;primaryKey;type:uuid"`
	UserID          string                                        `gorm:"column:userId;type:uuid;index;not null"`
	ShippingAddress valueobject.JSON[valueobject.ShippingAddress] `gorm:"column:shippingAddress;type:jsonb;not null"`
	PaymentMethod   string                                        `gorm:"column:paymentMethod;type:text;not null"`
	PaymentResult   valueobject.JSON[valueobject.PaymentResult]   `gorm:"column:paymentResult;type:jsonb"`
	ShippingPrice   decimal.Decimal                               `gorm:"column:shippingPrice;type:numeric(12,2);not null"`
	TaxPrice        decimal.Decimal                               `gorm:"column:taxPrice;type:numeric(12,2);not null"`
	PaidAt          *time.Time                                    `gorm:"column:paidAt"`
	DeliveredAt     *time.Time                                    `gorm:"column:deliveredAt"`
	CreatedAt       time.Time                                     `gorm:"column:createdAt;autoCreateTime"`

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

func (o Order) Amounts() valueobject.Amounts {
	subtotal := decimal.Zero
	for _, item := range o.OrderItems {
		subtotal = subtotal.Add(item.Price.Mul(decimal.NewFromInt(int64(item.Qty))))
	}
	return valueobject.NewAmounts(subtotal, o.ShippingPrice, o.TaxPrice)
}
