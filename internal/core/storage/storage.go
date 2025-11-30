package storage

import (
	"context"
	"time"
)

// ServiceType represents the type of service
type ServiceType string

const (
	ServiceTypeWeb      ServiceType = "web"
	ServiceTypeWorker   ServiceType = "worker"
	ServiceTypeCron     ServiceType = "cron"
	ServiceTypeDatabase ServiceType = "database"
)

// BuilderType represents the builder used to build the service
type BuilderType string

const (
	BuilderNixpacks    BuilderType = "nixpacks"
	BuilderRailpacks   BuilderType = "railpacks"
	BuilderDockerfile  BuilderType = "dockerfile"
	BuilderDockerImage BuilderType = "docker_image"
	BuilderBuildpacks  BuilderType = "buildpacks"
)

// Project represents a project entity (container for services)
type Project struct {
	ID          string
	Name        string // unique slug
	DisplayName string
	Description string
	GitRepo     string // main repo for monorepos
	GitBranch   string // default branch
	Environment string // JSON encoded shared env vars
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Service represents a service within a project
type Service struct {
	ID        string
	ProjectID string
	Name      string // e.g., "backend", "frontend", "worker"
	Type      ServiceType

	// Build configuration (not used for type=database)
	Builder      BuilderType
	GitRepo      string // override project's repo
	GitBranch    string // override project's branch
	Subdirectory string // for monorepos: "apps/api"
	DockerImage  string // if builder=docker_image

	// Database configuration (only for type=database)
	DatabaseType    string // postgres, mysql, redis, mongodb
	DatabaseVersion string

	// Runtime configuration
	Port        int
	Command     string // custom start command
	Environment string // JSON encoded, merged with Project.Environment
	Replicas    int    // number of instances (default 1)

	// State
	Status string // running, stopped, failed

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Domain represents a domain routing configuration
type Domain struct {
	ID         string
	ProjectID  string
	ServiceID  string
	Domain     string // e.g., "api.example.com"
	PathPrefix string // e.g., "/api" (empty = root)
	ActiveSlot string // blue/green
	SSLEnabled bool
	CreatedAt  time.Time
}

// Deployment represents a deployment entity
type Deployment struct {
	ID           string
	AppID        string // legacy: project_id
	ServiceID    string // new: links to service
	Version      string
	Slot         string
	Status       string
	SourceConfig string // JSON encoded
	Environment  string // JSON encoded
	ErrorMessage string
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// Route represents a route entity (legacy, replaced by Domain)
type Route struct {
	ID         string
	AppID      string
	Domain     string
	ActiveSlot string
	SSLEnabled bool
	BluePort   int
	GreenPort  int
	CreatedAt  time.Time
}

// Container represents a container entity
type Container struct {
	ID           string
	DeploymentID string
	ContainerID  string
	Name         string
	Status       string
	Port         int
	CreatedAt    time.Time
}

// Database represents a managed database entity (legacy, migrated to Service)
type Database struct {
	ID          string
	AppID       *string
	Name        string
	Type        string // postgres, mysql, redis, mongodb
	Version     string
	Status      string
	ContainerID string
	Host        string
	Port        int
	Username    string
	Password    string
	Database    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DatabaseBackup represents a database backup entity
type DatabaseBackup struct {
	ID         string
	DatabaseID string
	Path       string
	SizeBytes  int64
	CreatedAt  time.Time
}

// BinaryBackup represents a binary backup for rollback
type BinaryBackup struct {
	ID         string
	Version    string
	BinaryPath string
	BinaryHash string
	CreatedAt  time.Time
}

// ProjectRepository handles project persistence
type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByName(ctx context.Context, name string) (*Project, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Project, error)
}

// ServiceRepository handles service persistence
type ServiceRepository interface {
	Create(ctx context.Context, service *Service) error
	GetByID(ctx context.Context, id string) (*Service, error)
	GetByProjectIDAndName(ctx context.Context, projectID, name string) (*Service, error)
	Update(ctx context.Context, service *Service) error
	Delete(ctx context.Context, id string) error
	ListByProjectID(ctx context.Context, projectID string) ([]*Service, error)
	List(ctx context.Context) ([]*Service, error)
}

// DomainRepository handles domain persistence
type DomainRepository interface {
	Create(ctx context.Context, domain *Domain) error
	GetByID(ctx context.Context, id string) (*Domain, error)
	GetByDomain(ctx context.Context, domain string) (*Domain, error)
	Update(ctx context.Context, domain *Domain) error
	Delete(ctx context.Context, id string) error
	ListByProjectID(ctx context.Context, projectID string) ([]*Domain, error)
	ListByServiceID(ctx context.Context, serviceID string) ([]*Domain, error)
	List(ctx context.Context) ([]*Domain, error)
}

// AppRepository handles application persistence (legacy, use ProjectRepository)
type AppRepository interface {
	Create(ctx context.Context, app *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	GetByName(ctx context.Context, name string) (*Project, error)
	Update(ctx context.Context, app *Project) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Project, error)
}

// DeploymentRepository handles deployment persistence
type DeploymentRepository interface {
	Create(ctx context.Context, deployment *Deployment) error
	GetByID(ctx context.Context, id string) (*Deployment, error)
	Update(ctx context.Context, deployment *Deployment) error
	Delete(ctx context.Context, id string) error
	ListByAppID(ctx context.Context, appID string) ([]*Deployment, error)
	ListByServiceID(ctx context.Context, serviceID string) ([]*Deployment, error)
	GetLatestByAppID(ctx context.Context, appID string) (*Deployment, error)
	GetLatestByServiceID(ctx context.Context, serviceID string) (*Deployment, error)
	GetByAppIDAndSlot(ctx context.Context, appID string, slot string) (*Deployment, error)
	GetByServiceIDAndSlot(ctx context.Context, serviceID string, slot string) (*Deployment, error)
}

// RouteRepository handles route persistence (legacy, use DomainRepository)
type RouteRepository interface {
	Create(ctx context.Context, route *Route) error
	GetByID(ctx context.Context, id string) (*Route, error)
	GetByDomain(ctx context.Context, domain string) (*Route, error)
	GetByAppID(ctx context.Context, appID string) (*Route, error)
	Update(ctx context.Context, route *Route) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Route, error)
}

// ContainerRepository handles container persistence
type ContainerRepository interface {
	Create(ctx context.Context, container *Container) error
	GetByID(ctx context.Context, id string) (*Container, error)
	Update(ctx context.Context, container *Container) error
	Delete(ctx context.Context, id string) error
	ListByDeploymentID(ctx context.Context, deploymentID string) ([]*Container, error)
	DeleteByDeploymentID(ctx context.Context, deploymentID string) error
}

// DatabaseRepository handles managed database persistence (legacy)
type DatabaseRepository interface {
	Create(ctx context.Context, db *Database) error
	GetByID(ctx context.Context, id string) (*Database, error)
	GetByName(ctx context.Context, name string) (*Database, error)
	Update(ctx context.Context, db *Database) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Database, error)
	ListByAppID(ctx context.Context, appID string) ([]*Database, error)
}

// BackupRepository handles database backup persistence
type BackupRepository interface {
	Create(ctx context.Context, backup *DatabaseBackup) error
	GetByID(ctx context.Context, id string) (*DatabaseBackup, error)
	Delete(ctx context.Context, id string) error
	ListByDatabaseID(ctx context.Context, databaseID string) ([]*DatabaseBackup, error)
}

// SettingsRepository handles system settings persistence
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

// BinaryBackupRepository handles binary backup persistence for rollback
type BinaryBackupRepository interface {
	Create(ctx context.Context, backup *BinaryBackup) error
	Get(ctx context.Context, id string) (*BinaryBackup, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*BinaryBackup, error)
}

// Store provides access to all repositories
type Store interface {
	// New repositories
	Projects() ProjectRepository
	Services() ServiceRepository
	Domains() DomainRepository

	// Legacy repositories (for backwards compatibility during migration)
	Apps() AppRepository
	Deployments() DeploymentRepository
	Routes() RouteRepository
	Containers() ContainerRepository
	Databases() DatabaseRepository
	Backups() BackupRepository
	BinaryBackups() BinaryBackupRepository
	Settings() SettingsRepository

	Close() error
	Migrate() error
}
