package repository

import (
	"context"
	"task-management/internal/domain"
)

type SearchHistoryRepository interface {
	RecordSearch(ctx context.Context, entry *domain.SearchHistory) error

	List(ctx context.Context, limit int) ([]*domain.SearchHistory, error)

	GetByID(ctx context.Context, id int64) (*domain.SearchHistory, error)

	Delete(ctx context.Context, id int64) error

	Clear(ctx context.Context) error

	Count(ctx context.Context) (int64, error)
}
