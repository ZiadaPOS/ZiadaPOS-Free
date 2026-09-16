package notifications

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes in-app notification HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a notifications Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) org(c *gin.Context) (uuid.UUID, bool) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, false
	}
	return orgID, true
}

func (h *Handler) storeID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(middleware.StoreID(c))
	return id
}

// List godoc: GET /notifications?category=&unread=true&page=&page_size=
func (h *Handler) List(c *gin.Context) {
	orgID, ok := h.org(c)
	if !ok {
		return
	}
	var f Filter
	f.Category = c.Query("category")
	if ur, err := strconv.ParseBool(c.Query("unread")); err == nil {
		f.Unread = ur
	}
	f.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	f.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "30"))

	items, _, unread, err := h.svc.repo.List(c.Request.Context(), orgID, h.storeID(c), f)
	if err != nil {
		writeErr(c, err)
		return
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, n := range items {
		out = append(out, notificationResponse(n))
	}
	response.OK(c, map[string]interface{}{
		"items":         out,
		"notifications": out,
		"unread_count":  unread,
	})
}

// UnreadCount godoc: GET /notifications/unread-count
func (h *Handler) UnreadCount(c *gin.Context) {
	orgID, ok := h.org(c)
	if !ok {
		return
	}
	count, err := h.svc.repo.UnreadCount(c.Request.Context(), orgID, h.storeID(c))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"count": count})
}

// MarkRead godoc: PATCH /notifications/{id}/read
func (h *Handler) MarkRead(c *gin.Context) {
	orgID, ok := h.org(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification id", nil)
		return
	}
	if err := h.svc.repo.MarkRead(c.Request.Context(), orgID, h.storeID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// MarkAllRead godoc: PATCH /notifications/read-all
func (h *Handler) MarkAllRead(c *gin.Context) {
	orgID, ok := h.org(c)
	if !ok {
		return
	}
	if err := h.svc.repo.MarkAllRead(c.Request.Context(), orgID, h.storeID(c)); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Delete godoc: DELETE /notifications/{id}
func (h *Handler) Delete(c *gin.Context) {
	orgID, ok := h.org(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification id", nil)
		return
	}
	if err := h.svc.repo.Delete(c.Request.Context(), orgID, h.storeID(c), id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// Create godoc: POST /notifications (owner/staff manual push for testing)
func (h *Handler) Create(c *gin.Context) {
	orgID, ok := h.org(c)
	if !ok {
		return
	}
	var req struct {
		StoreID  *uuid.UUID `json:"store_id"`
		Category string     `json:"category"`
		Title    string     `json:"title" binding:"required"`
		Body     string     `json:"body"`
		Link     string     `json:"link"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	n := &Notification{
		Category: req.Category,
		Title:    req.Title,
		Body:     req.Body,
		Link:     req.Link,
	}
	var storeID uuid.UUID
	if req.StoreID != nil {
		storeID = *req.StoreID
	}
	if err := h.svc.Create(c.Request.Context(), orgID, storeID, n); err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, notificationResponse(*n), "Notification sent")
}

func notificationResponse(n Notification) map[string]interface{} {
	group := "earlier"
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	switch {
	case n.CreatedAt.Year() == now.Year() && n.CreatedAt.YearDay() == now.YearDay():
		group = "today"
	case n.CreatedAt.Year() == yesterday.Year() && n.CreatedAt.YearDay() == yesterday.YearDay():
		group = "yesterday"
	}
	return map[string]interface{}{
		"id":         n.ID,
		"category":   n.Category,
		"title":      n.Title,
		"body":       n.Body,
		"link":       n.Link,
		"read":       n.Read,
		"group":      group,
		"created_at": n.CreatedAt,
	}
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
