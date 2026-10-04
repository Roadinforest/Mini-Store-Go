package database

import (
	"context"
	_ "embed"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"mini-store-go/backend/internal/domain/model"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
)

//go:embed testdata/legacy.sql
var legacySQL string

//go:embed migrations/0001_normalize_commerce.down.sql
var rollbackSQL string

func legacyDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.EmptyPostgres(t)
	if err := db.Exec(legacySQL).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLegacyMigrationPreservesSnapshotsImageOrderAndRollback(t *testing.T) {
	db := legacyDatabase(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	store := gormrepo.NewStore(db)
	product, err := store.Products.GetByID(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(product.ImageURLs(), ",") != "cover,detail,cover" {
		t.Fatalf("images=%+v", product.Images)
	}
	cart, err := store.Carts.GetBySessionCartID(context.Background(), "legacy-session")
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 2 || cart.Items[0].ProductID != "p2" || cart.Items[1].Name != "Snapshot one" || !cart.Amounts().TotalPrice.Equal(decimal.RequireFromString("10.58")) {
		t.Fatalf("cart=%+v amounts=%+v", cart, cart.Amounts())
	}
	order, err := store.Orders.GetByID(context.Background(), "00000000-0000-0000-0000-000000000003")
	if err != nil {
		t.Fatal(err)
	}
	if !order.IsPaid() || order.IsDelivered() || order.OrderItems[0].Name != "Historical name" || !order.Amounts().TotalPrice.Equal(decimal.NewFromInt(28)) {
		t.Fatalf("order=%+v", order)
	}
	for _, col := range []struct{ table, column string }{{"Product", "images"}, {"Cart", "items"}, {"Cart", "totalPrice"}, {"Order", "itemsPrice"}, {"Order", "isPaid"}} {
		if db.Migrator().HasColumn(col.table, col.column) {
			t.Fatalf("old column remains: %+v", col)
		}
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(rollbackSQL).Error }); err != nil {
		t.Fatal(err)
	}
	var legacy struct {
		ItemsPrice decimal.Decimal `gorm:"column:itemsPrice"`
		TotalPrice decimal.Decimal `gorm:"column:totalPrice"`
		IsPaid     bool            `gorm:"column:isPaid"`
	}
	if err := db.Table("Order").Select(`"itemsPrice", "totalPrice", "isPaid"`).Scan(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if !legacy.IsPaid || !legacy.ItemsPrice.Equal(decimal.NewFromInt(20)) || !legacy.TotalPrice.Equal(decimal.NewFromInt(28)) {
		t.Fatalf("rollback=%+v", legacy)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migration after rollback: %v", err)
	}
}

func TestMigrationRejectsInvalidDataWithoutDroppingLegacyColumns(t *testing.T) {
	for _, tc := range []struct{ name, mutate, message string }{
		{"duplicate carts", `INSERT INTO "Cart" SELECT '00000000-0000-0000-0000-000000000099', "userId", "sessionCartId", items, "itemsPrice", "shippingPrice", "taxPrice", "totalPrice", "createdAt" FROM "Cart" WHERE "sessionCartId" = 'legacy-session'`, "Duplicate carts"},
		{"duplicate reviews", `INSERT INTO "Review" (id, "userId", "productId", rating, title, description) VALUES ('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'p1', 4, 'Title', 'Description'), ('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'p1', 4, 'Title', 'Description')`, "Duplicate reviews"},
		{"cart amount", `UPDATE "Cart" SET "totalPrice" = 999 WHERE "sessionCartId" = 'legacy-session'`, "Cart amounts"},
		{"historical amount", `UPDATE "Order" SET "itemsPrice" = 999`, "Order amounts"},
		{"status", `UPDATE "Order" SET "isPaid" = false`, "Order flags"},
		{"missing product", `UPDATE "Cart" SET items = ARRAY['{"product_id":"missing","qty":1,"price":"0.50","name":"Missing","slug":"missing"}'::json] WHERE "sessionCartId" = 'legacy-session'`, "foreign key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := legacyDatabase(t)
			if err := db.Exec(tc.mutate).Error; err != nil {
				t.Fatal(err)
			}
			err := AutoMigrate(db)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("expected %q, got %v", tc.message, err)
			}
			if !db.Migrator().HasColumn("Cart", "items") || !db.Migrator().HasColumn("Product", "images") || db.Migrator().HasTable(&model.CartItem{}) || db.Migrator().HasTable(&model.ProductImage{}) {
				t.Fatal("failed migration changed schema")
			}
		})
	}
}

func TestAutoMigrateCreatesFreshSchema(t *testing.T) {
	db := testutil.EmptyPostgres(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable(&model.CartItem{}) || !db.Migrator().HasTable(&model.ProductImage{}) || db.Migrator().HasColumn("Cart", "items") {
		t.Fatal("incorrect fresh schema")
	}
}
