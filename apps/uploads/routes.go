package uploads

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate has no tables (files live in object storage) but exists so every app
// keeps a consistent Migrate/Register contract.
func Migrate(_ *gorm.DB) error { return nil }

// RegisterRoutes wires the uploads router (requires auth).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	up := base.Group("/uploads", middleware.Auth(deps.Tokens))
	{
		up.POST("", func(c *gin.Context) { Upload(c, deps.Storage) })
		up.DELETE("", func(c *gin.Context) { Delete(c, deps.Storage) })
	}
}
