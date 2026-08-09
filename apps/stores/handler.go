package stores

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes store HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a stores Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) storeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid store id", nil)
		return uuid.Nil, false
	}
	return id, true
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

// List godoc: GET /stores
func (h *Handler) List(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	stores, err := h.svc.List(c.Request.Context(), orgID, c.Query("status"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"items": stores, "count": len(stores)})
}

// Get godoc: GET /stores/{id}
func (h *Handler) Get(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	id, ok2 := h.storeID(c)
	if !ok2 {
		return
	}
	store, err := h.svc.Get(c.Request.Context(), orgID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, store)
}

// Create godoc: POST /stores
func (h *Handler) Create(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	var req StoreInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	store, err := h.svc.Create(c.Request.Context(), orgID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, store, "Store created")
}

// Update godoc: PATCH /stores/{id}
func (h *Handler) Update(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	id, ok2 := h.storeID(c)
	if !ok2 {
		return
	}
	var req StoreInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	store, err := h.svc.Update(c.Request.Context(), orgID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, store)
}

// Delete godoc: DELETE /stores/{id}
func (h *Handler) Delete(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	id, ok2 := h.storeID(c)
	if !ok2 {
		return
	}
	if err := h.svc.Deactivate(c.Request.Context(), orgID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Stats godoc: GET /stores/stats
func (h *Handler) Stats(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	stats, err := h.svc.OrgStats(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, stats)
}

// Staff godoc: GET /stores/{id}/staff
func (h *Handler) Staff(c *gin.Context) {
	id, ok := h.storeID(c)
	if !ok {
		return
	}
	roster, err := h.svc.StaffRoster(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"items": roster, "count": len(roster)})
}

// Week godoc: GET /stores/{id}/week
func (h *Handler) Week(c *gin.Context) {
	id, ok := h.storeID(c)
	if !ok {
		return
	}
	week, err := h.svc.WeekBreakdown(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, week)
}

// SetStatus godoc: PATCH /stores/{id}/status
func (h *Handler) SetStatus(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	id, ok2 := h.storeID(c)
	if !ok2 {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.SetStatus(c.Request.Context(), orgID, id, req.Status)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}
