-- Run with psql --single-transaction -v ON_ERROR_STOP=1 after stopping writers.
LOCK TABLE "Product", "ProductImage", "Cart", "CartItem", "Order", "OrderItem", "Review" IN ACCESS EXCLUSIVE MODE;

ALTER TABLE "Product" ADD COLUMN images text[] NOT NULL DEFAULT '{}';
UPDATE "Product" p SET images = COALESCE((SELECT array_agg(i.url ORDER BY i.position) FROM "ProductImage" i WHERE i."productId" = p.id), '{}');

ALTER TABLE "Cart" ADD COLUMN items json[] NOT NULL DEFAULT '{}',
    ADD COLUMN "itemsPrice" numeric(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN "shippingPrice" numeric(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN "taxPrice" numeric(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN "totalPrice" numeric(12,2) NOT NULL DEFAULT 0;
UPDATE "Cart" c SET items = COALESCE((SELECT array_agg(json_build_object(
    'product_id', i."productId", 'name', i.name, 'slug', i.slug, 'qty', i.qty,
    'image', i.image, 'price', i.price::text) ORDER BY i."createdAt", i."productId")
    FROM "CartItem" i WHERE i."cartId" = c.id), '{}'),
    "itemsPrice" = COALESCE((SELECT SUM(i.price * i.qty) FROM "CartItem" i WHERE i."cartId" = c.id), 0);
UPDATE "Cart" SET "shippingPrice" = CASE WHEN "itemsPrice" > 0 AND "itemsPrice" <= 100 THEN 10 ELSE 0 END,
    "taxPrice" = ROUND("itemsPrice" * 0.15, 2);
UPDATE "Cart" SET "totalPrice" = "itemsPrice" + "shippingPrice" + "taxPrice";

ALTER TABLE "Order" ADD COLUMN "itemsPrice" numeric(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN "totalPrice" numeric(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN "isPaid" boolean NOT NULL DEFAULT false,
    ADD COLUMN "isDelivered" boolean NOT NULL DEFAULT false;
UPDATE "Order" o SET "itemsPrice" = COALESCE((SELECT SUM(i.price * i.qty) FROM "OrderItem" i WHERE i."orderId" = o.id), 0),
    "isPaid" = ("paidAt" IS NOT NULL), "isDelivered" = ("deliveredAt" IS NOT NULL);
UPDATE "Order" SET "totalPrice" = "itemsPrice" + "shippingPrice" + "taxPrice";

DROP TABLE "CartItem";
DROP TABLE "ProductImage";
DROP INDEX cart_session_idx;
DROP INDEX cart_user_idx;
DROP INDEX review_user_product_idx;
ALTER TABLE "Review" ALTER COLUMN "isVerifiedPurchase" SET DEFAULT true;
