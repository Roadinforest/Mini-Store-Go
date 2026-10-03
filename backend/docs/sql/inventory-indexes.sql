-- Run each statement outside a transaction on existing databases.
-- New databases using AutoMigrate receive the same indexes from model tags.
CREATE INDEX CONCURRENTLY IF NOT EXISTS order_reservation_idx
    ON "Order" ("isPaid", "createdAt");
CREATE INDEX CONCURRENTLY IF NOT EXISTS order_item_product_idx
    ON "OrderItem" ("productId");
