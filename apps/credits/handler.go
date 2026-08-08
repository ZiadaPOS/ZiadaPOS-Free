package credits

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes credit HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a credits Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type paymentRequest struct {
	Amount    int64  `json:"amount" binding:"required"`
	Method    string `json:"method"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

type messageRequest struct {
	Kind string `json:"kind"`
	Body string `json:"body"`
}

type noteRequest struct {
	Body string `json:"body" binding:"required"`
}

type writeOffRequest struct {
	Reason string `json:"reason"`
}

// List godoc: GET /credits?status=&customer_id=&overdue=&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	var f TabFilter
	f.Status = c.Query("status")
	if v := c.Query("customer_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.CustomerID = id
		}
	}
	f.Overdue, _ = strconv.ParseBool(c.Query("overdue"))
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.ListTabs(c.Request.Context(), storeID, f)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Profile godoc: GET /credits/customers/{customer_id}
func (h *Handler) Profile(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	out, err := h.svc.CustomerProfile(c.Request.Context(), storeID, customerID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Pay godoc: POST /credits/customers/{customer_id}/payments
func (h *Handler) Pay(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	cashierID, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}
	var req paymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	pay, err := h.svc.RecordPayment(c.Request.Context(), storeID, customerID, cashierID, req.Amount, req.Method, req.Reference, req.Note)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, pay, "Payment recorded")
}

// WriteOff godoc: POST /credits/{tab_id}/write-off
func (h *Handler) WriteOff(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	tabID, err := uuid.Parse(c.Param("tab_id"))
	if err != nil {
		response.BadRequest(c, "Invalid tab id", nil)
		return
	}
	var req writeOffRequest
	_ = c.ShouldBindJSON(&req)
	tab, err := h.svc.WriteOff(c.Request.Context(), storeID, tabID, req.Reason)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, tabResponse(tab))
}

// SendReminder godoc: POST /credits/customers/{customer_id}/remind
func (h *Handler) SendReminder(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	msg, err := h.svc.SendReminder(c.Request.Context(), storeID, customerID, middleware.UserID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, msg, "Reminder sent")
}

// AddMessage godoc: POST /credits/customers/{customer_id}/messages
func (h *Handler) AddMessage(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	var req messageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	msg, err := h.svc.AddMessage(c.Request.Context(), storeID, customerID, req.Kind, req.Body, middleware.UserID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, msg, "Message logged")
}

// AddNote godoc: POST /credits/customers/{customer_id}/notes
func (h *Handler) AddNote(c *gin.Context) {
	storeID, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return
	}
	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		response.BadRequest(c, "Invalid customer id", nil)
		return
	}
	by, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}
	var req noteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	note, err := h.svc.AddNote(c.Request.Context(), storeID, customerID, by, req.Body)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, note, "Note added")
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
