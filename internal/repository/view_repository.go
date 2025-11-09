package repository

import (
	"context"

	"task-management/internal/domain"
)

type ViewRepository interface {
	Create(ctx context.Context, view *domain.SavedView) error
	GetByID(ctx context.Context, id int64) (*domain.SavedView, error)
	GetByName(ctx context.Context, name string) (*domain.SavedView, error)
	Update(ctx context.Context, view *domain.SavedView) error
	Delete(ctx context.Context, id int64) error

	List(ctx context.Context, filter ViewFilter) ([]*domain.SavedView, error)
	Count(ctx context.Context, filter ViewFilter) (int64, error)
	Search(ctx context.Context, query string, limit int) ([]*domain.SavedView, error)

	GetByHotKey(ctx context.Context, hotKey int) (*domain.SavedView, error)
	SetHotKey(ctx context.Context, viewID int64, hotKey *int) error  // nil to clear
	GetFavorites(ctx context.Context) ([]*domain.SavedView, error)
	SetFavorite(ctx context.Context, viewID int64, isFavorite bool) error

	GetRecentViews(ctx context.Context, limit int) ([]*domain.SavedView, error)
	RecordViewAccess(ctx context.Context, viewID int64) error
}

type ViewFilter struct {
	IsFavorite  *bool
	HasHotKey   bool
	SearchQuery string
	SortBy      string
	SortOrder   string
	Limit       int
	Offset      int
}
