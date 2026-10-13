package customers

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides customer data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *gorm.DB { return r.db }

func (r *Repo) Create(c *Customer) error { return r.db.Create(c).Error }

func (r *Repo) Get(storeID, id uuid.UUID) (*Customer, error) {
	var c Customer
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) GetByID(id uuid.UUID) (*Customer, error) {
	var c Customer
	if err := r.db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// FindByPhone looks up a customer by phone within a store.
func (r *Repo) FindByPhone(storeID uuid.UUID, phone string) (*Customer, error) {
	var c Customer
	err := r.db.Where("store_id = ? AND phone = ?", storeID, phone).First(&c).Error
	return &c, err
}

func (r *Repo) Update(c *Customer) error { return r.db.Save(c).Error }

// Filter constrains customer list queries.
type Filter struct {
	StoreID   uuid.UUID
	Search    string
	Segment   string
	HasCredit *bool
	Page      int
	PageSize  int
}

func (r *Repo) List(f Filter) ([]Customer, int64, error) {
	q := r.db.Model(&Customer{}).Where("store_id = ? AND is_active = true", f.StoreID)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(lower(name) LIKE lower(?) OR phone LIKE ?)", like, like)
	}
	if f.Segment != "" {
		q = q.Where("segment = ?", f.Segment)
	}
	if f.HasCredit != nil && *f.HasCredit {
		q = q.Where("open_credit > 0")
	}
	if f.HasCredit != nil && !*f.HasCredit {
		q = q.Where("open_credit = 0")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("total_spent desc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var items []Customer
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CreditCustomers returns active customers with open credit, with their
// outstanding balance and overdue tabs.
func (r *Repo) CreditCustomers(storeID uuid.UUID, onlyOverdue bool) ([]Customer, error) {
	q := r.db.Where("store_id = ? AND is_active = true AND open_credit > 0", storeID)
	if onlyOverdue {
		q = q.Where("EXISTS (SELECT 1 FROM credit_tabs ct WHERE ct.customer_id = customers.id AND ct.status IN ('open','partially_paid') AND ct.due_date < ?)", time.Now())
	}
	var items []Customer
	if err := q.Order("open_credit desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
