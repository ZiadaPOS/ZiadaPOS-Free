package accounts

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
)

// AuthHandler exposes auth HTTP endpoints.
type AuthHandler struct {
	svc     *AuthService
	siteURL string
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(svc *AuthService, siteURL string) *AuthHandler {
	return &AuthHandler{svc: svc, siteURL: siteURL}
}

// Register godoc: POST /auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid registration payload", nil)
		return
	}
	out, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Account created")
}

// SendOTP godoc: POST /auth/otp/send
func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	if err := h.svc.SendOTP(c.Request.Context(), req.Phone, req.Purpose); err != nil {
		writeErr(c, err)
		return
	}
	response.Message(c, http.StatusOK, "Code sent")
}

// VerifyOTP godoc: POST /auth/otp/verify
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req OTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	pair, err := h.svc.VerifyOTP(c.Request.Context(), req.Phone, req.Purpose, req.Code)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, pair)
}

// Login godoc: POST /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	pair, user, err := h.svc.Login(c.Request.Context(), req.Identifier, req.Password, c.ClientIP())
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
		"token_type":    pair.TokenType,
		"user":          toUserResponse(user),
	})
}

// Refresh godoc: POST /auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "refresh_token is required", nil)
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, pair)
}

// ConfirmEmail godoc: GET /auth/confirm-email/?token=
func (h *AuthHandler) ConfirmEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		response.BadRequest(c, "token is required", nil)
		return
	}
	if err := h.svc.ConfirmEmail(c.Request.Context(), token); err != nil {
		writeErr(c, err)
		return
	}
	c.Redirect(http.StatusFound, h.siteURL+"/login?verified=1")
}

// ResendConfirmation godoc: POST /auth/resend-confirmation
func (h *AuthHandler) ResendConfirmation(c *gin.Context) {
	uid, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	if err := h.svc.ResendConfirmation(c.Request.Context(), uid); err != nil {
		writeErr(c, err)
		return
	}
	response.Message(c, http.StatusOK, "Confirmation email sent")
}

// PasswordForgot godoc: POST /auth/password/forgot
func (h *AuthHandler) PasswordForgot(c *gin.Context) {
	var req struct {
		Identifier string `json:"identifier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "identifier is required", nil)
		return
	}
	_ = h.svc.PasswordForgot(c.Request.Context(), req.Identifier)
	response.Message(c, http.StatusOK, "If the account exists, a reset code has been sent")
}

// PasswordReset godoc: POST /auth/password/reset
func (h *AuthHandler) PasswordReset(c *gin.Context) {
	var req struct {
		Identifier      string `json:"identifier" binding:"required"`
		OTP             string `json:"otp" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		response.BadRequest(c, "Passwords do not match", nil)
		return
	}
	if err := h.svc.PasswordReset(c.Request.Context(), req.Identifier, req.OTP, req.NewPassword); err != nil {
		writeErr(c, err)
		return
	}
	response.Message(c, http.StatusOK, "Password reset successfully")
}

// Logout godoc: POST /auth/logout (stateless - client discards tokens)
func (h *AuthHandler) Logout(c *gin.Context) {
	response.Message(c, http.StatusOK, "Logged out")
}

// ── Account handlers ────────────────────────────────────────────────────────

// AccountHandler exposes /accounts endpoints.
type AccountHandler struct {
	svc *AccountService
	sms *sms.Client
}

// NewAccountHandler builds an AccountHandler.
func NewAccountHandler(svc *AccountService, sms *sms.Client) *AccountHandler { return &AccountHandler{svc: svc, sms: sms} }

// Me godoc: GET /accounts/me
func (h *AccountHandler) Me(c *gin.Context) {
	uid, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	profile, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, profile)
}

// UpdateMe godoc: PATCH /accounts/me
func (h *AccountHandler) UpdateMe(c *gin.Context) {
	uid, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.UpdateProfile(c.Request.Context(), uid, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Organisation godoc: GET /accounts/organisation
func (h *AccountHandler) Organisation(c *gin.Context) {
	uid, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	profile, err := h.svc.Me(c.Request.Context(), uid)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, profile.Organisation)
}

// UpdateOrganisation godoc: PATCH /accounts/organisation
func (h *AccountHandler) UpdateOrganisation(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	var req UpdateOrganisationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.UpdateOrganisation(c.Request.Context(), orgID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// SwitchStore godoc: POST /accounts/switch-store
func (h *AccountHandler) SwitchStore(c *gin.Context) {
	uid, err := uuid.Parse(middleware.UserID(c))
	if err != nil {
		response.Unauthorized(c, "Invalid session")
		return
	}
	var req struct {
		StoreID uuid.UUID `json:"store_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "store_id is required", nil)
		return
	}
	out, err := h.svc.SwitchStore(c.Request.Context(), uid, req.StoreID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// Users godoc: GET /accounts/users
func (h *AccountHandler) Users(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	users, err := h.svc.ListStaff(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, users)
}

// CreateUser godoc: POST /accounts/users
func (h *AccountHandler) CreateUser(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	var req CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request", nil)
		return
	}
	out, err := h.svc.CreateStaff(c.Request.Context(), orgID, &req)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.Created(c, out, "Staff member created")
}

// DeactivateUser godoc: DELETE /accounts/users/{id}
func (h *AccountHandler) DeactivateUser(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid user id", nil)
		return
	}
	if err := h.svc.DeactivateUser(c.Request.Context(), orgID, targetID); err != nil {
		writeErr(c, err)
		return
	}
	response.NoContent(c)
}

// SmSBalance godoc: GET /accounts/sms-balance — remaining SendAfrica credits.
func (h *AccountHandler) SmSBalance(c *gin.Context) {
	if h.sms == nil {
		response.OK(c, gin.H{"balance": 0, "configured": false})
		return
	}
	bal, err := h.sms.Balance(c.Request.Context())
	if err != nil {
		response.OK(c, gin.H{"balance": 0, "configured": false})
		return
	}
	response.OK(c, gin.H{"balance": bal, "configured": true})
}

// AICredits godoc: GET /accounts/ai-credits
func (h *AccountHandler) AICredits(c *gin.Context) {
	orgID, err := uuid.Parse(middleware.OrgID(c))
	if err != nil {
		response.NotFound(c, "Organisation not found")
		return
	}
	out, err := h.svc.GetAICredits(c.Request.Context(), orgID)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

// ── helpers ─────────────────────────────────────────────────────────────────

func writeErr(c *gin.Context, err error) {
	e := apierr.Translate(err)
	response.Error(c, e.Status, e.Message, e.Fields)
}
