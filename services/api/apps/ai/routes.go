package ai

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the ai schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Conversation{}, &Message{}, &AICredit{})
}

// RegisterRoutes wires the ai router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.AI, deps.Cfg, deps.Secrets)
	h := NewHandler(svc)

	authed := base.Group("/ai", middleware.Auth(deps.Tokens))
	{
		authed.POST("/chat", h.StartChat)
		authed.POST("/conversations/:id/chat", h.ContinueChat)
		authed.GET("/conversations", h.List)
		authed.GET("/conversations/:id", h.Get)
		authed.PATCH("/conversations/:id", h.Update)
		authed.GET("/suggestions", h.Suggestions)
		authed.GET("/usage", h.Usage)
		authed.GET("/models", middleware.RequireOwner(), h.Models)
	}
}
