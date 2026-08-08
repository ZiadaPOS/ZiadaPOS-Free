package customers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the customers schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Customer{})
}

// RegisterHandlers keeps customer spend stats in sync with POS events.
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.Cache, deps.DB)

	// On a completed sale, bump spend / last visit / segment.
	bus.On("transaction.created", func(ctx context.Context, e events.Event) error {
		p, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		customerID := parseUUID(p["customer_id"])
		total, _ := p["total"].(int64)
		if customerID == nil {
			return nil
		}
		return svc.RecordSale(ctx, customerID, total)
	})

	// On a refund, reduce spend stats.
	bus.On("transaction.refunded", func(ctx context.Context, e events.Event) error {
		p, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		customerID := parseUUID(p["customer_id"])
		total, _ := p["total"].(int64)
		if customerID == nil {
			return nil
		}
		return svc.RecordRefund(ctx, customerID, total)
	})
}

// RegisterRoutes wires the customers router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.Cache, deps.DB)
	h := NewHandler(svc)

	customers := base.Group("/customers", middleware.Auth(deps.Tokens))
	{
		customers.GET("", h.List)
		customers.POST("", h.Create)
		customers.GET("/credits", h.CreditCustomers)
		customers.GET("/:id", h.Get)
		customers.PATCH("/:id", h.Update)
	}
}

func parseUUID(v interface{}) *uuid.UUID {
	switch t := v.(type) {
	case string:
		uid, err := uuid.Parse(t)
		if err != nil {
			return nil
		}
		return &uid
	case uuid.UUID:
		if t == uuid.Nil {
			return nil
		}
		return &t
	case *uuid.UUID:
		return t
	default:
		return nil
	}
}
