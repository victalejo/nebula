package nixpacks

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

// Builder builds images using nixpacks
type Builder struct {
	runtime container.Runtime
	log     logger.Logger
}

// New creates a new nixpacks builder
func New(runtime container.Runtime, log logger.Logger) *Builder {
	return &Builder{
		runtime: runtime,
		log:     log,
	}
}

// Name returns the builder name
func (b *Builder) Name() storage.BuilderType {
	return storage.BuilderNixpacks
}

// Build builds a container image using nixpacks
func (b *Builder) Build(ctx context.Context, buildCtx *builder.BuildContext) (*builder.BuildResult, error) {
	sourceDir := buildCtx.SourceDir
	if buildCtx.Subdirectory != "" && buildCtx.Subdirectory != "." {
		sourceDir = filepath.Join(sourceDir, buildCtx.Subdirectory)
	}

	b.log.Info("building with nixpacks", "source", sourceDir)

	// Check if nixpacks is available
	if _, err := exec.LookPath("nixpacks"); err != nil {
		return nil, fmt.Errorf("nixpacks not found in PATH. Please install nixpacks: https://nixpacks.com/docs/getting-started")
	}

	// Build image name
	imageName := buildCtx.ImageName
	if buildCtx.ImageTag != "" {
		imageName = fmt.Sprintf("%s:%s", buildCtx.ImageName, buildCtx.ImageTag)
	}

	// Build arguments
	args := []string{"build", sourceDir, "--name", imageName}

	// Add start command if specified
	if buildCtx.Command != "" {
		args = append(args, "--start-cmd", buildCtx.Command)
	}

	// Add build args
	for key, val := range buildCtx.BuildArgs {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, val))
	}

	b.log.Debug("running nixpacks", "args", args)

	cmd := exec.CommandContext(ctx, "nixpacks", args...)
	cmd.Dir = sourceDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("nixpacks build failed: %w\n%s", err, string(output))
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
		BuildLogs: fmt.Sprintf("Nixpacks build output:\n%s", string(output)),
		Port:      port,
	}, nil
}

// Detect checks if nixpacks can build this source
func (b *Builder) Detect(ctx context.Context, sourceDir string) (bool, int) {
	// Nixpacks can build most things, give it medium priority
	// Check for common project files
	projectFiles := []string{
		"package.json",     // Node.js
		"requirements.txt", // Python
		"Gemfile",          // Ruby
		"go.mod",           // Go
		"Cargo.toml",       // Rust
		"build.gradle",     // Java/Kotlin
		"pom.xml",          // Java Maven
		"composer.json",    // PHP
		"mix.exs",          // Elixir
	}

	for _, file := range projectFiles {
		if _, err := os.Stat(filepath.Join(sourceDir, file)); err == nil {
			return true, 60 // Medium-high priority
		}
	}

	// Check for index.html (static site)
	if _, err := os.Stat(filepath.Join(sourceDir, "index.html")); err == nil {
		return true, 40
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

	// Check for Rust
	if _, err := os.Stat(filepath.Join(sourceDir, "Cargo.toml")); err == nil {
		return 8080
	}

	return 8080 // Default
}
