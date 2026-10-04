package gormrepo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
	"mini-store-go/backend/internal/repository"
)

type cartRepository struct {
	db *gorm.DB
}

func NewCartRepository(db *gorm.DB) repository.CartRepository {
	return &cartRepository{db: db}
}

func (r *cartRepository) GetByID(ctx context.Context, id string) (*model.Cart, error) {
	var cart model.Cart
	if err := cartQuery(r.db.WithContext(ctx)).First(&cart, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) GetByUserID(ctx context.Context, userID string) (*model.Cart, error) {
	var cart model.Cart
	if err := cartQuery(r.db.WithContext(ctx)).First(&cart, `"userId" = ?`, userID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) GetBySessionCartID(ctx context.Context, sessionCartID string) (*model.Cart, error) {
	var cart model.Cart
	if err := cartQuery(r.db.WithContext(ctx)).First(&cart, `"sessionCartId" = ?`, sessionCartID).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *cartRepository) Create(ctx context.Context, cart *model.Cart) error {
	return r.db.WithContext(ctx).Create(cart).Error
}

func (r *cartRepository) Update(ctx context.Context, cart *model.Cart) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.Cart
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", cart.ID).Error; err != nil {
			return err
		}
		return replaceCart(tx, cart)
	})
}

func (r *cartRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.Cart{}, "id = ?", id).Error
}

func cartQuery(db *gorm.DB) *gorm.DB {
	return db.Preload("Items", func(tx *gorm.DB) *gorm.DB { return tx.Order(`"createdAt" ASC`).Order(`"productId" ASC`) })
}

func (r *cartRepository) WithCurrent(ctx context.Context, sessionID string, userID *string, create bool, change func(*model.Cart) error) (*model.Cart, error) {
	if userID != nil && *userID == "" {
		userID = nil
	}
	cart := &model.Cart{SessionCartID: sessionID, UserID: userID, Items: []model.CartItem{}}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The user lock also serializes creation from different sessions for one user.
		if userID != nil {
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&user, "id = ?", *userID).Error; err != nil {
				return err
			}
		}
		load := func() error {
			locked := tx.Clauses(clause.Locking{Strength: "UPDATE"})
			if userID != nil {
				err := locked.Where(`"userId" = ?`, *userID).First(cart).Error
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where(`"sessionCartId" = ? AND ("userId" IS NULL OR "userId" = ?)`, sessionID, userID).First(cart).Error
		}
		err := load()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cart = &model.Cart{SessionCartID: sessionID, UserID: userID, Items: []model.CartItem{}}
			if !create {
				return nil
			}
			cart.ID = uuid.NewString()
			if err := tx.Omit("Items", "User").Clauses(clause.OnConflict{DoNothing: true}).Create(cart).Error; err != nil {
				return err
			}
			// Re-read after INSERT to handle concurrent creation for this session.
			cart.ID = ""
			if err := load(); err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return apperror.New(apperror.CodeConflict, "cart session belongs to another user")
				}
				return err
			}
		} else if err != nil {
			return err
		}
		if err := tx.Where(`"cartId" = ?`, cart.ID).Order(`"createdAt" ASC`).Order(`"productId" ASC`).Find(&cart.Items).Error; err != nil {
			return err
		}
		if userID != nil && cart.UserID == nil {
			cart.UserID = userID
			if err := tx.Model(cart).Update("userId", userID).Error; err != nil {
				return err
			}
		}
		if change == nil {
			return nil
		}
		if err := change(cart); err != nil {
			return err
		}
		return replaceCart(tx, cart)
	})
	return cart, err
}

func replaceCart(tx *gorm.DB, cart *model.Cart) error {
	if err := tx.Omit("Items", "User").Save(cart).Error; err != nil {
		return err
	}
	if err := tx.Where(`"cartId" = ?`, cart.ID).Delete(&model.CartItem{}).Error; err != nil {
		return err
	}
	if len(cart.Items) == 0 {
		return nil
	}
	for i := range cart.Items {
		cart.Items[i].CartID = cart.ID
	}
	return tx.Omit("Product").Create(&cart.Items).Error
}
