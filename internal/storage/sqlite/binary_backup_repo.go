package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// BinaryBackupRepository is the SQLite implementation of BinaryBackupRepository
type BinaryBackupRepository struct {
	db *sql.DB
}

// NewBinaryBackupRepository creates a new binary backup repository
func NewBinaryBackupRepository(db *sql.DB) *BinaryBackupRepository {
	return &BinaryBackupRepository{db: db}
}

// Create creates a new binary backup record
func (r *BinaryBackupRepository) Create(ctx context.Context, backup *storage.BinaryBackup) error {
	query := `
		INSERT INTO binary_backups (id, version, binary_path, binary_hash, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	backup.CreatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		backup.ID,
		backup.Version,
		backup.BinaryPath,
		backup.BinaryHash,
		backup.CreatedAt,
	)
	return err
}

// Get retrieves a binary backup by ID
func (r *BinaryBackupRepository) Get(ctx context.Context, id string) (*storage.BinaryBackup, error) {
	query := `
		SELECT id, version, binary_path, binary_hash, created_at
		FROM binary_backups
		WHERE id = ?
	`
	backup := &storage.BinaryBackup{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&backup.ID,
		&backup.Version,
		&backup.BinaryPath,
		&backup.BinaryHash,
		&backup.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return backup, nil
}

// Delete deletes a binary backup record
func (r *BinaryBackupRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM binary_backups WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List returns all binary backups ordered by creation date
func (r *BinaryBackupRepository) List(ctx context.Context) ([]*storage.BinaryBackup, error) {
	query := `
		SELECT id, version, binary_path, binary_hash, created_at
		FROM binary_backups
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []*storage.BinaryBackup
	for rows.Next() {
		backup := &storage.BinaryBackup{}
		if err := rows.Scan(
			&backup.ID,
			&backup.Version,
			&backup.BinaryPath,
			&backup.BinaryHash,
			&backup.CreatedAt,
		); err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}
	return backups, rows.Err()
}
