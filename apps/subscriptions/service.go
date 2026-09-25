package subscriptions

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/server"
	"github.com/troubleman96/ziadapos/internal/util"
)

// Service implements subscription business rules.
type Service struct {
	repo *Repo
	deps *server.Deps
}

// NewService wires a subscriptions Service.
func NewService(repo *Repo, deps ...*server.Deps) *Service {
	s := &Service{repo: repo}
	if len(deps) > 0 {
		s.deps = deps[0]
	}
	return s
}

// ── Plans ─────────────────────────────────────────────────────────────────────

// ListPlans returns pricing plans (publicly only active ones).
func (s *Service) ListPlans(ctx context.Context, includeInactive bool) ([]map[string]interface{}, error) {
	plans, err := s.repo.ListPlans(includeInactive)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(plans))
	for i := range plans {
		out = append(out, planResponse(&plans[i]))
	}
	return out, nil
}

// CreatePlan adds a new pricing tier (admin).
func (s *Service) CreatePlan(ctx context.Context, in *PlanInput) (*map[string]interface{}, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apierr.BadRequest("Plan name is required")
	}
	if in.PricePerMonth < 0 {
		return nil, apierr.BadRequest("Invalid price").WithField("price_per_month", "cannot be negative")
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = util.Slugify(in.Name)
	}
	p := &SubscriptionPlan{
		Name:                    strings.TrimSpace(in.Name),
		Slug:                    slug,
		Description:             in.Description,
		PricePerMonth:           in.PricePerMonth,
		DurationMonths:          in.DurationMonths,
		IncludedStores:          in.IncludedStores,
		ExtraStorePricePerMonth: in.ExtraStorePricePerMonth,
		IsActive:                true,
		SortOrder:               in.SortOrder,
	}
	if p.DurationMonths <= 0 {
		p.DurationMonths = 1
	}
	if p.IncludedStores <= 0 {
		p.IncludedStores = DefaultIncludedStores
	}
	if p.ExtraStorePricePerMonth <= 0 {
		p.ExtraStorePricePerMonth = DefaultExtraStorePrice
	}
	if err := s.repo.CreatePlan(p); err != nil {
		return nil, err
	}
	out := planResponse(p)
	return &out, nil
}

// UpdatePlan edits a pricing tier (admin).
func (s *Service) UpdatePlan(ctx context.Context, id uuid.UUID, in *PlanInput) (*map[string]interface{}, error) {
	p, err := s.repo.GetPlan(id)
	if err != nil {
		return nil, apierr.NotFound("Plan not found")
	}
	if strings.TrimSpace(in.Name) != "" {
		p.Name = strings.TrimSpace(in.Name)
	}
	if in.Slug != "" {
		p.Slug = strings.TrimSpace(in.Slug)
	}
	if in.Description != "" {
		p.Description = in.Description
	}
	if in.PricePerMonth > 0 {
		p.PricePerMonth = in.PricePerMonth
	}
	if in.DurationMonths > 0 {
		p.DurationMonths = in.DurationMonths
	}
	if in.IncludedStores > 0 {
		p.IncludedStores = in.IncludedStores
	}
	if in.ExtraStorePricePerMonth > 0 {
		p.ExtraStorePricePerMonth = in.ExtraStorePricePerMonth
	}
	if in.SortOrder >= 0 {
		p.SortOrder = in.SortOrder
	}
	if err := s.repo.SavePlan(p); err != nil {
		return nil, err
	}
	out := planResponse(p)
	return &out, nil
}

// DeactivatePlan soft-deactivates a pricing tier (admin).
func (s *Service) DeactivatePlan(ctx context.Context, id uuid.UUID) error {
	p, err := s.repo.GetPlan(id)
	if err != nil {
		return apierr.NotFound("Plan not found")
	}
	p.IsActive = false
	return s.repo.SavePlan(p)
}

// ── Owner endpoints ───────────────────────────────────────────────────────────

// MySubscription returns the most recent subscription for an organisation.
func (s *Service) MySubscription(ctx context.Context, orgID uuid.UUID) (*map[string]interface{}, error) {
	sub, err := s.repo.LatestSubscription(orgID)
	if err != nil {
		return nil, apierr.NotFound("No subscription found")
	}
	out := subscriptionResponse(sub)
	return &out, nil
}

// StoreLimit computes the can-add-store gate for an owner.
func (s *Service) StoreLimit(ctx context.Context, orgID uuid.UUID) (map[string]interface{}, error) {
	current, err := s.repo.CountActiveStores(orgID)
	if err != nil {
		return nil, err
	}
	maxAllowed, err := s.repo.OrgMaxStores(orgID)
	if err != nil {
		return nil, apierr.NotFound("Organisation not found")
	}
	if maxAllowed <= 0 {
		maxAllowed = DefaultIncludedStores
	}
	remaining := maxAllowed - int(current)
	if remaining < 0 {
		remaining = 0
	}
	canAdd := remaining > 0

	extraPrice := int64(DefaultExtraStorePrice)
	sub, _ := s.repo.LatestSubscription(orgID)
	var subStatus *string
	subActive := false
	daysRemaining := 0
	if sub != nil {
		ss := sub.Status
		subStatus = &ss
		subActive = sub.IsActiveNow()
		daysRemaining = sub.DaysRemaining()
		if sub.Plan != nil && sub.Plan.ExtraStorePricePerMonth > 0 {
			extraPrice = sub.Plan.ExtraStorePricePerMonth
		}
	}

	message := ""
	if canAdd {
		message = "You can add more stores."
	} else {
		message = "You have reached your store limit. Additional stores cost TZS and require support."
	}
	_ = message

	return map[string]interface{}{
		"can_add_store":               canAdd,
		"current_active_stores":       current,
		"max_stores_allowed":          maxAllowed,
		"remaining_slots":             remaining,
		"extra_store_price_per_month": extraPrice,
		"subscription_status":         subStatus,
		"subscription_is_active":      subActive,
		"days_remaining":              daysRemaining,
	}, nil
}

// ── Admin endpoints ───────────────────────────────────────────────────────────

// AdminList returns all subscriptions with optional filters.
func (s *Service) AdminList(ctx context.Context, status, orgID string, page, pageSize int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	items, total, err := s.repo.ListSubscriptions(status, orgID, page, pageSize)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, subscriptionResponse(&items[i]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": page, "page_size": pageSize,
	}, nil
}

// AdminGet returns one subscription for admin review.
func (s *Service) AdminGet(ctx context.Context, id uuid.UUID) (*map[string]interface{}, error) {
	sub, err := s.repo.GetSubscription(id)
	if err != nil {
		return nil, apierr.NotFound("Subscription not found")
	}
	out := subscriptionResponse(sub)
	return &out, nil
}

// Activate confirms payment and sets the subscription status (admin).
func (s *Service) Activate(ctx context.Context, id uuid.UUID, in *ActivateInput) (*map[string]interface{}, error) {
	if in.Status != StatusTrial && in.Status != StatusActive {
		return nil, apierr.BadRequest("Invalid status").WithField("status", "must be trial or active")
	}
	sub, err := s.repo.GetSubscription(id)
	if err != nil {
		return nil, apierr.NotFound("Subscription not found")
	}
	sub.Status = in.Status
	if in.PaymentReference != "" {
		sub.PaymentReference = in.PaymentReference
	}
	if in.PaymentDate != nil {
		sub.PaymentDate = in.PaymentDate
	} else {
		now := time.Now()
		sub.PaymentDate = &now
	}
	if in.AmountPaid > 0 {
		sub.AmountPaid = in.AmountPaid
	}
	if in.Notes != "" {
		sub.Notes = in.Notes
	}
	if err := s.repo.SaveSubscription(sub); err != nil {
		return nil, err
	}
	if err := s.syncOrg(sub); err != nil {
		return nil, err
	}
	out := subscriptionResponse(sub)
	return &out, nil
}

// ExtraStores grants additional store slots (admin).
func (s *Service) ExtraStores(ctx context.Context, id uuid.UUID, extra int) (map[string]interface{}, error) {
	if extra < 0 {
		return nil, apierr.BadRequest("Invalid extra_stores").WithField("extra_stores", "cannot be negative")
	}
	sub, err := s.repo.GetSubscription(id)
	if err != nil {
		return nil, apierr.NotFound("Subscription not found")
	}
	sub.ExtraStores = extra
	if err := s.repo.SaveSubscription(sub); err != nil {
		return nil, err
	}
	if err := s.syncOrg(sub); err != nil {
		return nil, err
	}
	resp := subscriptionResponse(sub)
	resp["org_max_stores_now"] = sub.MaxStoresAllowed()
	return resp, nil
}

// syncOrg keeps organisation.plan and organisation.max_stores in sync.
func (s *Service) syncOrg(sub *Subscription) error {
	maxStores := sub.MaxStoresAllowed()
	plan := "free"
	if sub.Status == StatusActive {
		plan = "pro"
	}
	if err := s.repo.UpdateOrgMaxStores(sub.OrganisationID, maxStores); err != nil {
		return err
	}
	return s.repo.UpdateOrgPlan(sub.OrganisationID, plan)
}

// ── Responses ────────────────────────────────────────────────────────────────

func planResponse(p *SubscriptionPlan) map[string]interface{} {
	return map[string]interface{}{
		"id":                          p.ID,
		"name":                        p.Name,
		"slug":                        p.Slug,
		"description":                 p.Description,
		"price_per_month":             p.PricePerMonth,
		"duration_months":             p.DurationMonths,
		"total_price":                 p.TotalPrice(),
		"included_stores":             p.IncludedStores,
		"extra_store_price_per_month": p.ExtraStorePricePerMonth,
		"extra_store_price_total":     p.ExtraStorePriceTotal(),
		"is_active":                   p.IsActive,
		"sort_order":                  p.SortOrder,
	}
}

func subscriptionResponse(s *Subscription) map[string]interface{} {
	return map[string]interface{}{
		"id":                 s.ID,
		"organisation":       s.OrganisationID,
		"plan":               s.PlanID,
		"plan_detail":        planOrNil(s.Plan),
		"status":             s.Status,
		"start_date":         dateOrNil(s.StartDate),
		"end_date":           dateOrNil(s.EndDate),
		"is_trial":           s.IsTrial,
		"trial_fee":          s.TrialFee,
		"extra_stores":       s.ExtraStores,
		"amount_paid":        s.AmountPaid,
		"payment_reference":  s.PaymentReference,
		"payment_date":       timeOrNil(s.PaymentDate),
		"is_active_now":      s.IsActiveNow(),
		"days_remaining":     s.DaysRemaining(),
		"max_stores_allowed": s.MaxStoresAllowed(),
		"total_amount_due":   s.TotalAmountDue(),
		"notes":              s.Notes,
		"created_at":         s.CreatedAt,
		"updated_at":         s.UpdatedAt,
	}
}

func planOrNil(p *SubscriptionPlan) interface{} {
	if p == nil {
		return nil
	}
	return planResponse(p)
}

func dateOrNil(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func timeOrNil(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t
}

// ── Inputs ────────────────────────────────────────────────────────────────────

// PlanInput is the create/update payload for pricing tiers.
type PlanInput struct {
	Name                    string `json:"name"`
	Slug                    string `json:"slug"`
	Description             string `json:"description"`
	PricePerMonth           int64  `json:"price_per_month"`
	DurationMonths          int    `json:"duration_months"`
	IncludedStores          int    `json:"included_stores"`
	ExtraStorePricePerMonth int64  `json:"extra_store_price_per_month"`
	SortOrder               int    `json:"sort_order"`
}

// ActivateInput confirms payment for a subscription.
type ActivateInput struct {
	Status           string     `json:"status"`
	PaymentReference string     `json:"payment_reference"`
	PaymentDate      *time.Time `json:"payment_date"`
	AmountPaid       int64      `json:"amount_paid"`
	Notes            string     `json:"notes"`
}
