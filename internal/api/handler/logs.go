package handler

import (
	"bufio"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/logger"
)

// LogHandler handles log streaming endpoints
type LogHandler struct {
	log logger.Logger
}

// NewLogHandler creates a new log handler
func NewLogHandler(log logger.Logger) *LogHandler {
	return &LogHandler{
		log: log,
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

	c.SSEvent("message", fmt.Sprintf("Log streaming for app %s (tail=%s, follow=%v)", appID, tail, follow))
	c.Writer.Flush()

	if !follow {
		return
	}

	// For follow mode, keep connection open
	notify := c.Request.Context().Done()

	// Create a channel for log messages
	// TODO: Implement actual log streaming from Docker containers

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

	// Get follow parameter
	follow := c.Query("follow") == "true"
	tail := c.DefaultQuery("tail", "100")

	// TODO: Get container IDs for the specific deployment and stream logs from them
	// For now, send a placeholder message

	c.SSEvent("message", fmt.Sprintf("Log streaming for deployment %s (app=%s, tail=%s, follow=%v)", deploymentID, appID, tail, follow))
	c.Writer.Flush()

	if !follow {
		return
	}

	// For follow mode, keep connection open
	notify := c.Request.Context().Done()

	// TODO: Implement actual log streaming from Docker containers for this deployment

	<-notify
	h.log.Info("deployment log stream closed", "app_id", appID, "deployment_id", deploymentID)
}

// streamContainerLogs streams logs from a container (helper function)
func streamContainerLogs(reader io.ReadCloser, c *gin.Context) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-c.Request.Context().Done():
			return
		default:
			line := scanner.Text()
			c.SSEvent("log", line)
			c.Writer.Flush()
		}
	}
}
