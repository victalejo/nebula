package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/service"
)

// DeployHandler handles deployment endpoints
type DeployHandler struct {
	deployService *service.DeployService
	log           logger.Logger
}

// NewDeployHandler creates a new deploy handler
func NewDeployHandler(deployService *service.DeployService, log logger.Logger) *DeployHandler {
	return &DeployHandler{
		deployService: deployService,
		log:           log,
	}
}

// DeployImage deploys an application from a Docker image
func (h *DeployHandler) DeployImage(c *gin.Context) {
	appID := c.Param("id")

	var req service.DeployImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	deployment, err := h.deployService.DeployImage(c.Request.Context(), appID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"data":    deployment,
		"message": "deployment started",
	})
}

// ListDeployments returns all deployments for an application
func (h *DeployHandler) ListDeployments(c *gin.Context) {
	appID := c.Param("id")

	deployments, err := h.deployService.ListDeployments(c.Request.Context(), appID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": deployments,
	})
}

// GetDeployment retrieves a deployment by ID
func (h *DeployHandler) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("did")

	deployment, err := h.deployService.GetDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": deployment,
	})
}
