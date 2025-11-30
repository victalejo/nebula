package image

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/core/builder"
	"github.com/victalejo/nebula/internal/core/logger"
	"github.com/victalejo/nebula/internal/core/storage"
)

// Builder handles pre-built Docker images (no building, just pulling)
type Builder struct {
	runtime container.Runtime
	log     logger.Logger
}

// New creates a new Docker image builder
func New(runtime container.Runtime, log logger.Logger) *Builder {
	return &Builder{
		runtime: runtime,
		log:     log,
	}
}

// Name returns the builder name
func (b *Builder) Name() storage.BuilderType {
	return storage.BuilderDockerImage
}

// Build pulls a pre-built image (no actual building)
func (b *Builder) Build(ctx context.Context, buildCtx *builder.BuildContext) (*builder.BuildResult, error) {
	// For docker_image mode, the ImageName should already be set to the remote image
	imageName := buildCtx.ImageName
	if buildCtx.ImageTag != "" && !strings.Contains(imageName, ":") {
		imageName = fmt.Sprintf("%s:%s", imageName, buildCtx.ImageTag)
	}

	b.log.Info("pulling Docker image", "image", imageName)

	// Pull the image (nil auth for public images)
	if err := b.runtime.PullImage(ctx, imageName, nil); err != nil {
		return nil, fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	// Get image info using docker CLI
	imageID, err := b.getImageID(ctx, imageName)
	if err != nil {
		b.log.Warn("could not get image ID", "error", err)
		imageID = imageName
	}

	// Use configured port or default
	port := buildCtx.Port
	if port == 0 {
		port = 8080
	}

	return &builder.BuildResult{
		ImageID:   imageID,
		ImageName: imageName,
		ImageTag:  buildCtx.ImageTag,
		BuildLogs: fmt.Sprintf("Pulled image: %s\n", imageName),
		Port:      port,
	}, nil
}

// Detect always returns false - this builder doesn't auto-detect
func (b *Builder) Detect(ctx context.Context, sourceDir string) (bool, int) {
	// Docker image mode is explicitly selected, never auto-detected
	return false, 0
}

// getImageID retrieves the image ID from Docker using CLI
func (b *Builder) getImageID(ctx context.Context, imageName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", imageName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
