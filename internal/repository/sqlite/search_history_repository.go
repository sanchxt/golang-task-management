package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type searchHistoryRepository struct {
	db *DB
}

func NewSearchHistoryRepository(db *DB) repository.SearchHistoryRepository {
	return &searchHistoryRepository{db: db}
}

func (r *searchHistoryRepository) RecordSearch(ctx context.Context, entry *domain.SearchHistory) error {
	if err := entry.Validate(); err != nil {
		return fmt.Errorf("invalid search history entry: %w", err)
	}

	var existingID int64
	checkQuery := `
		SELECT id FROM search_history
		WHERE query_text = ? AND search_mode = ? AND query_type = ?
		LIMIT 1
	`

	err := r.db.QueryRowContext(ctx, checkQuery, entry.QueryText, entry.SearchMode, entry.QueryType).Scan(&existingID)

	if err == nil {
		updateQuery := `
			UPDATE search_history
			SET updated_at = CURRENT_TIMESTAMP,
			    result_count = ?,
			    project_filter = ?,
			    fuzzy_threshold = ?
			WHERE id = ?
		`
		_, err := r.db.ExecContext(ctx, updateQuery, entry.ResultCount, nullString(entry.ProjectFilter), intPtrToNullInt64(entry.FuzzyThreshold), existingID)
		if err != nil {
			return fmt.Errorf("failed to update search history: %w", err)
		}
		entry.ID = existingID
		return nil
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing search history: %w", err)
	}

	insertQuery := `
		INSERT INTO search_history (
			query_text, search_mode, fuzzy_threshold, query_type,
			project_filter, result_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`

	result, err := r.db.ExecContext(ctx, insertQuery,
		entry.QueryText,
		entry.SearchMode,
		intPtrToNullInt64(entry.FuzzyThreshold),
		entry.QueryType,
		nullString(entry.ProjectFilter),
		entry.ResultCount,
	)
	if err != nil {
		return fmt.Errorf("failed to create search history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	entry.ID = id

	return nil
}

func (r *searchHistoryRepository) List(ctx context.Context, limit int) ([]*domain.SearchHistory, error) {
	query := `
		SELECT id, query_text, search_mode, fuzzy_threshold, query_type,
		       project_filter, result_count, created_at, updated_at
		FROM search_history
		ORDER BY updated_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query search history: %w", err)
	}
	defer rows.Close()

	var entries []*domain.SearchHistory
	for rows.Next() {
		var entry domain.SearchHistory
		var fuzzyThreshold sql.NullInt64
		var projectFilter sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.QueryText,
			&entry.SearchMode,
			&fuzzyThreshold,
			&entry.QueryType,
			&projectFilter,
			&entry.ResultCount,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search history row: %w", err)
		}

		if fuzzyThreshold.Valid {
			threshold := int(fuzzyThreshold.Int64)
			entry.FuzzyThreshold = &threshold
		}
		if projectFilter.Valid {
			entry.ProjectFilter = projectFilter.String
		}

		entries = append(entries, &entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search history rows: %w", err)
	}

	return entries, nil
}

func (r *searchHistoryRepository) GetByID(ctx context.Context, id int64) (*domain.SearchHistory, error) {
	query := `
		SELECT id, query_text, search_mode, fuzzy_threshold, query_type,
		       project_filter, result_count, created_at, updated_at
		FROM search_history
		WHERE id = ?
	`

	var entry domain.SearchHistory
	var fuzzyThreshold sql.NullInt64
	var projectFilter sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID,
		&entry.QueryText,
		&entry.SearchMode,
		&fuzzyThreshold,
		&entry.QueryType,
		&projectFilter,
		&entry.ResultCount,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("search history entry with id %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query search history: %w", err)
	}

	if fuzzyThreshold.Valid {
		threshold := int(fuzzyThreshold.Int64)
		entry.FuzzyThreshold = &threshold
	}
	if projectFilter.Valid {
		entry.ProjectFilter = projectFilter.String
	}

	return &entry, nil
}

func (r *searchHistoryRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM search_history WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete search history: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("search history entry with id %d not found", id)
	}

	return nil
}

func (r *searchHistoryRepository) Clear(ctx context.Context) error {
	query := `DELETE FROM search_history`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear search history: %w", err)
	}

	return nil
}

func (r *searchHistoryRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM search_history`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count search history: %w", err)
	}

	return count, nil
}

func intPtrToNullInt64(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}
