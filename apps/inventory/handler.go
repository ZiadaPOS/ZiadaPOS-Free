package inventory

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes inventory HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds an inventory Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ── Categories ──────────────────────────────────────────────────────────────

// CreateCategory godoc: POST /inventory/categories
func (h *Handler) CreateCategory(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	cat, err := h.svc.CreateCategory(c.Request.Context(), orgID, req.Name, req.Description)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, cat, "Category created")
}

// ListCategories godoc: GET /inventory/categories
func (h *Handler) ListCategories(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	cats, err := h.svc.ListCategories(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, cats)
}

// UpdateCategory godoc: PATCH /inventory/categories/{id}
func (h *Handler) UpdateCategory(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category id", nil)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
		IsActive    *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	cat, err := h.svc.UpdateCategory(c.Request.Context(), orgID, id, req.Name, req.Description, req.SortOrder, req.IsActive)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, cat)
}

// DeleteCategory godoc: DELETE /inventory/categories/{id}
func (h *Handler) DeleteCategory(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid category id", nil)
		return
	}
	if err := h.svc.DeleteCategory(c.Request.Context(), orgID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.NoContent(c)
}

// ── Products ────────────────────────────────────────────────────────────────

// CreateProduct godoc: POST /inventory/products
func (h *Handler) CreateProduct(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	var req ProductInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	req.CreatedBy, _ = uuid.Parse(middleware.UserID(c))
	p, err := h.svc.CreateProduct(c.Request.Context(), storeID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, productResponse(p), "Product created")
}

// ListProducts godoc: GET /inventory/products
func (h *Handler) ListProducts(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	f := ProductFilter{StoreID: storeID, Search: c.Query("search")}
	f.CategoryID, _ = uuid.Parse(c.Query("category_id"))
	f.Status = c.Query("status")
	f.LowStock, _ = strconv.ParseBool(c.Query("low_stock"))
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListProducts(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// GetProduct godoc: GET /inventory/products/{id}
func (h *Handler) GetProduct(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product id", nil)
		return
	}
	p, err := h.svc.GetProduct(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, productResponse(p))
}

// UpdateProduct godoc: PATCH /inventory/products/{id}
func (h *Handler) UpdateProduct(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product id", nil)
		return
	}
	var req UpdateProductInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	p, err := h.svc.UpdateProduct(c.Request.Context(), storeID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, productResponse(p))
}

// DeleteProduct godoc: DELETE /inventory/products/{id}
func (h *Handler) DeleteProduct(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product id", nil)
		return
	}
	if err := h.svc.DeleteProduct(c.Request.Context(), storeID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.NoContent(c)
}

// ByBarcode godoc: GET /inventory/products/barcode/{code}
func (h *Handler) ByBarcode(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	p, err := h.svc.ByBarcode(c.Request.Context(), storeID, c.Param("code"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, productResponse(p))
}

// LowStock godoc: GET /inventory/low-stock
func (h *Handler) LowStock(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	items, err := h.svc.LowStock(c.Request.Context(), storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, items)
}

// AdjustStock godoc: POST /inventory/products/{id}/adjust
func (h *Handler) AdjustStock(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product id", nil)
		return
	}
	userID, _ := uuid.Parse(middleware.UserID(c))
	var req struct {
		QtyChange int    `json:"qty_change" binding:"required"`
		Reason    string `json:"reason" binding:"required"`
		Note      string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	p, err := h.svc.AdjustStock(c.Request.Context(), storeID, productID, userID, req.QtyChange, req.Reason, req.Note)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, productResponse(p))
}

// Adjustments godoc: GET /inventory/products/{id}/adjustments
func (h *Handler) Adjustments(c *gin.Context) {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid product id", nil)
		return
	}
	limit := 50
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 && l <= 200 {
		limit = l
	}
	adj, err := h.svc.repo.ListAdjustments(productID, limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, adj)
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
