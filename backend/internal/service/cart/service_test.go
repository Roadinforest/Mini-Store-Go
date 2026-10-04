package cartservice

import (
	"context"
	"errors"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
	"testing"
)

func TestCartTotalsStockLimitAndRemoval(t *testing.T) {
	db := testutil.Postgres(t)
	product := model.Product{ID: "phone", Name: "Phone", Slug: "phone", Stock: 2, Price: decimal.NewFromInt(60), Images: pq.StringArray{}}
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
	if !cart.TotalPrice.Equal(decimal.NewFromInt(79)) {
		t.Fatalf("total=%s want79 (60+9tax+10shipping)", cart.TotalPrice)
	}
	cart, err = service.AddItem(ctx, "guest", nil, product.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !cart.TotalPrice.Equal(decimal.NewFromInt(138)) {
		t.Fatalf("total=%s want138 with free shipping", cart.TotalPrice)
	}
	if _, err = service.AddItem(ctx, "guest", nil, product.ID); err == nil || !isOutOfStock(err) {
		t.Fatalf("stock limit: %v", err)
	}
	cart, err = service.GetCurrentCart(ctx, "guest", nil)
	if err != nil || cart.Items.Data[0].Qty != 2 {
		t.Fatalf("failed add changed cart: %#v %v", cart, err)
	}
	for i := 0; i < 2; i++ {
		cart, err = service.RemoveItem(ctx, "guest", nil, product.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(cart.Items.Data) != 0 || !cart.TotalPrice.IsZero() {
		t.Fatalf("empty cart has charges: %#v", cart)
	}
}

func isOutOfStock(err error) bool {
	var app *apperror.Error
	return errors.As(err, &app) && app.Code == apperror.CodeOutOfStock
}
