package model

func All() []interface{} {
	return []interface{}{
		&Product{},
		&ProductImage{},
		&User{},
		&Account{},
		&Session{},
		&VerificationToken{},
		&Cart{},
		&CartItem{},
		&Order{},
		&OrderItem{},
		&Review{},
	}
}
