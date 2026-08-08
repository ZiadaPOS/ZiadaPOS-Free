package inventory

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the inventory schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Category{},
		&Product{},
		&StockAdjustment{},
	)
}

// RegisterRoutes wires the inventory router (all endpoints require auth).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	repo := NewRepo(deps.DB)
	svc := NewService(repo, deps.Cache, deps.Bus)
	h := NewHandler(svc)

	inv := base.Group("/inventory", middleware.Auth(deps.Tokens))
	{
		inv.GET("/categories", h.ListCategories)
		inv.POST("/categories", h.CreateCategory)
		inv.PATCH("/categories/:id", h.UpdateCategory)
		inv.DELETE("/categories/:id", h.DeleteCategory)

		inv.GET("/products", h.ListProducts)
		inv.POST("/products", h.CreateProduct)
		inv.GET("/products/:id", h.GetProduct)
		inv.PATCH("/products/:id", h.UpdateProduct)
		inv.DELETE("/products/:id", h.DeleteProduct)
		inv.GET("/products/barcode/:code", h.ByBarcode)
		inv.POST("/products/:id/adjust", h.AdjustStock)
		inv.GET("/products/:id/adjustments", h.Adjustments)

		inv.GET("/low-stock", h.LowStock)
	}
}
