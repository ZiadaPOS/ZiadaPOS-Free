package stores

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates store tables (managed by the accounts app).
func Migrate(db *gorm.DB) error { return nil }

// RegisterHandlers wires store event listeners (none yet).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {}

// RegisterRoutes wires the stores router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB))
	h := NewHandler(svc)

	group := base.Group("/stores", middleware.Auth(deps.Tokens), middleware.RequireOwner())
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/stats", h.Stats)
		group.GET("/:id", h.Get)
		group.PATCH("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
		group.GET("/:id/staff", h.Staff)
		group.GET("/:id/week", h.Week)
		group.PATCH("/:id/status", h.SetStatus)
	}
}
