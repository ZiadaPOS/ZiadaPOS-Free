package suppliers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes supplier HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a suppliers Handler.
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

// Create godoc: POST /suppliers
func (h *Handler) Create(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var req SupplierInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	s, err := h.svc.Create(c.Request.Context(), orgID, storeID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, supplierResponse(s, 0), "Supplier created")
}

// List godoc: GET /suppliers?search=&status=&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var f SupplierFilter
	f.Search = c.Query("search")
	f.Status = c.Query("status")
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.List(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Stats godoc: GET /suppliers/stats
func (h *Handler) Stats(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	out, err := h.svc.Stats(c.Request.Context(), storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Get godoc: GET /suppliers/{id}
func (h *Handler) Get(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid supplier id", nil)
		return
	}
	s, err := h.svc.Get(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	balance, err := h.svc.outstandingBalance(c.Request.Context(), storeID, id)
	if err != nil {
		balance = 0
	}
	response.OK(c, supplierResponse(s, balance))
}

// Update godoc: PATCH /suppliers/{id}
func (h *Handler) Update(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid supplier id", nil)
		return
	}
	var req SupplierInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	s, err := h.svc.Update(c.Request.Context(), storeID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, supplierResponse(s, 0))
}

// Delete godoc: DELETE /suppliers/{id}
func (h *Handler) Delete(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid supplier id", nil)
		return
	}
	s, err := h.svc.Deactivate(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, supplierResponse(s, 0))
}

// Deliveries godoc: GET|POST /suppliers/{id}/deliveries
func (h *Handler) Deliveries(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid supplier id", nil)
		return
	}
	if c.Request.Method == "POST" {
		var req DeliveryInput
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid request", nil)
			return
		}
		d, err := h.svc.RecordDelivery(c.Request.Context(), orgID, storeID, id, h.userID(c), &req)
		if err != nil {
			writeErr(c, err)
			return
		}
		response.Created(c, deliveryResponse(d, ""), "Delivery recorded")
		return
	}
	var f DeliveryFilter
	f.SupplierID = id
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListDeliveries(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// SupplierPayments godoc: GET /suppliers/{id}/payments
func (h *Handler) SupplierPayments(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid supplier id", nil)
		return
	}
	var f PaymentFilter
	f.SupplierID = id
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListPayments(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Pay godoc: POST /suppliers/{id}/pay
func (h *Handler) Pay(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid supplier id", nil)
		return
	}
	var req PaymentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	p, err := h.svc.RecordPayment(c.Request.Context(), orgID, storeID, id, h.userID(c), &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, paymentResponse(p, ""), "Payment recorded")
}

// AllDeliveries godoc: GET /suppliers/deliveries?unpaid=&supplier=&page=
func (h *Handler) AllDeliveries(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var f DeliveryFilter
	f.Unpaid = c.Query("unpaid") == "true"
	if v := c.Query("supplier"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.SupplierID = id
		}
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListDeliveries(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// CreateDeliveryGlobal godoc: POST /suppliers/deliveries
// Body: { supplier_id, invoice_number, delivery_date, total_value, ... }
func (h *Handler) CreateDeliveryGlobal(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var req struct {
		SupplierID string `json:"supplier_id" binding:"required"`
		DeliveryInput
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	supplierID, err := uuid.Parse(req.SupplierID)
	if err != nil {
		response.BadRequest(c, "Invalid supplier_id", nil)
		return
	}
	d, err := h.svc.RecordDelivery(c.Request.Context(), orgID, storeID, supplierID, h.userID(c), &req.DeliveryInput)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, deliveryResponse(d, ""), "Delivery recorded")
}

// GetDelivery godoc: GET /suppliers/deliveries/{id}
func (h *Handler) GetDelivery(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid delivery id", nil)
		return
	}
	d, err := h.svc.GetDelivery(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, deliveryResponse(d, ""))
}

// UpdateDelivery godoc: PATCH /suppliers/deliveries/{id}
func (h *Handler) UpdateDelivery(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid delivery id", nil)
		return
	}
	var req DeliveryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	d, err := h.svc.UpdateDelivery(c.Request.Context(), storeID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, deliveryResponse(d, ""))
}

// DeleteDelivery godoc: DELETE /suppliers/deliveries/{id}
func (h *Handler) DeleteDelivery(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid delivery id", nil)
		return
	}
	if err := h.svc.DeleteDelivery(c.Request.Context(), storeID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// DeliveryPay godoc: POST /suppliers/deliveries/{id}/pay
func (h *Handler) DeliveryPay(c *gin.Context) {
	orgID, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid delivery id", nil)
		return
	}
	d, err := h.svc.GetDelivery(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	var req PaymentInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	req.DeliveryID = d.ID.String()
	p, err := h.svc.RecordPayment(c.Request.Context(), orgID, storeID, d.SupplierID, h.userID(c), &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, paymentResponse(p, ""), "Payment recorded")
}

// AllPayments godoc: GET /suppliers/payments?supplier=&page=
func (h *Handler) AllPayments(c *gin.Context) {
	_, storeID, ok := h.orgStore(c)
	if !ok {
		return
	}
	var f PaymentFilter
	if v := c.Query("supplier"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.SupplierID = id
		}
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListPayments(c.Request.Context(), storeID, f)
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
