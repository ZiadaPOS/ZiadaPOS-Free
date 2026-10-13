package suppliers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
)

type ReorderSuggestion struct {
	ProductID         uuid.UUID  `json:"product_id"`
	ProductName       string     `json:"product_name"`
	SKU               string     `json:"sku"`
	UnitCost          int64      `json:"unit_cost"`
	SupplierID        *uuid.UUID `json:"supplier_id,omitempty"`
	SupplierName      string     `json:"supplier_name,omitempty"`
	SalesLast30Days   int        `json:"sales_last_30_days"`
	DailyVelocity     int        `json:"daily_velocity"`
	LeadTimeDays      int        `json:"lead_time_days"`
	SafetyStock       int        `json:"safety_stock"`
	CurrentStock      int        `json:"current_stock"`
	LeadTimeDemand    int        `json:"lead_time_demand"`
	SuggestedQuantity int        `json:"suggested_quantity"`
	Reason            string     `json:"reason"`
}

// ReorderSuggestions uses the foundation's deterministic 30-day sales,
// supplier lead-time, safety-stock and current-stock formula. For pharmacy,
// current stock means positive-quantity batches that have not expired.
func (s *Service) ReorderSuggestions(ctx context.Context, orgID, storeID uuid.UUID) ([]ReorderSuggestion, error) {
	var businessType string
	if err := s.repo.DB().WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType != "pharmacy" {
		return nil, apierr.Conflict("Reorder suggestions are available for pharmacy accounts")
	}
	var products []inventory.Product
	if err := s.repo.DB().WithContext(ctx).Where("store_id = ? AND is_active = true AND is_service = false", storeID).Order("name asc").Find(&products).Error; err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return []ReorderSuggestion{}, nil
	}
	ids := make([]uuid.UUID, len(products))
	supplierIDs := make([]uuid.UUID, 0)
	seenSuppliers := make(map[uuid.UUID]bool)
	for i, product := range products {
		ids[i] = product.ID
		if product.SupplierID != nil && !seenSuppliers[*product.SupplierID] {
			supplierIDs = append(supplierIDs, *product.SupplierID)
			seenSuppliers[*product.SupplierID] = true
		}
	}
	var sales []struct {
		ProductID       uuid.UUID `gorm:"column:product_id"`
		SalesLast30Days int       `gorm:"column:sales_last_30_days"`
	}
	cutoff := time.Now().AddDate(0, 0, -30)
	if err := s.repo.DB().WithContext(ctx).Table("transaction_lines").Select("transaction_lines.product_id, COALESCE(SUM(transaction_lines.qty), 0) AS sales_last_30_days").Joins("JOIN transactions ON transactions.id = transaction_lines.transaction_id").Where("transactions.store_id = ? AND transactions.status IN ? AND transactions.created_at >= ? AND transaction_lines.product_id IN ?", storeID, []string{"paid", "credit"}, cutoff, ids).Group("transaction_lines.product_id").Scan(&sales).Error; err != nil {
		return nil, err
	}
	salesByProduct := make(map[uuid.UUID]int, len(sales))
	for _, row := range sales {
		salesByProduct[row.ProductID] = row.SalesLast30Days
	}
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		location = time.FixedZone("EAT", 3*60*60)
	}
	today := time.Now().In(location)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)
	var stockRows []struct {
		ProductID     uuid.UUID `gorm:"column:product_id"`
		SellableStock int       `gorm:"column:sellable_stock"`
	}
	if err := s.repo.DB().WithContext(ctx).Table("medicine_batches").Select("product_id, COALESCE(SUM(quantity), 0) AS sellable_stock").Where("store_id = ? AND product_id IN ? AND stock_status = ? AND quantity > 0 AND expiry_date >= ?", storeID, ids, "available", today.Format("2006-01-02")).Group("product_id").Scan(&stockRows).Error; err != nil {
		return nil, err
	}
	stockByProduct := make(map[uuid.UUID]int, len(stockRows))
	for _, row := range stockRows {
		stockByProduct[row.ProductID] = row.SellableStock
	}
	suppliersByID := make(map[uuid.UUID]Supplier, len(supplierIDs))
	if len(supplierIDs) > 0 {
		var supplierRows []Supplier
		if err := s.repo.DB().WithContext(ctx).Where("store_id = ? AND id IN ?", storeID, supplierIDs).Find(&supplierRows).Error; err != nil {
			return nil, err
		}
		for _, supplier := range supplierRows {
			suppliersByID[supplier.ID] = supplier
		}
	}
	suggestions := make([]ReorderSuggestion, 0)
	for _, product := range products {
		leadTime := 7
		supplierName := ""
		if product.SupplierID != nil {
			if supplier, ok := suppliersByID[*product.SupplierID]; ok {
				if supplier.LeadTimeDays > 0 {
					leadTime = supplier.LeadTimeDays
				}
				supplierName = supplier.Name
			}
		}
		sold := salesByProduct[product.ID]
		current := stockByProduct[product.ID]
		velocity, demand, suggested := calculateReorder(sold, leadTime, product.ReorderPoint, current)
		if suggested <= 0 {
			continue
		}
		sku := ""
		if product.SKU != nil {
			sku = *product.SKU
		}
		suggestions = append(suggestions, ReorderSuggestion{
			ProductID: product.ID, ProductName: product.Name, SKU: sku,
			UnitCost:   product.CostPrice,
			SupplierID: product.SupplierID, SupplierName: supplierName,
			SalesLast30Days: sold, DailyVelocity: velocity, LeadTimeDays: leadTime,
			SafetyStock: product.ReorderPoint, CurrentStock: current, LeadTimeDemand: demand,
			SuggestedQuantity: suggested,
			Reason:            fmt.Sprintf("%d sold in 30 days / 30 × %d lead-time days + %d safety stock − %d sellable stock", sold, leadTime, product.ReorderPoint, current),
		})
	}
	return suggestions, nil
}

func calculateReorder(salesLast30Days, leadTimeDays, safetyStock, currentStock int) (dailyVelocity, leadTimeDemand, suggested int) {
	dailyVelocity = salesLast30Days / 30
	leadTimeDemand = dailyVelocity * leadTimeDays
	suggested = max(0, leadTimeDemand+safetyStock-currentStock)
	return
}
