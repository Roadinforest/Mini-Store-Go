BEGIN;
-- Stop application writers while applying this migration.
ALTER TABLE "Order" ADD COLUMN IF NOT EXISTS "expiresAt" timestamptz;
ALTER TABLE "Order" ADD COLUMN IF NOT EXISTS "expiredAt" timestamptz;
UPDATE "Order" SET "expiresAt" = "createdAt" + INTERVAL '15 minutes'
WHERE "expiresAt" IS NULL;
CREATE INDEX IF NOT EXISTS "idx_Order_expires_at" ON "Order" ("expiresAt");
-- Existing unpaid orders retain their original 15-minute payment window.
UPDATE "Order" SET "expiredAt" = clock_timestamp()
WHERE "paidAt" IS NULL AND "expiredAt" IS NULL AND "expiresAt" <= clock_timestamp();
COMMIT;
