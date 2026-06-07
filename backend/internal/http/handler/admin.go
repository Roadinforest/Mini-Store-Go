package handler

import (
	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	"mini-store-go/backend/internal/http/response"
	adminservice "mini-store-go/backend/internal/service/admin"
	"mini-store-go/backend/internal/validation"
)

type AdminHandler struct {
	validator *validation.Validator
	service   *adminservice.Service
}

func NewAdminHandler(validator *validation.Validator, service *adminservice.Service) *AdminHandler {
	return &AdminHandler{
		validator: validator,
		service:   service,
	}
}

func (h *AdminHandler) Overview(c *gin.Context) {
	overview, err := h.service.Overview(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toOverviewResponse(overview))
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var filter dto.UserListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		writeBadRequest(c, "invalid query params", err.Error())
		return
	}
	if err := h.validator.Validate(filter); err != nil {
		writeError(c, err)
		return
	}

	users, meta, err := h.service.ListUsers(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toPagedUsers(users, meta))
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	var input dto.UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, middleware.NewAuthenticatedUser(user))
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	currentUser := middleware.CurrentUser(c)
	if currentUser == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), c.Param("id"), currentUser.ID); err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
