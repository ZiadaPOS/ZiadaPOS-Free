// Package inventory implements products, categories and stock adjustments
// for every ZiadaPOS store.
package inventory

import (
	"time"

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
	IsDefault      bool      `gorm:"not null;default:false" json:"is_default"`

	ProductCount int64 `gorm:"-" json:"product_count"`
}

// Unit is an organisation-scoped product/dispensing unit (for example tablet or bottle).
type Unit struct {
	models.BaseModel
	OrganisationID uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_org_unit" json:"organisation_id"`
	Name           string    `gorm:"size:100;not null;uniqueIndex:idx_org_unit" json:"name"`
	Description    string    `gorm:"size:500" json:"description"`
	IsActive       bool      `gorm:"default:true;index" json:"is_active"`
	IsDefault      bool      `gorm:"not null;default:false" json:"is_default"`
	UsageCount     int64     `gorm:"-" json:"usage_count"`
}

// MedicineBatch tracks one received lot. Quantity is reduced only by sales or
// recorded stock events, so expiry and lot traceability remain available.
type MedicineBatch struct {
	models.BaseModel
	StoreID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"store_id"`
	ProductID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"product_id"`
	SourceBatchID *uuid.UUID `gorm:"type:uuid;index" json:"source_batch_id,omitempty"`
	BatchNumber   string     `gorm:"size:100;not null" json:"batch_number"`
	Quantity      int        `gorm:"not null" json:"quantity"`
	ExpiryDate    time.Time  `gorm:"type:date;not null;index" json:"expiry_date"`
	CostPrice     int64      `gorm:"not null;default:0" json:"cost_price"`
	SupplierName  string     `gorm:"size:200" json:"supplier_name,omitempty"`
	StockStatus   string     `gorm:"size:20;not null;default:available;index" json:"stock_status"`
	ProductName   string     `gorm:"-" json:"product_name,omitempty"`
}

// MedicineBatchAllocation records which lots supplied a sold transaction line.
type MedicineBatchAllocation struct {
	models.BaseModel
	BatchID           uuid.UUID `gorm:"type:uuid;not null;index" json:"batch_id"`
	TransactionLineID uuid.UUID `gorm:"type:uuid;not null;index" json:"transaction_line_id"`
	Quantity          int       `gorm:"not null" json:"quantity"`
}

// MedicineStockMovement is the append-only audit ledger for a medicine lot.
// MedicineBatch.Quantity is its materialized on-hand projection.
type MedicineStockMovement struct {
	models.BaseModel
	StoreID      uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	ProductID    uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	BatchID      uuid.UUID `gorm:"type:uuid;not null;index" json:"batch_id"`
	Quantity     int       `gorm:"not null" json:"quantity"`
	MovementType string    `gorm:"size:30;not null;index" json:"movement_type"`
	ReferenceID  string    `gorm:"size:100;index" json:"reference_id,omitempty"`
	Reason       string    `gorm:"size:200" json:"reason,omitempty"`
	PerformedBy  uuid.UUID `gorm:"type:uuid;index" json:"performed_by"`
}

// Product is a sellable item in a store's inventory.
type Product struct {
	models.BaseModel
	StoreID          uuid.UUID  `gorm:"type:uuid;not null;index;uniqueIndex:idx_store_sku,idx_store_barcode" json:"store_id"`
	CategoryID       *uuid.UUID `gorm:"type:uuid;index" json:"category_id,omitempty"`
	SupplierID       *uuid.UUID `gorm:"type:uuid;index" json:"supplier_id,omitempty"`
	Name             string     `gorm:"size:200;not null;index" json:"name"`
	Unit             string     `gorm:"size:100;not null;default:''" json:"unit"`
	SKU              *string    `gorm:"size:64;uniqueIndex:idx_store_sku" json:"sku"`
	Barcode          *string    `gorm:"size:64;uniqueIndex:idx_store_barcode" json:"barcode"`
	CostPrice        int64      `gorm:"not null;default:0" json:"cost_price"`    // TZS
	SellingPrice     int64      `gorm:"not null;default:0" json:"selling_price"` // TZS
	WholesalePrice   *int64     `gorm:"default:null" json:"wholesale_price,omitempty"`
	WholesaleMinimum int        `gorm:"not null;default:1" json:"wholesale_minimum"`
	Stock            int        `gorm:"not null;default:0" json:"stock"`
	ReorderPoint     int        `gorm:"default:5" json:"reorder_point"`
	Image            string     `gorm:"size:500" json:"image"`
	TaxRate          float64    `gorm:"default:18" json:"tax_rate"`
	Description      string     `gorm:"size:1000" json:"description"`
	IsActive         bool       `gorm:"default:true;index" json:"is_active"`
	IsService        bool       `gorm:"default:false" json:"is_service"` // non-stocked item

	Category *Category `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category,omitempty"`
}

// IsLowStock reports whether the product is at or below its reorder point.
func (p *Product) IsLowStock() bool {
	return !p.IsService && p.Stock <= p.ReorderPoint
}

// StockAdjustment records manual stock changes with a reason trail.
type StockAdjustment struct {
	models.BaseModel
	ProductID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"product_id"`
	MedicineBatchID *uuid.UUID `gorm:"type:uuid;index" json:"medicine_batch_id,omitempty"`
	QtyChange       int        `gorm:"not null" json:"qty_change"`
	Reason          string     `gorm:"size:50" json:"reason"`
	Note            string     `gorm:"size:500" json:"note"`
	CreatedBy       uuid.UUID  `gorm:"type:uuid;index" json:"created_by"`
}
