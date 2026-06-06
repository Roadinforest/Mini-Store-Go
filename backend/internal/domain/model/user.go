package model

import (
	"time"

	"mini-store-go/backend/internal/domain/valueobject"
)

type User struct {
	ID            string                                        `gorm:"column:id;primaryKey;type:uuid"`
	Name          string                                        `gorm:"column:name;type:text;not null;default:NO_NAME"`
	Email         string                                        `gorm:"column:email;type:text;uniqueIndex:user_email_idx;not null"`
	EmailVerified *time.Time                                    `gorm:"column:email_verified"`
	Image         *string                                       `gorm:"column:image;type:text"`
	Password      *string                                       `gorm:"column:password;type:text"`
	Role          string                                        `gorm:"column:role;type:text;not null;default:user"`
	Address       valueobject.JSON[valueobject.ShippingAddress] `gorm:"column:address;type:jsonb"`
	PaymentMethod *string                                       `gorm:"column:payment_method;type:text"`
	CreatedAt     time.Time                                     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time                                     `gorm:"column:updated_at;autoUpdateTime"`

	Accounts []Account `gorm:"foreignKey:UserID;references:ID"`
	Sessions []Session `gorm:"foreignKey:UserID;references:ID"`
	Carts    []Cart    `gorm:"foreignKey:UserID;references:ID"`
	Orders   []Order   `gorm:"foreignKey:UserID;references:ID"`
	Reviews  []Review  `gorm:"foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "users"
}
