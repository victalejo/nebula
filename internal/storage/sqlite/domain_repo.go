package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// DomainRepository is the SQLite implementation of DomainRepository
type DomainRepository struct {
	db *sql.DB
}

// NewDomainRepository creates a new domain repository
func NewDomainRepository(db *sql.DB) *DomainRepository {
	return &DomainRepository{db: db}
}

// Create creates a new domain
func (r *DomainRepository) Create(ctx context.Context, domain *storage.Domain) error {
	query := `
		INSERT INTO domains (id, project_id, service_id, domain, path_prefix, active_slot, ssl_enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	domain.CreatedAt = now

	if domain.PathPrefix == "" {
		domain.PathPrefix = "/"
	}
	if domain.ActiveSlot == "" {
		domain.ActiveSlot = "blue"
	}

	_, err := r.db.ExecContext(ctx, query,
		domain.ID,
		domain.ProjectID,
		domain.ServiceID,
		domain.Domain,
		domain.PathPrefix,
		domain.ActiveSlot,
		domain.SSLEnabled,
		domain.CreatedAt,
	)
	return err
}

// GetByID retrieves a domain by ID
func (r *DomainRepository) GetByID(ctx context.Context, id string) (*storage.Domain, error) {
	query := `
		SELECT id, project_id, service_id, domain, COALESCE(path_prefix, '/'),
		       COALESCE(active_slot, 'blue'), COALESCE(ssl_enabled, 1), created_at
		FROM domains
		WHERE id = ?
	`
	return r.scanDomain(r.db.QueryRowContext(ctx, query, id))
}

// GetByDomain retrieves a domain by domain name
func (r *DomainRepository) GetByDomain(ctx context.Context, domainName string) (*storage.Domain, error) {
	query := `
		SELECT id, project_id, service_id, domain, COALESCE(path_prefix, '/'),
		       COALESCE(active_slot, 'blue'), COALESCE(ssl_enabled, 1), created_at
		FROM domains
		WHERE domain = ?
	`
	return r.scanDomain(r.db.QueryRowContext(ctx, query, domainName))
}

func (r *DomainRepository) scanDomain(row *sql.Row) (*storage.Domain, error) {
	domain := &storage.Domain{}
	err := row.Scan(
		&domain.ID,
		&domain.ProjectID,
		&domain.ServiceID,
		&domain.Domain,
		&domain.PathPrefix,
		&domain.ActiveSlot,
		&domain.SSLEnabled,
		&domain.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return domain, nil
}

// Update updates a domain
func (r *DomainRepository) Update(ctx context.Context, domain *storage.Domain) error {
	query := `
		UPDATE domains
		SET service_id = ?, domain = ?, path_prefix = ?, active_slot = ?, ssl_enabled = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		domain.ServiceID,
		domain.Domain,
		domain.PathPrefix,
		domain.ActiveSlot,
		domain.SSLEnabled,
		domain.ID,
	)
	return err
}

// Delete deletes a domain
func (r *DomainRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM domains WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByProjectID returns all domains for a project
func (r *DomainRepository) ListByProjectID(ctx context.Context, projectID string) ([]*storage.Domain, error) {
	query := `
		SELECT id, project_id, service_id, domain, COALESCE(path_prefix, '/'),
		       COALESCE(active_slot, 'blue'), COALESCE(ssl_enabled, 1), created_at
		FROM domains
		WHERE project_id = ?
		ORDER BY domain ASC
	`
	return r.scanDomains(r.db.QueryContext(ctx, query, projectID))
}

// ListByServiceID returns all domains for a service
func (r *DomainRepository) ListByServiceID(ctx context.Context, serviceID string) ([]*storage.Domain, error) {
	query := `
		SELECT id, project_id, service_id, domain, COALESCE(path_prefix, '/'),
		       COALESCE(active_slot, 'blue'), COALESCE(ssl_enabled, 1), created_at
		FROM domains
		WHERE service_id = ?
		ORDER BY domain ASC
	`
	return r.scanDomains(r.db.QueryContext(ctx, query, serviceID))
}

// List returns all domains
func (r *DomainRepository) List(ctx context.Context) ([]*storage.Domain, error) {
	query := `
		SELECT id, project_id, service_id, domain, COALESCE(path_prefix, '/'),
		       COALESCE(active_slot, 'blue'), COALESCE(ssl_enabled, 1), created_at
		FROM domains
		ORDER BY domain ASC
	`
	return r.scanDomains(r.db.QueryContext(ctx, query))
}

func (r *DomainRepository) scanDomains(rows *sql.Rows, err error) ([]*storage.Domain, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*storage.Domain
	for rows.Next() {
		domain := &storage.Domain{}
		if err := rows.Scan(
			&domain.ID,
			&domain.ProjectID,
			&domain.ServiceID,
			&domain.Domain,
			&domain.PathPrefix,
			&domain.ActiveSlot,
			&domain.SSLEnabled,
			&domain.CreatedAt,
		); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}
