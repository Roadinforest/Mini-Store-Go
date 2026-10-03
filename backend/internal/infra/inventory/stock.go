// Package inventory keeps physical stock and active reservations in PostgreSQL.
// Unpaid orders reserve stock for ReservationTTL; expiry is evaluated at read time,
// so it needs neither a Redis write nor a background release/compensation job.
package inventory

import (
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-store-go/backend/internal/apperror"
	"mini-store-go/backend/internal/domain/model"
)

const ReservationTTL = 15 * time.Minute

// Available uses one database snapshot and the database clock. excludeOrderID
// lets a payment consume its own reservation without consuming another order's.
func Available(db *gorm.DB, productID, excludeOrderID string) (int, error) {
	var result struct{ Available int }
	err := db.Model(&model.Product{}).Select(`stock - COALESCE((
  SELECT SUM(i.qty) FROM "OrderItem" i JOIN "Order" o ON o.id = i."orderId"
  WHERE i."productId" = "Product".id AND o."isPaid" = false
    AND o."createdAt" > statement_timestamp() - (? * interval '1 second')
    AND o.id::text <> ?
 ), 0) AS available`, ReservationTTL.Seconds(), excludeOrderID).
		Where("id = ?", productID).Take(&result).Error
	return result.Available, err
}

// Check locks every product in a stable order. Call inside the transaction that
// creates the order or deducts physical stock, keeping locks until commit.
func Check(tx *gorm.DB, quantities map[string]int, excludeOrderID string) error {
	ids := make([]string, 0, len(quantities))
	for id, qty := range quantities {
		if qty <= 0 {
			return apperror.New(apperror.CodeBadRequest, "quantity must be positive")
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var products []model.Product
	if err := tx.Select("id").Where("id IN ?", ids).Order("id").Clauses(clause.Locking{Strength: "UPDATE"}).Find(&products).Error; err != nil {
		return err
	}
	if len(products) != len(ids) {
		return apperror.New(apperror.CodeNotFound, "product not found")
	}
	for _, id := range ids {
		available, err := Available(tx, id, excludeOrderID)
		if err != nil {
			return err
		}
		if available < quantities[id] {
			return apperror.New(apperror.CodeOutOfStock, "not enough stock")
		}
	}
	return nil
}
