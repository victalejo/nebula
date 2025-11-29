package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/service"
)

// AppHandler handles application endpoints
type AppHandler struct {
	appService *service.AppService
	log        logger.Logger
}

// NewAppHandler creates a new app handler
func NewAppHandler(appService *service.AppService, log logger.Logger) *AppHandler {
	return &AppHandler{
		appService: appService,
		log:        log,
	}
}

// List returns all applications
func (h *AppHandler) List(c *gin.Context) {
	apps, err := h.appService.List(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": apps,
	})
}

// Create creates a new application
func (h *AppHandler) Create(c *gin.Context) {
	var req service.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	app, err := h.appService.Create(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": app,
	})
}

// Get retrieves an application by ID
func (h *AppHandler) Get(c *gin.Context) {
	id := c.Param("id")

	app, err := h.appService.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": app,
	})
}

// Update updates an application
func (h *AppHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	app, err := h.appService.Update(c.Request.Context(), id, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": app,
	})
}

// Delete deletes an application
func (h *AppHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.appService.Delete(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "application deleted",
	})
}

// handleError handles application errors
func handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		response := gin.H{
			"error":   appErr.Message,
			"code":    appErr.Type,
		}
		if appErr.Details != nil {
			response["details"] = appErr.Details
		}
		c.JSON(appErr.StatusCode, response)
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}
