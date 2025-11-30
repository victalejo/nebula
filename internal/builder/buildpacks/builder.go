package buildpacks

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

// Default buildpack builder images
const (
	HerokuBuilder    = "heroku/builder:22"
	PaketoBuilder    = "paketobuildpacks/builder:base"
	GoogleBuilder    = "gcr.io/buildpacks/builder:v1"
	DefaultBuilder   = PaketoBuilder
)

// Builder builds images using Cloud Native Buildpacks (CNB)
type Builder struct {
	runtime      container.Runtime
	log          logger.Logger
	builderImage string
}

// New creates a new CNB buildpacks builder
func New(runtime container.Runtime, log logger.Logger) *Builder {
	return &Builder{
		runtime:      runtime,
		log:          log,
		builderImage: DefaultBuilder,
	}
}

// NewWithBuilder creates a new CNB builder with a specific builder image
func NewWithBuilder(runtime container.Runtime, log logger.Logger, builderImage string) *Builder {
	return &Builder{
		runtime:      runtime,
		log:          log,
		builderImage: builderImage,
	}
}

// Name returns the builder name
func (b *Builder) Name() storage.BuilderType {
	return storage.BuilderBuildpacks
}

// Build builds a container image using pack CLI (CNB)
func (b *Builder) Build(ctx context.Context, buildCtx *builder.BuildContext) (*builder.BuildResult, error) {
	sourceDir := buildCtx.SourceDir
	if buildCtx.Subdirectory != "" && buildCtx.Subdirectory != "." {
		sourceDir = filepath.Join(sourceDir, buildCtx.Subdirectory)
	}

	b.log.Info("building with Cloud Native Buildpacks", "source", sourceDir, "builder", b.builderImage)

	// Check if pack CLI is available
	if _, err := exec.LookPath("pack"); err != nil {
		return nil, fmt.Errorf("pack CLI not found in PATH. Please install pack: https://buildpacks.io/docs/tools/pack/")
	}

	// Build image name
	imageName := buildCtx.ImageName
	if buildCtx.ImageTag != "" {
		imageName = fmt.Sprintf("%s:%s", buildCtx.ImageName, buildCtx.ImageTag)
	}

	// Build arguments
	args := []string{
		"build", imageName,
		"--path", sourceDir,
		"--builder", b.builderImage,
	}

	// Add environment variables
	for key, val := range buildCtx.BuildArgs {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, val))
	}

	// Add default PORT env if not specified
	if buildCtx.Port > 0 {
		args = append(args, "--env", fmt.Sprintf("PORT=%d", buildCtx.Port))
	}

	b.log.Debug("running pack build", "args", args)

	cmd := exec.CommandContext(ctx, "pack", args...)
	cmd.Dir = sourceDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pack build failed: %w\n%s", err, string(output))
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
		BuildLogs: fmt.Sprintf("Cloud Native Buildpacks (pack) output:\nBuilder: %s\n%s", b.builderImage, string(output)),
		Port:      port,
	}, nil
}

// Detect checks if CNB can build this source
func (b *Builder) Detect(ctx context.Context, sourceDir string) (bool, int) {
	// CNB can build most things but usually has a longer build time
	// Give it lower priority than nixpacks/railpacks
	projectFiles := []string{
		"package.json",     // Node.js
		"requirements.txt", // Python
		"Gemfile",          // Ruby
		"go.mod",           // Go
		"build.gradle",     // Java/Kotlin
		"pom.xml",          // Java Maven
		"composer.json",    // PHP
	}

	for _, file := range projectFiles {
		if _, err := os.Stat(filepath.Join(sourceDir, file)); err == nil {
			return true, 50 // Lower priority than nixpacks/railpacks
		}
	}

	// Check for Procfile (Heroku-style)
	if _, err := os.Stat(filepath.Join(sourceDir, "Procfile")); err == nil {
		return true, 70 // Higher priority if Procfile exists
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

	// Check for Java
	if _, err := os.Stat(filepath.Join(sourceDir, "pom.xml")); err == nil {
		return 8080
	}

	return 8080 // Default
}
