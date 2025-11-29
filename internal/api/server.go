package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/api/handler"
	"github.com/victalejo/nebula/internal/api/middleware"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/service"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host          string
	Port          int
	JWTSecret     string
	TokenDuration time.Duration
}

// Server represents the API server
type Server struct {
	config        ServerConfig
	router        *gin.Engine
	httpServer    *http.Server
	appService    *service.AppService
	deployService *service.DeployService
	log           logger.Logger
}

// NewServer creates a new API server
func NewServer(
	config ServerConfig,
	appService *service.AppService,
	deployService *service.DeployService,
	log logger.Logger,
) *Server {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	server := &Server{
		config:        config,
		router:        router,
		appService:    appService,
		deployService: deployService,
		log:           log,
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

	// API v1
	v1 := s.router.Group("/api/v1")

	// Auth routes (no auth required)
	authHandler := handler.NewAuthHandler(s.config.JWTSecret, s.config.TokenDuration, s.log)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.Auth(s.config.JWTSecret))

	// App routes
	appHandler := handler.NewAppHandler(s.appService, s.log)
	protected.GET("/apps", appHandler.List)
	protected.POST("/apps", appHandler.Create)
	protected.GET("/apps/:id", appHandler.Get)
	protected.PUT("/apps/:id", appHandler.Update)
	protected.DELETE("/apps/:id", appHandler.Delete)

	// Deployment routes
	deployHandler := handler.NewDeployHandler(s.deployService, s.log)
	protected.POST("/apps/:id/deploy/image", deployHandler.DeployImage)
	protected.GET("/apps/:id/deployments", deployHandler.ListDeployments)
	protected.GET("/apps/:id/deployments/:did", deployHandler.GetDeployment)

	// Log routes
	logHandler := handler.NewLogHandler(s.log)
	protected.GET("/apps/:id/logs", logHandler.StreamLogs)
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
