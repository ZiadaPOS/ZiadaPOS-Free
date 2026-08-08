package suppliers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the suppliers schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Supplier{}, &SupplierDelivery{}, &SupplierPayment{})
}

// RegisterRoutes wires the suppliers router (all endpoints require auth).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.Cache, deps.Bus)
	h := NewHandler(svc)

	sup := base.Group("/suppliers", middleware.Auth(deps.Tokens))
	{
		sup.GET("", h.List)
		sup.POST("", h.Create)
		sup.GET("/stats", h.Stats)
		sup.GET("/deliveries", h.AllDeliveries)
		sup.POST("/deliveries", h.CreateDeliveryGlobal)
		sup.GET("/deliveries/:id", h.GetDelivery)
		sup.PATCH("/deliveries/:id", h.UpdateDelivery)
		sup.DELETE("/deliveries/:id", h.DeleteDelivery)
		sup.POST("/deliveries/:id/pay", middleware.RequireOwner(), h.DeliveryPay)
		sup.GET("/payments", h.AllPayments)
		sup.GET("/:id", h.Get)
		sup.PATCH("/:id", h.Update)
		sup.DELETE("/:id", h.Delete)
		sup.GET("/:id/deliveries", h.Deliveries)
		sup.POST("/:id/deliveries", h.Deliveries)
		sup.GET("/:id/payments", h.SupplierPayments)
		sup.POST("/:id/pay", middleware.RequireOwner(), h.Pay)
	}
}
