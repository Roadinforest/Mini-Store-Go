package orderservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
)

func fixture(t *testing.T) (*gorm.DB, *Service, model.User, model.Product) {
	t.Helper()
	db := testutil.Postgres(t)
	method := "cash"
	user := model.User{ID: uuid.NewString(), Email: uuid.NewString() + "@test.invalid", PaymentMethod: &method,
		Address: valueobject.JSON[valueobject.ShippingAddress]{Valid: true, Data: valueobject.ShippingAddress{FullName: "Test User", StreetAddress: "Test Street", City: "Test City", PostalCode: "12345", Country: "Test"}}}
	product := model.Product{ID: "product-a", Slug: "product-a", Name: "Product", Images: pq.StringArray{}, Stock: 100, Price: decimal.NewFromInt(10)}
	for _, row := range []interface{}{&user, &product} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := gormrepo.NewStore(db)
	return db, NewService(db, store.Orders, store.Carts, store.Users, store.Products, nil), user, product
}

func seedOrder(t *testing.T, db *gorm.DB, user model.User, items ...model.OrderItem) model.Order {
	t.Helper()
	order := model.Order{ID: uuid.NewString(), UserID: user.ID, ShippingAddress: user.Address, PaymentMethod: *user.PaymentMethod, OrderItems: items}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	return order
}

func seedCart(t *testing.T, db *gorm.DB, user model.User, product model.Product, qty int) model.Cart {
	t.Helper()
	cart := model.Cart{ID: uuid.NewString(), UserID: &user.ID, SessionCartID: uuid.NewString(), Items: valueobject.NewJSONArray([]valueobject.CartItem{{ProductID: product.ID, Qty: qty, Name: product.Name, Slug: product.Slug, Price: "10.00"}})}
	if err := db.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	return cart
}

func concurrently(n int, fn func() error) []error {
	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); <-start; errs[i] = fn() }(i)
	}
	close(start)
	wg.Wait()
	return errs
}

func TestConcurrentPaymentDeductsStockOnce(t *testing.T) {
	db, service, user, product := fixture(t)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, err := range concurrently(16, func() error {
		paid, err := service.MarkPaid(ctx, order.ID)
		if err == nil && (!paid.IsPaid || paid.PaidAt == nil) {
			return fmt.Errorf("order not paid")
		}
		return err
	}) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 98 {
		t.Fatalf("stock = %d, want 98", product.Stock)
	}
}

func TestPaymentRollsBackAllStockWhenAnItemIsUnavailable(t *testing.T) {
	db, service, user, product := fixture(t)
	other := model.Product{ID: "product-z", Slug: "product-z", Images: pq.StringArray{}, Stock: 0}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2}, model.OrderItem{ProductID: other.ID, Qty: 1})
	_, err := service.MarkPaid(context.Background(), order.ID)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeOutOfStock {
		t.Fatalf("expected out of stock, got %v", err)
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 100 {
		t.Fatalf("stock = %d, want 100", product.Stock)
	}
	reloaded, err := service.GetByID(context.Background(), order.ID)
	if err != nil || reloaded.IsPaid {
		t.Fatalf("order must remain unpaid: %v", err)
	}
}

func TestConcurrentCheckoutConsumesCartOnce(t *testing.T) {
	db, service, user, product := fixture(t)
	cart := seedCart(t, db, user, product, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	successes := 0
	for _, err := range concurrently(16, func() error { _, err := service.Create(ctx, user.ID, cart.SessionCartID); return err }) {
		if err == nil {
			successes++
			continue
		}
		var appErr *apperror.Error
		if !errors.As(err, &appErr) || appErr.Code != apperror.CodeBadRequest {
			t.Fatalf("unexpected checkout error: %v", err)
		}
	}
	var count int64
	if err := db.Model(&model.Order{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if successes != 1 || count != 1 {
		t.Fatalf("successes=%d orders=%d, want one each", successes, count)
	}
	if err := db.First(&cart, "id = ?", cart.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(cart.Items.Data) != 0 {
		t.Fatal("cart not emptied")
	}
}
