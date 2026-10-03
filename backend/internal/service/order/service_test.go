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
	"mini-store-go/backend/internal/infra/inventory"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	cartservice "mini-store-go/backend/internal/service/cart"
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
	return db, NewService(db, store.Orders, store.Users), user, product
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

func assertAvailable(t *testing.T, db *gorm.DB, productID string, want int) {
	t.Helper()
	got, err := inventory.Available(db, productID, "")
	if err != nil || got != want {
		t.Fatalf("available=%d, want=%d, err=%v", got, want, err)
	}
}

func TestRestockPreservesReservationsAndIsImmediatelyVisible(t *testing.T) {
	db, service, user, product := fixture(t)
	product.Stock = 5
	repo := gormrepo.NewProductRepository(db)
	if err := repo.Update(context.Background(), &product); err != nil {
		t.Fatal(err)
	}
	cart := seedCart(t, db, user, product, 2)
	order, err := service.Create(context.Background(), user.ID, cart.SessionCartID)
	if err != nil {
		t.Fatal(err)
	}
	assertAvailable(t, db, product.ID, 3)
	product.Stock = 10
	if err := repo.Update(context.Background(), &product); err != nil {
		t.Fatal(err)
	}
	assertAvailable(t, db, product.ID, 8)
	product.Stock = 1
	err = repo.Update(context.Background(), &product)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeConflict {
		t.Fatalf("expected reservation conflict, got %v", err)
	}
	assertAvailable(t, db, product.ID, 8)
	if _, err := service.MarkPaid(context.Background(), order.ID); err != nil {
		t.Fatal(err)
	}
	assertAvailable(t, db, product.ID, 8) // Physical deduction replaces, rather than doubles, the reservation.
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 8 {
		t.Fatalf("physical stock=%d, want 8", product.Stock)
	}
}

func TestExpiredReservationReleasesWithoutAnotherCheckout(t *testing.T) {
	db, _, user, product := fixture(t)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 100})
	assertAvailable(t, db, product.ID, 0)
	if err := db.Model(&order).Update("createdAt", gorm.Expr("statement_timestamp() - interval '16 minutes'")).Error; err != nil {
		t.Fatal(err)
	}
	assertAvailable(t, db, product.ID, 100)
	// Cart also reads current availability, with no Redis or cleanup request.
	store := gormrepo.NewStore(db)
	cartService := cartservice.NewService(store.Carts, store.Products, db)
	if _, err := cartService.AddItem(context.Background(), uuid.NewString(), &user.ID, product.ID); err != nil {
		t.Fatal(err)
	}
}

func TestLatePaymentCannotConsumeAnotherOrdersReservation(t *testing.T) {
	db, service, user, product := fixture(t)
	if err := db.Model(&product).Update("stock", 1).Error; err != nil {
		t.Fatal(err)
	}
	old := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 1})
	if err := db.Model(&old).Update("createdAt", gorm.Expr("statement_timestamp() - interval '16 minutes'")).Error; err != nil {
		t.Fatal(err)
	}
	current := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 1})
	_, err := service.MarkPaid(context.Background(), old.ID)
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeOutOfStock {
		t.Fatalf("expected stock conflict, got %v", err)
	}
	if _, err := service.MarkPaid(context.Background(), current.ID); err != nil {
		t.Fatal(err)
	}
	assertAvailable(t, db, product.ID, 0)
}

func TestExpiredOrderCanPayWhenStockIsStillAvailable(t *testing.T) {
	db, service, user, product := fixture(t)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	if err := db.Model(&order).Update("createdAt", gorm.Expr("statement_timestamp() - interval '16 minutes'")).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.MarkPaid(context.Background(), order.ID); err != nil {
		t.Fatal(err)
	}
	assertAvailable(t, db, product.ID, 98)
}

func TestConcurrentCheckoutsCannotOverReserve(t *testing.T) {
	db, service, user, product := fixture(t)
	if err := db.Model(&product).Update("stock", 1).Error; err != nil {
		t.Fatal(err)
	}
	const buyers = 8
	users := make([]model.User, buyers)
	carts := make([]model.Cart, buyers)
	for i := range users {
		users[i] = user
		users[i].ID = uuid.NewString()
		users[i].Email = uuid.NewString() + "@test.invalid"
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatal(err)
		}
		carts[i] = seedCart(t, db, users[i], product, 1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make(chan error, buyers)
	for i := range users {
		go func(i int) { <-start; _, err := service.Create(ctx, users[i].ID, carts[i].SessionCartID); errs <- err }(i)
	}
	close(start)
	successes := 0
	for range users {
		err := <-errs
		if err == nil {
			successes++
			continue
		}
		var appErr *apperror.Error
		if !errors.As(err, &appErr) || appErr.Code != apperror.CodeOutOfStock {
			t.Fatalf("unexpected checkout error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful checkouts=%d, want 1", successes)
	}
	assertAvailable(t, db, product.ID, 0)
}

func TestFailedCheckoutDoesNotLeaveAReservation(t *testing.T) {
	db, service, user, product := fixture(t)
	cart := seedCart(t, db, user, product, 2)
	// Force failure after order insertion, when checkout empties the cart.
	if err := db.Exec(`ALTER TABLE "Cart" ADD CONSTRAINT nonempty_test_cart CHECK (cardinality(items) > 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), user.ID, cart.SessionCartID); err == nil {
		t.Fatal("expected cart write failure")
	}
	assertAvailable(t, db, product.ID, 100)
	var count int64
	if err := db.Model(&model.Order{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("orders=%d after rollback", count)
	}
}

func TestPaymentWriteFailureRollsBackStockAndPreservesReservation(t *testing.T) {
	db, service, user, product := fixture(t)
	order := seedOrder(t, db, user, model.OrderItem{ProductID: product.ID, Qty: 2})
	if err := db.Exec(`ALTER TABLE "Order" ADD CONSTRAINT unpaid_test_order CHECK ("isPaid" = false)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.MarkPaid(context.Background(), order.ID); err == nil {
		t.Fatal("expected order write failure")
	}
	if err := db.First(&product, "id = ?", product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 100 {
		t.Fatalf("physical stock=%d, want 100", product.Stock)
	}
	assertAvailable(t, db, product.ID, 98)
}
