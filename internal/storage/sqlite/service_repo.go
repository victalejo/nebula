package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// ServiceRepository is the SQLite implementation of ServiceRepository
type ServiceRepository struct {
	db *sql.DB
}

// NewServiceRepository creates a new service repository
func NewServiceRepository(db *sql.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// Create creates a new service
func (r *ServiceRepository) Create(ctx context.Context, service *storage.Service) error {
	query := `
		INSERT INTO services (
			id, project_id, name, type,
			builder, git_repo, git_branch, subdirectory, docker_image,
			database_type, database_version,
			port, command, environment, replicas, status,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	service.CreatedAt = now
	service.UpdatedAt = now

	if service.Status == "" {
		service.Status = "stopped"
	}
	if service.Replicas == 0 {
		service.Replicas = 1
	}
	if service.Port == 0 {
		service.Port = 8080
	}
	if service.Subdirectory == "" {
		service.Subdirectory = "."
	}
	if service.Environment == "" {
		service.Environment = "{}"
	}

	_, err := r.db.ExecContext(ctx, query,
		service.ID,
		service.ProjectID,
		service.Name,
		string(service.Type),
		string(service.Builder),
		nullString(service.GitRepo),
		nullString(service.GitBranch),
		service.Subdirectory,
		nullString(service.DockerImage),
		nullString(service.DatabaseType),
		nullString(service.DatabaseVersion),
		service.Port,
		nullString(service.Command),
		service.Environment,
		service.Replicas,
		service.Status,
		service.CreatedAt,
		service.UpdatedAt,
	)
	return err
}

// GetByID retrieves a service by ID
func (r *ServiceRepository) GetByID(ctx context.Context, id string) (*storage.Service, error) {
	query := `
		SELECT id, project_id, name, type,
		       COALESCE(builder, 'nixpacks'), COALESCE(git_repo, ''), COALESCE(git_branch, ''),
		       COALESCE(subdirectory, '.'), COALESCE(docker_image, ''),
		       COALESCE(database_type, ''), COALESCE(database_version, ''),
		       COALESCE(port, 8080), COALESCE(command, ''), COALESCE(environment, '{}'),
		       COALESCE(replicas, 1), COALESCE(status, 'stopped'),
		       created_at, updated_at
		FROM services
		WHERE id = ?
	`
	return r.scanService(r.db.QueryRowContext(ctx, query, id))
}

// GetByProjectIDAndName retrieves a service by project ID and name
func (r *ServiceRepository) GetByProjectIDAndName(ctx context.Context, projectID, name string) (*storage.Service, error) {
	query := `
		SELECT id, project_id, name, type,
		       COALESCE(builder, 'nixpacks'), COALESCE(git_repo, ''), COALESCE(git_branch, ''),
		       COALESCE(subdirectory, '.'), COALESCE(docker_image, ''),
		       COALESCE(database_type, ''), COALESCE(database_version, ''),
		       COALESCE(port, 8080), COALESCE(command, ''), COALESCE(environment, '{}'),
		       COALESCE(replicas, 1), COALESCE(status, 'stopped'),
		       created_at, updated_at
		FROM services
		WHERE project_id = ? AND name = ?
	`
	return r.scanService(r.db.QueryRowContext(ctx, query, projectID, name))
}

func (r *ServiceRepository) scanService(row *sql.Row) (*storage.Service, error) {
	service := &storage.Service{}
	var serviceType, builder string
	err := row.Scan(
		&service.ID,
		&service.ProjectID,
		&service.Name,
		&serviceType,
		&builder,
		&service.GitRepo,
		&service.GitBranch,
		&service.Subdirectory,
		&service.DockerImage,
		&service.DatabaseType,
		&service.DatabaseVersion,
		&service.Port,
		&service.Command,
		&service.Environment,
		&service.Replicas,
		&service.Status,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	service.Type = storage.ServiceType(serviceType)
	service.Builder = storage.BuilderType(builder)
	return service, nil
}

// Update updates a service
func (r *ServiceRepository) Update(ctx context.Context, service *storage.Service) error {
	query := `
		UPDATE services
		SET name = ?, type = ?,
		    builder = ?, git_repo = ?, git_branch = ?, subdirectory = ?, docker_image = ?,
		    database_type = ?, database_version = ?,
		    port = ?, command = ?, environment = ?, replicas = ?, status = ?,
		    updated_at = ?
		WHERE id = ?
	`
	service.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		service.Name,
		string(service.Type),
		string(service.Builder),
		nullString(service.GitRepo),
		nullString(service.GitBranch),
		service.Subdirectory,
		nullString(service.DockerImage),
		nullString(service.DatabaseType),
		nullString(service.DatabaseVersion),
		service.Port,
		nullString(service.Command),
		service.Environment,
		service.Replicas,
		service.Status,
		service.UpdatedAt,
		service.ID,
	)
	return err
}

// Delete deletes a service
func (r *ServiceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM services WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByProjectID returns all services for a project
func (r *ServiceRepository) ListByProjectID(ctx context.Context, projectID string) ([]*storage.Service, error) {
	query := `
		SELECT id, project_id, name, type,
		       COALESCE(builder, 'nixpacks'), COALESCE(git_repo, ''), COALESCE(git_branch, ''),
		       COALESCE(subdirectory, '.'), COALESCE(docker_image, ''),
		       COALESCE(database_type, ''), COALESCE(database_version, ''),
		       COALESCE(port, 8080), COALESCE(command, ''), COALESCE(environment, '{}'),
		       COALESCE(replicas, 1), COALESCE(status, 'stopped'),
		       created_at, updated_at
		FROM services
		WHERE project_id = ?
		ORDER BY created_at ASC
	`
	return r.scanServices(r.db.QueryContext(ctx, query, projectID))
}

// List returns all services
func (r *ServiceRepository) List(ctx context.Context) ([]*storage.Service, error) {
	query := `
		SELECT id, project_id, name, type,
		       COALESCE(builder, 'nixpacks'), COALESCE(git_repo, ''), COALESCE(git_branch, ''),
		       COALESCE(subdirectory, '.'), COALESCE(docker_image, ''),
		       COALESCE(database_type, ''), COALESCE(database_version, ''),
		       COALESCE(port, 8080), COALESCE(command, ''), COALESCE(environment, '{}'),
		       COALESCE(replicas, 1), COALESCE(status, 'stopped'),
		       created_at, updated_at
		FROM services
		ORDER BY created_at DESC
	`
	return r.scanServices(r.db.QueryContext(ctx, query))
}

func (r *ServiceRepository) scanServices(rows *sql.Rows, err error) ([]*storage.Service, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*storage.Service
	for rows.Next() {
		service := &storage.Service{}
		var serviceType, builder string
		if err := rows.Scan(
			&service.ID,
			&service.ProjectID,
			&service.Name,
			&serviceType,
			&builder,
			&service.GitRepo,
			&service.GitBranch,
			&service.Subdirectory,
			&service.DockerImage,
			&service.DatabaseType,
			&service.DatabaseVersion,
			&service.Port,
			&service.Command,
			&service.Environment,
			&service.Replicas,
			&service.Status,
			&service.CreatedAt,
			&service.UpdatedAt,
		); err != nil {
			return nil, err
		}
		service.Type = storage.ServiceType(serviceType)
		service.Builder = storage.BuilderType(builder)
		services = append(services, service)
	}
	return services, rows.Err()
}
