package cartservice

import (
	"context"
	"errors"
	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
	"testing"
)

func TestCartTotalsStockLimitAndRemoval(t *testing.T) {
	db := testutil.Postgres(t)
	product := model.Product{ID: "phone", Name: "Phone", Slug: "phone", Stock: 2, Price: decimal.NewFromInt(60), Images: []model.ProductImage{}}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	store := gormrepo.NewStore(db)
	service := NewService(store.Carts, store.Products, nil)
	ctx := context.Background()
	cart, err := service.AddItem(ctx, "guest", nil, product.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !cart.Amounts().TotalPrice.Equal(decimal.NewFromInt(79)) {
		t.Fatalf("total=%s want79 (60+9tax+10shipping)", cart.Amounts().TotalPrice)
	}
	cart, err = service.AddItem(ctx, "guest", nil, product.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !cart.Amounts().TotalPrice.Equal(decimal.NewFromInt(138)) {
		t.Fatalf("total=%s want138 with free shipping", cart.Amounts().TotalPrice)
	}
	if _, err = service.AddItem(ctx, "guest", nil, product.ID); err == nil || !isOutOfStock(err) {
		t.Fatalf("stock limit: %v", err)
	}
	cart, err = service.GetCurrentCart(ctx, "guest", nil)
	if err != nil || cart.Items[0].Qty != 2 {
		t.Fatalf("failed add changed cart: %#v %v", cart, err)
	}
	for i := 0; i < 2; i++ {
		cart, err = service.RemoveItem(ctx, "guest", nil, product.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(cart.Items) != 0 || !cart.Amounts().TotalPrice.IsZero() {
		t.Fatalf("empty cart has charges: %#v", cart)
	}
}

func isOutOfStock(err error) bool {
	var app *apperror.Error
	return errors.As(err, &app) && app.Code == apperror.CodeOutOfStock
}

func TestConcurrentCartCreationAndAddsDoNotLoseItems(t *testing.T) {
	db := testutil.Postgres(t)
	product := model.Product{ID: "concurrent", Name: "Phone", Slug: "concurrent", Stock: 40, Price: decimal.RequireFromString("0.10")}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	store := gormrepo.NewStore(db)
	service := NewService(store.Carts, store.Products, nil)
	ctx := context.Background()
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		go func() { _, err := service.AddItem(ctx, "concurrent-session", nil, product.ID); errs <- err }()
	}
	for i := 0; i < 20; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	cart, err := service.GetCurrentCart(ctx, "concurrent-session", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Qty != 20 || !cart.Amounts().TotalPrice.Equal(decimal.RequireFromString("12.30")) {
		t.Fatalf("cart=%+v amounts=%+v", cart, cart.Amounts())
	}
	var count int64
	if err := db.Model(&model.Cart{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("cart count=%d err=%v", count, err)
	}
	if _, err := service.ClearCart(ctx, "concurrent-session", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CartItem{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("item count=%d err=%v", count, err)
	}
}
