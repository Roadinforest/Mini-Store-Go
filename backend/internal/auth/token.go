package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"mini-store-go/backend/internal/config"
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type Claims struct {
	UserID string    `json:"uid"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken       string
	RefreshToken      string
	AccessExpiresAt   time.Time
	RefreshExpiresAt  time.Time
	AccessCookieName  string
	RefreshCookieName string
}

type Manager struct {
	cfg config.AuthConfig
}

func NewManager(cfg config.AuthConfig) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) IssueTokenPair(userID, email, role string, now time.Time) (*TokenPair, error) {
	accessExpiresAt := now.Add(m.cfg.AccessTTL)
	refreshExpiresAt := now.Add(m.cfg.RefreshTTL)

	accessToken, err := m.sign(Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}, m.cfg.AccessSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.sign(Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}, m.cfg.RefreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:       accessToken,
		RefreshToken:      refreshToken,
		AccessExpiresAt:   accessExpiresAt,
		RefreshExpiresAt:  refreshExpiresAt,
		AccessCookieName:  m.cfg.AccessCookieName,
		RefreshCookieName: m.cfg.RefreshCookieName,
	}, nil
}

func (m *Manager) ParseAccessToken(token string) (*Claims, error) {
	return m.parse(token, m.cfg.AccessSecret, TokenTypeAccess)
}

func (m *Manager) ParseRefreshToken(token string) (*Claims, error) {
	return m.parse(token, m.cfg.RefreshSecret, TokenTypeRefresh)
}

func (m *Manager) sign(claims Claims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (m *Manager) parse(tokenString, secret string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Type != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}

	return claims, nil
}
