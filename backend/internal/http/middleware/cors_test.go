package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/config"
)

func TestCORSAllowsConfiguredOriginOnPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(CORS(config.CORSConfig{
		AllowedOrigins: []string{"https://mini-store-go-web.vercel.app"},
		AllowedMethods: []string{http.MethodGet, http.MethodOptions},
		AllowedHeaders: []string{"Content-Type"},
	}))
	engine.GET("/products/categories", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/products/categories", nil)
	req.Header.Set("Origin", "https://mini-store-go-web.vercel.app")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://mini-store-go-web.vercel.app" {
		t.Fatalf("unexpected access-control-allow-origin: %q", got)
	}
}
