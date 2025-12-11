package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	apperrors "github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// ServiceService handles service business logic
type ServiceService struct {
	store storage.Store
	log   logger.Logger
}

// NewServiceService creates a new service service
func NewServiceService(store storage.Store, log logger.Logger) *ServiceService {
	return &ServiceService{
		store: store,
		log:   log,
	}
}

// CreateServiceRequest represents a request to create a service
type CreateServiceRequest struct {
	Name            string            `json:"name" binding:"required"`
	Type            string            `json:"type"`             // web, worker, cron, database
	Builder         string            `json:"builder"`          // nixpacks, railpacks, dockerfile, docker_image, buildpacks
	GitRepo         string            `json:"git_repo"`         // override project's repo
	GitBranch       string            `json:"git_branch"`       // override project's branch
	Subdirectory    string            `json:"subdirectory"`     // for monorepos
	DockerImage     string            `json:"docker_image"`     // if builder=docker_image
	DatabaseType    string            `json:"database_type"`    // postgres, mysql, redis, mongodb
	DatabaseVersion string            `json:"database_version"` // version for database
	Port            int               `json:"port"`
	Command         string            `json:"command"`
	Environment     map[string]string `json:"environment"`
}

// ServiceResponse represents a service response
type ServiceResponse struct {
	ID              string            `json:"id"`
	ProjectID       string            `json:"project_id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	Builder         string            `json:"builder"`
	GitRepo         string            `json:"git_repo,omitempty"`
	GitBranch       string            `json:"git_branch,omitempty"`
	Subdirectory    string            `json:"subdirectory,omitempty"`
	DockerImage     string            `json:"docker_image,omitempty"`
	DatabaseType    string            `json:"database_type,omitempty"`
	DatabaseVersion string            `json:"database_version,omitempty"`
	// Database connection info (only for type=database)
	DatabaseHost     string `json:"database_host,omitempty"`
	DatabasePort     int    `json:"database_port,omitempty"`
	DatabaseUser     string `json:"database_user,omitempty"`
	DatabasePassword string `json:"database_password,omitempty"`
	DatabaseName     string `json:"database_name,omitempty"`
	DatabaseExposed  bool   `json:"database_exposed,omitempty"`
	Port             int               `json:"port"`
	Command          string            `json:"command,omitempty"`
	Environment      map[string]string `json:"environment"`
	Status           string            `json:"status"`
	CreatedAt        string            `json:"created_at"`
	UpdatedAt        string            `json:"updated_at"`
}

// Create creates a new service within a project
func (s *ServiceService) Create(ctx context.Context, projectID string, req CreateServiceRequest) (*ServiceResponse, error) {
	s.log.Info("creating service", "project_id", projectID, "name", req.Name)

	// Verify project exists
	project, err := s.store.Projects().GetByID(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		// Try by name
		project, err = s.store.Projects().GetByName(ctx, projectID)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", projectID)
	}

	// Check if service already exists in project
	existing, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, req.Name)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to check existing service", err)
	}
	if existing != nil {
		return nil, apperrors.NewConflictError("service with this name already exists in project")
	}

	// Set defaults
	serviceType := storage.ServiceType(req.Type)
	if serviceType == "" {
		serviceType = storage.ServiceTypeWeb
	}

	builder := storage.BuilderType(req.Builder)
	if builder == "" {
		if serviceType == storage.ServiceTypeDatabase {
			builder = "" // No builder for databases
		} else {
			builder = storage.BuilderNixpacks
		}
	}

	port := req.Port
	if port == 0 {
		port = 8080
	}

	subdirectory := req.Subdirectory
	if subdirectory == "" {
		subdirectory = "."
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

	service := &storage.Service{
		ID:              uuid.New().String(),
		ProjectID:       project.ID,
		Name:            req.Name,
		Type:            serviceType,
		Builder:         builder,
		GitRepo:         req.GitRepo,
		GitBranch:       req.GitBranch,
		Subdirectory:    subdirectory,
		DockerImage:     req.DockerImage,
		DatabaseType:    req.DatabaseType,
		DatabaseVersion: req.DatabaseVersion,
		Port:            port,
		Command:         req.Command,
		Environment:     envJSON,
		Status:          "stopped",
	}

	if err := s.store.Services().Create(ctx, service); err != nil {
		return nil, apperrors.NewInternalError("failed to create service", err)
	}

	s.log.Info("service created", "id", service.ID, "name", service.Name, "project_id", project.ID)

	return s.toResponse(service), nil
}

// Get retrieves a service by project and name
func (s *ServiceService) Get(ctx context.Context, projectID, serviceName string) (*ServiceResponse, error) {
	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceName)
	}

	return s.toResponse(service), nil
}

// GetByID retrieves a service by ID
func (s *ServiceService) GetByID(ctx context.Context, serviceID string) (*ServiceResponse, error) {
	service, err := s.store.Services().GetByID(ctx, serviceID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceID)
	}

	return s.toResponse(service), nil
}

// List returns all services for a project
func (s *ServiceService) List(ctx context.Context, projectID string) ([]*ServiceResponse, error) {
	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	services, err := s.store.Services().ListByProjectID(ctx, project.ID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list services", err)
	}

	responses := make([]*ServiceResponse, len(services))
	for i, svc := range services {
		responses[i] = s.toResponse(svc)
	}

	return responses, nil
}

// UpdateServiceRequest represents a request to update a service
type UpdateServiceRequest struct {
	Builder         *string           `json:"builder"`
	GitRepo         *string           `json:"git_repo"`
	GitBranch       *string           `json:"git_branch"`
	Subdirectory    *string           `json:"subdirectory"`
	DockerImage     *string           `json:"docker_image"`
	DatabaseVersion *string           `json:"database_version"`
	Port            *int              `json:"port"`
	Command         *string           `json:"command"`
	Environment     map[string]string `json:"environment"`
}

// Update updates a service
func (s *ServiceService) Update(ctx context.Context, projectID, serviceName string, req UpdateServiceRequest) (*ServiceResponse, error) {
	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceName)
	}

	// Apply updates
	if req.Builder != nil {
		service.Builder = storage.BuilderType(*req.Builder)
	}
	if req.GitRepo != nil {
		service.GitRepo = *req.GitRepo
	}
	if req.GitBranch != nil {
		service.GitBranch = *req.GitBranch
	}
	if req.Subdirectory != nil {
		service.Subdirectory = *req.Subdirectory
	}
	if req.DockerImage != nil {
		service.DockerImage = *req.DockerImage
	}
	if req.DatabaseVersion != nil {
		service.DatabaseVersion = *req.DatabaseVersion
	}
	if req.Port != nil {
		service.Port = *req.Port
	}
	if req.Command != nil {
		service.Command = *req.Command
	}
	if req.Environment != nil {
		data, err := json.Marshal(req.Environment)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to encode environment", err)
		}
		service.Environment = string(data)
	}

	if err := s.store.Services().Update(ctx, service); err != nil {
		return nil, apperrors.NewInternalError("failed to update service", err)
	}

	s.log.Info("service updated", "id", service.ID, "name", service.Name)

	return s.toResponse(service), nil
}

// Delete deletes a service
func (s *ServiceService) Delete(ctx context.Context, projectID, serviceName string) error {
	// Resolve project
	project, err := s.resolveProject(ctx, projectID)
	if err != nil {
		return err
	}

	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return apperrors.NewNotFoundError("service", serviceName)
	}

	// Delete related deployments and containers
	deployments, err := s.store.Deployments().ListByServiceID(ctx, service.ID)
	if err != nil {
		return apperrors.NewInternalError("failed to list deployments", err)
	}

	for _, d := range deployments {
		// Delete containers for this deployment
		if err := s.store.Containers().DeleteByDeploymentID(ctx, d.ID); err != nil {
			s.log.Warn("failed to delete containers for deployment", "deploymentID", d.ID, "error", err)
		}
		// Delete the deployment
		if err := s.store.Deployments().Delete(ctx, d.ID); err != nil {
			s.log.Warn("failed to delete deployment", "deploymentID", d.ID, "error", err)
		}
	}

	if err := s.store.Services().Delete(ctx, service.ID); err != nil {
		return apperrors.NewInternalError("failed to delete service", err)
	}

	s.log.Info("service deleted", "id", service.ID, "name", service.Name)

	return nil
}

// UpdateStatus updates the status of a service
func (s *ServiceService) UpdateStatus(ctx context.Context, serviceID, status string) error {
	service, err := s.store.Services().GetByID(ctx, serviceID)
	if err != nil {
		return apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return apperrors.NewNotFoundError("service", serviceID)
	}

	service.Status = status
	if err := s.store.Services().Update(ctx, service); err != nil {
		return apperrors.NewInternalError("failed to update service status", err)
	}

	return nil
}

func (s *ServiceService) resolveProject(ctx context.Context, projectIDOrName string) (*storage.Project, error) {
	project, err := s.store.Projects().GetByID(ctx, projectIDOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Projects().GetByName(ctx, projectIDOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", projectIDOrName)
	}
	return project, nil
}

func (s *ServiceService) toResponse(service *storage.Service) *ServiceResponse {
	var env map[string]string
	if service.Environment != "" {
		_ = json.Unmarshal([]byte(service.Environment), &env)
	}

	return &ServiceResponse{
		ID:               service.ID,
		ProjectID:        service.ProjectID,
		Name:             service.Name,
		Type:             string(service.Type),
		Builder:          string(service.Builder),
		GitRepo:          service.GitRepo,
		GitBranch:        service.GitBranch,
		Subdirectory:     service.Subdirectory,
		DockerImage:      service.DockerImage,
		DatabaseType:     service.DatabaseType,
		DatabaseVersion:  service.DatabaseVersion,
		DatabaseHost:     service.DatabaseHost,
		DatabasePort:     service.DatabasePort,
		DatabaseUser:     service.DatabaseUser,
		DatabasePassword: service.DatabasePassword,
		DatabaseName:     service.DatabaseName,
		DatabaseExposed:  service.DatabaseExposed,
		Port:             service.Port,
		Command:          service.Command,
		Environment:      env,
		Status:           service.Status,
		CreatedAt:        service.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        service.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
