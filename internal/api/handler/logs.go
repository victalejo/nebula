package handler

import (
	"bufio"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	nebulacontainer "github.com/victalejo/nebula/internal/core/container"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// LogHandler handles log streaming endpoints
type LogHandler struct {
	runtime     nebulacontainer.ContainerRuntime
	containers  storage.ContainerRepository
	deployments storage.DeploymentRepository
	log         logger.Logger
}

// NewLogHandler creates a new log handler
func NewLogHandler(runtime nebulacontainer.ContainerRuntime, containers storage.ContainerRepository, deployments storage.DeploymentRepository, log logger.Logger) *LogHandler {
	return &LogHandler{
		runtime:     runtime,
		containers:  containers,
		deployments: deployments,
		log:         log,
	}
}

// StreamLogs streams logs for an application via Server-Sent Events
func (h *LogHandler) StreamLogs(c *gin.Context) {
	appID := c.Param("id")

	h.log.Info("starting log stream", "app_id", appID)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Get follow parameter
	follow := c.Query("follow") == "true"
	tail := c.DefaultQuery("tail", "100")

	// TODO: Get container IDs for the app and stream logs from them
	// For now, send a placeholder message
	c.SSEvent("message", "Log streaming for app "+appID+" (tail="+tail+")")
	c.Writer.Flush()

	if !follow {
		return
	}

	// For follow mode, keep connection open
	notify := c.Request.Context().Done()
	<-notify
	h.log.Info("log stream closed", "app_id", appID)
}

// StreamDeploymentLogs streams logs for a specific deployment via Server-Sent Events
func (h *LogHandler) StreamDeploymentLogs(c *gin.Context) {
	appID := c.Param("id")
	deploymentID := c.Param("did")

	h.log.Info("starting deployment log stream", "app_id", appID, "deployment_id", deploymentID)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Get follow and tail parameters
	follow := c.Query("follow") == "true"
	tailStr := c.DefaultQuery("tail", "100")

	// Get containers for this deployment
	containers, err := h.containers.ListByDeploymentID(c.Request.Context(), deploymentID)
	if err != nil {
		h.log.Error("failed to get containers", "error", err, "deployment_id", deploymentID)
		c.SSEvent("error", "Error al obtener contenedores: "+err.Error())
		c.Writer.Flush()
		return
	}

	// If no containers, try to show stored logs from the deployment
	if len(containers) == 0 {
		h.sendStoredLogs(c, deploymentID)
		return
	}

	// Stream logs from first container (usually there's only one per deployment)
	container := containers[0]
	h.log.Info("streaming logs from container", "container_id", container.ContainerID, "deployment_id", deploymentID)

	// Get logs from Docker
	logReader, err := h.runtime.ContainerLogs(c.Request.Context(), container.ContainerID, nebulacontainer.LogOptions{
		Follow:     follow,
		Tail:       tailStr,
		Stdout:     true,
		Stderr:     true,
		Timestamps: true,
	})
	if err != nil {
		h.log.Warn("failed to get container logs, trying stored logs", "error", err, "container_id", container.ContainerID)
		// Container might not exist anymore, try to show stored logs
		h.sendStoredLogs(c, deploymentID)
		return
	}
	defer logReader.Close()

	// Stream logs to client
	h.streamContainerLogs(logReader, c)

	h.log.Info("deployment log stream closed", "app_id", appID, "deployment_id", deploymentID)
}

// sendStoredLogs sends stored logs from the deployment record
func (h *LogHandler) sendStoredLogs(c *gin.Context, deploymentID string) {
	deployment, err := h.deployments.GetByID(c.Request.Context(), deploymentID)
	if err != nil || deployment == nil {
		c.SSEvent("message", "No hay contenedores activos para este despliegue")
		c.Writer.Flush()
		return
	}

	if deployment.Logs == "" {
		c.SSEvent("message", "No hay logs disponibles para este despliegue")
		c.Writer.Flush()
		return
	}

	// Send stored logs line by line
	c.SSEvent("message", "--- Logs guardados del despliegue ---")
	c.Writer.Flush()

	lines := strings.Split(deployment.Logs, "\n")
	for _, line := range lines {
		if line != "" {
			c.SSEvent("message", line)
			c.Writer.Flush()
		}
	}

	c.SSEvent("message", "--- Fin de logs guardados ---")
	c.Writer.Flush()
}

// streamContainerLogs streams logs from a container (helper function)
func (h *LogHandler) streamContainerLogs(reader io.ReadCloser, c *gin.Context) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long log lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-c.Request.Context().Done():
			return
		default:
			line := scanner.Text()
			// Docker log format has 8 bytes header for stream type
			// Skip header if present
			if len(line) > 8 {
				// Check if first byte indicates stdout(1) or stderr(2)
				if line[0] == 1 || line[0] == 2 {
					line = line[8:]
				}
			}
			c.SSEvent("message", line)
			c.Writer.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		h.log.Error("error reading container logs", "error", err)
	}
}
