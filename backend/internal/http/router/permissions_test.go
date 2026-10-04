package router

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
	"mini-store-go/backend/internal/auth"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	"mini-store-go/backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRouterPermissionsAndSignOutCartIsolation(t *testing.T) {
	db := testutil.Postgres(t)
	cfg := &config.Config{Auth: config.AuthConfig{AccessSecret: "test-access", RefreshSecret: "test-refresh", AccessTTL: time.Hour, RefreshTTL: time.Hour, AccessCookieName: "access", RefreshCookieName: "refresh", SessionCartCookieName: "cart-session", CookieHTTPOnly: true}}
	owner := model.User{ID: uuid.NewString(), Email: "owner@test.invalid", Role: "user"}
	other := model.User{ID: uuid.NewString(), Email: "other@test.invalid", Role: "user"}
	admin := model.User{ID: uuid.NewString(), Email: "admin@test.invalid", Role: "admin"}
	order := model.Order{ID: uuid.NewString(), UserID: owner.ID, ShippingAddress: valueobject.JSON[valueobject.ShippingAddress]{Valid: true, Data: valueobject.ShippingAddress{}}, PaymentMethod: "cash"}
	for _, row := range []any{&owner, &other, &admin, &order} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	router, err := New(cfg, zap.NewNop(), db, nil)
	if err != nil {
		t.Fatal(err)
	}
	manager := auth.NewManager(cfg.Auth)
	token := func(user model.User) string {
		pair, err := manager.IssueTokenPair(user.ID, user.Email, user.Role, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return pair.AccessToken
	}
	for _, test := range []struct {
		path, token string
		status      int
	}{
		{"/api/v1/orders/" + order.ID, "", 401},
		{"/api/v1/orders/" + order.ID, token(other), 403},
		{"/api/v1/orders/" + order.ID, token(owner), 200},
		{"/api/v1/orders/" + order.ID, token(admin), 200},
		{"/api/v1/admin/overview", token(other), 403},
		{"/api/v1/admin/overview", token(admin), 200},
		{"/readyz", "", 200},
	} {
		request := httptest.NewRequest("GET", test.path, nil)
		if test.token != "" {
			request.Header.Set("Authorization", "Bearer "+test.token)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != test.status {
			t.Fatalf("%s status=%d want%d body=%s", test.path, recorder.Code, test.status, recorder.Body)
		}
	}
	request := httptest.NewRequest("POST", "/api/v1/auth/sign-out", nil)
	request.AddCookie(&http.Cookie{Name: "cart-session", Value: "old-account-cart"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	rotated := false
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "cart-session" && cookie.Value != "" && cookie.Value != "old-account-cart" {
			rotated = true
		}
	}
	if recorder.Code != 200 || !rotated {
		t.Fatalf("logout did not rotate guest cart cookie: %s", recorder.Body)
	}
}
