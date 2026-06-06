package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/http/response"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	response.OK(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"services": gin.H{
			"database": healthStatus(h.db != nil),
			"redis":    healthStatus(h.redis != nil),
		},
	})
}

func (h *HealthHandler) Ping(c *gin.Context) {
	response.OK(c, gin.H{
		"message":   "pong",
		"timestamp": time.Now().UTC(),
	})
}

func healthStatus(ready bool) string {
	if ready {
		return "ready"
	}
	return "disabled"
}

func AbortServiceUnavailable(c *gin.Context, service string) {
	response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", service+" unavailable", nil)
	c.Abort()
}
