package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/auth"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/http/handler"
	"mini-store-go/backend/internal/http/middleware"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	authservice "mini-store-go/backend/internal/service/auth"
	userservice "mini-store-go/backend/internal/service/user"
	"mini-store-go/backend/internal/validation"
)

func New(cfg *config.Config, log *zap.Logger, db *gorm.DB, redisClient *redis.Client) *gin.Engine {
	gin.SetMode(resolveGinMode(cfg.App.Env))

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(log))
	engine.Use(middleware.Recovery(log))
	engine.Use(middleware.CORS(cfg.CORS))
	engine.Use(middleware.SessionCartCookie(cfg.Auth))

	healthHandler := handler.NewHealthHandler(db, redisClient)

	engine.GET("/healthz", healthHandler.Healthz)

	store := gormrepo.NewStore(db)
	validator := validation.New()
	tokenManager := auth.NewManager(cfg.Auth)
	passwordHasher := auth.NewPasswordHasher(cfg.Auth.PasswordSecret)
	authHandler := handler.NewAuthHandler(
		cfg.Auth,
		validator,
		authservice.NewService(store.Users, tokenManager, passwordHasher),
	)
	userHandler := handler.NewUserHandler(
		validator,
		userservice.NewService(store.Users),
	)

	engine.Use(middleware.Authenticate(cfg.Auth, tokenManager, store.Users))

	api := engine.Group("/api/v1")
	{
		api.GET("/ping", healthHandler.Ping)

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/sign-up", authHandler.SignUp)
			authGroup.POST("/sign-in", authHandler.SignIn)
			authGroup.POST("/sign-out", authHandler.SignOut)
			authGroup.POST("/refresh", authHandler.Refresh)
			authGroup.GET("/me", middleware.RequireAuth(), authHandler.Me)
		}

		meGroup := api.Group("/users/me", middleware.RequireAuth())
		{
			meGroup.GET("", userHandler.Me)
			meGroup.PUT("", userHandler.UpdateProfile)
			meGroup.PUT("/profile", userHandler.UpdateProfile)
			meGroup.PUT("/address", userHandler.UpdateAddress)
			meGroup.PUT("/payment-method", userHandler.UpdatePaymentMethod)
		}
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
