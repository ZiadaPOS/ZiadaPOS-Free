package subscriptions

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// Handler exposes subscription HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler builds a subscriptions Handler.
func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) orgID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return uuid.Nil, false
	}
	return id, true
}

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}

// ListPlans godoc: GET /subscriptions/plans (public — active plans only)
func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.svc.ListPlans(c.Request.Context(), false)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"plans": plans, "count": len(plans)})
}

// AdminPlans lists active and inactive plans for the platform console.
func (h *Handler) AdminPlans(c *gin.Context) {
	plans, err := h.svc.ListPlans(c.Request.Context(), true)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, map[string]interface{}{"plans": plans, "count": len(plans)})
}

// GetPlan godoc: GET /subscriptions/plans/{id} (public)
func (h *Handler) GetPlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid plan id", nil)
		return
	}
	out, err := h.svc.repo.GetPlan(id)
	if err != nil {
		writeErr(c, apierr.NotFound("Plan not found"))
		return
	}
	response.OK(c, planResponse(out))
}

// CreatePlan godoc: POST /subscriptions/plans (admin)
func (h *Handler) CreatePlan(c *gin.Context) {
	var req PlanInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.CreatePlan(c.Request.Context(), &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Plan created")
}

// UpdatePlan godoc: PATCH /subscriptions/plans/{id} (admin)
func (h *Handler) UpdatePlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid plan id", nil)
		return
	}
	var req PlanInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.UpdatePlan(c.Request.Context(), id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// DeactivatePlan godoc: DELETE /subscriptions/plans/{id} (admin)
func (h *Handler) DeactivatePlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid plan id", nil)
		return
	}
	if err := h.svc.DeactivatePlan(c.Request.Context(), id); err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, nil)
}

// MySubscription godoc: GET /subscriptions/my-subscription (owner)
func (h *Handler) MySubscription(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	out, err := h.svc.MySubscription(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// StoreLimit godoc: GET /subscriptions/store-limit (owner)
func (h *Handler) StoreLimit(c *gin.Context) {
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	out, err := h.svc.StoreLimit(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// AdminList godoc: GET /subscriptions/all (admin)
func (h *Handler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	out, err := h.svc.AdminList(c.Request.Context(), c.Query("status"), c.Query("organisation"), page, pageSize)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// AdminGet godoc: GET /subscriptions/all/{id} (admin)
func (h *Handler) AdminGet(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid subscription id", nil)
		return
	}
	out, err := h.svc.AdminGet(c.Request.Context(), id)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Activate godoc: POST /subscriptions/all/{id}/activate (admin)
func (h *Handler) Activate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid subscription id", nil)
		return
	}
	var req ActivateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.Activate(c.Request.Context(), id, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// ExtraStores godoc: PATCH /subscriptions/all/{id}/extra-stores (admin)
func (h *Handler) ExtraStores(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid subscription id", nil)
		return
	}
	var req struct {
		ExtraStores int `json:"extra_stores"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.ExtraStores(c.Request.Context(), id, req.ExtraStores)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// OnUserRegistered creates the initial 7-day trial subscription.
func OnUserRegistered(repo *Repo) func(ctx context.Context, e events.Event) error {
	return func(ctx context.Context, e events.Event) error {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		var orgID uuid.UUID
		switch v := payload["org_id"].(type) {
		case uuid.UUID:
			orgID = v
		case string:
			id, err := uuid.Parse(v)
			if err != nil {
				return nil
			}
			orgID = id
		default:
			return nil
		}
		now := time.Now()
		end := now.AddDate(0, 0, trialDays)
		sub := &Subscription{
			OrganisationID: orgID,
			Status:         StatusTrial,
			StartDate:      &now,
			EndDate:        &end,
			IsTrial:        true,
			TrialFee:       DefaultTrialFee,
		}
		return repo.CreateSubscription(sub)
	}
}

const trialDays = 7
