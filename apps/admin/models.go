// Package admin exposes platform-level management endpoints. Every route is
// gated behind Auth + RequireAdmin, and all queries are cross-tenant (they do
// not read org/store from the JWT — admins manage the whole platform).
package admin

import "time"

// Scan projections used by repository aggregate queries. These match column
// aliases defined in the associated SQL SELECT statements.

// orgRow is one row for the organisations list plus cheap aggregates.
type orgRow struct {
	ID              string     `gorm:"column:id"`
	Name            string     `gorm:"column:name"`
	Slug            string     `gorm:"column:slug"`
	LegalName       string     `gorm:"column:legal_name"`
	TIN             string     `gorm:"column:tin"`
	BusinessType    string     `gorm:"column:business_type"`
	Region          string     `gorm:"column:region"`
	District        string     `gorm:"column:district"`
	Phone           string     `gorm:"column:phone"`
	Email           string     `gorm:"column:email"`
	Plan            string     `gorm:"column:plan"`
	MaxStores       int        `gorm:"column:max_stores"`
	TrialEndsAt     *time.Time `gorm:"column:trial_ends_at"`
	IsActive        bool       `gorm:"column:is_active"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	StoreCount      int64      `gorm:"column:store_count"`
	UserCount       int64      `gorm:"column:user_count"`
	ActiveUserCount int64      `gorm:"column:active_user_count"`
	Revenue30d      int64      `gorm:"column:revenue_30d"`
	TxnCount30d     int64      `gorm:"column:txn_count_30d"`
	Subscription    string     `gorm:"column:subscription_status"`
}

// userRow is one row for the users list.
type userRow struct {
	ID               string     `gorm:"column:id"`
	Phone            string     `gorm:"column:phone"`
	Email            *string    `gorm:"column:email"`
	FirstName        string     `gorm:"column:first_name"`
	LastName         string     `gorm:"column:last_name"`
	Role             string     `gorm:"column:role"`
	OrganisationID   *string    `gorm:"column:organisation_id"`
	OrganisationName string     `gorm:"column:organisation_name"`
	StoreID          *string    `gorm:"column:store_id"`
	StoreName        string     `gorm:"column:store_name"`
	Shift            string     `gorm:"column:shift"`
	EmploymentStatus string     `gorm:"column:employment_status"`
	CanRefund        bool       `gorm:"column:can_refund"`
	CanDiscount      bool       `gorm:"column:can_discount"`
	CanViewReports   bool       `gorm:"column:can_view_reports"`
	IsPhoneVerified  bool       `gorm:"column:is_phone_verified"`
	IsEmailVerified  bool       `gorm:"column:is_email_verified"`
	IsActive         bool       `gorm:"column:is_active"`
	LastLogin        *time.Time `gorm:"column:last_login"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
}

// storeRow is one row for the stores list.
type storeRow struct {
	ID               string    `gorm:"column:id"`
	Name             string    `gorm:"column:name"`
	Code             string    `gorm:"column:code"`
	Phone            string    `gorm:"column:phone"`
	Email            string    `gorm:"column:email"`
	Area             string    `gorm:"column:area"`
	Address          string    `gorm:"column:address"`
	Status           string    `gorm:"column:status"`
	IsActive         bool      `gorm:"column:is_active"`
	VATEnabled       bool      `gorm:"column:vat_enabled"`
	IsMainStore      bool      `gorm:"column:is_main_store"`
	OrganisationID   string    `gorm:"column:organisation_id"`
	OrganisationName string    `gorm:"column:organisation_name"`
	StaffCount       int64     `gorm:"column:staff_count"`
	Revenue30d       int64     `gorm:"column:revenue_30d"`
	TxnCount30d      int64     `gorm:"column:txn_count_30d"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

// productRow is one row for the cross-tenant products list.
type productRow struct {
	ID           string    `gorm:"column:id"`
	StoreID      string    `gorm:"column:store_id"`
	StoreName    string    `gorm:"column:store_name"`
	OrgID        string    `gorm:"column:organisation_id"`
	OrgName      string    `gorm:"column:organisation_name"`
	CategoryID   *string   `gorm:"column:category_id"`
	CategoryName string    `gorm:"column:category_name"`
	Name         string    `gorm:"column:name"`
	SKU          *string   `gorm:"column:sku"`
	Barcode      *string   `gorm:"column:barcode"`
	CostPrice    int64     `gorm:"column:cost_price"`
	SellingPrice int64     `gorm:"column:selling_price"`
	Stock        int       `gorm:"column:stock"`
	ReorderPoint int       `gorm:"column:reorder_point"`
	IsActive     bool      `gorm:"column:is_active"`
	IsService    bool      `gorm:"column:is_service"`
	Image        string    `gorm:"column:image"`
	TaxRate      float64   `gorm:"column:tax_rate"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// txnRow is one row for the cross-tenant transaction list.
type txnRow struct {
	ID               string    `gorm:"column:id"`
	TxnNumber        string    `gorm:"column:txn_number"`
	StoreID          string    `gorm:"column:store_id"`
	StoreName        string    `gorm:"column:store_name"`
	OrgID            string    `gorm:"column:organisation_id"`
	OrgName          string    `gorm:"column:organisation_name"`
	PaymentMethod    string    `gorm:"column:payment_method"`
	PaymentReference string    `gorm:"column:payment_reference"`
	Status           string    `gorm:"column:status"`
	Subtotal         int64     `gorm:"column:subtotal"`
	DiscountAmount   int64     `gorm:"column:discount_amount"`
	TaxAmount        int64     `gorm:"column:tax_amount"`
	Total            int64     `gorm:"column:total"`
	CostTotal        int64     `gorm:"column:cost_total"`
	Profit           int64     `gorm:"column:profit"`
	CashierID        string    `gorm:"column:cashier_id"`
	CashierName      string    `gorm:"column:cashier_name"`
	TillNumber       string    `gorm:"column:till_number"`
	CustomerName     string    `gorm:"column:customer_name"`
	CustomerPhone    string    `gorm:"column:customer_phone"`
	CustomerID       *string   `gorm:"column:customer_id"`
	ItemCount        int       `gorm:"column:item_count"`
	SkuCount         int       `gorm:"column:sku_count"`
	Notes            string    `gorm:"column:notes"`
	Channel          string    `gorm:"column:channel"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

// txnLineRow is a transaction line for the detail view.
type txnLineRow struct {
	ID            string  `gorm:"column:id"`
	TransactionID string  `gorm:"column:transaction_id"`
	ProductID     *string `gorm:"column:product_id"`
	ProductName   string  `gorm:"column:product_name"`
	ProductSKU    string  `gorm:"column:product_sku"`
	UnitPrice     int64   `gorm:"column:unit_price"`
	UnitCost      int64   `gorm:"column:unit_cost"`
	Qty           int     `gorm:"column:qty"`
}

// customerRow is one row for the cross-tenant customers list.
type customerRow struct {
	ID          string     `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Phone       *string    `gorm:"column:phone"`
	Email       string     `gorm:"column:email"`
	Segment     string     `gorm:"column:segment"`
	TotalSpent  int64      `gorm:"column:total_spent"`
	LastVisit   *time.Time `gorm:"column:last_visit"`
	AvgTicket   int64      `gorm:"column:avg_ticket"`
	OpenCredit  int64      `gorm:"column:open_credit"`
	CreditLimit *int64     `gorm:"column:credit_limit"`
	AvatarHue   int        `gorm:"column:avatar_hue"`
	Notes       string     `gorm:"column:notes"`
	IsActive    bool       `gorm:"column:is_active"`
	StoreID     string     `gorm:"column:store_id"`
	StoreName   string     `gorm:"column:store_name"`
	OrgID       string     `gorm:"column:organisation_id"`
	OrgName     string     `gorm:"column:organisation_name"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

// creditRow is one row for the cross-tenant credit tabs list.
type creditRow struct {
	ID             string     `gorm:"column:id"`
	CustomerID     string     `gorm:"column:customer_id"`
	CustomerName   string     `gorm:"column:customer_name"`
	CustomerPhone  *string    `gorm:"column:customer_phone"`
	TransactionID  *string    `gorm:"column:transaction_id"`
	Amount         int64      `gorm:"column:amount"`
	AmountPaid     int64      `gorm:"column:amount_paid"`
	Balance        int64      `gorm:"column:balance"`
	Status         string     `gorm:"column:status"`
	DueDate        *time.Time `gorm:"column:due_date"`
	CashierID      string     `gorm:"column:cashier_id"`
	TillNumber     string     `gorm:"column:till_number"`
	WriteOffReason string     `gorm:"column:write_off_reason"`
	StoreName      string     `gorm:"column:store_name"`
	OrgName        string     `gorm:"column:organisation_name"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

// supplierRow is one row for the cross-tenant suppliers list.
type supplierRow struct {
	ID           string    `gorm:"column:id"`
	Name         string    `gorm:"column:name"`
	Category     string    `gorm:"column:category"`
	Rep          string    `gorm:"column:rep"`
	Phone        string    `gorm:"column:phone"`
	Email        string    `gorm:"column:email"`
	Address      string    `gorm:"column:address"`
	PaymentTerms int       `gorm:"column:payment_terms"`
	Status       string    `gorm:"column:status"`
	Notes        string    `gorm:"column:notes"`
	Outstanding  int64     `gorm:"column:outstanding"`
	StoreName    string    `gorm:"column:store_name"`
	OrgName      string    `gorm:"column:organisation_name"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

// deliveryRow is one row for the deliveries list.
type deliveryRow struct {
	ID            string    `gorm:"column:id"`
	SupplierID    string    `gorm:"column:supplier_id"`
	SupplierName  string    `gorm:"column:supplier_name"`
	InvoiceNumber string    `gorm:"column:invoice_number"`
	DeliveryDate  time.Time `gorm:"column:delivery_date"`
	ItemsCount    int       `gorm:"column:items_count"`
	TotalValue    int64     `gorm:"column:total_value"`
	Status        string    `gorm:"column:status"`
	IsPaid        bool      `gorm:"column:is_paid"`
	Notes         string    `gorm:"column:notes"`
	StoreName     string    `gorm:"column:store_name"`
	OrgName       string    `gorm:"column:organisation_name"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

// paymentRow is one row for the supplier payments list.
type paymentRow struct {
	ID            string    `gorm:"column:id"`
	SupplierID    string    `gorm:"column:supplier_id"`
	SupplierName  string    `gorm:"column:supplier_name"`
	DeliveryID    *string   `gorm:"column:delivery_id"`
	Amount        int64     `gorm:"column:amount"`
	PaymentDate   time.Time `gorm:"column:payment_date"`
	PaymentMethod string    `gorm:"column:payment_method"`
	Reference     string    `gorm:"column:reference"`
	Notes         string    `gorm:"column:notes"`
	StoreName     string    `gorm:"column:store_name"`
	OrgName       string    `gorm:"column:organisation_name"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

// expenseRow is one row for the cross-tenant expenses list.
type expenseRow struct {
	ID               string    `gorm:"column:id"`
	Title            string    `gorm:"column:title"`
	Category         string    `gorm:"column:category"`
	Amount           int64     `gorm:"column:amount"`
	PaymentMethod    string    `gorm:"column:payment_method"`
	PaymentReference string    `gorm:"column:payment_reference"`
	Notes            string    `gorm:"column:notes"`
	ReceiptURL       string    `gorm:"column:receipt_url"`
	RecordedBy       string    `gorm:"column:recorded_by"`
	StoreName        string    `gorm:"column:store_name"`
	OrgName          string    `gorm:"column:organisation_name"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

// reviewRow is one row for the review moderation list.
type reviewRow struct {
	ID                  string    `gorm:"column:id"`
	OrganisationID      string    `gorm:"column:organisation_id"`
	OrganisationName    string    `gorm:"column:organisation_name"`
	BusinessTypeDisplay string    `gorm:"column:business_type_display"`
	RegionDisplay       string    `gorm:"column:region_display"`
	ReviewerID          *string   `gorm:"column:reviewer_id"`
	ReviewerName        string    `gorm:"column:reviewer_name"`
	Rating              int       `gorm:"column:rating"`
	Title               string    `gorm:"column:title"`
	Body                string    `gorm:"column:body"`
	Trigger             string    `gorm:"column:trigger"`
	IsPublic            bool      `gorm:"column:is_public"`
	CreatedAt           time.Time `gorm:"column:created_at"`
}

// noteRow is one row for the notes list.
type noteRow struct {
	ID            string    `gorm:"column:id"`
	Title         string    `gorm:"column:title"`
	Content       string    `gorm:"column:content"`
	Tags          string    `gorm:"column:tags"`
	CreatedBy     *string   `gorm:"column:created_by"`
	CreatedByName string    `gorm:"column:created_by_name"`
	StoreName     string    `gorm:"column:store_name"`
	OrgName       string    `gorm:"column:organisation_name"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

// notificationRow is one row for the notifications list.
type notificationRow struct {
	ID             string    `gorm:"column:id"`
	Category       string    `gorm:"column:category"`
	Title          string    `gorm:"column:title"`
	Body           string    `gorm:"column:body"`
	Link           string    `gorm:"column:link"`
	Read           bool      `gorm:"column:read"`
	OrganisationID string    `gorm:"column:organisation_id"`
	OrgName        string    `gorm:"column:organisation_name"`
	StoreID        string    `gorm:"column:store_id"`
	StoreName      string    `gorm:"column:store_name"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}
