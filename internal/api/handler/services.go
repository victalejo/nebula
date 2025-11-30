package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/service"
)

// ServiceHandler handles service endpoints
type ServiceHandler struct {
	serviceService *service.ServiceService
	log            logger.Logger
}

// NewServiceHandler creates a new service handler
func NewServiceHandler(serviceService *service.ServiceService, log logger.Logger) *ServiceHandler {
	return &ServiceHandler{
		serviceService: serviceService,
		log:            log,
	}
}

// List returns all services for a project
func (h *ServiceHandler) List(c *gin.Context) {
	projectID := c.Param("id")

	services, err := h.serviceService.List(c.Request.Context(), projectID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": services,
	})
}

// Create creates a new service within a project
func (h *ServiceHandler) Create(c *gin.Context) {
	projectID := c.Param("id")

	var req service.CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	svc, err := h.serviceService.Create(c.Request.Context(), projectID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": svc,
	})
}

// Get retrieves a service by project and name
func (h *ServiceHandler) Get(c *gin.Context) {
	projectID := c.Param("id")
	serviceName := c.Param("serviceName")

	svc, err := h.serviceService.Get(c.Request.Context(), projectID, serviceName)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": svc,
	})
}

// GetByID retrieves a service by ID
func (h *ServiceHandler) GetByID(c *gin.Context) {
	serviceID := c.Param("serviceId")

	svc, err := h.serviceService.GetByID(c.Request.Context(), serviceID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": svc,
	})
}

// Update updates a service
func (h *ServiceHandler) Update(c *gin.Context) {
	projectID := c.Param("id")
	serviceName := c.Param("serviceName")

	var req service.UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	svc, err := h.serviceService.Update(c.Request.Context(), projectID, serviceName, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": svc,
	})
}

// Delete deletes a service
func (h *ServiceHandler) Delete(c *gin.Context) {
	projectID := c.Param("id")
	serviceName := c.Param("serviceName")

	if err := h.serviceService.Delete(c.Request.Context(), projectID, serviceName); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "service deleted",
	})
}
