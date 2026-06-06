package dto

import "mini-store-go/backend/internal/domain/valueobject"

type SignUpInput struct {
	Name            string `json:"name" validate:"required,min=3,max=120"`
	Email           string `json:"email" validate:"required,email,max=200"`
	Password        string `json:"password" validate:"required,min=6,max=72"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=6,max=72"`
}

type SignInInput struct {
	Email    string `json:"email" validate:"required,email,max=200"`
	Password string `json:"password" validate:"required,min=6,max=72"`
}

type UpdateProfileInput struct {
	Name  string `json:"name" validate:"required,min=3,max=120"`
	Email string `json:"email" validate:"required,email,max=200"`
}

type UpdatePaymentMethodInput struct {
	Type string `json:"type" validate:"required,min=2,max=64"`
}

type UpdateAddressInput = valueobject.ShippingAddress
