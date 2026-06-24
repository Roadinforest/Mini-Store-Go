package router

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"mini-store-go/backend/internal/config"
)

func TestNewRequiresDatabase(t *testing.T) {
	engine, err := New(&config.Config{}, zap.NewNop(), nil, nil)
	if err == nil {
		t.Fatal("expected nil database to return an error")
	}
	if engine != nil {
		t.Fatal("expected engine to be nil")
	}
	if !strings.Contains(err.Error(), "database is required") {
		t.Fatalf("expected database required error, got %q", err.Error())
	}
}
