-- Relevant tables from the pre-normalization GORM models.
CREATE TABLE "User" (
 id uuid PRIMARY KEY, name text NOT NULL DEFAULT 'NO_NAME', email text NOT NULL,
 "emailVerified" timestamptz, image text, password text, role text NOT NULL DEFAULT 'user',
 address jsonb, "paymentMethod" text, "createdAt" timestamptz, "updatedAt" timestamptz
);
CREATE UNIQUE INDEX user_email_idx ON "User" (email);
CREATE TABLE "Product" (
 id text PRIMARY KEY, name text NOT NULL, slug text NOT NULL, category text NOT NULL,
 images text[] NOT NULL, brand text NOT NULL, description text NOT NULL, stock integer NOT NULL,
 price numeric(12,2) NOT NULL DEFAULT 0, rating numeric(3,2) NOT NULL DEFAULT 0,
 "numReviews" integer NOT NULL DEFAULT 0, "isFeatured" boolean NOT NULL DEFAULT false,
 banner text, "createdAt" timestamptz
);
CREATE UNIQUE INDEX product_slug_idx ON "Product" (slug);
CREATE TABLE "Cart" (
 id uuid PRIMARY KEY, "userId" uuid REFERENCES "User"(id), "sessionCartId" text NOT NULL,
 items json[], "itemsPrice" numeric(12,2) NOT NULL, "shippingPrice" numeric(12,2) NOT NULL,
 "taxPrice" numeric(12,2) NOT NULL, "totalPrice" numeric(12,2) NOT NULL, "createdAt" timestamptz
);
CREATE TABLE "Order" (
 id uuid PRIMARY KEY, "userId" uuid NOT NULL REFERENCES "User"(id), "shippingAddress" jsonb NOT NULL,
 "paymentMethod" text NOT NULL, "paymentResult" jsonb, "itemsPrice" numeric(12,2) NOT NULL,
 "shippingPrice" numeric(12,2) NOT NULL, "taxPrice" numeric(12,2) NOT NULL,
 "totalPrice" numeric(12,2) NOT NULL, "isPaid" boolean NOT NULL DEFAULT false,
 "paidAt" timestamptz, "isDelivered" boolean NOT NULL DEFAULT false, "deliveredAt" timestamptz,
 "createdAt" timestamptz
);
CREATE TABLE "OrderItem" (
 "orderId" uuid NOT NULL REFERENCES "Order"(id), "productId" text NOT NULL REFERENCES "Product"(id),
 qty integer NOT NULL, price numeric(12,2) NOT NULL, name text NOT NULL, slug text NOT NULL,
 image text NOT NULL, PRIMARY KEY ("orderId", "productId")
);
CREATE TABLE "Review" (
 id uuid PRIMARY KEY, "userId" uuid NOT NULL REFERENCES "User"(id), "productId" text NOT NULL REFERENCES "Product"(id),
 rating integer NOT NULL, title text NOT NULL, description text NOT NULL,
 "isVerifiedPurchase" boolean NOT NULL DEFAULT true, "createdAt" timestamptz
);
INSERT INTO "User" (id, email, "createdAt", "updatedAt") VALUES ('00000000-0000-0000-0000-000000000001', 'legacy@test.invalid', now(), now());
INSERT INTO "Product" (id, name, slug, category, images, brand, description, stock, price, "createdAt") VALUES
 ('p1', 'Current name', 'p1', 'Test', ARRAY['cover', 'detail', 'cover'], 'Test', 'Test', 10, 99, now()),
 ('p2', 'Second', 'p2', 'Test', '{}', 'Test', 'Test', 10, 1, now());
INSERT INTO "Cart" (id, "userId", "sessionCartId", items, "itemsPrice", "shippingPrice", "taxPrice", "totalPrice", "createdAt") VALUES
 ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'legacy-session',
 ARRAY['{"product_id":"p2","name":"Snapshot two","slug":"old-p2","qty":1,"price":"0.10","image":"old-two"}'::json,
       '{"product_id":"p1","name":"Snapshot one","slug":"old-p1","qty":2,"price":"0.20","image":"old-one"}'::json], 0.50, 10, 0.08, 10.58, now()),
 ('00000000-0000-0000-0000-000000000004', NULL, 'empty-session', NULL, 0, 0, 0, 0, now());
INSERT INTO "Order" (id, "userId", "shippingAddress", "paymentMethod", "itemsPrice", "shippingPrice", "taxPrice", "totalPrice", "isPaid", "paidAt", "createdAt") VALUES
 ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000001', '{"full_name":"Historical User"}', 'cash', 20, 7, 1, 28, true, now(), now());
INSERT INTO "OrderItem" ("orderId", "productId", qty, price, name, slug, image) VALUES
 ('00000000-0000-0000-0000-000000000003', 'p1', 2, 10, 'Historical name', 'old-p1', 'old-image');
