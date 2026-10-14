// Package server wires shared dependencies and the Gin engine together.
package server

import (
	"net/http"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/integrations/ai"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/mailer"
	"github.com/troubleman96/ziadapos/internal/middleware"
	"github.com/troubleman96/ziadapos/internal/scheduler"
	"github.com/troubleman96/ziadapos/internal/security"
	"github.com/troubleman96/ziadapos/internal/storage"
	"gorm.io/gorm"
)

// Deps is the shared dependency container passed to every app so they can
// register routes without each constructing their own clients.
type Deps struct {
	Cfg       *config.Config
	DB        *gorm.DB
	Cache     *cache.Cache
	Storage   *storage.Storage
	Mailer    *mailer.Mailer
	SMS       *sms.Client
	AI        *ai.Client
	Bus       *events.Bus
	Tokens    *security.Manager
	Secrets   *security.SecretBox
	Scheduler *scheduler.Scheduler
}

// Server owns the HTTP engine and the API router group.
type Server struct {
	*Deps
	Engine *gin.Engine
	Base   *gin.RouterGroup
}

// New builds the Gin engine and applies global middleware.
func New(deps *Deps) *Server {
	gin.SetMode(gin.ReleaseMode)
	if deps.Cfg.Debug {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(),
		middleware.Logger(),
		middleware.CORS(deps.Cfg.CORSAllowedOrigins),
	)
	engine.RedirectTrailingSlash = true

	base := engine.Group(deps.Cfg.APIBasePath)

	s := &Server{Deps: deps, Engine: engine, Base: base}

	// Liveness + readiness + deep health.
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    deps.Cfg.AppName,
			"version": "1.0.0",
			"env":     deps.Cfg.Environment,
		})
	})
	base.GET("/health", s.health)

	return s
}

func (s *Server) health(c *gin.Context) {
	status := "ok"

	sqlDB, err := s.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"app":     s.Cfg.AppName,
		"env":     s.Cfg.Environment,
		"db":      status,
		"redis":   "unknown",
		"uptime":  runtime.NumGoroutine(),
		"runtime": runtime.Version(),
	})
}
