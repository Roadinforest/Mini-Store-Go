package app

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"mini-store-go/backend/internal/config"
)

func TestInitDatabaseRequiresDSN(t *testing.T) {
	db, err := initDatabase(&config.Config{}, zap.NewNop())
	if err == nil {
		t.Fatal("expected empty database dsn to return an error")
	}
	if db != nil {
		t.Fatal("expected database handle to be nil")
	}
	if !strings.Contains(err.Error(), "database dsn is required") {
		t.Fatalf("expected database dsn error, got %q", err.Error())
	}
}
