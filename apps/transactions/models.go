// Package transactions implements POS sales, refunds and KPIs.
package transactions

import (
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Payment methods supported at the counter.
const (
	MethodCash    = "Cash"
	MethodMPesa   = "M-Pesa"
	MethodTigo    = "Tigo Pesa"
	MethodBank    = "Bank"
	MethodCredit  = "Credit"
)

// Transaction statuses.
const (
	StatusPaid     = "paid"
	StatusCredit   = "credit"
	StatusRefunded = "refunded"
	StatusVoid     = "void"
)

// ValidMethods lists every supported payment method.
var ValidMethods = []string{MethodCash, MethodMPesa, MethodTigo, MethodBank, MethodCredit}

// Transaction is one completed sale ("ticket") at the point of sale.
type Transaction struct {
	models.BaseModel
	OrganisationID   uuid.UUID `gorm:"type:uuid;not null;index" json:"organisation_id"`
	StoreID          uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_txn_number_store,priority:1" json:"store_id"`
	TxnNumber        string    `gorm:"size:30;not null;uniqueIndex:idx_txn_number_store,priority:2" json:"txn_number"`
	PaymentMethod    string    `gorm:"size:20;not null;index" json:"payment_method"`
	PaymentReference string    `gorm:"size:100" json:"payment_reference"`
	Status           string    `gorm:"size:20;not null;default:paid;index" json:"status"`
	Subtotal         int64     `gorm:"not null;default:0" json:"subtotal"`
	DiscountPct      float64   `gorm:"not null;default:0" json:"discount_pct"`
	DiscountAmount   int64     `gorm:"not null;default:0" json:"discount_amount"`
	TaxAmount        int64     `gorm:"not null;default:0" json:"tax_amount"`
	Total            int64     `gorm:"not null;default:0" json:"total"`
	CostTotal        int64     `gorm:"not null;default:0" json:"cost_total"`
	Profit           int64     `gorm:"not null;default:0" json:"profit"`
	CashierID        uuid.UUID `gorm:"type:uuid;not null;index" json:"cashier_id"`
	TillNumber       string    `gorm:"size:20;default:Till #1" json:"till_number"`
	CustomerName     string    `gorm:"size:200;default:Walk-in;index" json:"customer_name"`
	CustomerPhone    string    `gorm:"size:30" json:"customer_phone"`
	CustomerID       *uuid.UUID `gorm:"type:uuid;index" json:"customer_id,omitempty"`
	Channel          string    `gorm:"size:20;default:counter" json:"channel"`
	Notes            string    `gorm:"size:1000" json:"notes"`

	Lines []TransactionLine `gorm:"foreignKey:TransactionID" json:"lines"`
}

// TransactionLine snapshots a product at the time of sale.
type TransactionLine struct {
	models.BaseModel
	TransactionID uuid.UUID  `gorm:"type:uuid;not null;index" json:"transaction_id"`
	ProductID     *uuid.UUID `gorm:"type:uuid;index" json:"product_id,omitempty"`
	ProductName   string     `gorm:"size:300;not null" json:"product_name"`
	ProductSKU    string     `gorm:"size:64" json:"product_sku"`
	UnitPrice     int64      `gorm:"not null" json:"unit_price"`
	UnitCost      int64      `gorm:"not null;default:0" json:"unit_cost"`
	Qty           int        `gorm:"not null" json:"qty"`
}

// TxnCounter holds the per-store sequential ticket counter.
type TxnCounter struct {
	StoreID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	LastNumber int       `gorm:"not null;default:0"`
}
