package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/core/deployer"
	apperrors "github.com/victalejo/nebula/internal/core/errors"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/proxy"
	"github.com/victalejo/nebula/internal/core/storage"
)

// DeployService handles deployment business logic
type DeployService struct {
	store        storage.Store
	registry     *deployer.DeployerRegistry
	proxyManager proxy.ProxyManager
	log          logger.Logger
}

// NewDeployService creates a new deploy service
func NewDeployService(
	store storage.Store,
	registry *deployer.DeployerRegistry,
	proxyManager proxy.ProxyManager,
	log logger.Logger,
) *DeployService {
	return &DeployService{
		store:        store,
		registry:     registry,
		proxyManager: proxyManager,
		log:          log,
	}
}

// DeployImageRequest represents a request to deploy from Docker image
type DeployImageRequest struct {
	Image        string            `json:"image" binding:"required"`
	Port         int               `json:"port" binding:"required"`
	Registry     string            `json:"registry"`
	RegistryAuth *RegistryAuthReq  `json:"registry_auth"`
	PullPolicy   string            `json:"pull_policy"`
	Environment  map[string]string `json:"environment"`
}

// DeployGitRequest represents a request to deploy from Git repository
type DeployGitRequest struct {
	Branch      string            `json:"branch"`
	Environment map[string]string `json:"environment"`
}

// DeployServiceRequest represents a request to deploy a specific service
type DeployServiceRequest struct {
	Environment map[string]string `json:"environment"`
}

// RegistryAuthReq represents registry authentication
type RegistryAuthReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeploymentResponse represents a deployment response
type DeploymentResponse struct {
	ID           string `json:"id"`
	AppID        string `json:"app_id"`
	ServiceID    string `json:"service_id,omitempty"`
	Version      string `json:"version"`
	Slot         string `json:"slot"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

// DeployImage deploys an application from a Docker image
func (s *DeployService) DeployImage(ctx context.Context, appID string, req DeployImageRequest) (*DeploymentResponse, error) {
	s.log.Info("starting image deployment",
		"app_id", appID,
		"image", req.Image,
	)

	// Get the project - try by ID first, then by name
	project, err := s.store.Apps().GetByID(ctx, appID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, appID)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", appID)
	}

	// Get main service and check if it's configured for image deployment
	mainService, _ := s.store.Services().GetByProjectIDAndName(ctx, project.ID, "main")
	if mainService != nil && mainService.Builder != storage.BuilderDockerImage {
		return nil, apperrors.NewValidationError("service is not configured for image deployment", nil)
	}

	// Get the image deployer
	imageDeployer, err := s.registry.Get(deployer.ModeImage)
	if err != nil {
		return nil, apperrors.NewInternalError("image deployer not available", err)
	}

	// Determine target slot (opposite of current active)
	targetSlot := s.getTargetSlot(ctx, project.ID)

	// Build source config
	sourceConfig := deployer.SourceConfig{
		Image:      req.Image,
		Port:       req.Port,
		PullPolicy: req.PullPolicy,
	}

	if req.RegistryAuth != nil {
		sourceConfig.RegistryAuth = &deployer.RegistryAuth{
			Registry: req.Registry,
			Username: req.RegistryAuth.Username,
			Password: req.RegistryAuth.Password,
		}
	}

	// Merge environment variables
	env := make(map[string]string)
	if project.Environment != "" {
		json.Unmarshal([]byte(project.Environment), &env)
	}
	for k, v := range req.Environment {
		env[k] = v
	}

	// Create deployment spec
	spec := &deployer.DeploymentSpec{
		AppID:       project.ID,
		Source:      sourceConfig,
		Environment: env,
		TargetSlot:  targetSlot,
	}

	// Validate
	if err := imageDeployer.Validate(ctx, spec); err != nil {
		return nil, apperrors.NewValidationError("invalid deployment spec", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create deployment record
	sourceJSON, _ := json.Marshal(sourceConfig)
	envJSON, _ := json.Marshal(env)

	serviceID := ""
	if mainService != nil {
		serviceID = mainService.ID
	}

	deployment := &storage.Deployment{
		ID:           uuid.New().String(),
		AppID:        project.ID,
		ServiceID:    serviceID,
		Version:      fmt.Sprintf("v%d", time.Now().Unix()),
		Slot:         string(targetSlot),
		Status:       string(deployer.StatusPending),
		SourceConfig: string(sourceJSON),
		Environment:  string(envJSON),
	}

	if err := s.store.Deployments().Create(ctx, deployment); err != nil {
		return nil, apperrors.NewInternalError("failed to create deployment record", err)
	}

	// Execute deployment asynchronously
	go s.executeDeployment(context.Background(), project, deployment, imageDeployer, spec)

	return &DeploymentResponse{
		ID:        deployment.ID,
		AppID:     deployment.AppID,
		ServiceID: deployment.ServiceID,
		Version:   deployment.Version,
		Slot:      deployment.Slot,
		Status:    deployment.Status,
		CreatedAt: deployment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// DeployGit deploys an application from a Git repository
func (s *DeployService) DeployGit(ctx context.Context, appID string, req DeployGitRequest) (*DeploymentResponse, error) {
	s.log.Info("starting git deployment", "app_id", appID)

	// Get the project - try by ID first, then by name
	project, err := s.store.Apps().GetByID(ctx, appID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, appID)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", appID)
	}

	// Get main service and check builder
	mainService, _ := s.store.Services().GetByProjectIDAndName(ctx, project.ID, "main")
	if mainService != nil && mainService.Builder == storage.BuilderDockerImage {
		return nil, apperrors.NewValidationError("service is not configured for git deployment", nil)
	}

	// Determine git repo - from service, then project
	gitRepo := project.GitRepo
	gitBranch := project.GitBranch
	if mainService != nil {
		if mainService.GitRepo != "" {
			gitRepo = mainService.GitRepo
		}
		if mainService.GitBranch != "" {
			gitBranch = mainService.GitBranch
		}
	}

	// Validate git repo is configured
	if gitRepo == "" {
		return nil, apperrors.NewValidationError("git repository URL is not configured", nil)
	}

	// Get the git deployer
	gitDeployer, err := s.registry.Get(deployer.ModeGit)
	if err != nil {
		return nil, apperrors.NewInternalError("git deployer not available", err)
	}

	// Determine target slot
	targetSlot := s.getTargetSlot(ctx, project.ID)

	// Determine branch
	branch := req.Branch
	if branch == "" {
		branch = gitBranch
	}
	if branch == "" {
		branch = "main"
	}

	// Merge environment variables
	env := make(map[string]string)
	if project.Environment != "" {
		json.Unmarshal([]byte(project.Environment), &env)
	}
	for k, v := range req.Environment {
		env[k] = v
	}

	// Create deployment spec
	spec := &deployer.DeploymentSpec{
		AppID:       project.ID,
		AppName:     project.Name,
		GitRepo:     gitRepo,
		GitBranch:   branch,
		Environment: env,
		TargetSlot:  targetSlot,
	}

	// Validate
	if err := gitDeployer.Validate(ctx, spec); err != nil {
		return nil, apperrors.NewValidationError("invalid deployment spec", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create deployment record
	sourceJSON, _ := json.Marshal(map[string]string{
		"git_repo":   gitRepo,
		"git_branch": branch,
	})
	envJSON, _ := json.Marshal(env)

	serviceID := ""
	if mainService != nil {
		serviceID = mainService.ID
	}

	deployment := &storage.Deployment{
		ID:           uuid.New().String(),
		AppID:        project.ID,
		ServiceID:    serviceID,
		Version:      fmt.Sprintf("v%d", time.Now().Unix()),
		Slot:         string(targetSlot),
		Status:       string(deployer.StatusPending),
		SourceConfig: string(sourceJSON),
		Environment:  string(envJSON),
	}

	if err := s.store.Deployments().Create(ctx, deployment); err != nil {
		return nil, apperrors.NewInternalError("failed to create deployment record", err)
	}

	// Execute deployment asynchronously
	go s.executeDeployment(context.Background(), project, deployment, gitDeployer, spec)

	return &DeploymentResponse{
		ID:        deployment.ID,
		AppID:     deployment.AppID,
		ServiceID: deployment.ServiceID,
		Version:   deployment.Version,
		Slot:      deployment.Slot,
		Status:    deployment.Status,
		CreatedAt: deployment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// executeDeployment runs the deployment process
func (s *DeployService) executeDeployment(
	ctx context.Context,
	project *storage.Project,
	deployment *storage.Deployment,
	dep deployer.Deployer,
	spec *deployer.DeploymentSpec,
) {
	s.log.Info("executing deployment",
		"deployment_id", deployment.ID,
		"app_id", project.ID,
	)

	// Update status to preparing
	now := time.Now()
	deployment.Status = string(deployer.StatusPreparing)
	deployment.StartedAt = &now
	s.store.Deployments().Update(ctx, deployment)

	// Prepare (pull image)
	_, err := dep.Prepare(ctx, spec)
	if err != nil {
		s.failDeployment(ctx, deployment, err)
		return
	}

	// Update status to deploying
	deployment.Status = string(deployer.StatusDeploying)
	s.store.Deployments().Update(ctx, deployment)

	// Deploy (create and start container)
	result, err := dep.Deploy(ctx, spec)
	if err != nil {
		s.failDeployment(ctx, deployment, err)
		return
	}

	// Store container info
	for _, containerID := range result.ContainerIDs {
		port := 0
		if p, ok := result.Ports["main"]; ok {
			port = p
		}

		container := &storage.Container{
			ID:           uuid.New().String(),
			DeploymentID: deployment.ID,
			ContainerID:  containerID,
			Name:         fmt.Sprintf("%s-%s", project.Name, spec.TargetSlot),
			Status:       "running",
			Port:         port,
		}
		s.store.Containers().Create(ctx, container)
	}

	// Health check
	healthResult, err := dep.HealthCheck(ctx, result)
	if err != nil || !healthResult.Healthy {
		errMsg := "health check failed"
		if err != nil {
			errMsg = err.Error()
		}
		s.failDeployment(ctx, deployment, fmt.Errorf(errMsg))

		// Cleanup failed deployment
		dep.Destroy(ctx, result.ContainerIDs)
		return
	}

	// Update route to point to new slot
	// Check if project has domains configured
	domains, _ := s.store.Domains().ListByProjectID(ctx, project.ID)
	if len(domains) > 0 {
		mainPort := 0
		if p, ok := result.Ports["main"]; ok {
			mainPort = p
		}

		for _, domain := range domains {
			route := proxy.Route{
				Domain: domain.Domain,
				AppID:  project.ID,
				BlueTarget: &proxy.Upstream{
					Host: "localhost",
					Port: mainPort,
				},
				ActiveSlot: proxy.Slot(spec.TargetSlot),
				SSLEnabled: domain.SSLEnabled,
			}

			if err := s.proxyManager.AddRoute(ctx, route); err != nil {
				s.log.Error("failed to update route", "error", err, "domain", domain.Domain)
			}
		}
	}

	// Mark deployment as running
	finishedAt := time.Now()
	deployment.Status = string(deployer.StatusRunning)
	deployment.FinishedAt = &finishedAt
	s.store.Deployments().Update(ctx, deployment)

	// Update route's active slot (legacy)
	if route, _ := s.store.Routes().GetByAppID(ctx, project.ID); route != nil {
		route.ActiveSlot = string(spec.TargetSlot)
		s.store.Routes().Update(ctx, route)
	}

	// Update domains' active slot
	for _, domain := range domains {
		domain.ActiveSlot = string(spec.TargetSlot)
		s.store.Domains().Update(ctx, domain)
	}

	// Stop old deployment (if exists)
	s.stopOldDeployment(ctx, project.ID, string(spec.TargetSlot.Opposite()), dep)

	s.log.Info("deployment completed successfully",
		"deployment_id", deployment.ID,
		"app_id", project.ID,
	)
}

// failDeployment marks a deployment as failed
func (s *DeployService) failDeployment(ctx context.Context, deployment *storage.Deployment, err error) {
	s.log.Error("deployment failed",
		"deployment_id", deployment.ID,
		"error", err,
	)

	finishedAt := time.Now()
	deployment.Status = string(deployer.StatusFailed)
	deployment.ErrorMessage = err.Error()
	deployment.FinishedAt = &finishedAt
	s.store.Deployments().Update(ctx, deployment)
}

// stopOldDeployment stops the old deployment
func (s *DeployService) stopOldDeployment(ctx context.Context, appID string, slot string, dep deployer.Deployer) {
	oldDeployment, err := s.store.Deployments().GetByAppIDAndSlot(ctx, appID, slot)
	if err != nil || oldDeployment == nil {
		return
	}

	containers, err := s.store.Containers().ListByDeploymentID(ctx, oldDeployment.ID)
	if err != nil {
		return
	}

	containerIDs := make([]string, len(containers))
	for i, c := range containers {
		containerIDs[i] = c.ContainerID
	}

	if err := dep.Stop(ctx, containerIDs); err != nil {
		s.log.Warn("failed to stop old containers", "error", err)
	}

	// Update old deployment status
	oldDeployment.Status = string(deployer.StatusStopped)
	s.store.Deployments().Update(ctx, oldDeployment)
}

// getTargetSlot determines which slot to deploy to
func (s *DeployService) getTargetSlot(ctx context.Context, appID string) deployer.Slot {
	route, _ := s.store.Routes().GetByAppID(ctx, appID)
	if route == nil || route.ActiveSlot == "" {
		return deployer.SlotBlue
	}

	if route.ActiveSlot == string(deployer.SlotBlue) {
		return deployer.SlotGreen
	}
	return deployer.SlotBlue
}

// GetDeployment retrieves a deployment by ID
func (s *DeployService) GetDeployment(ctx context.Context, id string) (*DeploymentResponse, error) {
	deployment, err := s.store.Deployments().GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get deployment", err)
	}
	if deployment == nil {
		return nil, apperrors.NewNotFoundError("deployment", id)
	}

	return &DeploymentResponse{
		ID:        deployment.ID,
		AppID:     deployment.AppID,
		ServiceID: deployment.ServiceID,
		Version:   deployment.Version,
		Slot:      deployment.Slot,
		Status:    deployment.Status,
		CreatedAt: deployment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// ListDeployments returns all deployments for an application
func (s *DeployService) ListDeployments(ctx context.Context, appIDOrName string) ([]*DeploymentResponse, error) {
	// Resolve app ID from name if needed
	project, err := s.store.Apps().GetByID(ctx, appIDOrName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, appIDOrName)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", appIDOrName)
	}

	deployments, err := s.store.Deployments().ListByAppID(ctx, project.ID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list deployments", err)
	}

	responses := make([]*DeploymentResponse, len(deployments))
	for i, d := range deployments {
		responses[i] = &DeploymentResponse{
			ID:        d.ID,
			AppID:     d.AppID,
			ServiceID: d.ServiceID,
			Version:   d.Version,
			Slot:      d.Slot,
			Status:    d.Status,
			CreatedAt: d.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return responses, nil
}

// ListServiceDeployments returns all deployments for a specific service
func (s *DeployService) ListServiceDeployments(ctx context.Context, projectID, serviceName string) ([]*DeploymentResponse, error) {
	// Get the project
	project, err := s.store.Apps().GetByID(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, projectID)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", projectID)
	}

	// Get the service
	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceName)
	}

	// Get deployments for this service
	deployments, err := s.store.Deployments().ListByServiceID(ctx, service.ID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to list deployments", err)
	}

	responses := make([]*DeploymentResponse, len(deployments))
	for i, d := range deployments {
		responses[i] = &DeploymentResponse{
			ID:           d.ID,
			AppID:        d.AppID,
			ServiceID:    d.ServiceID,
			Version:      d.Version,
			Slot:         d.Slot,
			Status:       d.Status,
			ErrorMessage: d.ErrorMessage,
			CreatedAt:    d.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if d.FinishedAt != nil {
			responses[i].FinishedAt = d.FinishedAt.Format("2006-01-02T15:04:05Z")
		}
	}

	return responses, nil
}

// DeployServiceByName deploys a specific service by its name
func (s *DeployService) DeployServiceByName(ctx context.Context, projectID, serviceName string, req DeployServiceRequest) (*DeploymentResponse, error) {
	s.log.Info("starting service deployment",
		"project_id", projectID,
		"service_name", serviceName,
	)

	// Get the project
	project, err := s.store.Apps().GetByID(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get project", err)
	}
	if project == nil {
		project, err = s.store.Apps().GetByName(ctx, projectID)
		if err != nil {
			return nil, apperrors.NewInternalError("failed to get project", err)
		}
	}
	if project == nil {
		return nil, apperrors.NewNotFoundError("project", projectID)
	}

	// Get the service
	service, err := s.store.Services().GetByProjectIDAndName(ctx, project.ID, serviceName)
	if err != nil {
		return nil, apperrors.NewInternalError("failed to get service", err)
	}
	if service == nil {
		return nil, apperrors.NewNotFoundError("service", serviceName)
	}

	// Handle database type differently
	if service.Type == storage.ServiceTypeDatabase {
		return s.deployDatabaseService(ctx, project, service, req)
	}

	// Determine deployer based on builder
	var dep deployer.Deployer
	var spec *deployer.DeploymentSpec

	targetSlot := s.getTargetSlotForService(ctx, service.ID)

	// Merge environment variables
	env := make(map[string]string)
	if project.Environment != "" {
		json.Unmarshal([]byte(project.Environment), &env)
	}
	if service.Environment != "" {
		var svcEnv map[string]string
		json.Unmarshal([]byte(service.Environment), &svcEnv)
		for k, v := range svcEnv {
			env[k] = v
		}
	}
	for k, v := range req.Environment {
		env[k] = v
	}

	switch service.Builder {
	case storage.BuilderDockerImage:
		dep, err = s.registry.Get(deployer.ModeImage)
		if err != nil {
			return nil, apperrors.NewInternalError("image deployer not available", err)
		}
		spec = &deployer.DeploymentSpec{
			AppID:      project.ID,
			AppName:    project.Name,
			ServiceID:  service.ID,
			Source: deployer.SourceConfig{
				Image: service.DockerImage,
				Port:  service.Port,
			},
			Environment: env,
			TargetSlot:  targetSlot,
		}

	case storage.BuilderNixpacks, storage.BuilderDockerfile, storage.BuilderBuildpacks:
		dep, err = s.registry.Get(deployer.ModeGit)
		if err != nil {
			return nil, apperrors.NewInternalError("git deployer not available", err)
		}

		gitRepo := service.GitRepo
		if gitRepo == "" {
			gitRepo = project.GitRepo
		}
		gitBranch := service.GitBranch
		if gitBranch == "" {
			gitBranch = project.GitBranch
		}
		if gitBranch == "" {
			gitBranch = "main"
		}

		if gitRepo == "" {
			return nil, apperrors.NewValidationError("git repository URL is not configured", nil)
		}

		spec = &deployer.DeploymentSpec{
			AppID:       project.ID,
			AppName:     project.Name,
			ServiceID:   service.ID,
			GitRepo:     gitRepo,
			GitBranch:   gitBranch,
			Environment: env,
			TargetSlot:  targetSlot,
		}

	default:
		return nil, apperrors.NewValidationError("unsupported builder type", map[string]interface{}{
			"builder": service.Builder,
		})
	}

	// Validate
	if err := dep.Validate(ctx, spec); err != nil {
		return nil, apperrors.NewValidationError("invalid deployment spec", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create deployment record
	sourceJSON, _ := json.Marshal(spec.Source)
	envJSON, _ := json.Marshal(env)

	deployment := &storage.Deployment{
		ID:           uuid.New().String(),
		AppID:        project.ID,
		ServiceID:    service.ID,
		Version:      fmt.Sprintf("v%d", time.Now().Unix()),
		Slot:         string(targetSlot),
		Status:       string(deployer.StatusPending),
		SourceConfig: string(sourceJSON),
		Environment:  string(envJSON),
	}

	if err := s.store.Deployments().Create(ctx, deployment); err != nil {
		return nil, apperrors.NewInternalError("failed to create deployment record", err)
	}

	// Update service status to building
	service.Status = "building"
	s.store.Services().Update(ctx, service)

	// Execute deployment asynchronously
	go s.executeServiceDeployment(context.Background(), project, service, deployment, dep, spec)

	return &DeploymentResponse{
		ID:        deployment.ID,
		AppID:     deployment.AppID,
		ServiceID: deployment.ServiceID,
		Version:   deployment.Version,
		Slot:      deployment.Slot,
		Status:    deployment.Status,
		CreatedAt: deployment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// deployDatabaseService handles database service deployment
func (s *DeployService) deployDatabaseService(ctx context.Context, project *storage.Project, service *storage.Service, req DeployServiceRequest) (*DeploymentResponse, error) {
	s.log.Info("deploying database service",
		"project_id", project.ID,
		"service_name", service.Name,
		"database_type", service.DatabaseType,
	)

	// Get image deployer for database containers
	dep, err := s.registry.Get(deployer.ModeImage)
	if err != nil {
		return nil, apperrors.NewInternalError("image deployer not available", err)
	}

	// Determine database image and port
	var image string
	var port int
	env := make(map[string]string)

	switch service.DatabaseType {
	case "postgres":
		version := service.DatabaseVersion
		if version == "" {
			version = "16"
		}
		image = fmt.Sprintf("postgres:%s", version)
		port = 5432
		env["POSTGRES_PASSWORD"] = "nebula_" + service.ID[:8]
		env["POSTGRES_DB"] = service.Name
	case "mysql":
		version := service.DatabaseVersion
		if version == "" {
			version = "8"
		}
		image = fmt.Sprintf("mysql:%s", version)
		port = 3306
		env["MYSQL_ROOT_PASSWORD"] = "nebula_" + service.ID[:8]
		env["MYSQL_DATABASE"] = service.Name
	case "redis":
		version := service.DatabaseVersion
		if version == "" {
			version = "7"
		}
		image = fmt.Sprintf("redis:%s", version)
		port = 6379
	case "mongodb":
		version := service.DatabaseVersion
		if version == "" {
			version = "7"
		}
		image = fmt.Sprintf("mongo:%s", version)
		port = 27017
		env["MONGO_INITDB_ROOT_USERNAME"] = "admin"
		env["MONGO_INITDB_ROOT_PASSWORD"] = "nebula_" + service.ID[:8]
	default:
		return nil, apperrors.NewValidationError("unsupported database type", map[string]interface{}{
			"database_type": service.DatabaseType,
		})
	}

	// Merge with request environment
	for k, v := range req.Environment {
		env[k] = v
	}

	targetSlot := s.getTargetSlotForService(ctx, service.ID)

	spec := &deployer.DeploymentSpec{
		AppID:     project.ID,
		AppName:   project.Name,
		ServiceID: service.ID,
		Source: deployer.SourceConfig{
			Image: image,
			Port:  port,
		},
		Environment: env,
		TargetSlot:  targetSlot,
	}

	// Create deployment record
	sourceJSON, _ := json.Marshal(spec.Source)
	envJSON, _ := json.Marshal(env)

	deployment := &storage.Deployment{
		ID:           uuid.New().String(),
		AppID:        project.ID,
		ServiceID:    service.ID,
		Version:      fmt.Sprintf("v%d", time.Now().Unix()),
		Slot:         string(targetSlot),
		Status:       string(deployer.StatusPending),
		SourceConfig: string(sourceJSON),
		Environment:  string(envJSON),
	}

	if err := s.store.Deployments().Create(ctx, deployment); err != nil {
		return nil, apperrors.NewInternalError("failed to create deployment record", err)
	}

	// Update service status
	service.Status = "building"
	service.Port = port
	s.store.Services().Update(ctx, service)

	// Execute deployment asynchronously
	go s.executeServiceDeployment(context.Background(), project, service, deployment, dep, spec)

	return &DeploymentResponse{
		ID:        deployment.ID,
		AppID:     deployment.AppID,
		ServiceID: deployment.ServiceID,
		Version:   deployment.Version,
		Slot:      deployment.Slot,
		Status:    deployment.Status,
		CreatedAt: deployment.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// executeServiceDeployment runs the deployment process for a service
func (s *DeployService) executeServiceDeployment(
	ctx context.Context,
	project *storage.Project,
	service *storage.Service,
	deployment *storage.Deployment,
	dep deployer.Deployer,
	spec *deployer.DeploymentSpec,
) {
	s.log.Info("executing service deployment",
		"deployment_id", deployment.ID,
		"service_id", service.ID,
	)

	// Update status to preparing
	now := time.Now()
	deployment.Status = string(deployer.StatusPreparing)
	deployment.StartedAt = &now
	s.store.Deployments().Update(ctx, deployment)

	// Prepare (pull image)
	_, err := dep.Prepare(ctx, spec)
	if err != nil {
		s.failServiceDeployment(ctx, service, deployment, err)
		return
	}

	// Update status to deploying
	deployment.Status = string(deployer.StatusDeploying)
	s.store.Deployments().Update(ctx, deployment)

	// Deploy (create and start container)
	result, err := dep.Deploy(ctx, spec)
	if err != nil {
		s.failServiceDeployment(ctx, service, deployment, err)
		return
	}

	// Store container info
	for _, containerID := range result.ContainerIDs {
		port := 0
		if p, ok := result.Ports["main"]; ok {
			port = p
		}

		container := &storage.Container{
			ID:           uuid.New().String(),
			DeploymentID: deployment.ID,
			ContainerID:  containerID,
			Name:         fmt.Sprintf("%s-%s-%s", project.Name, service.Name, spec.TargetSlot),
			Status:       "running",
			Port:         port,
		}
		s.store.Containers().Create(ctx, container)
	}

	// Health check
	healthResult, err := dep.HealthCheck(ctx, result)
	if err != nil || !healthResult.Healthy {
		errMsg := "health check failed"
		if err != nil {
			errMsg = err.Error()
		}
		s.failServiceDeployment(ctx, service, deployment, fmt.Errorf(errMsg))

		// Cleanup failed deployment
		dep.Destroy(ctx, result.ContainerIDs)
		return
	}

	// Mark deployment as running
	finishedAt := time.Now()
	deployment.Status = string(deployer.StatusRunning)
	deployment.FinishedAt = &finishedAt
	s.store.Deployments().Update(ctx, deployment)

	// Update service status
	service.Status = "running"
	s.store.Services().Update(ctx, service)

	// Stop old deployment for this service
	s.stopOldServiceDeployment(ctx, service.ID, string(spec.TargetSlot.Opposite()), dep)

	s.log.Info("service deployment completed successfully",
		"deployment_id", deployment.ID,
		"service_id", service.ID,
	)
}

// failServiceDeployment marks a service deployment as failed
func (s *DeployService) failServiceDeployment(ctx context.Context, service *storage.Service, deployment *storage.Deployment, err error) {
	s.log.Error("service deployment failed",
		"deployment_id", deployment.ID,
		"service_id", service.ID,
		"error", err,
	)

	finishedAt := time.Now()
	deployment.Status = string(deployer.StatusFailed)
	deployment.ErrorMessage = err.Error()
	deployment.FinishedAt = &finishedAt
	s.store.Deployments().Update(ctx, deployment)

	service.Status = "failed"
	s.store.Services().Update(ctx, service)
}

// stopOldServiceDeployment stops the old deployment for a service
func (s *DeployService) stopOldServiceDeployment(ctx context.Context, serviceID string, slot string, dep deployer.Deployer) {
	oldDeployment, err := s.store.Deployments().GetByServiceIDAndSlot(ctx, serviceID, slot)
	if err != nil || oldDeployment == nil {
		return
	}

	containers, err := s.store.Containers().ListByDeploymentID(ctx, oldDeployment.ID)
	if err != nil {
		return
	}

	containerIDs := make([]string, len(containers))
	for i, c := range containers {
		containerIDs[i] = c.ContainerID
	}

	if err := dep.Stop(ctx, containerIDs); err != nil {
		s.log.Warn("failed to stop old service containers", "error", err)
	}

	oldDeployment.Status = string(deployer.StatusStopped)
	s.store.Deployments().Update(ctx, oldDeployment)
}

// getTargetSlotForService determines which slot to deploy a service to
func (s *DeployService) getTargetSlotForService(ctx context.Context, serviceID string) deployer.Slot {
	// Check existing deployments for this service
	deployments, _ := s.store.Deployments().ListByServiceID(ctx, serviceID)
	if len(deployments) == 0 {
		return deployer.SlotBlue
	}

	// Find the running deployment and use opposite slot
	for _, d := range deployments {
		if d.Status == string(deployer.StatusRunning) {
			if d.Slot == string(deployer.SlotBlue) {
				return deployer.SlotGreen
			}
			return deployer.SlotBlue
		}
	}

	return deployer.SlotBlue
}
