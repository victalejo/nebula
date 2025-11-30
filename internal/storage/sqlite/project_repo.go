package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/victalejo/nebula/internal/core/storage"
)

// ProjectRepository is the SQLite implementation of ProjectRepository
type ProjectRepository struct {
	db *sql.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, project *storage.Project) error {
	query := `
		INSERT INTO applications (id, name, display_name, description, deployment_mode, domain, git_repo, git_branch, environment, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'git', '', ?, ?, ?, ?, ?)
	`
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		project.ID,
		project.Name,
		nullString(project.DisplayName),
		nullString(project.Description),
		nullString(project.GitRepo),
		nullString(project.GitBranch),
		project.Environment,
		project.CreatedAt,
		project.UpdatedAt,
	)
	return err
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*storage.Project, error) {
	query := `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(git_repo, ''), COALESCE(git_branch, ''), COALESCE(environment, '{}'),
		       created_at, updated_at
		FROM applications
		WHERE id = ?
	`
	project := &storage.Project{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID,
		&project.Name,
		&project.DisplayName,
		&project.Description,
		&project.GitRepo,
		&project.GitBranch,
		&project.Environment,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return project, nil
}

// GetByName retrieves a project by name
func (r *ProjectRepository) GetByName(ctx context.Context, name string) (*storage.Project, error) {
	query := `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(git_repo, ''), COALESCE(git_branch, ''), COALESCE(environment, '{}'),
		       created_at, updated_at
		FROM applications
		WHERE name = ?
	`
	project := &storage.Project{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&project.ID,
		&project.Name,
		&project.DisplayName,
		&project.Description,
		&project.GitRepo,
		&project.GitBranch,
		&project.Environment,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return project, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, project *storage.Project) error {
	query := `
		UPDATE applications
		SET name = ?, display_name = ?, description = ?, git_repo = ?, git_branch = ?, environment = ?, updated_at = ?
		WHERE id = ?
	`
	project.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, query,
		project.Name,
		nullString(project.DisplayName),
		nullString(project.Description),
		nullString(project.GitRepo),
		nullString(project.GitBranch),
		project.Environment,
		project.UpdatedAt,
		project.ID,
	)
	return err
}

// Delete deletes a project
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM applications WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List returns all projects
func (r *ProjectRepository) List(ctx context.Context) ([]*storage.Project, error) {
	query := `
		SELECT id, name, COALESCE(display_name, ''), COALESCE(description, ''),
		       COALESCE(git_repo, ''), COALESCE(git_branch, ''), COALESCE(environment, '{}'),
		       created_at, updated_at
		FROM applications
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*storage.Project
	for rows.Next() {
		project := &storage.Project{}
		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.DisplayName,
			&project.Description,
			&project.GitRepo,
			&project.GitBranch,
			&project.Environment,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}
