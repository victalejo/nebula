package sqlite

import (
	"context"
	"database/sql"
	"time"
)

// SettingsRepository handles settings persistence
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting value by key
func (r *SettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		"SELECT value FROM settings WHERE key = ?",
		key,
	).Scan(&value)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return value, nil
}

// Set creates or updates a setting value
func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?`,
		key, value, time.Now(), value, time.Now(),
	)
	return err
}

// Delete removes a setting by key
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM settings WHERE key = ?",
		key,
	)
	return err
}
