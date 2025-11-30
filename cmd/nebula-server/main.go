package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victalejo/nebula/internal/api"
	"github.com/victalejo/nebula/internal/config"
	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/container/docker"
	"github.com/victalejo/nebula/internal/core/deployer"
	"github.com/victalejo/nebula/internal/core/logger"
	gitdeployer "github.com/victalejo/nebula/internal/deployer/git"
	imagedeployer "github.com/victalejo/nebula/internal/deployer/image"
	"github.com/victalejo/nebula/internal/proxy/caddy"
	"github.com/victalejo/nebula/internal/service"
	"github.com/victalejo/nebula/internal/storage/sqlite"
	"github.com/victalejo/nebula/internal/version"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	var log logger.Logger
	if cfg.Log.Format == "text" {
		log = logger.NewText(cfg.Log.Level)
	} else {
		log = logger.New(cfg.Log.Level)
	}

	log.Info("starting nebula server",
		"version", version.Version,
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)

	// Initialize storage
	store, err := sqlite.NewStore(cfg.Database.Path)
	if err != nil {
		log.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Run migrations
	if err := store.Migrate(); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize Docker client
	dockerClient, err := docker.NewClient(cfg.Docker.Host)
	if err != nil {
		log.Error("failed to initialize docker client", "error", err)
		os.Exit(1)
	}

	// Initialize Caddy proxy manager
	proxyManager := caddy.NewManager(cfg.Caddy.AdminAPI, cfg.Caddy.Network, log)

	// Initialize deployer registry
	registry := deployer.NewRegistry()

	// Register image deployer
	imgDeployer := imagedeployer.New(dockerClient, cfg.Docker.Network, log)
	registry.Register(imgDeployer)

	// Register git deployer
	runtimeAdapter := container.NewRuntimeAdapter(dockerClient)
	gitDep := gitdeployer.New(runtimeAdapter, log, "./data", store.Settings())
	registry.Register(gitDep)

	// Initialize services
	appService := service.NewAppService(store, log)
	serviceService := service.NewServiceService(store, log)
	domainService := service.NewDomainService(store, log)
	deployService := service.NewDeployService(store, registry, proxyManager, log)
	updateService := service.NewUpdateService(cfg.Update, store, log)

	// Initialize API server
	server := api.NewServer(api.ServerConfig{
		Host:          cfg.Server.Host,
		Port:          cfg.Server.Port,
		JWTSecret:     cfg.Auth.JWTSecret,
		TokenDuration: time.Duration(cfg.Auth.TokenDuration) * time.Hour,
		AdminUsername: cfg.Auth.AdminUsername,
		AdminPassword: cfg.Auth.AdminPassword,
	}, appService, serviceService, domainService, deployService, updateService, store.Settings(), log)

	// Start background update checker
	go updateService.StartBackgroundChecker(context.Background())

	// Start server
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		log.Info("server listening", "addr", addr)
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server stopped")
}
