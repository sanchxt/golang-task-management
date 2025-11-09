package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type StatisticsRepository struct {
	db *DB
}

func NewStatisticsRepository(db *DB) *StatisticsRepository {
	return &StatisticsRepository{db: db}
}

func (r *StatisticsRepository) GetProjectStatistics(ctx context.Context, projectID int64, includeDescendants bool) (*domain.ProjectStats, error) {
	var projectName, projectPath string
	err := r.db.QueryRowContext(ctx, `
		SELECT name FROM projects WHERE id = ?
	`, projectID).Scan(&projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	stats := domain.NewProjectStats(projectID, projectName)
	stats.ProjectPath = projectPath // TODO: build full path
	stats.IncludeDescendants = includeDescendants

	projectIDs := []int64{projectID}
	if includeDescendants {
		descendants, err := r.getDescendantIDs(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get descendants: %w", err)
		}
		projectIDs = append(projectIDs, descendants...)
		stats.DescendantCount = len(descendants)
	}

	statusCounts, err := r.getTaskCountsByStatus(ctx, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	stats.PendingTasks = statusCounts["pending"]
	stats.InProgressTasks = statusCounts["in_progress"]
	stats.CompletedTasks = statusCounts["completed"]
	stats.CancelledTasks = statusCounts["cancelled"]
	stats.TotalTasks = stats.PendingTasks + stats.InProgressTasks + stats.CompletedTasks + stats.CancelledTasks

	priorityCounts, err := r.getTaskCountsByPriority(ctx, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get priority counts: %w", err)
	}

	stats.LowPriorityTasks = priorityCounts["low"]
	stats.MediumPriorityTasks = priorityCounts["medium"]
	stats.HighPriorityTasks = priorityCounts["high"]
	stats.UrgentPriorityTasks = priorityCounts["urgent"]

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	stats.OverdueTasks, err = r.getOverdueTaskCount(ctx, projectIDs, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue count: %w", err)
	}

	stats.DueSoonTasks, err = r.getDueSoonTaskCount(ctx, projectIDs, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get due soon count: %w", err)
	}

	stats.RecentTasks, err = r.getRecentTaskCount(ctx, projectIDs, sevenDaysAgo)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent count: %w", err)
	}

	stats.RecentlyUpdated, err = r.getRecentlyUpdatedCount(ctx, projectIDs, sevenDaysAgo)
	if err != nil {
		return nil, fmt.Errorf("failed to get recently updated count: %w", err)
	}

	stats.CompletionRate = stats.GetCompletionRate()

	stats.CalculatedAt = time.Now()

	return stats, nil
}

func (r *StatisticsRepository) GetGlobalStatistics(ctx context.Context) (*domain.GlobalStats, error) {
	stats := domain.NewGlobalStats()

	var activeCount, archivedCount, completedCount, favoriteCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) as active,
			SUM(CASE WHEN status = 'archived' THEN 1 ELSE 0 END) as archived,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN is_favorite = 1 THEN 1 ELSE 0 END) as favorite
		FROM projects
	`).Scan(&stats.TotalProjects, &activeCount, &archivedCount, &completedCount, &favoriteCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get project counts: %w", err)
	}

	stats.ActiveProjects = activeCount
	stats.ArchivedProjects = archivedCount
	stats.CompletedProjects = completedCount
	stats.FavoriteProjects = favoriteCount

	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END) as in_progress,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled
		FROM tasks
	`).Scan(&stats.TotalTasks, &stats.PendingTasks, &stats.InProgressTasks, &stats.CompletedTasks, &stats.CancelledTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get task counts: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT
			SUM(CASE WHEN priority = 'low' THEN 1 ELSE 0 END) as low,
			SUM(CASE WHEN priority = 'medium' THEN 1 ELSE 0 END) as medium,
			SUM(CASE WHEN priority = 'high' THEN 1 ELSE 0 END) as high,
			SUM(CASE WHEN priority = 'urgent' THEN 1 ELSE 0 END) as urgent
		FROM tasks
	`).Scan(&stats.LowPriorityTasks, &stats.MediumPriorityTasks, &stats.HighPriorityTasks, &stats.UrgentPriorityTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get priority counts: %w", err)
	}

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE due_date IS NOT NULL
		AND due_date < ?
		AND status NOT IN ('completed', 'cancelled')
	`, now).Scan(&stats.OverdueTasks)
	if err != nil {
		stats.OverdueTasks = 0
	}

	sevenDaysFromNow := now.AddDate(0, 0, 7)
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE due_date IS NOT NULL
		AND due_date BETWEEN ? AND ?
		AND status NOT IN ('completed', 'cancelled')
	`, now, sevenDaysFromNow).Scan(&stats.DueSoonTasks)
	if err != nil {
		stats.DueSoonTasks = 0
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tasks
		WHERE created_at >= ?
	`, sevenDaysAgo).Scan(&stats.RecentTasks)
	if err != nil {
		stats.RecentTasks = 0
	}

	stats.OverallCompletionRate = stats.GetCompletionRate()

	topProjects, err := r.GetTopProjectsByTaskCount(ctx, 5)
	if err == nil {
		stats.TopProjectsByTaskCount = topProjects
	}

	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN is_favorite = 1 THEN 1 ELSE 0 END) as favorite
		FROM saved_views
	`).Scan(&stats.TotalViews, &stats.FavoriteViews)
	if err != nil {
		stats.TotalViews = 0
		stats.FavoriteViews = 0
	}

	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM project_templates`).Scan(&stats.TotalTemplates)
	if err != nil {
		stats.TotalTemplates = 0
	}

	stats.CalculatedAt = time.Now()

	return stats, nil
}

func (r *StatisticsRepository) GetTopProjectsByTaskCount(ctx context.Context, limit int) ([]domain.ProjectTaskCount, error) {
	query := `
		SELECT
			p.id,
			p.name,
			p.icon,
			COUNT(t.id) as task_count
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.status != 'archived'
		GROUP BY p.id, p.name, p.icon
		ORDER BY task_count DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top projects: %w", err)
	}
	defer rows.Close()

	var results []domain.ProjectTaskCount
	for rows.Next() {
		var pc domain.ProjectTaskCount
		var icon sql.NullString
		err := rows.Scan(&pc.ProjectID, &pc.ProjectName, &icon, &pc.TaskCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		if icon.Valid {
			pc.Icon = icon.String
		}
		results = append(results, pc)
	}

	return results, rows.Err()
}


func (r *StatisticsRepository) getDescendantIDs(ctx context.Context, projectID int64) ([]int64, error) {
	query := `
		WITH RECURSIVE descendants AS (
			SELECT id FROM projects WHERE parent_id = ?
			UNION ALL
			SELECT p.id FROM projects p
			INNER JOIN descendants d ON p.parent_id = d.id
		)
		SELECT id FROM descendants
	`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (r *StatisticsRepository) getTaskCountsByStatus(ctx context.Context, projectIDs []int64) (map[string]int, error) {
	if len(projectIDs) == 0 {
		return map[string]int{}, nil
	}

	query, args := buildINQuery(`
		SELECT status, COUNT(*) as count
		FROM tasks
		WHERE project_id IN (?)
		GROUP BY status
	`, projectIDs)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

func (r *StatisticsRepository) getTaskCountsByPriority(ctx context.Context, projectIDs []int64) (map[string]int, error) {
	if len(projectIDs) == 0 {
		return map[string]int{}, nil
	}

	query, args := buildINQuery(`
		SELECT priority, COUNT(*) as count
		FROM tasks
		WHERE project_id IN (?)
		GROUP BY priority
	`, projectIDs)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var priority string
		var count int
		if err := rows.Scan(&priority, &count); err != nil {
			return nil, err
		}
		counts[priority] = count
	}

	return counts, rows.Err()
}

func (r *StatisticsRepository) getOverdueTaskCount(ctx context.Context, projectIDs []int64, now time.Time) (int, error) {
	if len(projectIDs) == 0 {
		return 0, nil
	}

	query, args := buildINQuery(`
		SELECT COUNT(*) FROM tasks
		WHERE project_id IN (?)
		AND due_date IS NOT NULL
		AND due_date < ?
		AND status NOT IN ('completed', 'cancelled')
	`, projectIDs)

	args = append(args, now)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticsRepository) getDueSoonTaskCount(ctx context.Context, projectIDs []int64, now time.Time) (int, error) {
	if len(projectIDs) == 0 {
		return 0, nil
	}

	sevenDaysFromNow := now.AddDate(0, 0, 7)

	query, args := buildINQuery(`
		SELECT COUNT(*) FROM tasks
		WHERE project_id IN (?)
		AND due_date IS NOT NULL
		AND due_date BETWEEN ? AND ?
		AND status NOT IN ('completed', 'cancelled')
	`, projectIDs)

	args = append(args, now, sevenDaysFromNow)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticsRepository) getRecentTaskCount(ctx context.Context, projectIDs []int64, since time.Time) (int, error) {
	if len(projectIDs) == 0 {
		return 0, nil
	}

	query, args := buildINQuery(`
		SELECT COUNT(*) FROM tasks
		WHERE project_id IN (?)
		AND created_at >= ?
	`, projectIDs)

	args = append(args, since)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *StatisticsRepository) getRecentlyUpdatedCount(ctx context.Context, projectIDs []int64, since time.Time) (int, error) {
	if len(projectIDs) == 0 {
		return 0, nil
	}

	query, args := buildINQuery(`
		SELECT COUNT(*) FROM tasks
		WHERE project_id IN (?)
		AND updated_at >= ?
	`, projectIDs)

	args = append(args, since)

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

var _ repository.StatisticsRepository = (*StatisticsRepository)(nil)
