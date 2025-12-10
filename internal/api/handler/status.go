package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/core/events"
	"github.com/victalejo/nebula/internal/core/logger"
)

// StatusHandler handles status streaming endpoints
type StatusHandler struct {
	eventBus *events.EventBus
	log      logger.Logger
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(eventBus *events.EventBus, log logger.Logger) *StatusHandler {
	return &StatusHandler{
		eventBus: eventBus,
		log:      log,
	}
}

// StreamProjectStatus streams status updates for a project via SSE
func (h *StatusHandler) StreamProjectStatus(c *gin.Context) {
	projectID := c.Param("id")

	h.log.Info("starting status stream", "project_id", projectID)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Subscribe to events
	subID := uuid.New().String()
	subscriber := h.eventBus.Subscribe(subID, projectID)
	defer h.eventBus.Unsubscribe(subID)

	// Send initial connected event
	c.SSEvent("connected", gin.H{"project_id": projectID})
	c.Writer.Flush()

	// Stream events
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			h.log.Info("status stream closed", "project_id", projectID)
			return
		case event, ok := <-subscriber.Events:
			if !ok {
				return
			}
			c.SSEvent("status", event)
			c.Writer.Flush()
		}
	}
}

// StreamGlobalStatus streams all status updates (for dashboard)
func (h *StatusHandler) StreamGlobalStatus(c *gin.Context) {
	h.log.Info("starting global status stream")

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Subscribe to all events (empty projectID)
	subID := uuid.New().String()
	subscriber := h.eventBus.Subscribe(subID, "")
	defer h.eventBus.Unsubscribe(subID)

	// Send initial connected event
	c.SSEvent("connected", gin.H{})
	c.Writer.Flush()

	// Stream events
	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			h.log.Info("global status stream closed")
			return
		case event, ok := <-subscriber.Events:
			if !ok {
				return
			}
			c.SSEvent("status", event)
			c.Writer.Flush()
		}
	}
}
