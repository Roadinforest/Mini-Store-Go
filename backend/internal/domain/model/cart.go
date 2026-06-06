package model

import (
	"time"

	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/valueobject"
)

type Cart struct {
	ID            string                                   `gorm:"column:id;primaryKey;type:uuid"`
	UserID        *string                                  `gorm:"column:user_id;type:uuid;index"`
	SessionCartID string                                   `gorm:"column:session_cart_id;type:text;index;not null"`
	Items         valueobject.JSON[[]valueobject.CartItem] `gorm:"column:items;type:jsonb"`
	ItemsPrice    decimal.Decimal                          `gorm:"column:items_price;type:numeric(12,2);not null"`
	TotalPrice    decimal.Decimal                          `gorm:"column:total_price;type:numeric(12,2);not null"`
	ShippingPrice decimal.Decimal                          `gorm:"column:shipping_price;type:numeric(12,2);not null"`
	TaxPrice      decimal.Decimal                          `gorm:"column:tax_price;type:numeric(12,2);not null"`
	CreatedAt     time.Time                                `gorm:"column:created_at;autoCreateTime"`

	User *User `gorm:"foreignKey:UserID;references:ID"`
}

func (Cart) TableName() string {
	return "carts"
}
