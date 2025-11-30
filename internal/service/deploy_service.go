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

// RegistryAuthReq represents registry authentication
type RegistryAuthReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DeploymentResponse represents a deployment response
type DeploymentResponse struct {
	ID        string `json:"id"`
	AppID     string `json:"app_id"`
	ServiceID string `json:"service_id,omitempty"`
	Version   string `json:"version"`
	Slot      string `json:"slot"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
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
