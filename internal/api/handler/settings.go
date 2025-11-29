package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

const (
	// SettingGitHubToken is the key for GitHub token setting
	SettingGitHubToken = "github_token"
)

// SettingsHandler handles settings endpoints
type SettingsHandler struct {
	store storage.SettingsRepository
	log   logger.Logger
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(store storage.SettingsRepository, log logger.Logger) *SettingsHandler {
	return &SettingsHandler{
		store: store,
		log:   log,
	}
}

// GitHubTokenRequest represents a request to set GitHub token
type GitHubTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// GitHubTokenStatusResponse represents the GitHub token status
type GitHubTokenStatusResponse struct {
	Configured bool `json:"configured"`
}

// GetGitHubTokenStatus returns whether GitHub token is configured
func (h *SettingsHandler) GetGitHubTokenStatus(c *gin.Context) {
	token, err := h.store.Get(c.Request.Context(), SettingGitHubToken)
	if err != nil {
		h.log.Error("failed to get github token status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get token status",
		})
		return
	}

	c.JSON(http.StatusOK, GitHubTokenStatusResponse{
		Configured: token != "",
	})
}

// SetGitHubToken sets the GitHub token
func (h *SettingsHandler) SetGitHubToken(c *gin.Context) {
	var req GitHubTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "token is required",
		})
		return
	}

	if err := h.store.Set(c.Request.Context(), SettingGitHubToken, req.Token); err != nil {
		h.log.Error("failed to set github token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "token saved successfully",
	})
}

// DeleteGitHubToken removes the GitHub token
func (h *SettingsHandler) DeleteGitHubToken(c *gin.Context) {
	if err := h.store.Delete(c.Request.Context(), SettingGitHubToken); err != nil {
		h.log.Error("failed to delete github token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "token deleted successfully",
	})
}
