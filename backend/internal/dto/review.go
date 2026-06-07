package dto

type UpsertReviewInput struct {
	ProductID   string `json:"product_id" validate:"required"`
	Title       string `json:"title" validate:"required,min=3"`
	Description string `json:"description" validate:"required,min=3"`
	Rating      int    `json:"rating" validate:"required,min=1,max=5"`
}

type ReviewListFilter struct {
	PageParams
	ProductID string `form:"product_id" json:"product_id"`
}
