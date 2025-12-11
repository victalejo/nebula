package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/service"
	"github.com/victalejo/nebula/internal/version"
)

// UpdateHandler handles system update endpoints
type UpdateHandler struct {
	updateService *service.UpdateService
	log           logger.Logger
}

// NewUpdateHandler creates a new update handler
func NewUpdateHandler(updateService *service.UpdateService, log logger.Logger) *UpdateHandler {
	return &UpdateHandler{
		updateService: updateService,
		log:           log,
	}
}

// GetSystemInfo returns system version information
func (h *UpdateHandler) GetSystemInfo(c *gin.Context) {
	info := version.GetInfo()
	cfg := h.updateService.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"version":       info.Version,
			"build_time":    info.BuildTime,
			"commit":        info.Commit,
			"update_mode":   cfg.Mode,
			"check_interval": cfg.CheckInterval,
		},
	})
}

// GetUpdateStatus returns the current update status and last check info
func (h *UpdateHandler) GetUpdateStatus(c *gin.Context) {
	status := h.updateService.GetStatus()
	lastCheck := h.updateService.GetLastCheck()

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":     status,
			"last_check": lastCheck,
		},
	})
}

// CheckForUpdates triggers an update check
func (h *UpdateHandler) CheckForUpdates(c *gin.Context) {
	info, err := h.updateService.CheckForUpdates(c.Request.Context())
	if err != nil {
		h.log.Error("failed to check for updates", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to check for updates: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": info,
	})
}

// ApplyUpdate applies a downloaded update
func (h *UpdateHandler) ApplyUpdate(c *gin.Context) {
	// First check if there's an update available
	lastCheck := h.updateService.GetLastCheck()
	if lastCheck == nil || !lastCheck.Available {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no update available",
		})
		return
	}

	// Start the download and apply process asynchronously
	go func() {
		ctx := context.Background()
		if err := h.updateService.DownloadAndApply(ctx); err != nil {
			h.log.Error("failed to download update", "error", err)
			return
		}
		// Apply the update regardless of mode (user explicitly requested it)
		if err := h.updateService.ApplyUpdate(ctx); err != nil {
			h.log.Error("failed to apply update", "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "update started",
	})
}

// DownloadUpdate downloads an available update
func (h *UpdateHandler) DownloadUpdate(c *gin.Context) {
	lastCheck := h.updateService.GetLastCheck()
	if lastCheck == nil || !lastCheck.Available {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no update available",
		})
		return
	}

	go func() {
		if err := h.updateService.DownloadAndApply(context.Background()); err != nil {
			h.log.Error("failed to download update", "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "download started",
	})
}

// ListBackups returns all available backups
func (h *UpdateHandler) ListBackups(c *gin.Context) {
	backups, err := h.updateService.ListBackups(c.Request.Context())
	if err != nil {
		h.log.Error("failed to list backups", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list backups: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": backups,
	})
}

// Rollback restores a previous version
func (h *UpdateHandler) Rollback(c *gin.Context) {
	backupID := c.Param("id")
	if backupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "backup ID is required",
		})
		return
	}

	go func() {
		if err := h.updateService.Rollback(context.Background(), backupID); err != nil {
			h.log.Error("failed to rollback", "error", err, "backup_id", backupID)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "rollback started",
	})
}

// UpdateConfigRequest represents a request to update config
type UpdateConfigRequest struct {
	Mode          string `json:"mode"`
	CheckInterval int    `json:"check_interval"`
}

// UpdateConfiguration updates the update configuration
func (h *UpdateHandler) UpdateConfiguration(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	// Validate mode
	if req.Mode != "" && req.Mode != "auto" && req.Mode != "notify" && req.Mode != "disabled" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid mode: must be 'auto', 'notify', or 'disabled'",
		})
		return
	}

	cfg := h.updateService.GetConfig()
	if req.Mode != "" {
		cfg.Mode = req.Mode
	}
	if req.CheckInterval > 0 {
		cfg.CheckInterval = req.CheckInterval
	}

	h.updateService.UpdateConfig(cfg)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"mode":           cfg.Mode,
			"check_interval": cfg.CheckInterval,
		},
	})
}

// GetConfiguration returns the current update configuration
func (h *UpdateHandler) GetConfiguration(c *gin.Context) {
	cfg := h.updateService.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"mode":           cfg.Mode,
			"check_interval": cfg.CheckInterval,
		},
	})
}
