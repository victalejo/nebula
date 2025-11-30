package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/victalejo/nebula/internal/core/storage"
)

// Store is the SQLite implementation of storage.Store
type Store struct {
	db *sql.DB

	// New repositories
	projects *ProjectRepository
	services *ServiceRepository
	domains  *DomainRepository

	// Legacy repositories
	apps          *AppRepository
	deployments   *DeploymentRepository
	routes        *RouteRepository
	containers    *ContainerRepository
	databases     *DatabaseRepository
	backups       *BackupRepository
	binaryBackups *BinaryBackupRepository
	settings      *SettingsRepository
}

// NewStore creates a new SQLite store
func NewStore(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	store := &Store{db: db}

	// Initialize new repositories
	store.projects = NewProjectRepository(db)
	store.services = NewServiceRepository(db)
	store.domains = NewDomainRepository(db)

	// Initialize legacy repositories
	store.apps = NewAppRepository(db)
	store.deployments = NewDeploymentRepository(db)
	store.routes = NewRouteRepository(db)
	store.containers = NewContainerRepository(db)
	store.databases = NewDatabaseRepository(db)
	store.backups = NewBackupRepository(db)
	store.binaryBackups = NewBinaryBackupRepository(db)
	store.settings = NewSettingsRepository(db)

	return store, nil
}

// Projects returns the project repository
func (s *Store) Projects() storage.ProjectRepository {
	return s.projects
}

// Services returns the service repository
func (s *Store) Services() storage.ServiceRepository {
	return s.services
}

// Domains returns the domain repository
func (s *Store) Domains() storage.DomainRepository {
	return s.domains
}

// Apps returns the application repository (legacy, wraps ProjectRepository)
func (s *Store) Apps() storage.AppRepository {
	return s.apps
}

// Deployments returns the deployment repository
func (s *Store) Deployments() storage.DeploymentRepository {
	return s.deployments
}

// Routes returns the route repository
func (s *Store) Routes() storage.RouteRepository {
	return s.routes
}

// Containers returns the container repository
func (s *Store) Containers() storage.ContainerRepository {
	return s.containers
}

// Databases returns the database repository
func (s *Store) Databases() storage.DatabaseRepository {
	return s.databases
}

// Backups returns the backup repository
func (s *Store) Backups() storage.BackupRepository {
	return s.backups
}

// BinaryBackups returns the binary backup repository
func (s *Store) BinaryBackups() storage.BinaryBackupRepository {
	return s.binaryBackups
}

// Settings returns the settings repository
func (s *Store) Settings() storage.SettingsRepository {
	return s.settings
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate runs database migrations
func (s *Store) Migrate() error {
	// Run V1 migration (creates tables if not exist)
	if _, err := s.db.Exec(migrationV1); err != nil {
		return fmt.Errorf("failed to run migration V1: %w", err)
	}

	// Run V2 migration (adds columns - ignore errors if columns exist)
	v2Columns := []string{
		"ALTER TABLE applications ADD COLUMN git_repo TEXT",
		"ALTER TABLE applications ADD COLUMN git_branch TEXT",
		"ALTER TABLE applications ADD COLUMN docker_image TEXT",
	}
	for _, col := range v2Columns {
		// Ignore errors - column might already exist
		s.db.Exec(col)
	}

	// Run V3 migration (new architecture: projects, services, domains)
	if _, err := s.db.Exec(migrationV3); err != nil {
		return fmt.Errorf("failed to run migration V3: %w", err)
	}

	// V3 schema changes - ignore errors if already applied
	v3Alterations := []string{
		// Add display_name and description to applications/projects
		"ALTER TABLE applications ADD COLUMN display_name TEXT",
		"ALTER TABLE applications ADD COLUMN description TEXT",
		// Add service_id to deployments
		"ALTER TABLE deployments ADD COLUMN service_id TEXT REFERENCES services(id)",
	}
	for _, alt := range v3Alterations {
		s.db.Exec(alt)
	}

	// Create index on service_id after column is added
	s.db.Exec("CREATE INDEX IF NOT EXISTS idx_deployments_service_id ON deployments(service_id)")

	return nil
}

const migrationV1 = `
CREATE TABLE IF NOT EXISTS applications (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    deployment_mode TEXT NOT NULL,
    domain TEXT,
    environment TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    slot TEXT NOT NULL,
    status TEXT NOT NULL,
    source_config TEXT NOT NULL,
    environment TEXT,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    finished_at DATETIME
);

CREATE TABLE IF NOT EXISTS routes (
    id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    domain TEXT NOT NULL UNIQUE,
    active_slot TEXT,
    ssl_enabled BOOLEAN DEFAULT TRUE,
    blue_port INTEGER,
    green_port INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS containers (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    port INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS databases (
    id TEXT PRIMARY KEY,
    app_id TEXT REFERENCES applications(id) ON DELETE SET NULL,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    version TEXT,
    status TEXT NOT NULL,
    container_id TEXT,
    host TEXT DEFAULT 'localhost',
    port INTEGER,
    username TEXT,
    password TEXT,
    database_name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS database_backups (
    id TEXT PRIMARY KEY,
    database_id TEXT NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    size_bytes INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_deployments_app_id ON deployments(app_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_containers_deployment_id ON containers(deployment_id);
CREATE INDEX IF NOT EXISTS idx_databases_app_id ON databases(app_id);
CREATE INDEX IF NOT EXISTS idx_backups_database_id ON database_backups(database_id);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS binary_backups (
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    binary_path TEXT NOT NULL,
    binary_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_binary_backups_version ON binary_backups(version);
`

const migrationV3 = `
-- Services table: represents individual services within a project/application
CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'web',

    -- Build configuration
    builder TEXT DEFAULT 'nixpacks',
    git_repo TEXT,
    git_branch TEXT,
    subdirectory TEXT DEFAULT '.',
    docker_image TEXT,

    -- Database configuration (only for type=database)
    database_type TEXT,
    database_version TEXT,

    -- Runtime configuration
    port INTEGER DEFAULT 8080,
    command TEXT,
    environment TEXT DEFAULT '{}',
    replicas INTEGER DEFAULT 1,

    -- State
    status TEXT DEFAULT 'stopped',

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(project_id, name)
);

-- Domains table: replaces routes with more flexible routing
CREATE TABLE IF NOT EXISTS domains (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    service_id TEXT NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    domain TEXT NOT NULL UNIQUE,
    path_prefix TEXT DEFAULT '/',
    active_slot TEXT DEFAULT 'blue',
    ssl_enabled BOOLEAN DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for new tables
CREATE INDEX IF NOT EXISTS idx_services_project_id ON services(project_id);
CREATE INDEX IF NOT EXISTS idx_services_type ON services(type);
CREATE INDEX IF NOT EXISTS idx_domains_project_id ON domains(project_id);
CREATE INDEX IF NOT EXISTS idx_domains_service_id ON domains(service_id);
`
