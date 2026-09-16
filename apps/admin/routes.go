package admin

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate is a no-op: the admin surface has no tables of its own and reads
// across existing schemas.
func Migrate(db *gorm.DB) error {
	return nil
}

// RegisterRoutes wires the platform-wide admin router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	repo := NewRepo(deps.DB)
	svc := NewService(repo)
	h := NewHandler(svc)

	r := base.Group("/admin", middleware.Auth(deps.Tokens), middleware.RequireAdmin())
	{
		r.GET("/dashboard", h.Dashboard)

		r.GET("/organisations", h.Organisations)
		r.GET("/organisations/:id", h.Organisation)
		r.GET("/organisations/:id/stores", h.OrganisationStores)
		r.GET("/organisations/:id/users", h.OrganisationUsers)
		r.PATCH("/organisations/:id", h.UpdateOrganisation)

		r.GET("/users", h.Users)
		r.PATCH("/users/:id", h.UpdateUser)

		r.GET("/stores", h.Stores)
		r.PATCH("/stores/:id", h.UpdateStore)

		r.GET("/products", h.Products)
		r.GET("/products/:id", h.Product)
		r.PATCH("/products/:id", h.UpdateProduct)
		r.GET("/categories", h.Categories)

		r.GET("/transactions", h.Transactions)
		r.GET("/transactions/:id", h.Transaction)
		r.GET("/transactions/summary", h.TransactionSummary)

		r.GET("/customers", h.Customers)
		r.GET("/credits", h.Credits)

		r.GET("/suppliers", h.Suppliers)
		r.GET("/suppliers/:id/deliveries", h.SupplierDeliveries)
		r.GET("/suppliers/:id/payments", h.SupplierPayments)

		r.GET("/expenses", h.Expenses)

		r.GET("/reviews", h.Reviews)
		r.PATCH("/reviews/:id", h.UpdateReview)
		r.DELETE("/reviews/:id", h.DeleteReview)

		r.GET("/notes", h.Notes)
		r.GET("/notifications", h.Notifications)

		r.GET("/ai/usage", h.AIUsage)

		r.GET("/subscriptions", h.Subscriptions)

		r.GET("/analytics/overview", h.AnalyticsOverview)
	}
}
