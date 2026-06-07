package handler

import (
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	"mini-store-go/backend/internal/http/response"
	cartservice "mini-store-go/backend/internal/service/cart"
	"mini-store-go/backend/internal/validation"
)

type CartHandler struct {
	validator *validation.Validator
	service   *cartservice.Service
}

func NewCartHandler(validator *validation.Validator, service *cartservice.Service) *CartHandler {
	return &CartHandler{
		validator: validator,
		service:   service,
	}
}

func (h *CartHandler) Get(c *gin.Context) {
	cart, err := h.service.GetCurrentCart(c.Request.Context(), middleware.SessionCartID(c), currentUserID(c))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toCartResponse(cart))
}

func (h *CartHandler) AddItem(c *gin.Context) {
	var input dto.AddCartItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	cart, err := h.service.AddItem(c.Request.Context(), middleware.SessionCartID(c), currentUserID(c), input.ProductID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toCartResponse(cart))
}

func (h *CartHandler) RemoveItem(c *gin.Context) {
	cart, err := h.service.RemoveItem(c.Request.Context(), middleware.SessionCartID(c), currentUserID(c), c.Param("productID"))
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toCartResponse(cart))
}

func currentUserID(c *gin.Context) *string {
	user := middleware.CurrentUser(c)
	if user == nil {
		return nil
	}
	return &user.ID
}
