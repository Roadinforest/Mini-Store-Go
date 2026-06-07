package handler

import (
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	"mini-store-go/backend/internal/http/response"
	orderservice "mini-store-go/backend/internal/service/order"
	"mini-store-go/backend/internal/validation"
)

type OrderHandler struct {
	validator *validation.Validator
	service   *orderservice.Service
}

func NewOrderHandler(validator *validation.Validator, service *orderservice.Service) *OrderHandler {
	return &OrderHandler{
		validator: validator,
		service:   service,
	}
}

func (h *OrderHandler) Create(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	order, err := h.service.Create(c.Request.Context(), user.ID, middleware.SessionCartID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}

func (h *OrderHandler) Get(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	order, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	if user.Role != "admin" && order.UserID != user.ID {
		writeError(c, apperror.New(apperror.CodeForbidden, "order access denied"))
		return
	}
	response.OK(c, toOrderResponse(order))
}

func (h *OrderHandler) ListMine(c *gin.Context) {
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

	orders, meta, err := h.service.ListMine(c.Request.Context(), user.ID, page)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toPagedOrders(orders, meta))
}

func (h *OrderHandler) List(c *gin.Context) {
	var page dto.PageParams
	if err := c.ShouldBindQuery(&page); err != nil {
		writeBadRequest(c, "invalid query params", err.Error())
		return
	}
	if err := h.validator.Validate(page); err != nil {
		writeError(c, err)
		return
	}

	orders, meta, err := h.service.List(c.Request.Context(), page)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toPagedOrders(orders, meta))
}

func (h *OrderHandler) MarkPaid(c *gin.Context) {
	order, err := h.service.MarkPaid(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}

func (h *OrderHandler) MarkDelivered(c *gin.Context) {
	order, err := h.service.MarkDelivered(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toOrderResponse(order))
}
