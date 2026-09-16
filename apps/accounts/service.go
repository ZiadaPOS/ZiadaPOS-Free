// Package accounts auth service: registration, OTP verification, login,
// refresh and password flows. All OTP SMS goes through SendAfrica and emails
// through the shared mailer; triggers are published on the event bus.
package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/mailer"
	"github.com/troubleman96/ziadapos/internal/security"
	"github.com/troubleman96/ziadapos/internal/util"
)

const (
	otpPurposeRegistration = "registration"
	otpPurposeLogin        = "login"
	otpPurposeReset        = "reset"

	otpTTL        = 5 * time.Minute
	otpMaxAttempt = 5
	otpCooldown   = 45 * time.Second
	otpPerHour    = 5

	lockoutAttempts = 5
	lockoutWindow   = 15 * time.Minute

	trialDays = 7
)

// AuthService implements the auth flows.
type AuthService struct {
	repo   *Repo
	cache  *cache.Cache
	mail   *mailer.Mailer
	sms    *sms.Client
	tokens *security.Manager
	bus    *events.Bus
	cfg    *config.Config
}

// NewAuthService wires an AuthService.
func NewAuthService(repo *Repo, cache *cache.Cache, mail *mailer.Mailer, sms *sms.Client, tokens *security.Manager, bus *events.Bus, cfg *config.Config) *AuthService {
	return &AuthService{repo: repo, cache: cache, mail: mail, tokens: tokens, bus: bus, sms: sms, cfg: cfg}
}

// ── DTOs ────────────────────────────────────────────────────────────────────

// ShopInput captures the shop + GPS location during registration.
type ShopInput struct {
	Name         string  `json:"name" binding:"required"`
	BusinessType string  `json:"business_type" binding:"required"`
	Region       string  `json:"region" binding:"required"`
	District     string  `json:"district"`
	Ward         string  `json:"ward"`
	Street       string  `json:"street"`
	Address      string  `json:"address"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

// RegisterRequest is the owner registration payload.
type RegisterRequest struct {
	FirstName       string    `json:"first_name" binding:"required"`
	LastName        string    `json:"last_name"`
	Phone           string    `json:"phone" binding:"required"`
	Email           string    `json:"email"`
	Password        string    `json:"password" binding:"required,min=6"`
	ConfirmPassword string    `json:"confirm_password" binding:"required"`
	Occupation      string    `json:"occupation" binding:"required"` // the user's job
	Shop            ShopInput `json:"shop" binding:"required"`
}

// RegisterResult is returned after a successful registration.
type RegisterResult struct {
	UserID            uuid.UUID `json:"user_id"`
	OrganisationID    uuid.UUID `json:"organisation_id"`
	StoreID           uuid.UUID `json:"store_id"`
	Status            string    `json:"status"`
	PendingVerifPhone bool      `json:"pending_phone_verification"`
	PendingVerifEmail bool      `json:"pending_email_verification"`
	TrialEndsAt       time.Time `json:"trial_ends_at"`
	Message           string    `json:"message"`
}

// OTPVerifyRequest verifies a code.
type OTPVerifyRequest struct {
	Phone   string `json:"phone" binding:"required"`
	Purpose string `json:"purpose" binding:"required"`
	Code    string `json:"code" binding:"required,len=6"`
}

// OTPRequest asks for a fresh code.
type OTPRequest struct {
	Phone   string `json:"phone" binding:"required"`
	Purpose string `json:"purpose" binding:"required"`
}

// LoginRequest logs a user in by phone/email + password.
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

// ── Registration ────────────────────────────────────────────────────────────

// Register creates the organisation, main store, owner user and trial
// subscription atomically, then fires welcome + verification triggers.
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResult, error) {
	if req.ConfirmPassword != req.Password {
		return nil, apierr.BadRequest("Passwords do not match").WithField("confirm_password", "must match password")
	}
	phone, err := validatePhone(req.Phone)
	if err != nil {
		return nil, apierr.BadRequest("Invalid Tanzanian phone number").WithField("phone", err.Error())
	}
	if !ValidOccupation(req.Occupation) {
		return nil, apierr.BadRequest("Unknown occupation").WithField("occupation", "must be one of the provided occupations")
	}
	if !ValidBusinessType(req.Shop.BusinessType) {
		return nil, apierr.BadRequest("Unknown business type").WithField("shop.business_type", "must be a supported shop type")
	}
	if !ValidRegion(req.Shop.Region) {
		return nil, apierr.BadRequest("Unknown region").WithField("shop.region", "must be one of the 31 Tanzania regions")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		email = ""
	}

	org := &Organisation{
		Name:         strings.TrimSpace(req.Shop.Name),
		Slug:         util.Slugify(req.Shop.Name) + "-" + shortID(),
		BusinessType: req.Shop.BusinessType,
		Occupation:   req.Occupation,
		Region:       req.Shop.Region,
		District:     strings.TrimSpace(req.Shop.District),
		Ward:         strings.TrimSpace(req.Shop.Ward),
		Street:       strings.TrimSpace(req.Shop.Street),
		Address:      strings.TrimSpace(req.Shop.Address),
		Latitude:     req.Shop.Latitude,
		Longitude:    req.Shop.Longitude,
		Phone:        phone,
		Email:        email,
		Plan:         PlanFree,
		TrialEndsAt:  ptr(time.Now().Add(trialDays * 24 * time.Hour)),
		IsActive:     true,
	}
	store := &Store{
		Name:        strings.TrimSpace(req.Shop.Name) + " — Main",
		Area:        req.Shop.Ward,
		Address:     req.Shop.Address,
		Latitude:    req.Shop.Latitude,
		Longitude:   req.Shop.Longitude,
		IsMainStore: true,
		Status:      "open",
		IsActive:    true,
	}

	user := &User{
		Phone:            phone,
		Email:            OptionalEmail(req.Email),
		FirstName:        strings.TrimSpace(req.FirstName),
		LastName:         strings.TrimSpace(req.LastName),
		Occupation:       req.Occupation,
		Role:             RoleOwner,
		PasswordHash:     string(hash),
		IsActive:         true,
		EmploymentStatus: "active",
	}

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}
		store.OrganisationID = org.ID
		if err := tx.Create(store).Error; err != nil {
			return err
		}
		user.OrganisationID = &org.ID
		user.StoreID = &store.ID
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, apierr.Conflict("An account with this phone or email already exists")
		}
		return nil, err
	}

	// Fire verification + welcome triggers (async).
	s.bus.Publish("user.registered", map[string]interface{}{
		"user_id": user.ID, "org_id": org.ID, "store_id": store.ID,
		"phone": phone, "email": email,
	})

	// Immediately issue a phone OTP for verification (best-effort).
	_, _ = s.generateAndSendOTP(ctx, phone, otpPurposeRegistration)

	return &RegisterResult{
		UserID:            user.ID,
		OrganisationID:    org.ID,
		StoreID:           store.ID,
		Status:            "active",
		PendingVerifPhone: !user.IsPhoneVerified,
		PendingVerifEmail: email != "" && !user.IsEmailVerified,
		TrialEndsAt:       *org.TrialEndsAt,
		Message:           fmt.Sprintf("Welcome to ZiadaPOS! Your %d-day free trial started today.", trialDays),
	}, nil
}

// ── OTP ─────────────────────────────────────────────────────────────────────

// SendOTP generates and delivers a 6-digit OTP for the given purpose.
func (s *AuthService) SendOTP(ctx context.Context, phone, purpose string) error {
	phone, err := validatePhone(phone)
	if err != nil {
		return apierr.BadRequest("Invalid Tanzanian phone number").WithField("phone", err.Error())
	}
	if purpose != otpPurposeRegistration && purpose != otpPurposeLogin && purpose != otpPurposeReset {
		return apierr.BadRequest("Unknown purpose").WithField("purpose", "must be registration, login or reset")
	}

	// Cooldown + hourly cap via Redis.
	if s.cache != nil {
		cooldownKey := "otp:cooldown:" + purpose + ":" + phone
		if v, _ := s.cache.Get(ctx, cooldownKey); v != "" {
			return apierr.RateLimited("Please wait before requesting another code")
		}
	}

	code, err := util.RandomDigits(6)
	if err != nil {
		return err
	}

	otp := &PhoneOTP{
		Phone:     phone,
		OTP:       code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(otpTTL),
	}
	if err := s.repo.CreateOTP(otp); err != nil {
		return err
	}

	if _, err := s.sms.Send(ctx, phone, "Your ZiadaPOS verification code is "+code+". It expires in 5 minutes.", ""); err != nil {
		return err
	}

	if s.cache != nil {
		_ = s.cache.SetTTL(ctx, "otp:cooldown:"+purpose+":"+phone, "1", otpCooldown)
		_, _ = s.cache.Incr(ctx, "otp:hour:"+purpose+":"+phone)
		_ = s.cache.Expire(ctx, "otp:hour:"+purpose+":"+phone, time.Hour)
	}
	return nil
}

// VerifyOTP validates a code and returns a token pair. Registration/login
// purposes also mark the phone verified.
func (s *AuthService) VerifyOTP(ctx context.Context, phone, purpose, code string) (*security.TokenPair, error) {
	phone, err := validatePhone(phone)
	if err != nil {
		return nil, apierr.BadRequest("Invalid phone number").WithField("phone", err.Error())
	}

	otp, err := s.repo.GetActiveOTP(phone, purpose)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierr.BadRequest("No active code for this phone").WithField("code", "request a new code")
		}
		return nil, err
	}
	if time.Now().After(otp.ExpiresAt) {
		return nil, apierr.BadRequest("Code has expired").WithField("code", "request a new code")
	}
	if otp.Attempts >= otpMaxAttempt {
		_ = s.repo.MarkOTPUsed(otp.ID)
		return nil, apierr.RateLimited("Too many wrong attempts, request a new code")
	}
	if otp.OTP != code {
		_ = s.repo.IncrementOTPAttempts(otp.ID)
		return nil, apierr.BadRequest("Incorrect code").WithField("code", "check the code and try again")
	}

	if err := s.repo.MarkOTPUsed(otp.ID); err != nil {
		return nil, err
	}

	// Mark the phone verified when it belongs to a registered user.
	user, err := s.repo.GetUserByPhone(phone)
	if err == nil && user != nil && !user.IsPhoneVerified {
		user.IsPhoneVerified = true
		_ = s.repo.UpdateUser(user)
		s.bus.Publish("user.phone_verified", map[string]interface{}{"user_id": user.ID, "phone": phone})
	}

	if purpose == otpPurposeLogin || purpose == otpPurposeReset {
		if user == nil {
			user, _ = s.repo.GetUserByPhone(phone)
		}
		if user == nil || !user.IsActive {
			return nil, apierr.Unauthorized("Account not found or inactive")
		}
		return s.issueTokens(user)
	}

	// registration purpose: log the user in
	if user != nil {
		return s.issueTokens(user)
	}
	return &security.TokenPair{}, nil
}

func (s *AuthService) issueTokens(u *User) (*security.TokenPair, error) {
	var orgID, storeID uuid.UUID
	if u.OrganisationID != nil {
		orgID = *u.OrganisationID
	}
	if u.StoreID != nil {
		storeID = *u.StoreID
	}
	return s.tokens.Issue(u.ID, u.Role, orgID, storeID)
}

// ── Login / refresh ─────────────────────────────────────────────────────────

// Login authenticates by identifier (phone or email) + password with Redis
// backed lockout after repeated failures.
func (s *AuthService) Login(ctx context.Context, identifier, password, ip string) (*security.TokenPair, *User, error) {
	identifier = strings.TrimSpace(identifier)

	if s.cache != nil {
		lockKey := "login:lock:" + strings.ToLower(identifier)
		attempts, _ := s.cache.Get(ctx, lockKey)
		if attempts != "" && attempts >= fmt.Sprint(lockoutAttempts) {
			return nil, nil, apierr.Locked("Account temporarily locked. Try again in 15 minutes or reset your password.")
		}
	}

	user, err := s.repo.GetUserByIdentifier(identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.recordFailedLogin(ctx, identifier)
			return nil, nil, apierr.Unauthorized("Invalid credentials")
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, apierr.Forbidden("Account is inactive")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.recordFailedLogin(ctx, identifier)
		return nil, nil, apierr.Unauthorized("Invalid credentials")
	}

	if s.cache != nil {
		_ = s.cache.Del(ctx, "login:lock:"+strings.ToLower(identifier))
	}

	now := time.Now()
	user.LastLogin = &now
	user.LastLoginIP = ip
	_ = s.repo.UpdateUser(user)

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}

	s.bus.Publish("user.logged_in", map[string]interface{}{"user_id": user.ID, "ip": ip})
	return tokens, user, nil
}

func (s *AuthService) recordFailedLogin(ctx context.Context, identifier string) {
	if s.cache == nil {
		return
	}
	key := "login:lock:" + strings.ToLower(identifier)
	n, _ := s.cache.Incr(ctx, key)
	if n == 1 {
		_ = s.cache.Expire(ctx, key, lockoutWindow)
	}
}

// Refresh validates a refresh token and issues a new pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*security.TokenPair, error) {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil, apierr.Unauthorized("Invalid or expired refresh token")
	}
	user, err := s.repo.GetUserByID(claims.UserID)
	if err != nil {
		return nil, apierr.Unauthorized("Account not found")
	}
	if !user.IsActive {
		return nil, apierr.Forbidden("Account is inactive")
	}
	return s.issueTokens(user)
}

// ── Email confirmation ──────────────────────────────────────────────────────

// ConfirmEmail validates a one-time email verification token.
func (s *AuthService) ConfirmEmail(ctx context.Context, token string) error {
	vt, err := s.repo.GetEmailToken(token)
	if err != nil {
		return apierr.BadRequest("Invalid or expired confirmation link")
	}
	user, err := s.repo.GetUserByID(vt.UserID)
	if err != nil {
		return apierr.NotFound("User not found")
	}
	user.IsEmailVerified = true
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}
	s.bus.Publish("user.email_verified", map[string]interface{}{"user_id": user.ID})
	return nil
}

// ResendConfirmation emails a fresh verification link.
func (s *AuthService) ResendConfirmation(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return apierr.NotFound("User not found")
	}
	if user.IsEmailVerified || user.Email == nil || *user.Email == "" {
		return apierr.BadRequest("Email already verified or not set")
	}
	return s.sendConfirmationEmail(user)
}

// PasswordForgot sends a reset OTP. Always succeeds to avoid leaking accounts.
func (s *AuthService) PasswordForgot(ctx context.Context, identifier string) error {
	_, err := s.repo.GetUserByIdentifier(strings.TrimSpace(identifier))
	if err != nil {
		return nil // never reveal whether the account exists
	}
	_ = s.SendOTP(ctx, identifier, otpPurposeReset)
	return nil
}

// PasswordReset verifies the reset OTP and sets a new password.
func (s *AuthService) PasswordReset(ctx context.Context, identifier, otp, newPassword string) error {
	user, err := s.repo.GetUserByIdentifier(strings.TrimSpace(identifier))
	if err != nil {
		return apierr.BadRequest("Account not found")
	}
	if len(newPassword) < 6 {
		return apierr.BadRequest("Password must be at least 6 characters").WithField("new_password", "too short")
	}
	if _, err := s.VerifyOTP(ctx, user.Phone, otpPurposeReset, otp); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}
	if s.cache != nil {
		_ = s.cache.Del(ctx, "login:lock:"+strings.ToLower(user.Phone))
		_ = s.cache.Del(ctx, "login:lock:"+strings.ToLower(EmailString(user.Email)))
	}
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (s *AuthService) generateAndSendOTP(ctx context.Context, phone, purpose string) (string, error) {
	code, err := util.RandomDigits(6)
	if err != nil {
		return "", err
	}
	otp := &PhoneOTP{Phone: phone, OTP: code, Purpose: purpose, ExpiresAt: time.Now().Add(otpTTL)}
	if err := s.repo.CreateOTP(otp); err != nil {
		return "", err
	}
	if _, err := s.sms.Send(ctx, phone, "Your ZiadaPOS verification code is "+code+". It expires in 5 minutes.", ""); err != nil {
		return "", err
	}
	return code, nil
}

func (s *AuthService) sendConfirmationEmail(user *User) error {
	token, err := util.RandomToken(32)
	if err != nil {
		return err
	}
	// One token per user (unique index on user_id) — replace any previous one.
	if err := s.repo.DeleteEmailTokenForUser(user.ID); err != nil {
		return err
	}
	vt := &EmailVerificationToken{UserID: user.ID, Token: token, ExpiresAt: time.Now().Add(72 * time.Hour)}
	if err := s.repo.CreateEmailToken(vt); err != nil {
		return err
	}
	link := fmt.Sprintf("%s/api/v1/auth/confirm-email/?token=%s", strings.TrimRight(s.cfg.SiteURL, "/"), token)
	shopName := ""
	if user.OrganisationID != nil {
		if org, err := s.repo.GetOrgByID(*user.OrganisationID); err == nil && org != nil {
			shopName = org.Name
		}
	}
	body := BuildWelcomeEmailBody(user.FullName(), shopName, link)
	return s.mail.Send(EmailString(user.Email), "Karibu ZiadaPOS — confirm your email & start selling", s.mail.Layout("Karibu ZiadaPOS", body))
}

// BuildWelcomeEmailBody renders the professional welcome + email-verification
// message. Pure function (no DB) so it can be previewed and tested.
func BuildWelcomeEmailBody(greeting, shopName, link string) string {
	return fmt.Sprintf(`
		<p>Karibu ZiadaPOS, <strong>%s</strong>!</p>
		<p>Your account%s has been created with a <strong>%d-day free trial</strong>. Confirm your email address below to activate it and start selling today.</p>
		<p><a href="%s" style="background:#4f46e5;color:#ffffff;padding:12px 20px;border-radius:8px;text-decoration:none;display:inline-block;">Confirm my email</a></p>
		<p>Or copy this link into your browser:<br><a href="%s">%s</a></p>
		<p>This link expires in 72 hours. Didn't create this account? Just ignore this email.</p>
		<hr style="border:none;border-top:1px solid #e5e7eb;margin:20px 0;">
		<p><strong>What you get with ZiadaPOS:</strong></p>
		<ul>
			<li>Fast POS sales — cash, M-Pesa, Tigo Pesa and bank, with receipts</li>
			<li>Stock control with low-stock alerts so you never run out</li>
			<li>Customer tabs (madeni) with balances, reminders and repayments</li>
			<li>Daily sales, profit and expense reports for your duka</li>
		</ul>
		<p><strong>Next steps:</strong></p>
		<ol>
			<li>Confirm your email using the button above</li>
			<li>Log in at <a href="https://app.ziadapos.com">app.ziadapos.com</a> with your phone number and password</li>
			<li>Add your products (or import them with CSV) and make your first sale</li>
		</ol>
		<p>Need help? Reply to this email or reach us at <a href="mailto:no-reply@ziadapos.com">no-reply@ziadapos.com</a>.</p>
		<p>Asante for choosing ZiadaPOS!<br><strong>The ZiadaPOS Team</strong></p>`,
		greeting, shopSuffix(shopName), trialDays, link, link, link)
}

// shopSuffix renders " for <shop>" when the shop name is known, else "".
func shopSuffix(shopName string) string {
	if strings.TrimSpace(shopName) == "" {
		return ""
	}
	return " for " + shopName
}

func validatePhone(phone string) (string, error) {
	n, err := sms.NormalizePhone(phone)
	if err != nil {
		return "", err
	}
	return n, nil
}

func shortID() string {
	return uuid.NewString()[:8]
}

func ptr(t time.Time) *time.Time { return &t }
