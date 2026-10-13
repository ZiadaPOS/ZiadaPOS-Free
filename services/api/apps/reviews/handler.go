package reviews

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Handler exposes review HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a reviews Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

// Submit godoc: POST /reviews (authenticated)
func (h *Handler) Submit(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.BadRequest(c, "No organisation linked to this account", nil)
		return
	}
	userID, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	var req SubmitInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.Submit(c.Request.Context(), orgID, userID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Review submitted. Thank you!")
}

// Public godoc: GET /reviews/public (unauthenticated landing page)
func (h *Handler) Public(c *gin.Context) {
	limit := atoi(c.DefaultQuery("limit", "20"))
	out, err := h.svc.Public(c.Request.Context(), limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"items": out, "count": len(out)})
}
