// Package analytics aggregates POS KPIs, trends and performance breakdowns.
package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	statusPaid    = "paid"
	statusCredit  = "credit"
	statusRefund  = "refunded"
	creditPartial = "partially_paid"
)

// Repo runs read-only aggregation queries over the shared schema.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

type kpiAgg struct {
	Revenue      int64 `gorm:"column:revenue"`
	CreditRevenue int64 `gorm:"column:credit_revenue"`
	Profit       int64 `gorm:"column:profit"`
	Cost         int64 `gorm:"column:cost"`
	Tax          int64 `gorm:"column:tax"`
	Transactions int64 `gorm:"column:transactions"`
	Customers    int64 `gorm:"column:customers"`
}

// KPIAgg returns headline aggregates for paid/credit transactions.
func (r *Repo) KPIAgg(ctx context.Context, storeID uuid.UUID, from, to time.Time) (kpiAgg, error) {
	var a kpiAgg
	err := r.db.Table("transactions").
		Select(`
			COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS revenue,
			COALESCE(SUM(total) FILTER (WHERE status = 'credit'), 0) AS credit_revenue,
			COALESCE(SUM(profit) FILTER (WHERE status = 'paid'), 0) AS profit,
			COALESCE(SUM(cost_total) FILTER (WHERE status = 'paid'), 0) AS cost,
			COALESCE(SUM(tax_amount) FILTER (WHERE status = 'paid'), 0) AS tax,
			COUNT(*) FILTER (WHERE status IN ('paid','credit')) AS transactions,
			COUNT(DISTINCT customer_id) FILTER (WHERE customer_id IS NOT NULL) AS customers`).
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Scan(&a).Error
	return a, err
}

// PriorRevenue returns revenue + transaction count for a prior period.
func (r *Repo) PriorRevenue(ctx context.Context, storeID uuid.UUID, from, to time.Time) (int64, int64, error) {
	var a struct {
		Revenue int64 `gorm:"column:revenue"`
		Count   int64 `gorm:"column:count"`
	}
	err := r.db.Table("transactions").
		Select("COALESCE(SUM(total),0) AS revenue, COUNT(*) AS count").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Scan(&a).Error
	return a.Revenue, a.Count, err
}

// RefundTotal returns the value of refunded transactions in range.
func (r *Repo) RefundTotal(ctx context.Context, storeID uuid.UUID, from, to time.Time) (int64, error) {
	var v int64
	err := r.db.Table("transactions").
		Select("COALESCE(SUM(total),0)").
		Where("store_id = ? AND status = ? AND created_at >= ? AND created_at < ?", storeID, statusRefund, from, to).
		Scan(&v).Error
	return v, err
}

// DiscountAgg returns total discount value and count of discounted tickets.
func (r *Repo) DiscountAgg(ctx context.Context, storeID uuid.UUID, from, to time.Time) (int64, int64, error) {
	var a struct {
		Total int64 `gorm:"column:total"`
		Count int64 `gorm:"column:count"`
	}
	err := r.db.Table("transactions").
		Select("COALESCE(SUM(discount_amount),0) AS total, COUNT(*) AS count").
		Where("store_id = ? AND status IN ? AND discount_amount > 0 AND created_at >= ? AND created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Scan(&a).Error
	return a.Total, a.Count, err
}

// ExpenseAgg returns total expense amount and count in range.
func (r *Repo) ExpenseAgg(ctx context.Context, storeID uuid.UUID, from, to time.Time) (int64, int64, error) {
	var a struct {
		Total int64 `gorm:"column:total"`
		Count int64 `gorm:"column:count"`
	}
	err := r.db.Table("expenses").
		Select("COALESCE(SUM(amount),0) AS total, COUNT(*) AS count").
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Scan(&a).Error
	return a.Total, a.Count, err
}

// trendRow is one day of aggregated revenue.
type trendRow struct {
	Day           time.Time `gorm:"column:day"`
	Revenue       int64     `gorm:"column:revenue"`
	CreditRevenue int64     `gorm:"column:credit_revenue"`
	Profit        int64     `gorm:"column:profit"`
	Transactions  int64     `gorm:"column:transactions"`
}

// RevenueTrend returns per-day revenue/profit/transaction counts (EAT dates).
func (r *Repo) RevenueTrend(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]trendRow, error) {
	var rows []trendRow
	err := r.db.Table("transactions").
		Select(`
			(created_at + interval '3 hours')::date AS day,
			COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS revenue,
			COALESCE(SUM(total) FILTER (WHERE status = 'credit'), 0) AS credit_revenue,
			COALESCE(SUM(profit) FILTER (WHERE status = 'paid'), 0) AS profit,
			COUNT(*) FILTER (WHERE status IN ('paid','credit')) AS transactions`).
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("day").Order("day asc").Scan(&rows).Error
	return rows, err
}

type methodRow struct {
	Method string `gorm:"column:payment_method"`
	Amount int64  `gorm:"column:amount"`
}

// PaymentMethodTotals returns revenue grouped by payment method (paid only).
func (r *Repo) PaymentMethodTotals(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]methodRow, error) {
	var rows []methodRow
	err := r.db.Table("transactions").
		Select("payment_method, COALESCE(SUM(total),0) AS amount").
		Where("store_id = ? AND status = ? AND created_at >= ? AND created_at < ?", storeID, statusPaid, from, to).
		Group("payment_method").Scan(&rows).Error
	return rows, err
}

// CreditIssued returns the value of credit-status tickets in range.
func (r *Repo) CreditIssued(ctx context.Context, storeID uuid.UUID, from, to time.Time) (int64, error) {
	var v int64
	err := r.db.Table("transactions").
		Select("COALESCE(SUM(total),0)").
		Where("store_id = ? AND status = ? AND created_at >= ? AND created_at < ?", storeID, statusCredit, from, to).
		Scan(&v).Error
	return v, err
}

type productRow struct {
	ProductID   uuid.UUID `gorm:"column:product_id"`
	ProductName string    `gorm:"column:product_name"`
	ProductSKU  string    `gorm:"column:product_sku"`
	Units       int       `gorm:"column:units"`
	Revenue     int64     `gorm:"column:revenue"`
	Cost        int64     `gorm:"column:cost"`
	Category    string    `gorm:"column:category"`
}

// TopProducts returns the top products by revenue in range.
func (r *Repo) TopProducts(ctx context.Context, storeID uuid.UUID, from, to time.Time, limit int) ([]productRow, error) {
	return r.products(ctx, storeID, from, to, "", limit)
}

// Products returns per-product performance, optionally filtered by category.
func (r *Repo) Products(ctx context.Context, storeID uuid.UUID, from, to time.Time, category string) ([]productRow, error) {
	return r.products(ctx, storeID, from, to, category, 0)
}

func (r *Repo) products(ctx context.Context, storeID uuid.UUID, from, to time.Time, category string, limit int) ([]productRow, error) {
	q := r.db.Table("transaction_lines AS l").
		Select(`
			l.product_id,
			l.product_name,
			COALESCE(l.product_sku,'') AS product_sku,
			COALESCE(c.name, 'Uncategorised') AS category,
			SUM(l.qty) AS units,
			SUM(l.unit_price*l.qty) AS revenue,
			SUM(l.unit_cost*l.qty) AS cost`).
		Joins("JOIN transactions AS t ON t.id = l.transaction_id").
		Joins("LEFT JOIN products AS p ON p.id = l.product_id").
		Joins("LEFT JOIN categories AS c ON c.id = p.category_id").
		Where("t.store_id = ? AND t.status IN ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to)
	if category != "" {
		q = q.Where("c.name = ?", category)
	}
	q = q.Group("l.product_id, l.product_name, l.product_sku, category").
		Order("revenue desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var rows []productRow
	err := q.Scan(&rows).Error
	return rows, err
}

// PriorProductRevenue returns prior-period revenue keyed by product id/name.
func (r *Repo) PriorProductRevenue(ctx context.Context, storeID uuid.UUID, from, to time.Time) (map[string]int64, error) {
	var rows []struct {
		ProductID uuid.UUID `gorm:"column:product_id"`
		Name      string    `gorm:"column:product_name"`
		Revenue   int64     `gorm:"column:revenue"`
	}
	err := r.db.Table("transaction_lines AS l").
		Select("l.product_id, l.product_name, SUM(l.unit_price*l.qty) AS revenue").
		Joins("JOIN transactions AS t ON t.id = l.transaction_id").
		Where("t.store_id = ? AND t.status IN ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("l.product_id, l.product_name").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		key := row.ProductID.String()
		if row.ProductID == uuid.Nil {
			key = row.Name
		}
		out[key] = row.Revenue
	}
	return out, nil
}

type catRow struct {
	Name    string `gorm:"column:name"`
	Revenue int64  `gorm:"column:revenue"`
}

// CategoryRevenue returns revenue per product category in range.
func (r *Repo) CategoryRevenue(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]catRow, error) {
	var rows []catRow
	err := r.db.Table("transaction_lines AS l").
		Select("COALESCE(c.name, 'Uncategorised') AS name, SUM(l.unit_price*l.qty) AS revenue").
		Joins("JOIN transactions AS t ON t.id = l.transaction_id").
		Joins("LEFT JOIN products AS p ON p.id = l.product_id").
		Joins("LEFT JOIN categories AS c ON c.id = p.category_id").
		Where("t.store_id = ? AND t.status IN ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("c.name").Scan(&rows).Error
	return rows, err
}

type dowRow struct {
	Dow   int   `gorm:"column:dow"`
	Total int64 `gorm:"column:total"`
}

// DayOfWeekTotals returns revenue grouped by day of week (Mon=0..Sun=6).
func (r *Repo) DayOfWeekTotals(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]dowRow, error) {
	var rows []dowRow
	err := r.db.Table("transactions").
		Select("(EXTRACT(DOW FROM created_at + interval '3 hours')::int + 6) % 7 AS dow, SUM(total) AS total").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("dow").Scan(&rows).Error
	return rows, err
}

type hourRow struct {
	Hour  int   `gorm:"column:hour"`
	Total int64 `gorm:"column:total"`
}

// HourlyTotals returns revenue grouped by local hour (EAT).
func (r *Repo) HourlyTotals(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]hourRow, error) {
	var rows []hourRow
	err := r.db.Table("transactions").
		Select("((EXTRACT(HOUR FROM created_at)::int + 3) % 24) AS hour, SUM(total) AS total").
		Where("store_id = ? AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("hour").Scan(&rows).Error
	return rows, err
}

type firstDateRow struct {
	CustomerID uuid.UUID `gorm:"column:customer_id"`
	FirstDate  time.Time `gorm:"column:first_date"`
}

// CustomerFirstDates returns each registered customer's first transaction.
func (r *Repo) CustomerFirstDates(ctx context.Context, storeID uuid.UUID) (map[uuid.UUID]time.Time, error) {
	var rows []firstDateRow
	err := r.db.Table("transactions").
		Select("customer_id, MIN(created_at) AS first_date").
		Where("store_id = ? AND customer_id IS NOT NULL AND status IN ?", storeID, []string{statusPaid, statusCredit}).
		Group("customer_id").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]time.Time, len(rows))
	for _, row := range rows {
		out[row.CustomerID] = row.FirstDate
	}
	return out, nil
}

type visitDayRow struct {
	CustomerID uuid.UUID `gorm:"column:customer_id"`
	Day        time.Time `gorm:"column:day"`
}

// CustomerVisitDays returns distinct (customer, EAT date) pairs in range.
func (r *Repo) CustomerVisitDays(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]visitDayRow, error) {
	var rows []visitDayRow
	err := r.db.Table("transactions").
		Select("customer_id, (created_at + interval '3 hours')::date AS day").
		Where("store_id = ? AND customer_id IS NOT NULL AND status IN ? AND created_at >= ? AND created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("customer_id, day").Scan(&rows).Error
	return rows, err
}

type segmentRow struct {
	Segment string `gorm:"column:segment"`
	Count   int64  `gorm:"column:count"`
	Spend   int64  `gorm:"column:spend"`
}

// CustomerSegments groups spend by customer segment.
func (r *Repo) CustomerSegments(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]segmentRow, error) {
	var rows []segmentRow
	err := r.db.Table("transactions AS t").
		Select("COALESCE(c.segment, 'Unknown') AS segment, COUNT(DISTINCT t.customer_id) AS count, SUM(t.total) AS spend").
		Joins("JOIN customers AS c ON c.id = t.customer_id").
		Where("t.store_id = ? AND t.status IN ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("segment").Scan(&rows).Error
	return rows, err
}

type monthRow struct {
	CustomerID uuid.UUID `gorm:"column:customer_id"`
	Month      time.Time `gorm:"column:month"`
}

// CustomerMonthlyActivity returns distinct (customer, month) activity.
func (r *Repo) CustomerMonthlyActivity(ctx context.Context, storeID uuid.UUID) ([]monthRow, error) {
	var rows []monthRow
	err := r.db.Table("transactions").
		Select("customer_id, date_trunc('month', created_at + interval '3 hours') AS month").
		Where("store_id = ? AND customer_id IS NOT NULL AND status IN ?", storeID, []string{statusPaid, statusCredit}).
		Group("customer_id, month").Scan(&rows).Error
	return rows, err
}

type topCustomerRow struct {
	CustomerID uuid.UUID `gorm:"column:customer_id"`
	Name       string    `gorm:"column:name"`
	AvatarHue  int       `gorm:"column:avatar_hue"`
	Visits     int64     `gorm:"column:visits"`
	Spent      int64     `gorm:"column:spent"`
	LastTxn    time.Time `gorm:"column:last_txn"`
}

// TopCustomers returns the highest-spending customers in range.
func (r *Repo) TopCustomers(ctx context.Context, storeID uuid.UUID, from, to time.Time, limit int) ([]topCustomerRow, error) {
	var rows []topCustomerRow
	err := r.db.Table("transactions AS t").
		Select("c.id AS customer_id, c.name, c.avatar_hue, COUNT(t.id) AS visits, SUM(t.total) AS spent, MAX(t.created_at) AS last_txn").
		Joins("JOIN customers AS c ON c.id = t.customer_id").
		Where("t.store_id = ? AND t.status IN ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, []string{statusPaid, statusCredit}, from, to).
		Group("c.id, c.name, c.avatar_hue").Order("spent desc").Limit(limit).Scan(&rows).Error
	return rows, err
}

type cashflowDayRow struct {
	Day           time.Time `gorm:"column:day"`
	Revenue       int64     `gorm:"column:revenue"`
	CreditRevenue int64     `gorm:"column:credit_revenue"`
	Cogs          int64     `gorm:"column:cogs"`
}

// CashflowDays returns per-day inflow/cogs for paid+credit transactions.
func (r *Repo) CashflowDays(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]cashflowDayRow, error) {
	var rows []cashflowDayRow
	err := r.db.Table("transactions").
		Select(`
			(created_at + interval '3 hours')::date AS day,
			COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS revenue,
			COALESCE(SUM(total) FILTER (WHERE status = 'credit'), 0) AS credit_revenue,
			COALESCE(SUM(cost_total) FILTER (WHERE status = 'paid'), 0) AS cogs`).
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("day").Order("day asc").Scan(&rows).Error
	return rows, err
}

type expenseDayRow struct {
	Day    time.Time `gorm:"column:day"`
	Amount int64     `gorm:"column:amount"`
}

// ExpenseDays returns per-day expense totals in range.
func (r *Repo) ExpenseDays(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]expenseDayRow, error) {
	var rows []expenseDayRow
	err := r.db.Table("expenses").
		Select("(created_at + interval '3 hours')::date AS day, SUM(amount) AS amount").
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("day").Scan(&rows).Error
	return rows, err
}

// ExpenseByCategory returns expense totals grouped by category.
func (r *Repo) ExpenseByCategory(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]catRow, error) {
	var rows []catRow
	err := r.db.Table("expenses").
		Select("category AS name, SUM(amount) AS amount").
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("category").Order("amount desc").Scan(&rows).Error
	return rows, err
}

type creditOutstanding struct {
	Total         int64 `gorm:"column:total"`
	CustomerCount int64 `gorm:"column:customer_count"`
	OverdueCount  int64 `gorm:"column:overdue_count"`
}

// CreditOutstanding returns open-credit totals for a store.
func (r *Repo) CreditOutstanding(ctx context.Context, storeID uuid.UUID) (creditOutstanding, error) {
	var c creditOutstanding
	err := r.db.Table("credit_tabs").
		Select(`
			COALESCE(SUM(amount - amount_paid), 0) AS total,
			COUNT(DISTINCT customer_id) AS customer_count,
			COUNT(*) FILTER (WHERE due_date IS NOT NULL AND due_date < now()) AS overdue_count`).
		Where("store_id = ? AND status IN ?", storeID, []string{"open", creditPartial}).
		Scan(&c).Error
	return c, err
}

// lowStockRow is a product at or below its reorder point.
type lowStockRow struct {
	ID         uuid.UUID `gorm:"column:id"`
	Name       string    `gorm:"column:name"`
	SKU        *string   `gorm:"column:sku"`
	Stock      int       `gorm:"column:stock"`
	Reorder    int       `gorm:"column:reorder_point"`
}

// LowStock returns active products at or below reorder point.
func (r *Repo) LowStock(ctx context.Context, storeID uuid.UUID, limit int) ([]lowStockRow, error) {
	var rows []lowStockRow
	err := r.db.Table("products").
		Select("id, name, sku, stock, reorder_point").
		Where("store_id = ? AND is_active = true AND is_service = false AND stock <= reorder_point", storeID).
		Order("stock asc").Limit(limit).Scan(&rows).Error
	return rows, err
}

// HourlyBreakdown returns today's transaction revenue by local hour.
func (r *Repo) HourlyBreakdown(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]hourBreakdown, error) {
	var rows []hourBreakdown
	err := r.db.Table("transactions").
		Select(`
			((EXTRACT(HOUR FROM created_at)::int + 3) % 24) AS hour,
			COALESCE(SUM(total) FILTER (WHERE status = 'paid'), 0) AS revenue,
			COALESCE(SUM(profit) FILTER (WHERE status = 'paid'), 0) AS profit,
			COALESCE(SUM(discount_amount), 0) AS discount_amount,
			COALESCE(SUM(total) FILTER (WHERE status = 'credit'), 0) AS credit_amount,
			COUNT(*) FILTER (WHERE status IN ('paid','credit')) AS txn_count`).
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("hour").Scan(&rows).Error
	return rows, err
}

// ExpenseHourly returns today's expense totals by local hour.
func (r *Repo) ExpenseHourly(ctx context.Context, storeID uuid.UUID, from, to time.Time) ([]hourRow, error) {
	var rows []hourRow
	err := r.db.Table("expenses").
		Select("((EXTRACT(HOUR FROM created_at)::int + 3) % 24) AS hour, SUM(amount) AS total").
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Group("hour").Scan(&rows).Error
	return rows, err
}

// hourBreakdown aggregates one hour's transactions.
type hourBreakdown struct {
	Hour           int   `gorm:"column:hour"`
	Revenue        int64 `gorm:"column:revenue"`
	Profit         int64 `gorm:"column:profit"`
	DiscountAmount int64 `gorm:"column:discount_amount"`
	CreditAmount   int64 `gorm:"column:credit_amount"`
	TxnCount       int64 `gorm:"column:txn_count"`
}
