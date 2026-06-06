package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/http/response"
)

func writeError(c *gin.Context, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		response.Error(c, http.StatusInternalServerError, apperror.CodeInternal, "internal server error", nil)
		return
	}

	response.Error(c, mapStatus(appErr.Code), appErr.Code, appErr.Message, appErr.Details)
}

func writeBadRequest(c *gin.Context, message string, details interface{}) {
	response.Error(c, http.StatusBadRequest, apperror.CodeBadRequest, message, details)
}

func mapStatus(code string) int {
	switch code {
	case apperror.CodeValidation, apperror.CodeBadRequest:
		return http.StatusBadRequest
	case apperror.CodeUnauthorized:
		return http.StatusUnauthorized
	case apperror.CodeForbidden:
		return http.StatusForbidden
	case apperror.CodeNotFound:
		return http.StatusNotFound
	case apperror.CodeConflict:
		return http.StatusConflict
	case apperror.CodeOutOfStock:
		return http.StatusUnprocessableEntity
	case apperror.CodeServiceDisabled:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
