package database

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed migrations/0002_align_commerce.up.sql
var alignCommerceSQL string

func TestIncrementalMigrationAlignsStatsTypesIndexesAndHistoricalTimezone(t *testing.T) {
	db := legacyDatabase(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	var schema string
	if err := db.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
		t.Fatal(err)
	}
	// Production SQL intentionally targets public; tests target their isolated schema.
	script := strings.ReplaceAll(alignCommerceSQL, "public", schema)
	// Model a database upgraded by standalone 0001, without GORM alignment.
	if err := db.Exec(normalizeCommerceSQL).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
        CREATE TABLE "Account" ("createdAt" timestamp(3) DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3));
        CREATE TABLE "Session" ("sessionToken" text PRIMARY KEY, "userId" uuid NOT NULL, expires timestamp(3), "createdAt" timestamp(3), "updatedAt" timestamp(3));
        CREATE TABLE "VerificationToken" (identifier text, token text, expires timestamp(3));
        ALTER TABLE "Order" ALTER COLUMN "paidAt" TYPE timestamp(3) USING "paidAt" AT TIME ZONE 'UTC',
            ALTER COLUMN "shippingAddress" TYPE json USING "shippingAddress"::json,
            ALTER COLUMN "paymentResult" TYPE json USING "paymentResult"::json;
        ALTER TABLE "User" ALTER COLUMN address TYPE json USING address::json;
        UPDATE "Order" SET "paidAt" = '2026-01-01 12:34:56.123';
        INSERT INTO "Account" VALUES ('2026-01-01 12:34:56.123', NULL);
        UPDATE "Product" SET rating = 4.9, "numReviews" = 12;
        INSERT INTO "Review" (id, "userId", "productId", rating, title, description) VALUES
            ('00000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000001','p1',3,'Test','Test');
    `).Error; err != nil {
		t.Fatal(err)
	}
	// A different session zone must not influence interpretation of legacy data.
	if err := db.Exec(`SET TIME ZONE 'America/New_York'`).Error; err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 2; run++ {
		if err := db.Exec(script).Error; err != nil {
			t.Fatal(err)
		}
		var ok bool
		if err := db.Raw(`SELECT
            (SELECT rating = 3 AND "numReviews" = 1 FROM "Product" WHERE id = 'p1')
            AND (SELECT rating = 0 AND "numReviews" = 0 FROM "Product" WHERE id = 'p2')
            AND (SELECT COUNT(*) = 2 FROM "ProductReviewStatsArchive")
            AND (SELECT "paidAt" = '2026-01-01T04:34:56.123Z'::timestamptz FROM "Order")
            AND (SELECT "createdAt" = '2026-01-01T04:34:56.123Z'::timestamptz AND "updatedAt" IS NULL FROM "Account")
            AND (SELECT COUNT(*) = 4 FROM pg_indexes WHERE schemaname = current_schema() AND indexname IN
                ('idx_Order_user_id','idx_Review_user_id','idx_Review_product_id','idx_Session_user_id'))
            AND (SELECT COUNT(*) = 3 FROM information_schema.columns WHERE table_schema = current_schema() AND data_type = 'jsonb'
                AND ((table_name = 'Order' AND column_name IN ('shippingAddress','paymentResult')) OR (table_name = 'User' AND column_name = 'address')))
            AND (SELECT datetime_precision = 3 AND data_type = 'timestamp with time zone' FROM information_schema.columns
                WHERE table_schema = current_schema() AND table_name = 'Order' AND column_name = 'paidAt')
            AND (SELECT COALESCE(SUM(i.qty*i.price),0)+o."legacyItemsAdjustment"+o."shippingPrice"+o."taxPrice"+o."legacyTotalAdjustment" = 28
                FROM "Order" o LEFT JOIN "OrderItem" i ON i."orderId" = o.id GROUP BY o.id)
        `).Scan(&ok).Error; err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("incremental migration invariants failed on run %d", run)
		}
	}
}
