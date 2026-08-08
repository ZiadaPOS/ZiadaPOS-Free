package suppliers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides supplier data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *gorm.DB { return r.db }

// ── Supplier ─────────────────────────────────────────────────────────────────

func (r *Repo) Create(s *Supplier) error { return r.db.Create(s).Error }

func (r *Repo) Get(storeID, id uuid.UUID) (*Supplier, error) {
	var s Supplier
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) Update(s *Supplier) error { return r.db.Save(s).Error }

// SupplierFilter constrains supplier list queries.
type SupplierFilter struct {
	StoreID  uuid.UUID
	Search   string
	Status   string
	Page     int
	PageSize int
}

func (r *Repo) List(f SupplierFilter) ([]Supplier, int64, error) {
	q := r.db.Model(&Supplier{}).Where("store_id = ?", f.StoreID)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(lower(name) LIKE lower(?) OR lower(category) LIKE lower(?) OR lower(rep) LIKE lower(?) OR phone LIKE ? OR lower(email) LIKE lower(?))",
			like, like, like, like, like)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("name asc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var items []Supplier
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Balance snapshots for a batch of suppliers (used by the list response).
type supplierBalance struct {
	SupplierID uuid.UUID
	Invoiced   int64
	Paid       int64
}

func (r *Repo) OutstandingBySupplier(storeID uuid.UUID) (map[uuid.UUID]int64, error) {
	var rows []supplierBalance
	err := r.db.Model(&SupplierDelivery{}).
		Select("supplier_id, COALESCE(SUM(total_value),0) AS invoiced, 0 AS paid").
		Where("store_id = ?", storeID).Group("supplier_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		out[row.SupplierID] = row.Invoiced
	}
	var payments []supplierBalance
	err = r.db.Model(&SupplierPayment{}).
		Select("supplier_id, 0 AS invoiced, COALESCE(SUM(amount),0) AS paid").
		Where("store_id = ?", storeID).Group("supplier_id").Scan(&payments).Error
	if err != nil {
		return nil, err
	}
	for _, row := range payments {
		if b, ok := out[row.SupplierID]; ok {
			b -= row.Paid
			if b < 0 {
				b = 0
			}
			out[row.SupplierID] = b
		}
	}
	return out, nil
}

// SupplierStats is the KPI header strip for the suppliers list page.
type SupplierStats struct {
	TotalSuppliers   int64
	ActiveCount      int64
	PendingInvoices  int64
	TotalOutstanding int64
}

func (r *Repo) Stats(storeID uuid.UUID) (*SupplierStats, error) {
	var s SupplierStats
	if err := r.db.Model(&Supplier{}).Where("store_id = ?", storeID).Count(&s.TotalSuppliers).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&Supplier{}).Where("store_id = ? AND status = ?", storeID, StatusActive).Count(&s.ActiveCount).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&SupplierDelivery{}).Where("store_id = ? AND is_paid = ?", storeID, false).Count(&s.PendingInvoices).Error; err != nil {
		return nil, err
	}
	var invoiced, paid int64
	if err := r.db.Model(&SupplierDelivery{}).Where("store_id = ?", storeID).Select("COALESCE(SUM(total_value),0)").Scan(&invoiced).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&SupplierPayment{}).Where("store_id = ?", storeID).Select("COALESCE(SUM(amount),0)").Scan(&paid).Error; err != nil {
		return nil, err
	}
	s.TotalOutstanding = invoiced - paid
	if s.TotalOutstanding < 0 {
		s.TotalOutstanding = 0
	}
	return &s, nil
}

// ── Deliveries ───────────────────────────────────────────────────────────────

func (r *Repo) CreateDelivery(d *SupplierDelivery) error { return r.db.Create(d).Error }

func (r *Repo) GetDelivery(storeID, id uuid.UUID) (*SupplierDelivery, error) {
	var d SupplierDelivery
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repo) UpdateDelivery(d *SupplierDelivery) error { return r.db.Save(d).Error }

func (r *Repo) DeleteDelivery(storeID, id uuid.UUID) error {
	return r.db.Where("id = ? AND store_id = ?", id, storeID).Delete(&SupplierDelivery{}).Error
}

// DeliveryFilter constrains delivery list queries.
type DeliveryFilter struct {
	StoreID    uuid.UUID
	SupplierID uuid.UUID
	Unpaid     bool
	Page       int
	PageSize   int
}

func (r *Repo) ListDeliveries(f DeliveryFilter) ([]SupplierDelivery, int64, error) {
	q := r.db.Model(&SupplierDelivery{}).Where("store_id = ?", f.StoreID)
	if f.SupplierID != uuid.Nil {
		q = q.Where("supplier_id = ?", f.SupplierID)
	}
	if f.Unpaid {
		q = q.Where("is_paid = ?", false)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("delivery_date desc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var items []SupplierDelivery
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ── Payments ─────────────────────────────────────────────────────────────────

func (r *Repo) CreatePayment(p *SupplierPayment) error { return r.db.Create(p).Error }

func (r *Repo) CreatePaymentTx(tx *gorm.DB, p *SupplierPayment) error { return tx.Create(p).Error }

// SumPaymentsForDelivery returns the total paid toward one delivery.
func (r *Repo) SumPaymentsForDelivery(tx *gorm.DB, deliveryID uuid.UUID) (int64, error) {
	var total int64
	err := tx.Model(&SupplierPayment{}).Where("delivery_id = ?", deliveryID).
		Select("COALESCE(SUM(amount),0)").Scan(&total).Error
	return total, err
}

func (r *Repo) UpdateDeliveryTx(tx *gorm.DB, d *SupplierDelivery) error { return tx.Save(d).Error }

// PaymentFilter constrains payment list queries.
type PaymentFilter struct {
	StoreID    uuid.UUID
	SupplierID uuid.UUID
	Page       int
	PageSize   int
}

func (r *Repo) ListPayments(f PaymentFilter) ([]SupplierPayment, int64, error) {
	q := r.db.Model(&SupplierPayment{}).Where("store_id = ?", f.StoreID)
	if f.SupplierID != uuid.Nil {
		q = q.Where("supplier_id = ?", f.SupplierID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("payment_date desc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var items []SupplierPayment
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// WithTx runs a closure in a transaction.
func (r *Repo) WithTx(fn func(tx *gorm.DB) error) error { return r.db.Transaction(fn) }
