package inventory

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReconcilePharmacyStock rebuilds batch balances from the append-only movement
// ledger and the product's tracked-stock projection from those batch balances.
// untrackedDelta is negative when existing unassigned stock is converted into a
// tracked batch; normal purchases and sales pass zero.
func ReconcilePharmacyStock(tx *gorm.DB, storeID, productID uuid.UUID, untrackedDelta int) error {
	var product Product
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", productID, storeID).First(&product).Error; err != nil {
		return err
	}
	var previousTracked int64
	if err := tx.Model(&MedicineBatch{}).Where("store_id = ? AND product_id = ?", storeID, productID).Select("COALESCE(SUM(quantity), 0)").Scan(&previousTracked).Error; err != nil {
		return err
	}
	untracked := int64(product.Stock) - previousTracked + int64(untrackedDelta)
	if untracked < 0 {
		return fmt.Errorf("pharmacy stock invariant failed: product %s has less stock than its tracked batches", productID)
	}
	var balances []struct {
		BatchID  uuid.UUID
		Quantity int64
	}
	err := tx.Table("medicine_batches").
		Select("medicine_batches.id AS batch_id, COALESCE(SUM(medicine_stock_movements.quantity), 0) AS quantity").
		Joins("LEFT JOIN medicine_stock_movements ON medicine_stock_movements.batch_id = medicine_batches.id").
		Where("medicine_batches.store_id = ? AND medicine_batches.product_id = ?", storeID, productID).
		Group("medicine_batches.id").Find(&balances).Error
	if err != nil {
		return err
	}
	var tracked int64
	for _, balance := range balances {
		if balance.Quantity < 0 {
			return fmt.Errorf("pharmacy batch %s has a negative ledger balance", balance.BatchID)
		}
		if err := tx.Model(&MedicineBatch{}).Where("id = ?", balance.BatchID).Update("quantity", balance.Quantity).Error; err != nil {
			return err
		}
		tracked += balance.Quantity
	}
	return tx.Model(&Product{}).Where("id = ? AND store_id = ?", productID, storeID).Update("stock", untracked+tracked).Error
}
