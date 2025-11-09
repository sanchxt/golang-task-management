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

	// Bulk operations
	BulkUpdate(ctx context.Context, filter TaskFilter, updates TaskUpdate) (int64, error)
	BulkMove(ctx context.Context, filter TaskFilter, projectID *int64) (int64, error)
	BulkAddTags(ctx context.Context, filter TaskFilter, tags []string) (int64, error)
	BulkRemoveTags(ctx context.Context, filter TaskFilter, tags []string) (int64, error)
	BulkDelete(ctx context.Context, filter TaskFilter) (int64, error)
}

type TaskFilter struct {
	// basic filters
	Status    domain.Status
	Priority  domain.Priority
	ProjectID *int64
	Tags      []string
	ExcludeTags []string

	// pagination
	Limit  int
	Offset int

	// search
	SearchQuery    string
	SearchMode     string
	FuzzyThreshold int

	// sorting
	SortBy    string
	SortOrder string

	// date range
	DueDateFrom *string
	DueDateTo   *string
	CreatedFrom *string
	CreatedTo   *string
	UpdatedFrom *string
	UpdatedTo   *string
}

type TaskUpdate struct {
	Status      *domain.Status
	Priority    *domain.Priority
	ProjectID   **int64 // double pointer to distinguish between "set to NULL" and "don't update"
	Description *string
	DueDate     **string // double pointer for the same rzn ^
}
