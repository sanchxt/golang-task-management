package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type ProjectRepository struct {
	db *DB
}

func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

type dbProject struct {
	ID          int64          `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"`
	ParentID    sql.NullInt64  `db:"parent_id"`
	Color       sql.NullString `db:"color"`
	Icon        sql.NullString `db:"icon"`
	Status      string         `db:"status"`
	IsFavorite  bool           `db:"is_favorite"`
	Aliases     sql.NullString `db:"aliases"`
	Notes       sql.NullString `db:"notes"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

func (dp *dbProject) toProject() (*domain.Project, error) {
	project := &domain.Project{
		ID:         dp.ID,
		Name:       dp.Name,
		Status:     domain.ProjectStatus(dp.Status),
		IsFavorite: dp.IsFavorite,
		CreatedAt:  dp.CreatedAt,
		UpdatedAt:  dp.UpdatedAt,
	}

	if dp.Description.Valid {
		project.Description = dp.Description.String
	}

	if dp.ParentID.Valid {
		project.ParentID = &dp.ParentID.Int64
	}

	if dp.Color.Valid {
		project.Color = dp.Color.String
	}

	if dp.Icon.Valid {
		project.Icon = dp.Icon.String
	}

	if dp.Aliases.Valid && dp.Aliases.String != "" {
		if err := json.Unmarshal([]byte(dp.Aliases.String), &project.Aliases); err != nil {
			return nil, fmt.Errorf("failed to parse aliases: %w", err)
		}
	} else {
		project.Aliases = make([]string, 0)
	}

	if dp.Notes.Valid {
		project.Notes = dp.Notes.String
	}

	return project, nil
}

func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if err := project.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	for _, alias := range project.Aliases {
		if err := r.ValidateAliasUniqueness(ctx, alias, nil); err != nil {
			return err
		}
	}

	if project.ParentID != nil {
		if err := r.ValidateHierarchy(ctx, 0, *project.ParentID); err != nil {
			return err
		}
	}

	aliasesJSON, err := json.Marshal(project.Aliases)
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if project.CreatedAt.IsZero() {
		project.CreatedAt = time.Now()
	}
	if project.UpdatedAt.IsZero() {
		project.UpdatedAt = time.Now()
	}

	if project.Status == "" {
		project.Status = domain.ProjectStatusActive
	}

	query := `
		INSERT INTO projects (name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		project.Name,
		nullString(project.Description),
		nullInt64(project.ParentID),
		nullString(project.Color),
		nullString(project.Icon),
		project.Status,
		project.IsFavorite,
		string(aliasesJSON),
		nullString(project.Notes),
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	project.ID = id
	return nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, id int64) (*domain.Project, error) {
	query := `
		SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at
		FROM projects
		WHERE id = ?
	`

	var dbProj dbProject
	if err := r.db.GetContext(ctx, &dbProj, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return dbProj.toProject()
}

func (r *ProjectRepository) GetByName(ctx context.Context, name string) (*domain.Project, error) {
	query := `
		SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at
		FROM projects
		WHERE name = ?
	`

	var dbProj dbProject
	if err := r.db.GetContext(ctx, &dbProj, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return dbProj.toProject()
}

func (r *ProjectRepository) List(ctx context.Context, filter repository.ProjectFilter) ([]*domain.Project, error) {
	query, args := r.buildWhereClause(filter, false)

	query += r.buildOrderClause(filter)

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	var dbProjects []dbProject
	if err := r.db.SelectContext(ctx, &dbProjects, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	projects := make([]*domain.Project, 0, len(dbProjects))
	for _, dbProj := range dbProjects {
		project, err := dbProj.toProject()
		if err != nil {
			return nil, fmt.Errorf("failed to convert project: %w", err)
		}

		if filter.IncludeTaskCount {
			count, err := r.GetTaskCount(ctx, project.ID)
			if err != nil {
				return nil, err
			}
			project.TaskCount = count
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepository) ListWithHierarchy(ctx context.Context, filter repository.ProjectFilter) ([]*domain.Project, error) {
	projects, err := r.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	projectMap := make(map[int64]*domain.Project)
	var roots []*domain.Project

	for _, p := range projects {
		projectMap[p.ID] = p
		p.Children = make([]*domain.Project, 0)

		if p.IsRoot() {
			roots = append(roots, p)
		}
	}

	for _, p := range projects {
		if p.ParentID != nil {
			if parent, ok := projectMap[*p.ParentID]; ok {
				parent.Children = append(parent.Children, p)
				p.Parent = parent
			}
		}
	}

	for _, p := range projects {
		p.Path = p.BuildPath()
	}

	return roots, nil
}

func (r *ProjectRepository) GetChildren(ctx context.Context, parentID int64) ([]*domain.Project, error) {
	filter := repository.ProjectFilter{
		ParentID: &parentID,
	}

	return r.List(ctx, filter)
}

func (r *ProjectRepository) GetDescendants(ctx context.Context, parentID int64) ([]*domain.Project, error) {
	query := `
		WITH RECURSIVE descendants AS (
			SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at
			FROM projects
			WHERE parent_id = ?

			UNION ALL

			SELECT p.id, p.name, p.description, p.parent_id, p.color, p.icon, p.status, p.is_favorite, p.aliases, p.notes, p.created_at, p.updated_at
			FROM projects p
			INNER JOIN descendants d ON p.parent_id = d.id
		)
		SELECT * FROM descendants
		ORDER BY name
	`

	var dbProjects []dbProject
	if err := r.db.SelectContext(ctx, &dbProjects, query, parentID); err != nil {
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}

	projects := make([]*domain.Project, 0, len(dbProjects))
	for _, dbProj := range dbProjects {
		project, err := dbProj.toProject()
		if err != nil {
			return nil, fmt.Errorf("failed to convert project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepository) GetPath(ctx context.Context, projectID int64) ([]*domain.Project, error) {
	query := `
		WITH RECURSIVE path AS (
			SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at, 0 as level
			FROM projects
			WHERE id = ?

			UNION ALL

			SELECT p.id, p.name, p.description, p.parent_id, p.color, p.icon, p.status, p.is_favorite, p.aliases, p.notes, p.created_at, p.updated_at, path.level + 1
			FROM projects p
			INNER JOIN path ON p.id = path.parent_id
		)
		SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at FROM path
		ORDER BY level DESC
	`

	var dbProjects []dbProject
	if err := r.db.SelectContext(ctx, &dbProjects, query, projectID); err != nil {
		return nil, fmt.Errorf("failed to get path: %w", err)
	}

	projects := make([]*domain.Project, 0, len(dbProjects))
	for _, dbProj := range dbProjects {
		project, err := dbProj.toProject()
		if err != nil {
			return nil, fmt.Errorf("failed to convert project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepository) GetRoots(ctx context.Context) ([]*domain.Project, error) {
	query := `
		SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at
		FROM projects
		WHERE parent_id IS NULL
		ORDER BY name
	`

	var dbProjects []dbProject
	if err := r.db.SelectContext(ctx, &dbProjects, query); err != nil {
		return nil, fmt.Errorf("failed to get roots: %w", err)
	}

	projects := make([]*domain.Project, 0, len(dbProjects))
	for _, dbProj := range dbProjects {
		project, err := dbProj.toProject()
		if err != nil {
			return nil, fmt.Errorf("failed to convert project: %w", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *ProjectRepository) Count(ctx context.Context, filter repository.ProjectFilter) (int64, error) {
	query, args := r.buildWhereClause(filter, true)

	var count int64
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count projects: %w", err)
	}

	return count, nil
}

func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	if err := project.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	for _, alias := range project.Aliases {
		if err := r.ValidateAliasUniqueness(ctx, alias, &project.ID); err != nil {
			return err
		}
	}

	if project.ParentID != nil {
		if err := r.ValidateHierarchy(ctx, project.ID, *project.ParentID); err != nil {
			return err
		}
	}

	aliasesJSON, err := json.Marshal(project.Aliases)
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	project.UpdatedAt = time.Now()

	query := `
		UPDATE projects
		SET name = ?, description = ?, parent_id = ?, color = ?, icon = ?, status = ?, is_favorite = ?, aliases = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		project.Name,
		nullString(project.Description),
		nullInt64(project.ParentID),
		nullString(project.Color),
		nullString(project.Icon),
		project.Status,
		project.IsFavorite,
		string(aliasesJSON),
		nullString(project.Notes),
		project.UpdatedAt,
		project.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("project not found: %d", project.ID)
	}

	return nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM projects WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("project not found: %d", id)
	}

	return nil
}

func (r *ProjectRepository) Archive(ctx context.Context, id int64) error {
	query := `UPDATE projects SET status = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, domain.ProjectStatusArchived, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("project not found: %d", id)
	}

	return nil
}

func (r *ProjectRepository) Unarchive(ctx context.Context, id int64) error {
	query := `UPDATE projects SET status = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, domain.ProjectStatusActive, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to unarchive project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("project not found: %d", id)
	}

	return nil
}

func (r *ProjectRepository) SetFavorite(ctx context.Context, id int64, isFavorite bool) error {
	query := `UPDATE projects SET is_favorite = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, isFavorite, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set favorite: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("project not found: %d", id)
	}

	return nil
}

func (r *ProjectRepository) GetFavorites(ctx context.Context) ([]*domain.Project, error) {
	filter := repository.ProjectFilter{
		IsFavorite: boolPtr(true),
	}

	return r.List(ctx, filter)
}

func (r *ProjectRepository) GetTaskCount(ctx context.Context, projectID int64) (int, error) {
	query := `SELECT COUNT(*) FROM tasks WHERE project_id = ?`

	var count int
	if err := r.db.GetContext(ctx, &count, query, projectID); err != nil {
		return 0, fmt.Errorf("failed to get task count: %w", err)
	}

	return count, nil
}

func (r *ProjectRepository) GetTaskCountByStatus(ctx context.Context, projectID int64) (map[domain.Status]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE project_id = ?
		GROUP BY status
	`

	type statusCount struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}

	var results []statusCount
	if err := r.db.SelectContext(ctx, &results, query, projectID); err != nil {
		return nil, fmt.Errorf("failed to get task count by status: %w", err)
	}

	counts := make(map[domain.Status]int)
	for _, r := range results {
		counts[domain.Status(r.Status)] = r.Count
	}

	return counts, nil
}

func (r *ProjectRepository) ValidateHierarchy(ctx context.Context, projectID int64, parentID int64) error {
	if projectID == parentID {
		return fmt.Errorf("project cannot be its own parent")
	}

	if projectID == 0 {
		return nil
	}

	query := `
		WITH RECURSIVE descendants AS (
			SELECT id
			FROM projects
			WHERE parent_id = ?

			UNION ALL

			SELECT p.id
			FROM projects p
			INNER JOIN descendants d ON p.parent_id = d.id
		)
		SELECT COUNT(*) FROM descendants WHERE id = ?
	`

	var count int
	if err := r.db.GetContext(ctx, &count, query, projectID, parentID); err != nil {
		return fmt.Errorf("failed to validate hierarchy: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot set parent: would create a cycle in project hierarchy")
	}

	return nil
}

func (r *ProjectRepository) Search(ctx context.Context, query string, limit int) ([]*domain.Project, error) {
	filter := repository.ProjectFilter{
		SearchQuery: query,
		Limit:       limit,
	}

	return r.List(ctx, filter)
}

func (r *ProjectRepository) GetByAlias(ctx context.Context, alias string) (*domain.Project, error) {
	query := `
		SELECT projects.id, projects.name, projects.description, projects.parent_id, projects.color, projects.icon,
		       projects.status, projects.is_favorite, projects.aliases, projects.notes, projects.created_at, projects.updated_at
		FROM projects, json_each(projects.aliases)
		WHERE LOWER(json_each.value) = LOWER(?)
		LIMIT 1
	`

	var dbProj dbProject
	if err := r.db.GetContext(ctx, &dbProj, query, alias); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found with alias: %s", alias)
		}
		return nil, fmt.Errorf("failed to get project by alias: %w", err)
	}

	return dbProj.toProject()
}

func (r *ProjectRepository) ValidateAliasUniqueness(ctx context.Context, alias string, excludeProjectID *int64) error {
	query := `
		SELECT projects.id
		FROM projects, json_each(projects.aliases)
		WHERE LOWER(json_each.value) = LOWER(?)
	`

	args := []interface{}{alias}

	if excludeProjectID != nil {
		query += " AND projects.id != ?"
		args = append(args, *excludeProjectID)
	}

	var existingID int64
	err := r.db.GetContext(ctx, &existingID, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check alias uniqueness: %w", err)
	}

	if err == nil {
		return fmt.Errorf("alias '%s' is already in use by another project", alias)
	}

	return nil
}

func (r *ProjectRepository) buildWhereClause(filter repository.ProjectFilter, isCount bool) (string, []interface{}) {
	var query string
	if isCount {
		query = "SELECT COUNT(*) FROM projects WHERE 1=1"
	} else {
		query = "SELECT id, name, description, parent_id, color, icon, status, is_favorite, aliases, notes, created_at, updated_at FROM projects WHERE 1=1"
	}

	args := make([]interface{}, 0)

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if filter.ParentID != nil {
		if *filter.ParentID == 0 {
			query += " AND parent_id IS NULL"
		} else {
			query += " AND parent_id = ?"
			args = append(args, *filter.ParentID)
		}
	}

	if filter.IsFavorite != nil {
		query += " AND is_favorite = ?"
		args = append(args, *filter.IsFavorite)
	}

	if filter.ExcludeArchived {
		query += " AND status != ?"
		args = append(args, domain.ProjectStatusArchived)
	}

	if filter.SearchQuery != "" {
		searchPattern := "%" + filter.SearchQuery + "%"
		query += " AND (name LIKE ? COLLATE NOCASE OR COALESCE(description, '') LIKE ? COLLATE NOCASE)"
		args = append(args, searchPattern, searchPattern)
	}

	return query, args
}

func (r *ProjectRepository) buildOrderClause(filter repository.ProjectFilter) string {
	sortBy := filter.SortBy
	sortOrder := filter.SortOrder

	if sortBy == "" {
		sortBy = "name"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "asc"
	}

	validColumns := map[string]string{
		"name":       "name",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"task_count": "task_count",
	}

	column, ok := validColumns[sortBy]
	if !ok {
		column = "name"
	}

	return fmt.Sprintf(" ORDER BY %s %s", column, strings.ToUpper(sortOrder))
}


func boolPtr(b bool) *bool {
	return &b
}
