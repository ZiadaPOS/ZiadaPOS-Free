package reports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Open credit tab statuses (mirror of the credits app).
const (
	StatusOpen    = "open"
	StatusPartial = "partially_paid"
)

// Repo provides report data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// taxDay is one day of gross sales, VAT collected and refunds.
type taxDay struct {
	Day     time.Time `gorm:"column:day"`
	Gross   int64     `gorm:"column:gross"`
	Tax     int64     `gorm:"column:tax"`
	Refunds int64     `gorm:"column:refunds"`
}

// TaxDaily returns per-day tax aggregates for paid + refunded tickets (EAT days).
func (r *Repo) TaxDaily(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]taxDay, error) {
	var rows []taxDay
	err := r.db.Table("transactions").
		Select(`
			(created_at + interval '3 hours')::date AS day,
			COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS gross,
			COALESCE(SUM(tax_amount) FILTER (WHERE status = 'paid'), 0) AS tax,
			COALESCE(SUM(total) FILTER (WHERE status = 'refunded'), 0) AS refunds`).
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("day").Order("day asc").Scan(&rows).Error
	return rows, err
}

// inventoryRow is one product's valuation snapshot.
type inventoryRow struct {
	ProductID uuid.UUID `gorm:"column:product_id"`
	Name      string    `gorm:"column:name"`
	SKU       string    `gorm:"column:sku"`
	Category  string    `gorm:"column:category"`
	Stock     int       `gorm:"column:stock"`
	CostPrice int64     `gorm:"column:cost_price"`
	SellPrice int64     `gorm:"column:selling_price"`
	IsService bool      `gorm:"column:is_service"`
}

// InventoryValuation returns active products with current stock/prices/category.
func (r *Repo) InventoryValuation(ctx context.Context, storeID uuid.UUID) ([]inventoryRow, error) {
	var rows []inventoryRow
	err := r.db.Table("products AS p").
		Select(`
			p.id AS product_id,
			p.name,
			COALESCE(p.sku, '') AS sku,
			COALESCE(c.name, 'Uncategorised') AS category,
			p.stock,
			p.cost_price,
			p.selling_price,
			p.is_service`).
		Joins("LEFT JOIN categories AS c ON c.id = p.category_id").
		Where("p.store_id = ? AND p.is_active = true AND p.is_service = false", storeID).
		Order("c.sort_order, c.name, p.name").
		Scan(&rows).Error
	return rows, err
}

// creditTabRow is an open credit tab with customer contact details.
type creditTabRow struct {
	TabID      uuid.UUID  `gorm:"column:tab_id"`
	CustomerID uuid.UUID  `gorm:"column:customer_id"`
	Name       string     `gorm:"column:name"`
	Phone      string     `gorm:"column:phone"`
	Amount     int64      `gorm:"column:amount"`
	AmountPaid int64      `gorm:"column:amount_paid"`
	DueDate    *time.Time `gorm:"column:due_date"`
}

// OpenCreditTabs returns all open/partial tabs for the aged-debtors report.
func (r *Repo) OpenCreditTabs(ctx context.Context, storeID uuid.UUID) ([]creditTabRow, error) {
	var rows []creditTabRow
	err := r.db.Table("credit_tabs AS ct").
		Select(`
			ct.id AS tab_id,
			ct.customer_id,
			COALESCE(c.name, 'Unknown') AS name,
			COALESCE(c.phone, '') AS phone,
			ct.amount,
			ct.amount_paid,
			ct.due_date`).
		Joins("JOIN customers AS c ON c.id = ct.customer_id").
		Where("ct.store_id = ? AND ct.status IN ?", storeID, []string{StatusOpen, StatusPartial}).
		Order("c.name, ct.due_date").
		Scan(&rows).Error
	return rows, err
}

// ── Export audit trail ───────────────────────────────────────────────────────

// storeName returns the store's display name.
func (r *Repo) storeName(ctx context.Context, storeID uuid.UUID) (string, error) {
	var name string
	err := r.db.Table("stores").Where("id = ?", storeID).Pluck("name", &name).Error
	return name, err
}

// CreateExport records a generated export.
func (r *Repo) CreateExport(ctx context.Context, e *ReportExport) error {
	return r.db.Create(e).Error
}

// ListExports returns the store's export history, newest first.
func (r *Repo) ListExports(ctx context.Context, storeID uuid.UUID, reportType, format string, limit int) ([]ReportExport, error) {
	q := r.db.Where("store_id = ?", storeID)
	if reportType != "" {
		q = q.Where("report_type = ?", reportType)
	}
	if format != "" {
		q = q.Where("format = ?", format)
	}
	var rows []ReportExport
	err := q.Order("created_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

// GetExport returns one export scoped to the store.
func (r *Repo) GetExport(ctx context.Context, storeID, id uuid.UUID) (*ReportExport, error) {
	var e ReportExport
	err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// CountExports returns the total export count for a store.
func (r *Repo) CountExports(ctx context.Context, storeID uuid.UUID, reportType, format string) (int64, error) {
	q := r.db.Model(&ReportExport{}).Where("store_id = ?", storeID)
	if reportType != "" {
		q = q.Where("report_type = ?", reportType)
	}
	if format != "" {
		q = q.Where("format = ?", format)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

// ── Scheduled reports ────────────────────────────────────────────────────────

// CreateScheduled inserts a new schedule.
func (r *Repo) CreateScheduled(ctx context.Context, s *ScheduledReport) error {
	return r.db.Create(s).Error
}

// ListScheduled returns all schedules for a store, newest first.
func (r *Repo) ListScheduled(ctx context.Context, storeID uuid.UUID) ([]ScheduledReport, error) {
	var rows []ScheduledReport
	err := r.db.Where("store_id = ?", storeID).Order("created_at desc").Find(&rows).Error
	return rows, err
}

// GetScheduled returns one schedule scoped to the store.
func (r *Repo) GetScheduled(ctx context.Context, storeID, id uuid.UUID) (*ScheduledReport, error) {
	var s ScheduledReport
	err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// UpdateScheduled persists changes to a schedule.
func (r *Repo) UpdateScheduled(ctx context.Context, s *ScheduledReport) error {
	return r.db.Save(s).Error
}

// DeleteScheduled removes a schedule.
func (r *Repo) DeleteScheduled(ctx context.Context, id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&ScheduledReport{}).Error
}

// CountActiveScheduled returns the number of enabled schedules for a store.
func (r *Repo) CountActiveScheduled(ctx context.Context, storeID uuid.UUID) (int, error) {
	var n int64
	err := r.db.Model(&ScheduledReport{}).
		Where("store_id = ? AND is_enabled = true", storeID).
		Count(&n).Error
	return int(n), err
}

// DueScheduled returns all enabled schedules whose next_send_at has passed.
func (r *Repo) DueScheduled(ctx context.Context, now time.Time) ([]ScheduledReport, error) {
	var rows []ScheduledReport
	err := r.db.
		Where("is_enabled = true AND (next_send_at IS NULL OR next_send_at <= ?)", now).
		Find(&rows).Error
	return rows, err
}
