package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/auth"
	"mini-store-go/backend/internal/config"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/repository"
)

const currentUserKey = "current_user"

type AuthenticatedUser struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Email         string      `json:"email"`
	Role          string      `json:"role"`
	Image         *string     `json:"image,omitempty"`
	PaymentMethod *string     `json:"payment_method,omitempty"`
	Address       interface{} `json:"address,omitempty"`
}

func NewAuthenticatedUser(user *model.User) *AuthenticatedUser {
	if user == nil {
		return nil
	}

	var address interface{}
	if user.Address.Valid {
		address = user.Address.Data
	}

	return &AuthenticatedUser{
		ID:            user.ID,
		Name:          user.Name,
		Email:         user.Email,
		Role:          user.Role,
		Image:         user.Image,
		PaymentMethod: user.PaymentMethod,
		Address:       address,
	}
}

func SessionCartCookie(cfg config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := c.Cookie(cfg.SessionCartCookieName); err != nil {
			c.SetSameSite(ParseSameSite(cfg.CookieSameSite))
			c.SetCookie(
				cfg.SessionCartCookieName,
				uuid.NewString(),
				int((365 * 24 * time.Hour).Seconds()),
				"/",
				cfg.CookieDomain,
				cfg.CookieSecure,
				cfg.CookieHTTPOnly,
			)
		}
		c.Next()
	}
}

func Authenticate(cfg config.AuthConfig, manager *auth.Manager, users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := tokenFromRequest(c, cfg.AccessCookieName)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := manager.ParseAccessToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		user, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err == nil {
			c.Set(currentUserKey, user)
		}

		c.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentUser(c) == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    apperror.CodeUnauthorized,
				"message": "authentication required",
			})
			return
		}
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    apperror.CodeUnauthorized,
				"message": "authentication required",
			})
			return
		}
		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    apperror.CodeForbidden,
				"message": "admin role required",
			})
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *model.User {
	value, exists := c.Get(currentUserKey)
	if !exists {
		return nil
	}
	user, _ := value.(*model.User)
	return user
}

func ParseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func ErrUnauthorized() error {
	return apperror.New(apperror.CodeUnauthorized, "authentication required")
}

func tokenFromRequest(c *gin.Context, cookieName string) string {
	if token, err := c.Cookie(cookieName); err == nil && token != "" {
		return token
	}

	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
