package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/http/handler"
	"mini-store-go/backend/internal/http/middleware"
)

func New(cfg *config.Config, log *zap.Logger, db *gorm.DB, redisClient *redis.Client) *gin.Engine {
	gin.SetMode(resolveGinMode(cfg.App.Env))

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(log))
	engine.Use(middleware.Recovery(log))
	engine.Use(middleware.CORS(cfg.CORS))

	healthHandler := handler.NewHealthHandler(db, redisClient)

	engine.GET("/healthz", healthHandler.Healthz)

	api := engine.Group("/api/v1")
	{
		api.GET("/ping", healthHandler.Ping)
	}

	return engine
}

func resolveGinMode(env string) string {
	if env == "production" {
		return gin.ReleaseMode
	}
	if env == "test" {
		return gin.TestMode
	}
	return gin.DebugMode
}
