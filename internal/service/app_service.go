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

// AppService handles application/project business logic
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

// CreateAppRequest represents a request to create an application/project
type CreateAppRequest struct {
	Name           string                  `json:"name" binding:"required"`
	DisplayName    string                  `json:"display_name"`
	Description    string                  `json:"description"`
	DeploymentMode deployer.DeploymentMode `json:"deployment_mode"` // legacy, used to auto-create service
	Domain         string                  `json:"domain"`          // legacy
	GitRepo        string                  `json:"git_repo"`
	GitBranch      string                  `json:"git_branch"`
	DockerImage    string                  `json:"docker_image"` // legacy
	Environment    map[string]string       `json:"environment"`
}

// AppResponse represents an application/project response
type AppResponse struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	DisplayName    string                  `json:"display_name,omitempty"`
	Description    string                  `json:"description,omitempty"`
	DeploymentMode deployer.DeploymentMode `json:"deployment_mode"` // legacy compatibility
	Domain         string                  `json:"domain"`          // legacy compatibility
	GitRepo        string                  `json:"git_repo,omitempty"`
	GitBranch      string                  `json:"git_branch,omitempty"`
	DockerImage    string                  `json:"docker_image,omitempty"` // legacy compatibility
	Environment    map[string]string       `json:"environment"`
	CreatedAt      string                  `json:"created_at"`
	UpdatedAt      string                  `json:"updated_at"`
}

// Create creates a new application/project
func (s *AppService) Create(ctx context.Context, req CreateAppRequest) (*AppResponse, error) {
	s.log.Info("creating project", "name", req.Name)

	// Check if project already exists
	existing, err := s.store.Apps().GetByName(ctx, req.Name)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to check existing project", err)
	}
	if existing != nil {
		return nil, apperrors.NewConflictError("project with this name already exists")
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

	project := &storage.Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		GitRepo:     req.GitRepo,
		GitBranch:   req.GitBranch,
		Environment: envJSON,
	}

	if err := s.store.Apps().Create(ctx, project); err != nil {
		return nil, apperrors.NewInternalError("failed to create project", err)
	}

	s.log.Info("project created", "id", project.ID, "name", project.Name)

	return s.toResponse(ctx, project), nil
}

// Get retrieves an application/project by ID or name
func (s *AppService) Get(ctx context.Context, idOrName string) (*AppResponse, error) {
	// Try by ID first
	project, err := s.store.Apps().GetByID(ctx, idOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}

	// If not found by ID, try by name
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, idOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}

	if project == nil {
		return nil, apperrors.NewNotFoundError("project", idOrName)
	}

	return s.toResponse(ctx, project), nil
}

// GetByName retrieves an application/project by name
func (s *AppService) GetByName(ctx context.Context, name string) (*AppResponse, error) {
	project, err := s.store.Apps().GetByName(ctx, name)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", name)
	}

	return s.toResponse(ctx, project), nil
}

// List returns all applications/projects
func (s *AppService) List(ctx context.Context) ([]*AppResponse, error) {
	projects, err := s.store.Apps().List(ctx)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list projects", err)
	}

	responses := make([]*AppResponse, len(projects))
	for i, project := range projects {
		responses[i] = s.toResponse(ctx, project)
	}

	return responses, nil
}

// UpdateAppRequest represents a request to update an application/project
type UpdateAppRequest struct {
	DisplayName *string           `json:"display_name"`
	Description *string           `json:"description"`
	Domain      *string           `json:"domain"` // legacy
	GitRepo     *string           `json:"git_repo"`
	GitBranch   *string           `json:"git_branch"`
	Environment map[string]string `json:"environment"`
}

// Update updates an application/project
func (s *AppService) Update(ctx context.Context, idOrName string, req UpdateAppRequest) (*AppResponse, error) {
	// Try by ID first
	project, err := s.store.Apps().GetByID(ctx, idOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	// If not found by ID, try by name
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, idOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", idOrName)
	}

	if req.DisplayName != nil {
		project.DisplayName = *req.DisplayName
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.GitRepo != nil {
		project.GitRepo = *req.GitRepo
	}
	if req.GitBranch != nil {
		project.GitBranch = *req.GitBranch
	}

	if req.Environment != nil {
		data, err := json.Marshal(req.Environment)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to encode environment", err)
		}
		project.Environment = string(data)
	}

	if err := s.store.Apps().Update(ctx, project); err != nil {
		return nil, apperrors.NewInternalError("failed to update project", err)
	}

	s.log.Info("project updated", "id", project.ID)

	return s.toResponse(ctx, project), nil
}

// Delete deletes an application/project
func (s *AppService) Delete(ctx context.Context, idOrName string) error {
	// Try by ID first
	project, err := s.store.Apps().GetByID(ctx, idOrName)
	if err != nil {
		return apperrors.NewInternalError("failed to get project", err)
	}
	// If not found by ID, try by name
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, idOrName)
		if err != nil {
			return apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return apperrors.NewNotFoundError("project", idOrName)
	}

	// TODO: Stop and remove containers, delete routes

	if err := s.store.Apps().Delete(ctx, project.ID); err != nil {
		return apperrors.NewInternalError("failed to delete project", err)
	}

	s.log.Info("project deleted", "id", project.ID)

	return nil
}

func (s *AppService) toResponse(ctx context.Context, project *storage.Project) *AppResponse {
	var env map[string]string
	if project.Environment != "" {
		_ = json.Unmarshal([]byte(project.Environment), &env)
	}

	response := &AppResponse{
		ID:          project.ID,
		Name:        project.Name,
		DisplayName: project.DisplayName,
		Description: project.Description,
		GitRepo:     project.GitRepo,
		GitBranch:   project.GitBranch,
		Environment: env,
		CreatedAt:   project.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   project.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Legacy compatibility: get info from "main" service if exists
	mainService, _ := s.store.Services().GetByProjectIDAndName(ctx, project.ID, "main")
	if mainService != nil {
		response.DockerImage = mainService.DockerImage
		if mainService.Builder == storage.BuilderDockerImage {
			response.DeploymentMode = deployer.ModeImage
		} else {
			response.DeploymentMode = deployer.ModeGit
		}
	}

	// Legacy: get domain from domains table
	domains, _ := s.store.Domains().ListByProjectID(ctx, project.ID)
	if len(domains) > 0 {
		response.Domain = domains[0].Domain
	}

	return response
}
