package dto

type PageParams struct {
	Page  int `form:"page" json:"page" validate:"omitempty,min=1"`
	Limit int `form:"limit" json:"limit" validate:"omitempty,min=1,max=100"`
}

func (p PageParams) Normalize(defaultLimit int) PageParams {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = defaultLimit
	}
	return p
}

func (p PageParams) Offset() int {
	return (p.Page - 1) * p.Limit
}

type PageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}

func NewPageMeta(page, limit int, total int64) PageMeta {
	totalPages := int64(0)
	if limit > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}
	return PageMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

type Paged[T any] struct {
	Items []T      `json:"items"`
	Meta  PageMeta `json:"meta"`
}
