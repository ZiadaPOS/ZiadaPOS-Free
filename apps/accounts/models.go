package accounts

import (
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Roles used across the platform.
const (
	RoleAdmin = "admin"
	RoleOwner = "owner"
	RoleStaff = "staff"
)

// Plans.
const (
	PlanFree       = "free"
	PlanPro        = "pro"
	PlanEnterprise = "enterprise"
)

// Organisation is the top-level registered business. It owns one or more
// Stores. Registration captures the owner's job and the business GPS
// coordinates so we know where every shop is on the map of Tanzania.
type Organisation struct {
	models.BaseModel
	Name      string `gorm:"size:200;not null" json:"name"`
	Slug      string `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	LegalName string `gorm:"size:300" json:"legal_name"`
	TIN       string `gorm:"size:50" json:"tin"`

	// Business classification
	BusinessType string `gorm:"size:50;index" json:"business_type"`
	Occupation   string `gorm:"size:50" json:"occupation"` // owner's job

	// Tanzania location
	Region   string `gorm:"size:100;index" json:"region"`
	District string `gorm:"size:100" json:"district"`
	Ward     string `gorm:"size:100" json:"ward"`
	Street   string `gorm:"size:200" json:"street"`
	Address  string `gorm:"type:text" json:"address"`

	// GPS coordinates captured during registration
	Latitude  float64 `gorm:"type:double precision;index" json:"latitude"`
	Longitude float64 `gorm:"type:double precision;index" json:"longitude"`

	Country  string `gorm:"size:2;default:TZ" json:"country"`
	Currency string `gorm:"size:3;default:TZS" json:"currency"`
	Phone    string `gorm:"size:30" json:"phone"`
	Email    string `gorm:"size:255" json:"email"`
	Logo     string `gorm:"size:500" json:"logo"`

	// Integration keys (pasted by owner in Settings -> Integrations)
	SendAfricaAPIKey string `gorm:"size:200" json:"-"`
	SMSSenderID      string `gorm:"size:20" json:"-"`
	NgamiaAPIKey     string `gorm:"size:200" json:"-"`

	MaxStores        int        `gorm:"default:3" json:"max_stores"`
	AICreditsMonthly int        `gorm:"default:500" json:"ai_credits_monthly"`
	Plan             string     `gorm:"size:20;default:free;index" json:"plan"`
	TrialEndsAt      *time.Time `json:"trial_ends_at"`
	IsActive         bool       `gorm:"default:true;index" json:"is_active"`
	Settings         JSONMap    `gorm:"type:jsonb" json:"settings,omitempty"`

	Stores []Store `gorm:"constraint:OnDelete:CASCADE" json:"stores,omitempty"`
}

// JSONMap is a JSONB-typed map used for flexible settings columns.
type JSONMap map[string]interface{}

// Store is a physical shop location of an Organisation.
type Store struct {
	models.BaseModel
	OrganisationID uuid.UUID     `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Organisation   *Organisation `gorm:"constraint:OnDelete:CASCADE" json:"organisation,omitempty"`

	Name    string `gorm:"size:200;not null" json:"name"`
	Code    string `gorm:"size:10" json:"code"`
	Phone   string `gorm:"size:30" json:"phone"`
	Email   string `gorm:"size:255" json:"email"`
	Area    string `gorm:"size:100;index" json:"area"`
	Address string `gorm:"type:text" json:"address"`

	// GPS position of this branch
	Latitude  float64 `gorm:"type:double precision;index" json:"latitude"`
	Longitude float64 `gorm:"type:double precision;index" json:"longitude"`

	OpenHours   string `gorm:"size:50;default:7:00 AM – 9:00 PM" json:"open_hours"`
	Color       string `gorm:"size:20;default:#6366f1" json:"color"`
	IsMainStore bool   `gorm:"default:false" json:"is_main_store"`
	Status      string `gorm:"size:20;default:open;index" json:"status"`
	IsActive    bool   `gorm:"default:true;index" json:"is_active"`
	VATEnabled  bool   `gorm:"default:true" json:"vat_enabled"`

	Staff []User `gorm:"constraint:OnDelete:SET NULL" json:"staff,omitempty"`
}

// User is the custom auth principal. Login is phone (or email) + password.
type User struct {
	models.BaseModel
	Phone      string  `gorm:"size:10;uniqueIndex;not null" json:"phone"`
	Email      *string `gorm:"size:255;uniqueIndex" json:"email,omitempty"` // nil when unset so the unique index allows many emails-less staff
	FirstName  string  `gorm:"size:100" json:"first_name"`
	LastName   string  `gorm:"size:100" json:"last_name"`
	Occupation string  `gorm:"size:50" json:"occupation"` // job asked at registration
	Role       string  `gorm:"size:20;default:staff;index" json:"role"`

	OrganisationID *uuid.UUID    `gorm:"type:uuid;index" json:"organisation_id"`
	Organisation   *Organisation `gorm:"constraint:OnDelete:SET NULL" json:"-"`
	StoreID        *uuid.UUID    `gorm:"type:uuid;index" json:"store_id"`
	Store          *Store        `gorm:"constraint:OnDelete:SET NULL" json:"-"`

	PasswordHash string `gorm:"size:200;not null" json:"-"`

	AvatarHue        int    `gorm:"default:260" json:"avatar_hue"`
	Shift            string `gorm:"size:20;default:morning" json:"shift"`
	EmploymentStatus string `gorm:"size:20;default:active" json:"employment_status"`
	PIN              string `gorm:"size:4" json:"-"`
	CanRefund        bool   `gorm:"default:false" json:"can_refund"`
	CanDiscount      bool   `gorm:"default:true" json:"can_discount"`
	CanViewReports   bool   `gorm:"default:false" json:"can_view_reports"`

	IsPhoneVerified bool `gorm:"default:false" json:"is_phone_verified"`
	IsEmailVerified bool `gorm:"default:false" json:"is_email_verified"`
	IsActive        bool `gorm:"default:true;index" json:"is_active"`

	LastLogin        *time.Time `json:"last_login"`
	LastLoginIP      string     `gorm:"size:45" json:"-"`
	ResetToken       string     `gorm:"size:64;index" json:"-"`
	ResetTokenExpiry *time.Time `json:"-"`
}

func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Phone
	}
	return u.FirstName + " " + u.LastName
}

func (u *User) Initials() string {
	f := u.FirstName
	l := u.LastName
	if len(f) > 0 && len(l) > 0 {
		return string(f[0]) + string(l[0])
	}
	if len(f) > 0 {
		return string(f[0])
	}
	if len(u.Phone) >= 2 {
		return u.Phone[:2]
	}
	return "U"
}

func (u *User) IsOwner() bool { return u.Role == RoleOwner || u.Role == RoleAdmin }

// PhoneOTP verifies phone numbers during registration, login and resets.
type PhoneOTP struct {
	models.BaseModel
	Phone     string    `gorm:"size:15;index;not null" json:"phone"`
	OTP       string    `gorm:"size:6;not null" json:"-"`
	Purpose   string    `gorm:"size:20;index" json:"purpose"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	IsUsed    bool      `gorm:"default:false" json:"is_used"`
	Attempts  int       `gorm:"default:0" json:"attempts"`
}

// EmailVerificationToken links to a user's email confirmation link.
type EmailVerificationToken struct {
	models.BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Token     string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
}

// AICredit tracks monthly AI credit usage per organisation.
type AICredit struct {
	models.BaseModel
	OrganisationID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_org_month" json:"organisation_id"`
	Year           int       `gorm:"uniqueIndex:idx_org_month" json:"year"`
	Month          int       `gorm:"uniqueIndex:idx_org_month" json:"month"`
	Used           int       `gorm:"default:0" json:"used"`
	Allocated      int       `gorm:"default:5000" json:"allocated"`
}

func (a *AICredit) Remaining() int { return max(0, a.Allocated-a.Used) }
