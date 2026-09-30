package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/database"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/integrations/ai"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/mailer"
	"github.com/troubleman96/ziadapos/internal/scheduler"
	"github.com/troubleman96/ziadapos/internal/security"
	"github.com/troubleman96/ziadapos/internal/server"
	"github.com/troubleman96/ziadapos/internal/storage"

	// apps (registered as they are built)
	"github.com/troubleman96/ziadapos/apps/accounts"
	appadmin "github.com/troubleman96/ziadapos/apps/admin"
	appai "github.com/troubleman96/ziadapos/apps/ai"
	"github.com/troubleman96/ziadapos/apps/analytics"
	"github.com/troubleman96/ziadapos/apps/credits"
	"github.com/troubleman96/ziadapos/apps/customers"
	"github.com/troubleman96/ziadapos/apps/expenses"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/apps/notebook"
	"github.com/troubleman96/ziadapos/apps/notifications"
	"github.com/troubleman96/ziadapos/apps/pharmacy"
	"github.com/troubleman96/ziadapos/apps/reports"
	"github.com/troubleman96/ziadapos/apps/reviews"
	"github.com/troubleman96/ziadapos/apps/staff"
	"github.com/troubleman96/ziadapos/apps/stores"
	"github.com/troubleman96/ziadapos/apps/subscriptions"
	"github.com/troubleman96/ziadapos/apps/suppliers"
	"github.com/troubleman96/ziadapos/apps/transactions"
	"github.com/troubleman96/ziadapos/apps/uploads"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	// Redis is optional at boot in development; production should have it.
	var rdb *cache.Cache
	if rdb, err = cache.New(ctx, cfg); err != nil {
		log.Printf("cache: running without Redis (boot continues)")
	}

	store, err := storage.New(ctx, cfg)
	if err != nil {
		log.Printf("storage: running without MinIO (boot continues): %v", err)
	}

	bus := events.New(10, 256)
	defer bus.Close()

	sched := scheduler.New(cfg.Timezone)

	deps := &server.Deps{
		Cfg:       cfg,
		DB:        db,
		Cache:     rdb,
		Storage:   store,
		Mailer:    mailer.New(cfg),
		SMS:       sms.New(cfg),
		AI:        ai.New(cfg.NgamiaBaseURL, cfg.NgamiaAPIKey, cfg.NgamiaModel),
		Bus:       bus,
		Tokens:    security.NewManager(cfg),
		Scheduler: sched,
	}

	srv := server.New(deps)

	registerApps(db, deps, srv)

	sched.Start()

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("server: listening on :%s (%s)", cfg.Port, cfg.Environment)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("server: shutting down...")
	sched.Stop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

// registerApps runs every app's AutoMigrate + route registration. Apps add
// themselves here as they ship.
func registerApps(db *gorm.DB, deps *server.Deps, srv *server.Server) {
	if err := accounts.Migrate(db); err != nil {
		log.Fatalf("accounts migrate: %v", err)
	}
	accounts.RegisterHandlers(deps.Bus, deps)
	accounts.RegisterRoutes(srv.Base, deps)

	if err := inventory.Migrate(db); err != nil {
		log.Fatalf("inventory migrate: %v", err)
	}
	inventory.RegisterRoutes(srv.Base, deps)

	if err := pharmacy.Migrate(db); err != nil {
		log.Fatalf("pharmacy migrate: %v", err)
	}
	pharmacy.RegisterRoutes(srv.Base, deps)

	if err := transactions.Migrate(db); err != nil {
		log.Fatalf("transactions migrate: %v", err)
	}
	transactions.RegisterRoutes(srv.Base, deps)

	if err := customers.Migrate(db); err != nil {
		log.Fatalf("customers migrate: %v", err)
	}
	customers.RegisterHandlers(deps.Bus, deps)
	customers.RegisterRoutes(srv.Base, deps)

	if err := credits.Migrate(db); err != nil {
		log.Fatalf("credits migrate: %v", err)
	}
	credits.RegisterHandlers(deps.Bus, deps)
	credits.RegisterRoutes(srv.Base, deps)

	if err := suppliers.Migrate(db); err != nil {
		log.Fatalf("suppliers migrate: %v", err)
	}
	suppliers.RegisterRoutes(srv.Base, deps)

	if err := expenses.Migrate(db); err != nil {
		log.Fatalf("expenses migrate: %v", err)
	}
	expenses.RegisterRoutes(srv.Base, deps)

	if err := staff.Migrate(db); err != nil {
		log.Fatalf("staff migrate: %v", err)
	}
	staff.RegisterRoutes(srv.Base, deps)

	if err := subscriptions.Migrate(db); err != nil {
		log.Fatalf("subscriptions migrate: %v", err)
	}
	subscriptions.RegisterHandlers(deps.Bus, deps)
	subscriptions.RegisterRoutes(srv.Base, deps)

	if err := analytics.Migrate(db); err != nil {
		log.Fatalf("analytics migrate: %v", err)
	}
	analytics.RegisterHandlers(deps.Bus, deps)
	analytics.RegisterRoutes(srv.Base, deps)

	if err := stores.Migrate(db); err != nil {
		log.Fatalf("stores migrate: %v", err)
	}
	stores.RegisterHandlers(deps.Bus, deps)
	stores.RegisterRoutes(srv.Base, deps)

	if err := notebook.Migrate(db); err != nil {
		log.Fatalf("notebook migrate: %v", err)
	}
	notebook.RegisterHandlers(deps.Bus, deps)
	notebook.RegisterRoutes(srv.Base, deps)

	if err := reviews.Migrate(db); err != nil {
		log.Fatalf("reviews migrate: %v", err)
	}
	reviews.RegisterHandlers(deps.Bus, deps)
	reviews.RegisterRoutes(srv.Base, deps)

	if err := appai.Migrate(db); err != nil {
		log.Fatalf("ai migrate: %v", err)
	}
	appai.RegisterRoutes(srv.Base, deps)

	if err := reports.Migrate(db); err != nil {
		log.Fatalf("reports migrate: %v", err)
	}
	reports.RegisterHandlers(deps.Bus, deps)
	reports.RegisterRoutes(srv.Base, deps)

	if err := notifications.Migrate(db); err != nil {
		log.Fatalf("notifications migrate: %v", err)
	}
	notifications.RegisterHandlers(deps.Bus, deps)
	notifications.RegisterRoutes(srv.Base, deps)

	if err := uploads.Migrate(db); err != nil {
		log.Fatalf("uploads migrate: %v", err)
	}
	uploads.RegisterRoutes(srv.Base, deps)

	if err := appadmin.Migrate(db); err != nil {
		log.Fatalf("admin migrate: %v", err)
	}
	appadmin.RegisterRoutes(srv.Base, deps)
}
