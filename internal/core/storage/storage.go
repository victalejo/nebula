package storage

import (
	"context"
	"time"
)

// Application represents an application entity
type Application struct {
	ID             string
	Name           string
	DeploymentMode string
	Domain         string
	GitRepo        string
	GitBranch      string
	DockerImage    string
	Environment    string // JSON encoded
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Deployment represents a deployment entity
type Deployment struct {
	ID           string
	AppID        string
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

// Route represents a route entity
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

// Database represents a managed database entity
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

// AppRepository handles application persistence
type AppRepository interface {
	Create(ctx context.Context, app *Application) error
	GetByID(ctx context.Context, id string) (*Application, error)
	GetByName(ctx context.Context, name string) (*Application, error)
	Update(ctx context.Context, app *Application) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*Application, error)
}

// DeploymentRepository handles deployment persistence
type DeploymentRepository interface {
	Create(ctx context.Context, deployment *Deployment) error
	GetByID(ctx context.Context, id string) (*Deployment, error)
	Update(ctx context.Context, deployment *Deployment) error
	Delete(ctx context.Context, id string) error
	ListByAppID(ctx context.Context, appID string) ([]*Deployment, error)
	GetLatestByAppID(ctx context.Context, appID string) (*Deployment, error)
	GetByAppIDAndSlot(ctx context.Context, appID string, slot string) (*Deployment, error)
}

// RouteRepository handles route persistence
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

// DatabaseRepository handles managed database persistence
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
