package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"

	"github.com/victalejo/nebula/internal/core/storage"
)

// Store is the SQLite implementation of storage.Store
type Store struct {
	db *sql.DB

	apps        *AppRepository
	deployments *DeploymentRepository
	routes      *RouteRepository
	containers  *ContainerRepository
	databases   *DatabaseRepository
	backups     *BackupRepository
}

// NewStore creates a new SQLite store
func NewStore(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	store := &Store{db: db}
	store.apps = NewAppRepository(db)
	store.deployments = NewDeploymentRepository(db)
	store.routes = NewRouteRepository(db)
	store.containers = NewContainerRepository(db)
	store.databases = NewDatabaseRepository(db)
	store.backups = NewBackupRepository(db)

	return store, nil
}

// Apps returns the application repository
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

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate runs database migrations
func (s *Store) Migrate() error {
	migrations := []string{
		migrationV1,
	}

	for i, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", i+1, err)
		}
	}

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
`
