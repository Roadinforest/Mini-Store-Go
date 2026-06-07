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
	productservice "mini-store-go/backend/internal/service/product"
	reviewservice "mini-store-go/backend/internal/service/review"
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
	productHandler := handler.NewProductHandler(
		validator,
		productservice.NewService(store.Products),
	)
	reviewHandler := handler.NewReviewHandler(
		validator,
		reviewservice.NewService(db, store.Reviews, store.Products),
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
			meGroup.GET("/reviews", reviewHandler.ListMine)
		}

		productGroup := api.Group("/products")
		{
			productGroup.GET("", productHandler.List)
			productGroup.GET("/featured", productHandler.Featured)
			productGroup.GET("/latest", productHandler.Latest)
			productGroup.GET("/categories", productHandler.Categories)
			productGroup.GET("/slug/:slug", productHandler.GetBySlug)
			productGroup.GET("/:id", productHandler.GetByID)
		}

		reviewGroup := api.Group("/reviews")
		{
			reviewGroup.GET("/product/:productID", reviewHandler.ListByProductID)
			reviewGroup.GET("/mine", middleware.RequireAuth(), reviewHandler.Mine)
			reviewGroup.POST("", middleware.RequireAuth(), reviewHandler.Upsert)
		}

		adminProductGroup := api.Group("/admin/products", middleware.RequireAdmin())
		{
			adminProductGroup.GET("", productHandler.List)
			adminProductGroup.POST("", productHandler.Create)
			adminProductGroup.GET("/:id", productHandler.GetByID)
			adminProductGroup.PUT("/:id", productHandler.Update)
			adminProductGroup.DELETE("/:id", productHandler.Delete)
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
