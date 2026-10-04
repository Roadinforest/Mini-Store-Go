package database

import (
	"context"
	_ "embed"
	"encoding/json"
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
	if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(migrationBody(rollbackSQL)).Error }); err != nil {
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
		{"invalid bytes", `UPDATE "Cart" SET items = ARRAY['256'::json] WHERE "sessionCartId" = 'legacy-session'`, "mixed/invalid JSON byte data"},
		{"invalid quantity", `UPDATE "Cart" SET items = ARRAY['{"product_id":"p1","qty":0,"price":"0.50","name":"Bad","slug":"bad"}'::json] WHERE "sessionCartId" = 'legacy-session'`, "missing/invalid item fields"},
		{"conflicting IDs", `UPDATE "Cart" SET items = ARRAY['{"product_id":"p1","productId":"p2","qty":1,"price":"0.50","name":"Bad","slug":"bad"}'::json] WHERE "sessionCartId" = 'legacy-session'`, "conflicting product ID fields"},
		{"repeated cart product", `UPDATE "Cart" SET items = ARRAY['{"product_id":"p1","qty":1,"price":"0.20","name":"One","slug":"one"}'::json, '{"product_id":"p1","qty":1,"price":"0.30","name":"Two","slug":"two"}'::json] WHERE "sessionCartId" = 'legacy-session'`, "Repeated product within one cart"},
		{"status", `UPDATE "Order" SET "isPaid" = false`, "Order flags"},
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

func TestStandaloneMigrationScriptsOwnTheirTransaction(t *testing.T) {
	db := legacyDatabase(t)
	if err := db.Exec(normalizeCommerceSQL).Error; err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn("Cart", "items") || !db.Migrator().HasTable(&model.CartItem{}) {
		t.Fatal("standalone upgrade failed")
	}
	if err := db.Exec(rollbackSQL).Error; err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasColumn("Cart", "items") || db.Migrator().HasTable(&model.CartItem{}) {
		t.Fatal("standalone rollback failed")
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
}

//go:embed migrations/0000_reconcile_duplicate_reviews.sql
var reconcileReviewsSQL string

func seedDuplicateReviews(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`INSERT INTO "Review" (id, "userId", "productId", rating, title, description, "createdAt") VALUES
 ('00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0000-000000000001', 'p1', 1, 'First', 'First', '2025-01-01'),
 ('00000000-0000-0000-0000-000000000006', '00000000-0000-0000-0000-000000000001', 'p1', 3, 'Second', 'Second', '2025-01-02'),
 ('00000000-0000-0000-0000-000000000007', '00000000-0000-0000-0000-000000000001', 'p1', 5, 'Third', 'Third', '2025-01-02')`).Error; err != nil {
		t.Fatal(err)
	}
}

func TestReviewReconciliationArchivesAllRecordsAndAllowsMigration(t *testing.T) {
	db := legacyDatabase(t)
	seedDuplicateReviews(t, db)
	if err := db.Exec(reconcileReviewsSQL).Error; err != nil {
		t.Fatal(err)
	}
	var reviews []model.Review
	if err := db.Find(&reviews).Error; err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 || reviews[0].ID != "00000000-0000-0000-0000-000000000007" || reviews[0].Description != "Third" {
		t.Fatalf("retained=%+v", reviews)
	}
	var archiveCount int64
	if err := db.Table("ReviewDuplicateArchive").Count(&archiveCount).Error; err != nil || archiveCount != 3 {
		t.Fatalf("archive=%d err=%v", archiveCount, err)
	}
	var snapshotDescription string
	if err := db.Raw(`SELECT "rowData"->>'description' FROM "ReviewDuplicateArchive" WHERE "reviewId" = '00000000-0000-0000-0000-000000000005'`).Scan(&snapshotDescription).Error; err != nil || snapshotDescription != "First" {
		t.Fatalf("original snapshot=%q err=%v", snapshotDescription, err)
	}
	var product model.Product
	if err := db.First(&product, "id = ?", "p1").Error; err != nil {
		t.Fatal(err)
	}
	if product.NumReviews != 1 || !product.Rating.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("stats=%+v", product)
	}
	if err := db.Exec(reconcileReviewsSQL).Error; err != nil {
		t.Fatalf("repeat cleanup: %v", err)
	}
	if err := db.Table("ReviewDuplicateArchive").Count(&archiveCount).Error; err != nil || archiveCount != 3 {
		t.Fatalf("repeated archive=%d err=%v", archiveCount, err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
}

func TestReviewReconciliationRollsBackArchiveAndDeletionOnFailure(t *testing.T) {
	db := legacyDatabase(t)
	seedDuplicateReviews(t, db)
	if err := db.Exec(`CREATE FUNCTION reject_review_deletion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'reject review deletion'; END $$;
 CREATE TRIGGER reject_review_deletion BEFORE DELETE ON "Review" FOR EACH ROW EXECUTE FUNCTION reject_review_deletion();`).Error; err != nil {
		t.Fatal(err)
	}
	err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(migrationBody(reconcileReviewsSQL)).Error })
	if err == nil || !strings.Contains(err.Error(), "reject review deletion") {
		t.Fatalf("expected rollback: %v", err)
	}
	var count int64
	if err := db.Model(&model.Review{}).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("reviews=%d err=%v", count, err)
	}
	if db.Migrator().HasTable("ReviewDuplicateArchive") {
		t.Fatal("failed cleanup left an archive table")
	}
}

func TestMigrationHandlesObservedLegacyFormatsWithoutChangingInvoices(t *testing.T) {
	db := legacyDatabase(t)
	seedDuplicateReviews(t, db)
	if err := db.Exec(`UPDATE "Cart" SET items = ARRAY[
 '{"productId":"p1","qty":1,"price":"79.99","name":"Polo","slug":"polo"}'::json,
 '{"productId":"missing","qty":1,"price":"2.99","name":"Removed product","slug":"removed"}'::json],
 "itemsPrice"=82.98,"shippingPrice"=10,"taxPrice"=12.45,"totalPrice"=105.43 WHERE "sessionCartId"='legacy-session';
 UPDATE "OrderItem" SET price=99.99;
 UPDATE "Order" SET "itemsPrice"=217.92,"shippingPrice"=0,"taxPrice"=32.69,"totalPrice"=250.61;`).Error; err != nil {
		t.Fatal(err)
	}
	// Reproduce an entire UTF-8 JSON array stored as numeric json[] elements.
	payload := []byte(`[{"product_id":"p2","qty":1,"price":"13.99","name":"手机","slug":"phone","image":"image"}]`)
	bytes := make([]int, len(payload))
	for i, b := range payload {
		bytes[i] = int(b)
	}
	encoded, err := json.Marshal(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO "Cart" (id,"sessionCartId",items,"itemsPrice","shippingPrice","taxPrice","totalPrice","createdAt")
 SELECT '00000000-0000-0000-0000-000000000008','byte-session',array_agg(value::json),13.99,10,2.10,26.09,now()
 FROM jsonb_array_elements_text(?::jsonb)`, string(encoded)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO "Order" (id,"userId","shippingAddress","paymentMethod","itemsPrice","shippingPrice","taxPrice","totalPrice","createdAt")
 VALUES ('00000000-0000-0000-0000-000000000009','00000000-0000-0000-0000-000000000001','{}','cash',0,7,1,11,now())`).Error; err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	store := gormrepo.NewStore(db)
	cart, err := store.Carts.GetBySessionCartID(context.Background(), "legacy-session")
	if err != nil {
		t.Fatal(err)
	}
	if len(cart.Items) != 1 || cart.Items[0].ProductID != "p1" || !cart.Amounts().TotalPrice.Equal(decimal.RequireFromString("101.99")) {
		t.Fatalf("normalized cart=%+v", cart)
	}
	byteCart, err := store.Carts.GetBySessionCartID(context.Background(), "byte-session")
	if err != nil {
		t.Fatal(err)
	}
	if len(byteCart.Items) != 1 || byteCart.Items[0].Name != "手机" || !byteCart.Amounts().TotalPrice.Equal(decimal.RequireFromString("26.09")) {
		t.Fatalf("byte cart=%+v", byteCart)
	}
	order, err := store.Orders.GetByID(context.Background(), "00000000-0000-0000-0000-000000000003")
	if err != nil {
		t.Fatal(err)
	}
	if !order.LegacyItemsAdjustment.Equal(decimal.RequireFromString("17.94")) || !order.Amounts().ItemsPrice.Equal(decimal.RequireFromString("217.92")) || !order.Amounts().TotalPrice.Equal(decimal.RequireFromString("250.61")) {
		t.Fatalf("invoice changed: %+v %+v", order, order.Amounts())
	}
	extra, err := store.Orders.GetByID(context.Background(), "00000000-0000-0000-0000-000000000009")
	if err != nil {
		t.Fatal(err)
	}
	if !extra.LegacyTotalAdjustment.Equal(decimal.NewFromInt(3)) || !extra.Amounts().TotalPrice.Equal(decimal.NewFromInt(11)) {
		t.Fatalf("total-only discrepancy changed: %+v", extra)
	}
	for _, tc := range []struct {
		table string
		count int64
	}{{"CartMigrationArchive", 3}, {"CartItemMigrationArchive", 1}, {"OrderMigrationArchive", 2}, {"ReviewDuplicateArchive", 3}, {"Review", 1}} {
		var count int64
		if err := db.Table(tc.table).Count(&count).Error; err != nil || count != tc.count {
			t.Fatalf("%s count=%d want=%d err=%v", tc.table, count, tc.count, err)
		}
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(migrationBody(rollbackSQL)).Error }); err != nil {
		t.Fatal(err)
	}
	var invoice struct {
		Items decimal.Decimal `gorm:"column:itemsPrice"`
		Total decimal.Decimal `gorm:"column:totalPrice"`
	}
	if err := db.Table("Order").Select(`"itemsPrice","totalPrice"`).Where("id = ?", order.ID).Scan(&invoice).Error; err != nil {
		t.Fatal(err)
	}
	if !invoice.Items.Equal(decimal.RequireFromString("217.92")) || !invoice.Total.Equal(decimal.RequireFromString("250.61")) {
		t.Fatalf("rollback changed invoice: %+v", invoice)
	}
	var reviews int64
	if err := db.Model(&model.Review{}).Count(&reviews).Error; err != nil || reviews != 3 {
		t.Fatalf("review restore=%d err=%v", reviews, err)
	}
	if !db.Migrator().HasTable("CartItemMigrationArchive") {
		t.Fatal("rollback removed unavailable-item archive")
	}
}

func TestRollbackDoesNotOverwriteReviewEditedAfterMigration(t *testing.T) {
	db := legacyDatabase(t)
	seedDuplicateReviews(t, db)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Review{}).Where("id = ?", "00000000-0000-0000-0000-000000000007").Update("description", "Edited after migration").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(migrationBody(rollbackSQL)).Error }); err != nil {
		t.Fatal(err)
	}
	var reviews []model.Review
	if err := db.Find(&reviews).Error; err != nil {
		t.Fatal(err)
	}
	if len(reviews) != 1 || reviews[0].Description != "Edited after migration" {
		t.Fatalf("rollback restored stale reviews: %+v", reviews)
	}
	var archives int64
	if err := db.Table("ReviewDuplicateArchive").Count(&archives).Error; err != nil || archives != 3 {
		t.Fatalf("archives=%d err=%v", archives, err)
	}
}
