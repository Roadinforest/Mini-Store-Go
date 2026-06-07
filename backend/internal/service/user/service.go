package userservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/domain/valueobject"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

type Service struct {
	users repository.UserRepository
}

func NewService(users repository.UserRepository) *Service {
	return &Service{users: users}
}

func (s *Service) GetByID(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "user not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load user", err)
	}
	return user, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, input dto.UpdateProfileInput) (*model.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if existing, err := s.users.GetByEmail(ctx, strings.ToLower(input.Email)); err == nil && existing.ID != user.ID {
		return nil, apperror.New(apperror.CodeConflict, "email already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to check email availability", err)
	}

	user.Name = input.Name
	user.Email = strings.ToLower(input.Email)
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to update profile", err)
	}
	return user, nil
}

func (s *Service) UpdateAddress(ctx context.Context, userID string, input dto.UpdateAddressInput) (*model.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Address = valueobject.NewJSON(valueobject.ShippingAddress{
		FullName:      input.FullName,
		StreetAddress: input.StreetAddress,
		City:          input.City,
		PostalCode:    input.PostalCode,
		Country:       input.Country,
		Lat:           input.Lat,
		Lng:           input.Lng,
	})
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to update address", err)
	}
	return user, nil
}

func (s *Service) UpdatePaymentMethod(ctx context.Context, userID string, input dto.UpdatePaymentMethodInput) (*model.User, error) {
	user, err := s.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	method := input.Type
	user.PaymentMethod = &method
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to update payment method", err)
	}
	return user, nil
}
