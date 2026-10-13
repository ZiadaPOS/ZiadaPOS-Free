package subscriptions

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides subscription and plan data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// ── Plans ─────────────────────────────────────────────────────────────────────

func (r *Repo) ListPlans(includeInactive bool) ([]SubscriptionPlan, error) {
	q := r.db.Model(&SubscriptionPlan{})
	if !includeInactive {
		q = q.Where("is_active = ?", true)
	}
	var plans []SubscriptionPlan
	err := q.Order("sort_order asc, price_per_month asc").Find(&plans).Error
	return plans, err
}

func (r *Repo) GetPlan(id uuid.UUID) (*SubscriptionPlan, error) {
	var p SubscriptionPlan
	if err := r.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) CreatePlan(p *SubscriptionPlan) error { return r.db.Create(p).Error }

func (r *Repo) SavePlan(p *SubscriptionPlan) error { return r.db.Save(p).Error }

// ── Subscriptions ────────────────────────────────────────────────────────────

func (r *Repo) CreateSubscription(s *Subscription) error { return r.db.Create(s).Error }

func (r *Repo) SaveSubscription(s *Subscription) error { return r.db.Save(s).Error }

func (r *Repo) GetSubscription(id uuid.UUID) (*Subscription, error) {
	var s Subscription
	err := r.db.Preload("Plan").First(&s, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) LatestSubscription(orgID uuid.UUID) (*Subscription, error) {
	var s Subscription
	err := r.db.Preload("Plan").Where("organisation_id = ?", orgID).
		Order("created_at desc").First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) ListSubscriptions(status, orgID string, page, pageSize int) ([]Subscription, int64, error) {
	q := r.db.Model(&Subscription{}).Preload("Plan")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if orgID != "" {
		q = q.Where("organisation_id = ?", orgID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("created_at desc")
	if page > 0 && pageSize > 0 {
		q = q.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	var items []Subscription
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ── Organisation helpers ──────────────────────────────────────────────────────

// OrgRow is the minimal organisation projection used for limit logic.
type OrgRow struct {
	ID        string
	MaxStores int
	Plan      string
}

// OrgByID loads a lightweight organisation projection.
func (r *Repo) OrgByID(orgID uuid.UUID) (*OrgRow, error) {
	var row struct {
		ID        string `gorm:"column:id"`
		MaxStores int    `gorm:"column:max_stores"`
		Plan      string `gorm:"column:plan"`
	}
	err := r.db.Raw(
		"SELECT id, max_stores, plan FROM organisations WHERE id = ?", orgID,
	).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &OrgRow{ID: row.ID, MaxStores: row.MaxStores, Plan: row.Plan}, nil
}

// CountActiveStores counts active stores belonging to an organisation.
func (r *Repo) CountActiveStores(orgID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.Table("stores").Where("organisation_id = ? AND is_active = ?", orgID, true).
		Count(&n).Error
	return n, err
}

// OrgMaxStores reads the organisation's configured store limit.
func (r *Repo) OrgMaxStores(orgID uuid.UUID) (int, error) {
	var max int
	err := r.db.Table("organisations").
		Select("COALESCE(max_stores, 0)").
		Where("id = ?", orgID).
		Scan(&max).Error
	return max, err
}

// UpdateOrgMaxStores synchronises the organisation's store limit.
func (r *Repo) UpdateOrgMaxStores(orgID uuid.UUID, maxStores int) error {
	return r.db.Table("organisations").
		Where("id = ?", orgID).
		Update("max_stores", maxStores).Error
}

// UpdateOrgPlan sets the organisation plan label (free/pro).
func (r *Repo) UpdateOrgPlan(orgID uuid.UUID, plan string) error {
	return r.db.Table("organisations").
		Where("id = ?", orgID).
		Update("plan", plan).Error
}
