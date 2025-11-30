package railpacks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/core/builder"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// Builder builds images using railpacks (Railway's builder)
type Builder struct {
	runtime container.Runtime
	log     logger.Logger
}

// New creates a new railpacks builder
func New(runtime container.Runtime, log logger.Logger) *Builder {
	return &Builder{
		runtime: runtime,
		log:     log,
	}
}

// Name returns the builder name
func (b *Builder) Name() storage.BuilderType {
	return storage.BuilderRailpacks
}

// Build builds a container image using railpacks
func (b *Builder) Build(ctx context.Context, buildCtx *builder.BuildContext) (*builder.BuildResult, error) {
	sourceDir := buildCtx.SourceDir
	if buildCtx.Subdirectory != "" && buildCtx.Subdirectory != "." {
		sourceDir = filepath.Join(sourceDir, buildCtx.Subdirectory)
	}

	b.log.Info("building with railpacks", "source", sourceDir)

	// Check if railpacks is available
	if _, err := exec.LookPath("railpacks"); err != nil {
		return nil, fmt.Errorf("railpacks not found in PATH. Please install railpacks: https://github.com/railwayapp/railpacks")
	}

	// Build image name
	imageName := buildCtx.ImageName
	if buildCtx.ImageTag != "" {
		imageName = fmt.Sprintf("%s:%s", buildCtx.ImageName, buildCtx.ImageTag)
	}

	// Build arguments
	args := []string{"build", "--name", imageName, sourceDir}

	// Add start command if specified
	if buildCtx.Command != "" {
		args = append(args, "--start-command", buildCtx.Command)
	}

	// Add build args as environment variables
	for key, val := range buildCtx.BuildArgs {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, val))
	}

	b.log.Debug("running railpacks", "args", args)

	cmd := exec.CommandContext(ctx, "railpacks", args...)
	cmd.Dir = sourceDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("railpacks build failed: %w\n%s", err, string(output))
	}

	// Get image ID
	imageID, err := b.getImageID(ctx, imageName)
	if err != nil {
		b.log.Warn("could not get image ID", "error", err)
		imageID = imageName
	}

	// Detect port
	port := buildCtx.Port
	if port == 0 {
		port = b.detectPort(sourceDir)
	}

	return &builder.BuildResult{
		ImageID:   imageID,
		ImageName: buildCtx.ImageName,
		ImageTag:  buildCtx.ImageTag,
		BuildLogs: fmt.Sprintf("Railpacks build output:\n%s", string(output)),
		Port:      port,
	}, nil
}

// Detect checks if railpacks can build this source
func (b *Builder) Detect(ctx context.Context, sourceDir string) (bool, int) {
	// Railpacks is similar to nixpacks, give it slightly lower priority
	// unless it's specifically detected
	projectFiles := []string{
		"package.json",     // Node.js
		"requirements.txt", // Python
		"Gemfile",          // Ruby
		"go.mod",           // Go
		"Cargo.toml",       // Rust
		"build.gradle",     // Java/Kotlin
		"pom.xml",          // Java Maven
		"composer.json",    // PHP
	}

	for _, file := range projectFiles {
		if _, err := os.Stat(filepath.Join(sourceDir, file)); err == nil {
			return true, 55 // Slightly lower than nixpacks by default
		}
	}

	return false, 0
}

// getImageID retrieves the image ID from Docker
func (b *Builder) getImageID(ctx context.Context, imageName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// detectPort tries to detect the application port
func (b *Builder) detectPort(sourceDir string) int {
	// Check package.json for port hints
	packageJSON := filepath.Join(sourceDir, "package.json")
	if content, err := os.ReadFile(packageJSON); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "next") || strings.Contains(contentStr, "nuxt") {
			return 3000
		}
		if strings.Contains(contentStr, "vite") {
			return 5173
		}
	}

	// Check for Go
	if _, err := os.Stat(filepath.Join(sourceDir, "go.mod")); err == nil {
		return 8080
	}

	// Check for Python
	if _, err := os.Stat(filepath.Join(sourceDir, "requirements.txt")); err == nil {
		return 8000
	}

	// Check for Ruby
	if _, err := os.Stat(filepath.Join(sourceDir, "Gemfile")); err == nil {
		return 3000
	}

	return 8080 // Default
}
