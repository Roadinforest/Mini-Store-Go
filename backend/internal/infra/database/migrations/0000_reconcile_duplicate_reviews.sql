BEGIN;

-- Run the complete script after stopping application writers.
-- Policy: retain the newest createdAt; break ties by id DESC. createdAt is NOT
-- an edit timestamp, so inspect 0000_report_duplicate_reviews.sql first.
LOCK TABLE "Product", "Review" IN ACCESS EXCLUSIVE MODE;

-- Keep complete original rows, including the retained record. No foreign key:
-- these snapshots must remain recoverable if users/products are later removed.
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

COMMIT;
