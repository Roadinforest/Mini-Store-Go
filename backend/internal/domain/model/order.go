package model

import (
	"time"

	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/valueobject"
)

type Order struct {
	ID              string                                        `gorm:"column:id;primaryKey;type:uuid"`
	UserID          string                                        `gorm:"column:user_id;type:uuid;index;not null"`
	ShippingAddress valueobject.JSON[valueobject.ShippingAddress] `gorm:"column:shipping_address;type:jsonb;not null"`
	PaymentMethod   string                                        `gorm:"column:payment_method;type:text;not null"`
	PaymentResult   valueobject.JSON[valueobject.PaymentResult]   `gorm:"column:payment_result;type:jsonb"`
	ItemsPrice      decimal.Decimal                               `gorm:"column:items_price;type:numeric(12,2);not null"`
	ShippingPrice   decimal.Decimal                               `gorm:"column:shipping_price;type:numeric(12,2);not null"`
	TaxPrice        decimal.Decimal                               `gorm:"column:tax_price;type:numeric(12,2);not null"`
	TotalPrice      decimal.Decimal                               `gorm:"column:total_price;type:numeric(12,2);not null"`
	IsPaid          bool                                          `gorm:"column:is_paid;not null;default:false"`
	PaidAt          *time.Time                                    `gorm:"column:paid_at"`
	IsDelivered     bool                                          `gorm:"column:is_delivered;not null;default:false"`
	DeliveredAt     *time.Time                                    `gorm:"column:delivered_at"`
	CreatedAt       time.Time                                     `gorm:"column:created_at;autoCreateTime"`

	User       User        `gorm:"foreignKey:UserID;references:ID"`
	OrderItems []OrderItem `gorm:"foreignKey:OrderID;references:ID"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	OrderID   string          `gorm:"column:order_id;primaryKey;type:uuid"`
	ProductID string          `gorm:"column:product_id;primaryKey;type:text"`
	Qty       int             `gorm:"column:qty;not null"`
	Price     decimal.Decimal `gorm:"column:price;type:numeric(12,2);not null"`
	Name      string          `gorm:"column:name;type:text;not null"`
	Slug      string          `gorm:"column:slug;type:text;not null"`
	Image     string          `gorm:"column:image;type:text;not null"`

	Order   Order   `gorm:"foreignKey:OrderID;references:ID"`
	Product Product `gorm:"foreignKey:ProductID;references:ID"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
