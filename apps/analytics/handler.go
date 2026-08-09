package analytics

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

// Handler exposes analytics HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds an analytics Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) storeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) rangeOf(c *gin.Context) Range {
	return ParseRange(c.Query)
}

func rangeMeta(rng Range) map[string]interface{} {
	return map[string]interface{}{
		"date_from": rng.FromDate.Format(dayLayout),
		"date_to":   rng.ToDate.Format(dayLayout),
	}
}

// Overview godoc: GET /analytics/overview
func (h *Handler) Overview(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	data := rangeMeta(rng)

	sections := splitSections(c.Query("sections"))
	if sections["kpis"] {
		k, err := h.svc.KPISummary(c.Request.Context(), storeID, rng)
		if err != nil {
			writeErr(c, err)
			return
		}
		data["kpis"] = k
	}
	if sections["trend"] {
		t, err := h.svc.RevenueTrend(c.Request.Context(), storeID, rng)
		if err != nil {
			writeErr(c, err)
			return
		}
		data["trend"] = t
	}
	if sections["mix"] {
		m, err := h.svc.PaymentMix(c.Request.Context(), storeID, rng)
		if err != nil {
			writeErr(c, err)
			return
		}
		data["payment_mix"] = m
	}
	if sections["products"] {
		p, err := h.svc.TopProducts(c.Request.Context(), storeID, rng, 5)
		if err != nil {
			writeErr(c, err)
			return
		}
		data["top_products"] = p
	}
	response.OK(c, data)
}

// Summary godoc: GET /analytics/summary
func (h *Handler) Summary(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	k, err := h.svc.KPISummary(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	data := rangeMeta(rng)
	for key, val := range k {
		data[key] = val
	}
	response.OK(c, data)
}

// Trend godoc: GET /analytics/trend
func (h *Handler) Trend(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	t, err := h.svc.RevenueTrend(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{
		"trend":     t,
		"date_from": rng.FromDate.Format(dayLayout),
		"date_to":   rng.ToDate.Format(dayLayout),
	})
}

// PaymentMix godoc: GET /analytics/payment-mix
func (h *Handler) PaymentMix(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	m, err := h.svc.PaymentMix(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"payment_mix": m})
}

// TopProducts godoc: GET /analytics/top-products
func (h *Handler) TopProducts(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	limit := atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	p, err := h.svc.TopProducts(c.Request.Context(), storeID, rng, limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{
		"top_products": p,
		"date_from":    rng.FromDate.Format(dayLayout),
		"date_to":      rng.ToDate.Format(dayLayout),
	})
}

// Sales godoc: GET /analytics/sales
func (h *Handler) Sales(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	k, err := h.svc.KPISummary(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	t, err := h.svc.RevenueTrend(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	b, err := h.svc.SalesBreakdown(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	data := rangeMeta(rng)
	data["kpis"] = k
	data["trend"] = t
	for key, val := range b {
		data[key] = val
	}
	response.OK(c, data)
}

// Products godoc: GET /analytics/products
func (h *Handler) Products(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	p, err := h.svc.ProductPerformance(c.Request.Context(), storeID, rng, c.Query("category"), c.Query("sort"))
	if err != nil {
		writeErr(c, err)
		return
	}
	data := rangeMeta(rng)
	for key, val := range p {
		data[key] = val
	}
	response.OK(c, data)
}

// Customers godoc: GET /analytics/customers
func (h *Handler) Customers(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	ca, err := h.svc.CustomerAnalytics(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	data := rangeMeta(rng)
	for key, val := range ca {
		data[key] = val
	}
	response.OK(c, data)
}

// Cashflow godoc: GET /analytics/cashflow
func (h *Handler) Cashflow(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	rng := h.rangeOf(c)
	cf, err := h.svc.Cashflow(c.Request.Context(), storeID, rng)
	if err != nil {
		writeErr(c, err)
		return
	}
	data := rangeMeta(rng)
	for key, val := range cf {
		data[key] = val
	}
	response.OK(c, data)
}

// Dashboard godoc: GET /analytics/dashboard
func (h *Handler) Dashboard(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	d, err := h.svc.Dashboard(c.Request.Context(), storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, d)
}

func splitSections(s string) map[string]bool {
	out := map[string]bool{}
	if s == "" {
		return map[string]bool{"kpis": true, "trend": true, "mix": true, "products": true}
	}
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out[p] = true
		}
	}
	return out
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
