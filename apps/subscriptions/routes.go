package subscriptions

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the subscriptions schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&SubscriptionPlan{}, &Subscription{})
}

// RegisterHandlers reacts to platform events (trial subscription on signup).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {
	repo := NewRepo(deps.DB)
	bus.On("user.registered", OnUserRegistered(repo))
}

// RegisterRoutes wires the subscriptions router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB))
	h := NewHandler(svc)

	// Public plan catalogue (registration / pricing page).
	pub := base.Group("/subscriptions")
	{
		pub.GET("/plans", h.ListPlans)
		pub.GET("/plans/:id", h.GetPlan)
	}

	// Authenticated owner endpoints.
	owner := base.Group("/subscriptions", middleware.Auth(deps.Tokens), middleware.RequireOwner())
	{
		owner.GET("/my-subscription", h.MySubscription)
		owner.GET("/store-limit", h.StoreLimit)
	}

	// Cameltech admin management.
	admin := base.Group("/subscriptions", middleware.Auth(deps.Tokens), middleware.RequireAdmin())
	{
		admin.POST("/plans", h.CreatePlan)
		admin.PATCH("/plans/:id", h.UpdatePlan)
		admin.DELETE("/plans/:id", h.DeactivatePlan)
		admin.GET("/all", h.AdminList)
		admin.GET("/all/:id", h.AdminGet)
		admin.POST("/all/:id/activate", h.Activate)
		admin.PATCH("/all/:id/extra-stores", h.ExtraStores)
	}
}
