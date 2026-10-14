package credits

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/apps/customers"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the credits schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&CreditTab{}, &CreditPayment{}, &CreditMessage{}, &CreditNote{})
}

// RegisterHandlers wires the credits Service and reacts to POS events:
//   - a credit sale auto-creates a tab
//   - a refunded/voided credit sale settles its tab
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {
	custSvc := customers.NewService(customers.NewRepo(deps.DB), deps.Cache, deps.DB, deps.SMS, deps.Secrets)
	svc := NewService(NewRepo(deps.DB), deps.Cache, bus, custSvc, deps.DB, deps.SMS)

	bus.On("transaction.created", func(ctx context.Context, e events.Event) error {
		p, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		if p["method"] != "Credit" {
			return nil
		}
		customerID := parseUUID(p["customer_id"])
		if customerID == nil {
			return nil
		}
		orgID := parseUUID(p["org_id"])
		storeID := parseUUID(p["store_id"])
		cashierID := parseUUID(p["cashier_id"])
		txnID := parseUUID(p["transaction_id"])
		till, _ := p["till_number"].(string)
		total, _ := p["total"].(int64)
		if orgID == nil || storeID == nil || txnID == nil {
			return nil
		}
		// Best-effort: failures are logged by the bus, never break the sale.
		return svc.CreateTabFromSale(ctx, *orgID, *storeID, *customerID, *txnID, total, uuidValue(cashierID), till)
	})

	settle := func(ctx context.Context, e events.Event) error {
		p, ok := e.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		txnID := parseUUID(p["transaction_id"])
		if txnID == nil {
			return nil
		}
		return svc.SettleTabForRefund(ctx, *txnID)
	}
	bus.On("transaction.refunded", settle)
	bus.On("transaction.voided", settle)
}

// RegisterRoutes wires the credits router.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	custSvc := customers.NewService(customers.NewRepo(deps.DB), deps.Cache, deps.DB, deps.SMS, deps.Secrets)
	svc := NewService(NewRepo(deps.DB), deps.Cache, deps.Bus, custSvc, deps.DB, deps.SMS)
	h := NewHandler(svc)

	credits := base.Group("/credits", middleware.Auth(deps.Tokens))
	{
		credits.GET("", h.List)
		credits.GET("/customers/:customer_id", h.Profile)
		credits.POST("/customers/:customer_id/payments", h.Pay)
		credits.POST("/customers/:customer_id/remind", h.SendReminder)
		credits.POST("/customers/:customer_id/messages", h.AddMessage)
		credits.POST("/customers/:customer_id/notes", h.AddNote)
		credits.POST("/:tab_id/write-off", h.WriteOff)
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

func uuidValue(v *uuid.UUID) uuid.UUID {
	if v == nil {
		return uuid.Nil
	}
	return *v
}
