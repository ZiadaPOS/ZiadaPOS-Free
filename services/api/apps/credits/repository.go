package credits

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides credit data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *gorm.DB { return r.db }

func (r *Repo) CreateTab(tx *gorm.DB, t *CreditTab) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Create(t).Error
}

func (r *Repo) GetTab(storeID, id uuid.UUID) (*CreditTab, error) {
	var t CreditTab
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repo) GetTabByTxn(txnID uuid.UUID) (*CreditTab, error) {
	var t CreditTab
	if err := r.db.Where("transaction_id = ?", txnID).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// TabFilter constrains tab list queries.
type TabFilter struct {
	StoreID    uuid.UUID
	Status     string
	CustomerID uuid.UUID
	Overdue    bool
	Page       int
	PageSize   int
}

func (r *Repo) ListTabs(f TabFilter) ([]CreditTab, int64, error) {
	q := r.db.Model(&CreditTab{}).Where("store_id = ?", f.StoreID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.CustomerID != uuid.Nil {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if f.Overdue {
		q = q.Where("status IN ? AND due_date < ?", []string{StatusOpen, StatusPartial}, time.Now())
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("created_at desc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var items []CreditTab
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// OpenTabs returns a customer's open/partial tabs oldest-first for
// payment distribution.
func (r *Repo) OpenTabs(tx *gorm.DB, customerID uuid.UUID) ([]CreditTab, error) {
	var tabs []CreditTab
	err := tx.Where("customer_id = ? AND status IN ?", customerID, []string{StatusOpen, StatusPartial}).
		Order("created_at asc").Find(&tabs).Error
	return tabs, err
}

func (r *Repo) UpdateTab(tx *gorm.DB, t *CreditTab) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Save(t).Error
}

func (r *Repo) CreatePayment(tx *gorm.DB, p *CreditPayment) error {
	if tx == nil {
		tx = r.db
	}
	return tx.Create(p).Error
}

func (r *Repo) ListPayments(storeID, customerID uuid.UUID, limit int) ([]CreditPayment, error) {
	q := r.db.Where("store_id = ?", storeID)
	if customerID != uuid.Nil {
		q = q.Where("customer_id = ?", customerID)
	}
	var items []CreditPayment
	err := q.Order("created_at desc").Limit(limit).Find(&items).Error
	return items, err
}

func (r *Repo) CreateMessage(m *CreditMessage) error { return r.db.Create(m).Error }

func (r *Repo) ListMessages(storeID, customerID uuid.UUID, limit int) ([]CreditMessage, error) {
	var items []CreditMessage
	err := r.db.Where("store_id = ? AND customer_id = ?", storeID, customerID).
		Order("created_at desc").Limit(limit).Find(&items).Error
	return items, err
}

func (r *Repo) CreateNote(n *CreditNote) error { return r.db.Create(n).Error }

func (r *Repo) ListNotes(storeID, customerID uuid.UUID, limit int) ([]CreditNote, error) {
	var items []CreditNote
	err := r.db.Where("store_id = ? AND customer_id = ?", storeID, customerID).
		Order("created_at desc").Limit(limit).Find(&items).Error
	return items, err
}

// TabBalanceByCustomer sums outstanding balances across open tabs.
func (r *Repo) TabBalanceByCustomer(tx *gorm.DB, customerID uuid.UUID) (int64, error) {
	var balance int64
	err := tx.Model(&CreditTab{}).
		Where("customer_id = ? AND status IN ?", customerID, []string{StatusOpen, StatusPartial}).
		Select("COALESCE(SUM(amount - amount_paid), 0)").Scan(&balance).Error
	return balance, err
}

// TabsForCustomer returns all tabs for a customer's credit profile.
func (r *Repo) TabsForCustomer(storeID, customerID uuid.UUID) ([]CreditTab, error) {
	var items []CreditTab
	err := r.db.Where("store_id = ? AND customer_id = ?", storeID, customerID).
		Order("created_at desc").Find(&items).Error
	return items, err
}

// WithTx runs a closure in a transaction.
func (r *Repo) WithTx(fn func(tx *gorm.DB) error) error { return r.db.Transaction(fn) }
