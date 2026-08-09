package notifications

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the notifications schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Notification{})
}

// RegisterRoutes wires the notifications router (all endpoints require auth).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	repo := NewRepo(deps.DB)
	svc := NewService(repo, deps.Bus)
	h := NewHandler(svc)

	notifs := base.Group("/notifications", middleware.Auth(deps.Tokens))
	{
		notifs.GET("", h.List)
		notifs.GET("/unread-count", h.UnreadCount)
		notifs.POST("", h.Create)
		notifs.PATCH("/read-all", h.MarkAllRead)
		notifs.PATCH("/:id/read", h.MarkRead)
		notifs.DELETE("/:id", h.Delete)
	}
}

// RegisterHandlers subscribes notifications to domain events.
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), bus)
	svc.SubscribeHandlers(bus)
}
