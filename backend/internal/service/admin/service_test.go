package adminservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
)

func TestSalesUsePaidOrderSnapshotsWithoutDuplicatingShipping(t *testing.T) {
	db := testutil.Postgres(t)
	user := model.User{ID: uuid.NewString(), Email: "sales@test.invalid"}
	product := model.Product{ID: "sales", Name: "Current", Slug: "sales", Price: decimal.NewFromInt(999)}
	other := model.Product{ID: "other", Name: "Other", Slug: "other", Price: decimal.NewFromInt(999)}
	for _, row := range []any{&user, &product, &other} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	paid := model.Order{ID: uuid.NewString(), UserID: user.ID, ShippingAddress: valueobject.NewJSON(valueobject.ShippingAddress{}), PaymentMethod: "cash", PaidAt: &now, LegacyItemsAdjustment: decimal.RequireFromString("17.94"), LegacyTotalAdjustment: decimal.NewFromInt(-1), ShippingPrice: decimal.NewFromInt(7), TaxPrice: decimal.NewFromInt(1), OrderItems: []model.OrderItem{{ProductID: product.ID, Qty: 2, Price: decimal.NewFromInt(10)}, {ProductID: other.ID, Qty: 1, Price: decimal.NewFromInt(5)}}}
	unpaid := model.Order{ID: uuid.NewString(), UserID: user.ID, ShippingAddress: paid.ShippingAddress, PaymentMethod: "cash", ShippingPrice: decimal.NewFromInt(100), OrderItems: []model.OrderItem{{ProductID: product.ID, Qty: 1, Price: decimal.NewFromInt(100)}}}
	for _, row := range []any{&paid, &unpaid} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := gormrepo.NewStore(db)
	overview, err := NewService(db, store.Users).Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if overview.OrderCount != 2 || !overview.TotalSales.Equal(decimal.RequireFromString("49.94")) {
		t.Fatalf("overview=%+v", overview)
	}
}
