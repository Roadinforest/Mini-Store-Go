package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"mini-store-go/backend/internal/auth"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/http/middleware"
	"mini-store-go/backend/internal/http/response"
	authservice "mini-store-go/backend/internal/service/auth"
	"mini-store-go/backend/internal/validation"
)

type AuthHandler struct {
	cfg       config.AuthConfig
	validator *validation.Validator
	service   *authservice.Service
}

func NewAuthHandler(cfg config.AuthConfig, validator *validation.Validator, service *authservice.Service) *AuthHandler {
	return &AuthHandler{
		cfg:       cfg,
		validator: validator,
		service:   service,
	}
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var input dto.SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	user, tokenPair, err := h.service.SignUp(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setAuthCookies(c, tokenPair)
	response.OK(c, middleware.NewAuthenticatedUser(user))
}

func (h *AuthHandler) SignIn(c *gin.Context) {
	var input dto.SignInInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeBadRequest(c, "invalid request body", err.Error())
		return
	}
	if err := h.validator.Validate(input); err != nil {
		writeError(c, err)
		return
	}

	user, tokenPair, err := h.service.SignIn(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	h.setAuthCookies(c, tokenPair)
	response.OK(c, middleware.NewAuthenticatedUser(user))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(h.cfg.RefreshCookieName)
	if err != nil || refreshToken == "" {
		writeError(c, middleware.ErrUnauthorized())
		return
	}

	user, tokenPair, serviceErr := h.service.Refresh(c.Request.Context(), refreshToken)
	if serviceErr != nil {
		writeError(c, serviceErr)
		return
	}

	h.setAuthCookies(c, tokenPair)
	response.OK(c, middleware.NewAuthenticatedUser(user))
}

func (h *AuthHandler) SignOut(c *gin.Context) {
	h.clearAuthCookies(c)
	response.OK(c, gin.H{"signed_out": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	if user == nil {
		writeError(c, middleware.ErrUnauthorized())
		return
	}
	response.OK(c, middleware.NewAuthenticatedUser(user))
}

func (h *AuthHandler) setAuthCookies(c *gin.Context, pair *auth.TokenPair) {
	setCookie(c, h.cfg, pair.AccessCookieName, pair.AccessToken, pair.AccessExpiresAt)
	setCookie(c, h.cfg, pair.RefreshCookieName, pair.RefreshToken, pair.RefreshExpiresAt)
}

func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	expiredAt := time.Unix(0, 0)
	setCookie(c, h.cfg, h.cfg.AccessCookieName, "", expiredAt)
	setCookie(c, h.cfg, h.cfg.RefreshCookieName, "", expiredAt)
}

func setCookie(c *gin.Context, cfg config.AuthConfig, name, value string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if value == "" || maxAge < 0 {
		maxAge = -1
	}

	c.SetSameSite(middleware.ParseSameSite(cfg.CookieSameSite))
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   cfg.CookieDomain,
		MaxAge:   maxAge,
		Secure:   cfg.CookieSecure,
		HttpOnly: cfg.CookieHTTPOnly,
		SameSite: middleware.ParseSameSite(cfg.CookieSameSite),
	})
}
