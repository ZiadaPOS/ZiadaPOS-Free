package accounts

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/util"
)

// AccountService handles /accounts/* endpoints.
type AccountService struct {
	repo  *Repo
	cache *cache.Cache
	bus   *events.Bus
	cfg   *config.Config
}

// NewAccountService wires an AccountService.
func NewAccountService(repo *Repo, cache *cache.Cache, bus *events.Bus, cfg *config.Config) *AccountService {
	return &AccountService{repo: repo, cache: cache, bus: bus, cfg: cfg}
}

// UserResponse is the public shape of a user.
type UserResponse struct {
	ID               uuid.UUID  `json:"id"`
	Phone            string     `json:"phone"`
	Email            string     `json:"email,omitempty"`
	FirstName        string     `json:"first_name"`
	LastName         string     `json:"last_name"`
	FullName         string     `json:"full_name"`
	Initials         string     `json:"initials"`
	Occupation       string     `json:"occupation"`
	Role             string     `json:"role"`
	OrganisationID   *uuid.UUID `json:"organisation_id,omitempty"`
	StoreID          *uuid.UUID `json:"store_id,omitempty"`
	AvatarHue        int        `json:"avatar_hue"`
	Shift            string     `json:"shift"`
	EmploymentStatus string     `json:"employment_status"`
	CanRefund        bool       `json:"can_refund"`
	CanDiscount      bool       `json:"can_discount"`
	CanViewReports   bool       `json:"can_view_reports"`
	IsPhoneVerified  bool       `json:"is_phone_verified"`
	IsEmailVerified  bool       `json:"is_email_verified"`
	IsActive         bool       `json:"is_active"`
	LastLogin        *time.Time `json:"last_login,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ToUserResponse builds the public representation of a user. Exported so
// other apps (staff, reports) can embed the shape in their responses.
func ToUserResponse(u *User) UserResponse {
	return buildUserResponse(u)
}

func toUserResponse(u *User) UserResponse {
	return buildUserResponse(u)
}

// OptionalEmail normalises a raw email string to a nullable pointer. Empty
// strings become nil so the unique index does not collide across users that
// have no email address (common for cashiers).
func OptionalEmail(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// EmailString safely dereferences a nullable email for display.
func EmailString(e *string) string {
	if e == nil {
		return ""
	}
	return *e
}

func buildUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:               u.ID,
		Phone:            u.Phone,
		Email:            EmailString(u.Email),
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		FullName:         u.FullName(),
		Initials:         u.Initials(),
		Occupation:       u.Occupation,
		Role:             u.Role,
		OrganisationID:   u.OrganisationID,
		StoreID:          u.StoreID,
		AvatarHue:        u.AvatarHue,
		Shift:            u.Shift,
		EmploymentStatus: u.EmploymentStatus,
		CanRefund:        u.CanRefund,
		CanDiscount:      u.CanDiscount,
		CanViewReports:   u.CanViewReports,
		IsPhoneVerified:  u.IsPhoneVerified,
		IsEmailVerified:  u.IsEmailVerified,
		IsActive:         u.IsActive,
		LastLogin:        u.LastLogin,
		CreatedAt:        u.CreatedAt,
	}
}

// OrgResponse is the public shape of an organisation.
type OrgResponse struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	BusinessType  string          `json:"business_type"`
	Occupation    string          `json:"occupation"`
	Region        string          `json:"region"`
	District      string          `json:"district"`
	Ward          string          `json:"ward"`
	Street        string          `json:"street"`
	Address       string          `json:"address"`
	Latitude      float64         `json:"latitude"`
	Longitude     float64         `json:"longitude"`
	Phone         string          `json:"phone"`
	Email         string          `json:"email"`
	Logo          string          `json:"logo"`
	TIN           string          `json:"tin"`
	Currency      string          `json:"currency"`
	Plan          string          `json:"plan"`
	TrialEndsAt   *time.Time      `json:"trial_ends_at"`
	MaxStores     int             `json:"max_stores"`
	CanAddStore   bool            `json:"can_add_store"`
	AIUsed        int             `json:"ai_credits_used"`
	AIAllocated   int             `json:"ai_credits_allocated"`
	SMSSenderID   string          `json:"sms_sender_id"`
	SMSConfigured bool            `json:"sms_configured"`
	NGAMIASetup   bool            `json:"ngamia_configured"`
	Stores        []StoreResponse `json:"stores,omitempty"`
}

// StoreResponse is the public shape of a store.
type StoreResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Phone       string    `json:"phone"`
	Email       string    `json:"email"`
	Area        string    `json:"area"`
	Address     string    `json:"address"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	OpenHours   string    `json:"open_hours"`
	Color       string    `json:"color"`
	IsMainStore bool      `json:"is_main_store"`
	Status      string    `json:"status"`
	IsActive    bool      `json:"is_active"`
	VATEnabled  bool      `json:"vat_enabled"`
}

func toStoreResponse(s *Store) StoreResponse {
	return StoreResponse{
		ID:          s.ID,
		Name:        s.Name,
		Code:        s.Code,
		Phone:       s.Phone,
		Email:       s.Email,
		Area:        s.Area,
		Address:     s.Address,
		Latitude:    s.Latitude,
		Longitude:   s.Longitude,
		OpenHours:   s.OpenHours,
		Color:       s.Color,
		IsMainStore: s.IsMainStore,
		Status:      s.Status,
		IsActive:    s.IsActive,
		VATEnabled:  s.VATEnabled,
	}
}

// ProfileContext bundles the authenticated user's full context.
type ProfileContext struct {
	User         UserResponse    `json:"user"`
	Organisation *OrgResponse    `json:"organisation,omitempty"`
	Store        *StoreResponse  `json:"store,omitempty"`
	AllStores    []StoreResponse `json:"stores,omitempty"`
}

// Me returns the current user's profile with organisation + store context.
func (s *AccountService) Me(ctx context.Context, userID uuid.UUID) (*ProfileContext, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, apierr.NotFound("User not found")
	}

	out := &ProfileContext{User: toUserResponse(user)}

	var orgID uuid.UUID
	if user.OrganisationID != nil {
		orgID = *user.OrganisationID
	} else if user.StoreID != nil {
		if st, err := s.repo.GetStoreByID(*user.StoreID); err == nil {
			orgID = st.OrganisationID
		}
	}

	if orgID != uuid.Nil {
		org, err := s.repo.GetOrgByID(orgID)
		if err != nil {
			return out, nil
		}
		stores, _ := s.repo.ListStoresByOrg(orgID)
		storeList := make([]StoreResponse, 0, len(stores))
		for i := range stores {
			storeList = append(storeList, toStoreResponse(&stores[i]))
		}

		now := time.Now()
		ac, _ := s.repo.GetAICredit(orgID, now.Year(), int(now.Month()))
		aiUsed, aiAlloc := 0, org.AICreditsMonthly
		if ac != nil {
			aiUsed, aiAlloc = ac.Used, ac.Allocated
		}

		storeCount, _ := s.repo.CountStoresByOrg(orgID)
		orgResp := &OrgResponse{
			ID:            org.ID,
			Name:          org.Name,
			Slug:          org.Slug,
			BusinessType:  org.BusinessType,
			Occupation:    org.Occupation,
			Region:        org.Region,
			District:      org.District,
			Ward:          org.Ward,
			Street:        org.Street,
			Address:       org.Address,
			Latitude:      org.Latitude,
			Longitude:     org.Longitude,
			Phone:         org.Phone,
			Email:         org.Email,
			Logo:          org.Logo,
			TIN:           org.TIN,
			Currency:      org.Currency,
			Plan:          org.Plan,
			TrialEndsAt:   org.TrialEndsAt,
			MaxStores:     org.MaxStores,
			CanAddStore:   storeCount < int64(org.MaxStores),
			AIUsed:        aiUsed,
			AIAllocated:   aiAlloc,
			SMSSenderID:   org.SMSSenderID,
			SMSConfigured: org.SendAfricaAPIKey != "",
			NGAMIASetup:   org.NgamiaAPIKey != "",
			Stores:        storeList,
		}
		out.Organisation = orgResp

		if user.StoreID != nil {
			for i := range storeList {
				if storeList[i].ID == *user.StoreID {
					store := storeList[i]
					out.Store = &store
					break
				}
			}
		}
	}
	return out, nil
}

// UpdateProfileRequest updates the current user.
type UpdateProfileRequest struct {
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	Email      *string `json:"email"`
	Occupation *string `json:"occupation"`
	Password   *string `json:"password"`
}

// UpdateProfile applies editable profile fields.
func (s *AccountService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) (*UserResponse, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, apierr.NotFound("User not found")
	}
	if req.FirstName != nil {
		user.FirstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		user.LastName = strings.TrimSpace(*req.LastName)
	}
	if req.Occupation != nil && *req.Occupation != "" {
		if !ValidOccupation(*req.Occupation) {
			return nil, apierr.BadRequest("Unknown occupation").WithField("occupation", "unsupported value")
		}
		user.Occupation = *req.Occupation
	}
	if req.Email != nil && strings.ToLower(strings.TrimSpace(*req.Email)) != strings.ToLower(EmailString(user.Email)) {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email != "" {
			if existing, _ := s.repo.GetUserByEmail(email); existing != nil && existing.ID != user.ID {
				return nil, apierr.Conflict("Email already in use")
			}
		}
		user.Email = OptionalEmail(email)
		user.IsEmailVerified = false
	}
	if req.Password != nil && *req.Password != "" {
		if len(*req.Password) < 6 {
			return nil, apierr.BadRequest("Password must be at least 6 characters").WithField("password", "too short")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}
	if err := s.repo.UpdateUser(user); err != nil {
		if isUniqueViolation(err) {
			return nil, apierr.Conflict("Email already in use")
		}
		return nil, err
	}
	out := toUserResponse(user)
	return &out, nil
}

// UpdateOrganisationRequest updates org settings including integrations.
type UpdateOrganisationRequest struct {
	Name          *string  `json:"name"`
	LegalName     *string  `json:"legal_name"`
	TIN           *string  `json:"tin"`
	BusinessType  *string  `json:"business_type"`
	Occupation    *string  `json:"occupation"`
	Region        *string  `json:"region"`
	District      *string  `json:"district"`
	Ward          *string  `json:"ward"`
	Street        *string  `json:"street"`
	Address       *string  `json:"address"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	Phone         *string  `json:"phone"`
	Email         *string  `json:"email"`
	Logo          *string  `json:"logo"`
	SendAfricaKey *string  `json:"sendafrica_api_key"`
	SMSSenderID   *string  `json:"sms_sender_id"`
	NgamiaAPIKey  *string  `json:"ngamia_api_key"`
}

// UpdateOrganisation persists org settings.
func (s *AccountService) UpdateOrganisation(ctx context.Context, orgID uuid.UUID, req *UpdateOrganisationRequest) (*OrgResponse, error) {
	org, err := s.repo.GetOrgByID(orgID)
	if err != nil {
		return nil, apierr.NotFound("Organisation not found")
	}
	if req.Name != nil {
		org.Name = strings.TrimSpace(*req.Name)
	}
	if req.LegalName != nil {
		org.LegalName = *req.LegalName
	}
	if req.TIN != nil {
		org.TIN = *req.TIN
	}
	if req.BusinessType != nil && *req.BusinessType != "" {
		if !ValidBusinessType(*req.BusinessType) {
			return nil, apierr.BadRequest("Unknown business type").WithField("business_type", "unsupported")
		}
		org.BusinessType = *req.BusinessType
	}
	if req.Occupation != nil && *req.Occupation != "" {
		if !ValidOccupation(*req.Occupation) {
			return nil, apierr.BadRequest("Unknown occupation").WithField("occupation", "unsupported")
		}
		org.Occupation = *req.Occupation
	}
	if req.Region != nil && *req.Region != "" {
		if !ValidRegion(*req.Region) {
			return nil, apierr.BadRequest("Unknown region").WithField("region", "unsupported")
		}
		org.Region = *req.Region
	}
	if req.District != nil {
		org.District = *req.District
	}
	if req.Ward != nil {
		org.Ward = *req.Ward
	}
	if req.Street != nil {
		org.Street = *req.Street
	}
	if req.Address != nil {
		org.Address = *req.Address
	}
	if req.Latitude != nil {
		org.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		org.Longitude = *req.Longitude
	}
	if req.Phone != nil {
		org.Phone = *req.Phone
	}
	if req.Email != nil {
		org.Email = *req.Email
	}
	if req.Logo != nil {
		org.Logo = *req.Logo
	}
	if req.SendAfricaKey != nil {
		org.SendAfricaAPIKey = strings.TrimSpace(*req.SendAfricaKey)
	}
	if req.SMSSenderID != nil {
		org.SMSSenderID = *req.SMSSenderID
	}
	if req.NgamiaAPIKey != nil {
		org.NgamiaAPIKey = strings.TrimSpace(*req.NgamiaAPIKey)
	}
	if err := s.repo.UpdateOrg(org); err != nil {
		return nil, err
	}
	_ = s.cacheDelete("accounts:org:" + orgID.String())
	return s.orgResponse(orgID)
}

// SwitchStore updates the user's active store.
func (s *AccountService) SwitchStore(ctx context.Context, userID uuid.UUID, storeID uuid.UUID) (*StoreResponse, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, apierr.NotFound("User not found")
	}
	var orgID uuid.UUID
	if user.OrganisationID != nil {
		orgID = *user.OrganisationID
	}
	if orgID == uuid.Nil && user.StoreID != nil {
		if st, err := s.repo.GetStoreByID(*user.StoreID); err == nil {
			orgID = st.OrganisationID
		}
	}
	store, err := s.repo.GetStoreByIDAndOrg(storeID, orgID)
	if err != nil {
		return nil, apierr.NotFound("Store not found or not part of your organisation")
	}
	user.StoreID = &store.ID
	if err := s.repo.UpdateUser(user); err != nil {
		return nil, err
	}
	out := toStoreResponse(store)
	return &out, nil
}

// CreateStaffRequest adds a staff member.
type CreateStaffRequest struct {
	FirstName      string     `json:"first_name" binding:"required"`
	LastName       string     `json:"last_name"`
	Phone          string     `json:"phone" binding:"required"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	StoreID        *uuid.UUID `json:"store_id"`
	CanRefund      bool       `json:"can_refund"`
	CanDiscount    bool       `json:"can_discount"`
	CanViewReports bool       `json:"can_view_reports"`
}

// CreateStaff adds a staff user to the organisation.
func (s *AccountService) CreateStaff(ctx context.Context, orgID uuid.UUID, req *CreateStaffRequest) (*UserResponse, error) {
	phone, err := sms.NormalizePhone(req.Phone)
	if err != nil {
		return nil, apierr.BadRequest("Invalid phone number").WithField("phone", "must be a valid Tanzanian number")
	}
	role := req.Role
	if role == "" {
		role = RoleStaff
	}
	if role != RoleStaff && role != RoleOwner {
		return nil, apierr.BadRequest("Invalid role").WithField("role", "must be staff or owner")
	}
	if req.StoreID != nil {
		if _, err := s.repo.GetStoreByIDAndOrg(*req.StoreID, orgID); err != nil {
			return nil, apierr.BadRequest("Store not found").WithField("store_id", "not part of your organisation")
		}
	}
	// default password
	pw, err := util.RandomDigits(8)
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &User{
		Phone:            phone,
		Email:            OptionalEmail(req.Email),
		FirstName:        strings.TrimSpace(req.FirstName),
		LastName:         strings.TrimSpace(req.LastName),
		Role:             role,
		OrganisationID:   &orgID,
		StoreID:          req.StoreID,
		PasswordHash:     string(hash),
		IsActive:         true,
		CanRefund:        req.CanRefund,
		CanDiscount:      req.CanDiscount,
		CanViewReports:   req.CanViewReports,
		EmploymentStatus: "active",
	}
	if err := s.repo.CreateUser(user); err != nil {
		if isUniqueViolation(err) {
			return nil, apierr.Conflict("A user with this phone or email already exists")
		}
		return nil, err
	}
	s.bus.Publish("staff.created", map[string]interface{}{"user_id": user.ID, "org_id": orgID, "temp_password": pw})
	out := toUserResponse(user)
	return &out, nil
}

// ListStaff returns all users in an organisation.
func (s *AccountService) ListStaff(ctx context.Context, orgID uuid.UUID) ([]UserResponse, error) {
	users, err := s.repo.ListUsersByOrg(orgID)
	if err != nil {
		return nil, err
	}
	out := make([]UserResponse, 0, len(users))
	for i := range users {
		out = append(out, toUserResponse(&users[i]))
	}
	return out, nil
}

// DeactivateUser soft-deactivates a staff member.
func (s *AccountService) DeactivateUser(ctx context.Context, orgID, targetID uuid.UUID) error {
	user, err := s.repo.GetUserByID(targetID)
	if err != nil {
		return apierr.NotFound("User not found")
	}
	if user.OrganisationID == nil || *user.OrganisationID != orgID {
		return apierr.NotFound("User not found")
	}
	if user.Role == RoleAdmin {
		return apierr.Forbidden("Cannot deactivate a platform admin")
	}
	user.IsActive = false
	user.EmploymentStatus = "inactive"
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}
	_ = s.cacheDelete("accounts:org:" + orgID.String())
	return nil
}

// ActivateUser re-enables a staff member.
func (s *AccountService) ActivateUser(ctx context.Context, orgID, targetID uuid.UUID) error {
	user, err := s.repo.GetUserByID(targetID)
	if err != nil {
		return apierr.NotFound("User not found")
	}
	if user.OrganisationID == nil || *user.OrganisationID != orgID {
		return apierr.NotFound("User not found")
	}
	user.IsActive = true
	user.EmploymentStatus = "active"
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}
	return nil
}

// GetAICredits returns the org's monthly AI credit usage.
func (s *AccountService) GetAICredits(ctx context.Context, orgID uuid.UUID) (map[string]interface{}, error) {
	org, err := s.repo.GetOrgByID(orgID)
	if err != nil {
		return nil, apierr.NotFound("Organisation not found")
	}
	now := time.Now()
	ac, _ := s.repo.GetAICredit(orgID, now.Year(), int(now.Month()))
	used, alloc := 0, org.AICreditsMonthly
	if ac != nil {
		used, alloc = ac.Used, ac.Allocated
	}
	return map[string]interface{}{
		"used":       used,
		"allocated":  alloc,
		"remaining":  max(0, alloc-used),
		"month":      now.Format("January"),
		"percentage": percentage(used, alloc),
	}, nil
}

func (s *AccountService) orgResponse(orgID uuid.UUID) (*OrgResponse, error) {
	org, err := s.repo.GetOrgByID(orgID)
	if err != nil {
		return nil, err
	}
	stores, _ := s.repo.ListStoresByOrg(orgID)
	storeList := make([]StoreResponse, 0, len(stores))
	for i := range stores {
		storeList = append(storeList, toStoreResponse(&stores[i]))
	}
	now := time.Now()
	ac, _ := s.repo.GetAICredit(orgID, now.Year(), int(now.Month()))
	aiUsed, aiAlloc := 0, org.AICreditsMonthly
	if ac != nil {
		aiUsed, aiAlloc = ac.Used, ac.Allocated
	}
	storeCount, _ := s.repo.CountStoresByOrg(orgID)
	return &OrgResponse{
		ID: org.ID, Name: org.Name, Slug: org.Slug,
		BusinessType: org.BusinessType, Occupation: org.Occupation,
		Region: org.Region, District: org.District, Ward: org.Ward,
		Street: org.Street, Address: org.Address,
		Latitude: org.Latitude, Longitude: org.Longitude,
		Phone: org.Phone, Email: org.Email, Logo: org.Logo, TIN: org.TIN,
		Currency: org.Currency, Plan: org.Plan, TrialEndsAt: org.TrialEndsAt,
		MaxStores: org.MaxStores, CanAddStore: storeCount < int64(org.MaxStores),
		AIUsed: aiUsed, AIAllocated: aiAlloc,
		SMSSenderID: org.SMSSenderID, SMSConfigured: org.SendAfricaAPIKey != "",
		NGAMIASetup: org.NgamiaAPIKey != "",
		Stores:      storeList,
	}, nil
}

func (s *AccountService) cacheDelete(key string) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Del(context.Background(), key)
}

func percentage(used, alloc int) float64 {
	if alloc == 0 {
		return 0
	}
	return round1(float64(used) / float64(alloc) * 100)
}

func round1(v float64) float64 {
	return float64(int(v*10)) / 10
}
