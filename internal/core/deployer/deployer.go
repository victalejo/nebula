package deployer

import (
	"context"
)

// Deployer is the core interface all deployment strategies must implement
type Deployer interface {
	// Mode returns the deployment mode this deployer handles
	Mode() DeploymentMode

	// Validate checks if the deployment spec is valid for this mode
	Validate(ctx context.Context, spec *DeploymentSpec) error

	// Prepare sets up prerequisites (pull images, clone repos, etc.)
	Prepare(ctx context.Context, spec *DeploymentSpec) (*PrepareResult, error)

	// Deploy creates and starts the containers in the target slot
	Deploy(ctx context.Context, spec *DeploymentSpec) (*DeploymentResult, error)

	// HealthCheck verifies the deployment is healthy before switching
	HealthCheck(ctx context.Context, result *DeploymentResult) (*HealthCheckResult, error)

	// Stop stops containers for a deployment
	Stop(ctx context.Context, containerIDs []string) error

	// Destroy completely removes containers and cleanup
	Destroy(ctx context.Context, containerIDs []string) error
}

// Registry manages available deployers
type Registry interface {
	Register(deployer Deployer)
	Get(mode DeploymentMode) (Deployer, error)
	List() []DeploymentMode
}

// DeployerRegistry is the default implementation of Registry
type DeployerRegistry struct {
	deployers map[DeploymentMode]Deployer
}

// NewRegistry creates a new deployer registry
func NewRegistry() *DeployerRegistry {
	return &DeployerRegistry{
		deployers: make(map[DeploymentMode]Deployer),
	}
}

// Register adds a deployer to the registry
func (r *DeployerRegistry) Register(deployer Deployer) {
	r.deployers[deployer.Mode()] = deployer
}

// Get retrieves a deployer by mode
func (r *DeployerRegistry) Get(mode DeploymentMode) (Deployer, error) {
	deployer, ok := r.deployers[mode]
	if !ok {
		return nil, &DeployerNotFoundError{Mode: mode}
	}
	return deployer, nil
}

// List returns all registered deployment modes
func (r *DeployerRegistry) List() []DeploymentMode {
	modes := make([]DeploymentMode, 0, len(r.deployers))
	for mode := range r.deployers {
		modes = append(modes, mode)
	}
	return modes
}

// DeployerNotFoundError is returned when a deployer is not found
type DeployerNotFoundError struct {
	Mode DeploymentMode
}

func (e *DeployerNotFoundError) Error() string {
	return "deployer not found for mode: " + string(e.Mode)
}
