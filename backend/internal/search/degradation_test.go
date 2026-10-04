package search

import (
	"context"
	"errors"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/domain/model"
	gormrepo "mini-store-go/backend/internal/repository/gorm"
	"mini-store-go/backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVectorFailureUsesTextSearchAndLogsDegradation(t *testing.T) {
	db := testutil.Postgres(t)
	product := model.Product{ID: "p1", Name: "phone", Slug: "phone", Images: pq.StringArray{}}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte("private upstream body"))
	}))
	defer upstream.Close()
	core, logs := observer.New(zap.InfoLevel)
	service := NewService(config.SearchConfig{Enabled: true, PineconeAPIKey: "test", PineconeHost: upstream.URL, QwenAPIKey: "test", QwenBaseURL: upstream.URL, Timeout: time.Second}, db, gormrepo.NewStore(db).Products, zap.New(core))
	// Redirect the fixed reranking URL to the same controlled failed upstream.
	service.client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		target, _ := http.NewRequestWithContext(r.Context(), r.Method, upstream.URL, r.Body)
		return http.DefaultTransport.RoundTrip(target)
	})
	results, err := service.SearchProducts(context.Background(), "phone", 10)
	if err != nil || len(results) != 1 || results[0].Product.ID != "p1" {
		t.Fatalf("fallback failed: %#v %v", results, err)
	}
	for _, stage := range []string{"vector", "rerank"} {
		found := false
		for _, entry := range logs.FilterMessage("search degraded").All() {
			if entry.ContextMap()["stage"] == stage {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s degradation log: %#v", stage, logs.All())
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDisabledSearchDoesNotWarnAndCancellationIsNotHidden(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	service := NewService(config.SearchConfig{}, nil, nil, zap.New(core))
	if _, err := service.SearchProducts(context.Background(), "phone", 10); err != nil {
		t.Fatal(err)
	}
	if logs.FilterMessage("search degraded").Len() != 0 {
		t.Fatal("disabled stage logged as failure")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.SearchProducts(ctx, "phone", 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation hidden: %v", err)
	}
}
