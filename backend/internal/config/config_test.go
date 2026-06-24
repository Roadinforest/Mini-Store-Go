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

func TestLoadReadsCORSAllowedOriginsFromEnvWithoutConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)
	t.Setenv("MINI_STORE_CORS_ALLOWED_ORIGINS", "https://mini-store-go-web.vercel.app, https://preview.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"https://mini-store-go-web.vercel.app", "https://preview.example.com"}
	if len(cfg.CORS.AllowedOrigins) != len(expected) {
		t.Fatalf("expected %d origins, got %d: %#v", len(expected), len(cfg.CORS.AllowedOrigins), cfg.CORS.AllowedOrigins)
	}
	for i, origin := range expected {
		if cfg.CORS.AllowedOrigins[i] != origin {
			t.Fatalf("expected origin %d to be %q, got %q", i, origin, cfg.CORS.AllowedOrigins[i])
		}
	}
}

func TestLoadReadsAuthCookieSettingsFromEnv(t *testing.T) {
	tempDir := t.TempDir()
	writeConfig(t, tempDir, `
auth:
  cookie_domain: ""
  cookie_secure: false
  cookie_same_site: lax
`)
	chdir(t, tempDir)
	t.Setenv("MINI_STORE_AUTH_COOKIE_DOMAIN", ".example.com")
	t.Setenv("MINI_STORE_AUTH_COOKIE_SECURE", "true")
	t.Setenv("MINI_STORE_AUTH_COOKIE_SAME_SITE", "none")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.CookieDomain != ".example.com" {
		t.Fatalf("expected cookie domain from env, got %q", cfg.Auth.CookieDomain)
	}
	if !cfg.Auth.CookieSecure {
		t.Fatal("expected secure cookies from env")
	}
	if cfg.Auth.CookieSameSite != "none" {
		t.Fatalf("expected same site from env, got %q", cfg.Auth.CookieSameSite)
	}
}

func TestLoadRejectsInvalidAuthCookieSecureEnv(t *testing.T) {
	tempDir := t.TempDir()
	chdir(t, tempDir)
	t.Setenv("MINI_STORE_AUTH_COOKIE_SECURE", "definitely")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid auth cookie secure env to return an error")
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
