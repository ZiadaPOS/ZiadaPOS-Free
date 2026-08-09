// Package stores manages org branches, their live KPIs and staff rosters.
package stores

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Store statuses.
const (
	StatusOpen   = "open"
	StatusClosed = "closed"
	StatusPaused = "paused"
)

// ValidStatuses lists accepted operational statuses.
var ValidStatuses = []string{StatusOpen, StatusClosed, StatusPaused}

// Store is one branch of an organisation.
type Store struct {
	ID             uuid.UUID `gorm:"column:id"`
	OrganisationID uuid.UUID `gorm:"column:organisation_id"`
	Name           string    `gorm:"column:name"`
	Code           string    `gorm:"column:code"`
	Phone          string    `gorm:"column:phone"`
	Email          string    `gorm:"column:email"`
	Area           string    `gorm:"column:area"`
	Address        string    `gorm:"column:address"`
	Latitude       float64   `gorm:"column:latitude"`
	Longitude      float64   `gorm:"column:longitude"`
	OpenHours      string    `gorm:"column:open_hours"`
	Color          string    `gorm:"column:color"`
	IsMainStore    bool      `gorm:"column:is_main_store"`
	Status         string    `gorm:"column:status"`
	IsActive       bool      `gorm:"column:is_active"`
	VATEnabled     bool      `gorm:"column:vat_enabled"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

// TableName maps to the shared stores table.
func (Store) TableName() string { return "stores" }

// StaffMember is one row from the users table scoped to a store.
type StaffMember struct {
	ID        uuid.UUID `gorm:"column:id"`
	FirstName string    `gorm:"column:first_name"`
	LastName  string    `gorm:"column:last_name"`
	Phone     string    `gorm:"column:phone"`
	Role      string    `gorm:"column:role"`
	AvatarHue int       `gorm:"column:avatar_hue"`
	StoreID   *uuid.UUID `gorm:"column:store_id"`
	IsActive  bool      `gorm:"column:is_active"`
}

// TableName maps to the shared users table.
func (StaffMember) TableName() string { return "users" }

// dayRow is one day of aggregated store revenue.
type dayRow struct {
	Day    time.Time `gorm:"column:day"`
	Total  int64     `gorm:"column:total"`
	Count  int64     `gorm:"column:count"`
}

// Repo provides store data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// ListStores returns active stores for an organisation, ordered by name.
func (r *Repo) ListStores(ctx context.Context, orgID uuid.UUID, status string) ([]Store, error) {
	var stores []Store
	q := r.db.Where("organisation_id = ? AND is_active = true", orgID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("name asc").Find(&stores).Error
	return stores, err
}

// GetStore returns one active store scoped to the organisation.
func (r *Repo) GetStore(ctx context.Context, orgID, id uuid.UUID) (*Store, error) {
	var s Store
	err := r.db.Where("id = ? AND organisation_id = ? AND is_active = true", id, orgID).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// OrgMaxStores returns the organisation's configured store limit.
func (r *Repo) OrgMaxStores(ctx context.Context, orgID uuid.UUID) (int, error) {
	var max int
	err := r.db.Table("organisations").
		Select("COALESCE(max_stores, 0)").
		Where("id = ?", orgID).
		Scan(&max).Error
	return max, err
}

// CountActiveStores counts active branches for the organisation.
func (r *Repo) CountActiveStores(ctx context.Context, orgID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.Table("stores").
		Where("organisation_id = ? AND is_active = true", orgID).
		Count(&n).Error
	return n, err
}

// CreateStore inserts a new branch.
func (r *Repo) CreateStore(ctx context.Context, s *Store) error {
	return r.db.Create(s).Error
}

// UpdateStore persists store fields.
func (r *Repo) UpdateStore(ctx context.Context, s *Store) error {
	return r.db.Model(&Store{}).
		Where("id = ? AND organisation_id = ?", s.ID, s.OrganisationID).
		Updates(map[string]interface{}{
			"name": s.Name, "code": s.Code, "phone": s.Phone, "email": s.Email,
			"area": s.Area, "address": s.Address, "latitude": s.Latitude,
			"longitude": s.Longitude, "open_hours": s.OpenHours, "color": s.Color,
			"vat_enabled": s.VATEnabled, "status": s.Status, "updated_at": time.Now(),
		}).Error
}

// SetStatus updates only the operational status.
func (r *Repo) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) error {
	return r.db.Model(&Store{}).
		Where("id = ? AND organisation_id = ?", id, orgID).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now()}).Error
}

// DeactivateStore soft-deletes a branch.
func (r *Repo) DeactivateStore(ctx context.Context, orgID, id uuid.UUID) error {
	return r.db.Model(&Store{}).
		Where("id = ? AND organisation_id = ?", id, orgID).
		Updates(map[string]interface{}{"is_active": false, "status": StatusPaused, "updated_at": time.Now()}).Error
}

// StoreStatus returns one store's operational status.
func (r *Repo) StoreStatus(ctx context.Context, storeID uuid.UUID) (string, error) {
	var status string
	err := r.db.Table("stores").
		Where("id = ?", storeID).
		Pluck("status", &status).Error
	return status, err
}

// TodayKPIs returns revenue + ticket count for today (EAT).
func (r *Repo) TodayKPIs(ctx context.Context, storeID uuid.UUID) (int64, int64, error) {
	start, end := eatToday()
	var a struct {
		Total int64 `gorm:"column:total"`
		Count int64 `gorm:"column:count"`
	}
	err := r.db.Table("transactions").
		Select("COALESCE(SUM(total),0) AS total, COUNT(*) AS count").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{"paid", "credit"}, start, end).
		Scan(&a).Error
	return a.Total, a.Count, err
}

// WeekDays returns revenue + txns grouped by EAT day within a date range.
func (r *Repo) WeekDays(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]dayRow, error) {
	var rows []dayRow
	err := r.db.Table("transactions").
		Select("(created_at + interval '3 hours')::date AS day, COALESCE(SUM(total),0) AS total, COUNT(*) AS count").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{"paid", "credit"}, from, to).
		Group("day").Scan(&rows).Error
	return rows, err
}

// ListStaff returns active staff assigned to a store.
func (r *Repo) ListStaff(ctx context.Context, storeID uuid.UUID) ([]StaffMember, error) {
	var staff []StaffMember
	err := r.db.
		Where("store_id = ? AND is_active = true", storeID).
		Order("role asc, first_name asc").
		Find(&staff).Error
	return staff, err
}

// CountStaff returns active staff at a store.
func (r *Repo) CountStaff(ctx context.Context, storeID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.Table("users").
		Where("store_id = ? AND is_active = true", storeID).
		Count(&n).Error
	return n, err
}

// Manager returns the first active owner/admin at a store.
func (r *Repo) Manager(ctx context.Context, storeID uuid.UUID) (*StaffMember, error) {
	var m StaffMember
	err := r.db.
		Where("store_id = ? AND is_active = true AND role IN ?", storeID, []string{"owner", "admin"}).
		Order("created_at asc").
		First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func eatToday() (time.Time, time.Time) {
	zone := time.FixedZone("EAT", 3*3600)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	return today.UTC(), today.AddDate(0, 0, 1).UTC()
}
