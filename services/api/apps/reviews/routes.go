package reviews

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the reviews schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&StoreReview{})
}

// RegisterHandlers wires review event listeners (none yet).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {}

// RegisterRoutes wires the reviews router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB))
	h := NewHandler(svc)

	base.GET("/reviews/public", h.Public)

	authed := base.Group("/reviews", middleware.Auth(deps.Tokens))
	{
		authed.POST("", h.Submit)
	}
}
