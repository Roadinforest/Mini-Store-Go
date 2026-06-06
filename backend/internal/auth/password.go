package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var ErrPasswordMismatch = errors.New("password mismatch")

type PasswordHasher struct {
	key []byte
}

func NewPasswordHasher(secret string) *PasswordHasher {
	return &PasswordHasher{
		key: []byte(secret),
	}
}

func (h *PasswordHasher) HashPassword(password string) (string, error) {
	mac := hmac.New(sha256.New, h.key)
	if _, err := mac.Write([]byte(password)); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (h *PasswordHasher) ComparePassword(hashedPassword, plainPassword string) error {
	computed, err := h.HashPassword(plainPassword)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(hashedPassword), []byte(computed)) {
		return ErrPasswordMismatch
	}
	return nil
}
