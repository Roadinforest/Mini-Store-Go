-- Run against the legacy schema with psql --single-transaction -v ON_ERROR_STOP=1.
-- Stop application writers before applying this migration.
LOCK TABLE "Product", "Cart", "Order", "OrderItem", "Review" IN ACCESS EXCLUSIVE MODE;

-- Never discard conflicting business records or silently change historical money.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM "Cart" GROUP BY "sessionCartId" HAVING COUNT(*) > 1)
       OR EXISTS (SELECT 1 FROM "Cart" WHERE "userId" IS NOT NULL GROUP BY "userId" HAVING COUNT(*) > 1) THEN
        RAISE EXCEPTION 'Duplicate carts: reconcile sessionCartId/userId before migration';
    END IF;
    IF EXISTS (SELECT 1 FROM "Review" GROUP BY "userId", "productId" HAVING COUNT(*) > 1) THEN
        RAISE EXCEPTION 'Duplicate reviews: reconcile userId/productId before migration';
    END IF;
    IF EXISTS (
        SELECT 1 FROM "Cart" c
        WHERE c."itemsPrice" <> COALESCE((SELECT SUM((i.value->>'price')::numeric * (i.value->>'qty')::integer) FROM unnest(c.items) i(value)), 0)
           OR c."shippingPrice" <> CASE WHEN c."itemsPrice" > 0 AND c."itemsPrice" <= 100 THEN 10 ELSE 0 END
           OR c."taxPrice" <> ROUND(c."itemsPrice" * 0.15, 2)
           OR c."totalPrice" <> c."itemsPrice" + c."shippingPrice" + c."taxPrice"
    ) THEN
        RAISE EXCEPTION 'Cart amounts disagree with items/current policy: reconcile amounts before migration';
    END IF;
    IF EXISTS (
        SELECT 1 FROM "Order" o
        WHERE o."itemsPrice" <> COALESCE((SELECT SUM(i.price * i.qty) FROM "OrderItem" i WHERE i."orderId" = o.id), 0)
           OR o."totalPrice" <> o."itemsPrice" + o."shippingPrice" + o."taxPrice"
    ) THEN
        RAISE EXCEPTION 'Order amounts disagree with line items: reconcile historical amounts before migration';
    END IF;
    IF EXISTS (SELECT 1 FROM "Order" WHERE "isPaid" IS DISTINCT FROM ("paidAt" IS NOT NULL)
                  OR "isDelivered" IS DISTINCT FROM ("deliveredAt" IS NOT NULL)) THEN
        RAISE EXCEPTION 'Order flags disagree with timestamps: reconcile status before migration';
    END IF;
END $$;

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
-- Ordinality becomes a timestamp offset so the existing cart display order survives.
INSERT INTO "CartItem" ("cartId", "productId", qty, price, name, slug, image, "createdAt")
SELECT c.id, item.value->>'product_id', (item.value->>'qty')::integer,
       (item.value->>'price')::numeric(12,2), item.value->>'name', item.value->>'slug',
       COALESCE(item.value->>'image', ''), c."createdAt" + item.ordinality * INTERVAL '1 microsecond'
FROM "Cart" c CROSS JOIN LATERAL unnest(c.items) WITH ORDINALITY AS item(value, ordinality);

CREATE UNIQUE INDEX cart_session_idx ON "Cart" ("sessionCartId");
CREATE UNIQUE INDEX cart_user_idx ON "Cart" ("userId") WHERE "userId" IS NOT NULL;
CREATE UNIQUE INDEX review_user_product_idx ON "Review" ("userId", "productId");
ALTER TABLE "Review" ALTER COLUMN "isVerifiedPurchase" SET DEFAULT false;

ALTER TABLE "Product" DROP COLUMN images;
ALTER TABLE "Cart" DROP COLUMN items, DROP COLUMN "itemsPrice", DROP COLUMN "shippingPrice",
    DROP COLUMN "taxPrice", DROP COLUMN "totalPrice";
ALTER TABLE "Order" DROP COLUMN "itemsPrice", DROP COLUMN "totalPrice",
    DROP COLUMN "isPaid", DROP COLUMN "isDelivered";
