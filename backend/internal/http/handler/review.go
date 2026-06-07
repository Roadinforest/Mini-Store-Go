package handler

import (
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	"mini-store-go/backend/internal/http/response"
	reviewservice "mini-store-go/backend/internal/service/review"
	"mini-store-go/backend/internal/validation"
)

type ReviewHandler struct {
	validator *validation.Validator
	service   *reviewservice.Service
}

func NewReviewHandler(validator *validation.Validator, service *reviewservice.Service) *ReviewHandler {
	return &ReviewHandler{
		validator: validator,
		service:   service,
	}
}

func (h *ReviewHandler) ListByProductID(c *gin.Context) {
	reviews, err := h.service.ListByProductID(c.Request.Context(), c.Param("productID"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toReviewResponses(reviews))
}

func (h *ReviewHandler) Mine(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	productID := c.Query("product_id")
	if productID == "" {
		writeBadRequest(c, "product_id is required", nil)
		return
	}

	review, err := h.service.GetByUserAndProduct(c.Request.Context(), user.ID, productID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toReviewResponse(review))
}

func (h *ReviewHandler) ListMine(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	var page dto.PageParams
	if err := c.ShouldBindQuery(&page); err != nil {
		writeBadRequest(c, "invalid query params", err.Error())
		return
	}
	if err := h.validator.Validate(page); err != nil {
		writeError(c, err)
		return
	}

	reviews, meta, err := h.service.ListByUserID(c.Request.Context(), user.ID, page)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, dto.Paged[reviewResponse]{
		Items: toReviewResponses(reviews),
		Meta:  meta,
	})
}

func (h *ReviewHandler) Upsert(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	var input dto.UpsertReviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	review, err := h.service.Upsert(c.Request.Context(), user.ID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toReviewResponse(review))
}
