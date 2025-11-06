package repository

import (
	"context"
	"task-management/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	List(ctx context.Context, filter TaskFilter) ([]*domain.Task, error)
	Count(ctx context.Context, filter TaskFilter) (int64, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id int64) error
}

// filtering options for tasks lists
type TaskFilter struct {
	// basic filters
	Status   domain.Status
	Priority domain.Priority
	Project  string
	Tags     []string

	// pagination
	Limit  int // max number of results (0 = no limit)
	Offset int // number of results to skip

	// search
	SearchQuery string // search query text
	SearchMode  string // "text" or "regex"

	// sorting
	SortBy    string // field to sort by: "created_at", "updated_at", "priority", "due_date", "title"
	SortOrder string // "asc" or "desc"

	// date range filtering
	DueDateFrom *string // ISO format date string
	DueDateTo   *string // ISO format date string
}
