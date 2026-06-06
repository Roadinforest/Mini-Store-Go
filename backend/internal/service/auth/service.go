package authservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/auth"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

type Service struct {
	users    repository.UserRepository
	tokens   *auth.Manager
	password *auth.PasswordHasher
}

func NewService(users repository.UserRepository, tokens *auth.Manager, password *auth.PasswordHasher) *Service {
	return &Service{
		users:    users,
		tokens:   tokens,
		password: password,
	}
}

func (s *Service) SignUp(ctx context.Context, input dto.SignUpInput) (*model.User, *auth.TokenPair, error) {
	if input.Password != input.ConfirmPassword {
		return nil, nil, apperror.New(apperror.CodeValidation, "passwords do not match")
	}

	_, err := s.users.GetByEmail(ctx, strings.ToLower(input.Email))
	if err == nil {
		return nil, nil, apperror.New(apperror.CodeConflict, "email already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to check existing user", err)
	}

	hashedPassword, err := s.password.HashPassword(input.Password)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to hash password", err)
	}

	user := &model.User{
		ID:        uuid.NewString(),
		Name:      input.Name,
		Email:     strings.ToLower(input.Email),
		Password:  pointer(hashedPassword),
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to create user", err)
	}

	tokenPair, err := s.tokens.IssueTokenPair(user.ID, user.Email, user.Role, time.Now().UTC())
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to issue token pair", err)
	}

	return user, tokenPair, nil
}

func (s *Service) SignIn(ctx context.Context, input dto.SignInInput) (*model.User, *auth.TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, strings.ToLower(input.Email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apperror.New(apperror.CodeUnauthorized, "invalid email or password")
		}
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to load user", err)
	}

	if user.Password == nil || s.password.ComparePassword(*user.Password, input.Password) != nil {
		return nil, nil, apperror.New(apperror.CodeUnauthorized, "invalid email or password")
	}

	tokenPair, err := s.tokens.IssueTokenPair(user.ID, user.Email, user.Role, time.Now().UTC())
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to issue token pair", err)
	}

	return user, tokenPair, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*model.User, *auth.TokenPair, error) {
	claims, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.CodeUnauthorized, "invalid refresh token", err)
	}

	user, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apperror.New(apperror.CodeUnauthorized, "user not found")
		}
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to load user", err)
	}

	tokenPair, err := s.tokens.IssueTokenPair(user.ID, user.Email, user.Role, time.Now().UTC())
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.CodeInternal, "failed to issue token pair", err)
	}

	return user, tokenPair, nil
}

func pointer[T any](value T) *T {
	return &value
}
