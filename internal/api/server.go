package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/api/handler"
	"github.com/victalejo/nebula/internal/api/middleware"
	nebulacontainer "github.com/victalejo/nebula/internal/core/container"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
	"github.com/victalejo/nebula/internal/service"
	"github.com/victalejo/nebula/web"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host          string
	Port          int
	JWTSecret     string
	TokenDuration time.Duration
	AdminUsername string
	AdminPassword string
}

// Server represents the API server
type Server struct {
	config         ServerConfig
	router         *gin.Engine
	httpServer     *http.Server
	appService     *service.AppService
	serviceService *service.ServiceService
	domainService  *service.DomainService
	deployService  *service.DeployService
	updateService  *service.UpdateService
	settingsStore  storage.SettingsRepository
	containerRuntime nebulacontainer.ContainerRuntime
	containerStore   storage.ContainerRepository
	log            logger.Logger
}

// NewServer creates a new API server
func NewServer(
	config ServerConfig,
	appService *service.AppService,
	serviceService *service.ServiceService,
	domainService *service.DomainService,
	deployService *service.DeployService,
	updateService *service.UpdateService,
	settingsStore storage.SettingsRepository,
	containerRuntime nebulacontainer.ContainerRuntime,
	containerStore storage.ContainerRepository,
	log logger.Logger,
) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	server := &Server{
		config:           config,
		router:           router,
		appService:       appService,
		serviceService:   serviceService,
		domainService:    domainService,
		deployService:    deployService,
		updateService:    updateService,
		settingsStore:    settingsStore,
		containerRuntime: containerRuntime,
		containerStore:   containerStore,
		log:              log,
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logging middleware
	s.router.Use(middleware.Logger(s.log))

	// CORS middleware
	s.router.Use(middleware.CORS())
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Serve embedded frontend
	frontendFS, err := web.GetFS()
	if err == nil {
		// Assets with cache busting (filename has hash)
		s.router.StaticFS("/assets", http.FS(mustSub(frontendFS, "assets")))

		// Index.html - no cache to ensure latest version
		s.router.GET("/", func(c *gin.Context) {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.FileFromFS("/", http.FS(frontendFS))
		})
		s.router.NoRoute(func(c *gin.Context) {
			// SPA fallback - serve index.html for non-API routes
			if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:5] == "/api/" {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.FileFromFS("/", http.FS(frontendFS))
		})
	}

	// API v1
	v1 := s.router.Group("/api/v1")

	// Auth routes (no auth required)
	authHandler := handler.NewAuthHandler(s.config.JWTSecret, s.config.TokenDuration, s.config.AdminUsername, s.config.AdminPassword, s.log)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.Auth(s.config.JWTSecret))

	// Auth routes (protected)
	protected.GET("/auth/me", authHandler.Me)

	// App/Project routes (legacy compatibility - apps endpoint maps to projects)
	appHandler := handler.NewAppHandler(s.appService, s.log)
	protected.GET("/apps", appHandler.List)
	protected.POST("/apps", appHandler.Create)
	protected.GET("/apps/:id", appHandler.Get)
	protected.PUT("/apps/:id", appHandler.Update)
	protected.DELETE("/apps/:id", appHandler.Delete)

	// Project routes (new canonical endpoint)
	protected.GET("/projects", appHandler.List)
	protected.POST("/projects", appHandler.Create)
	protected.GET("/projects/:id", appHandler.Get)
	protected.PUT("/projects/:id", appHandler.Update)
	protected.DELETE("/projects/:id", appHandler.Delete)

	// Service routes
	serviceHandler := handler.NewServiceHandler(s.serviceService, s.log)
	protected.GET("/projects/:id/services", serviceHandler.List)
	protected.POST("/projects/:id/services", serviceHandler.Create)
	protected.GET("/projects/:id/services/:serviceName", serviceHandler.Get)
	protected.PUT("/projects/:id/services/:serviceName", serviceHandler.Update)
	protected.DELETE("/projects/:id/services/:serviceName", serviceHandler.Delete)
	protected.GET("/services/:serviceId", serviceHandler.GetByID)

	// Domain routes
	domainHandler := handler.NewDomainHandler(s.domainService, s.log)
	protected.GET("/projects/:id/domains", domainHandler.ListByProject)
	protected.GET("/projects/:id/services/:serviceName/domains", domainHandler.ListByService)
	protected.POST("/projects/:id/services/:serviceName/domains", domainHandler.Create)
	protected.GET("/domains/:domain", domainHandler.Get)
	protected.PUT("/domains/:domain", domainHandler.Update)
	protected.DELETE("/domains/:domain", domainHandler.Delete)

	// Deployment routes
	deployHandler := handler.NewDeployHandler(s.deployService, s.log)
	protected.POST("/apps/:id/deploy/image", deployHandler.DeployImage)
	protected.POST("/apps/:id/deploy/git", deployHandler.DeployGit)
	protected.GET("/apps/:id/deployments", deployHandler.ListDeployments)
	protected.GET("/apps/:id/deployments/:did", deployHandler.GetDeployment)

	// Service deployment routes
	protected.POST("/projects/:id/services/:serviceName/deploy", deployHandler.DeployService)
	protected.GET("/projects/:id/services/:serviceName/deployments", deployHandler.ListServiceDeployments)

	// Log routes
	logHandler := handler.NewLogHandler(s.containerRuntime, s.containerStore, s.log)
	protected.GET("/apps/:id/logs", logHandler.StreamLogs)
	protected.GET("/apps/:id/deployments/:did/logs", logHandler.StreamDeploymentLogs)

	// Settings routes
	settingsHandler := handler.NewSettingsHandler(s.settingsStore, s.log)
	protected.GET("/settings/github-token", settingsHandler.GetGitHubTokenStatus)
	protected.PUT("/settings/github-token", settingsHandler.SetGitHubToken)
	protected.DELETE("/settings/github-token", settingsHandler.DeleteGitHubToken)

	// System/Update routes
	updateHandler := handler.NewUpdateHandler(s.updateService, s.log)
	protected.GET("/system/info", updateHandler.GetSystemInfo)
	protected.GET("/system/updates", updateHandler.GetUpdateStatus)
	protected.POST("/system/updates/check", updateHandler.CheckForUpdates)
	protected.POST("/system/updates/apply", updateHandler.ApplyUpdate)
	protected.GET("/system/updates/config", updateHandler.GetConfiguration)
	protected.PUT("/system/updates/config", updateHandler.UpdateConfiguration)
	protected.GET("/system/backups", updateHandler.ListBackups)
	protected.POST("/system/rollback/:id", updateHandler.Rollback)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
