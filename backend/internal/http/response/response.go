package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

func JSON(c *gin.Context, status int, code, message string, data interface{}) {
	c.JSON(status, Envelope{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, "OK", "success", data)
}

func Error(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, Envelope{
		Code:    code,
		Message: message,
		Details: details,
	})
}
