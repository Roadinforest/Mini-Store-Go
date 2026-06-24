package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsConfigFromProjectRoot(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "backend", "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	content := []byte(`
database:
  dsn: postgres://user:pass@localhost:5432/mini_store?sslmode=disable
`)
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != "postgres://user:pass@localhost:5432/mini_store?sslmode=disable" {
		t.Fatalf("unexpected database dsn: %q", cfg.Database.DSN)
	}
}
