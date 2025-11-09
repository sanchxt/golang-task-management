package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"task-management/internal/domain"
	"task-management/internal/fuzzy"
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
	ProjectID   sql.NullInt64  `db:"project_id"`
	ProjectName sql.NullString `db:"project_name"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
	DueDate     sql.NullTime   `db:"due_date"`
}

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

	if dt.Tags.Valid && dt.Tags.String != "" {
		if err := json.Unmarshal([]byte(dt.Tags.String), &task.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
	} else {
		task.Tags = make([]string, 0)
	}

	if dt.ProjectID.Valid {
		task.ProjectID = &dt.ProjectID.Int64
	}

	if dt.ProjectName.Valid {
		task.ProjectName = dt.ProjectName.String
	}

	if dt.DueDate.Valid {
		task.DueDate = &dt.DueDate.Time
	}

	return task, nil
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = time.Now()
	}

	if task.Priority == "" {
		task.Priority = domain.PriorityMedium
	}
	if task.Status == "" {
		task.Status = domain.StatusPending
	}

	query := `
		INSERT INTO tasks (title, description, priority, status, tags, project_id, created_at, updated_at, due_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		string(tagsJSON),
		nullInt64(task.ProjectID),
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

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	query := `
		SELECT
			t.id, t.title, t.description, t.priority, t.status, t.tags,
			t.project_id, p.name as project_name,
			t.created_at, t.updated_at, t.due_date
		FROM tasks t
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE t.id = ?
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

func (r *TaskRepository) Count(ctx context.Context, filter repository.TaskFilter) (int64, error) {
	if filter.SearchMode == "fuzzy" && filter.SearchQuery != "" {
		return r.countWithFuzzySearch(ctx, filter)
	}

	query, args := r.buildWhereClause(filter, true)

	var count int64
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return count, nil
}

func (r *TaskRepository) countWithFuzzySearch(ctx context.Context, filter repository.TaskFilter) (int64, error) {
	filterNoPagination := filter
	filterNoPagination.Limit = 0
	filterNoPagination.Offset = 0

	results, err := r.listWithFuzzySearch(ctx, filterNoPagination)
	if err != nil {
		return 0, err
	}

	return int64(len(results)), nil
}

func (r *TaskRepository) List(ctx context.Context, filter repository.TaskFilter) ([]*domain.Task, error) {
	if filter.SearchMode == "fuzzy" && filter.SearchQuery != "" {
		return r.listWithFuzzySearch(ctx, filter)
	}

	query, args := r.buildWhereClause(filter, false)

	orderClause := r.buildOrderClause(filter)
	query += orderClause

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

func (r *TaskRepository) buildWhereClause(filter repository.TaskFilter, isCount bool) (string, []interface{}) {
	var query string
	if isCount {
		query = `SELECT COUNT(*) FROM tasks t
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE 1=1`
	} else {
		query = `SELECT
			t.id, t.title, t.description, t.priority, t.status, t.tags,
			t.project_id, p.name as project_name,
			t.created_at, t.updated_at, t.due_date
		FROM tasks t
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE 1=1`
	}

	args := make([]interface{}, 0)

	if filter.Status != "" {
		query += " AND t.status = ?"
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		query += " AND t.priority = ?"
		args = append(args, filter.Priority)
	}
	if filter.ProjectID != nil {
		query += " AND t.project_id = ?"
		args = append(args, *filter.ProjectID)
	}

	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query += " AND EXISTS (SELECT 1 FROM json_each(t.tags) WHERE value = ?)"
			args = append(args, tag)
		}
	}

	if len(filter.ExcludeTags) > 0 {
		for _, tag := range filter.ExcludeTags {
			query += " AND NOT EXISTS (SELECT 1 FROM json_each(t.tags) WHERE value = ?)"
			args = append(args, tag)
		}
	}

	if filter.SearchQuery != "" {
		if filter.SearchMode == "regex" {
			query += ` AND (
				t.title REGEXP ? OR
				COALESCE(t.description, '') REGEXP ? OR
				COALESCE(p.name, '') REGEXP ? OR
				COALESCE(t.tags, '') REGEXP ?
			)`
			for range 4 {
				args = append(args, filter.SearchQuery)
			}
		} else {
			searchPattern := "%" + filter.SearchQuery + "%"
			query += ` AND (
				t.title LIKE ? COLLATE NOCASE OR
				COALESCE(t.description, '') LIKE ? COLLATE NOCASE OR
				COALESCE(p.name, '') LIKE ? COLLATE NOCASE OR
				COALESCE(t.tags, '') LIKE ? COLLATE NOCASE
			)`
			for range 4 {
				args = append(args, searchPattern)
			}
		}
	}

	if filter.DueDateFrom != nil {
		if *filter.DueDateFrom == "none" {
			query += " AND t.due_date IS NULL"
		} else {
			query += " AND t.due_date >= ?"
			args = append(args, *filter.DueDateFrom)
		}
	}
	if filter.DueDateTo != nil {
		query += " AND t.due_date <= ?"
		args = append(args, *filter.DueDateTo)
	}

	if filter.CreatedFrom != nil {
		query += " AND t.created_at >= ?"
		args = append(args, *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query += " AND t.created_at <= ?"
		args = append(args, *filter.CreatedTo)
	}

	if filter.UpdatedFrom != nil {
		query += " AND t.updated_at >= ?"
		args = append(args, *filter.UpdatedFrom)
	}
	if filter.UpdatedTo != nil {
		query += " AND t.updated_at <= ?"
		args = append(args, *filter.UpdatedTo)
	}

	return query, args
}

func (r *TaskRepository) buildOrderClause(filter repository.TaskFilter) string {
	sortBy := filter.SortBy
	sortOrder := filter.SortOrder

	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	if sortBy == "priority" {
		return fmt.Sprintf(` ORDER BY
			CASE t.priority
				WHEN 'urgent' THEN 4
				WHEN 'high' THEN 3
				WHEN 'medium' THEN 2
				WHEN 'low' THEN 1
				ELSE 0
			END %s, t.created_at DESC`, sortOrder)
	}

	validColumns := map[string]string{
		"created_at": "t.created_at",
		"updated_at": "t.updated_at",
		"due_date":   "t.due_date",
		"title":      "t.title",
	}

	column, ok := validColumns[sortBy]
	if !ok {
		column = "t.created_at"
	}

	if sortBy == "due_date" {
		if sortOrder == "asc" {
			return fmt.Sprintf(" ORDER BY %s IS NULL, %s ASC", column, column)
		}
		return fmt.Sprintf(" ORDER BY %s IS NULL, %s DESC", column, column)
	}

	return fmt.Sprintf(" ORDER BY %s %s", column, sortOrder)
}

type taskWithScore struct {
	task  *domain.Task
	score int
}

func (r *TaskRepository) listWithFuzzySearch(ctx context.Context, filter repository.TaskFilter) ([]*domain.Task, error) {
	threshold := filter.FuzzyThreshold
	if threshold == 0 {
		threshold = 60
	}

	filterWithoutSearch := filter
	filterWithoutSearch.SearchQuery = ""
	filterWithoutSearch.SearchMode = ""
	filterWithoutSearch.Limit = 0
	filterWithoutSearch.Offset = 0

	query, args := r.buildWhereClause(filterWithoutSearch, false)
	query += " ORDER BY t.created_at DESC"

	var dbTasks []dbTask
	if err := r.db.SelectContext(ctx, &dbTasks, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list tasks for fuzzy search: %w", err)
	}

	candidateTasks := make([]*domain.Task, 0, len(dbTasks))
	for _, dbTask := range dbTasks {
		task, err := dbTask.toTask()
		if err != nil {
			return nil, err
		}
		candidateTasks = append(candidateTasks, task)
	}

	scoredTasks := make([]taskWithScore, 0)
	for _, task := range candidateTasks {
		searchableTexts := []string{
			task.Title,
			task.Description,
			task.ProjectName,
		}

		if len(task.Tags) > 0 {
			searchableTexts = append(searchableTexts, strings.Join(task.Tags, " "))
		}

		bestScore := 0
		for _, text := range searchableTexts {
			if text == "" {
				continue
			}
			score := fuzzy.Match(filter.SearchQuery, text)
			if score > bestScore {
				bestScore = score
			}
		}

		if bestScore >= threshold {
			scoredTasks = append(scoredTasks, taskWithScore{
				task:  task,
				score: bestScore,
			})
		}
	}

	sort.Slice(scoredTasks, func(i, j int) bool {
		return scoredTasks[i].score > scoredTasks[j].score
	})

	start := filter.Offset
	end := len(scoredTasks)

	if start >= end {
		return []*domain.Task{}, nil
	}

	if filter.Limit > 0 {
		end = start + filter.Limit
		if end > len(scoredTasks) {
			end = len(scoredTasks)
		}
	}

	results := make([]*domain.Task, 0, end-start)
	for i := start; i < end; i++ {
		results = append(results, scoredTasks[i].task)
	}

	return results, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	task.UpdatedAt = time.Now()

	query := `
		UPDATE tasks
		SET title = ?, description = ?, priority = ?, status = ?, tags = ?, project_id = ?, updated_at = ?, due_date = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		task.Title,
		task.Description,
		task.Priority,
		task.Status,
		string(tagsJSON),
		nullInt64(task.ProjectID),
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

func (r *TaskRepository) BulkUpdate(ctx context.Context, filter repository.TaskFilter, updates repository.TaskUpdate) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	query := "UPDATE tasks SET updated_at = ?"
	args := []interface{}{time.Now()}

	if updates.Status != nil {
		query += ", status = ?"
		args = append(args, *updates.Status)
	}
	if updates.Priority != nil {
		query += ", priority = ?"
		args = append(args, *updates.Priority)
	}
	if updates.Description != nil {
		query += ", description = ?"
		args = append(args, *updates.Description)
	}
	if updates.ProjectID != nil {
		query += ", project_id = ?"
		if *updates.ProjectID == nil {
			args = append(args, nil)
		} else {
			args = append(args, **updates.ProjectID)
		}
	}
	if updates.DueDate != nil {
		query += ", due_date = ?"
		if *updates.DueDate == nil {
			args = append(args, nil)
		} else {
			args = append(args, **updates.DueDate)
		}
	}

	whereQuery, whereArgs := r.buildBulkWhereClause(filter)
	query += whereQuery
	args = append(args, whereArgs...)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk update tasks: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *TaskRepository) BulkMove(ctx context.Context, filter repository.TaskFilter, projectID *int64) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	query := "UPDATE tasks SET project_id = ?, updated_at = ?"
	args := []interface{}{nullInt64(projectID), time.Now()}

	whereQuery, whereArgs := r.buildBulkWhereClause(filter)
	query += whereQuery
	args = append(args, whereArgs...)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk move tasks: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *TaskRepository) BulkAddTags(ctx context.Context, filter repository.TaskFilter, tags []string) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	tasks, err := r.List(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to get tasks: %w", err)
	}

	var count int64
	for _, task := range tasks {
		tagSet := make(map[string]bool)
		for _, existingTag := range task.Tags {
			tagSet[existingTag] = true
		}
		for _, newTag := range tags {
			tagSet[newTag] = true
		}

		updatedTags := make([]string, 0, len(tagSet))
		for tag := range tagSet {
			updatedTags = append(updatedTags, tag)
		}

		tagsJSON, err := json.Marshal(updatedTags)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal tags: %w", err)
		}

		query := "UPDATE tasks SET tags = ?, updated_at = ? WHERE id = ?"
		result, err := tx.ExecContext(ctx, query, string(tagsJSON), time.Now(), task.ID)
		if err != nil {
			return 0, fmt.Errorf("failed to update task tags: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to get rows affected: %w", err)
		}
		count += rows
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *TaskRepository) BulkRemoveTags(ctx context.Context, filter repository.TaskFilter, tags []string) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	tasks, err := r.List(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to get tasks: %w", err)
	}

	tagsToRemove := make(map[string]bool)
	for _, tag := range tags {
		tagsToRemove[tag] = true
	}

	var count int64
	for _, task := range tasks {
		updatedTags := make([]string, 0)
		for _, existingTag := range task.Tags {
			if !tagsToRemove[existingTag] {
				updatedTags = append(updatedTags, existingTag)
			}
		}

		tagsJSON, err := json.Marshal(updatedTags)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal tags: %w", err)
		}

		query := "UPDATE tasks SET tags = ?, updated_at = ? WHERE id = ?"
		result, err := tx.ExecContext(ctx, query, string(tagsJSON), time.Now(), task.ID)
		if err != nil {
			return 0, fmt.Errorf("failed to update task tags: %w", err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to get rows affected: %w", err)
		}
		count += rows
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *TaskRepository) BulkDelete(ctx context.Context, filter repository.TaskFilter) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	query := "DELETE FROM tasks"
	whereQuery, args := r.buildBulkWhereClause(filter)
	query += whereQuery

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk delete tasks: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return count, nil
}

func (r *TaskRepository) buildBulkWhereClause(filter repository.TaskFilter) (string, []interface{}) {
	query := " WHERE 1=1"
	args := make([]interface{}, 0)

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		query += " AND priority = ?"
		args = append(args, filter.Priority)
	}
	if filter.ProjectID != nil {
		query += " AND project_id = ?"
		args = append(args, *filter.ProjectID)
	}

	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query += " AND EXISTS (SELECT 1 FROM json_each(tags) WHERE value = ?)"
			args = append(args, tag)
		}
	}

	if filter.SearchQuery != "" {
		if filter.SearchMode == "regex" {
			query += ` AND (
				title REGEXP ? OR
				COALESCE(description, '') REGEXP ? OR
				COALESCE(tags, '') REGEXP ?
			)`
			for range 3 {
				args = append(args, filter.SearchQuery)
			}
		} else {
			searchPattern := "%" + filter.SearchQuery + "%"
			query += ` AND (
				title LIKE ? COLLATE NOCASE OR
				COALESCE(description, '') LIKE ? COLLATE NOCASE OR
				COALESCE(tags, '') LIKE ? COLLATE NOCASE
			)`
			for range 3 {
				args = append(args, searchPattern)
			}
		}
	}

	if filter.DueDateFrom != nil {
		if *filter.DueDateFrom == "none" {
			query += " AND due_date IS NULL"
		} else {
			query += " AND due_date >= ?"
			args = append(args, *filter.DueDateFrom)
		}
	}
	if filter.DueDateTo != nil {
		query += " AND due_date <= ?"
		args = append(args, *filter.DueDateTo)
	}

	return query, args
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
