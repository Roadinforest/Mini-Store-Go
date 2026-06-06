package handler

import (
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	"mini-store-go/backend/internal/http/response"
	userservice "mini-store-go/backend/internal/service/user"
	"mini-store-go/backend/internal/validation"
)

type UserHandler struct {
	validator *validation.Validator
	service   *userservice.Service
}

func NewUserHandler(validator *validation.Validator, service *userservice.Service) *UserHandler {
	return &UserHandler{
		validator: validator,
		service:   service,
	}
}

func (h *UserHandler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}
	response.OK(c, middleware.NewAuthenticatedUser(user))
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	var input dto.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	updatedUser, err := h.service.UpdateProfile(c.Request.Context(), user.ID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, middleware.NewAuthenticatedUser(updatedUser))
}

func (h *UserHandler) UpdateAddress(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	var input dto.UpdateAddressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	updatedUser, err := h.service.UpdateAddress(c.Request.Context(), user.ID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, middleware.NewAuthenticatedUser(updatedUser))
}

func (h *UserHandler) UpdatePaymentMethod(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	var input dto.UpdatePaymentMethodInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	updatedUser, err := h.service.UpdatePaymentMethod(c.Request.Context(), user.ID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, middleware.NewAuthenticatedUser(updatedUser))
}
