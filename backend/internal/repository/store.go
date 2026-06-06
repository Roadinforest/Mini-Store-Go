package repository

type Store struct {
	Products ProductRepository
	Users    UserRepository
	Carts    CartRepository
	Orders   OrderRepository
	Reviews  ReviewRepository
}
