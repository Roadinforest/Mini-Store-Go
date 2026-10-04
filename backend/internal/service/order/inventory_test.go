package orderservice

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/infra/rediscache"
	inventoryservice "mini-store-go/backend/internal/service/inventory"
	productservice "mini-store-go/backend/internal/service/product"
)

func inventoryFixture(t *testing.T, service *Service) (*rediscache.StockStore, *redis.Client) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set TEST_REDIS_ADDR for inventory integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	stock := rediscache.NewStockStore(client)
	service.stockStore = stock
	service.inventory = inventoryservice.NewService(service.db, stock)
	t.Cleanup(func() {
		_ = stock.Rebuild(context.Background(), map[string]int{}, nil)
		_ = client.Close()
	})
	return stock, client
}

func assertStock(t *testing.T, stock *rediscache.StockStore, product string, want int) {
	t.Helper()
	got, exists, err := stock.Available(context.Background(), product)
	if err != nil || !exists || got != want {
		t.Fatalf("Redis stock=%d exists=%v err=%v, want %d", got, exists, err, want)
	}
}

func TestPaymentClosesExpiredOrderBeforeMaintenance(t *testing.T) {
	db, service, user, product := fixture(t)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	past := time.Now().Add(-time.Second)
	if err := db.Model(&order).Update("expiresAt", past).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		_, err := service.MarkPaid(context.Background(), order.ID)
		var appErr *apperror.Error
		if !errors.As(err, &appErr) || appErr.Code != apperror.CodeConflict {
			t.Fatalf("expired payment: %v", err)
		}
	}
	retained, err := service.GetByID(context.Background(), order.ID)
	if err != nil || retained.IsPaid() || retained.ExpiredAt == nil || len(retained.OrderItems) != 1 {
		t.Fatalf("expired order must be retained: %+v %v", retained, err)
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 100 {
		t.Fatalf("expired payment changed stock to %d", product.Stock)
	}
}

func TestExpirationRestoresStockOnceEvenWhenRedisIndexIsLost(t *testing.T) {
	db, service, user, product := fixture(t)
	stock, client := inventoryFixture(t, service)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	if err := service.inventory.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertStock(t, stock, product.ID, 98)
	if err := db.Model(&order).Update("expiresAt", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	client.ZRem(context.Background(), "reservation:order:expirations", order.ID)
	for i, want := range []int64{1, 0} {
		count, err := service.inventory.ExpireDue(context.Background())
		if err != nil || count != want {
			t.Fatalf("expiration run %d count=%d err=%v", i, count, err)
		}
		assertStock(t, stock, product.ID, 100)
	}
	retained, err := service.GetByID(context.Background(), order.ID)
	if err != nil || retained.ExpiredAt == nil || retained.IsPaid() {
		t.Fatalf("order=%+v %v", retained, err)
	}
}

func TestReplenishmentAndRedisRebuildPreserveOpenReservations(t *testing.T) {
	db, service, user, product := fixture(t)
	stock, client := inventoryFixture(t, service)
	cart := seedCart(t, db, user, product, 2)
	order, err := service.Create(context.Background(), user.ID, cart.SessionCartID)
	if err != nil {
		t.Fatal(err)
	}
	if order.ExpiresAt == nil || order.ExpiresAt.Sub(order.CreatedAt) != model.OrderReservationTTL {
		t.Fatal("missing payment deadline")
	}
	assertStock(t, stock, product.ID, 98)
	products := productservice.NewService(service.products, service.inventory)
	_, err = products.Update(context.Background(), product.ID, dto.UpsertProductInput{
		Name: product.Name, Slug: product.Slug, Stock: 120, Price: "10.00",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertStock(t, stock, product.ID, 118)
	// Lose both the stock key and reservation data, as after a Redis restart.
	client.Del(context.Background(), "stock:product:"+product.ID, "reservation:order:"+order.ID)
	client.ZRem(context.Background(), "reservation:order:expirations", order.ID)
	for i := 0; i < 2; i++ {
		if err := service.inventory.Sync(context.Background()); err != nil {
			t.Fatal(err)
		}
		assertStock(t, stock, product.ID, 118)
		qty, err := client.HGet(context.Background(), "reservation:order:"+order.ID, product.ID).Int()
		if err != nil || qty != 2 {
			t.Fatalf("reservation qty=%d err=%v", qty, err)
		}
	}
	if _, err := service.MarkPaid(context.Background(), order.ID); err != nil {
		t.Fatal(err)
	}
	assertStock(t, stock, product.ID, 118) // Payment consumes physical stock and removes the hold.
	if exists := client.Exists(context.Background(), "reservation:order:"+order.ID).Val(); exists != 0 {
		t.Fatal("paid reservation remains")
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 118 {
		t.Fatalf("physical stock=%d", product.Stock)
	}
}

func TestPaymentAndMaintenanceDoNotReleasePaidInventory(t *testing.T) {
	_, service, user, product := fixture(t)
	stock, _ := inventoryFixture(t, service)
	order := seedOrder(t, service.db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); _, err := service.MarkPaid(context.Background(), order.ID); errs <- err }()
	go func() { defer wg.Done(); _, err := service.inventory.ExpireDue(context.Background()); errs <- err }()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertStock(t, stock, product.ID, 98)
	retained, err := service.GetByID(context.Background(), order.ID)
	if err != nil || !retained.IsPaid() || retained.ExpiredAt != nil {
		t.Fatalf("order=%+v err=%v", retained, err)
	}
}

func TestMaintenanceClosesOrdersWithoutCheckoutTraffic(t *testing.T) {
	db, service, user, product := fixture(t)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	if err := db.Model(&order).Update("expiresAt", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); service.inventory.Run(ctx, zap.NewNop()) }()
	defer func() { cancel(); <-done }()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		retained, err := service.GetByID(context.Background(), order.ID)
		if err != nil {
			t.Fatal(err)
		}
		if retained.ExpiredAt != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("maintenance did not expire the order")
}
