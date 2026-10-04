-- Read-only: inspect every duplicated user/product group before reconciliation.
WITH ranked AS (
    SELECT r.*,
           COUNT(*) OVER (PARTITION BY "userId", "productId") AS "duplicateCount",
           ROW_NUMBER() OVER (
               PARTITION BY "userId", "productId"
               ORDER BY "createdAt" DESC NULLS LAST, id DESC
           ) AS "retentionRank"
    FROM "Review" r
)
SELECT id, "userId", "productId", rating, title, description,
       "isVerifiedPurchase", "createdAt", "duplicateCount",
       ("retentionRank" = 1) AS "willRetain"
FROM ranked
WHERE "duplicateCount" > 1
ORDER BY "userId", "productId", "retentionRank";
