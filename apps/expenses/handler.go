package expenses

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes expense HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds an expenses Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) orgStore(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, uuid.Nil, false
	}
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, storeID, true
}

func (h *Handler) userID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(middleware.UserID(c))
	return id
}

// List godoc: GET /expenses?search=&category=&payment_method=&ordering=&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var f Filter
	f.Search = c.Query("search")
	f.Category = c.Query("category")
	f.PaymentMethod = c.Query("payment_method")
	f.Ordering = c.Query("ordering")
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.List(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Create godoc: POST /expenses
func (h *Handler) Create(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var req Input
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	e, err := h.svc.Create(c.Request.Context(), orgID, storeID, h.userID(c), &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, expenseResponse(e), "Expense recorded")
}

// Get godoc: GET /expenses/{id}
func (h *Handler) Get(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid expense id", nil)
		return
	}
	e, err := h.svc.Get(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, expenseResponse(e))
}

// Update godoc: PATCH /expenses/{id}
func (h *Handler) Update(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid expense id", nil)
		return
	}
	var req Input
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	e, err := h.svc.Update(c.Request.Context(), storeID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, expenseResponse(e))
}

// Delete godoc: DELETE /expenses/{id}
func (h *Handler) Delete(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid expense id", nil)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), storeID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Summary godoc: GET /expenses/summary
func (h *Handler) Summary(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	out, err := h.svc.Summary(c.Request.Context(), storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
