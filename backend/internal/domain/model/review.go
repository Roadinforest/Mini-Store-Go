package model

import "time"

type Review struct {
	ID                 string    `gorm:"column:id;primaryKey;type:uuid"`
	UserID             string    `gorm:"column:user_id;type:uuid;index;not null"`
	ProductID          string    `gorm:"column:product_id;type:text;index;not null"`
	Rating             int       `gorm:"column:rating;not null"`
	Title              string    `gorm:"column:title;type:text;not null"`
	Description        string    `gorm:"column:description;type:text;not null"`
	IsVerifiedPurchase bool      `gorm:"column:is_verified_purchase;not null;default:true"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime"`

	Product Product `gorm:"foreignKey:ProductID;references:ID"`
	User    User    `gorm:"foreignKey:UserID;references:ID"`
}

func (Review) TableName() string {
	return "reviews"
}
