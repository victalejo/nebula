package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// DeploymentRepository is the SQLite implementation of DeploymentRepository
type DeploymentRepository struct {
	db *sql.DB
}

// NewDeploymentRepository creates a new deployment repository
func NewDeploymentRepository(db *sql.DB) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

// Create creates a new deployment
func (r *DeploymentRepository) Create(ctx context.Context, deployment *storage.Deployment) error {
	query := `
		INSERT INTO deployments (id, app_id, service_id, version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	deployment.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		deployment.ID,
		deployment.AppID,
		nullString(deployment.ServiceID),
		deployment.Version,
		deployment.Slot,
		deployment.Status,
		deployment.SourceConfig,
		deployment.Environment,
		deployment.ErrorMessage,
		deployment.CreatedAt,
		deployment.StartedAt,
		deployment.FinishedAt,
	)
	return err
}

// GetByID retrieves a deployment by ID
func (r *DeploymentRepository) GetByID(ctx context.Context, id string) (*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE id = ?
	`
	d := &storage.Deployment{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.AppID,
		&d.ServiceID,
		&d.Version,
		&d.Slot,
		&d.Status,
		&d.SourceConfig,
		&d.Environment,
		&d.ErrorMessage,
		&d.CreatedAt,
		&d.StartedAt,
		&d.FinishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Update updates a deployment
func (r *DeploymentRepository) Update(ctx context.Context, deployment *storage.Deployment) error {
	query := `
		UPDATE deployments
		SET status = ?, error_message = ?, started_at = ?, finished_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		deployment.Status,
		deployment.ErrorMessage,
		deployment.StartedAt,
		deployment.FinishedAt,
		deployment.ID,
	)
	return err
}

// Delete deletes a deployment
func (r *DeploymentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM deployments WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByAppID returns all deployments for an application
func (r *DeploymentRepository) ListByAppID(ctx context.Context, appID string) ([]*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE app_id = ?
		ORDER BY created_at DESC
	`
	return r.scanDeployments(r.db.QueryContext(ctx, query, appID))
}

// ListByServiceID returns all deployments for a service
func (r *DeploymentRepository) ListByServiceID(ctx context.Context, serviceID string) ([]*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE service_id = ?
		ORDER BY created_at DESC
	`
	return r.scanDeployments(r.db.QueryContext(ctx, query, serviceID))
}

func (r *DeploymentRepository) scanDeployments(rows *sql.Rows, err error) ([]*storage.Deployment, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []*storage.Deployment
	for rows.Next() {
		d := &storage.Deployment{}
		if err := rows.Scan(
			&d.ID,
			&d.AppID,
			&d.ServiceID,
			&d.Version,
			&d.Slot,
			&d.Status,
			&d.SourceConfig,
			&d.Environment,
			&d.ErrorMessage,
			&d.CreatedAt,
			&d.StartedAt,
			&d.FinishedAt,
		); err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, rows.Err()
}

// GetLatestByAppID returns the latest deployment for an application
func (r *DeploymentRepository) GetLatestByAppID(ctx context.Context, appID string) (*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE app_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	d := &storage.Deployment{}
	err := r.db.QueryRowContext(ctx, query, appID).Scan(
		&d.ID,
		&d.AppID,
		&d.ServiceID,
		&d.Version,
		&d.Slot,
		&d.Status,
		&d.SourceConfig,
		&d.Environment,
		&d.ErrorMessage,
		&d.CreatedAt,
		&d.StartedAt,
		&d.FinishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetLatestByServiceID returns the latest deployment for a service
func (r *DeploymentRepository) GetLatestByServiceID(ctx context.Context, serviceID string) (*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE service_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	d := &storage.Deployment{}
	err := r.db.QueryRowContext(ctx, query, serviceID).Scan(
		&d.ID,
		&d.AppID,
		&d.ServiceID,
		&d.Version,
		&d.Slot,
		&d.Status,
		&d.SourceConfig,
		&d.Environment,
		&d.ErrorMessage,
		&d.CreatedAt,
		&d.StartedAt,
		&d.FinishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetByAppIDAndSlot returns the deployment for an application and slot
func (r *DeploymentRepository) GetByAppIDAndSlot(ctx context.Context, appID string, slot string) (*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE app_id = ? AND slot = ? AND status = 'running'
		ORDER BY created_at DESC
		LIMIT 1
	`
	d := &storage.Deployment{}
	err := r.db.QueryRowContext(ctx, query, appID, slot).Scan(
		&d.ID,
		&d.AppID,
		&d.ServiceID,
		&d.Version,
		&d.Slot,
		&d.Status,
		&d.SourceConfig,
		&d.Environment,
		&d.ErrorMessage,
		&d.CreatedAt,
		&d.StartedAt,
		&d.FinishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetByServiceIDAndSlot returns the deployment for a service and slot
func (r *DeploymentRepository) GetByServiceIDAndSlot(ctx context.Context, serviceID string, slot string) (*storage.Deployment, error) {
	query := `
		SELECT id, app_id, COALESCE(service_id, ''), version, slot, status, source_config, environment, error_message, created_at, started_at, finished_at
		FROM deployments
		WHERE service_id = ? AND slot = ? AND status = 'running'
		ORDER BY created_at DESC
		LIMIT 1
	`
	d := &storage.Deployment{}
	err := r.db.QueryRowContext(ctx, query, serviceID, slot).Scan(
		&d.ID,
		&d.AppID,
		&d.ServiceID,
		&d.Version,
		&d.Slot,
		&d.Status,
		&d.SourceConfig,
		&d.Environment,
		&d.ErrorMessage,
		&d.CreatedAt,
		&d.StartedAt,
		&d.FinishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}
