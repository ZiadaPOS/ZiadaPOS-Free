package admin

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes cross-tenant platform admin endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds an admin Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func writeErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, errNotFound) {
		response.NotFound(c, "Not found")
		return
	}
	if errors.Is(err, errInvalid) {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	response.Internal(c, err.Error())
}

var errNotFound = errors.New("not found")
var errInvalid = errors.New("invalid")

func (h *Handler) opts(c *gin.Context) Options {
	return Options{
		Search:   c.Query("search"),
		OrgID:    c.Query("org_id"),
		StoreID:  c.Query("store_id"),
		Status:   c.Query("status"),
		Role:     c.Query("role"),
		Page:     atoi(c.DefaultQuery("page", "1")),
		PageSize: atoi(c.DefaultQuery("page_size", "20")),
	}
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func (h *Handler) id(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid id", nil)
		return uuid.Nil, false
	}
	return id, true
}

// ── Dashboard ─────────────────────────────────────────────────────────────

// Dashboard godoc: GET /admin/dashboard
func (h *Handler) Dashboard(c *gin.Context) {
	counts, err := h.svc.repo.DashboardCounts()
	if err != nil {
		writeErr(c, err)
		return
	}
	money, err := h.svc.repo.PlatformMoney()
	if err != nil {
		writeErr(c, err)
		return
	}
	series, err := h.svc.repo.RevenueSeries(14)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{
		"counts":         counts,
		"money":          money,
		"revenue_series": series,
	})
}

// ── Organisations ─────────────────────────────────────────────────────────

// Organisations godoc: GET /admin/organisations?search=&status=&page=&page_size=
func (h *Handler) Organisations(c *gin.Context) {
	rows, total, err := h.svc.repo.Organisations(h.opts(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// Organisation godoc: GET /admin/organisations/{id}
func (h *Handler) Organisation(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	row, err := h.svc.repo.OrganisationByID(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if row.ID == "" {
		response.NotFound(c, "Organisation not found")
		return
	}
	response.OK(c, row)
}

// OrganisationStores godoc: GET /admin/organisations/{id}/stores
func (h *Handler) OrganisationStores(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	o := h.opts(c)
	rows, total, err := h.svc.repo.OrgStores(id, o)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// OrganisationUsers godoc: GET /admin/organisations/{id}/users
func (h *Handler) OrganisationUsers(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	rows, err := h.svc.repo.OrgUsersAll(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, rows)
}

// UpdateOrganisation godoc: PATCH /admin/organisations/{id}
func (h *Handler) UpdateOrganisation(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	var p OrgPatch
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	fields, err := h.svc.OrgPatchFields(p)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	matched, err := h.svc.repo.Update("organisations", id, fields)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Organisation not found")
		return
	}
	row, err := h.svc.repo.OrganisationByID(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, row)
}

// ── Users ─────────────────────────────────────────────────────────────────

// Users godoc: GET /admin/users?search=&org_id=&store_id=&role=&status=&page=&page_size=
func (h *Handler) Users(c *gin.Context) {
	rows, total, err := h.svc.repo.Users(h.opts(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// UpdateUser godoc: PATCH /admin/users/{id}
func (h *Handler) UpdateUser(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	var p UserPatch
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	fields, err := h.svc.UserPatchFields(p)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	matched, err := h.svc.repo.Update("users", id, fields)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "User not found")
		return
	}
	response.OK(c, gin.H{"id": id, "updated": true})
}

// ── Stores ────────────────────────────────────────────────────────────────

// Stores godoc: GET /admin/stores?search=&org_id=&status=&page=&page_size=
func (h *Handler) Stores(c *gin.Context) {
	rows, total, err := h.svc.repo.Stores(h.opts(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// UpdateStore godoc: PATCH /admin/stores/{id}
func (h *Handler) UpdateStore(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	var p StorePatch
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	fields, err := h.svc.StorePatchFields(p)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	matched, err := h.svc.repo.Update("stores", id, fields)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Store not found")
		return
	}
	response.OK(c, gin.H{"id": id, "updated": true})
}

// ── Products ──────────────────────────────────────────────────────────────

// Products godoc: GET /admin/products?search=&org_id=&store_id=&category=&status=&low_stock=&page=&page_size=
func (h *Handler) Products(c *gin.Context) {
	o := h.opts(c)
	category := c.Query("category")
	lowStock := c.Query("low_stock") == "true"
	rows, total, err := h.svc.repo.Products(o, category, lowStock)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// Product godoc: GET /admin/products/{id}
func (h *Handler) Product(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	row, err := h.svc.repo.ProductByID(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if row.ID == "" {
		response.NotFound(c, "Product not found")
		return
	}
	response.OK(c, row)
}

// UpdateProduct godoc: PATCH /admin/products/{id}
func (h *Handler) UpdateProduct(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	var p ProductPatch
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	fields, err := h.svc.ProductPatchFields(p)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	matched, err := h.svc.repo.Update("products", id, fields)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Product not found")
		return
	}
	response.OK(c, gin.H{"id": id, "updated": true})
}

// Categories godoc: GET /admin/categories
func (h *Handler) Categories(c *gin.Context) {
	rows, err := h.svc.repo.Categories()
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, rows)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	var input CategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	orgID, fields, err := h.svc.CategoryFields(input, true)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	id, err := h.svc.repo.CreateCategoryRecord(orgID, fields["name"].(string), fields["description"].(string), fields["sort_order"].(int), fields["is_active"].(bool))
	if err != nil {
		writeErr(c, err)
		return
	}
	row, err := h.svc.repo.CategoryByID(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, row, "Category created")
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	var input CategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	_, fields, err := h.svc.CategoryFields(input, false)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	matched, err := h.svc.repo.Update("categories", id, fields)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Category not found")
		return
	}
	row, err := h.svc.repo.CategoryByID(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, row)
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	count, err := h.svc.repo.CategoryProductCount(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if count > 0 {
		response.Conflict(c, "Category is assigned to products. Deactivate it instead.")
		return
	}
	matched, err := h.svc.repo.Delete("categories", id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Category not found")
		return
	}
	response.Message(c, 200, "Category deleted")
}

// ── Transactions ──────────────────────────────────────────────────────────

// Transactions godoc: GET /admin/transactions?search=&status=&method=&org_id=&store_id=&customer=&from=&to=&page=&page_size=
func (h *Handler) Transactions(c *gin.Context) {
	o := h.opts(c)
	f := TxnFilter{
		Options:  o,
		Method:   c.Query("method"),
		Customer: c.Query("customer"),
		DateFrom: c.Query("from"),
		DateTo:   c.Query("to"),
	}
	rows, total, err := h.svc.repo.Transactions(f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// Transaction godoc: GET /admin/transactions/{id}
func (h *Handler) Transaction(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	row, err := h.svc.repo.TransactionByID(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if row.ID == "" {
		response.NotFound(c, "Transaction not found")
		return
	}
	lines, err := h.svc.repo.TransactionLines(id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"transaction": row, "lines": lines})
}

// TransactionSummary godoc: GET /admin/transactions/summary?status=&method=&org_id=&store_id=&from=&to=
func (h *Handler) TransactionSummary(c *gin.Context) {
	o := h.opts(c)
	f := TxnFilter{
		Options:  o,
		Method:   c.Query("method"),
		DateFrom: c.Query("from"),
		DateTo:   c.Query("to"),
	}
	totals, err := h.svc.repo.TransactionTotals(f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, totals)
}

// ── Customers ─────────────────────────────────────────────────────────────

// Customers godoc: GET /admin/customers?search=&segment=&org_id=&store_id=&has_credit=&page=&page_size=
func (h *Handler) Customers(c *gin.Context) {
	o := h.opts(c)
	segment := c.Query("segment")
	hasCredit := c.Query("has_credit") == "true"
	rows, total, err := h.svc.repo.Customers(o, segment, hasCredit)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// ── Credits ───────────────────────────────────────────────────────────────

// Credits godoc: GET /admin/credits?status=&customer=&org_id=&store_id=&overdue=&page=&page_size=
func (h *Handler) Credits(c *gin.Context) {
	o := h.opts(c)
	customer := c.Query("customer")
	overdue := c.Query("overdue") == "true"
	rows, total, err := h.svc.repo.Credits(o, customer, overdue)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// ── Suppliers ─────────────────────────────────────────────────────────────

// Suppliers godoc: GET /admin/suppliers?search=&status=&org_id=&page=&page_size=
func (h *Handler) Suppliers(c *gin.Context) {
	rows, total, err := h.svc.repo.Suppliers(h.opts(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// SupplierDeliveries godoc: GET /admin/suppliers/{id}/deliveries?org_id=&unpaid=&page=&page_size=
func (h *Handler) SupplierDeliveries(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	o := h.opts(c)
	unpaid := c.Query("unpaid") == "true"
	rows, total, err := h.svc.repo.Deliveries(o, id.String(), unpaid)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// SupplierPayments godoc: GET /admin/suppliers/{id}/payments?org_id=&page=&page_size=
func (h *Handler) SupplierPayments(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	o := h.opts(c)
	rows, total, err := h.svc.repo.Payments(o, id.String())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// ── Expenses ──────────────────────────────────────────────────────────────

// Expenses godoc: GET /admin/expenses?search=&category=&org_id=&store_id=&from=&to=&page=&page_size=
func (h *Handler) Expenses(c *gin.Context) {
	o := h.opts(c)
	category := c.Query("category")
	rows, total, err := h.svc.repo.Expenses(o, category, c.Query("from"), c.Query("to"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// ── Reviews ───────────────────────────────────────────────────────────────

// Reviews godoc: GET /admin/reviews?status=&rating=&org_id=&page=&page_size=
func (h *Handler) Reviews(c *gin.Context) {
	o := h.opts(c)
	rating := atoi(c.DefaultQuery("rating", "0"))
	rows, total, err := h.svc.repo.Reviews(o.Status, rating, o)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// UpdateReview godoc: PATCH /admin/reviews/{id}
func (h *Handler) UpdateReview(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	var p ReviewPatch
	if err := c.ShouldBindJSON(&p); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	fields, err := h.svc.ReviewPatchFields(p)
	if err != nil {
		response.BadRequest(c, err.Error(), nil)
		return
	}
	matched, err := h.svc.repo.Update("store_reviews", id, fields)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Review not found")
		return
	}
	response.OK(c, gin.H{"id": id, "updated": true})
}

// DeleteReview godoc: DELETE /admin/reviews/{id}
func (h *Handler) DeleteReview(c *gin.Context) {
	id, ok := h.id(c)
	if !ok {
		return
	}
	matched, err := h.svc.repo.Delete("store_reviews", id)
	if err != nil {
		writeErr(c, err)
		return
	}
	if !matched {
		response.NotFound(c, "Review not found")
		return
	}
	response.Message(c, 200, "Review deleted")
}

// ── Notes & notifications ─────────────────────────────────────────────────

// Notes godoc: GET /admin/notes?search=&org_id=&page=&page_size=
func (h *Handler) Notes(c *gin.Context) {
	rows, total, err := h.svc.repo.Notes(h.opts(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// Notifications godoc: GET /admin/notifications?category=&org_id=&page=&page_size=
func (h *Handler) Notifications(c *gin.Context) {
	o := h.opts(c)
	category := c.Query("category")
	rows, total, err := h.svc.repo.Notifications(o, category)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.List(c, rows, total, atoi(c.DefaultQuery("page", "1")), atoi(c.DefaultQuery("page_size", "20")))
}

// ── AI usage ──────────────────────────────────────────────────────────────

// AIUsage godoc: GET /admin/ai/usage?month=&org_id=
func (h *Handler) AIUsage(c *gin.Context) {
	rows, err := h.svc.repo.AIUsage(atoi(c.DefaultQuery("month", "0")), c.Query("org_id"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, rows)
}

// ── Subscriptions ─────────────────────────────────────────────────────────

// Subscriptions godoc: GET /admin/subscriptions
func (h *Handler) Subscriptions(c *gin.Context) {
	summary, err := h.svc.repo.SubscriptionsRevenue()
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, summary)
}

// ── Analytics ─────────────────────────────────────────────────────────────

// AnalyticsOverview godoc: GET /admin/analytics/overview
func (h *Handler) AnalyticsOverview(c *gin.Context) {
	money, err := h.svc.repo.PlatformMoney()
	if err != nil {
		writeErr(c, err)
		return
	}
	series, err := h.svc.repo.RevenueSeries(30)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"money": money, "revenue_series": series})
}

// MonitoringOverview godoc: GET /admin/monitoring/overview
func (h *Handler) MonitoringOverview(c *gin.Context) {
	out, err := h.svc.repo.MonitoringOverview()
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"overview": out, "generated_at": time.Now()})
}

// MonitoringActivity godoc: GET /admin/monitoring/activity?limit=50
func (h *Handler) MonitoringActivity(c *gin.Context) {
	rows, err := h.svc.repo.MonitoringActivity(atoi(c.DefaultQuery("limit", "50")))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"activity": rows, "count": len(rows)})
}

// MonitoringHealth godoc: GET /admin/monitoring/health
func (h *Handler) MonitoringHealth(c *gin.Context) {
	status := "ok"
	sqlDB, err := h.svc.repo.db.DB()
	dbStatus := "ok"
	if err != nil || sqlDB.Ping() != nil {
		status = "degraded"
		dbStatus = "error"
	}
	response.OK(c, gin.H{"status": status, "database": dbStatus, "api": "ok", "generated_at": time.Now()})
}
