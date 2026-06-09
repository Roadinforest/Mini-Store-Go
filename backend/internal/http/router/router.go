package router

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/ai"
	"mini-store-go/backend/internal/auth"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/http/handler"
	"mini-store-go/backend/internal/http/middleware"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	adminservice "mini-store-go/backend/internal/service/admin"
	authservice "mini-store-go/backend/internal/service/auth"
	cartservice "mini-store-go/backend/internal/service/cart"
	orderservice "mini-store-go/backend/internal/service/order"
	productservice "mini-store-go/backend/internal/service/product"
	reviewservice "mini-store-go/backend/internal/service/review"
	uploadservice "mini-store-go/backend/internal/service/upload"
	userservice "mini-store-go/backend/internal/service/user"
	"mini-store-go/backend/internal/validation"
)

func New(cfg *config.Config, log *zap.Logger, db *gorm.DB, redisClient *redis.Client) (*gin.Engine, error) {
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
	cartHandler := handler.NewCartHandler(
		validator,
		cartservice.NewService(store.Carts, store.Products),
	)
	orderHandler := handler.NewOrderHandler(
		validator,
		orderservice.NewService(db, store.Orders, store.Carts, store.Users, store.Products),
	)
	adminHandler := handler.NewAdminHandler(
		validator,
		adminservice.NewService(db, store.Users),
	)
	uploadSvc, err := uploadservice.NewService(cfg.Upload)
	if err != nil {
		return nil, fmt.Errorf("init upload service: %w", err)
	}
	uploadHandler := handler.NewUploadHandler(uploadSvc)
	aiModel, err := ai.NewEinoChatModel(context.Background(), cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("init ai model: %w", err)
	}
	aiHandler := handler.NewAIHandler(
		validator,
		ai.NewService(cfg.AI, aiModel, store.Products),
	)

	engine.Use(middleware.Authenticate(cfg.Auth, tokenManager, store.Users))
	engine.StaticFS(cfg.Upload.PublicBasePath, gin.Dir(filepath.Clean(cfg.Upload.StorageDir), false))

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

		cartGroup := api.Group("/cart")
		{
			cartGroup.GET("", cartHandler.Get)
			cartGroup.POST("/items", cartHandler.AddItem)
			cartGroup.DELETE("/items/:productID", cartHandler.RemoveItem)
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

		aiGroup := api.Group("/ai")
		{
			aiGroup.POST("/chat", aiHandler.Chat)
			aiGroup.POST("/chat/stream", aiHandler.Stream)
		}

		uploadGroup := api.Group("/uploads", middleware.RequireAuth())
		{
			uploadGroup.POST("/images", uploadHandler.UploadImage)
		}

		orderGroup := api.Group("/orders", middleware.RequireAuth())
		{
			orderGroup.POST("", orderHandler.Create)
			orderGroup.GET("/mine", orderHandler.ListMine)
			orderGroup.GET("/:id", orderHandler.Get)
		}

		adminProductGroup := api.Group("/admin/products", middleware.RequireAdmin())
		{
			adminProductGroup.GET("", productHandler.List)
			adminProductGroup.POST("", productHandler.Create)
			adminProductGroup.GET("/:id", productHandler.GetByID)
			adminProductGroup.PUT("/:id", productHandler.Update)
			adminProductGroup.DELETE("/:id", productHandler.Delete)
		}

		adminOrderGroup := api.Group("/admin/orders", middleware.RequireAdmin())
		{
			adminOrderGroup.GET("", orderHandler.List)
			adminOrderGroup.PUT("/:id/pay", orderHandler.MarkPaid)
			adminOrderGroup.PUT("/:id/deliver", orderHandler.MarkDelivered)
		}

		adminUserGroup := api.Group("/admin/users", middleware.RequireAdmin())
		{
			adminUserGroup.GET("", adminHandler.ListUsers)
			adminUserGroup.PUT("/:id", adminHandler.UpdateUser)
			adminUserGroup.DELETE("/:id", adminHandler.DeleteUser)
		}

		adminGroup := api.Group("/admin", middleware.RequireAdmin())
		{
			adminGroup.GET("/overview", adminHandler.Overview)
		}
	}

	return engine, nil
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
