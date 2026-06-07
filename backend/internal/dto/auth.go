package dto

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

type UpdateAddressInput struct {
	FullName      string   `json:"fullName" validate:"required,min=3,max=120"`
	StreetAddress string   `json:"streetAddress" validate:"required,min=3,max=200"`
	City          string   `json:"city" validate:"required,min=2,max=80"`
	PostalCode    string   `json:"postalCode" validate:"required,min=2,max=20"`
	Country       string   `json:"country" validate:"required,min=2,max=80"`
	Lat           *float64 `json:"lat,omitempty"`
	Lng           *float64 `json:"lng,omitempty"`
}
