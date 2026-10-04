package database

import (
	_ "embed"
	"strings"
	"testing"

	"gorm.io/gorm"
	"mini-store-go/backend/internal/testutil"
)

//go:embed migrations/0003_remove_unused_auth_tables.up.sql
var removeLegacyAuthSQL string

func authCleanupFixture(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	db := legacyDatabase(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	var schema string
	if err := db.Raw("SELECT current_schema()").Scan(&schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
        CREATE TABLE "Account" ("userId" uuid REFERENCES "User"(id), provider text, "providerAccountId" text, access_token text, PRIMARY KEY(provider,"providerAccountId"));
        CREATE TABLE "Session" ("sessionToken" text PRIMARY KEY, "userId" uuid REFERENCES "User"(id), expires timestamp);
        CREATE TABLE "VerificationToken" (identifier text, token text, expires timestamp, PRIMARY KEY(identifier,token));
        INSERT INTO "Account" VALUES ('00000000-0000-0000-0000-000000000001','test-provider','test-id','synthetic-token');
        INSERT INTO "Session" VALUES ('synthetic-session','00000000-0000-0000-0000-000000000001','2026-10-04 12:00:00');
        INSERT INTO "VerificationToken" VALUES ('test@example.invalid','synthetic-verification','2026-10-04 13:00:00');
    `).Error; err != nil {
		t.Fatal(err)
	}
	return db, strings.ReplaceAll(removeLegacyAuthSQL, "public", schema)
}

func TestRemoveUnusedAuthTablesPreservesRowsAndDoesNotRecreateTables(t *testing.T) {
	db, script := authCleanupFixture(t)
	for run := 0; run < 2; run++ {
		if err := db.Exec(script).Error; err != nil {
			t.Fatal(err)
		}
		if err := AutoMigrate(db); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"Account", "Session", "VerificationToken"} {
			if db.Migrator().HasTable(table) {
				t.Fatalf("unused table recreated: %s", table)
			}
		}
		var ok bool
		if err := db.Raw(`SELECT
            (SELECT count(*)=3 FROM "LegacyAuthArchive")
            AND EXISTS(SELECT 1 FROM "LegacyAuthArchive" WHERE "tableName"='Account' AND "rowData"->>'access_token'='synthetic-token')
            AND EXISTS(SELECT 1 FROM "LegacyAuthArchive" WHERE "tableName"='Session' AND "rowData"->>'sessionToken'='synthetic-session')
            AND EXISTS(SELECT 1 FROM "LegacyAuthArchive" WHERE "tableName"='VerificationToken' AND "rowData"->>'token'='synthetic-verification')
            AND (SELECT count(*)=1 FROM "User")
            AND (SELECT count(*)=1 FROM "Order")
        `).Scan(&ok).Error; err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("cleanup lost archived or active business records")
		}
	}
}

func TestRemoveUnusedAuthTablesRollsBackWhenUnexpectedDependencyExists(t *testing.T) {
	db, script := authCleanupFixture(t)
	if err := db.Exec(`CREATE VIEW dependent_session AS SELECT * FROM "Session"`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return tx.Exec(migrationBody(script)).Error }); err == nil {
		t.Fatal("expected dependency to reject cleanup")
	}
	for _, table := range []string{"Account", "Session", "VerificationToken"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("partial cleanup removed %s", table)
		}
	}
	if db.Migrator().HasTable("LegacyAuthArchive") {
		t.Fatal("failed cleanup left archive behind")
	}
}

func TestFreshDatabaseDoesNotCreateUnusedAuthTables(t *testing.T) {
	db := testutil.EmptyPostgres(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"Account", "Session", "VerificationToken"} {
		if db.Migrator().HasTable(table) {
			t.Fatalf("fresh database creates unused %s", table)
		}
	}
}
