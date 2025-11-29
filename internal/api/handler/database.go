package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/service"
)

type DatabaseHandler struct {
	dbService *service.DatabaseService
}

func NewDatabaseHandler(dbService *service.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{
		dbService: dbService,
	}
}

type CreateDatabaseRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Version string `json:"version,omitempty"`
	AppID   string `json:"app_id,omitempty"`
}

type DatabaseResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Username      string `json:"username"`
	Password      string `json:"password,omitempty"`
	Database      string `json:"database"`
	ConnectionURL string `json:"connection_url,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// Create handles POST /api/v1/databases
func (h *DatabaseHandler) Create(c *gin.Context) {
	var req CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.Create(c.Request.Context(), service.CreateDatabaseInput{
		Name:    req.Name,
		Type:    service.DatabaseType(req.Type),
		Version: req.Version,
		AppID:   req.AppID,
	})
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": toDatabaseResponse(db, true),
	})
}

// List handles GET /api/v1/databases
func (h *DatabaseHandler) List(c *gin.Context) {
	dbs, err := h.dbService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]DatabaseResponse, len(dbs))
	for i, db := range dbs {
		result[i] = toDatabaseResponse(db, false)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

// Get handles GET /api/v1/databases/:name
func (h *DatabaseHandler) Get(c *gin.Context) {
	name := c.Param("name")

	db, err := h.dbService.Get(c.Request.Context(), name)
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	// Check if requesting credentials
	showCredentials := c.Query("credentials") == "true"

	c.JSON(http.StatusOK, gin.H{
		"data": toDatabaseResponse(db, showCredentials),
	})
}

// Delete handles DELETE /api/v1/databases/:name
func (h *DatabaseHandler) Delete(c *gin.Context) {
	name := c.Param("name")

	if err := h.dbService.Delete(c.Request.Context(), name); err != nil {
		handleDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "database deleted",
	})
}

// GetStatus handles GET /api/v1/databases/:name/status
func (h *DatabaseHandler) GetStatus(c *gin.Context) {
	name := c.Param("name")

	status, err := h.dbService.GetStatus(c.Request.Context(), name)
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status": status,
		},
	})
}

// Restart handles POST /api/v1/databases/:name/restart
func (h *DatabaseHandler) Restart(c *gin.Context) {
	name := c.Param("name")

	if err := h.dbService.Restart(c.Request.Context(), name); err != nil {
		handleDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "database restarted",
	})
}

// GetCredentials handles GET /api/v1/databases/:name/credentials
func (h *DatabaseHandler) GetCredentials(c *gin.Context) {
	name := c.Param("name")

	db, err := h.dbService.Get(c.Request.Context(), name)
	if err != nil {
		handleDatabaseError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"host":           db.Host,
			"port":           db.Port,
			"username":       db.Username,
			"password":       db.Password,
			"database":       db.Database,
			"connection_url": db.ConnectionURL,
		},
	})
}

func toDatabaseResponse(db *service.DatabaseInfo, showCredentials bool) DatabaseResponse {
	resp := DatabaseResponse{
		ID:        db.ID,
		Name:      db.Name,
		Type:      string(db.Type),
		Host:      db.Host,
		Port:      db.Port,
		Username:  db.Username,
		Database:  db.Database,
		Status:    db.Status,
		CreatedAt: db.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if showCredentials {
		resp.Password = db.Password
		resp.ConnectionURL = db.ConnectionURL
	}

	return resp
}

func handleDatabaseError(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		switch appErr.Type {
		case errors.ErrTypeNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": appErr.Message})
		case errors.ErrTypeValidation:
			c.JSON(http.StatusBadRequest, gin.H{"error": appErr.Message})
		case errors.ErrTypeConflict:
			c.JSON(http.StatusConflict, gin.H{"error": appErr.Message})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": appErr.Message})
		}
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
