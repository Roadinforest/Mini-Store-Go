package handler

import (
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	adminservice "mini-store-go/backend/internal/service/admin"
)

type overviewResponse struct {
	OrderCount   int64   `json:"order_count"`
	ProductCount int64   `json:"product_count"`
	UserCount    int64   `json:"user_count"`
	TotalSales   float64 `json:"total_sales"`
}

func toOverviewResponse(overview *adminservice.Overview) overviewResponse {
	return overviewResponse{
		OrderCount:   overview.OrderCount,
		ProductCount: overview.ProductCount,
		UserCount:    overview.UserCount,
		TotalSales:   overview.TotalSales.InexactFloat64(),
	}
}

func toUserResponses(users []model.User) []*middleware.AuthenticatedUser {
	items := make([]*middleware.AuthenticatedUser, 0, len(users))
	for i := range users {
		items = append(items, middleware.NewAuthenticatedUser(&users[i]))
	}
	return items
}

func toPagedUsers(users []model.User, meta dto.PageMeta) dto.Paged[*middleware.AuthenticatedUser] {
	return dto.Paged[*middleware.AuthenticatedUser]{
		Items: toUserResponses(users),
		Meta:  meta,
	}
}
