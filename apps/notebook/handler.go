package notebook

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes notebook HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a notebook Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) storeID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.StoreID(c))
	if err != nil {
		response.NotFound(c, "Store not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) userID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(middleware.UserID(c))
	return id
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

// List godoc: GET /notebook
func (h *Handler) List(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	notes, err := h.svc.List(c.Request.Context(), storeID, c.Query("tag"), c.Query("search"), c.Query("ordering"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"items": notes, "count": len(notes)})
}

// Get godoc: GET /notebook/{id}
func (h *Handler) Get(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid note id", nil)
		return
	}
	note, err := h.svc.Get(c.Request.Context(), storeID, id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, note)
}

// Create godoc: POST /notebook
func (h *Handler) Create(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	orgID, ok2 := h.orgID(c)
	if !ok2 {
		return
	}
	var req NoteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	note, err := h.svc.Create(c.Request.Context(), orgID, storeID, h.userID(c), &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, note, "Note created")
}

// Update godoc: PATCH /notebook/{id}
func (h *Handler) Update(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid note id", nil)
		return
	}
	var req NoteInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	note, err := h.svc.Update(c.Request.Context(), storeID, id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, note)
}

// Delete godoc: DELETE /notebook/{id}
func (h *Handler) Delete(c *gin.Context) {
	storeID, ok := h.storeID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid note id", nil)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), storeID, id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}
