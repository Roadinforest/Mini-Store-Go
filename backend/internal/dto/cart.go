package dto

type AddCartItemInput struct {
	ProductID string `json:"product_id" validate:"required"`
}
