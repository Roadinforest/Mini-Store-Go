package database

import (
	_ "embed"
	"testing"
	"time"
)

//go:embed migrations/0004_order_expiration.up.sql
var orderExpirationSQL string

func TestOrderExpirationMigrationPreservesPaidOrdersAndIsRepeatable(t *testing.T) {
	db := legacyDatabase(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	// Simulate the normalized pre-expiration schema, with one historic paid
	// order and two unpaid orders on opposite sides of the deadline.
	if err := db.Exec(`
		ALTER TABLE "Order" DROP COLUMN "expiresAt", DROP COLUMN "expiredAt";
		INSERT INTO "Order" (id, "userId", "shippingAddress", "paymentMethod", "shippingPrice", "taxPrice", "createdAt")
		SELECT '00000000-0000-0000-0000-000000000090', "userId", "shippingAddress", "paymentMethod", "shippingPrice", "taxPrice", clock_timestamp() - INTERVAL '20 minutes' FROM "Order" LIMIT 1;
		INSERT INTO "Order" (id, "userId", "shippingAddress", "paymentMethod", "shippingPrice", "taxPrice", "createdAt")
		SELECT '00000000-0000-0000-0000-000000000091', "userId", "shippingAddress", "paymentMethod", "shippingPrice", "taxPrice", clock_timestamp() FROM "Order" LIMIT 1;
	`).Error; err != nil {
		t.Fatal(err)
	}
	var originalClosure time.Time
	for run := 0; run < 2; run++ {
		if err := db.Exec(orderExpirationSQL).Error; err != nil {
			t.Fatal(err)
		}
		var ok bool
		if err := db.Raw(`SELECT
			(SELECT "expiredAt" IS NULL AND "paidAt" IS NOT NULL FROM "Order" WHERE id='00000000-0000-0000-0000-000000000003')
			AND (SELECT "expiredAt" IS NOT NULL AND "paidAt" IS NULL FROM "Order" WHERE id='00000000-0000-0000-0000-000000000090')
			AND (SELECT "expiredAt" IS NULL AND "expiresAt" > clock_timestamp() FROM "Order" WHERE id='00000000-0000-0000-0000-000000000091')
			AND (SELECT bool_and("expiresAt" = "createdAt" + INTERVAL '15 minutes') FROM "Order")
		`).Scan(&ok).Error; err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("migration invariants failed on run %d", run)
		}
		var closure time.Time
		if err := db.Raw(`SELECT "expiredAt" FROM "Order" WHERE id='00000000-0000-0000-0000-000000000090'`).Scan(&closure).Error; err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			originalClosure = closure
		} else if !originalClosure.Equal(closure) {
			t.Fatal("repeat migration rewrote closure time")
		}
	}
}
