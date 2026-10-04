package model

func All() []interface{} {
	return []interface{}{
		&Product{},
		&ProductImage{},
		&User{},
		&Cart{},
		&CartItem{},
		&Order{},
		&OrderItem{},
		&Review{},
	}
}
