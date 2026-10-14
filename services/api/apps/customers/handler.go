package customers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes customer HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a customers Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Create godoc: POST /customers
func (h *Handler) Create(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	var req Input
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	cust, err := h.svc.Create(c.Request.Context(), orgID, storeID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, customerResponse(cust), "Customer created")
}

// Get godoc: GET /customers/{id}
func (h *Handler) Get(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	cust, err := h.svc.Get(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, customerResponse(cust))
}

// Update godoc: PATCH /customers/{id}
func (h *Handler) Update(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	var req Input
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	cust, err := h.svc.Update(c.Request.Context(), storeID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, customerResponse(cust))
}

// List godoc: GET /customers
func (h *Handler) List(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	var f Filter
	f.Search = c.Query("search")
	f.Segment = c.Query("segment")
	if v := c.Query("has_credit"); v != "" {
		b, _ := strconv.ParseBool(v)
		f.HasCredit = &b
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.List(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// CreditCustomers godoc: GET /customers/credits?overdue=
func (h *Handler) CreditCustomers(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	overdue, _ := strconv.ParseBool(c.Query("overdue"))
	out, err := h.svc.CreditCustomers(c.Request.Context(), storeID, overdue)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// SendSms godoc: POST /customers/{id}/sms — send an SMS to a customer.
func (h *Handler) SendSms(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	var req struct {
		Message   string `json:"message"`
		Confirmed bool   `json:"confirmed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	if !req.Confirmed {
		response.BadRequest(c, "Confirm the recipient, message, and provider charges before sending", nil)
		return
	}
	out, err := h.svc.SendSMS(c.Request.Context(), orgID, storeID, id, req.Message)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "SendAfrica accepted the SMS request; handset delivery is not confirmed")
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
