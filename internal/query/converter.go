package query

import (
	"context"
	"fmt"
	"strings"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type ConverterContext struct {
	ProjectRepo ProjectRepository
}

type ProjectRepository interface {
	GetByName(ctx context.Context, name string) (*domain.Project, error)
	GetByAlias(ctx context.Context, alias string) (*domain.Project, error)
	Search(ctx context.Context, query string, limit int) ([]*domain.Project, error)
}

func ConvertToTaskFilter(ctx context.Context, parsed *ParsedQuery, converterCtx *ConverterContext) (repository.TaskFilter, error) {
	filter := repository.TaskFilter{}
	var errors []error

	for _, qf := range parsed.Filters {
		if err := applyFilter(&filter, qf, ctx, converterCtx); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return filter, fmt.Errorf("conversion errors: %v", errors)
	}

	return filter, nil
}

func applyFilter(filter *repository.TaskFilter, qf QueryFilter, ctx context.Context, converterCtx *ConverterContext) error {
	switch qf.Field {
	case "status":
		return applyStatusFilter(filter, qf)
	case "priority":
		return applyPriorityFilter(filter, qf)
	case "project":
		return applyProjectFilter(filter, qf, ctx, converterCtx)
	case "tag":
		return applyTagFilter(filter, qf)
	case "due":
		return applyDueDateFilter(filter, qf)
	case "created":
		return applyCreatedDateFilter(filter, qf)
	case "updated":
		return applyUpdatedDateFilter(filter, qf)
	default:
		return fmt.Errorf("unknown filter field: %s", qf.Field)
	}
}

func applyStatusFilter(filter *repository.TaskFilter, qf QueryFilter) error {
	if qf.IsNot {
		return fmt.Errorf("negated status filters not supported yet")
	}
	if qf.Operator != ":" && qf.Operator != "=" {
		return fmt.Errorf("status only supports exact match (:, =), got: %s", qf.Operator)
	}

	status := domain.Status(strings.ToLower(qf.Value))

	switch status {
	case domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled:
		filter.Status = status
		return nil
	default:
		return fmt.Errorf("invalid status value: %s (must be pending, in_progress, completed, or cancelled)", qf.Value)
	}
}

func applyPriorityFilter(filter *repository.TaskFilter, qf QueryFilter) error {
	if qf.IsNot {
		return fmt.Errorf("negated priority filters not supported yet")
	}
	if qf.Operator != ":" && qf.Operator != "=" {
		return fmt.Errorf("priority only supports exact match (:, =), got: %s", qf.Operator)
	}

	priority := domain.Priority(strings.ToLower(qf.Value))

	switch priority {
	case domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent:
		filter.Priority = priority
		return nil
	default:
		return fmt.Errorf("invalid priority value: %s (must be low, medium, high, or urgent)", qf.Value)
	}
}

func applyProjectFilter(filter *repository.TaskFilter, qf QueryFilter, ctx context.Context, converterCtx *ConverterContext) error {
	if qf.IsNot {
		return fmt.Errorf("negated project filters not supported yet")
	}
	if qf.Operator != ":" && qf.Operator != "=" {
		return fmt.Errorf("project only supports exact match (:, =), got: %s", qf.Operator)
	}

	if converterCtx == nil || converterCtx.ProjectRepo == nil {
		return fmt.Errorf("project repository not available for project lookup")
	}

	var projectID *int64

	if qf.IsFuzzy {
		limit := 10
		projects, err := converterCtx.ProjectRepo.Search(ctx, qf.Value, limit)
		if err != nil {
			return fmt.Errorf("fuzzy project search failed: %w", err)
		}
		if len(projects) == 0 {
			return fmt.Errorf("no project found matching '%s' (fuzzy)", qf.Value)
		}
		projectID = &projects[0].ID
	} else {
		project, err := converterCtx.ProjectRepo.GetByName(ctx, qf.Value)
		if err != nil {
			project, aliasErr := converterCtx.ProjectRepo.GetByAlias(ctx, qf.Value)
			if aliasErr != nil {
				return fmt.Errorf("project not found: '%s' (tried name and alias)", qf.Value)
			}
			projectID = &project.ID
		} else {
			projectID = &project.ID
		}
	}

	filter.ProjectID = projectID
	return nil
}

func applyTagFilter(filter *repository.TaskFilter, qf QueryFilter) error {
	if qf.Operator != ":" && qf.Operator != "=" {
		return fmt.Errorf("tag only supports exact match (:, =), got: %s", qf.Operator)
	}

	tag := strings.TrimSpace(qf.Value)
	if tag == "" {
		return fmt.Errorf("tag value cannot be empty")
	}

	if qf.IsNot {
		filter.ExcludeTags = append(filter.ExcludeTags, tag)
	} else {
		filter.Tags = append(filter.Tags, tag)
	}

	return nil
}

func applyDueDateFilter(filter *repository.TaskFilter, qf QueryFilter) error {
	if qf.IsNot {
		return fmt.Errorf("negated due date filters not supported yet")
	}

	startDate, endDate, err := ParseDateRange(qf.Value, qf.Operator)
	if err != nil {
		return fmt.Errorf("invalid due date value '%s': %w", qf.Value, err)
	}

	if startDate != nil {
		dateStr := FormatDateForSQL(*startDate)
		filter.DueDateFrom = &dateStr
	}
	if endDate != nil {
		dateStr := FormatDateForSQL(*endDate)
		filter.DueDateTo = &dateStr
	}

	return nil
}

func applyCreatedDateFilter(filter *repository.TaskFilter, qf QueryFilter) error {
	if qf.IsNot {
		return fmt.Errorf("negated created date filters not supported yet")
	}

	startDate, endDate, err := ParseDateRange(qf.Value, qf.Operator)
	if err != nil {
		return fmt.Errorf("invalid created date value '%s': %w", qf.Value, err)
	}

	if startDate != nil {
		dateStr := FormatDateForSQL(*startDate)
		filter.CreatedFrom = &dateStr
	}
	if endDate != nil {
		dateStr := FormatDateForSQL(*endDate)
		filter.CreatedTo = &dateStr
	}

	return nil
}

func applyUpdatedDateFilter(filter *repository.TaskFilter, qf QueryFilter) error {
	if qf.IsNot {
		return fmt.Errorf("negated updated date filters not supported yet")
	}

	startDate, endDate, err := ParseDateRange(qf.Value, qf.Operator)
	if err != nil {
		return fmt.Errorf("invalid updated date value '%s': %w", qf.Value, err)
	}

	if startDate != nil {
		dateStr := FormatDateForSQL(*startDate)
		filter.UpdatedFrom = &dateStr
	}
	if endDate != nil {
		dateStr := FormatDateForSQL(*endDate)
		filter.UpdatedTo = &dateStr
	}

	return nil
}
