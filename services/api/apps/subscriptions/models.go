// Package subscriptions implements the billing tiers and per-organisation
// subscription lifecycle (trial, pending payment, active, expired) for ZiadaPOS.
package subscriptions

import (
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Subscription statuses.
const (
	StatusTrial    = "trial"
	StatusActive   = "active"
	StatusExpired  = "expired"
	StatusCanceled = "cancelled"
	StatusPending  = "pending_payment"
)

// ValidStatuses lists accepted subscription statuses.
var ValidStatuses = []string{StatusTrial, StatusActive, StatusExpired, StatusCanceled, StatusPending}

// DefaultTrialFee is the one-off trial fee in TZS charged during onboarding.
const DefaultTrialFee = 10000

// DefaultExtraStorePrice is the fallback per-extra-store monthly price in TZS.
const DefaultExtraStorePrice = 12000

// DefaultIncludedStores is how many stores are bundled when no plan exists.
const DefaultIncludedStores = 3

// SubscriptionPlan is an admin-configurable pricing tier.
type SubscriptionPlan struct {
	models.BaseModel
	Name                     string `gorm:"size:100;not null" json:"name"`
	Slug                     string `gorm:"size:50;uniqueIndex;not null" json:"slug"`
	Description              string `gorm:"type:text" json:"description"`
	PricePerMonth            int64  `gorm:"not null" json:"price_per_month"`
	DurationMonths           int    `gorm:"default:1" json:"duration_months"`
	IncludedStores           int    `gorm:"default:3" json:"included_stores"`
	ExtraStorePricePerMonth  int64  `gorm:"default:12000" json:"extra_store_price_per_month"`
	IsActive                 bool   `gorm:"default:true;index" json:"is_active"`
	SortOrder                int    `gorm:"default:0" json:"sort_order"`
}

// TotalPrice is the amount billed per payment cycle (price x months).
func (p *SubscriptionPlan) TotalPrice() int64 { return p.PricePerMonth * int64(p.DurationMonths) }

// ExtraStorePriceTotal is the extra-store cost across the whole cycle.
func (p *SubscriptionPlan) ExtraStorePriceTotal() int64 {
	return p.ExtraStorePricePerMonth * int64(p.DurationMonths)
}

// Subscription tracks one billing period for an Organisation.
type Subscription struct {
	models.BaseModel
	OrganisationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	PlanID         *uuid.UUID `gorm:"type:uuid;index" json:"plan_id,omitempty"`
	Plan           *SubscriptionPlan `gorm:"constraint:OnDelete:SET NULL" json:"plan_detail,omitempty"`

	Status    string     `gorm:"size:20;default:pending_payment;index" json:"status"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`

	IsTrial    bool  `gorm:"default:false" json:"is_trial"`
	TrialFee   int64 `gorm:"default:10000" json:"trial_fee"`
	ExtraStores int   `gorm:"default:0" json:"extra_stores"`

	AmountPaid       int64      `gorm:"default:0" json:"amount_paid"`
	PaymentReference string     `gorm:"size:200" json:"payment_reference"`
	PaymentDate      *time.Time `json:"payment_date,omitempty"`
	ActivatedBy      *uuid.UUID `gorm:"type:uuid;index" json:"activated_by,omitempty"`
	Notes            string     `gorm:"type:text" json:"notes"`
}

// IsActiveNow reports whether this subscription grants current access.
func (s *Subscription) IsActiveNow() bool {
	if s.Status != StatusTrial && s.Status != StatusActive {
		return false
	}
	if s.EndDate == nil {
		return true
	}
	return !s.EndDate.Before(time.Now())
}

// DaysRemaining is how many days are left until expiry (never negative).
func (s *Subscription) DaysRemaining() int {
	if s.EndDate == nil {
		return 0
	}
	now := time.Now()
	if s.EndDate.Before(now) {
		return 0
	}
	return int(s.EndDate.Sub(now).Hours()/24) + 1
}

// MaxStoresAllowed is the total stores permitted: plan quota + extra slots.
func (s *Subscription) MaxStoresAllowed() int {
	base := DefaultIncludedStores
	if s.Plan != nil {
		base = s.Plan.IncludedStores
	}
	return base + s.ExtraStores
}

// TotalAmountDue is what the organisation owes for this cycle.
func (s *Subscription) TotalAmountDue() int64 {
	if s.IsTrial {
		return s.TrialFee
	}
	if s.Plan == nil {
		return 0
	}
	return s.Plan.TotalPrice() + s.ExtraStoresPrice()
}

// ExtraStoresPrice is the additional cost of extra store slots this cycle.
func (s *Subscription) ExtraStoresPrice() int64 {
	if s.Plan == nil {
		return 0
	}
	return s.Plan.ExtraStorePricePerMonth * int64(s.Plan.DurationMonths) * int64(s.ExtraStores)
}
