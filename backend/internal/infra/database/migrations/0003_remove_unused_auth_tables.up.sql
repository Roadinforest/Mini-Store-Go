BEGIN;
SET LOCAL search_path = public, pg_catalog;
SET LOCAL lock_timeout = '10s';

-- Current authentication uses User + JWT. Run with application writers stopped.
-- Archive full legacy rows, including provider tokens, before dropping tables.
CREATE TABLE IF NOT EXISTS "LegacyAuthArchive" (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "tableName" text NOT NULL CHECK ("tableName" IN ('Account', 'Session', 'VerificationToken')),
    "archivedAt" timestamptz NOT NULL,
    "rowData" jsonb NOT NULL
);
DO $$
DECLARE
    legacy_table text;
BEGIN
    -- Lock every existing source before copying any rows.
    FOREACH legacy_table IN ARRAY ARRAY['Account', 'Session', 'VerificationToken'] LOOP
        IF to_regclass(format('public.%I', legacy_table)) IS NOT NULL THEN
            EXECUTE format('LOCK TABLE public.%I IN ACCESS EXCLUSIVE MODE', legacy_table);
        END IF;
    END LOOP;
    FOREACH legacy_table IN ARRAY ARRAY['Account', 'Session', 'VerificationToken'] LOOP
        IF to_regclass(format('public.%I', legacy_table)) IS NOT NULL THEN
            EXECUTE format(
                'INSERT INTO public."LegacyAuthArchive" ("tableName", "archivedAt", "rowData") SELECT %L, transaction_timestamp(), to_jsonb(t) FROM public.%I t',
                legacy_table, legacy_table);
        END IF;
    END LOOP;
END $$;
-- No CASCADE: unexpected external dependencies abort and roll back the archive
-- and deletions together, rather than removing unrelated objects.
DROP TABLE IF EXISTS "Account", "Session", "VerificationToken";
SELECT "tableName", COUNT(*) AS archived_records
FROM "LegacyAuthArchive" WHERE "archivedAt" = transaction_timestamp()
GROUP BY "tableName" ORDER BY "tableName";
COMMIT;
