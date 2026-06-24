package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/config"
)

func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	allowAnyOrigin := false
	for _, origin := range cfg.AllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			allowAnyOrigin = true
			continue
		}
		allowedOrigins[origin] = struct{}{}
	}

	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := int(cfg.MaxAge.Seconds())

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowedOrigins[origin]; ok || allowAnyOrigin {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
					c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
		}

		if allowedMethods != "" {
			c.Writer.Header().Set("Access-Control-Allow-Methods", allowedMethods)
		}
		if allowedHeaders != "" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		}
		if exposedHeaders != "" {
			c.Writer.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
		}
		if maxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
