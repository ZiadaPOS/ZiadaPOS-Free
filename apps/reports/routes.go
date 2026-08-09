package reports

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/server"
)

// Migrate creates the reports schema.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ScheduledReport{}, &ReportExport{})
}

// RegisterHandlers wires reports event listeners (none yet).
func RegisterHandlers(bus *events.Bus, deps *server.Deps) {}

// RegisterRoutes wires the reports router and the scheduled-dispatch cron.
func RegisterRoutes(base *gin.RouterGroup, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.Mailer)
	h := NewHandler(svc)

	authed := base.Group("/reports", middleware.Auth(deps.Tokens))
	{
		authed.GET("/types", h.ReportTypes)
		authed.POST("/generate", h.Generate)
		authed.GET("/exports", h.Exports)
		authed.GET("/exports/:id/download", h.DownloadExport)
	}

	managers := base.Group("/reports", middleware.Auth(deps.Tokens), middleware.RequireOwner())
	{
		managers.GET("/scheduled", h.ListScheduled)
		managers.POST("/scheduled", h.CreateScheduled)
		managers.PATCH("/scheduled/:id", h.UpdateScheduled)
		managers.DELETE("/scheduled/:id", h.DeleteScheduled)
		managers.POST("/scheduled/:id/send", h.SendScheduled)
	}

	if deps.Scheduler != nil {
		_ = deps.Scheduler.AddDaily(6, 30, "reports:dispatch-due", func() {
			DispatchDue(context.Background(), deps)
		})
	}
}

// DispatchDue sends every enabled scheduled report whose next_send_at has
// arrived. Runs as part of the daily scheduler.
func DispatchDue(ctx context.Context, deps *server.Deps) {
	svc := NewService(NewRepo(deps.DB), deps.Mailer)
	due, err := svc.repo.DueScheduled(ctx, time.Now())
	if err != nil {
		return
	}
	for i := range due {
		if !due[i].IsEnabled {
			continue
		}
		_, _ = svc.dispatch(ctx, &due[i])
	}
}
