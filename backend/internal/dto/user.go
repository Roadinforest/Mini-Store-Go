package dto

type UserListFilter struct {
	PageParams
	Query string `form:"q" json:"q"`
}
