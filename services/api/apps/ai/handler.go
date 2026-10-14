package ai

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes Ziada AI HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds an ai Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

func (h *Handler) orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) storeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.BadRequest(c, "Store not found — please select an active store", nil)
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) userID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.NotFound(c, "User not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) convID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid conversation id", nil)
		return uuid.Nil, false
	}
	return id, true
}

// StartChat godoc: POST /ai/chat — start a new conversation.
func (h *Handler) StartChat(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	var req struct {
		Message            string `json:"message" binding:"required"`
		Title              string `json:"title"`
		DataSharingConsent bool   `json:"data_sharing_consent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Message is required", nil)
		return
	}
	if !req.DataSharingConsent {
		response.BadRequest(c, "Confirm that your message and minimized shop context may be sent to the configured AI provider", nil)
		return
	}
	out, err := h.svc.StartChat(c.Request.Context(), orgID, storeID, userID, req.Message, req.Title)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Conversation started")
}

// ContinueChat godoc: POST /ai/conversations/{id}/chat — continue a thread.
func (h *Handler) ContinueChat(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	convID, ok := h.convID(c)
	if !ok {
		return
	}
	var req struct {
		Message            string `json:"message" binding:"required"`
		DataSharingConsent bool   `json:"data_sharing_consent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Message is required", nil)
		return
	}
	if !req.DataSharingConsent {
		response.BadRequest(c, "Confirm that your message and minimized shop context may be sent to the configured AI provider", nil)
		return
	}
	out, err := h.svc.ContinueChat(c.Request.Context(), userID, convID, req.Message)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// List godoc: GET /ai/conversations
func (h *Handler) List(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	out, err := h.svc.List(c.Request.Context(), userID, storeID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"conversations": out, "items": out, "count": len(out)})
}

// Get godoc: GET /ai/conversations/{id}
func (h *Handler) Get(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	convID, ok := h.convID(c)
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), userID, convID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Update godoc: PATCH /ai/conversations/{id} — rename or archive.
func (h *Handler) Update(c *gin.Context) {
	userID, ok := h.userID(c)
	if !ok {
		return
	}
	convID, ok := h.convID(c)
	if !ok {
		return
	}
	var req struct {
		Title    string `json:"title"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.Update(c.Request.Context(), userID, convID, req.Title, req.IsActive)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Models godoc: GET /ai/models — owner-triggered Ngamia model catalog lookup.
func (h *Handler) Models(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	models, err := h.svc.ListModels(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, models)
}

// Suggestions godoc: GET /ai/suggestions — contextual prompt chips + credits.
func (h *Handler) Suggestions(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	out, err := h.svc.Suggestions(c.Request.Context(), storeID, orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Usage godoc: GET /ai/usage — token + credit usage log.
func (h *Handler) Usage(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	out, err := h.svc.Usage(c.Request.Context(), storeID, orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}
