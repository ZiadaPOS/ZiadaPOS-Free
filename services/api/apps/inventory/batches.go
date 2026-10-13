package inventory

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/internal/apierr"
)

// ParseMedicineExpiryDate parses a lot expiry date and rejects stock that is
// already expired in the pharmacy's Tanzania business timezone.
func ParseMedicineExpiryDate(raw string, now time.Time) (time.Time, error) {
	expiryDate, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, apierr.BadRequest("Expiry date must use YYYY-MM-DD")
	}
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		location = time.FixedZone("EAT", 3*60*60)
	}
	if expiryDate.Format("2006-01-02") < now.In(location).Format("2006-01-02") {
		return time.Time{}, apierr.BadRequest("Cannot receive a medicine batch that is already expired")
	}
	return expiryDate, nil
}

type ReceiveBatchInput struct {
	BatchNumber            string `json:"batch_number"`
	Quantity               int    `json:"quantity"`
	ExpiryDate             string `json:"expiry_date"`
	CostPrice              int64  `json:"cost_price"`
	SupplierName           string `json:"supplier_name"`
	ReconcileExistingStock bool   `json:"reconcile_existing_stock"`
}

// ListBatches returns lots for a store, including expired and depleted lots
// for traceability. Supplying productID narrows the result to one product.
func (s *Service) ListBatches(ctx context.Context, storeID, productID uuid.UUID) ([]MedicineBatch, error) {
	query := s.repo.db.WithContext(ctx).Where("store_id = ?", storeID)
	if productID != uuid.Nil {
		query = query.Where("product_id = ?", productID)
	}
	var batches []MedicineBatch
	if err := query.Order("expiry_date asc, created_at asc").Find(&batches).Error; err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return batches, nil
	}
	productIDs := make([]uuid.UUID, 0, len(batches))
	seen := map[uuid.UUID]bool{}
	for _, batch := range batches {
		if !seen[batch.ProductID] {
			productIDs = append(productIDs, batch.ProductID)
			seen[batch.ProductID] = true
		}
	}
	var products []Product
	if err := s.repo.db.WithContext(ctx).Select("id", "name").Where("store_id = ? AND id IN ?", storeID, productIDs).Find(&products).Error; err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(products))
	for _, product := range products {
		names[product.ID] = product.Name
	}
	for i := range batches {
		batches[i].ProductName = names[batches[i].ProductID]
	}
	return batches, nil
}

// ReceiveBatch records a pharmacy lot and updates its product's stock in the
// same database transaction. The organisation must be registered as pharmacy.
func (s *Service) ReceiveBatch(ctx context.Context, storeID, productID, userID uuid.UUID, in ReceiveBatchInput) (*MedicineBatch, error) {
	in.BatchNumber = strings.TrimSpace(in.BatchNumber)
	in.SupplierName = strings.TrimSpace(in.SupplierName)
	expiryDate, err := ParseMedicineExpiryDate(in.ExpiryDate, time.Now())
	if in.BatchNumber == "" || in.Quantity <= 0 || err != nil || in.CostPrice < 0 {
		return nil, apierr.BadRequest("Batch number, positive quantity and expiry date are required")
	}
	orgID, err := s.repo.StoreOrgID(storeID)
	if err != nil {
		return nil, apierr.NotFound("Store not found")
	}
	var businessType string
	if err := s.repo.db.WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType != "pharmacy" {
		return nil, apierr.Conflict("Batch tracking is available for pharmacy accounts")
	}
	batch := &MedicineBatch{StoreID: storeID, ProductID: productID, BatchNumber: in.BatchNumber, Quantity: 0, ExpiryDate: expiryDate, CostPrice: in.CostPrice, SupplierName: in.SupplierName}
	err = s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ? AND is_active = true", productID, storeID).First(&product).Error; err != nil {
			return apierr.NotFound("Product not found")
		}
		if product.IsService {
			return apierr.BadRequest("Service items cannot have medicine batches")
		}
		if in.ReconcileExistingStock {
			var tracked int64
			if err := tx.Model(&MedicineBatch{}).Where("store_id = ? AND product_id = ?", storeID, productID).Select("COALESCE(SUM(quantity), 0)").Scan(&tracked).Error; err != nil {
				return err
			}
			untracked := int64(product.Stock) - tracked
			if untracked <= 0 || int64(in.Quantity) > untracked {
				return apierr.Conflict("The quantity exceeds stock that is not yet assigned to a batch")
			}
		}
		if err := tx.Create(batch).Error; err != nil {
			return err
		}
		if in.ReconcileExistingStock {
			if err := tx.Create(&MedicineStockMovement{StoreID: storeID, ProductID: productID, BatchID: batch.ID, Quantity: in.Quantity, MovementType: "opening_balance", ReferenceID: batch.ID.String(), Reason: "Assigned previously untracked stock to a batch", PerformedBy: userID}).Error; err != nil {
				return err
			}
			return ReconcilePharmacyStock(tx, storeID, productID, -in.Quantity)
		}
		if err := tx.Create(&MedicineStockMovement{StoreID: storeID, ProductID: productID, BatchID: batch.ID, Quantity: in.Quantity, MovementType: "purchase", ReferenceID: batch.ID.String(), Reason: "Received batch " + in.BatchNumber, PerformedBy: userID}).Error; err != nil {
			return err
		}
		if err := ReconcilePharmacyStock(tx, storeID, productID, 0); err != nil {
			return err
		}
		return tx.Create(&StockAdjustment{ProductID: productID, QtyChange: in.Quantity, Reason: ReasonRestock, Note: "Received batch " + in.BatchNumber, CreatedBy: userID}).Error
	})
	if err != nil {
		return nil, err
	}
	batch.Quantity = in.Quantity
	s.invalidate(&storeID, &orgID, productID)
	return batch, nil
}
