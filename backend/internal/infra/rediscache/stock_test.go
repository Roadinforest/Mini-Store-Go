package rediscache

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"os"
	"sync"
	"testing"
	"time"
)

func stockFixture(t *testing.T) (*StockStore, string) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set TEST_REDIS_ADDR for Redis integration tests")
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return NewStockStore(client), uuid.NewString()
}
func TestReserveIsAtomicAcrossProducts(t *testing.T) {
	store, prefix := stockFixture(t)
	ctx := context.Background()
	a, b, order := prefix+"a", prefix+"b", prefix+"order"
	t.Cleanup(func() {
		store.client.Del(ctx, stockKey(a), stockKey(b), reservationKey(order))
		store.client.ZRem(ctx, reservationExpirationsKey, order)
	})
	if err := store.PrimeStocks(ctx, map[string]int{a: 5, b: 0}); err != nil {
		t.Fatal(err)
	}
	err := store.Reserve(ctx, order, []StockItem{{a, 2}, {b, 1}}, time.Minute)
	if !errors.Is(err, ErrInsufficient) {
		t.Fatalf("reserve error: %v", err)
	}
	value, _, err := store.Available(ctx, a)
	if err != nil || value != 5 {
		t.Fatalf("partial deduction: stock=%d err=%v", value, err)
	}
}
func TestConcurrentReserveAndRepeatedRelease(t *testing.T) {
	store, prefix := stockFixture(t)
	ctx := context.Background()
	product := prefix + "product"
	t.Cleanup(func() { store.client.Del(ctx, stockKey(product)) })
	if err := store.PrimeStocks(ctx, map[string]int{product: 10}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	outcomes := make(chan error, 32)
	for i := 0; i < 32; i++ {
		order := fmt.Sprintf("%s-%d", prefix, i)
		t.Cleanup(func() {
			store.Release(ctx, order)
			store.client.Del(ctx, reservationKey(order))
			store.client.ZRem(ctx, reservationExpirationsKey, order)
		})
		wg.Add(1)
		go func() { defer wg.Done(); outcomes <- store.Reserve(ctx, order, []StockItem{{product, 1}}, time.Minute) }()
	}
	wg.Wait()
	close(outcomes)
	successes := 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrInsufficient) {
			t.Fatal(err)
		}
	}
	if successes != 10 {
		t.Fatalf("successful reservations=%d want10", successes)
	}
	for i := 0; i < 32; i++ {
		order := fmt.Sprintf("%s-%d", prefix, i)
		for j := 0; j < 2; j++ {
			if err := store.Release(ctx, order); err != nil {
				t.Fatal(err)
			}
		}
	}
	stock, _, err := store.Available(ctx, product)
	if err != nil || stock != 10 {
		t.Fatalf("release stock=%d err=%v", stock, err)
	}
}
