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
	if err := db.AutoMigrate(
		&Transaction{},
		&TransactionLine{},
		&TxnCounter{},
	); err != nil {
		return err
	}

	// TXN numbers are sequential per store; replace the legacy global unique
	// index on txn_number with a composite (store_id, txn_number) index.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_transactions_txn_number`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_txn_number_store ON transactions (store_id, txn_number)`).Error
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
