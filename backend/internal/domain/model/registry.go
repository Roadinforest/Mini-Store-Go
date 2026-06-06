package model

func All() []interface{} {
	return []interface{}{
		&Product{},
		&User{},
		&Account{},
		&Session{},
		&VerificationToken{},
		&Cart{},
		&Order{},
		&OrderItem{},
		&Review{},
	}
}
