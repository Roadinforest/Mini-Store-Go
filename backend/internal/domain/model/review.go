package model

import "time"

type Review struct {
	ID                 string    `gorm:"column:id;primaryKey;type:uuid"`
	UserID             string    `gorm:"column:userId;type:uuid;index;not null"`
	ProductID          string    `gorm:"column:productId;type:text;index;not null"`
	Rating             int       `gorm:"column:rating;not null"`
	Title              string    `gorm:"column:title;type:text;not null"`
	Description        string    `gorm:"column:description;type:text;not null"`
	IsVerifiedPurchase bool      `gorm:"column:isVerifiedPurchase;not null;default:true"`
	CreatedAt          time.Time `gorm:"column:createdAt;autoCreateTime"`

	Product Product `gorm:"foreignKey:ProductID;references:ID"`
	User    User    `gorm:"foreignKey:UserID;references:ID"`
}

func (Review) TableName() string {
	return "Review"
}
