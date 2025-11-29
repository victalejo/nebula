package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// DatabaseRepository is the SQLite implementation of DatabaseRepository
type DatabaseRepository struct {
	db *sql.DB
}

// NewDatabaseRepository creates a new database repository
func NewDatabaseRepository(db *sql.DB) *DatabaseRepository {
	return &DatabaseRepository{db: db}
}

// Create creates a new managed database record
func (r *DatabaseRepository) Create(ctx context.Context, database *storage.Database) error {
	query := `
		INSERT INTO databases (id, app_id, name, type, version, status, container_id, host, port, username, password, database_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	database.CreatedAt = now
	database.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		database.ID,
		database.AppID,
		database.Name,
		database.Type,
		database.Version,
		database.Status,
		database.ContainerID,
		database.Host,
		database.Port,
		database.Username,
		database.Password,
		database.Database,
		database.CreatedAt,
		database.UpdatedAt,
	)
	return err
}

// GetByID retrieves a database by ID
func (r *DatabaseRepository) GetByID(ctx context.Context, id string) (*storage.Database, error) {
	query := `
		SELECT id, app_id, name, type, version, status, container_id, host, port, username, password, database_name, created_at, updated_at
		FROM databases
		WHERE id = ?
	`
	d := &storage.Database{}
	var appID sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&appID,
		&d.Name,
		&d.Type,
		&d.Version,
		&d.Status,
		&d.ContainerID,
		&d.Host,
		&d.Port,
		&d.Username,
		&d.Password,
		&d.Database,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if appID.Valid {
		d.AppID = &appID.String
	}
	return d, nil
}

// GetByName retrieves a database by name
func (r *DatabaseRepository) GetByName(ctx context.Context, name string) (*storage.Database, error) {
	query := `
		SELECT id, app_id, name, type, version, status, container_id, host, port, username, password, database_name, created_at, updated_at
		FROM databases
		WHERE name = ?
	`
	d := &storage.Database{}
	var appID sql.NullString
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&d.ID,
		&appID,
		&d.Name,
		&d.Type,
		&d.Version,
		&d.Status,
		&d.ContainerID,
		&d.Host,
		&d.Port,
		&d.Username,
		&d.Password,
		&d.Database,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if appID.Valid {
		d.AppID = &appID.String
	}
	return d, nil
}

// Update updates a database record
func (r *DatabaseRepository) Update(ctx context.Context, database *storage.Database) error {
	query := `
		UPDATE databases
		SET status = ?, container_id = ?, host = ?, port = ?, username = ?, password = ?, database_name = ?, updated_at = ?
		WHERE id = ?
	`
	database.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		database.Status,
		database.ContainerID,
		database.Host,
		database.Port,
		database.Username,
		database.Password,
		database.Database,
		database.UpdatedAt,
		database.ID,
	)
	return err
}

// Delete deletes a database record
func (r *DatabaseRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM databases WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List returns all databases
func (r *DatabaseRepository) List(ctx context.Context) ([]*storage.Database, error) {
	query := `
		SELECT id, app_id, name, type, version, status, container_id, host, port, username, password, database_name, created_at, updated_at
		FROM databases
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []*storage.Database
	for rows.Next() {
		d := &storage.Database{}
		var appID sql.NullString
		if err := rows.Scan(
			&d.ID,
			&appID,
			&d.Name,
			&d.Type,
			&d.Version,
			&d.Status,
			&d.ContainerID,
			&d.Host,
			&d.Port,
			&d.Username,
			&d.Password,
			&d.Database,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if appID.Valid {
			d.AppID = &appID.String
		}
		databases = append(databases, d)
	}
	return databases, rows.Err()
}

// ListByAppID returns all databases for an application
func (r *DatabaseRepository) ListByAppID(ctx context.Context, appID string) ([]*storage.Database, error) {
	query := `
		SELECT id, app_id, name, type, version, status, container_id, host, port, username, password, database_name, created_at, updated_at
		FROM databases
		WHERE app_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []*storage.Database
	for rows.Next() {
		d := &storage.Database{}
		var appIDVal sql.NullString
		if err := rows.Scan(
			&d.ID,
			&appIDVal,
			&d.Name,
			&d.Type,
			&d.Version,
			&d.Status,
			&d.ContainerID,
			&d.Host,
			&d.Port,
			&d.Username,
			&d.Password,
			&d.Database,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if appIDVal.Valid {
			d.AppID = &appIDVal.String
		}
		databases = append(databases, d)
	}
	return databases, rows.Err()
}

// BackupRepository is the SQLite implementation of BackupRepository
type BackupRepository struct {
	db *sql.DB
}

// NewBackupRepository creates a new backup repository
func NewBackupRepository(db *sql.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

// Create creates a new backup record
func (r *BackupRepository) Create(ctx context.Context, backup *storage.DatabaseBackup) error {
	query := `
		INSERT INTO database_backups (id, database_id, path, size_bytes, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	backup.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		backup.ID,
		backup.DatabaseID,
		backup.Path,
		backup.SizeBytes,
		backup.CreatedAt,
	)
	return err
}

// GetByID retrieves a backup by ID
func (r *BackupRepository) GetByID(ctx context.Context, id string) (*storage.DatabaseBackup, error) {
	query := `
		SELECT id, database_id, path, size_bytes, created_at
		FROM database_backups
		WHERE id = ?
	`
	b := &storage.DatabaseBackup{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID,
		&b.DatabaseID,
		&b.Path,
		&b.SizeBytes,
		&b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Delete deletes a backup record
func (r *BackupRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM database_backups WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ListByDatabaseID returns all backups for a database
func (r *BackupRepository) ListByDatabaseID(ctx context.Context, databaseID string) ([]*storage.DatabaseBackup, error) {
	query := `
		SELECT id, database_id, path, size_bytes, created_at
		FROM database_backups
		WHERE database_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, databaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []*storage.DatabaseBackup
	for rows.Next() {
		b := &storage.DatabaseBackup{}
		if err := rows.Scan(
			&b.ID,
			&b.DatabaseID,
			&b.Path,
			&b.SizeBytes,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}
