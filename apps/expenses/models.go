// Package expenses records store operating costs by category for reporting.
package expenses

import (
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Expense categories.
const (
	CatRent        = "Rent"
	CatUtilities   = "Utilities"
	CatSalaries    = "Salaries"
	CatSupplies    = "Supplies"
	CatTransport   = "Transport"
	CatMarketing   = "Marketing"
	CatMaintenance = "Maintenance"
	CatLicences    = "Licences & permits"
	CatInsurance   = "Insurance"
	CatOther       = "Other"
)

// ValidCategories lists accepted expense categories.
var ValidCategories = []string{
	CatRent, CatUtilities, CatSalaries, CatSupplies, CatTransport,
	CatMarketing, CatMaintenance, CatLicences, CatInsurance, CatOther,
}

// Payment methods used for expenses.
const (
	MethodCash   = "Cash"
	MethodMPesa  = "M-Pesa"
	MethodTigo   = "Tigo Pesa"
	MethodBank   = "Bank"
	MethodAirtel = "Airtel Money"
)

// ValidPaymentMethods lists accepted expense payment methods.
var ValidPaymentMethods = []string{MethodCash, MethodMPesa, MethodTigo, MethodBank, MethodAirtel}

// Expense is a store operating cost.
type Expense struct {
	models.BaseModel
	OrganisationID   uuid.UUID `gorm:"type:uuid;not null;index" json:"organisation_id"`
	StoreID          uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	Title            string    `gorm:"size:255;not null;index" json:"title"`
	Category         string    `gorm:"size:50;not null;default:Other;index" json:"category"`
	Amount           int64     `gorm:"not null" json:"amount"`
	PaymentMethod    string    `gorm:"size:20;not null;default:Cash;index" json:"payment_method"`
	PaymentReference string    `gorm:"size:100" json:"payment_reference"`
	Notes            string    `gorm:"type:text" json:"notes"`
	ReceiptURL       string    `gorm:"size:500" json:"receipt_url"`
	RecordedBy       uuid.UUID `gorm:"type:uuid;index" json:"recorded_by"`
}
