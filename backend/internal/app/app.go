package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/http/router"
	"mini-store-go/backend/internal/infra/database"
	"mini-store-go/backend/internal/infra/rediscache"
	"mini-store-go/backend/internal/logger"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Log)
	if err != nil {
		return err
	}
	defer func() {
		_ = log.Sync()
	}()

	db, err := initDatabase(cfg, log)
	if err != nil {
		return err
	}
	defer closeDatabase(db, log)

	redisClient, err := initRedis(ctx, cfg, log)
	if err != nil {
		return err
	}
	defer closeRedis(redisClient, log)

	engine, err := router.New(cfg, log, db, redisClient)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port),
		Handler:      engine,
		ReadTimeout:  cfg.App.ReadTimeout,
		WriteTimeout: cfg.App.WriteTimeout,
		IdleTimeout:  cfg.App.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server starting",
			zap.String("name", cfg.App.Name),
			zap.String("env", cfg.App.Env),
			zap.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return fmt.Errorf("serve http: %w", err)
	case <-shutdownCtx.Done():
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()

	log.Info("server shutting down")
	if err := server.Shutdown(timeoutCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func initDatabase(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	db, err := database.New(cfg.Database)
	if err != nil {
		return nil, err
	}
	if db == nil {
		return nil, fmt.Errorf("database dsn is required")
	}
	if cfg.Database.AutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			return nil, err
		}
		log.Info("database auto migrate completed")
	}
	log.Info("database ready")
	return db, nil
}

func initRedis(ctx context.Context, cfg *config.Config, log *zap.Logger) (*redis.Client, error) {
	client, err := rediscache.New(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}
	if client == nil {
		log.Warn("redis disabled: empty addr")
		return nil, nil
	}
	log.Info("redis ready", zap.String("addr", cfg.Redis.Addr))
	return client, nil
}

func closeDatabase(db *gorm.DB, log *zap.Logger) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Warn("database close skipped", zap.Error(err))
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Warn("database close failed", zap.Error(err))
	}
}

func closeRedis(client *redis.Client, log *zap.Logger) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		log.Warn("redis close failed", zap.Error(err))
	}
}
