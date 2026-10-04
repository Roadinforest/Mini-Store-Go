package handler

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestLivenessAndReadinessAreIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name            string
		database, redis func(context.Context) error
		status          int
	}{
		{"healthy without optional redis", func(context.Context) error { return nil }, nil, 200},
		{"database failed", func(context.Context) error { return errors.New("offline") }, nil, 503},
		{"redis failed", func(context.Context) error { return nil }, func(context.Context) error { return errors.New("offline") }, 503},
		{"request canceled", func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, nil, 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := &HealthHandler{databaseProbe: test.database, redisProbe: test.redis}
			router := gin.New()
			router.GET("/healthz", handler.Healthz)
			router.GET("/readyz", handler.Readyz)
			live := httptest.NewRecorder()
			router.ServeHTTP(live, httptest.NewRequest("GET", "/healthz", nil))
			if live.Code != 200 {
				t.Fatal("liveness depends on dependencies")
			}
			request := httptest.NewRequest("GET", "/readyz", nil)
			if test.name == "request canceled" {
				ctx, cancel := context.WithCancel(request.Context())
				cancel()
				request = request.WithContext(ctx)
			}
			ready := httptest.NewRecorder()
			router.ServeHTTP(ready, request)
			if ready.Code != test.status {
				t.Fatalf("status=%d body=%s", ready.Code, ready.Body)
			}
		})
	}
}
