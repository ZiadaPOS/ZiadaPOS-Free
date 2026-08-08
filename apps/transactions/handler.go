package transactions

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes POS transaction HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a transactions Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func parseID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.BadRequest(c, "Invalid "+param, nil)
		return uuid.Nil, false
	}
	return id, true
}

// CompleteSale godoc: POST /pos/sale
func (h *Handler) CompleteSale(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	cashierID, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	var req SaleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	txn, err := h.svc.CompleteSale(c.Request.Context(), orgID, storeID, cashierID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, txnResponse(txn), "Sale completed")
}

// ListTransactions godoc: GET /pos/transactions
func (h *Handler) ListTransactions(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	var f Filter
	f.Status = c.Query("status")
	f.Method = c.Query("method")
	f.Search = c.Query("search")
	f.CustomerID, _ = uuid.Parse(c.Query("customer_id"))
	if df, err := time.Parse("2006-01-02", c.Query("date_from")); err == nil {
		f.DateFrom = df
	}
	if df, err := time.Parse("2006-01-02", c.Query("date_to")); err == nil {
		f.DateTo = df.Add(24 * time.Hour)
	}
	f.Page = atoi(c.DefaultQuery("page", "1"))
	f.PageSize = atoi(c.DefaultQuery("page_size", "20"))

	out, err := h.svc.ListTransactions(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// GetTransaction godoc: GET /pos/transactions/{id}
func (h *Handler) GetTransaction(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	txn, err := h.svc.GetTransaction(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, txnResponse(txn))
}

// Refund godoc: POST /pos/transactions/{id}/refund
func (h *Handler) Refund(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	userID, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	txn, err := h.svc.Refund(c.Request.Context(), orgID, storeID, userID, id, req.Reason)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, txnResponse(txn))
}

// Void godoc: POST /pos/transactions/{id}/void
func (h *Handler) Void(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	txn, err := h.svc.Void(c.Request.Context(), orgID, storeID, id, req.Reason)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, txnResponse(txn))
}

// Summary godoc: GET /pos/summary?date_from=&date_to=
func (h *Handler) Summary(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	from := parseDate(c.DefaultQuery("date_from", time.Now().Format("2006-01-02")))
	to := parseDate(c.DefaultQuery("date_to", time.Now().Format("2006-01-02"))).Add(24 * time.Hour)
	out, err := h.svc.Summary(c.Request.Context(), storeID, from, to)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// HourlySales godoc: GET /pos/summary/hourly?date=YYYY-MM-DD
func (h *Handler) HourlySales(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	day := parseDate(c.DefaultQuery("date", time.Now().Format("2006-01-02")))
	rows, err := h.svc.HourlySales(c.Request.Context(), storeID, day)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, rows)
}

func parseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Now()
	}
	return t
}

func atoi(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
