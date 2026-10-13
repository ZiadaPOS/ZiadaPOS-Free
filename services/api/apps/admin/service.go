package admin

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Valid plan slugs for an organisation.
var validPlans = map[string]bool{
	"free":       true,
	"starter":    true,
	"growth":     true,
	"pro":        true,
	"enterprise": true,
}

var validStoreStatuses = map[string]bool{
	"open": true, "closed": true, "paused": true, "renovation": true,
}

var validUserRoles = map[string]bool{
	"admin": true, "owner": true, "staff": true,
}

var validUserEmployment = map[string]bool{
	"active": true, "probation": true, "on_leave": true, "terminated": true,
}

var validReviewStatuses = map[string]bool{
	"public": true, "hidden": true,
}

// Service validates admin write operations before touching persistence.
type Service struct {
	repo *Repo
}

// NewService builds an admin Service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// ── Patch DTOs ─────────────────────────────────────────────────────────────

type OrgPatch struct {
	Name         *string    `json:"name"`
	Slug         *string    `json:"slug"`
	LegalName    *string    `json:"legal_name"`
	TIN          *string    `json:"tin"`
	Phone        *string    `json:"phone"`
	Email        *string    `json:"email"`
	Region       *string    `json:"region"`
	District     *string    `json:"district"`
	Ward         *string    `json:"ward"`
	Street       *string    `json:"street"`
	Address      *string    `json:"address"`
	BusinessType *string    `json:"business_type"`
	Occupation   *string    `json:"occupation"`
	Plan         *string    `json:"plan"`
	MaxStores    *int       `json:"max_stores"`
	AICredits    *int       `json:"ai_credits_monthly"`
	TrialEndsAt  *time.Time `json:"trial_ends_at"`
	IsActive     *bool      `json:"is_active"`
}

// OrgPatchFields turns a validated patch into a column map for GORM Updates.
func (s *Service) OrgPatchFields(p OrgPatch) (map[string]interface{}, error) {
	f := map[string]interface{}{}
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" {
			return nil, errors.New("name cannot be empty")
		}
		f["name"] = n
	}
	if p.Slug != nil {
		n := strings.TrimSpace(*p.Slug)
		if n == "" {
			return nil, errors.New("slug cannot be empty")
		}
		f["slug"] = n
	}
	if p.LegalName != nil {
		f["legal_name"] = strings.TrimSpace(*p.LegalName)
	}
	if p.TIN != nil {
		f["tin"] = strings.TrimSpace(*p.TIN)
	}
	if p.Phone != nil {
		f["phone"] = *p.Phone
	}
	if p.Email != nil {
		f["email"] = *p.Email
	}
	if p.Region != nil {
		f["region"] = *p.Region
	}
	if p.District != nil {
		f["district"] = *p.District
	}
	if p.Ward != nil {
		f["ward"] = *p.Ward
	}
	if p.Street != nil {
		f["street"] = *p.Street
	}
	if p.Address != nil {
		f["address"] = *p.Address
	}
	if p.BusinessType != nil {
		f["business_type"] = *p.BusinessType
	}
	if p.Occupation != nil {
		f["occupation"] = *p.Occupation
	}
	if p.Plan != nil {
		plan := strings.ToLower(*p.Plan)
		if !validPlans[plan] {
			return nil, errors.New("invalid plan")
		}
		f["plan"] = plan
	}
	if p.MaxStores != nil {
		if *p.MaxStores < 0 {
			return nil, errors.New("max_stores cannot be negative")
		}
		f["max_stores"] = *p.MaxStores
	}
	if p.AICredits != nil {
		if *p.AICredits < 0 {
			return nil, errors.New("ai_credits_monthly cannot be negative")
		}
		f["ai_credits_monthly"] = *p.AICredits
	}
	if p.TrialEndsAt != nil {
		f["trial_ends_at"] = *p.TrialEndsAt
	}
	if p.IsActive != nil {
		f["is_active"] = *p.IsActive
	}
	if len(f) == 0 {
		return nil, errors.New("no fields to update")
	}
	return f, nil
}

type UserPatch struct {
	Role             *string `json:"role"`
	EmploymentStatus *string `json:"employment_status"`
	StoreID          *string `json:"store_id"`
	Shift            *string `json:"shift"`
	CanRefund        *bool   `json:"can_refund"`
	CanDiscount      *bool   `json:"can_discount"`
	CanViewReports   *bool   `json:"can_view_reports"`
	IsActive         *bool   `json:"is_active"`
}

func (s *Service) UserPatchFields(p UserPatch) (map[string]interface{}, error) {
	f := map[string]interface{}{}
	if p.Role != nil {
		role := strings.ToLower(*p.Role)
		if !validUserRoles[role] {
			return nil, errors.New("invalid role")
		}
		f["role"] = role
	}
	if p.EmploymentStatus != nil {
		st := strings.ToLower(*p.EmploymentStatus)
		if !validUserEmployment[st] {
			return nil, errors.New("invalid employment_status")
		}
		f["employment_status"] = st
	}
	if p.StoreID != nil {
		if strings.TrimSpace(*p.StoreID) == "" {
			f["store_id"] = nil
		} else {
			id, err := uuid.Parse(*p.StoreID)
			if err != nil {
				return nil, errors.New("invalid store_id")
			}
			f["store_id"] = id
		}
	}
	if p.Shift != nil {
		f["shift"] = *p.Shift
	}
	if p.CanRefund != nil {
		f["can_refund"] = *p.CanRefund
	}
	if p.CanDiscount != nil {
		f["can_discount"] = *p.CanDiscount
	}
	if p.CanViewReports != nil {
		f["can_view_reports"] = *p.CanViewReports
	}
	if p.IsActive != nil {
		f["is_active"] = *p.IsActive
	}
	if len(f) == 0 {
		return nil, errors.New("no fields to update")
	}
	return f, nil
}

type StorePatch struct {
	Name       *string `json:"name"`
	Code       *string `json:"code"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Area       *string `json:"area"`
	Address    *string `json:"address"`
	Status     *string `json:"status"`
	IsActive   *bool   `json:"is_active"`
	VatEnabled *bool   `json:"vat_enabled"`
}

func (s *Service) StorePatchFields(p StorePatch) (map[string]interface{}, error) {
	f := map[string]interface{}{}
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" {
			return nil, errors.New("name cannot be empty")
		}
		f["name"] = n
	}
	if p.Code != nil {
		f["code"] = strings.TrimSpace(*p.Code)
	}
	if p.Phone != nil {
		f["phone"] = *p.Phone
	}
	if p.Email != nil {
		f["email"] = *p.Email
	}
	if p.Area != nil {
		f["area"] = *p.Area
	}
	if p.Address != nil {
		f["address"] = *p.Address
	}
	if p.Status != nil {
		st := strings.ToLower(*p.Status)
		if !validStoreStatuses[st] {
			return nil, errors.New("invalid store status")
		}
		f["status"] = st
	}
	if p.IsActive != nil {
		f["is_active"] = *p.IsActive
	}
	if p.VatEnabled != nil {
		f["vat_enabled"] = *p.VatEnabled
	}
	if len(f) == 0 {
		return nil, errors.New("no fields to update")
	}
	return f, nil
}

type ProductPatch struct {
	Name         *string  `json:"name"`
	SellingPrice *float64 `json:"selling_price"`
	CostPrice    *float64 `json:"cost_price"`
	Stock        *float64 `json:"stock"`
	ReorderPoint *float64 `json:"reorder_point"`
	IsActive     *bool    `json:"is_active"`
	TaxRate      *float64 `json:"tax_rate"`
}

func (s *Service) ProductPatchFields(p ProductPatch) (map[string]interface{}, error) {
	f := map[string]interface{}{}
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" {
			return nil, errors.New("name cannot be empty")
		}
		f["name"] = n
	}
	if p.SellingPrice != nil && *p.SellingPrice < 0 {
		return nil, errors.New("selling_price cannot be negative")
	}
	if p.CostPrice != nil && *p.CostPrice < 0 {
		return nil, errors.New("cost_price cannot be negative")
	}
	if p.Stock != nil && *p.Stock < 0 {
		return nil, errors.New("stock cannot be negative")
	}
	if p.ReorderPoint != nil && *p.ReorderPoint < 0 {
		return nil, errors.New("reorder_point cannot be negative")
	}
	if p.TaxRate != nil && *p.TaxRate < 0 {
		return nil, errors.New("tax_rate cannot be negative")
	}
	if p.SellingPrice != nil {
		f["selling_price"] = *p.SellingPrice
	}
	if p.CostPrice != nil {
		f["cost_price"] = *p.CostPrice
	}
	if p.Stock != nil {
		f["stock"] = *p.Stock
	}
	if p.ReorderPoint != nil {
		f["reorder_point"] = *p.ReorderPoint
	}
	if p.TaxRate != nil {
		f["tax_rate"] = *p.TaxRate
	}
	if p.IsActive != nil {
		f["is_active"] = *p.IsActive
	}
	if len(f) == 0 {
		return nil, errors.New("no fields to update")
	}
	return f, nil
}

type ReviewPatch struct {
	IsPublic *bool   `json:"is_public"`
	Status   *string `json:"status"`
}

type CategoryInput struct {
	OrganisationID string `json:"organisation_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	SortOrder      int    `json:"sort_order"`
	IsActive       *bool  `json:"is_active"`
}

func (s *Service) CategoryFields(p CategoryInput, creating bool) (uuid.UUID, map[string]interface{}, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return uuid.Nil, nil, errors.New("name is required")
	}
	fields := map[string]interface{}{"name": name, "description": strings.TrimSpace(p.Description), "sort_order": p.SortOrder}
	if p.IsActive != nil {
		fields["is_active"] = *p.IsActive
	} else if creating {
		fields["is_active"] = true
	}
	if creating {
		orgID, err := uuid.Parse(strings.TrimSpace(p.OrganisationID))
		if err != nil {
			return uuid.Nil, nil, errors.New("valid organisation_id is required")
		}
		return orgID, fields, nil
	}
	return uuid.Nil, fields, nil
}

func (s *Service) ReviewPatchFields(p ReviewPatch) (map[string]interface{}, error) {
	f := map[string]interface{}{}
	if p.IsPublic != nil {
		f["is_public"] = *p.IsPublic
	}
	if p.Status != nil {
		st := strings.ToLower(*p.Status)
		if !validReviewStatuses[st] {
			return nil, errors.New("invalid review status")
		}
		f["is_public"] = st == "public"
	}
	if len(f) == 0 {
		return nil, errors.New("no fields to update")
	}
	return f, nil
}
