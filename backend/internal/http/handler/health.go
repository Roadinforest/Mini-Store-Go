package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/http/response"
)

type HealthHandler struct {
	databaseProbe func(context.Context) error
	redisProbe    func(context.Context) error
}

func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	handler := &HealthHandler{}
	handler.databaseProbe = func(ctx context.Context) error {
		if db == nil {
			return fmt.Errorf("database disabled")
		}
		connection, err := db.DB()
		if err != nil {
			return err
		}
		return connection.PingContext(ctx)
	}
	if redis != nil {
		handler.redisProbe = func(ctx context.Context) error { return redis.Ping(ctx).Err() }
	}
	return handler
}

func (h *HealthHandler) Healthz(c *gin.Context) {
	response.OK(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
	})
}

func (h *HealthHandler) Ping(c *gin.Context) {
	response.OK(c, gin.H{
		"message":   "pong",
		"timestamp": time.Now().UTC(),
	})
}

// Readiness checks dependencies concurrently under one total deadline.
func (h *HealthHandler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	type result struct{ name, status string }
	results := make(chan result, 2)
	probes := map[string]func(context.Context) error{"database": h.databaseProbe, "redis": h.redisProbe}
	services := gin.H{}
	count := 0
	for name, probe := range probes {
		if probe == nil {
			services[name] = "disabled"
			continue
		}
		count++
		go func(name string, probe func(context.Context) error) {
			status := "ready"
			if probe(ctx) != nil {
				status = "unavailable"
			}
			results <- result{name, status}
		}(name, probe)
	}
	status := http.StatusOK
	for count > 0 {
		select {
		case result := <-results:
			services[result.name] = result.status
			if result.status != "ready" {
				status = http.StatusServiceUnavailable
			}
			count--
		case <-ctx.Done():
			for name := range probes {
				if _, ok := services[name]; !ok {
					services[name] = "unavailable"
				}
			}
			status = http.StatusServiceUnavailable
			count = 0
		}
	}
	if status != http.StatusOK {
		response.Error(c, status, "SERVICE_UNAVAILABLE", "dependencies unavailable", services)
		return
	}
	response.OK(c, gin.H{"status": "ready", "services": services})
}

func AbortServiceUnavailable(c *gin.Context, service string) {
	response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", service+" unavailable", nil)
	c.Abort()
}
