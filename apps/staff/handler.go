package staff

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes staff endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates the staff HTTP handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func parseID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		response.BadRequest(c, "Invalid staff id", nil)
		return uuid.Nil, false
	}
	return id, true
}

func orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.Unauthorized(c, "Organisation context missing")
		return uuid.Nil, false
	}
	return id, true
}

func storeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return uuid.Nil, false
	}
	return id, true
}

// List godoc: GET /staff
func (h *Handler) List(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	opts := ListOptions{
		Search:   strings.TrimSpace(c.Query("search")),
		Role:     c.Query("role"),
		Status:   c.Query("status"),
		Shift:    c.Query("shift"),
		Ordering: c.Query("ordering"),
	}
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		opts.Page = p
	}
	if s, err := strconv.Atoi(c.DefaultQuery("size", "20")); err == nil {
		opts.Size = s
	}
	withStats := c.Query("include_stats") == "true" || c.Query("include_stats") == "1"
	out, total, err := h.svc.List(c.Request.Context(), store, opts, withStats)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"items": out, "total": total})
}

// Create godoc: POST /staff
func (h *Handler) Create(c *gin.Context) {
	oid, ok := orgID(c)
	if !ok {
		return
	}
	store, ok := storeID(c)
	if !ok {
		return
	}
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid payload", nil)
		return
	}
	out, err := h.svc.Create(c.Request.Context(), oid, store, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Staff created")
}

// Get godoc: GET /staff/:id
func (h *Handler) Get(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), store, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Update godoc: PATCH /staff/:id
func (h *Handler) Update(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid payload", nil)
		return
	}
	out, err := h.svc.Update(c.Request.Context(), store, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Deactivate godoc: DELETE /staff/:id
func (h *Handler) Deactivate(c *gin.Context) {
	oid, ok := orgID(c)
	if !ok {
		return
	}
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	actorID, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "User context missing")
		return
	}
	if err := h.svc.Deactivate(c.Request.Context(), oid, store, id, actorID); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"deactivated": true})
}

// Stats godoc: GET /staff/:id/stats
func (h *Handler) Stats(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	out, err := h.svc.Stats(c.Request.Context(), store, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// KPIs godoc: GET /staff/kpis
func (h *Handler) KPIs(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	out, err := h.svc.KPIs(c.Request.Context(), store)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// UpdateShift godoc: PATCH /staff/:id/shift
func (h *Handler) UpdateShift(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req ShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid payload", nil)
		return
	}
	out, err := h.svc.UpdateShift(c.Request.Context(), store, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// UpdatePermissions godoc: PATCH /staff/:id/permissions
func (h *Handler) UpdatePermissions(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req PermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid payload", nil)
		return
	}
	out, err := h.svc.UpdatePermissions(c.Request.Context(), store, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Activity godoc: GET /staff/:id/activity
func (h *Handler) Activity(c *gin.Context) {
	store, ok := storeID(c)
	if !ok {
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	day, err := time.Parse("2006-01-02", c.DefaultQuery("date", time.Now().Format("2006-01-02")))
	if err != nil {
		response.BadRequest(c, "Invalid date", nil)
		return
	}
	out, err := h.svc.Activity(c.Request.Context(), store, id, day)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{"date": day.Format("2006-01-02"), "items": out})
}

func writeErr(c *gin.Context, err error) {
	e, ok := err.(*apierr.Error)
	if ok {
		response.Error(c, e.Status, e.Message, e.Fields)
		return
	}
	response.Internal(c, "Internal server error")
}
