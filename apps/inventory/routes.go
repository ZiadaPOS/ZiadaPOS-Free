package inventory

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the inventory schema.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&Category{},
		&Unit{},
		&Product{},
		&StockAdjustment{},
		&MedicineBatch{},
		&MedicineBatchAllocation{},
		&MedicineStockMovement{},
	); err != nil {
		return err
	}
	// Existing wholesale catalogues retain their current price as the starting
	// wholesale price; owners can then set tier pricing without losing sales.
	return db.Exec(`UPDATE products SET wholesale_price = products.selling_price FROM stores JOIN organisations ON organisations.id = stores.organisation_id WHERE products.store_id = stores.id AND organisations.business_type IN ('wholesale', 'retail_wholesale') AND products.wholesale_price IS NULL`).Error
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
		inv.GET("/units", h.ListUnits)
		inv.POST("/units", h.CreateUnit)
		inv.PATCH("/units/:id", h.UpdateUnit)
		inv.DELETE("/units/:id", h.DeleteUnit)

		inv.GET("/products", h.ListProducts)
		inv.POST("/products", h.CreateProduct)
		inv.POST("/import", h.Import)
		inv.GET("/products/:id", h.GetProduct)
		inv.PATCH("/products/:id", h.UpdateProduct)
		inv.DELETE("/products/:id", h.DeleteProduct)
		inv.GET("/products/barcode/:code", h.ByBarcode)
		inv.POST("/products/:id/adjust", h.AdjustStock)
		inv.GET("/products/:id/adjustments", h.Adjustments)
		inv.GET("/batches", h.ListBatches)
		inv.POST("/products/:id/batches", h.ReceiveBatch)

		inv.GET("/low-stock", h.LowStock)
	}
}
