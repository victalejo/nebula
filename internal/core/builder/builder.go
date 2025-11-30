package builder

import (
	"context"
	"fmt"
	"sync"

	"github.com/victalejo/nebula/internal/core/storage"
)

// BuildContext contains all information needed to build an image
type BuildContext struct {
	// Project information
	ProjectID   string
	ProjectName string

	// Service information
	ServiceID   string
	ServiceName string
	ServiceType storage.ServiceType

	// Source information
	SourceDir    string // Local directory with source code
	Subdirectory string // Subdirectory within source for monorepos

	// Build configuration
	Builder   storage.BuilderType
	Port      int
	Command   string
	BuildArgs map[string]string

	// Output
	ImageName string
	ImageTag  string
}

// BuildResult contains the result of a build operation
type BuildResult struct {
	ImageID   string
	ImageName string
	ImageTag  string
	BuildLogs string
	Port      int // Detected or configured port
}

// Builder is the interface for building container images
type Builder interface {
	// Name returns the builder name
	Name() storage.BuilderType

	// Build builds a container image from the build context
	Build(ctx context.Context, buildCtx *BuildContext) (*BuildResult, error)

	// Detect checks if this builder can build the source in the given directory
	// Returns true and a confidence score (0-100) if it can build
	Detect(ctx context.Context, sourceDir string) (bool, int)
}

// Registry manages available builders
type Registry struct {
	mu       sync.RWMutex
	builders map[storage.BuilderType]Builder
}

// NewRegistry creates a new builder registry
func NewRegistry() *Registry {
	return &Registry{
		builders: make(map[storage.BuilderType]Builder),
	}
}

// Register adds a builder to the registry
func (r *Registry) Register(b Builder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[b.Name()] = b
}

// Get returns a builder by name
func (r *Registry) Get(name storage.BuilderType) (Builder, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.builders[name]
	if !ok {
		return nil, fmt.Errorf("builder not found: %s", name)
	}
	return b, nil
}

// List returns all registered builder names
func (r *Registry) List() []storage.BuilderType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]storage.BuilderType, 0, len(r.builders))
	for name := range r.builders {
		names = append(names, name)
	}
	return names
}

// AutoDetect tries to detect the best builder for the source directory
func (r *Registry) AutoDetect(ctx context.Context, sourceDir string) (storage.BuilderType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var bestBuilder storage.BuilderType
	bestScore := 0

	for name, b := range r.builders {
		canBuild, score := b.Detect(ctx, sourceDir)
		if canBuild && score > bestScore {
			bestBuilder = name
			bestScore = score
		}
	}

	if bestBuilder == "" {
		return "", fmt.Errorf("no suitable builder found for source directory")
	}

	return bestBuilder, nil
}

// DefaultRegistry is the global builder registry
var DefaultRegistry = NewRegistry()

// Register adds a builder to the default registry
func Register(b Builder) {
	DefaultRegistry.Register(b)
}

// Get returns a builder from the default registry
func Get(name storage.BuilderType) (Builder, error) {
	return DefaultRegistry.Get(name)
}

// AutoDetect auto-detects the best builder from the default registry
func AutoDetect(ctx context.Context, sourceDir string) (storage.BuilderType, error) {
	return DefaultRegistry.AutoDetect(ctx, sourceDir)
}
