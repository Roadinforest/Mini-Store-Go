package model

import (
	"time"

	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/valueobject"
)

type Cart struct {
	ID            string                                      `gorm:"column:id;primaryKey;type:uuid"`
	UserID        *string                                     `gorm:"column:userId;type:uuid;index"`
	SessionCartID string                                      `gorm:"column:sessionCartId;type:text;index;not null"`
	Items         valueobject.JSONArray[valueobject.CartItem] `gorm:"column:items;type:json[]"`
	ItemsPrice    decimal.Decimal                             `gorm:"column:itemsPrice;type:numeric(12,2);not null"`
	TotalPrice    decimal.Decimal                             `gorm:"column:totalPrice;type:numeric(12,2);not null"`
	ShippingPrice decimal.Decimal                             `gorm:"column:shippingPrice;type:numeric(12,2);not null"`
	TaxPrice      decimal.Decimal                             `gorm:"column:taxPrice;type:numeric(12,2);not null"`
	CreatedAt     time.Time                                   `gorm:"column:createdAt;autoCreateTime"`

	User *User `gorm:"foreignKey:UserID;references:ID"`
}

func (Cart) TableName() string {
	return "Cart"
}
