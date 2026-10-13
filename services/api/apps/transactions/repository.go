package transactions

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
)

// Repo provides transaction data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *gorm.DB { return r.db }

// NextTxnNumber atomically increments the per-store ticket counter and
// returns the display number for a new receipt.
func (r *Repo) NextTxnNumber(tx *gorm.DB, storeID uuid.UUID) (int, error) {
	var n int
	err := tx.Raw(
		`INSERT INTO txn_counters (store_id, last_number) VALUES (?, 1)
		 ON CONFLICT (store_id)
		 DO UPDATE SET last_number = txn_counters.last_number + 1
		 RETURNING last_number`, storeID,
	).Scan(&n).Error
	return n, err
}

// CreateSaleTx persists a transaction with its lines atomically.
func (r *Repo) CreateSaleTx(tx *gorm.DB, t *Transaction) error {
	// Omit prevents GORM's automatic association save (which would double
	// insert the lines); we persist them explicitly below.
	if err := tx.Omit("Lines").Create(t).Error; err != nil {
		return err
	}
	for i := range t.Lines {
		t.Lines[i].TransactionID = t.ID
	}
	if len(t.Lines) > 0 {
		return tx.Create(&t.Lines).Error
	}
	return nil
}

func (r *Repo) GetTransaction(storeID, id uuid.UUID) (*Transaction, error) {
	var t Transaction
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).
		Preload("Lines", func(db *gorm.DB) *gorm.DB {
			return db.Order("transaction_lines.created_at asc")
		}).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Filter constrains transaction list queries.
type Filter struct {
	StoreID    uuid.UUID
	Status     string
	Method     string
	Search     string
	DateFrom   time.Time
	DateTo     time.Time
	CustomerID uuid.UUID
	Page       int
	PageSize   int
}

func (r *Repo) ListTransactions(f Filter) ([]Transaction, int64, error) {
	q := r.db.Model(&Transaction{}).Where("store_id = ?", f.StoreID)
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Method != "" {
		q = q.Where("payment_method = ?", f.Method)
	}
	if f.CustomerID != uuid.Nil {
		q = q.Where("customer_id = ?", f.CustomerID)
	}
	if !f.DateFrom.IsZero() {
		q = q.Where("created_at >= ?", f.DateFrom)
	}
	if !f.DateTo.IsZero() {
		q = q.Where("created_at < ?", f.DateTo)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(txn_number ILIKE ? OR lower(customer_name) ILIKE ? OR payment_reference ILIKE ?)", like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("created_at desc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}
	var txns []Transaction
	if err := q.Find(&txns).Error; err != nil {
		return nil, 0, err
	}
	return txns, total, nil
}

// LoadLines fetches lines for a set of transactions (avoid N+1).
func (r *Repo) LoadLines(txns []Transaction) error {
	if len(txns) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(txns))
	for _, t := range txns {
		ids = append(ids, t.ID)
	}
	var lines []TransactionLine
	if err := r.db.Where("transaction_id IN ?", ids).Order("created_at asc").Find(&lines).Error; err != nil {
		return err
	}
	byTxn := map[uuid.UUID][]TransactionLine{}
	for _, l := range lines {
		byTxn[l.TransactionID] = append(byTxn[l.TransactionID], l)
	}
	for i := range txns {
		txns[i].Lines = byTxn[txns[i].ID]
	}
	return nil
}

// ── Summary queries ──────────────────────────────────────────────────────────

// Row aggregates one transaction for summary computation.
type Row struct {
	Status string
	Total  int64
	Profit int64
	Tax    int64
}

func (r *Repo) RangeRows(storeID uuid.UUID, from, to time.Time) ([]Row, error) {
	var rows []Row
	err := r.db.Model(&Transaction{}).
		Select("status, total, profit, tax_amount AS tax").
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Find(&rows).Error
	return rows, err
}

// MethodTotals returns total sales per payment method in a range.
func (r *Repo) MethodTotals(storeID uuid.UUID, from, to time.Time) (map[string]int64, error) {
	type mr struct {
		PaymentMethod string
		Total         int64
	}
	var rows []mr
	err := r.db.Model(&Transaction{}).
		Select("payment_method, sum(total) AS total").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{StatusPaid, StatusCredit}, from, to).
		Group("payment_method").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r := range rows {
		out[r.PaymentMethod] = r.Total
	}
	return out, nil
}

// TopProducts returns the best sellers by quantity in a range.
func (r *Repo) TopProducts(storeID uuid.UUID, from, to time.Time, limit int) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := r.db.Table("transaction_lines AS l").
		Select("l.product_id, l.product_name, SUM(l.qty) AS qty_sold, SUM(l.unit_price*l.qty) AS sales").
		Joins("JOIN transactions AS t ON t.id = l.transaction_id").
		Where("t.store_id = ? AND t.status IN ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, []string{StatusPaid, StatusCredit}, from, to).
		Group("l.product_id, l.product_name").
		Order("qty_sold desc").
		Limit(limit).Scan(&rows).Error
	return rows, err
}

// HourlySales returns total sales grouped by hour for a day.
func (r *Repo) HourlySales(storeID uuid.UUID, day time.Time) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := r.db.Model(&Transaction{}).
		Select("EXTRACT(HOUR FROM created_at)::int AS hour, sum(total) AS total, count(*) AS count").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{StatusPaid, StatusCredit}, day, day.Add(24*time.Hour)).
		Group("hour").Order("hour asc").Scan(&rows).Error
	return rows, err
}

// UpdateStatus changes a transaction status (for refund/void).
func (r *Repo) UpdateStatus(tx *gorm.DB, id uuid.UUID, status, notes string) error {
	result := tx.Model(&Transaction{}).Where("id = ? AND status IN ?", id, []string{StatusPaid, StatusCredit}).
		Updates(map[string]interface{}{"status": status, "notes": notes})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return apierr.Conflict("Transaction already closed")
	}
	return nil
}

// GetStatus returns the current status of a transaction.
func (r *Repo) GetStatus(storeID, id uuid.UUID) (string, error) {
	var status string
	if err := r.db.Model(&Transaction{}).
		Where("id = ? AND store_id = ?", id, storeID).
		Pluck("status", &status).Error; err != nil {
		return "", err
	}
	return status, nil
}

// ErrNotFound wraps gorm not-found for callers.
var ErrNotFound = gorm.ErrRecordNotFound

// IsNotFound reports a gorm not-found error.
func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

// WithTx runs a closure in a transaction.
func (r *Repo) WithTx(fn func(tx *gorm.DB) error) error { return r.db.Transaction(fn) }

// DecrementStockAtomically locks a product row and reduces stock by qty.
func (r *Repo) DecrementStockAtomically(tx *gorm.DB, storeID, productID uuid.UUID, qty int) error {
	var p inventory.Product
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", productID, storeID).First(&p).Error; err != nil {
		return err
	}
	if !p.IsService {
		if p.Stock < qty {
			return apierr.Conflict("Insufficient stock").WithField("product", fmt.Sprintf("%s only has %d units", p.Name, p.Stock))
		}
	}
	return tx.Model(&inventory.Product{}).Where("id = ?", productID).
		Update("stock", gorm.Expr("stock - ?", qty)).Error
}

// RestoreStockAtomically returns stock after a refund/void.
func (r *Repo) RestoreStockAtomically(tx *gorm.DB, productID uuid.UUID, qty int) error {
	return tx.Model(&inventory.Product{}).Where("id = ?", productID).
		Update("stock", gorm.Expr("stock + ?", qty)).Error
}
