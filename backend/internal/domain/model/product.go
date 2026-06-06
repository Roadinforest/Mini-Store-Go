package model

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type Product struct {
	ID          string          `gorm:"column:id;primaryKey;type:text"`
	Name        string          `gorm:"column:name;type:text;not null"`
	Slug        string          `gorm:"column:slug;type:text;uniqueIndex:product_slug_idx;not null"`
	Category    string          `gorm:"column:category;type:text;not null"`
	Images      pq.StringArray  `gorm:"column:images;type:text[];not null"`
	Brand       string          `gorm:"column:brand;type:text;not null"`
	Description string          `gorm:"column:description;type:text;not null"`
	Stock       int             `gorm:"column:stock;not null"`
	Price       decimal.Decimal `gorm:"column:price;type:numeric(12,2);not null;default:0"`
	Rating      decimal.Decimal `gorm:"column:rating;type:numeric(3,2);not null;default:0"`
	NumReviews  int             `gorm:"column:numReviews;not null;default:0"`
	IsFeatured  bool            `gorm:"column:isFeatured;not null;default:false"`
	Banner      *string         `gorm:"column:banner;type:text"`
	CreatedAt   time.Time       `gorm:"column:createdAt;autoCreateTime"`

	OrderItems []OrderItem `gorm:"foreignKey:ProductID;references:ID"`
	Reviews    []Review    `gorm:"foreignKey:ProductID;references:ID"`
}

func (Product) TableName() string {
	return "Product"
}
