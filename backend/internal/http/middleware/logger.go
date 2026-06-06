package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(startedAt)
		fields := []zap.Field{
			zap.String("request_id", requestIDFromContext(c)),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", rawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.Int("body_size", c.Writer.Size()),
		}

		if len(c.Errors) > 0 {
			log.Error("request completed with errors", append(fields, zap.String("errors", c.Errors.String()))...)
			return
		}

		log.Info("request completed", fields...)
	}
}

func requestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get(RequestIDKey)
	if requestIDString, ok := requestID.(string); ok {
		return requestIDString
	}
	return ""
}
