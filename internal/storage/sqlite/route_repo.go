package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// RouteRepository is the SQLite implementation of RouteRepository
type RouteRepository struct {
	db *sql.DB
}

// NewRouteRepository creates a new route repository
func NewRouteRepository(db *sql.DB) *RouteRepository {
	return &RouteRepository{db: db}
}

// Create creates a new route
func (r *RouteRepository) Create(ctx context.Context, route *storage.Route) error {
	query := `
		INSERT INTO routes (id, app_id, domain, active_slot, ssl_enabled, blue_port, green_port, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	route.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		route.ID,
		route.AppID,
		route.Domain,
		route.ActiveSlot,
		route.SSLEnabled,
		route.BluePort,
		route.GreenPort,
		route.CreatedAt,
	)
	return err
}

// GetByID retrieves a route by ID
func (r *RouteRepository) GetByID(ctx context.Context, id string) (*storage.Route, error) {
	query := `
		SELECT id, app_id, domain, active_slot, ssl_enabled, blue_port, green_port, created_at
		FROM routes
		WHERE id = ?
	`
	route := &storage.Route{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&route.ID,
		&route.AppID,
		&route.Domain,
		&route.ActiveSlot,
		&route.SSLEnabled,
		&route.BluePort,
		&route.GreenPort,
		&route.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return route, nil
}

// GetByDomain retrieves a route by domain
func (r *RouteRepository) GetByDomain(ctx context.Context, domain string) (*storage.Route, error) {
	query := `
		SELECT id, app_id, domain, active_slot, ssl_enabled, blue_port, green_port, created_at
		FROM routes
		WHERE domain = ?
	`
	route := &storage.Route{}
	err := r.db.QueryRowContext(ctx, query, domain).Scan(
		&route.ID,
		&route.AppID,
		&route.Domain,
		&route.ActiveSlot,
		&route.SSLEnabled,
		&route.BluePort,
		&route.GreenPort,
		&route.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return route, nil
}

// GetByAppID retrieves a route by app ID
func (r *RouteRepository) GetByAppID(ctx context.Context, appID string) (*storage.Route, error) {
	query := `
		SELECT id, app_id, domain, active_slot, ssl_enabled, blue_port, green_port, created_at
		FROM routes
		WHERE app_id = ?
	`
	route := &storage.Route{}
	err := r.db.QueryRowContext(ctx, query, appID).Scan(
		&route.ID,
		&route.AppID,
		&route.Domain,
		&route.ActiveSlot,
		&route.SSLEnabled,
		&route.BluePort,
		&route.GreenPort,
		&route.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return route, nil
}

// Update updates a route
func (r *RouteRepository) Update(ctx context.Context, route *storage.Route) error {
	query := `
		UPDATE routes
		SET domain = ?, active_slot = ?, ssl_enabled = ?, blue_port = ?, green_port = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		route.Domain,
		route.ActiveSlot,
		route.SSLEnabled,
		route.BluePort,
		route.GreenPort,
		route.ID,
	)
	return err
}

// Delete deletes a route
func (r *RouteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM routes WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List returns all routes
func (r *RouteRepository) List(ctx context.Context) ([]*storage.Route, error) {
	query := `
		SELECT id, app_id, domain, active_slot, ssl_enabled, blue_port, green_port, created_at
		FROM routes
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []*storage.Route
	for rows.Next() {
		route := &storage.Route{}
		if err := rows.Scan(
			&route.ID,
			&route.AppID,
			&route.Domain,
			&route.ActiveSlot,
			&route.SSLEnabled,
			&route.BluePort,
			&route.GreenPort,
			&route.CreatedAt,
		); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}
