package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/core/deployer"
	apperrors "github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// AppService handles application business logic
type AppService struct {
	store storage.Store
	log   logger.Logger
}

// NewAppService creates a new app service
func NewAppService(store storage.Store, log logger.Logger) *AppService {
	return &AppService{
		store: store,
		log:   log,
	}
}

// CreateAppRequest represents a request to create an application
type CreateAppRequest struct {
	Name           string                  `json:"name" binding:"required"`
	DeploymentMode deployer.DeploymentMode `json:"deployment_mode" binding:"required"`
	Domain         string                  `json:"domain"`
	Environment    map[string]string       `json:"environment"`
}

// AppResponse represents an application response
type AppResponse struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	DeploymentMode deployer.DeploymentMode `json:"deployment_mode"`
	Domain         string                  `json:"domain"`
	Environment    map[string]string       `json:"environment"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
}

// Create creates a new application
func (s *AppService) Create(ctx context.Context, req CreateAppRequest) (*AppResponse, error) {
	s.log.Info("creating application", "name", req.Name, "mode", req.DeploymentMode)

	// Check if app already exists
	existing, err := s.store.Apps().GetByName(ctx, req.Name)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to check existing app", err)
	}
	if existing != nil {
		return nil, apperrors.NewConflictError("application with this name already exists")
	}

	// Validate deployment mode
	if req.DeploymentMode != deployer.ModeGit &&
		req.DeploymentMode != deployer.ModeImage &&
		req.DeploymentMode != deployer.ModeCompose {
		return nil, apperrors.NewValidationError("invalid deployment mode", map[string]interface{}{
			"deployment_mode": req.DeploymentMode,
			"allowed":         []string{string(deployer.ModeGit), string(deployer.ModeImage), string(deployer.ModeCompose)},
		})
	}

	// Encode environment
	envJSON := "{}"
	if len(req.Environment) > 0 {
		data, err := json.Marshal(req.Environment)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to encode environment", err)
		}
		envJSON = string(data)
	}

	app := &storage.Application{
		ID:             uuid.New().String(),
		Name:           req.Name,
		DeploymentMode: string(req.DeploymentMode),
		Domain:         req.Domain,
		Environment:    envJSON,
	}

	if err := s.store.Apps().Create(ctx, app); err != nil {
		return nil, apperrors.NewInternalError("failed to create application", err)
	}

	s.log.Info("application created", "id", app.ID, "name", app.Name)

	return s.toResponse(app), nil
}

// Get retrieves an application by ID or name
func (s *AppService) Get(ctx context.Context, idOrName string) (*AppResponse, error) {
	// Try by ID first
	app, err := s.store.Apps().GetByID(ctx, idOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get application", err)
	}

	// If not found by ID, try by name
	if app == nil {
		app, err = s.store.Apps().GetByName(ctx, idOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get application", err)
		}
	}

	if app == nil {
		return nil, apperrors.NewNotFoundError("application", idOrName)
	}

	return s.toResponse(app), nil
}

// GetByName retrieves an application by name
func (s *AppService) GetByName(ctx context.Context, name string) (*AppResponse, error) {
	app, err := s.store.Apps().GetByName(ctx, name)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get application", err)
	}
	if app == nil {
		return nil, apperrors.NewNotFoundError("application", name)
	}

	return s.toResponse(app), nil
}

// List returns all applications
func (s *AppService) List(ctx context.Context) ([]*AppResponse, error) {
	apps, err := s.store.Apps().List(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list applications", err)
	}

	responses := make([]*AppResponse, len(apps))
	for i, app := range apps {
		responses[i] = s.toResponse(app)
	}

	return responses, nil
}

// UpdateAppRequest represents a request to update an application
type UpdateAppRequest struct {
	Domain      *string           `json:"domain"`
	Environment map[string]string `json:"environment"`
}

// Update updates an application
func (s *AppService) Update(ctx context.Context, idOrName string, req UpdateAppRequest) (*AppResponse, error) {
	// Try by ID first
	app, err := s.store.Apps().GetByID(ctx, idOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get application", err)
	}
	// If not found by ID, try by name
	if app == nil {
		app, err = s.store.Apps().GetByName(ctx, idOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get application", err)
		}
	}
	if app == nil {
		return nil, apperrors.NewNotFoundError("application", idOrName)
	}

	if req.Domain != nil {
		app.Domain = *req.Domain
	}

	if req.Environment != nil {
		data, err := json.Marshal(req.Environment)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to encode environment", err)
		}
		app.Environment = string(data)
	}

	if err := s.store.Apps().Update(ctx, app); err != nil {
		return nil, apperrors.NewInternalError("failed to update application", err)
	}

	s.log.Info("application updated", "id", app.ID)

	return s.toResponse(app), nil
}

// Delete deletes an application
func (s *AppService) Delete(ctx context.Context, idOrName string) error {
	// Try by ID first
	app, err := s.store.Apps().GetByID(ctx, idOrName)
	if err != nil {
		return apperrors.NewInternalError("failed to get application", err)
	}
	// If not found by ID, try by name
	if app == nil {
		app, err = s.store.Apps().GetByName(ctx, idOrName)
		if err != nil {
			return apperrors.NewInternalError("failed to get application", err)
		}
	}
	if app == nil {
		return apperrors.NewNotFoundError("application", idOrName)
	}

	// TODO: Stop and remove containers, delete routes

	if err := s.store.Apps().Delete(ctx, app.ID); err != nil {
		return apperrors.NewInternalError("failed to delete application", err)
	}

	s.log.Info("application deleted", "id", app.ID)

	return nil
}

func (s *AppService) toResponse(app *storage.Application) *AppResponse {
	var env map[string]string
	if app.Environment != "" {
		json.Unmarshal([]byte(app.Environment), &env)
	}

	return &AppResponse{
		ID:             app.ID,
		Name:           app.Name,
		DeploymentMode: deployer.DeploymentMode(app.DeploymentMode),
		Domain:         app.Domain,
		Environment:    env,
		CreatedAt:      app.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      app.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
