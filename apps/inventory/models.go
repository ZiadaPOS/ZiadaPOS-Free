// Package inventory implements products, categories and stock adjustments
// for every ZiadaPOS store.
package inventory

import (
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Category groups products. Categories are organisation-scoped so all stores
// of a business share one product catalogue.
type Category struct {
	models.BaseModel
	OrganisationID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_org_category" json:"organisation_id"`
	Name           string    `gorm:"size:150;not null;uniqueIndex:idx_org_category" json:"name"`
	Description    string    `gorm:"size:500" json:"description"`
	SortOrder      int       `gorm:"default:0" json:"sort_order"`
	IsActive       bool      `gorm:"default:true;index" json:"is_active"`

	ProductCount int64 `gorm:"-" json:"product_count"`
}

// Product is a sellable item in a store's inventory.
type Product struct {
	models.BaseModel
	StoreID       uuid.UUID  `gorm:"type:uuid;not null;index;uniqueIndex:idx_store_sku,idx_store_barcode" json:"store_id"`
	CategoryID    *uuid.UUID `gorm:"type:uuid;index" json:"category_id,omitempty"`
	SupplierID    *uuid.UUID `gorm:"type:uuid;index" json:"supplier_id,omitempty"`
	Name          string     `gorm:"size:200;not null;index" json:"name"`
	SKU           *string    `gorm:"size:64;uniqueIndex:idx_store_sku" json:"sku"`
	Barcode       *string    `gorm:"size:64;uniqueIndex:idx_store_barcode" json:"barcode"`
	CostPrice     int64      `gorm:"not null;default:0" json:"cost_price"`    // TZS
	SellingPrice  int64      `gorm:"not null;default:0" json:"selling_price"` // TZS
	Stock         int        `gorm:"not null;default:0" json:"stock"`
	ReorderPoint  int        `gorm:"default:5" json:"reorder_point"`
	Image         string     `gorm:"size:500" json:"image"`
	TaxRate       float64    `gorm:"default:18" json:"tax_rate"`
	Description   string     `gorm:"size:1000" json:"description"`
	IsActive      bool       `gorm:"default:true;index" json:"is_active"`
	IsService     bool       `gorm:"default:false" json:"is_service"` // non-stocked item

	Category *Category `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category,omitempty"`
}

// IsLowStock reports whether the product is at or below its reorder point.
func (p *Product) IsLowStock() bool {
	return !p.IsService && p.Stock <= p.ReorderPoint
}

// StockAdjustment records manual stock changes with a reason trail.
type StockAdjustment struct {
	models.BaseModel
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	QtyChange int       `gorm:"not null" json:"qty_change"`
	Reason    string    `gorm:"size:50" json:"reason"`
	Note      string    `gorm:"size:500" json:"note"`
	CreatedBy uuid.UUID `gorm:"type:uuid;index" json:"created_by"`
}
