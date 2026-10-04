package inventorylock

import "gorm.io/gorm"

// Inventory writers and Redis snapshot readers share this transaction-scoped
// lock. The small-store implementation deliberately serializes stock changes.
func Acquire(tx *gorm.DB) error {
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(current_schema() || ':mini-store-inventory', 0))`).Error
}
