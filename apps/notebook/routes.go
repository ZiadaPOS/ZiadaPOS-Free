package notebook

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the notes schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Note{})
}

// RegisterHandlers wires notebook event listeners (none yet).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {}

// RegisterRoutes wires the notebook router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB))
	h := NewHandler(svc)

	group := base.Group("/notebook", middleware.Auth(deps.Tokens))
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/:id", h.Get)
		group.PATCH("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}
