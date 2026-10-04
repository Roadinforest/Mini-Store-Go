package inventoryservice

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/infra/inventorylock"
	"mini-store-go/backend/internal/infra/rediscache"
)

const MaintenanceInterval = 5 * time.Second

type Service struct {
	db    *gorm.DB
	stock *rediscache.StockStore
}

func NewService(db *gorm.DB, stock *rediscache.StockStore) *Service {
	return &Service{db: db, stock: stock}
}

// Sync uses current database facts rather than replaying Redis increments.
// Repeating it repairs missed synchronization and an empty Redis alike.
func (s *Service) Sync(ctx context.Context) error {
	if s == nil || !s.stock.Enabled() {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := inventorylock.Acquire(tx); err != nil {
			return err
		}
		return s.SyncTx(ctx, tx)
	})
}

// SyncTx requires inventorylock.Acquire. Unclosed unpaid orders continue occupying
// inventory until expiration is committed, even if their deadline has passed.
func (s *Service) SyncTx(ctx context.Context, tx *gorm.DB) error {
	if s == nil || !s.stock.Enabled() {
		return nil
	}
	var products []model.Product
	if err := tx.Select("id", "stock").Find(&products).Error; err != nil {
		return err
	}
	stocks := make(map[string]int, len(products))
	for _, product := range products {
		stocks[product.ID] = product.Stock
	}
	var orders []model.Order
	if err := tx.Where(`"paidAt" IS NULL AND "expiredAt" IS NULL`).Preload("OrderItems").Find(&orders).Error; err != nil {
		return err
	}
	reservations := make([]rediscache.Reservation, 0, len(orders))
	for _, order := range orders {
		items := make([]rediscache.StockItem, 0, len(order.OrderItems))
		for _, item := range order.OrderItems {
			stocks[item.ProductID] -= item.Qty
			items = append(items, rediscache.StockItem{ProductID: item.ProductID, Qty: item.Qty})
		}
		reservations = append(reservations, rediscache.Reservation{OrderID: order.ID, ExpiresAt: order.Deadline().Unix(), Items: items})
	}
	for id, stock := range stocks {
		if stock < 0 {
			stocks[id] = 0
		}
	}
	return s.stock.Rebuild(ctx, stocks, reservations)
}

// ExpireDue commits order closure before Redis inventory is made available.
func (s *Service) ExpireDue(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := inventorylock.Acquire(tx); err != nil {
			return err
		}
		result := tx.Model(&model.Order{}).
			Where(`"paidAt" IS NULL AND "expiredAt" IS NULL AND COALESCE("expiresAt", "createdAt" + INTERVAL '15 minutes') <= clock_timestamp()`).
			Update("expiredAt", gorm.Expr("clock_timestamp()"))
		count = result.RowsAffected
		return result.Error
	})
	if err != nil {
		return 0, err
	}
	return count, s.Sync(ctx)
}

// Run reconciles on startup and at a fixed interval, without checkout traffic.
func (s *Service) Run(ctx context.Context, log *zap.Logger) {
	ticker := time.NewTicker(MaintenanceInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := s.ExpireDue(runCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			log.Warn("inventory maintenance failed; retrying on next tick", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
