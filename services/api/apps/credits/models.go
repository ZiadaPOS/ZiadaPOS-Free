// Package credits manages credit ("madeni") tabs, payments, reminders and
// staff notes tied to customers.
package credits

import (
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Credit tab statuses.
const (
	StatusOpen       = "open"
	StatusPartial    = "partially_paid"
	StatusSettled    = "settled"
	StatusWrittenOff = "written_off"
)

// Payment methods accepted against a balance.
const (
	MethodCash  = "Cash"
	MethodMPesa = "M-Pesa"
	MethodTigo  = "Tigo Pesa"
	MethodBank  = "Bank"
)

// ValidPaymentMethods lists accepted credit payment methods.
var ValidPaymentMethods = []string{MethodCash, MethodMPesa, MethodTigo, MethodBank}

// Message channels used for reminders.
const (
	KindWhatsApp = "whatsapp"
	KindCall     = "call"
	KindSMS      = "sms"

	DirectionIn  = "in"
	DirectionOut = "out"
)

// CreditTab is one credit sale a customer owes for.
type CreditTab struct {
	models.BaseModel
	CustomerID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"customer_id"`
	StoreID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"store_id"`
	TransactionID  *uuid.UUID `gorm:"type:uuid;uniqueIndex;index" json:"transaction_id,omitempty"`
	Amount         int64      `gorm:"not null" json:"amount"`
	AmountPaid     int64      `gorm:"not null;default:0" json:"amount_paid"`
	Status         string     `gorm:"size:20;not null;default:open;index" json:"status"`
	DueDate        *time.Time `gorm:"index" json:"due_date,omitempty"`
	WriteOffReason string     `gorm:"size:500" json:"write_off_reason"`
	CashierID      uuid.UUID  `gorm:"type:uuid;index" json:"cashier_id"`
	TillNumber     string     `gorm:"size:20;default:Till #1" json:"till_number"`
}

// Balance returns the outstanding amount on this tab.
func (t *CreditTab) Balance() int64 {
	b := t.Amount - t.AmountPaid
	if b < 0 {
		return 0
	}
	return b
}

// IsOverdue reports whether an open tab is past its due date.
func (t *CreditTab) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	if t.Status != StatusOpen && t.Status != StatusPartial {
		return false
	}
	return t.DueDate.Before(time.Now())
}

// CreditPayment is a lump-sum payment distributed across open tabs.
type CreditPayment struct {
	models.BaseModel
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	StoreID    uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	Amount     int64     `gorm:"not null" json:"amount"`
	Method     string    `gorm:"size:20;default:Cash" json:"method"`
	Reference  string    `gorm:"size:100" json:"reference"`
	CashierID  uuid.UUID `gorm:"type:uuid;index" json:"cashier_id"`
	Note       string    `gorm:"size:500" json:"note"`
}

// CreditMessage is a follow-up communication (reminder or reply).
type CreditMessage struct {
	models.BaseModel
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	StoreID    uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	Kind       string    `gorm:"size:20;default:whatsapp" json:"kind"`
	Direction  string    `gorm:"size:5;default:out" json:"direction"`
	Body       string    `gorm:"size:2000;not null" json:"body"`
	Who        string    `gorm:"size:100" json:"who"`
	SentBy     uuid.UUID `gorm:"type:uuid;index" json:"sent_by"`
}

// CreditNote is an internal staff note about a customer's credit.
type CreditNote struct {
	models.BaseModel
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	StoreID    uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	Body       string    `gorm:"size:2000;not null" json:"body"`
	By         uuid.UUID `gorm:"type:uuid;index" json:"by"`
}
