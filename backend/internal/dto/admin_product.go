package dto

type UpsertProductInput struct {
	Name        string   `json:"name" validate:"required,min=3"`
	Slug        string   `json:"slug" validate:"required,min=3"`
	Category    string   `json:"category" validate:"required,min=3"`
	Brand       string   `json:"brand" validate:"required,min=3"`
	Description string   `json:"description" validate:"required,min=3"`
	Stock       int      `json:"stock" validate:"min=0"`
	Images      []string `json:"images" validate:"required,min=1,dive,required"`
	IsFeatured  bool     `json:"is_featured"`
	Banner      *string  `json:"banner"`
	Price       string   `json:"price" validate:"required"`
	Rating      string   `json:"rating,omitempty"`
	NumReviews  int      `json:"num_reviews" validate:"min=0"`
}
