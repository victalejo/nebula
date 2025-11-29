package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// ContainerRepository is the SQLite implementation of ContainerRepository
type ContainerRepository struct {
	db *sql.DB
}

// NewContainerRepository creates a new container repository
func NewContainerRepository(db *sql.DB) *ContainerRepository {
	return &ContainerRepository{db: db}
}

// Create creates a new container record
func (r *ContainerRepository) Create(ctx context.Context, container *storage.Container) error {
	query := `
		INSERT INTO containers (id, deployment_id, container_id, name, status, port, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	container.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		container.ID,
		container.DeploymentID,
		container.ContainerID,
		container.Name,
		container.Status,
		container.Port,
		container.CreatedAt,
	)
	return err
}

// GetByID retrieves a container by ID
func (r *ContainerRepository) GetByID(ctx context.Context, id string) (*storage.Container, error) {
	query := `
		SELECT id, deployment_id, container_id, name, status, port, created_at
		FROM containers
		WHERE id = ?
	`
	c := &storage.Container{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.DeploymentID,
		&c.ContainerID,
		&c.Name,
		&c.Status,
		&c.Port,
		&c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Update updates a container record
func (r *ContainerRepository) Update(ctx context.Context, container *storage.Container) error {
	query := `
		UPDATE containers
		SET status = ?, port = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		container.Status,
		container.Port,
		container.ID,
	)
	return err
}

// Delete deletes a container record
func (r *ContainerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM containers WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByDeploymentID returns all containers for a deployment
func (r *ContainerRepository) ListByDeploymentID(ctx context.Context, deploymentID string) ([]*storage.Container, error) {
	query := `
		SELECT id, deployment_id, container_id, name, status, port, created_at
		FROM containers
		WHERE deployment_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var containers []*storage.Container
	for rows.Next() {
		c := &storage.Container{}
		if err := rows.Scan(
			&c.ID,
			&c.DeploymentID,
			&c.ContainerID,
			&c.Name,
			&c.Status,
			&c.Port,
			&c.CreatedAt,
		); err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}
	return containers, rows.Err()
}

// DeleteByDeploymentID deletes all containers for a deployment
func (r *ContainerRepository) DeleteByDeploymentID(ctx context.Context, deploymentID string) error {
	query := `DELETE FROM containers WHERE deployment_id = ?`
	_, err := r.db.ExecContext(ctx, query, deploymentID)
	return err
}
