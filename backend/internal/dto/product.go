package dto

type ProductListFilter struct {
	PageParams
	Query    string `form:"q" json:"q"`
	Category string `form:"category" json:"category"`
	Price    string `form:"price" json:"price"`
	Rating   string `form:"rating" json:"rating"`
	Sort     string `form:"sort" json:"sort"`
}
