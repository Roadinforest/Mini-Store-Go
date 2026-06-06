package database

import (
	"fmt"

	"gorm.io/gorm"

	"mini-store-go/backend/internal/domain/model"
)

func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.AutoMigrate(model.All()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
