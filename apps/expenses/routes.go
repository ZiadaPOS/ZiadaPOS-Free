package expenses

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the expenses schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Expense{}, &ExpenseCategory{})
}

// RegisterRoutes wires the expenses router (all endpoints require auth).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.Cache, deps.Bus)
	h := NewHandler(svc)

	exp := base.Group("/expenses", middleware.Auth(deps.Tokens))
	{
		exp.GET("/categories", h.Categories)
		exp.POST("/categories", h.CreateCategory)
		exp.PATCH("/categories/:id", h.UpdateCategory)
		exp.DELETE("/categories/:id", h.DeleteCategory)
		exp.GET("", h.List)
		exp.POST("", h.Create)
		exp.GET("/summary", h.Summary)
		exp.GET("/:id", h.Get)
		exp.PATCH("/:id", h.Update)
		exp.DELETE("/:id", h.Delete)
	}
}
