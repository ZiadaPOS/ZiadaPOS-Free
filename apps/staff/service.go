package staff

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/util"
	"gorm.io/gorm"
)

// Service orchestrates staff management and performance stats.
type Service struct {
	repo    *Repo
	account *accounts.AccountService
	cache   *cache.Cache
}

// NewService wires the staff service, reusing the accounts app for identity
// CRUD so that staff.created keeps firing the SMS temp-password listener.
func NewService(repo *Repo, db *gorm.DB, cacheSvc *cache.Cache, bus *events.Bus, cfg *config.Config) *Service {
	return &Service{
		repo:    repo,
		account: accounts.NewAccountService(accounts.NewRepo(db), cacheSvc, bus, cfg),
		cache:   cacheSvc,
	}
}

// CreateRequest is the payload for creating a staff member.
type CreateRequest struct {
	FirstName        string     `json:"first_name"`
	LastName         string     `json:"last_name"`
	Phone            string     `json:"phone"`
	Email            string     `json:"email"`
	Role             string     `json:"role"`
	StoreID          *uuid.UUID `json:"store_id"`
	Shift            string     `json:"shift"`
	EmploymentStatus string     `json:"employment_status"`
	AvatarHue        *int       `json:"avatar_hue"`
	PIN              string     `json:"pin"`
	CanRefund        bool       `json:"can_refund"`
	CanDiscount      bool       `json:"can_discount"`
	CanViewReports   bool       `json:"can_view_reports"`
}

// StaffResponse is the public representation of a staff member.
type StaffResponse struct {
	accounts.UserResponse
	ShiftDisplay string       `json:"shift_display"`
	Store        *StoreDetail `json:"store_detail,omitempty"`
	Stats        *StaffStats  `json:"stats,omitempty"`
}

// Create adds a staff member, assigning a random temp password and firing
// the staff.created event (SMS + logging).
func (s *Service) Create(ctx context.Context, orgID, storeID uuid.UUID, req *CreateRequest) (*StaffResponse, error) {
	role := req.Role
	if role == "" {
		role = accounts.RoleStaff
	}
	if role != accounts.RoleStaff && role != accounts.RoleOwner {
		return nil, apierr.BadRequest("Invalid role").WithField("role", "must be staff or owner")
	}
	if req.Shift != "" && !contains(ValidShifts, req.Shift) {
		return nil, apierr.BadRequest("Invalid shift").WithField("shift", "must be one of morning, evening, full_day, weekend")
	}
	if req.PIN != "" && !validPIN(req.PIN) {
		return nil, apierr.BadRequest("Invalid pin").WithField("pin", "must be exactly 4 digits")
	}
	if req.EmploymentStatus != "" && !contains(ValidStatuses, req.EmploymentStatus) {
		return nil, apierr.BadRequest("Invalid employment_status").WithField("employment_status", "must be one of active, on_leave, inactive")
	}
	store := req.StoreID
	if store == nil {
		store = &storeID
	} else if *store != storeID {
		return nil, apierr.Forbidden("Staff must belong to the current store")
	}

	// Role-defaulted permissions: owners/admins get refund + reports, discount
	// is enabled for everyone by default (old Python behaviour).
	canRefund := req.CanRefund
	canViewReports := req.CanViewReports
	canDiscount := true
	if role != accounts.RoleStaff {
		canRefund = true
		canViewReports = true
	}

	out, err := s.account.CreateStaff(ctx, orgID, &accounts.CreateStaffRequest{
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Phone:          req.Phone,
		Email:          req.Email,
		Role:           role,
		StoreID:        store,
		CanRefund:      canRefund,
		CanDiscount:    canDiscount,
		CanViewReports: canViewReports,
	})
	if err != nil {
		return nil, err
	}
	resp := &StaffResponse{UserResponse: *out, ShiftDisplay: ShiftDisplay(req.Shift)}

	u, err := s.repo.Get(storeID, out.ID)
	if err != nil {
		return resp, nil
	}
	updated := false
	if req.Shift != "" && u.Shift != req.Shift {
		u.Shift = req.Shift
		updated = true
	}
	if req.EmploymentStatus != "" && u.EmploymentStatus != req.EmploymentStatus {
		u.EmploymentStatus = req.EmploymentStatus
		u.IsActive = req.EmploymentStatus == EmpActive
		updated = true
	}
	if req.AvatarHue != nil && *req.AvatarHue != u.AvatarHue {
		u.AvatarHue = *req.AvatarHue
		updated = true
	}
	if req.PIN != "" && u.PIN != req.PIN {
		u.PIN = req.PIN
		updated = true
	}
	if updated {
		_ = s.repo.Update(u)
	}
	return s.toResponse(storeID, u), nil
}

// List returns staff with optional per-member stats.
func (s *Service) List(ctx context.Context, storeID uuid.UUID, opts ListOptions, withStats bool) ([]StaffResponse, int64, error) {
	users, total, err := s.repo.List(storeID, opts)
	if err != nil {
		return nil, 0, err
	}
	now := time.Now()
	today := util.StartOfDay(now)
	month := util.StartOfMonth(now)
	out := make([]StaffResponse, 0, len(users))
	for i := range users {
		item := s.toResponse(storeID, &users[i])
		if withStats {
			st, err := s.repo.Stats(storeID, users[i].ID, today, month)
			if err == nil {
				item.Stats = st
			}
		}
		out = append(out, *item)
	}
	return out, total, nil
}

// Get returns a single staff member with stats.
func (s *Service) Get(ctx context.Context, storeID, userID uuid.UUID) (*StaffResponse, error) {
	u, err := s.repo.Get(storeID, userID)
	if err != nil {
		return nil, apierr.NotFound("Staff member not found")
	}
	now := time.Now()
	st, err := s.repo.Stats(storeID, userID, util.StartOfDay(now), util.StartOfMonth(now))
	if err != nil {
		return nil, err
	}
	out := s.toResponse(storeID, u)
	out.Stats = st
	return out, nil
}

// toResponse builds the public staff shape, embedding the store and shift label.
func (s *Service) toResponse(storeID uuid.UUID, u *accounts.User) *StaffResponse {
	out := &StaffResponse{
		UserResponse: accounts.ToUserResponse(u),
		ShiftDisplay: ShiftDisplay(u.Shift),
	}
	if store, err := s.repo.Store(storeID); err == nil {
		out.Store = store
	}
	return out
}

// UpdateRequest patches a staff member's profile.
type UpdateRequest struct {
	FirstName        *string `json:"first_name"`
	LastName         *string `json:"last_name"`
	Email            *string `json:"email"`
	Phone            *string `json:"phone"`
	Role             *string `json:"role"`
	Shift            *string `json:"shift"`
	EmploymentStatus *string `json:"employment_status"`
	AvatarHue        *int    `json:"avatar_hue"`
	PIN              *string `json:"pin"`
	IsActive         *bool   `json:"is_active"`
}

// Update patches profile fields.
func (s *Service) Update(ctx context.Context, storeID, userID uuid.UUID, req *UpdateRequest) (*StaffResponse, error) {
	u, err := s.repo.Get(storeID, userID)
	if err != nil {
		return nil, apierr.NotFound("Staff member not found")
	}
	if req.FirstName != nil {
		u.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		u.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.Email != nil {
		u.Email = accounts.OptionalEmail(*req.Email)
	}
	if req.Phone != nil {
		phone, err := normalizePhone(*req.Phone)
		if err != nil {
			return nil, apierr.BadRequest("Invalid phone").WithField("phone", err.Error())
		}
		u.Phone = phone
	}
	if req.Role != nil {
		if *req.Role != accounts.RoleStaff && *req.Role != accounts.RoleOwner {
			return nil, apierr.BadRequest("Invalid role").WithField("role", "must be staff or owner")
		}
		u.Role = *req.Role
	}
	if req.Shift != nil {
		if !contains(ValidShifts, *req.Shift) {
			return nil, apierr.BadRequest("Invalid shift").WithField("shift", "must be one of morning, evening, full_day, weekend")
		}
		u.Shift = *req.Shift
	}
	if req.EmploymentStatus != nil {
		if !contains(ValidStatuses, *req.EmploymentStatus) {
			return nil, apierr.BadRequest("Invalid employment_status").WithField("employment_status", "must be one of active, on_leave, inactive")
		}
		u.EmploymentStatus = *req.EmploymentStatus
		u.IsActive = *req.EmploymentStatus == EmpActive
	}
	if req.AvatarHue != nil && *req.AvatarHue >= 0 && *req.AvatarHue <= 360 {
		u.AvatarHue = *req.AvatarHue
	}
	if req.PIN != nil {
		if *req.PIN != "" && !validPIN(*req.PIN) {
			return nil, apierr.BadRequest("Invalid pin").WithField("pin", "must be exactly 4 digits")
		}
		u.PIN = *req.PIN
	}
	if req.IsActive != nil {
		u.IsActive = *req.IsActive
		if *req.IsActive {
			if u.EmploymentStatus == EmpInactive {
				u.EmploymentStatus = EmpActive
			}
		} else {
			u.EmploymentStatus = EmpInactive
		}
	}
	if err := s.repo.Update(u); err != nil {
		return nil, err
	}
	now := time.Now()
	st, err := s.repo.Stats(storeID, userID, util.StartOfDay(now), util.StartOfMonth(now))
	if err != nil {
		return nil, err
	}
	out := s.toResponse(storeID, u)
	out.Stats = st
	return out, nil
}

// ShiftRequest updates shift and employment status.
type ShiftRequest struct {
	Shift            *string `json:"shift"`
	EmploymentStatus *string `json:"employment_status"`
}

// UpdateShift changes a staff member's shift assignment.
func (s *Service) UpdateShift(ctx context.Context, storeID, userID uuid.UUID, req *ShiftRequest) (*StaffResponse, error) {
	u, err := s.repo.Get(storeID, userID)
	if err != nil {
		return nil, apierr.NotFound("Staff member not found")
	}
	if req.Shift != nil {
		if !contains(ValidShifts, *req.Shift) {
			return nil, apierr.BadRequest("Invalid shift").WithField("shift", "must be one of morning, evening, full_day, weekend")
		}
		u.Shift = *req.Shift
	}
	if req.EmploymentStatus != nil {
		if !contains(ValidStatuses, *req.EmploymentStatus) {
			return nil, apierr.BadRequest("Invalid employment_status").WithField("employment_status", "must be one of active, on_leave, inactive")
		}
		u.EmploymentStatus = *req.EmploymentStatus
		u.IsActive = *req.EmploymentStatus == EmpActive
	}
	if err := s.repo.Update(u); err != nil {
		return nil, err
	}
	return s.toResponse(storeID, u), nil
}

// PermissionsRequest updates POS permission flags.
type PermissionsRequest struct {
	CanRefund      *bool `json:"can_refund"`
	CanDiscount    *bool `json:"can_discount"`
	CanViewReports *bool `json:"can_view_reports"`
}

// UpdatePermissions changes a staff member's POS permissions.
func (s *Service) UpdatePermissions(ctx context.Context, storeID, userID uuid.UUID, req *PermissionsRequest) (*StaffResponse, error) {
	u, err := s.repo.Get(storeID, userID)
	if err != nil {
		return nil, apierr.NotFound("Staff member not found")
	}
	if u.IsOwner() {
		// Owners/admins always keep elevated permissions.
		return s.toResponse(storeID, u), nil
	}
	if req.CanRefund != nil {
		u.CanRefund = *req.CanRefund
	}
	if req.CanDiscount != nil {
		u.CanDiscount = *req.CanDiscount
	}
	if req.CanViewReports != nil {
		u.CanViewReports = *req.CanViewReports
	}
	if err := s.repo.Update(u); err != nil {
		return nil, err
	}
	return s.toResponse(storeID, u), nil
}

// Deactivate disables a staff member.
func (s *Service) Deactivate(ctx context.Context, orgID, storeID, userID, actorID uuid.UUID) error {
	u, err := s.repo.Get(storeID, userID)
	if err != nil {
		return apierr.NotFound("Staff member not found")
	}
	if u.ID == actorID {
		return apierr.BadRequest("You cannot deactivate your own account")
	}
	return s.account.DeactivateUser(ctx, orgID, userID)
}

// StatsResult wraps a staff member's stats with identity context.
type StatsResult struct {
	UserID           uuid.UUID   `json:"user_id"`
	FullName         string      `json:"full_name"`
	Role             string      `json:"role"`
	EmploymentStatus string      `json:"employment_status"`
	Shift            string      `json:"shift"`
	ShiftDisplay     string      `json:"shift_display"`
	Stats            *StaffStats `json:"stats"`
}

// Stats returns a staff member's performance.
func (s *Service) Stats(ctx context.Context, storeID, userID uuid.UUID) (*StatsResult, error) {
	u, err := s.repo.Get(storeID, userID)
	if err != nil {
		return nil, apierr.NotFound("Staff member not found")
	}
	now := time.Now()
	st, err := s.repo.Stats(storeID, userID, util.StartOfDay(now), util.StartOfMonth(now))
	if err != nil {
		return nil, err
	}
	return &StatsResult{
		UserID:           u.ID,
		FullName:         u.FullName(),
		Role:             u.Role,
		EmploymentStatus: u.EmploymentStatus,
		Shift:            u.Shift,
		ShiftDisplay:     ShiftDisplay(u.Shift),
		Stats:            st,
	}, nil
}

// KPIs returns the store-level staff summary.
func (s *Service) KPIs(ctx context.Context, storeID uuid.UUID) (*StoreKPIs, error) {
	now := time.Now()
	return s.repo.StoreKPIs(storeID, util.StartOfDay(now), util.StartOfMonth(now))
}

// Activity returns a cashier's transactions for a day.
func (s *Service) Activity(ctx context.Context, storeID, userID uuid.UUID, day time.Time) ([]ActivityItem, error) {
	if _, err := s.repo.Get(storeID, userID); err != nil {
		return nil, apierr.NotFound("Staff member not found")
	}
	return s.repo.Activity(storeID, userID, day, day.Add(24*time.Hour))
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func validPIN(pin string) bool {
	if len(pin) != 4 {
		return false
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizePhone(phone string) (string, error) {
	digits := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}
	switch {
	case len(digits) == 13 && strings.HasPrefix(digits, "255"):
		return "0" + digits[3:], nil
	case len(digits) == 12 && strings.HasPrefix(digits, "255"):
		return "0" + digits[3:], nil
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return digits, nil
	case len(digits) == 9 && strings.HasPrefix(digits, "7"):
		return "0" + digits, nil
	}
	return "", errors.New("invalid phone number")
}
