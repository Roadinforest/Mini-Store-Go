package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID          string          `gorm:"column:id;primaryKey;type:text"`
	Name        string          `gorm:"column:name;type:text;not null"`
	Slug        string          `gorm:"column:slug;type:text;uniqueIndex:product_slug_idx;not null"`
	Category    string          `gorm:"column:category;type:text;not null"`
	Images      []ProductImage  `gorm:"foreignKey:ProductID;references:ID;constraint:OnDelete:CASCADE"`
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

// Position preserves the product's image order, including its cover image.
type ProductImage struct {
	ProductID string `gorm:"column:productId;primaryKey;type:text"`
	Position  int    `gorm:"column:position;primaryKey;autoIncrement:false;check:product_image_position_nonnegative,position >= 0"`
	URL       string `gorm:"column:url;type:text;not null"`
}

func (ProductImage) TableName() string { return "ProductImage" }

func (p Product) ImageURLs() []string {
	urls := make([]string, 0, len(p.Images))
	for _, image := range p.Images {
		urls = append(urls, image.URL)
	}
	return urls
}

func (p Product) FirstImage() string {
	if len(p.Images) == 0 {
		return ""
	}
	return p.Images[0].URL
}
