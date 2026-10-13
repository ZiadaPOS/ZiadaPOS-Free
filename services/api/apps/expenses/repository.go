package expenses

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides expense data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Create(e *Expense) error { return r.db.Create(e).Error }

func (r *Repo) Get(storeID, id uuid.UUID) (*Expense, error) {
	var e Expense
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&e).Error; err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repo) Update(e *Expense) error { return r.db.Save(e).Error }

func (r *Repo) Delete(storeID, id uuid.UUID) error {
	return r.db.Where("id = ? AND store_id = ?", id, storeID).Delete(&Expense{}).Error
}

// Filter constrains expense list queries.
type Filter struct {
	StoreID       uuid.UUID
	Search        string
	Category      string
	PaymentMethod string
	Ordering      string
	Page          int
	PageSize      int
}

var allowedOrderings = map[string]bool{
	"amount": true, "-amount": true,
	"created_at": true, "-created_at": true,
	"category": true, "-category": true,
}

// columnAndDirection turns "-created_at" into "created_at DESC" so GORM never
// passes a raw unary minus into the ORDER BY clause.
func columnAndDirection(order string) string {
	if len(order) > 1 && order[0] == '-' {
		return order[1:] + " DESC"
	}
	return order + " ASC"
}

func (r *Repo) List(f Filter) ([]Expense, int64, error) {
	q := r.db.Model(&Expense{}).Where("store_id = ?", f.StoreID)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(lower(title) LIKE lower(?) OR lower(notes) LIKE lower(?) OR lower(payment_reference) LIKE lower(?))", like, like, like)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.PaymentMethod != "" {
		q = q.Where("payment_method = ?", f.PaymentMethod)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := f.Ordering
	if !allowedOrderings[order] {
		order = "-created_at"
	}
	q = q.Order(columnAndDirection(order))
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var items []Expense
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Summary is the aggregated expense report.
type Summary struct {
	TotalAmount int64 `json:"total_amount"`
	TotalCount  int64 `json:"total_count"`
}

// byCategoryRow aggregates one category bucket.
type byCategoryRow struct {
	Category string
	Amount   int64
	Count    int64
}

// byMethodRow aggregates one payment method bucket.
type byMethodRow struct {
	Method string
	Amount int64
	Count  int64
}

func (r *Repo) SummaryAgg(storeID uuid.UUID) (*Summary, error) {
	var s Summary
	err := r.db.Model(&Expense{}).Where("store_id = ?", storeID).
		Select("COALESCE(SUM(amount),0) AS total_amount, COUNT(*) AS total_count").
		Scan(&s).Error
	return &s, err
}

func (r *Repo) ByCategory(storeID uuid.UUID) ([]byCategoryRow, error) {
	var rows []byCategoryRow
	err := r.db.Model(&Expense{}).Where("store_id = ?", storeID).
		Select("category, COALESCE(SUM(amount),0) AS amount, COUNT(*) AS count").
		Group("category").Order("amount desc").Scan(&rows).Error
	return rows, err
}

func (r *Repo) ByMethod(storeID uuid.UUID) ([]byMethodRow, error) {
	var rows []byMethodRow
	err := r.db.Model(&Expense{}).Where("store_id = ?", storeID).
		Select("payment_method AS method, COALESCE(SUM(amount),0) AS amount, COUNT(*) AS count").
		Group("payment_method").Order("amount desc").Scan(&rows).Error
	return rows, err
}

func (r *Repo) Recent(storeID uuid.UUID, limit int) ([]Expense, error) {
	var items []Expense
	err := r.db.Where("store_id = ?", storeID).Order("created_at desc").Limit(limit).Find(&items).Error
	return items, err
}
