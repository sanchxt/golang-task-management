package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type TaskRepository struct {
	db *DB
}

func NewTaskRepository(db *DB) *TaskRepository {
	return &TaskRepository{db: db}
}

type dbTask struct {
	ID          int64          `db:"id"`
	Title       string         `db:"title"`
	Description string         `db:"description"`
	Priority    string         `db:"priority"`
	Status      string         `db:"status"`
	Tags        sql.NullString `db:"tags"`
	Project     sql.NullString `db:"project"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	DueDate     sql.NullTime   `db:"due_date"`
}

// converts dbTask to a domain.Task
func (dt *dbTask) toTask() (*domain.Task, error) {
	task := &domain.Task{
		ID:          dt.ID,
		Title:       dt.Title,
		Description: dt.Description,
		Priority:    domain.Priority(dt.Priority),
		Status:      domain.Status(dt.Status),
		CreatedAt:   dt.CreatedAt,
		UpdatedAt:   dt.UpdatedAt,
	}

	// parse tags JSON
	if dt.Tags.Valid && dt.Tags.String != "" {
		if err := json.Unmarshal([]byte(dt.Tags.String), &task.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
	} else {
		task.Tags = make([]string, 0)
	}

	// handle nullable project
	if dt.Project.Valid {
		task.Project = dt.Project.String
	}

	// handle nullable due date
	if dt.DueDate.Valid {
		task.DueDate = &dt.DueDate.Time
	}

	return task, nil
}

// insert a new task
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// serialize tags to JSON
	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	// set timestamps
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = time.Now()
	}

	// set default values
	if task.Priority == "" {
		task.Priority = domain.PriorityMedium
	}
	if task.Status == "" {
		task.Status = domain.StatusPending
	}

	query := `
		INSERT INTO tasks (title, description, priority, status, tags, project, created_at, updated_at, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		string(tagsJSON),
		nullString(task.Project),
		task.CreatedAt,
		task.UpdatedAt,
		nullTime(task.DueDate),
	)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	task.ID = id
	return nil
}

// get a task by its ID
func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	query := `
		SELECT id, title, description, priority, status, tags, project, created_at, updated_at, due_date
		FROM tasks
		WHERE id = ?
	`

	var dbTask dbTask
	if err := r.db.GetContext(ctx, &dbTask, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return dbTask.toTask()
}

// count tasks with filtering (for pagination)
func (r *TaskRepository) Count(ctx context.Context, filter repository.TaskFilter) (int64, error) {
	query, args := r.buildWhereClause(filter, true)

	var count int64
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return count, nil
}

// get all tasks (with filters)
func (r *TaskRepository) List(ctx context.Context, filter repository.TaskFilter) ([]*domain.Task, error) {
	query, args := r.buildWhereClause(filter, false)

	// add sorting
	orderClause := r.buildOrderClause(filter)
	query += orderClause

	// add pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	var dbTasks []dbTask
	if err := r.db.SelectContext(ctx, &dbTasks, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	tasks := make([]*domain.Task, 0, len(dbTasks))
	for _, dbTask := range dbTasks {
		task, err := dbTask.toTask()
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// constructs the WHERE clause with all filters
func (r *TaskRepository) buildWhereClause(filter repository.TaskFilter, isCount bool) (string, []interface{}) {
	var query string
	if isCount {
		query = "SELECT COUNT(*) FROM tasks WHERE 1=1"
	} else {
		query = "SELECT id, title, description, priority, status, tags, project, created_at, updated_at, due_date FROM tasks WHERE 1=1"
	}

	args := make([]interface{}, 0)

	// basic filters
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filter.Priority)
	}
	if filter.Project != "" {
		query += " AND project = ?"
		args = append(args, filter.Project)
	}

	// tag filtering using JSON functions
	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query += " AND EXISTS (SELECT 1 FROM json_each(tasks.tags) WHERE value = ?)"
			args = append(args, tag)
		}
	}

	// search query
	if filter.SearchQuery != "" {
		if filter.SearchMode == "regex" {
			query += ` AND (
				title REGEXP ? OR
				COALESCE(description, '') REGEXP ? OR
				COALESCE(project, '') REGEXP ? OR
				COALESCE(tags, '') REGEXP ?
			)`
			for range 4 {
				args = append(args, filter.SearchQuery)
			}
		} else {
			searchPattern := "%" + filter.SearchQuery + "%"
			query += ` AND (
				title LIKE ? COLLATE NOCASE OR
				COALESCE(description, '') LIKE ? COLLATE NOCASE OR
				COALESCE(project, '') LIKE ? COLLATE NOCASE OR
				COALESCE(tags, '') LIKE ? COLLATE NOCASE
			)`
			for range 4 {
				args = append(args, searchPattern)
			}
		}
	}

	// date range filtering
	if filter.DueDateFrom != nil {
		query += " AND due_date >= ?"
		args = append(args, *filter.DueDateFrom)
	}
	if filter.DueDateTo != nil {
		query += " AND due_date <= ?"
		args = append(args, *filter.DueDateTo)
	}

	return query, args
}

// constructs the ORDER BY clause
func (r *TaskRepository) buildOrderClause(filter repository.TaskFilter) string {
	sortBy := filter.SortBy
	sortOrder := filter.SortOrder

	// default sort
	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// validate sortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	// map priority to numeric values for sorting
	if sortBy == "priority" {
		return fmt.Sprintf(` ORDER BY
			CASE priority
				WHEN 'urgent' THEN 4
				WHEN 'high' THEN 3
				WHEN 'medium' THEN 2
				WHEN 'low' THEN 1
				ELSE 0
			END %s, created_at DESC`, sortOrder)
	}

	// map to actual column names
	validColumns := map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"due_date":   "due_date",
		"title":      "title",
	}

	column, ok := validColumns[sortBy]
	if !ok {
		column = "created_at"
	}

	// handle null values for due_date (nulls last)
	if column == "due_date" {
		if sortOrder == "asc" {
			return fmt.Sprintf(" ORDER BY %s IS NULL, %s ASC", column, column)
		}
		return fmt.Sprintf(" ORDER BY %s IS NULL, %s DESC", column, column)
	}

	return fmt.Sprintf(" ORDER BY %s %s", column, sortOrder)
}

// modify a task
func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// serialize tags to JSON
	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	task.UpdatedAt = time.Now()

	query := `
		UPDATE tasks
		SET title = ?, description = ?, priority = ?, status = ?, tags = ?, project = ?, updated_at = ?, due_date = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		string(tagsJSON),
		nullString(task.Project),
		task.UpdatedAt,
		nullTime(task.DueDate),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found: %d", task.ID)
	}

	return nil
}

// remove a task
func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM tasks WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task not found: %d", id)
	}

	return nil
}

// helpers
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
