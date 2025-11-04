package repository

import (
	"context"
	"task-management/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	List(ctx context.Context, filter TaskFilter) ([]*domain.Task, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id int64) error
}

// filtering options for tasks lists
type TaskFilter struct {
	Status   domain.Status
	Priority domain.Priority
	Project  string
	Tags     []string
}
