package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/service"
)

// DomainHandler handles domain endpoints
type DomainHandler struct {
	domainService *service.DomainService
	log           logger.Logger
}

// NewDomainHandler creates a new domain handler
func NewDomainHandler(domainService *service.DomainService, log logger.Logger) *DomainHandler {
	return &DomainHandler{
		domainService: domainService,
		log:           log,
	}
}

// ListByProject returns all domains for a project
func (h *DomainHandler) ListByProject(c *gin.Context) {
	projectID := c.Param("id")

	domains, err := h.domainService.ListByProject(c.Request.Context(), projectID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": domains,
	})
}

// ListByService returns all domains for a service
func (h *DomainHandler) ListByService(c *gin.Context) {
	projectID := c.Param("id")
	serviceName := c.Param("serviceName")

	domains, err := h.domainService.ListByService(c.Request.Context(), projectID, serviceName)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": domains,
	})
}

// Create creates a new domain for a service
func (h *DomainHandler) Create(c *gin.Context) {
	projectID := c.Param("id")
	serviceName := c.Param("serviceName")

	var req service.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	domain, err := h.domainService.Create(c.Request.Context(), projectID, serviceName, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": domain,
	})
}

// Get retrieves a domain by domain name
func (h *DomainHandler) Get(c *gin.Context) {
	domainName := c.Param("domain")

	domain, err := h.domainService.Get(c.Request.Context(), domainName)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": domain,
	})
}

// Update updates a domain
func (h *DomainHandler) Update(c *gin.Context) {
	domainName := c.Param("domain")

	var req service.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	domain, err := h.domainService.Update(c.Request.Context(), domainName, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": domain,
	})
}

// Delete deletes a domain
func (h *DomainHandler) Delete(c *gin.Context) {
	domainName := c.Param("domain")

	if err := h.domainService.Delete(c.Request.Context(), domainName); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "domain deleted",
	})
}
