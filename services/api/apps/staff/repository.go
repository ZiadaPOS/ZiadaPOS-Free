package staff

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/credits"
	"github.com/troubleman96/ziadapos/apps/transactions"
	"gorm.io/gorm"
)

// Repo provides staff and sales-stat data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo creates a staff repository.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// ListOptions filters a staff listing.
type ListOptions struct {
	Search   string
	Role     string
	Status   string
	Shift    string
	Store    *uuid.UUID
	Ordering string
	Page     int
	Size     int
}

// List returns staff users matching the filters.
func (r *Repo) List(storeID uuid.UUID, opts ListOptions) ([]accounts.User, int64, error) {
	q := r.db.Model(&accounts.User{}).Where("store_id = ?", storeID)
	if opts.Role != "" {
		q = q.Where("role = ?", opts.Role)
	}
	if opts.Status != "" {
		q = q.Where("employment_status = ?", opts.Status)
	}
	if opts.Shift != "" {
		q = q.Where("shift = ?", opts.Shift)
	}
	if s := strings.TrimSpace(opts.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("first_name ILIKE ? OR last_name ILIKE ? OR phone ILIKE ? OR email ILIKE ?",
			like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Size < 1 || opts.Size > 100 {
		opts.Size = 20
	}
	if order := orderClause(opts.Ordering); order != "" {
		q = q.Order(order)
	} else {
		q = q.Order("created_at DESC")
	}
	var users []accounts.User
	if err := q.Offset((opts.Page - 1) * opts.Size).Limit(opts.Size).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// orderClause maps a whitelisted ordering token to a SQL order expression.
func orderClause(ordering string) string {
	switch strings.TrimSpace(ordering) {
	case "name":
		return "first_name ASC"
	case "-name":
		return "first_name DESC"
	case "joined":
		return "created_at ASC"
	case "-joined":
		return "created_at DESC"
	case "role":
		return "role ASC"
	case "-role":
		return "role DESC"
	case "status":
		return "employment_status ASC"
	case "-status":
		return "employment_status DESC"
	}
	return ""
}

// Get returns one staff member in a store.
func (r *Repo) Get(storeID, userID uuid.UUID) (*accounts.User, error) {
	var u accounts.User
	if err := r.db.Where("store_id = ? AND id = ?", storeID, userID).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// Update persists changes to a staff member.
func (r *Repo) Update(u *accounts.User) error { return r.db.Save(u).Error }

// Stats computes a cashier's performance aggregates.
func (r *Repo) Stats(storeID, userID uuid.UUID, todayStart, monthStart time.Time) (*StaffStats, error) {
	out := &StaffStats{}
	todayEnd := todayStart.Add(24 * time.Hour)
	monthEnd := monthStart.AddDate(0, 1, 0)

	type agg struct {
		Sum   int64
		Count int64
	}

	var t agg
	err := r.db.Model(&transactions.Transaction{}).
		Where("cashier_id = ? AND store_id = ? AND status = ?", userID, storeID, transactions.StatusPaid).
		Select("COALESCE(SUM(total),0) AS sum, COUNT(*) AS count").Scan(&t).Error
	if err != nil {
		return nil, err
	}
	out.TotalSales = t.Sum
	out.TxnsTotal = t.Count
	if t.Count > 0 {
		out.AvgTicket = t.Sum / t.Count
	}

	var today agg
	err = r.db.Model(&transactions.Transaction{}).
		Where("cashier_id = ? AND store_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
			userID, storeID, transactions.StatusPaid, todayStart, todayEnd).
		Select("COALESCE(SUM(total),0) AS sum, COUNT(*) AS count").Scan(&today).Error
	if err != nil {
		return nil, err
	}
	out.SalesToday = today.Sum
	out.TxnsToday = today.Count

	var month agg
	err = r.db.Model(&transactions.Transaction{}).
		Where("cashier_id = ? AND store_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
			userID, storeID, transactions.StatusPaid, monthStart, monthEnd).
		Select("COALESCE(SUM(total),0) AS sum, COUNT(*) AS count").Scan(&month).Error
	if err != nil {
		return nil, err
	}
	out.SalesThisMonth = month.Sum
	out.TxnsThisMonth = month.Count

	var credit agg
	err = r.db.Model(&credits.CreditPayment{}).
		Where("cashier_id = ? AND store_id = ?", userID, storeID).
		Select("COALESCE(SUM(amount),0) AS sum, COUNT(*) AS count").Scan(&credit).Error
	if err != nil {
		return nil, err
	}
	out.CreditCollected = credit.Sum

	var creditToday agg
	err = r.db.Model(&credits.CreditPayment{}).
		Where("cashier_id = ? AND store_id = ? AND created_at >= ? AND created_at < ?",
			userID, storeID, todayStart, todayEnd).
		Select("COALESCE(SUM(amount),0) AS sum, COUNT(*) AS count").Scan(&creditToday).Error
	if err != nil {
		return nil, err
	}
	out.CreditCollectedToday = creditToday.Sum

	return out, nil
}

// KPIResult is a store-level aggregate row.
type KPIResult struct {
	TotalStaff int64
	Active     int64
	OnLeave    int64
	Inactive   int64
	OnShift    int64
}

// StoreKPIs computes the store-level staff summary.
func (r *Repo) StoreKPIs(storeID uuid.UUID, todayStart, monthStart time.Time) (*StoreKPIs, error) {
	k := &StoreKPIs{}
	todayEnd := todayStart.Add(24 * time.Hour)
	monthEnd := monthStart.AddDate(0, 1, 0)

	var res KPIResult
	err := r.db.Model(&accounts.User{}).
		Where("store_id = ?", storeID).
		Select(`COUNT(*) AS total_staff,
			COUNT(*) FILTER (WHERE employment_status = ?) AS active,
			COUNT(*) FILTER (WHERE employment_status = ?) AS on_leave,
			COUNT(*) FILTER (WHERE employment_status = ?) AS inactive,
			COUNT(*) FILTER (WHERE employment_status = ? AND is_active = true) AS on_shift`,
			EmpActive, EmpOnLeave, EmpInactive, EmpActive).
		Scan(&res).Error
	if err != nil {
		return nil, err
	}
	k.TotalStaff = res.TotalStaff
	k.Active = res.Active
	k.OnLeave = res.OnLeave
	k.Inactive = res.Inactive
	k.OnShiftToday = res.OnShift

	type agg struct {
		Sum   int64
		Count int64
	}
	var t, m agg
	if err := r.db.Model(&transactions.Transaction{}).
		Where("store_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
			storeID, transactions.StatusPaid, todayStart, todayEnd).
		Select("COALESCE(SUM(total),0) AS sum, COUNT(*) AS count").Scan(&t).Error; err != nil {
		return nil, err
	}
	k.SalesToday = t.Sum
	k.TotalSalesToday = t.Sum
	k.TxnsToday = t.Count
	if t.Count > 0 {
		k.AvgTicketToday = t.Sum / t.Count
	}
	if err := r.db.Model(&transactions.Transaction{}).
		Where("store_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
			storeID, transactions.StatusPaid, monthStart, monthEnd).
		Select("COALESCE(SUM(total),0) AS sum, COUNT(*) AS count").Scan(&m).Error; err != nil {
		return nil, err
	}
	k.SalesThisMonth = m.Sum

	type top struct {
		Name  string
		Sales int64
	}
	var row top
	err = r.db.Table("transactions AS t").
		Select("u.first_name || ' ' || COALESCE(u.last_name, '') AS name, COALESCE(SUM(t.total),0) AS sales").
		Joins("JOIN users u ON u.id = t.cashier_id").
		Where("t.store_id = ? AND t.status = ? AND t.created_at >= ? AND t.created_at < ?",
			storeID, transactions.StatusPaid, monthStart, monthEnd).
		Group("u.id").Order("sales DESC").Limit(1).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	k.TopCashierName = row.Name
	k.TopCashier = row.Name
	k.TopCashierSales = row.Sales

	return k, nil
}

// StoreDetail is a lightweight store embed used in staff responses.
type StoreDetail struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Area  string    `json:"area"`
	Phone string    `json:"phone,omitempty"`
}

// Store returns the store detail for a staff member's store.
func (r *Repo) Store(storeID uuid.UUID) (*StoreDetail, error) {
	var s accounts.Store
	if err := r.db.Select("id, name, area, phone").Where("id = ?", storeID).First(&s).Error; err != nil {
		return nil, err
	}
	return &StoreDetail{ID: s.ID, Name: s.Name, Area: s.Area, Phone: s.Phone}, nil
}

// ActivityItem is one transaction in a cashier's activity feed.
type ActivityItem struct {
	ID             uuid.UUID `json:"id"`
	TxnNumber      string    `json:"txn_number"`
	PaymentMethod  string    `json:"payment_method"`
	Status         string    `json:"status"`
	Subtotal       int64     `json:"subtotal"`
	DiscountAmount int64     `json:"discount_amount"`
	TaxAmount      int64     `json:"tax_amount"`
	Total          int64     `json:"total"`
	Profit         int64     `json:"profit"`
	CustomerName   string    `json:"customer_name"`
	CustomerPhone  string    `json:"customer_phone"`
	TillNumber     string    `json:"till_number"`
	Channel        string    `json:"channel"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
}

// Activity returns a cashier's transactions within a time range.
func (r *Repo) Activity(storeID, userID uuid.UUID, from, to time.Time) ([]ActivityItem, error) {
	var rows []ActivityItem
	err := r.db.Model(&transactions.Transaction{}).
		Select("id, txn_number, payment_method, status, subtotal, discount_amount, tax_amount, total, profit, customer_name, customer_phone, till_number, channel, notes, created_at").
		Where("store_id = ? AND cashier_id = ? AND created_at >= ? AND created_at < ?", storeID, userID, from, to).
		Order("created_at DESC").Limit(100).Scan(&rows).Error
	return rows, err
}
