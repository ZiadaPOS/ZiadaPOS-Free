package transactions

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the transactions schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Transaction{},
		&TransactionLine{},
		&TxnCounter{},
	)
}

// RegisterRoutes wires the POS router (all endpoints require auth).
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	repo := NewRepo(deps.DB)
	invSvc := inventory.NewService(inventory.NewRepo(deps.DB), deps.Cache, deps.Bus)
	svc := NewService(repo, deps.Cache, deps.Bus, invSvc, deps.DB)
	h := NewHandler(svc)

	pos := base.Group("/pos", middleware.Auth(deps.Tokens))
	{
		pos.POST("/sale", h.CompleteSale)
		pos.GET("/transactions", h.ListTransactions)
		pos.GET("/transactions/:id", h.GetTransaction)
		pos.POST("/transactions/:id/refund", h.Refund)
		pos.POST("/transactions/:id/void", h.Void)
		pos.GET("/summary", h.Summary)
		pos.GET("/summary/hourly", h.HourlySales)
	}
}
