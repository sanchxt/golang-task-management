package repository

import (
	"context"
	"task-management/internal/domain"
)

type StatisticsRepository interface {
	GetProjectStatistics(ctx context.Context, projectID int64, includeDescendants bool) (*domain.ProjectStats, error)

	GetGlobalStatistics(ctx context.Context) (*domain.GlobalStats, error)

	GetTopProjectsByTaskCount(ctx context.Context, limit int) ([]domain.ProjectTaskCount, error)
}
