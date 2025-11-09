package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type TemplateRepository struct {
	db *DB
}

func NewTemplateRepository(db *DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

type dbTemplate struct {
	ID               int64          `db:"id"`
	Name             string         `db:"name"`
	Description      sql.NullString `db:"description"`
	TaskDefinitions  string         `db:"task_definitions"`
	ProjectDefaults  sql.NullString `db:"project_defaults"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

func (dt *dbTemplate) toTemplate() (*domain.ProjectTemplate, error) {
	template := &domain.ProjectTemplate{
		ID:   dt.ID,
		Name: dt.Name,
	}

	if dt.Description.Valid {
		template.Description = dt.Description.String
	}

	if err := json.Unmarshal([]byte(dt.TaskDefinitions), &template.TaskDefinitions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task definitions: %w", err)
	}

	if dt.ProjectDefaults.Valid && dt.ProjectDefaults.String != "" {
		var defaults domain.ProjectDefaults
		if err := json.Unmarshal([]byte(dt.ProjectDefaults.String), &defaults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal project defaults: %w", err)
		}
		template.ProjectDefaults = &defaults
	}

	template.CreatedAt = dt.CreatedAt
	template.UpdatedAt = dt.UpdatedAt

	return template, nil
}

func (r *TemplateRepository) Create(ctx context.Context, template *domain.ProjectTemplate) error {
	if err := template.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	taskDefsJSON, err := json.Marshal(template.TaskDefinitions)
	if err != nil {
		return fmt.Errorf("failed to marshal task definitions: %w", err)
	}

	var projectDefaultsJSON sql.NullString
	if template.ProjectDefaults != nil {
		defaultsBytes, err := json.Marshal(template.ProjectDefaults)
		if err != nil {
			return fmt.Errorf("failed to marshal project defaults: %w", err)
		}
		projectDefaultsJSON = sql.NullString{String: string(defaultsBytes), Valid: true}
	}

	query := `
		INSERT INTO project_templates (name, description, task_definitions, project_defaults)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		template.Name,
		nullString(template.Description),
		string(taskDefsJSON),
		projectDefaultsJSON,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("template with name %q already exists", template.Name)
		}
		return fmt.Errorf("failed to insert template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	template.ID = id

	created, err := r.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch created template: %w", err)
	}

	template.CreatedAt = created.CreatedAt
	template.UpdatedAt = created.UpdatedAt

	return nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*domain.ProjectTemplate, error) {
	query := `
		SELECT id, name, description, task_definitions, project_defaults, created_at, updated_at
		FROM project_templates
		WHERE id = ?
	`

	var dt dbTemplate
	err := r.db.GetContext(ctx, &dt, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("template with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return dt.toTemplate()
}

func (r *TemplateRepository) GetByName(ctx context.Context, name string) (*domain.ProjectTemplate, error) {
	query := `
		SELECT id, name, description, task_definitions, project_defaults, created_at, updated_at
		FROM project_templates
		WHERE name = ?
	`

	var dt dbTemplate
	err := r.db.GetContext(ctx, &dt, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("template with name %q not found", name)
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	return dt.toTemplate()
}

func (r *TemplateRepository) Update(ctx context.Context, template *domain.ProjectTemplate) error {
	if err := template.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	taskDefsJSON, err := json.Marshal(template.TaskDefinitions)
	if err != nil {
		return fmt.Errorf("failed to marshal task definitions: %w", err)
	}

	var projectDefaultsJSON sql.NullString
	if template.ProjectDefaults != nil {
		defaultsBytes, err := json.Marshal(template.ProjectDefaults)
		if err != nil {
			return fmt.Errorf("failed to marshal project defaults: %w", err)
		}
		projectDefaultsJSON = sql.NullString{String: string(defaultsBytes), Valid: true}
	}

	query := `
		UPDATE project_templates
		SET name = ?, description = ?, task_definitions = ?, project_defaults = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		template.Name,
		nullString(template.Description),
		string(taskDefsJSON),
		projectDefaultsJSON,
		template.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("template with name %q already exists", template.Name)
		}
		return fmt.Errorf("failed to update template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template with ID %d not found", template.ID)
	}

	updated, err := r.GetByID(ctx, template.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch updated template: %w", err)
	}

	template.UpdatedAt = updated.UpdatedAt

	return nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM project_templates WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("template with ID %d not found", id)
	}

	return nil
}

func (r *TemplateRepository) List(ctx context.Context, filter repository.TemplateFilter) ([]*domain.ProjectTemplate, error) {
	query := `
		SELECT id, name, description, task_definitions, project_defaults, created_at, updated_at
		FROM project_templates
	`

	var conditions []string
	var args []interface{}

	if filter.SearchQuery != "" {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		searchPattern := "%" + filter.SearchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	sortBy := "created_at"
	if filter.SortBy != "" {
		switch filter.SortBy {
		case "name", "created_at", "updated_at":
			sortBy = filter.SortBy
		case "task_count":
			sortBy = "json_array_length(task_definitions)"
		}
	}

	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	var dbTemplates []dbTemplate
	err := r.db.SelectContext(ctx, &dbTemplates, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	templates := make([]*domain.ProjectTemplate, 0, len(dbTemplates))
	for _, dt := range dbTemplates {
		template, err := dt.toTemplate()
		if err != nil {
			return nil, fmt.Errorf("failed to convert template: %w", err)
		}
		templates = append(templates, template)
	}

	return templates, nil
}

func (r *TemplateRepository) Count(ctx context.Context, filter repository.TemplateFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM project_templates`

	var conditions []string
	var args []interface{}

	if filter.SearchQuery != "" {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		searchPattern := "%" + filter.SearchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count templates: %w", err)
	}

	return count, nil
}

func (r *TemplateRepository) Search(ctx context.Context, query string, limit int) ([]*domain.ProjectTemplate, error) {
	filter := repository.TemplateFilter{
		SearchQuery: query,
		SortBy:      "name",
		SortOrder:   "asc",
		Limit:       limit,
	}

	return r.List(ctx, filter)
}
