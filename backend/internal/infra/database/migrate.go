package database

import (
	_ "embed"
	"fmt"
	"gorm.io/gorm"
	"mini-store-go/backend/internal/domain/model"
)

//go:embed migrations/0001_normalize_commerce.up.sql
var normalizeCommerceSQL string

// AutoMigrate upgrades the supported legacy schema before creating current tables.
// It runs only when database.auto_migrate is explicitly enabled.
func AutoMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(784527109)").Error; err != nil {
			return err
		}
		if tx.Migrator().HasColumn("Cart", "items") {
			if err := tx.Exec(normalizeCommerceSQL).Error; err != nil {
				return fmt.Errorf("normalize commerce schema: %w", err)
			}
		}
		return tx.AutoMigrate(model.All()...)
	})
	if err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
