// Package suppliers manages vendors, their stock deliveries (invoices) and
// the payments made to settle them.
package suppliers

import (
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Supplier statuses.
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusOnHold   = "on_hold"
)

// ValidStatuses lists accepted supplier statuses.
var ValidStatuses = []string{StatusActive, StatusInactive, StatusOnHold}

// Payment terms in days.
const (
	Terms30 = 30
	Terms60 = 60
	Terms90 = 90
)

// ValidTerms lists accepted payment terms.
var ValidTerms = []int{Terms30, Terms60, Terms90}

// Delivery statuses.
const (
	DeliveryDelivered = "delivered"
	DeliveryPending   = "pending"
	DeliveryPartial   = "partial"
)

// ValidDeliveryStatuses lists accepted delivery statuses.
var ValidDeliveryStatuses = []string{DeliveryDelivered, DeliveryPending, DeliveryPartial}

// Payment methods used against supplier invoices.
const (
	MethodCash   = "cash"
	MethodBank   = "bank"
	MethodMobile = "mobile"
)

// ValidPaymentMethods lists accepted supplier payment methods.
var ValidPaymentMethods = []string{MethodCash, MethodBank, MethodMobile}

// Supplier is a vendor that supplies goods to a store.
type Supplier struct {
	models.BaseModel
	OrganisationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organisation_id"`
	StoreID        uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	Name           string    `gorm:"size:200;not null;index" json:"name"`
	Category       string    `gorm:"size:100" json:"category"`
	Rep            string    `gorm:"size:200" json:"rep"`
	Phone          string    `gorm:"size:50" json:"phone"`
	Email          string    `gorm:"size:150" json:"email"`
	Address        string    `gorm:"type:text" json:"address"`
	PaymentTerms   int       `gorm:"not null;default:30" json:"payment_terms"`
	Status         string    `gorm:"size:20;not null;default:active;index" json:"status"`
	Notes          string    `gorm:"type:text" json:"notes"`
}

// IsActive reports whether the supplier is currently delivering.
func (s *Supplier) IsActive() bool { return s.Status == StatusActive }

// SupplierDelivery is one stock invoice received from a supplier.
type SupplierDelivery struct {
	models.BaseModel
	SupplierID     uuid.UUID `gorm:"type:uuid;not null;index" json:"supplier_id"`
	StoreID        uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	OrganisationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organisation_id"`
	InvoiceNumber  string    `gorm:"size:100" json:"invoice_number"`
	DeliveryDate   time.Time `gorm:"index" json:"delivery_date"`
	ItemsCount     int       `gorm:"not null;default:1" json:"items_count"`
	TotalValue     int64     `gorm:"not null" json:"total_value"`
	Status         string    `gorm:"size:20;not null;default:delivered" json:"status"`
	IsPaid         bool      `gorm:"not null;default:false;index" json:"is_paid"`
	RecordedBy     uuid.UUID `gorm:"type:uuid;index" json:"recorded_by"`
	Notes          string    `gorm:"type:text" json:"notes"`
}

// SupplierPayment is an immutable ledger entry of money paid to a supplier.
type SupplierPayment struct {
	models.BaseModel
	SupplierID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"supplier_id"`
	StoreID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"store_id"`
	OrganisationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	DeliveryID     *uuid.UUID `gorm:"type:uuid;index" json:"delivery_id,omitempty"`
	Amount         int64      `gorm:"not null" json:"amount"`
	PaymentDate    time.Time  `gorm:"index" json:"payment_date"`
	PaymentMethod  string     `gorm:"size:20;not null;default:cash" json:"payment_method"`
	Reference      string     `gorm:"size:200" json:"reference"`
	RecordedBy     uuid.UUID  `gorm:"type:uuid;index" json:"recorded_by"`
	Notes          string     `gorm:"type:text" json:"notes"`
}
