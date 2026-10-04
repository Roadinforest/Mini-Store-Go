BEGIN;
SET LOCAL search_path = public, pg_catalog;
SET LOCAL lock_timeout = '10s';
-- Historical timestamp storage zone confirmed by the database owner.
SET LOCAL mini_store.legacy_timestamp_timezone = 'Asia/Shanghai';

-- Run after 0001, with application writers stopped.
LOCK TABLE "Product", "Review", "Order", "User", "Cart", "CartItem" IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF to_regclass('public."CartItem"') IS NULL
       OR EXISTS (SELECT 1 FROM information_schema.columns
                  WHERE table_schema = 'public' AND table_name = 'Cart' AND column_name = 'items') THEN
        RAISE EXCEPTION 'Run 0001_normalize_commerce.up.sql first';
    END IF;
END $$;

-- Preserve the original cache values; retain this archive for audit/recovery.
CREATE TABLE IF NOT EXISTS "ProductReviewStatsArchive" (
    "productId" text NOT NULL,
    "archivedAt" timestamptz NOT NULL,
    rating numeric(3,2) NOT NULL,
    "numReviews" integer NOT NULL,
    PRIMARY KEY ("productId", "archivedAt")
);
CREATE TEMP TABLE corrected_product_review_stats ON COMMIT DROP AS
SELECT p.id AS "productId", COALESCE(r.rating, 0)::numeric(3,2) AS rating,
       COALESCE(r."numReviews", 0)::integer AS "numReviews"
FROM "Product" p
LEFT JOIN (
    SELECT "productId", ROUND(AVG(rating), 2) AS rating, COUNT(*) AS "numReviews"
    FROM "Review" GROUP BY "productId"
) r ON r."productId" = p.id
WHERE p.rating IS DISTINCT FROM COALESCE(r.rating, 0)
   OR p."numReviews" IS DISTINCT FROM COALESCE(r."numReviews", 0);

INSERT INTO "ProductReviewStatsArchive" ("productId", "archivedAt", rating, "numReviews")
SELECT p.id, transaction_timestamp(), p.rating, p."numReviews"
FROM "Product" p JOIN corrected_product_review_stats c ON c."productId" = p.id;
UPDATE "Product" p SET rating = c.rating, "numReviews" = c."numReviews"
FROM corrected_product_review_stats c WHERE c."productId" = p.id;

-- Names match GORM's indexes for the current models.
CREATE INDEX IF NOT EXISTS "idx_Order_user_id" ON "Order" ("userId");
CREATE INDEX IF NOT EXISTS "idx_Review_user_id" ON "Review" ("userId");
CREATE INDEX IF NOT EXISTS "idx_Review_product_id" ON "Review" ("productId");

ALTER TABLE "Order"
    ALTER COLUMN "shippingAddress" TYPE jsonb USING "shippingAddress"::jsonb,
    ALTER COLUMN "paymentResult" TYPE jsonb USING "paymentResult"::jsonb;
ALTER TABLE "User" ALTER COLUMN address TYPE jsonb USING address::jsonb;

DO $$
DECLARE
    source_timezone text := NULLIF(current_setting('mini_store.legacy_timestamp_timezone', true), '');
    col record;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'public' AND table_name = 'Order'
                     AND column_name = 'legacyItemsAdjustment') THEN
        RAISE EXCEPTION 'Run 0001_normalize_commerce.up.sql first';
    END IF;
    IF source_timezone IS NULL OR NOT EXISTS (SELECT 1 FROM pg_timezone_names WHERE name = source_timezone) THEN
        RAISE EXCEPTION 'Set mini_store.legacy_timestamp_timezone to the confirmed historical time zone in this connection before running 0002';
    END IF;
    -- Only current model timestamp columns are converted. Existing timestamptz
    -- columns, archives and Prisma migration metadata are left unchanged.
    FOR col IN
        SELECT c.table_name, c.column_name, c.datetime_precision
        FROM information_schema.columns c
        JOIN (VALUES
            ('User', 'emailVerified'), ('User', 'createdAt'), ('User', 'updatedAt'),
            ('Product', 'createdAt'),
            ('Cart', 'createdAt'), ('CartItem', 'createdAt'),
            ('Order', 'paidAt'), ('Order', 'deliveredAt'), ('Order', 'createdAt'),
            ('Review', 'createdAt')
        ) expected(table_name, column_name)
        ON c.table_name = expected.table_name AND c.column_name = expected.column_name
        WHERE c.table_schema = 'public' AND c.data_type = 'timestamp without time zone'
        ORDER BY c.table_name, c.ordinal_position
    LOOP
        EXECUTE format(
            'ALTER TABLE public.%I ALTER COLUMN %I TYPE timestamptz(%s) USING %I AT TIME ZONE %L',
            col.table_name, col.column_name, col.datetime_precision, col.column_name, source_timezone);
    END LOOP;
END $$;

SELECT COUNT(*) AS corrected_products FROM corrected_product_review_stats;
COMMIT;
