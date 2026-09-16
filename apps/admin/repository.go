package admin

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides cross-tenant data access for the platform admin surface.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds an admin Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// Options are shared list filters.
type Options struct {
	Search   string
	OrgID    string
	StoreID  string
	Status   string
	Role     string
	Page     int
	PageSize int
}

func (o *Options) clamp() {
	if o.Page <= 0 {
		o.Page = 1
	}
	if o.PageSize <= 0 {
		o.PageSize = 20
	}
	if o.PageSize > 200 {
		o.PageSize = 200
	}
}

func orgUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

// ── Organisations ──────────────────────────────────────────────────────────

func (r *Repo) Organisations(o Options) ([]orgRow, int64, error) {
	o.clamp()
	var total int64
	cq := r.db.Model(&orgInv{})
	if o.Search != "" {
		like := "%" + o.Search + "%"
		cq = cq.Where("(lower(name) LIKE lower(?) OR lower(slug) LIKE lower(?) OR lower(phone) LIKE lower(?) OR lower(email) LIKE lower(?))",
			like, like, like, like)
	}
	if o.OrgID != "" {
		cq = cq.Where("id = ?", orgUUID(o.OrgID))
	}
	if o.Status == "active" {
		cq = cq.Where("is_active = true")
	} else if o.Status == "inactive" {
		cq = cq.Where("is_active = false")
	} else if o.Status == "trial" {
		cq = cq.Where("plan = 'free' AND trial_ends_at > now()")
	} else if o.Status != "" {
		cq = cq.Where("plan = ?", o.Status)
	}
	if err := cq.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	selectSQL := `
		SELECT org.id, org.name, org.slug, org.legal_name, org.tin,
		       org.business_type, org.region, org.district, org.phone, org.email,
		       org.plan, org.max_stores, org.trial_ends_at, org.is_active, org.created_at,
		       (SELECT COUNT(*) FROM stores s WHERE s.organisation_id = org.id) AS store_count,
		       (SELECT COUNT(*) FROM users u WHERE u.organisation_id = org.id) AS user_count,
		       (SELECT COUNT(*) FROM users u WHERE u.organisation_id = org.id AND u.is_active = true) AS active_user_count,
		       (SELECT COALESCE(SUM(t.total), 0) FROM transactions t
		         WHERE t.organisation_id = org.id AND t.status IN ('paid','credit')
		           AND t.created_at >= now() - interval '30 days') AS revenue_30d,
		       (SELECT COUNT(*) FROM transactions t
		         WHERE t.organisation_id = org.id AND t.status IN ('paid','credit')
		           AND t.created_at >= now() - interval '30 days') AS txn_count_30d,
		       (SELECT sub.status FROM subscriptions sub
		         WHERE sub.organisation_id = org.id
		         ORDER BY sub.created_at DESC LIMIT 1) AS subscription_status
		FROM organisations org`
	q := r.db.Table("organisations as org").Select(selectSQL)
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(org.name) LIKE lower(?) OR lower(org.slug) LIKE lower(?) OR lower(org.phone) LIKE lower(?) OR lower(org.email) LIKE lower(?))",
			like, like, like, like)
	}
	if o.OrgID != "" {
		q = q.Where("org.id = ?", orgUUID(o.OrgID))
	}
	if o.Status == "active" {
		q = q.Where("org.is_active = true")
	} else if o.Status == "inactive" {
		q = q.Where("org.is_active = false")
	} else if o.Status == "trial" {
		q = q.Where("org.plan = 'free' AND org.trial_ends_at > now()")
	} else if o.Status != "" {
		q = q.Where("org.plan = ?", o.Status)
	}
	var rows []orgRow
	err := q.Order("org.created_at desc").
		Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).
		Scan(&rows).Error
	return rows, total, err
}

// orgInv is a stand-in model for counting organisations without importing the
// accounts model (keeps the admin package dependency-free for counts).
type orgInv struct{}

func (orgInv) TableName() string { return "organisations" }

func (r *Repo) OrganisationByID(id uuid.UUID) (*orgRow, error) {
	var row orgRow
	selectSQL := `
		SELECT org.id, org.name, org.slug, org.legal_name, org.tin,
		       org.business_type, org.occupation, org.region, org.district, org.ward,
		       org.street, org.address, org.latitude, org.longitude,
		       org.country, org.currency, org.phone, org.email, org.logo,
		       org.plan, org.max_stores, org.ai_credits_monthly, org.trial_ends_at,
		       org.is_active, org.created_at, org.settings,
		       (SELECT COUNT(*) FROM stores s WHERE s.organisation_id = org.id) AS store_count,
		       (SELECT COUNT(*) FROM users u WHERE u.organisation_id = org.id) AS user_count,
		       (SELECT COUNT(*) FROM users u WHERE u.organisation_id = org.id AND u.is_active = true) AS active_user_count,
		       (SELECT COALESCE(SUM(t.total), 0) FROM transactions t
		         WHERE t.organisation_id = org.id AND t.status IN ('paid','credit')) AS revenue_30d,
		       (SELECT sub.status FROM subscriptions sub
		         WHERE sub.organisation_id = org.id ORDER BY sub.created_at DESC LIMIT 1) AS subscription_status
		FROM organisations org WHERE org.id = ?`
	err := r.db.Raw(selectSQL, id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ── Users ─────────────────────────────────────────────────────────────────

func (r *Repo) Users(o Options) ([]userRow, int64, error) {
	o.clamp()
	var total int64
	cq := r.db.Table("users u")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		cq = cq.Where("(lower(u.phone) LIKE lower(?) OR lower(u.first_name) LIKE lower(?) OR lower(u.last_name) LIKE lower(?) OR lower(COALESCE(u.email,'')) LIKE lower(?))",
			like, like, like, like)
	}
	if o.OrgID != "" {
		cq = cq.Where("u.organisation_id = ?", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		cq = cq.Where("u.store_id = ?", orgUUID(o.StoreID))
	}
	if o.Role != "" {
		cq = cq.Where("u.role = ?", o.Role)
	}
	if o.Status == "active" {
		cq = cq.Where("u.is_active = true")
	} else if o.Status == "inactive" {
		cq = cq.Where("u.is_active = false")
	} else if o.Status != "" {
		cq = cq.Where("u.employment_status = ?", o.Status)
	}
	if err := cq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.db.Table("users u").
		Select(`u.id, u.phone, u.email, u.first_name, u.last_name, u.role,
		        u.organisation_id, org.name AS organisation_name,
		        u.store_id, s.name AS store_name,
		        u.shift, u.employment_status, u.can_refund, u.can_discount,
		        u.can_view_reports, u.is_phone_verified, u.is_email_verified,
		        u.is_active, u.last_login, u.created_at`).
		Joins("LEFT JOIN organisations org ON org.id = u.organisation_id").
		Joins("LEFT JOIN stores s ON s.id = u.store_id")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(u.phone) LIKE lower(?) OR lower(u.first_name) LIKE lower(?) OR lower(u.last_name) LIKE lower(?) OR lower(COALESCE(u.email,'')) LIKE lower(?))",
			like, like, like, like)
	}
	if o.OrgID != "" {
		q = q.Where("u.organisation_id = ?", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		q = q.Where("u.store_id = ?", orgUUID(o.StoreID))
	}
	if o.Role != "" {
		q = q.Where("u.role = ?", o.Role)
	}
	if o.Status == "active" {
		q = q.Where("u.is_active = true")
	} else if o.Status == "inactive" {
		q = q.Where("u.is_active = false")
	} else if o.Status != "" {
		q = q.Where("u.employment_status = ?", o.Status)
	}
	var rows []userRow
	err := q.Order("u.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Stores ────────────────────────────────────────────────────────────────

func (r *Repo) Stores(o Options) ([]storeRow, int64, error) {
	o.clamp()
	var total int64
	cq := r.db.Table("stores s")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		cq = cq.Where("(lower(s.name) LIKE lower(?) OR lower(s.code) LIKE lower(?) OR lower(s.area) LIKE lower(?))", like, like, like)
	}
	if o.OrgID != "" {
		cq = cq.Where("s.organisation_id = ?", orgUUID(o.OrgID))
	}
	if o.Status != "" {
		cq = cq.Where("s.status = ?", o.Status)
	}
	if err := cq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.db.Table("stores s").
		Select(`s.id, s.name, s.code, s.phone, s.email, s.area, s.address,
		        s.status, s.is_active, s.vat_enabled, s.is_main_store,
		        s.organisation_id, org.name AS organisation_name,
		        (SELECT COUNT(*) FROM users u WHERE u.store_id = s.id AND u.is_active = true) AS staff_count,
		        (SELECT COALESCE(SUM(t.total), 0) FROM transactions t
		          WHERE t.store_id = s.id AND t.status IN ('paid','credit')
		            AND t.created_at >= now() - interval '30 days') AS revenue_30d,
		        (SELECT COUNT(*) FROM transactions t
		          WHERE t.store_id = s.id AND t.status IN ('paid','credit')
		            AND t.created_at >= now() - interval '30 days') AS txn_count_30d,
		        s.created_at`).
		Joins("LEFT JOIN organisations org ON org.id = s.organisation_id")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(s.name) LIKE lower(?) OR lower(s.code) LIKE lower(?) OR lower(s.area) LIKE lower(?))", like, like, like)
	}
	if o.OrgID != "" {
		q = q.Where("s.organisation_id = ?", orgUUID(o.OrgID))
	}
	if o.Status != "" {
		q = q.Where("s.status = ?", o.Status)
	}
	var rows []storeRow
	err := q.Order("s.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Products ──────────────────────────────────────────────────────────────

func (r *Repo) Products(o Options, category string, lowStock bool) ([]productRow, int64, error) {
	o.clamp()
	var total int64
	cq := r.db.Table("products p")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		cq = cq.Where("(lower(p.name) LIKE lower(?) OR lower(p.sku) LIKE lower(?) OR lower(p.barcode) LIKE lower(?))", like, like, like)
	}
	if o.OrgID != "" {
		cq = cq.Where("p.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		cq = cq.Where("p.store_id = ?", orgUUID(o.StoreID))
	}
	if category != "" {
		cq = cq.Where("p.category_id = ?", orgUUID(category))
	}
	if o.Status == "active" {
		cq = cq.Where("p.is_active = true")
	} else if o.Status == "inactive" {
		cq = cq.Where("p.is_active = false")
	}
	if lowStock {
		cq = cq.Where("p.is_service = false AND p.is_active = true AND p.stock <= p.reorder_point")
	}
	if err := cq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.db.Table("products p").
		Select(`p.id, p.store_id, s.name AS store_name, s.organisation_id,
		        org.name AS organisation_name, p.category_id, c.name AS category_name,
		        p.name, p.sku, p.barcode, p.cost_price, p.selling_price, p.stock,
		        p.reorder_point, p.is_active, p.is_service, p.image, p.tax_rate,
		        p.created_at, p.updated_at`).
		Joins("LEFT JOIN stores s ON s.id = p.store_id").
		Joins("LEFT JOIN organisations org ON org.id = s.organisation_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(p.name) LIKE lower(?) OR lower(p.sku) LIKE lower(?) OR lower(p.barcode) LIKE lower(?))", like, like, like)
	}
	if o.OrgID != "" {
		q = q.Where("p.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		q = q.Where("p.store_id = ?", orgUUID(o.StoreID))
	}
	if category != "" {
		q = q.Where("p.category_id = ?", orgUUID(category))
	}
	if o.Status == "active" {
		q = q.Where("p.is_active = true")
	} else if o.Status == "inactive" {
		q = q.Where("p.is_active = false")
	}
	if lowStock {
		q = q.Where("p.is_service = false AND p.is_active = true AND p.stock <= p.reorder_point")
	}
	var rows []productRow
	err := q.Order("p.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) ProductByID(id uuid.UUID) (*productRow, error) {
	var row productRow
	err := r.db.Table("products p").
		Select(`p.id, p.store_id, s.name AS store_name, s.organisation_id,
		        org.name AS organisation_name, p.category_id, c.name AS category_name,
		        p.name, p.sku, p.barcode, p.cost_price, p.selling_price, p.stock,
		        p.reorder_point, p.is_active, p.is_service, p.image, p.tax_rate,
		        p.description, p.created_at, p.updated_at`).
		Joins("LEFT JOIN stores s ON s.id = p.store_id").
		Joins("LEFT JOIN organisations org ON org.id = s.organisation_id").
		Joins("LEFT JOIN categories c ON c.id = p.category_id").
		Where("p.id = ?", id).Scan(&row).Error
	return &row, err
}

// Categories lists all product categories across the platform.
func (r *Repo) Categories() ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := r.db.Table("categories c").
		Select(`c.id, c.name, c.description, c.colour, c.created_at,
		        (SELECT COUNT(*) FROM products p WHERE p.category_id = c.id) AS product_count`).
		Order("c.name asc").Limit(500).Scan(&rows).Error
	return rows, err
}

// ── Transactions ──────────────────────────────────────────────────────────

// TxnFilter constrains the cross-tenant transaction list.
type TxnFilter struct {
	Options
	Method   string
	Customer string
	DateFrom string
	DateTo   string
}

func (r *Repo) Transactions(f TxnFilter) ([]txnRow, int64, error) {
	f.clamp()
	q := r.db.Table("transactions t").
		Select(`t.id, t.txn_number, t.store_id, s.name AS store_name,
		        t.organisation_id, org.name AS organisation_name,
		        t.payment_method, t.payment_reference, t.status, t.subtotal,
		        t.discount_amount, t.tax_amount, t.total, t.cost_total, t.profit,
		        t.cashier_id, COALESCE(cash.first_name || ' ' || cash.last_name, '') AS cashier_name,
		        t.till_number, t.customer_name, t.customer_phone, t.customer_id,
		        t.notes, t.channel, t.created_at,
		        (SELECT COALESCE(SUM(tl.qty), 0) FROM transaction_lines tl WHERE tl.transaction_id = t.id) AS item_count,
		        (SELECT COUNT(*) FROM transaction_lines tl WHERE tl.transaction_id = t.id) AS sku_count`).
		Joins("LEFT JOIN stores s ON s.id = t.store_id").
		Joins("LEFT JOIN organisations org ON org.id = t.organisation_id").
		Joins("LEFT JOIN users cash ON cash.id = t.cashier_id")
	base := r.db.Table("transactions t")
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(lower(t.txn_number) LIKE lower(?) OR lower(t.customer_name) LIKE lower(?) OR lower(t.payment_reference) LIKE lower(?))", like, like, like)
		base = base.Where("(lower(t.txn_number) LIKE lower(?) OR lower(t.customer_name) LIKE lower(?) OR lower(t.payment_reference) LIKE lower(?))", like, like, like)
	}
	if f.Status != "" {
		q = q.Where("t.status = ?", f.Status)
		base = base.Where("t.status = ?", f.Status)
	}
	if f.Method != "" {
		q = q.Where("t.payment_method = ?", f.Method)
		base = base.Where("t.payment_method = ?", f.Method)
	}
	if f.OrgID != "" {
		q = q.Where("t.organisation_id = ?", orgUUID(f.OrgID))
		base = base.Where("t.organisation_id = ?", orgUUID(f.OrgID))
	}
	if f.StoreID != "" {
		q = q.Where("t.store_id = ?", orgUUID(f.StoreID))
		base = base.Where("t.store_id = ?", orgUUID(f.StoreID))
	}
	if f.Customer != "" {
		q = q.Where("t.customer_id = ?", orgUUID(f.Customer))
		base = base.Where("t.customer_id = ?", orgUUID(f.Customer))
	}
	if f.DateFrom != "" {
		q = q.Where("t.created_at >= ?", f.DateFrom)
		base = base.Where("t.created_at >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("t.created_at < ?", f.DateTo+" 23:59:59")
		base = base.Where("t.created_at < ?", f.DateTo+" 23:59:59")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []txnRow
	err := q.Order("t.created_at desc").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) TransactionByID(id uuid.UUID) (*txnRow, error) {
	var row txnRow
	err := r.db.Table("transactions t").
		Select(`t.id, t.txn_number, t.store_id, s.name AS store_name,
		        t.organisation_id, org.name AS organisation_name,
		        t.payment_method, t.payment_reference, t.status, t.subtotal,
		        t.discount_amount, t.tax_amount, t.total, t.cost_total, t.profit,
		        t.cashier_id, COALESCE(cash.first_name || ' ' || cash.last_name, '') AS cashier_name,
		        t.till_number, t.customer_name, t.customer_phone, t.customer_id,
		        t.notes, t.channel, t.created_at,
		        (SELECT COALESCE(SUM(tl.qty), 0) FROM transaction_lines tl WHERE tl.transaction_id = t.id) AS item_count,
		        (SELECT COUNT(*) FROM transaction_lines tl WHERE tl.transaction_id = t.id) AS sku_count`).
		Joins("LEFT JOIN stores s ON s.id = t.store_id").
		Joins("LEFT JOIN organisations org ON org.id = t.organisation_id").
		Joins("LEFT JOIN users cash ON cash.id = t.cashier_id").
		Where("t.id = ?", id).Scan(&row).Error
	return &row, err
}

func (r *Repo) TransactionLines(txnID uuid.UUID) ([]txnLineRow, error) {
	var rows []txnLineRow
	err := r.db.Table("transaction_lines tl").
		Select("tl.id, tl.transaction_id, tl.product_id, tl.product_name, tl.product_sku, tl.unit_price, tl.unit_cost, tl.qty").
		Where("tl.transaction_id = ?", txnID).
		Order("tl.created_at asc").Scan(&rows).Error
	return rows, err
}

func (r *Repo) TransactionTotals(f TxnFilter) (map[string]int64, error) {
	q := r.db.Table("transactions t").
		Select(`
			COALESCE(SUM(t.total) FILTER (WHERE t.status IN ('paid','credit')), 0) AS gross,
			COALESCE(SUM(t.total) FILTER (WHERE t.status = 'refunded'), 0) AS refunded,
			COALESCE(SUM(t.profit) FILTER (WHERE t.status IN ('paid','credit')), 0) AS profit,
			COALESCE(SUM(t.tax_amount) FILTER (WHERE t.status IN ('paid','credit')), 0) AS tax,
			COUNT(*) FILTER (WHERE t.status IN ('paid','credit')) AS txns`)
	if f.Status != "" {
		q = q.Where("t.status = ?", f.Status)
	}
	if f.Method != "" {
		q = q.Where("t.payment_method = ?", f.Method)
	}
	if f.OrgID != "" {
		q = q.Where("t.organisation_id = ?", orgUUID(f.OrgID))
	}
	if f.StoreID != "" {
		q = q.Where("t.store_id = ?", orgUUID(f.StoreID))
	}
	if f.DateFrom != "" {
		q = q.Where("t.created_at >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		q = q.Where("t.created_at < ?", f.DateTo+" 23:59:59")
	}
	var out struct {
		Gross    int64 `gorm:"column:gross"`
		Refunded int64 `gorm:"column:refunded"`
		Profit   int64 `gorm:"column:profit"`
		Tax      int64 `gorm:"column:tax"`
		Txns     int64 `gorm:"column:txns"`
	}
	if err := q.Scan(&out).Error; err != nil {
		return nil, err
	}
	return map[string]int64{
		"gross_revenue": out.Gross,
		"refunded":      out.Refunded,
		"net_revenue":   out.Gross - out.Refunded,
		"profit":        out.Profit,
		"tax":           out.Tax,
		"transactions":  out.Txns,
	}, nil
}

// ── Customers ─────────────────────────────────────────────────────────────

func (r *Repo) Customers(o Options, segment string, hasCredit bool) ([]customerRow, int64, error) {
	o.clamp()
	q := r.db.Table("customers c").
		Select(`c.id, c.name, c.phone, c.email, c.segment, c.total_spent, c.last_visit,
		        c.avg_ticket, c.open_credit, c.credit_limit, c.avatar_hue, c.notes,
		        c.is_active, c.store_id, s.name AS store_name, s.organisation_id,
		        org.name AS organisation_name, c.created_at`).
		Joins("LEFT JOIN stores s ON s.id = c.store_id").
		Joins("LEFT JOIN organisations org ON org.id = s.organisation_id")
	base := r.db.Table("customers c")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(c.name) LIKE lower(?) OR lower(COALESCE(c.phone,'')) LIKE lower(?))", like, like)
		base = base.Where("(lower(c.name) LIKE lower(?) OR lower(COALESCE(c.phone,'')) LIKE lower(?))", like, like)
	}
	if segment != "" {
		q = q.Where("c.segment = ?", segment)
		base = base.Where("c.segment = ?", segment)
	}
	if o.OrgID != "" {
		q = q.Where("c.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
		base = base.Where("c.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		q = q.Where("c.store_id = ?", orgUUID(o.StoreID))
		base = base.Where("c.store_id = ?", orgUUID(o.StoreID))
	}
	if hasCredit {
		q = q.Where("c.open_credit > 0 AND c.is_active = true")
		base = base.Where("c.open_credit > 0 AND c.is_active = true")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []customerRow
	err := q.Order("c.total_spent desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Credits ───────────────────────────────────────────────────────────────

func (r *Repo) Credits(o Options, customer string, overdue bool) ([]creditRow, int64, error) {
	o.clamp()
	q := r.db.Table("credit_tabs t").
		Select(`t.id, t.customer_id, c.name AS customer_name, c.phone AS customer_phone,
		        t.transaction_id, t.amount, t.amount_paid,
		        GREATEST(t.amount - t.amount_paid, 0) AS balance,
		        t.status, t.due_date, t.cashier_id, t.till_number, t.write_off_reason,
		        s.name AS store_name, org.name AS organisation_name, t.created_at`).
		Joins("LEFT JOIN customers c ON c.id = t.customer_id").
		Joins("LEFT JOIN stores s ON s.id = t.store_id").
		Joins("LEFT JOIN organisations org ON org.id = s.organisation_id")
	base := r.db.Table("credit_tabs t")
	if o.Status != "" {
		q = q.Where("t.status = ?", o.Status)
		base = base.Where("t.status = ?", o.Status)
	}
	if customer != "" {
		q = q.Where("t.customer_id = ?", orgUUID(customer))
		base = base.Where("t.customer_id = ?", orgUUID(customer))
	}
	if o.OrgID != "" {
		q = q.Where("t.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
		base = base.Where("t.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		q = q.Where("t.store_id = ?", orgUUID(o.StoreID))
		base = base.Where("t.store_id = ?", orgUUID(o.StoreID))
	}
	if overdue {
		q = q.Where("t.status IN ('open','partially_paid') AND t.due_date < now()")
		base = base.Where("t.status IN ('open','partially_paid') AND t.due_date < now()")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []creditRow
	err := q.Order("t.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Suppliers ─────────────────────────────────────────────────────────────

func (r *Repo) Suppliers(o Options) ([]supplierRow, int64, error) {
	o.clamp()
	q := r.db.Table("suppliers sup").
		Select(`sup.id, sup.name, sup.category, sup.rep, sup.phone, sup.email, sup.address,
		        sup.payment_terms, sup.status, sup.notes,
		        (SELECT GREATEST(COALESCE((SELECT SUM(d.total_value) FROM supplier_deliveries d WHERE d.supplier_id = sup.id), 0) -
		                COALESCE((SELECT SUM(p.amount) FROM supplier_payments p WHERE p.supplier_id = sup.id), 0), 0))
		        AS outstanding,
		        s.name AS store_name, org.name AS organisation_name, sup.created_at`).
		Joins("LEFT JOIN stores s ON s.id = sup.store_id").
		Joins("LEFT JOIN organisations org ON org.id = sup.organisation_id")
	base := r.db.Table("suppliers sup")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(sup.name) LIKE lower(?) OR lower(COALESCE(sup.category,'')) LIKE lower(?) OR lower(COALESCE(sup.rep,'')) LIKE lower(?))", like, like, like)
		base = base.Where("(lower(sup.name) LIKE lower(?) OR lower(COALESCE(sup.category,'')) LIKE lower(?) OR lower(COALESCE(sup.rep,'')) LIKE lower(?))", like, like, like)
	}
	if o.Status != "" {
		q = q.Where("sup.status = ?", o.Status)
		base = base.Where("sup.status = ?", o.Status)
	}
	if o.OrgID != "" {
		q = q.Where("sup.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
		base = base.Where("sup.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []supplierRow
	err := q.Order("sup.name asc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) Deliveries(o Options, supplier string, unpaid bool) ([]deliveryRow, int64, error) {
	o.clamp()
	q := r.db.Table("supplier_deliveries d").
		Select(`d.id, d.supplier_id, sup.name AS supplier_name, d.invoice_number,
		        d.delivery_date, d.items_count, d.total_value, d.status, d.is_paid, d.notes,
		        s.name AS store_name, org.name AS organisation_name, d.created_at`).
		Joins("LEFT JOIN suppliers sup ON sup.id = d.supplier_id").
		Joins("LEFT JOIN stores s ON s.id = d.store_id").
		Joins("LEFT JOIN organisations org ON org.id = d.organisation_id")
	base := r.db.Table("supplier_deliveries d")
	if supplier != "" {
		q = q.Where("d.supplier_id = ?", orgUUID(supplier))
		base = base.Where("d.supplier_id = ?", orgUUID(supplier))
	}
	if unpaid {
		q = q.Where("d.is_paid = false")
		base = base.Where("d.is_paid = false")
	}
	if o.OrgID != "" {
		q = q.Where("d.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
		base = base.Where("d.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []deliveryRow
	err := q.Order("d.delivery_date desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) Payments(o Options, supplier string) ([]paymentRow, int64, error) {
	o.clamp()
	q := r.db.Table("supplier_payments p").
		Select(`p.id, p.supplier_id, sup.name AS supplier_name, p.delivery_id, p.amount,
		        p.payment_date, p.payment_method, p.reference, p.notes,
		        s.name AS store_name, org.name AS organisation_name, p.created_at`).
		Joins("LEFT JOIN suppliers sup ON sup.id = p.supplier_id").
		Joins("LEFT JOIN stores s ON s.id = p.store_id").
		Joins("LEFT JOIN organisations org ON org.id = p.organisation_id")
	base := r.db.Table("supplier_payments p")
	if supplier != "" {
		q = q.Where("p.supplier_id = ?", orgUUID(supplier))
		base = base.Where("p.supplier_id = ?", orgUUID(supplier))
	}
	if o.OrgID != "" {
		q = q.Where("p.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
		base = base.Where("p.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []paymentRow
	err := q.Order("p.payment_date desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Expenses ──────────────────────────────────────────────────────────────

func (r *Repo) Expenses(o Options, category, dateFrom, dateTo string) ([]expenseRow, int64, error) {
	o.clamp()
	q := r.db.Table("expenses e").
		Select(`e.id, e.title, e.category, e.amount, e.payment_method, e.payment_reference,
		        e.notes, e.receipt_url, e.recorded_by,
		        s.name AS store_name, org.name AS organisation_name, e.created_at`).
		Joins("LEFT JOIN stores s ON s.id = e.store_id").
		Joins("LEFT JOIN organisations org ON org.id = e.organisation_id")
	base := r.db.Table("expenses e")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(e.title) LIKE lower(?) OR lower(e.notes) LIKE lower(?) OR lower(e.payment_reference) LIKE lower(?))", like, like, like)
		base = base.Where("(lower(e.title) LIKE lower(?) OR lower(e.notes) LIKE lower(?) OR lower(e.payment_reference) LIKE lower(?))", like, like, like)
	}
	if category != "" {
		q = q.Where("e.category = ?", category)
		base = base.Where("e.category = ?", category)
	}
	if o.OrgID != "" {
		q = q.Where("e.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
		base = base.Where("e.store_id IN (SELECT id FROM stores WHERE organisation_id = ?)", orgUUID(o.OrgID))
	}
	if o.StoreID != "" {
		q = q.Where("e.store_id = ?", orgUUID(o.StoreID))
		base = base.Where("e.store_id = ?", orgUUID(o.StoreID))
	}
	if dateFrom != "" {
		q = q.Where("e.created_at >= ?", dateFrom)
		base = base.Where("e.created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		q = q.Where("e.created_at < ?", dateTo+" 23:59:59")
		base = base.Where("e.created_at < ?", dateTo+" 23:59:59")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []expenseRow
	err := q.Order("e.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Reviews ───────────────────────────────────────────────────────────────

func (r *Repo) Reviews(status string, rating int, o Options) ([]reviewRow, int64, error) {
	o.clamp()
	q := r.db.Table("store_reviews rv").
		Select(`rv.id, rv.organisation_id, org.name AS organisation_name,
		        rv.business_type_display, rv.region_display, rv.reviewer_id,
		        COALESCE(u.first_name || ' ' || u.last_name, u.phone, 'Anonymous') AS reviewer_name,
		        rv.rating, rv.title, rv.body, rv.trigger, rv.is_public, rv.created_at`).
		Joins("LEFT JOIN organisations org ON org.id = rv.organisation_id").
		Joins("LEFT JOIN users u ON u.id = rv.reviewer_id")
	base := r.db.Table("store_reviews rv")
	if status == "public" {
		q = q.Where("rv.is_public = true")
		base = base.Where("rv.is_public = true")
	} else if status == "hidden" {
		q = q.Where("rv.is_public = false")
		base = base.Where("rv.is_public = false")
	}
	if rating > 0 {
		q = q.Where("rv.rating = ?", rating)
		base = base.Where("rv.rating = ?", rating)
	}
	if o.OrgID != "" {
		q = q.Where("rv.organisation_id = ?", orgUUID(o.OrgID))
		base = base.Where("rv.organisation_id = ?", orgUUID(o.OrgID))
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []reviewRow
	err := q.Order("rv.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── Notes & notifications ─────────────────────────────────────────────────

func (r *Repo) Notes(o Options) ([]noteRow, int64, error) {
	o.clamp()
	q := r.db.Table("notes n").
		Select(`n.id, n.title, n.content, n.tags::text AS tags, n.created_by,
		        COALESCE(u.first_name || ' ' || u.last_name, u.phone, '') AS created_by_name,
		        s.name AS store_name, org.name AS organisation_name, n.created_at, n.updated_at`).
		Joins("LEFT JOIN users u ON u.id = n.created_by").
		Joins("LEFT JOIN stores s ON s.id = n.store_id").
		Joins("LEFT JOIN organisations org ON org.id = n.organisation_id")
	base := r.db.Table("notes n")
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(n.title) LIKE lower(?) OR lower(COALESCE(n.content,'')) LIKE lower(?))", like, like)
		base = base.Where("(lower(n.title) LIKE lower(?) OR lower(COALESCE(n.content,'')) LIKE lower(?))", like, like)
	}
	if o.OrgID != "" {
		q = q.Where("n.organisation_id = ?", orgUUID(o.OrgID))
		base = base.Where("n.organisation_id = ?", orgUUID(o.OrgID))
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []noteRow
	err := q.Order("n.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) Notifications(o Options, category string) ([]notificationRow, int64, error) {
	o.clamp()
	q := r.db.Table("notifications n").
		Select(`n.id, n.category, n.title, n.body, n.link, n.read,
		        n.organisation_id, org.name AS organisation_name,
		        n.store_id, s.name AS store_name, n.created_at`).
		Joins("LEFT JOIN organisations org ON org.id = n.organisation_id").
		Joins("LEFT JOIN stores s ON s.id = n.store_id")
	base := r.db.Table("notifications n")
	if category != "" {
		q = q.Where("n.category = ?", category)
		base = base.Where("n.category = ?", category)
	}
	if o.OrgID != "" {
		q = q.Where("n.organisation_id = ?", orgUUID(o.OrgID))
		base = base.Where("n.organisation_id = ?", orgUUID(o.OrgID))
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []notificationRow
	err := q.Order("n.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

// ── AI usage ──────────────────────────────────────────────────────────────

type aiUsageRow struct {
	OrgName   string `gorm:"column:organisation_name"`
	OrgID     string `gorm:"column:organisation_id"`
	Year      int    `gorm:"column:year"`
	Month     int    `gorm:"column:month"`
	Used      int    `gorm:"column:used"`
	Allocated int    `gorm:"column:allocated"`
}

func (r *Repo) AIUsage(month int, orgID string) ([]aiUsageRow, error) {
	q := r.db.Table("ai_credits ac").
		Select(`ac.organisation_id, org.name AS organisation_name,
		        ac.year, ac.month, ac.used, ac.allocated`).
		Joins("LEFT JOIN organisations org ON org.id = ac.organisation_id")
	if month > 0 {
		q = q.Where("ac.month = ?", month)
	}
	if orgID != "" {
		q = q.Where("ac.organisation_id = ?", orgUUID(orgID))
	}
	var rows []aiUsageRow
	err := q.Order("ac.year desc, ac.month desc, ac.used desc").Limit(200).Scan(&rows).Error
	return rows, err
}

// ── Write helpers ─────────────────────────────────────────────────────────

// Exists reports whether a row with id exists in table.
func (r *Repo) Exists(table string, id uuid.UUID) (bool, error) {
	var n int64
	err := r.db.Table(table).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

// Update applies fields to a row and reports whether a row matched.
func (r *Repo) Update(table string, id uuid.UUID, fields map[string]interface{}) (bool, error) {
	res := r.db.Table(table).Where("id = ?", id).Updates(fields)
	return res.RowsAffected > 0, res.Error
}

// Delete removes a row and reports whether a row matched.
func (r *Repo) Delete(table string, id uuid.UUID) (bool, error) {
	res := r.db.Table(table).Where("id = ?", id).Delete(nil)
	return res.RowsAffected > 0, res.Error
}

func (r *Repo) OrgStores(id uuid.UUID, o Options) ([]storeRow, int64, error) {
	o.clamp()
	var total int64
	if err := r.db.Table("stores").Where("organisation_id = ?", id).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := r.db.Table("stores s").
		Select(`s.id, s.name, s.code, s.phone, s.email, s.area, s.address,
		        s.status, s.is_active, s.vat_enabled, s.is_main_store,
		        s.organisation_id, org.name AS organisation_name,
		        (SELECT COUNT(*) FROM users u WHERE u.store_id = s.id AND u.is_active = true) AS staff_count,
		        (SELECT COALESCE(SUM(t.total), 0) FROM transactions t
		          WHERE t.store_id = s.id AND t.status IN ('paid','credit')) AS revenue_30d,
		        s.created_at`).
		Joins("LEFT JOIN organisations org ON org.id = s.organisation_id").
		Where("s.organisation_id = ?", id)
	if o.Search != "" {
		like := "%" + o.Search + "%"
		q = q.Where("(lower(s.name) LIKE lower(?) OR lower(s.code) LIKE lower(?))", like, like)
	}
	var rows []storeRow
	err := q.Order("s.created_at desc").Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).Scan(&rows).Error
	return rows, total, err
}

func (r *Repo) OrgUsersAll(id uuid.UUID) ([]userRow, error) {
	var rows []userRow
	err := r.db.Table("users u").
		Select(`u.id, u.phone, u.email, u.first_name, u.last_name, u.role,
		        u.organisation_id, org.name AS organisation_name,
		        u.store_id, s.name AS store_name,
		        u.shift, u.employment_status, u.can_refund, u.can_discount,
		        u.can_view_reports, u.is_phone_verified, u.is_email_verified,
		        u.is_active, u.last_login, u.created_at`).
		Joins("LEFT JOIN organisations org ON org.id = u.organisation_id").
		Joins("LEFT JOIN stores s ON s.id = u.store_id").
		Where("u.organisation_id = ?", id).
		Order("u.created_at desc").Scan(&rows).Error
	return rows, err
}

// ── Dashboard aggregates ───────────────────────────────────────────────────

type dashRow struct {
	Organisations  int64 `gorm:"column:organisations"`
	ActiveOrgs     int64 `gorm:"column:active_orgs"`
	Stores         int64 `gorm:"column:stores"`
	Users          int64 `gorm:"column:users"`
	Staff          int64 `gorm:"column:staff"`
	ActiveStaff    int64 `gorm:"column:active_staff"`
	Products       int64 `gorm:"column:products"`
	ActiveProducts int64 `gorm:"column:active_products"`
	Customers      int64 `gorm:"column:customers"`
	CreditTabs     int64 `gorm:"column:credit_tabs"`
	OpenCreditTab  int64 `gorm:"column:open_credit_tabs"`
	Reviews        int64 `gorm:"column:reviews"`
	PendingReviews int64 `gorm:"column:pending_reviews"`
}

func (r *Repo) DashboardCounts() (*dashRow, error) {
	var out dashRow
	selectSQL := `
		SELECT
		  (SELECT COUNT(*) FROM organisations) AS organisations,
		  (SELECT COUNT(*) FROM organisations WHERE is_active = true) AS active_orgs,
		  (SELECT COUNT(*) FROM stores) AS stores,
		  (SELECT COUNT(*) FROM users) AS users,
		  (SELECT COUNT(*) FROM users WHERE role = 'staff') AS staff,
		  (SELECT COUNT(*) FROM users WHERE role = 'staff' AND is_active = true) AS active_staff,
		  (SELECT COUNT(*) FROM products) AS products,
		  (SELECT COUNT(*) FROM products WHERE is_active = true) AS active_products,
		  (SELECT COUNT(*) FROM customers) AS customers,
		  (SELECT COUNT(*) FROM credit_tabs) AS credit_tabs,
		  (SELECT COUNT(*) FROM credit_tabs WHERE status IN ('open','partially_paid')) AS open_credit_tabs,
		  (SELECT COUNT(*) FROM store_reviews) AS reviews,
		  (SELECT COUNT(*) FROM store_reviews WHERE is_public = false) AS pending_reviews`
	err := r.db.Raw(selectSQL).Scan(&out).Error
	return &out, err
}

// money rows for dashboard/analytics
type moneyRow struct {
	Gross     int64 `gorm:"column:gross"`
	Refunded  int64 `gorm:"column:refunded"`
	Profit    int64 `gorm:"column:profit"`
	Tax       int64 `gorm:"column:tax"`
	Txns      int64 `gorm:"column:txns"`
	Expenses  int64 `gorm:"column:expenses"`
	CreditOut int64 `gorm:"column:credit_out"`
}

func (r *Repo) PlatformMoney() (*moneyRow, error) {
	var out moneyRow
	selectSQL := `
		SELECT
		  COALESCE((SELECT SUM(total) FROM transactions WHERE status IN ('paid','credit')), 0) AS gross,
		  COALESCE((SELECT SUM(total) FROM transactions WHERE status = 'refunded'), 0) AS refunded,
		  COALESCE((SELECT SUM(profit) FROM transactions WHERE status IN ('paid','credit')), 0) AS profit,
		  COALESCE((SELECT SUM(tax_amount) FROM transactions WHERE status IN ('paid','credit')), 0) AS tax,
		  (SELECT COUNT(*) FROM transactions WHERE status IN ('paid','credit')) AS txns,
		  COALESCE((SELECT SUM(amount) FROM expenses), 0) AS expenses,
		  COALESCE((SELECT SUM(GREATEST(amount - amount_paid, 0)) FROM credit_tabs WHERE status IN ('open','partially_paid')), 0) AS credit_out`
	err := r.db.Raw(selectSQL).Scan(&out).Error
	return &out, err
}

// todayRow is a compact per-day series for the admin dashboard.
type todayRow struct {
	Day   string `gorm:"column:day"`
	Gross int64  `gorm:"column:gross"`
	Txns  int64  `gorm:"column:txns"`
}

func (r *Repo) RevenueSeries(days int) ([]todayRow, error) {
	var rows []todayRow
	err := r.db.Raw(`
		SELECT to_char(d.day, 'YYYY-MM-DD') AS day,
		       COALESCE(SUM(t.total), 0) AS gross,
		       COUNT(t.id) AS txns
		FROM generate_series(current_date - (? || ' days')::int, current_date, interval '1 day') AS d(day)
		LEFT JOIN transactions t ON t.created_at::date = d.day::date AND t.status IN ('paid','credit')
		GROUP BY d.day ORDER BY d.day ASC`, days).Scan(&rows).Error
	return rows, err
}

func (r *Repo) SubscriptionsRevenue() (*subscriptionsSummaryRow, error) {
	var out subscriptionsSummaryRow
	err := r.db.Raw(`
		SELECT
		  (SELECT COUNT(*) FROM subscriptions) AS total,
		  (SELECT COUNT(*) FROM subscriptions WHERE status = 'active') AS active,
		  (SELECT COUNT(*) FROM subscriptions WHERE status = 'trial') AS trial,
		  (SELECT COUNT(*) FROM subscriptions WHERE status IN ('expired','cancelled')) AS closed,
		  COALESCE((SELECT SUM(amount_paid) FROM subscriptions WHERE status IN ('active','trial')), 0) AS collected`).
		Scan(&out).Error
	return &out, err
}

type subscriptionsSummaryRow struct {
	Total     int64 `gorm:"column:total"`
	Active    int64 `gorm:"column:active"`
	Trial     int64 `gorm:"column:trial"`
	Closed    int64 `gorm:"column:closed"`
	Collected int64 `gorm:"column:collected"`
}
