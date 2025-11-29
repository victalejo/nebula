package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/victalejo/nebula/internal/container"
	"github.com/victalejo/nebula/internal/core/deployer"
	"github.com/victalejo/nebula/internal/core/logger"
)

// BuildpackConfig represents buildpack detection configuration
type BuildpackConfig struct {
	Name       string
	Detect     func(dir string) bool
	Dockerfile func(dir string) string
}

var defaultBuildpacks = []BuildpackConfig{
	{
		Name: "nodejs",
		Detect: func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, "package.json"))
			return err == nil
		},
		Dockerfile: func(dir string) string {
			// Check if using yarn
			useYarn := false
			if _, err := os.Stat(filepath.Join(dir, "yarn.lock")); err == nil {
				useYarn = true
			}

			installCmd := "npm ci --only=production"
			if useYarn {
				installCmd = "yarn install --production --frozen-lockfile"
			}

			return fmt.Sprintf(`FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN %s
COPY . .
RUN npm run build --if-present
EXPOSE 3000
CMD ["npm", "start"]
`, installCmd)
		},
	},
	{
		Name: "go",
		Detect: func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, "go.mod"))
			return err == nil
		},
		Dockerfile: func(dir string) string {
			return `FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
`
		},
	},
	{
		Name: "python",
		Detect: func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, "requirements.txt"))
			return err == nil
		},
		Dockerfile: func(dir string) string {
			return `FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["python", "-m", "gunicorn", "--bind", "0.0.0.0:8000", "app:app"]
`
		},
	},
	{
		Name: "static",
		Detect: func(dir string) bool {
			_, err := os.Stat(filepath.Join(dir, "index.html"))
			return err == nil
		},
		Dockerfile: func(dir string) string {
			return `FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
`
		},
	},
}

type Deployer struct {
	runtime    container.Runtime
	log        logger.Logger
	dataDir    string
	buildpacks []BuildpackConfig
}

func New(runtime container.Runtime, log logger.Logger, dataDir string) *Deployer {
	return &Deployer{
		runtime:    runtime,
		log:        log,
		dataDir:    dataDir,
		buildpacks: defaultBuildpacks,
	}
}

func (d *Deployer) Mode() deployer.DeploymentMode {
	return deployer.ModeGit
}

func (d *Deployer) Validate(ctx context.Context, spec *deployer.DeploymentSpec) error {
	if spec.GitRepo == "" {
		return fmt.Errorf("git repository URL is required")
	}

	// Basic URL validation
	if !strings.HasPrefix(spec.GitRepo, "http://") &&
		!strings.HasPrefix(spec.GitRepo, "https://") &&
		!strings.HasPrefix(spec.GitRepo, "git@") {
		return fmt.Errorf("invalid git repository URL")
	}

	return nil
}

func (d *Deployer) Prepare(ctx context.Context, spec *deployer.DeploymentSpec) (*deployer.PrepareResult, error) {
	// Create build directory
	buildDir := filepath.Join(d.dataDir, "builds", spec.AppName, uuid.New().String()[:8])
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build directory: %w", err)
	}

	// Clone repository
	d.log.Info("cloning repository", "repo", spec.GitRepo, "branch", spec.GitBranch)

	branch := spec.GitBranch
	if branch == "" {
		branch = "main"
	}

	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", "--branch", branch, spec.GitRepo, buildDir)
	cloneOutput, err := cloneCmd.CombinedOutput()
	if err != nil {
		// Try with master branch if main fails
		if branch == "main" {
			cloneCmd = exec.CommandContext(ctx, "git", "clone", "--depth=1", "--branch", "master", spec.GitRepo, buildDir)
			cloneOutput, err = cloneCmd.CombinedOutput()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w\n%s", err, string(cloneOutput))
		}
	}

	// Check for existing Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	hasDockerfile := false
	if _, err := os.Stat(dockerfilePath); err == nil {
		hasDockerfile = true
	}

	var buildLogs strings.Builder
	buildLogs.WriteString(fmt.Sprintf("Cloned %s (branch: %s)\n", spec.GitRepo, branch))

	// If no Dockerfile, detect and generate one
	if !hasDockerfile {
		buildLogs.WriteString("No Dockerfile found, detecting buildpack...\n")

		var detected *BuildpackConfig
		for i := range d.buildpacks {
			bp := &d.buildpacks[i]
			if bp.Detect(buildDir) {
				detected = bp
				break
			}
		}

		if detected == nil {
			return nil, fmt.Errorf("could not detect application type, please provide a Dockerfile")
		}

		buildLogs.WriteString(fmt.Sprintf("Detected: %s\n", detected.Name))

		// Generate Dockerfile
		dockerfile := detected.Dockerfile(buildDir)
		if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
			return nil, fmt.Errorf("failed to write generated Dockerfile: %w", err)
		}

		buildLogs.WriteString("Generated Dockerfile\n")
	} else {
		buildLogs.WriteString("Using existing Dockerfile\n")
	}

	// Build Docker image
	imageName := fmt.Sprintf("nebula/%s:%s", spec.AppName, spec.Slot)
	d.log.Info("building image", "image", imageName)

	buildLogs.WriteString(fmt.Sprintf("Building image %s...\n", imageName))

	imageID, buildOutput, err := d.runtime.BuildImage(ctx, buildDir, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to build image: %w\n%s", err, buildOutput)
	}

	buildLogs.WriteString(buildOutput)
	buildLogs.WriteString(fmt.Sprintf("\nBuild complete: %s\n", imageID))

	// Cleanup build directory (keep last 3 builds)
	d.cleanupOldBuilds(spec.AppName)

	return &deployer.PrepareResult{
		ImageID:   imageID,
		BuildLogs: buildLogs.String(),
	}, nil
}

func (d *Deployer) Deploy(ctx context.Context, spec *deployer.DeploymentSpec) (*deployer.DeploymentResult, error) {
	imageName := fmt.Sprintf("nebula/%s:%s", spec.AppName, spec.Slot)
	containerName := fmt.Sprintf("nebula-%s-%s", spec.AppName, spec.Slot)

	// Prepare environment variables
	env := make([]string, 0, len(spec.EnvVars))
	for key, val := range spec.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}

	// Add default PORT if not set
	hasPort := false
	for _, e := range env {
		if strings.HasPrefix(e, "PORT=") {
			hasPort = true
			break
		}
	}
	if !hasPort {
		env = append(env, "PORT=8080")
	}

	config := &container.ContainerConfig{
		Name:  containerName,
		Image: imageName,
		Env:   env,
		Ports: []container.PortMapping{
			{HostPort: 0, ContainerPort: 8080}, // Common default
			{HostPort: 0, ContainerPort: 3000}, // Node.js default
			{HostPort: 0, ContainerPort: 80},   // Nginx default
		},
		Labels: map[string]string{
			"nebula.app":  spec.AppName,
			"nebula.slot": string(spec.Slot),
			"nebula.mode": string(deployer.ModeGit),
		},
		RestartPolicy: "unless-stopped",
	}

	// Create and start container
	containerID, err := d.runtime.CreateContainer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	if err := d.runtime.StartContainer(ctx, containerID); err != nil {
		_ = d.runtime.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get assigned port
	info, err := d.runtime.InspectContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var port int
	for _, pm := range info.Ports {
		if pm.HostPort > 0 {
			port = pm.HostPort
			break
		}
	}

	return &deployer.DeploymentResult{
		ContainerIDs: []string{containerID},
		Port:         port,
		Version:      uuid.New().String()[:8],
	}, nil
}

func (d *Deployer) HealthCheck(ctx context.Context, result *deployer.DeploymentResult) (*deployer.HealthCheckResult, error) {
	if len(result.ContainerIDs) == 0 {
		return &deployer.HealthCheckResult{
			Healthy: false,
			Message: "no containers",
		}, nil
	}

	containerID := result.ContainerIDs[0]
	info, err := d.runtime.InspectContainer(ctx, containerID)
	if err != nil {
		return &deployer.HealthCheckResult{
			Healthy: false,
			Message: fmt.Sprintf("failed to inspect: %v", err),
		}, nil
	}

	if info.State != "running" {
		return &deployer.HealthCheckResult{
			Healthy: false,
			Message: fmt.Sprintf("container not running: %s", info.State),
		}, nil
	}

	return &deployer.HealthCheckResult{
		Healthy: true,
		Message: "container running",
	}, nil
}

func (d *Deployer) Stop(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		if err := d.runtime.StopContainer(ctx, id, 30*time.Second); err != nil {
			return fmt.Errorf("failed to stop container %s: %w", id[:12], err)
		}
	}
	return nil
}

func (d *Deployer) Destroy(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		_ = d.runtime.StopContainer(ctx, id, 10*time.Second)
		if err := d.runtime.RemoveContainer(ctx, id); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", id[:12], err)
		}
	}
	return nil
}

func (d *Deployer) cleanupOldBuilds(appName string) {
	buildsDir := filepath.Join(d.dataDir, "builds", appName)
	entries, err := os.ReadDir(buildsDir)
	if err != nil {
		return
	}

	// Keep only the 3 most recent builds
	if len(entries) <= 3 {
		return
	}

	// Sort by modification time (oldest first)
	type buildEntry struct {
		path    string
		modTime time.Time
	}
	var builds []buildEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		builds = append(builds, buildEntry{
			path:    filepath.Join(buildsDir, entry.Name()),
			modTime: info.ModTime(),
		})
	}

	// Sort by mod time
	for i := 0; i < len(builds)-1; i++ {
		for j := i + 1; j < len(builds); j++ {
			if builds[i].modTime.After(builds[j].modTime) {
				builds[i], builds[j] = builds[j], builds[i]
			}
		}
	}

	// Remove oldest builds
	for i := 0; i < len(builds)-3; i++ {
		os.RemoveAll(builds[i].path)
	}
}
