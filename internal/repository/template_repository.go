package repository

import (
	"context"

	"task-management/internal/domain"
)

type TemplateRepository interface {
	Create(ctx context.Context, template *domain.ProjectTemplate) error
	GetByID(ctx context.Context, id int64) (*domain.ProjectTemplate, error)
	GetByName(ctx context.Context, name string) (*domain.ProjectTemplate, error)
	Update(ctx context.Context, template *domain.ProjectTemplate) error
	Delete(ctx context.Context, id int64) error

	List(ctx context.Context, filter TemplateFilter) ([]*domain.ProjectTemplate, error)
	Count(ctx context.Context, filter TemplateFilter) (int64, error)
	Search(ctx context.Context, query string, limit int) ([]*domain.ProjectTemplate, error)
}

type TemplateFilter struct {
	SearchQuery string
	SortBy      string
	SortOrder   string
	Limit       int    
	Offset      int
}
