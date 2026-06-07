package dto

type UserListFilter struct {
	PageParams
	Query string `form:"q" json:"q"`
}

type UpdateUserInput struct {
	Name  string `json:"name" validate:"required,min=3,max=120"`
	Email string `json:"email" validate:"required,email,max=200"`
	Role  string `json:"role" validate:"required,oneof=admin user"`
}
