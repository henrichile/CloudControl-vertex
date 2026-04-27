package api

import (
	"github.com/etasoft/cloudcontrol/internal/api/handlers"
	"github.com/etasoft/cloudcontrol/internal/api/middleware"
	"github.com/gin-gonic/gin"
)

type Router struct {
	health     *handlers.HealthHandler
	auth       *handlers.AuthHandler
	containers *handlers.ContainerHandler
	projects   *handlers.ProjectHandler
	aiops      *handlers.AIOpsHandler
	jwtSecret  string
}

func NewRouter(
	health *handlers.HealthHandler,
	auth *handlers.AuthHandler,
	containers *handlers.ContainerHandler,
	projects *handlers.ProjectHandler,
	aiops *handlers.AIOpsHandler,
	jwtSecret string,
) *Router {
	return &Router{
		health:     health,
		auth:       auth,
		containers: containers,
		projects:   projects,
		aiops:      aiops,
		jwtSecret:  jwtSecret,
	}
}

func (r *Router) Register(engine *gin.Engine) {
	engine.Use(middleware.CORS())

	engine.GET("/api/v1/health", r.health.Check)

	// Auth (public)
	auth := engine.Group("/api/v1/auth")
	{
		auth.POST("/register", r.auth.Register)
		auth.POST("/login", r.auth.Login)
	}

	// Protected routes
	v1 := engine.Group("/api/v1")
	v1.Use(middleware.NewJWTMiddleware(r.jwtSecret))
	{
		v1.GET("/auth/me", r.auth.Me)

		// Containers
		v1.GET("/containers", r.containers.List)
		v1.GET("/containers/:id", r.containers.Inspect)
		v1.POST("/containers/:id/start", r.containers.Start)
		v1.POST("/containers/:id/stop", r.containers.Stop)
		v1.DELETE("/containers/:id", r.containers.Remove)
		v1.GET("/containers/:id/logs", r.containers.Logs)
		v1.GET("/containers/:id/stats", r.containers.Stats)
		v1.GET("/containers/:id/stats/ws", r.containers.StatsWS)
		v1.PATCH("/containers/:id/limits", r.containers.UpdateLimits)

		// Projects
		v1.GET("/projects", r.projects.List)
		v1.GET("/projects/:id", r.projects.Get)
		v1.POST("/projects", r.projects.Create)
		v1.POST("/projects/:id/up", r.projects.Up)
		v1.GET("/projects/:id/up/stream", r.projects.UpStream)
		v1.POST("/projects/:id/down", r.projects.Down)
		v1.DELETE("/projects/:id", r.projects.Delete)

		// Stacks (templates)
		v1.GET("/stacks", r.projects.ListStacks)

		// AIOps
		v1.POST("/aiops/analyze", r.aiops.AnalyzeContainer)
		v1.POST("/aiops/audit", r.aiops.AuditConfig)
		v1.POST("/aiops/logs", r.aiops.AnalyzeLogs)
	}
}
