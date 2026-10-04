BEGIN;

-- Run the complete script after stopping application writers.
-- Historical rows are archived before any removal or dropped columns.
LOCK TABLE "Product", "Cart", "Order", "OrderItem", "Review" IN ACCESS EXCLUSIVE MODE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM "Cart" GROUP BY "sessionCartId" HAVING COUNT(*) > 1)
       OR EXISTS (SELECT 1 FROM "Cart" WHERE "userId" IS NOT NULL GROUP BY "userId" HAVING COUNT(*) > 1) THEN
        RAISE EXCEPTION 'Duplicate carts: reconcile sessionCartId/userId before migration';
    END IF;
    IF EXISTS (SELECT 1 FROM "Order" WHERE "isPaid" IS DISTINCT FROM ("paidAt" IS NOT NULL)
                  OR "isDelivered" IS DISTINCT FROM ("deliveredAt" IS NOT NULL)) THEN
        RAISE EXCEPTION 'Order flags disagree with timestamps: reconcile status before migration';
    END IF;
END $$;

-- Preserve the original cached amounts, formats and historical invoices.
CREATE TABLE IF NOT EXISTS "CartMigrationArchive" (
    "cartId" uuid NOT NULL, "archivedAt" timestamptz NOT NULL, "rowData" jsonb NOT NULL,
    PRIMARY KEY ("cartId", "archivedAt")
);
CREATE TABLE IF NOT EXISTS "CartItemMigrationArchive" (
    "cartId" uuid NOT NULL, position integer NOT NULL, "archivedAt" timestamptz NOT NULL,
    reason text NOT NULL, "rowData" jsonb NOT NULL,
    PRIMARY KEY ("cartId", position, "archivedAt")
);
CREATE TABLE IF NOT EXISTS "OrderMigrationArchive" (
    "orderId" uuid NOT NULL, "archivedAt" timestamptz NOT NULL, "rowData" jsonb NOT NULL,
    PRIMARY KEY ("orderId", "archivedAt")
);
INSERT INTO "CartMigrationArchive" SELECT id, transaction_timestamp(), to_jsonb(c) FROM "Cart" c;
INSERT INTO "OrderMigrationArchive" SELECT id, transaction_timestamp(), to_jsonb(o) FROM "Order" o;

-- Some legacy json[] values contain UTF-8 bytes rather than JSON objects.
-- Normalize in temporary storage; never overwrite the original before archiving.
CREATE OR REPLACE FUNCTION pg_temp.normalize_cart_items(cart_id uuid, input_items json[])
RETURNS jsonb LANGUAGE plpgsql AS $$
DECLARE
    result jsonb := to_jsonb(COALESCE(input_items, '{}'::json[]));
    encoded_hex text;
    element jsonb;
    normalized jsonb := '[]'::jsonb;
    product_id text;
    quantity integer;
    item_price numeric;
BEGIN
    IF jsonb_array_length(result) = 0 THEN RETURN result; END IF;
    IF EXISTS (SELECT 1 FROM jsonb_array_elements(result) e WHERE jsonb_typeof(e) = 'number') THEN
        IF EXISTS (SELECT 1 FROM jsonb_array_elements(result) e
                   WHERE jsonb_typeof(e) <> 'number' OR (e #>> '{}') !~ '^[0-9]{1,3}$'
                      OR (e #>> '{}')::numeric > 255) THEN
            RAISE EXCEPTION 'Cart % has mixed/invalid JSON byte data', cart_id;
        END IF;
        SELECT string_agg(lpad(to_hex((value #>> '{}')::integer), 2, '0'), '' ORDER BY ordinality)
        INTO encoded_hex FROM jsonb_array_elements(result) WITH ORDINALITY;
        BEGIN
            result := convert_from(decode(encoded_hex, 'hex'), 'UTF8')::jsonb;
        EXCEPTION WHEN OTHERS THEN
            RAISE EXCEPTION 'Cart % cannot decode JSON byte data', cart_id;
        END;
        IF jsonb_typeof(result) <> 'array' THEN
            RAISE EXCEPTION 'Cart % decoded data must be a JSON array', cart_id;
        END IF;
    END IF;
    FOR element IN SELECT value FROM jsonb_array_elements(result) LOOP
        IF jsonb_typeof(element) = 'string' THEN element := (element #>> '{}')::jsonb; END IF;
        IF jsonb_typeof(element) IS DISTINCT FROM 'object' THEN
            RAISE EXCEPTION 'Cart % contains a non-object item', cart_id;
        END IF;
        IF element->>'product_id' IS NOT NULL AND element->>'productId' IS NOT NULL
           AND element->>'product_id' <> element->>'productId' THEN
            RAISE EXCEPTION 'Cart % has conflicting product ID fields', cart_id;
        END IF;
        product_id := COALESCE(NULLIF(element->>'product_id', ''), NULLIF(element->>'productId', ''));
        IF product_id IS NULL OR element->>'name' IS NULL OR element->>'slug' IS NULL
           OR COALESCE(element->>'qty', '') !~ '^[1-9][0-9]*$' OR element->>'price' IS NULL THEN
            RAISE EXCEPTION 'Cart % contains missing/invalid item fields', cart_id;
        END IF;
        quantity := (element->>'qty')::integer;
        item_price := (element->>'price')::numeric;
        IF item_price < 0 OR item_price = 'NaN'::numeric OR item_price >= 10000000000 THEN
            RAISE EXCEPTION 'Cart % contains an invalid price', cart_id;
        END IF;
        normalized := normalized || jsonb_build_array(jsonb_build_object(
            'product_id', product_id, 'name', element->>'name', 'slug', element->>'slug',
            'qty', quantity, 'price', item_price, 'image', COALESCE(element->>'image', '')));
    END LOOP;
    RETURN normalized;
END $$;

CREATE TEMP TABLE normalized_cart_items ON COMMIT DROP AS
SELECT c.id AS "cartId", (item.ordinality - 1)::integer AS position, item.value AS payload
FROM "Cart" c CROSS JOIN LATERAL jsonb_array_elements(
    pg_temp.normalize_cart_items(c.id, c.items)) WITH ORDINALITY item(value, ordinality);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM normalized_cart_items GROUP BY "cartId", payload->>'product_id' HAVING COUNT(*) > 1) THEN
        RAISE EXCEPTION 'Repeated product within one cart: reconcile quantities/snapshots before migration';
    END IF;
END $$;

-- Missing products cannot be checked out. Preserve their complete item snapshots
-- and remove only those unavailable items from the active relational cart.
INSERT INTO "CartItemMigrationArchive" ("cartId", position, "archivedAt", reason, "rowData")
SELECT i."cartId", i.position, transaction_timestamp(), 'missing_product', i.payload
FROM normalized_cart_items i LEFT JOIN "Product" p ON p.id = i.payload->>'product_id'
WHERE p.id IS NULL;

-- Match the current one-review-per-user/product rule without losing originals.
-- Retain newest createdAt, then id DESC; archive every row of each affected group.
CREATE TABLE IF NOT EXISTS "ReviewDuplicateArchive" (
    "reviewId" uuid NOT NULL,
    "retainedReviewId" uuid NOT NULL,
    "archivedAt" timestamptz NOT NULL,
    "rowData" jsonb NOT NULL,
    PRIMARY KEY ("reviewId", "archivedAt")
);

CREATE TEMP TABLE review_reconciliation_candidates ON COMMIT DROP AS
SELECT id, "productId", "retainedReviewId", "retentionRank"
FROM (
    SELECT id, "productId",
           FIRST_VALUE(id) OVER (
               PARTITION BY "userId", "productId"
               ORDER BY "createdAt" DESC NULLS LAST, id DESC
           ) AS "retainedReviewId",
           ROW_NUMBER() OVER (
               PARTITION BY "userId", "productId"
               ORDER BY "createdAt" DESC NULLS LAST, id DESC
           ) AS "retentionRank",
           COUNT(*) OVER (PARTITION BY "userId", "productId") AS "duplicateCount"
    FROM "Review"
) ranked
WHERE "duplicateCount" > 1;

INSERT INTO "ReviewDuplicateArchive" ("reviewId", "retainedReviewId", "archivedAt", "rowData")
SELECT r.id, c."retainedReviewId", transaction_timestamp(), to_jsonb(r)
FROM "Review" r JOIN review_reconciliation_candidates c ON c.id = r.id;

DELETE FROM "Review" r
USING review_reconciliation_candidates c
WHERE r.id = c.id AND c."retentionRank" > 1;

UPDATE "Product" p
SET rating = COALESCE((SELECT ROUND(AVG(r.rating), 2) FROM "Review" r WHERE r."productId" = p.id), 0),
    "numReviews" = (SELECT COUNT(*) FROM "Review" r WHERE r."productId" = p.id)
WHERE p.id IN (SELECT DISTINCT "productId" FROM review_reconciliation_candidates);

CREATE TABLE "ProductImage" (
    "productId" text NOT NULL,
    position integer NOT NULL,
    url text NOT NULL,
    PRIMARY KEY ("productId", position),
    CONSTRAINT product_image_position_nonnegative CHECK (position >= 0),
    CONSTRAINT "fk_Product_Images" FOREIGN KEY ("productId") REFERENCES "Product"(id) ON DELETE CASCADE
);
INSERT INTO "ProductImage" ("productId", position, url)
SELECT p.id, (image.ordinality - 1)::integer, image.url
FROM "Product" p CROSS JOIN LATERAL unnest(p.images) WITH ORDINALITY AS image(url, ordinality);

CREATE TABLE "CartItem" (
    "cartId" uuid NOT NULL,
    "productId" text NOT NULL,
    qty integer NOT NULL,
    price numeric(12,2) NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    image text NOT NULL,
    "createdAt" timestamptz NOT NULL,
    PRIMARY KEY ("cartId", "productId"),
    CONSTRAINT cart_item_qty_positive CHECK (qty > 0),
    CONSTRAINT cart_item_price_nonnegative CHECK (price >= 0),
    CONSTRAINT "fk_Cart_Items" FOREIGN KEY ("cartId") REFERENCES "Cart"(id) ON DELETE CASCADE,
    CONSTRAINT "fk_CartItem_Product" FOREIGN KEY ("productId") REFERENCES "Product"(id)
);
INSERT INTO "CartItem" ("cartId", "productId", qty, price, name, slug, image, "createdAt")
SELECT i."cartId", i.payload->>'product_id', (i.payload->>'qty')::integer,
       (i.payload->>'price')::numeric(12,2), i.payload->>'name', i.payload->>'slug',
       i.payload->>'image', COALESCE(c."createdAt", transaction_timestamp()) + i.position * INTERVAL '1 microsecond'
FROM normalized_cart_items i JOIN "Product" p ON p.id = i.payload->>'product_id'
JOIN "Cart" c ON c.id = i."cartId";

-- These deltas preserve unexplained historical differences; they do not label
-- them as tax, shipping or a confirmed fee. New orders default to zero.
ALTER TABLE "Order"
    ADD COLUMN "legacyItemsAdjustment" numeric(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN "legacyTotalAdjustment" numeric(12,2) NOT NULL DEFAULT 0;
UPDATE "Order" o SET
    "legacyItemsAdjustment" = o."itemsPrice" - COALESCE((SELECT SUM(i.price * i.qty) FROM "OrderItem" i WHERE i."orderId" = o.id), 0),
    "legacyTotalAdjustment" = o."totalPrice" - o."itemsPrice" - o."shippingPrice" - o."taxPrice";

CREATE UNIQUE INDEX cart_session_idx ON "Cart" ("sessionCartId");
CREATE UNIQUE INDEX cart_user_idx ON "Cart" ("userId") WHERE "userId" IS NOT NULL;
CREATE UNIQUE INDEX review_user_product_idx ON "Review" ("userId", "productId");
ALTER TABLE "Review" ALTER COLUMN "isVerifiedPurchase" SET DEFAULT false;

ALTER TABLE "Product" DROP COLUMN images;
ALTER TABLE "Cart" DROP COLUMN items, DROP COLUMN "itemsPrice", DROP COLUMN "shippingPrice",
    DROP COLUMN "taxPrice", DROP COLUMN "totalPrice";
ALTER TABLE "Order" DROP COLUMN "itemsPrice", DROP COLUMN "totalPrice",
    DROP COLUMN "isPaid", DROP COLUMN "isDelivered";

SELECT
    (SELECT COUNT(*) FROM "ReviewDuplicateArchive" WHERE "archivedAt" = transaction_timestamp()) AS archived_review_records,
    (SELECT COUNT(*) FROM "CartItemMigrationArchive" WHERE "archivedAt" = transaction_timestamp()) AS unavailable_cart_items,
    (SELECT COUNT(*) FROM "Order" WHERE "legacyItemsAdjustment" <> 0 OR "legacyTotalAdjustment" <> 0) AS orders_with_historical_deltas;

COMMIT;
