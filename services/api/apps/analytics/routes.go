package analytics

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates analytics tables (none — reads from shared schema).
func Migrate(db *gorm.DB) error { return nil }

// RegisterHandlers wires analytics event listeners (none yet).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {}

// RegisterRoutes wires the analytics router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB))
	h := NewHandler(svc)

	group := base.Group("/analytics", middleware.Auth(deps.Tokens), middleware.RequireOwner())
	{
		group.GET("/overview", h.Overview)
		group.GET("/summary", h.Summary)
		group.GET("/trend", h.Trend)
		group.GET("/payment-mix", h.PaymentMix)
		group.GET("/top-products", h.TopProducts)
		group.GET("/sales", h.Sales)
		group.GET("/products", h.Products)
		group.GET("/customers", h.Customers)
		group.GET("/cashflow", h.Cashflow)
		group.GET("/dashboard", h.Dashboard)
		group.GET("/detail/:section", h.Detail)
	}
}
