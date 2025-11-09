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

type ViewRepository struct {
	db *DB
}

func NewViewRepository(db *DB) *ViewRepository {
	return &ViewRepository{db: db}
}

type dbView struct {
	ID           int64          `db:"id"`
	Name         string         `db:"name"`
	Description  sql.NullString `db:"description"`
	FilterConfig string         `db:"filter_config"`
	IsFavorite   bool           `db:"is_favorite"`
	HotKey       sql.NullInt64  `db:"hot_key"`
	LastAccessed sql.NullTime   `db:"last_accessed"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func (dv *dbView) toView() (*domain.SavedView, error) {
	view := &domain.SavedView{
		ID:         dv.ID,
		Name:       dv.Name,
		IsFavorite: dv.IsFavorite,
		CreatedAt:  dv.CreatedAt,
		UpdatedAt:  dv.UpdatedAt,
	}

	if dv.Description.Valid {
		view.Description = dv.Description.String
	}

	if dv.HotKey.Valid {
		hotKey := int(dv.HotKey.Int64)
		view.HotKey = &hotKey
	}

	if dv.LastAccessed.Valid {
		view.LastAccessed = &dv.LastAccessed.Time
	}

	if err := json.Unmarshal([]byte(dv.FilterConfig), &view.FilterConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filter config: %w", err)
	}

	return view, nil
}

func (r *ViewRepository) Create(ctx context.Context, view *domain.SavedView) error {
	if err := view.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	filterJSON, err := json.Marshal(view.FilterConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal filter config: %w", err)
	}

	if view.CreatedAt.IsZero() {
		view.CreatedAt = time.Now()
	}
	if view.UpdatedAt.IsZero() {
		view.UpdatedAt = time.Now()
	}

	if view.HotKey != nil {
		existing, err := r.GetByHotKey(ctx, *view.HotKey)
		if err == nil && existing != nil {
			return fmt.Errorf("hot key %d is already assigned to view '%s'", *view.HotKey, existing.Name)
		}
	}

	query := `
		INSERT INTO saved_views (name, description, filter_config, is_favorite, hot_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		view.Name,
		nullString(view.Description),
		string(filterJSON),
		view.IsFavorite,
		nullInt64Ptr(view.HotKey),
		view.CreatedAt,
		view.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("view with name %q already exists", view.Name)
		}
		return fmt.Errorf("failed to insert view: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	view.ID = id
	return nil
}

func (r *ViewRepository) GetByID(ctx context.Context, id int64) (*domain.SavedView, error) {
	query := `
		SELECT id, name, description, filter_config, is_favorite, hot_key, last_accessed, created_at, updated_at
		FROM saved_views
		WHERE id = ?
	`

	var dv dbView
	err := r.db.GetContext(ctx, &dv, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("view with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get view: %w", err)
	}

	return dv.toView()
}

func (r *ViewRepository) GetByName(ctx context.Context, name string) (*domain.SavedView, error) {
	query := `
		SELECT id, name, description, filter_config, is_favorite, hot_key, last_accessed, created_at, updated_at
		FROM saved_views
		WHERE name = ?
	`

	var dv dbView
	err := r.db.GetContext(ctx, &dv, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("view with name %q not found", name)
		}
		return nil, fmt.Errorf("failed to get view: %w", err)
	}

	return dv.toView()
}

func (r *ViewRepository) Update(ctx context.Context, view *domain.SavedView) error {
	if err := view.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	filterJSON, err := json.Marshal(view.FilterConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal filter config: %w", err)
	}

	if view.HotKey != nil {
		existing, err := r.GetByHotKey(ctx, *view.HotKey)
		if err == nil && existing != nil && existing.ID != view.ID {
			return fmt.Errorf("hot key %d is already assigned to view '%s'", *view.HotKey, existing.Name)
		}
	}

	query := `
		UPDATE saved_views
		SET name = ?, description = ?, filter_config = ?, is_favorite = ?, hot_key = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		view.Name,
		nullString(view.Description),
		string(filterJSON),
		view.IsFavorite,
		nullInt64Ptr(view.HotKey),
		time.Now(),
		view.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("view with name %q already exists", view.Name)
		}
		return fmt.Errorf("failed to update view: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("view with ID %d not found", view.ID)
	}

	view.UpdatedAt = time.Now()
	return nil
}

func (r *ViewRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM saved_views WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete view: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("view with ID %d not found", id)
	}

	return nil
}

func (r *ViewRepository) List(ctx context.Context, filter repository.ViewFilter) ([]*domain.SavedView, error) {
	query := `SELECT id, name, description, filter_config, is_favorite, hot_key, last_accessed, created_at, updated_at FROM saved_views`

	var whereClauses []string
	var args []interface{}

	if filter.IsFavorite != nil {
		whereClauses = append(whereClauses, "is_favorite = ?")
		args = append(args, *filter.IsFavorite)
	}

	if filter.HasHotKey {
		whereClauses = append(whereClauses, "hot_key IS NOT NULL")
	}

	if filter.SearchQuery != "" {
		whereClauses = append(whereClauses, "(name LIKE ? OR description LIKE ?)")
		searchPattern := "%" + filter.SearchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		switch filter.SortBy {
		case "name", "created_at", "updated_at", "last_accessed":
			orderBy = filter.SortBy
			if filter.SortOrder != "" && (filter.SortOrder == "asc" || filter.SortOrder == "desc") {
				orderBy += " " + filter.SortOrder
			} else {
				orderBy += " DESC"
			}
		}
	}

	query += " ORDER BY " + orderBy

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	var dbViews []dbView
	if err := r.db.SelectContext(ctx, &dbViews, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list views: %w", err)
	}

	views := make([]*domain.SavedView, 0, len(dbViews))
	for _, dv := range dbViews {
		view, err := dv.toView()
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

func (r *ViewRepository) Count(ctx context.Context, filter repository.ViewFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM saved_views`

	var whereClauses []string
	var args []interface{}

	if filter.IsFavorite != nil {
		whereClauses = append(whereClauses, "is_favorite = ?")
		args = append(args, *filter.IsFavorite)
	}

	if filter.HasHotKey {
		whereClauses = append(whereClauses, "hot_key IS NOT NULL")
	}

	if filter.SearchQuery != "" {
		whereClauses = append(whereClauses, "(name LIKE ? OR description LIKE ?)")
		searchPattern := "%" + filter.SearchQuery + "%"
		args = append(args, searchPattern, searchPattern)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var count int64
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count views: %w", err)
	}

	return count, nil
}

func (r *ViewRepository) Search(ctx context.Context, query string, limit int) ([]*domain.SavedView, error) {
	if limit <= 0 {
		limit = 10
	}

	searchPattern := "%" + query + "%"
	sqlQuery := `
		SELECT id, name, description, filter_config, is_favorite, hot_key, last_accessed, created_at, updated_at
		FROM saved_views
		WHERE name LIKE ? OR description LIKE ?
		ORDER BY name ASC
		LIMIT ?
	`

	var dbViews []dbView
	if err := r.db.SelectContext(ctx, &dbViews, sqlQuery, searchPattern, searchPattern, limit); err != nil {
		return nil, fmt.Errorf("failed to search views: %w", err)
	}

	views := make([]*domain.SavedView, 0, len(dbViews))
	for _, dv := range dbViews {
		view, err := dv.toView()
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

func (r *ViewRepository) GetByHotKey(ctx context.Context, hotKey int) (*domain.SavedView, error) {
	query := `
		SELECT id, name, description, filter_config, is_favorite, hot_key, last_accessed, created_at, updated_at
		FROM saved_views
		WHERE hot_key = ?
	`

	var dv dbView
	err := r.db.GetContext(ctx, &dv, query, hotKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no view assigned to hot key %d", hotKey)
		}
		return nil, fmt.Errorf("failed to get view by hot key: %w", err)
	}

	return dv.toView()
}

func (r *ViewRepository) SetHotKey(ctx context.Context, viewID int64, hotKey *int) error {
	if hotKey != nil {
		existing, err := r.GetByHotKey(ctx, *hotKey)
		if err == nil && existing != nil && existing.ID != viewID {
			return fmt.Errorf("hot key %d is already assigned to view '%s'", *hotKey, existing.Name)
		}
	}

	query := `UPDATE saved_views SET hot_key = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, nullInt64Ptr(hotKey), viewID)
	if err != nil {
		return fmt.Errorf("failed to set hot key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("view with ID %d not found", viewID)
	}

	return nil
}

func (r *ViewRepository) GetFavorites(ctx context.Context) ([]*domain.SavedView, error) {
	filter := repository.ViewFilter{
		IsFavorite: &[]bool{true}[0],
		SortBy:     "name",
		SortOrder:  "asc",
	}
	return r.List(ctx, filter)
}

func (r *ViewRepository) SetFavorite(ctx context.Context, viewID int64, isFavorite bool) error {
	query := `UPDATE saved_views SET is_favorite = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, isFavorite, viewID)
	if err != nil {
		return fmt.Errorf("failed to set favorite: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("view with ID %d not found", viewID)
	}

	return nil
}

func (r *ViewRepository) GetRecentViews(ctx context.Context, limit int) ([]*domain.SavedView, error) {
	if limit <= 0 {
		limit = 5
	}

	query := `
		SELECT id, name, description, filter_config, is_favorite, hot_key, last_accessed, created_at, updated_at
		FROM saved_views
		WHERE last_accessed IS NOT NULL
		ORDER BY last_accessed DESC
		LIMIT ?
	`

	var dbViews []dbView
	if err := r.db.SelectContext(ctx, &dbViews, query, limit); err != nil {
		return nil, fmt.Errorf("failed to get recent views: %w", err)
	}

	views := make([]*domain.SavedView, 0, len(dbViews))
	for _, dv := range dbViews {
		view, err := dv.toView()
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

func (r *ViewRepository) RecordViewAccess(ctx context.Context, viewID int64) error {
	query := `UPDATE saved_views SET last_accessed = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, time.Now(), viewID)
	if err != nil {
		return fmt.Errorf("failed to record view access: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("view with ID %d not found", viewID)
	}

	return nil
}

func nullInt64Ptr(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}
