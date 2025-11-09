package repository

import (
	"context"

	"task-management/internal/domain"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error

	GetByID(ctx context.Context, id int64) (*domain.Project, error)

	GetByName(ctx context.Context, name string) (*domain.Project, error)

	GetByAlias(ctx context.Context, alias string) (*domain.Project, error)

	ValidateAliasUniqueness(ctx context.Context, alias string, excludeProjectID *int64) error

	List(ctx context.Context, filter ProjectFilter) ([]*domain.Project, error)

	ListWithHierarchy(ctx context.Context, filter ProjectFilter) ([]*domain.Project, error)

	GetChildren(ctx context.Context, parentID int64) ([]*domain.Project, error)

	GetDescendants(ctx context.Context, parentID int64) ([]*domain.Project, error)

	GetPath(ctx context.Context, projectID int64) ([]*domain.Project, error)

	GetRoots(ctx context.Context) ([]*domain.Project, error)

	Count(ctx context.Context, filter ProjectFilter) (int64, error)

	Update(ctx context.Context, project *domain.Project) error

	Delete(ctx context.Context, id int64) error

	Archive(ctx context.Context, id int64) error

	Unarchive(ctx context.Context, id int64) error

	SetFavorite(ctx context.Context, id int64, isFavorite bool) error

	GetFavorites(ctx context.Context) ([]*domain.Project, error)

	GetTaskCount(ctx context.Context, projectID int64) (int, error)

	GetTaskCountByStatus(ctx context.Context, projectID int64) (map[domain.Status]int, error)

	ValidateHierarchy(ctx context.Context, projectID int64, parentID int64) error

	Search(ctx context.Context, query string, limit int) ([]*domain.Project, error)
}

type ProjectFilter struct {
	Status domain.ProjectStatus

	ParentID *int64

	IsFavorite *bool

	ExcludeArchived bool

	IncludeTaskCount bool

	SearchQuery string

	SortBy string

	SortOrder string

	Limit  int
	Offset int
}
