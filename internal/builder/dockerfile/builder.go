package dockerfile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/core/builder"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// Builder builds images using an existing Dockerfile
type Builder struct {
	runtime container.Runtime
	log     logger.Logger
}

// New creates a new Dockerfile builder
func New(runtime container.Runtime, log logger.Logger) *Builder {
	return &Builder{
		runtime: runtime,
		log:     log,
	}
}

// Name returns the builder name
func (b *Builder) Name() storage.BuilderType {
	return storage.BuilderDockerfile
}

// Build builds a container image using the Dockerfile
func (b *Builder) Build(ctx context.Context, buildCtx *builder.BuildContext) (*builder.BuildResult, error) {
	sourceDir := buildCtx.SourceDir
	if buildCtx.Subdirectory != "" && buildCtx.Subdirectory != "." {
		sourceDir = filepath.Join(sourceDir, buildCtx.Subdirectory)
	}

	// Check for Dockerfile
	dockerfilePath := filepath.Join(sourceDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Dockerfile not found in %s", sourceDir)
	}

	b.log.Info("building with Dockerfile", "path", dockerfilePath)

	// Build image
	imageName := buildCtx.ImageName
	if buildCtx.ImageTag != "" {
		imageName = fmt.Sprintf("%s:%s", buildCtx.ImageName, buildCtx.ImageTag)
	}

	imageID, buildOutput, err := b.runtime.BuildImage(ctx, sourceDir, imageName)
	if err != nil {
		return nil, fmt.Errorf("docker build failed: %w\n%s", err, buildOutput)
	}

	// Detect port from Dockerfile
	port := buildCtx.Port
	if port == 0 {
		port = detectPortFromDockerfile(dockerfilePath)
	}

	return &builder.BuildResult{
		ImageID:   imageID,
		ImageName: buildCtx.ImageName,
		ImageTag:  buildCtx.ImageTag,
		BuildLogs: fmt.Sprintf("Using Dockerfile\n%s", buildOutput),
		Port:      port,
	}, nil
}

// Detect checks if a Dockerfile exists
func (b *Builder) Detect(ctx context.Context, sourceDir string) (bool, int) {
	dockerfilePath := filepath.Join(sourceDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); err == nil {
		return true, 100 // Highest priority if Dockerfile exists
	}
	return false, 0
}

// detectPortFromDockerfile tries to detect exposed port from Dockerfile
func detectPortFromDockerfile(dockerfilePath string) int {
	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		return 8080
	}

	// Simple EXPOSE detection
	// In a real implementation, we'd parse the Dockerfile properly
	contentStr := string(content)
	ports := []int{8080, 3000, 5000, 80, 443}
	for _, port := range ports {
		if contains(contentStr, fmt.Sprintf("EXPOSE %d", port)) {
			return port
		}
	}

	return 8080
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func init() {
	// Note: Registration happens in main or during initialization
	// builder.Register(New(nil, nil)) // Can't register here without runtime
}
