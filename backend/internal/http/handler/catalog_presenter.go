package handler

import (
	"time"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
)

type productResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Category    string    `json:"category"`
	Images      []string  `json:"images"`
	Brand       string    `json:"brand"`
	Description string    `json:"description"`
	Stock       int       `json:"stock"`
	Price       float64   `json:"price"`
	Rating      float64   `json:"rating"`
	NumReviews  int       `json:"num_reviews"`
	IsFeatured  bool      `json:"is_featured"`
	Banner      *string   `json:"banner,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type reviewResponse struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	ProductID          string    `json:"product_id"`
	Rating             int       `json:"rating"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	IsVerifiedPurchase bool      `json:"is_verified_purchase"`
	CreatedAt          time.Time `json:"created_at"`
	User               *struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"user,omitempty"`
	Product *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Slug  string `json:"slug"`
		Image string `json:"image,omitempty"`
	} `json:"product,omitempty"`
}

func toProductResponse(product *model.Product) productResponse {
	return productResponse{
		ID:          product.ID,
		Name:        product.Name,
		Slug:        product.Slug,
		Category:    product.Category,
		Images:      append([]string(nil), product.Images...),
		Brand:       product.Brand,
		Description: product.Description,
		Stock:       product.Stock,
		Price:       product.Price.InexactFloat64(),
		Rating:      product.Rating.InexactFloat64(),
		NumReviews:  product.NumReviews,
		IsFeatured:  product.IsFeatured,
		Banner:      product.Banner,
		CreatedAt:   product.CreatedAt,
	}
}

func toProductResponses(products []model.Product) []productResponse {
	items := make([]productResponse, 0, len(products))
	for i := range products {
		items = append(items, toProductResponse(&products[i]))
	}
	return items
}

func toPagedProducts(products []model.Product, meta dto.PageMeta) dto.Paged[productResponse] {
	return dto.Paged[productResponse]{
		Items: toProductResponses(products),
		Meta:  meta,
	}
}

func toReviewResponse(review *model.Review) reviewResponse {
	item := reviewResponse{
		ID:                 review.ID,
		UserID:             review.UserID,
		ProductID:          review.ProductID,
		Rating:             review.Rating,
		Title:              review.Title,
		Description:        review.Description,
		IsVerifiedPurchase: review.IsVerifiedPurchase,
		CreatedAt:          review.CreatedAt,
	}

	if review.User.ID != "" {
		item.User = &struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   review.User.ID,
			Name: review.User.Name,
		}
	}

	if review.Product.ID != "" {
		image := ""
		if len(review.Product.Images) > 0 {
			image = review.Product.Images[0]
		}
		item.Product = &struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Slug  string `json:"slug"`
			Image string `json:"image,omitempty"`
		}{
			ID:    review.Product.ID,
			Name:  review.Product.Name,
			Slug:  review.Product.Slug,
			Image: image,
		}
	}

	return item
}

func toReviewResponses(reviews []model.Review) []reviewResponse {
	items := make([]reviewResponse, 0, len(reviews))
	for i := range reviews {
		items = append(items, toReviewResponse(&reviews[i]))
	}
	return items
}
