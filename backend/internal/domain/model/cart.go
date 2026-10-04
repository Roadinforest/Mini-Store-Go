package model

import (
	"time"

	"github.com/shopspring/decimal"

	"mini-store-go/backend/internal/domain/valueobject"
)

type Cart struct {
	ID            string     `gorm:"column:id;primaryKey;type:uuid"`
	UserID        *string    `gorm:"column:userId;type:uuid;uniqueIndex:cart_user_idx,where:\"userId\" IS NOT NULL"`
	SessionCartID string     `gorm:"column:sessionCartId;type:text;uniqueIndex:cart_session_idx;not null"`
	CreatedAt     time.Time  `gorm:"column:createdAt;autoCreateTime"`
	Items         []CartItem `gorm:"foreignKey:CartID;references:ID;constraint:OnDelete:CASCADE"`
	User          *User      `gorm:"foreignKey:UserID;references:ID"`
}

func (Cart) TableName() string { return "Cart" }

// CartItem stores the price and presentation captured when the item was added.
type CartItem struct {
	CartID    string          `gorm:"column:cartId;primaryKey;type:uuid"`
	ProductID string          `gorm:"column:productId;primaryKey;type:text"`
	Qty       int             `gorm:"column:qty;not null;check:cart_item_qty_positive,qty > 0"`
	Price     decimal.Decimal `gorm:"column:price;type:numeric(12,2);not null;check:cart_item_price_nonnegative,price >= 0"`
	Name      string          `gorm:"column:name;type:text;not null"`
	Slug      string          `gorm:"column:slug;type:text;not null"`
	Image     string          `gorm:"column:image;type:text;not null"`
	CreatedAt time.Time       `gorm:"column:createdAt;not null;autoCreateTime"`
	Product   Product         `gorm:"foreignKey:ProductID;references:ID"`
}

func (CartItem) TableName() string { return "CartItem" }

func (c Cart) Amounts() valueobject.Amounts {
	subtotal := decimal.Zero
	for _, item := range c.Items {
		subtotal = subtotal.Add(item.Price.Mul(decimal.NewFromInt(int64(item.Qty))))
	}
	return valueobject.CartAmounts(subtotal)
}
