package accounts

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/response"
	"github.com/troubleman96/ziadapos/internal/server"
	"github.com/troubleman96/ziadapos/internal/uuidx"
)

// Migrate creates the accounts schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Organisation{},
		&Store{},
		&User{},
		&PhoneOTP{},
		&EmailVerificationToken{},
		&AICredit{},
	)
}

// RegisterHandlers subscribes accounts services to domain events (async
// email/SMS triggers).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {
	repo := NewRepo(deps.DB)

	// On registration: send the branded welcome email with a verification link.
	bus.On("user.registered", func(ctx context.Context, e events.Event) error {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		uid := uuidx.Parse(payload["user_id"])
		if uid == nil {
			return nil
		}
		user, err := repo.GetUserByID(*uid)
		if err != nil || user.Email == nil || *user.Email == "" {
			return err
		}
		auth := NewAuthService(repo, deps.Cache, deps.Mailer, deps.SMS, deps.Tokens, deps.Bus, deps.Cfg)
		return auth.sendConfirmationEmail(user)
	})

	// On staff creation: SMS the temporary password to the new staff member.
	bus.On("staff.created", func(ctx context.Context, e events.Event) error {
		payload, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		phone, _ := payload["phone"].(string)
		if phone == "" {
			if uid := uuidx.Parse(payload["user_id"]); uid != nil {
				if u, err := repo.GetUserByID(*uid); err == nil {
					phone = u.Phone
				}
			}
		}
		tempPW, _ := payload["temp_password"].(string)
		if phone == "" || tempPW == "" {
			return nil
		}
		_, err := deps.SMS.Send(ctx, phone, "Karibu ZiadaPOS! Your temporary login password is: "+tempPW+". Change it after your first login.", deps.Cfg.SendAfricaSenderID)
		return err
	})
}

// RegisterRoutes wires the auth + accounts routers.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	repo := NewRepo(deps.DB)

	authSvc := NewAuthService(repo, deps.Cache, deps.Mailer, deps.SMS, deps.Tokens, deps.Bus, deps.Cfg)
	accSvc := NewAccountService(repo, deps.Cache, deps.Bus, deps.Cfg, deps.Secrets)

	authH := NewAuthHandler(authSvc, deps.Cfg.SiteURL)
	accH := NewAccountHandler(accSvc, deps.SMS)

	auth := base.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/otp/send", authH.SendOTP)
		auth.POST("/otp/verify", authH.VerifyOTP)
		auth.POST("/login", authH.Login)
		auth.POST("/mfa/totp/verify", authH.VerifyAdminTOTP)
		auth.POST("/refresh", authH.Refresh)
		auth.GET("/confirm-email", authH.ConfirmEmail)
		auth.POST("/password/forgot", authH.PasswordForgot)
		auth.POST("/password/reset", authH.PasswordReset)
	}

	adminMFA := base.Group("/accounts/mfa/totp", middleware.Auth(deps.Tokens), middleware.RequireAdminMFA(deps.DB))
	{
		adminMFA.GET("/status", authH.AdminTOTPStatus)
		adminMFA.POST("/setup", authH.BeginAdminTOTPSetup)
		adminMFA.POST("/verify", authH.VerifyAdminTOTPSetup)
		adminMFA.POST("/disable", authH.DisableAdminTOTP)
	}

	authed := auth.Group("", middleware.Auth(deps.Tokens))
	{
		authed.POST("/logout", authH.Logout)
		authed.POST("/resend-confirmation", authH.ResendConfirmation)
	}

	accounts := base.Group("/accounts", middleware.Auth(deps.Tokens))
	{
		accounts.GET("/me", accH.Me)
		accounts.PATCH("/me", accH.UpdateMe)
		accounts.GET("/organisation", middleware.RequireOwner(), accH.Organisation)
		accounts.PATCH("/organisation", middleware.RequireOwner(), accH.UpdateOrganisation)
		accounts.POST("/switch-store", accH.SwitchStore)
		accounts.GET("/users", middleware.RequireOwner(), accH.Users)
		accounts.POST("/users", middleware.RequireOwner(), accH.CreateUser)
		accounts.DELETE("/users/:id", middleware.RequireOwner(), accH.DeactivateUser)
		accounts.GET("/ai-credits", accH.AICredits)
		accounts.GET("/sms-balance", middleware.RequireOwner(), accH.SmSBalance)
	}

	// Public reference data for onboarding UIs.
	meta := base.Group("/meta")
	{
		meta.GET("/business-types", func(c *gin.Context) { response.OK(c, BusinessTypes) })
		meta.GET("/regions", func(c *gin.Context) { response.OK(c, Regions) })
		meta.GET("/regions/:slug/districts", func(c *gin.Context) {
			d := RegionDistricts(c.Param("slug"))
			if d == nil {
				response.NotFound(c, "Region not found")
				return
			}
			response.OK(c, d)
		})
		meta.GET("/outlet-types", func(c *gin.Context) { response.OK(c, PharmacyOutletTypes) })
	}
}
