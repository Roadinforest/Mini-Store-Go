package adminservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/dto"
	"mini-store-go/backend/internal/repository"
)

type Overview struct {
	OrderCount   int64
	ProductCount int64
	UserCount    int64
	TotalSales   decimal.Decimal
}

type Service struct {
	db    *gorm.DB
	users repository.UserRepository
}

func NewService(db *gorm.DB, users repository.UserRepository) *Service {
	return &Service{
		db:    db,
		users: users,
	}
}

func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	var overview Overview

	if err := s.db.WithContext(ctx).Model(&model.Order{}).Count(&overview.OrderCount).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to count orders", err)
	}
	if err := s.db.WithContext(ctx).Model(&model.Product{}).Count(&overview.ProductCount).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to count products", err)
	}
	if err := s.db.WithContext(ctx).Model(&model.User{}).Count(&overview.UserCount).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to count users", err)
	}

	var totalSales struct {
		Amount decimal.Decimal `gorm:"column:amount"`
	}
	if err := s.db.WithContext(ctx).
		Model(&model.Order{}).
		Select(`COALESCE(SUM("totalPrice"), 0) AS amount`).
		Where(`"isPaid" = ?`, true).
		Scan(&totalSales).Error; err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to sum sales", err)
	}
	overview.TotalSales = totalSales.Amount

	return &overview, nil
}

func (s *Service) ListUsers(ctx context.Context, filter dto.UserListFilter) ([]model.User, dto.PageMeta, error) {
	filter.PageParams = filter.PageParams.Normalize(20)

	users, total, err := s.users.List(ctx, filter)
	if err != nil {
		return nil, dto.PageMeta{}, apperror.Wrap(apperror.CodeInternal, "failed to list users", err)
	}

	return users, dto.NewPageMeta(filter.Page, filter.Limit, total), nil
}

func (s *Service) UpdateUser(ctx context.Context, userID string, input dto.UpdateUserInput) (*model.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(apperror.CodeNotFound, "user not found")
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to load user", err)
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))
	existing, err := s.users.GetByEmail(ctx, normalizedEmail)
	if err == nil && existing.ID != user.ID {
		return nil, apperror.New(apperror.CodeConflict, "email already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to check email availability", err)
	}

	user.Name = strings.TrimSpace(input.Name)
	user.Email = normalizedEmail
	user.Role = input.Role
	user.UpdatedAt = time.Now().UTC()

	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to update user", err)
	}

	return user, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID, actorUserID string) error {
	if userID == actorUserID {
		return apperror.New(apperror.CodeBadRequest, "cannot delete current user")
	}

	if _, err := s.users.GetByID(ctx, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.New(apperror.CodeNotFound, "user not found")
		}
		return apperror.Wrap(apperror.CodeInternal, "failed to load user", err)
	}

	if err := s.users.Delete(ctx, userID); err != nil {
		return apperror.Wrap(apperror.CodeInternal, "failed to delete user", err)
	}

	return nil
}
