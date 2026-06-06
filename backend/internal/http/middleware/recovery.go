package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"mini-store-go/backend/internal/http/response"
)

func Recovery(log *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Error("panic recovered",
			zap.String("request_id", requestIDFromContext(c)),
			zap.Any("panic", recovered),
			zap.ByteString("stack", debug.Stack()),
		)

		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
		c.Abort()
	})
}
