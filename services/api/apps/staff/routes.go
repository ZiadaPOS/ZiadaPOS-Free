package staff

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate is a no-op for staff: identity reuses accounts.users and stats are
// computed against the transactions ledger, so no new tables are needed.
func Migrate(db *gorm.DB) error { return nil }

// RegisterRoutes mounts the staff surface under /api/v1/staff. The whole
// group is manager-level (owner/admin).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	repo := NewRepo(deps.DB)
	svc := NewService(repo, deps.DB, deps.Cache, deps.Bus, deps.Cfg)
	h := NewHandler(svc)

	g := base.Group("/staff", middleware.Auth(deps.Tokens), middleware.RequireOwner())
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/kpis", h.KPIs)
		g.GET("/:id", h.Get)
		g.PATCH("/:id", h.Update)
		g.DELETE("/:id", h.Deactivate)
		g.GET("/:id/stats", h.Stats)
		g.GET("/:id/activity", h.Activity)
		g.PATCH("/:id/shift", h.UpdateShift)
		g.PATCH("/:id/permissions", h.UpdatePermissions)
	}
}
