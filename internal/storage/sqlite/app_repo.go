package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// AppRepository is the SQLite implementation of AppRepository
type AppRepository struct {
	db *sql.DB
}

// NewAppRepository creates a new app repository
func NewAppRepository(db *sql.DB) *AppRepository {
	return &AppRepository{db: db}
}

// Create creates a new application
func (r *AppRepository) Create(ctx context.Context, app *storage.Application) error {
	query := `
		INSERT INTO applications (id, name, deployment_mode, domain, environment, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	app.CreatedAt = now
	app.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		app.ID,
		app.Name,
		app.DeploymentMode,
		app.Domain,
		app.Environment,
		app.CreatedAt,
		app.UpdatedAt,
	)
	return err
}

// GetByID retrieves an application by ID
func (r *AppRepository) GetByID(ctx context.Context, id string) (*storage.Application, error) {
	query := `
		SELECT id, name, deployment_mode, domain, environment, created_at, updated_at
		FROM applications
		WHERE id = ?
	`
	app := &storage.Application{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&app.ID,
		&app.Name,
		&app.DeploymentMode,
		&app.Domain,
		&app.Environment,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return app, nil
}

// GetByName retrieves an application by name
func (r *AppRepository) GetByName(ctx context.Context, name string) (*storage.Application, error) {
	query := `
		SELECT id, name, deployment_mode, domain, environment, created_at, updated_at
		FROM applications
		WHERE name = ?
	`
	app := &storage.Application{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&app.ID,
		&app.Name,
		&app.DeploymentMode,
		&app.Domain,
		&app.Environment,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return app, nil
}

// Update updates an application
func (r *AppRepository) Update(ctx context.Context, app *storage.Application) error {
	query := `
		UPDATE applications
		SET name = ?, deployment_mode = ?, domain = ?, environment = ?, updated_at = ?
		WHERE id = ?
	`
	app.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		app.Name,
		app.DeploymentMode,
		app.Domain,
		app.Environment,
		app.UpdatedAt,
		app.ID,
	)
	return err
}

// Delete deletes an application
func (r *AppRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM applications WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List returns all applications
func (r *AppRepository) List(ctx context.Context) ([]*storage.Application, error) {
	query := `
		SELECT id, name, deployment_mode, domain, environment, created_at, updated_at
		FROM applications
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*storage.Application
	for rows.Next() {
		app := &storage.Application{}
		if err := rows.Scan(
			&app.ID,
			&app.Name,
			&app.DeploymentMode,
			&app.Domain,
			&app.Environment,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}
