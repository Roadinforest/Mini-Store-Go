package valueobject

type ShippingAddress struct {
	FullName      string   `json:"full_name" validate:"required,min=3,max=120"`
	StreetAddress string   `json:"street_address" validate:"required,min=3,max=200"`
	City          string   `json:"city" validate:"required,min=2,max=80"`
	PostalCode    string   `json:"postal_code" validate:"required,min=2,max=20"`
	Country       string   `json:"country" validate:"required,min=2,max=80"`
	Lat           *float64 `json:"lat,omitempty"`
	Lng           *float64 `json:"lng,omitempty"`
}

type PaymentResult struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	EmailAddress string `json:"email_address"`
	PricePaid    string `json:"price_paid"`
}

type CartItem struct {
	ProductID string `json:"product_id" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Slug      string `json:"slug" validate:"required"`
	Qty       int    `json:"qty" validate:"required,min=1"`
	Image     string `json:"image" validate:"required"`
	Price     string `json:"price" validate:"required"`
}
