package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsConfigFromProjectRoot(t *testing.T) {
	tempDir := t.TempDir()
	writeConfig(t, tempDir, `
database:
  dsn: postgres://user:pass@localhost:5432/mini_store?sslmode=disable
`)
	chdir(t, tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "postgres://user:pass@localhost:5432/mini_store?sslmode=disable" {
		t.Fatalf("unexpected database dsn: %q", cfg.Database.DSN)
	}
}

func TestLoadReadsDatabaseDSNFromEnvWithoutConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)
	t.Setenv("MINI_STORE_DATABASE_DSN", "postgres://env-user:env-pass@localhost:5432/mini_store?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "postgres://env-user:env-pass@localhost:5432/mini_store?sslmode=disable" {
		t.Fatalf("unexpected database dsn: %q", cfg.Database.DSN)
	}
}

func TestLoadUsesPlatformPortEnv(t *testing.T) {
	tempDir := t.TempDir()
	writeConfig(t, tempDir, `
app:
  port: 8080
database:
  dsn: postgres://user:pass@localhost:5432/mini_store?sslmode=disable
`)
	chdir(t, tempDir)
	t.Setenv("PORT", "4321")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.App.Port != 4321 {
		t.Fatalf("expected app port from PORT env, got %d", cfg.App.Port)
	}
}

func TestLoadRejectsInvalidPlatformPortEnv(t *testing.T) {
	tempDir := t.TempDir()
	writeConfig(t, tempDir, `
database:
  dsn: postgres://user:pass@localhost:5432/mini_store?sslmode=disable
`)
	chdir(t, tempDir)
	t.Setenv("PORT", "not-a-port")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid PORT env to return an error")
	}
}

func writeConfig(t *testing.T, root string, content string) {
	t.Helper()

	configDir := filepath.Join(root, "backend", "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}
