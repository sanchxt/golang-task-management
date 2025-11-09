package query

import (
	"context"
	"errors"
	"strings"
	"testing"

	"task-management/internal/domain"
	"task-management/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProjectRepo struct {
	projects       map[string]*domain.Project
	fuzzyThreshold int
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: map[string]*domain.Project{
			"backend": {ID: 1, Name: "backend"},
			"frontend": {ID: 2, Name: "frontend"},
			"mobile-app": {ID: 3, Name: "mobile-app"},
		},
	}
}

func (m *mockProjectRepo) GetByName(ctx context.Context, name string) (*domain.Project, error) {
	if project, ok := m.projects[name]; ok {
		return project, nil
	}
	return nil, errors.New("project not found")
}

func (m *mockProjectRepo) GetByAlias(ctx context.Context, alias string) (*domain.Project, error) {
	for _, project := range m.projects {
		if project.Name == alias || (project.Aliases != nil && project.HasAlias(alias)) {
			return project, nil
		}
	}
	return nil, errors.New("project not found by alias")
}

func (m *mockProjectRepo) Search(ctx context.Context, query string, limit int) ([]*domain.Project, error) {
	var results []*domain.Project
	for _, project := range m.projects {
		if strings.Contains(project.Name, query) {
			results = append(results, project)
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func TestConvertToTaskFilter_Status(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkFilter func(*testing.T, repository.TaskFilter)
	}{
		{
			name:        "valid status",
			query:       "status:pending",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, domain.StatusPending, filter.Status)
			},
		},
		{
			name:        "all valid statuses",
			query:       "status:completed",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, domain.StatusCompleted, filter.Status)
			},
		},
		{
			name:        "invalid status",
			query:       "status:invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseQuery(tt.query)
			require.NoError(t, err)

			ctx := context.Background()
			converterCtx := &ConverterContext{
				ProjectRepo: newMockProjectRepo(),
			}

			filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFilter != nil {
					tt.checkFilter(t, filter)
				}
			}
		})
	}
}

func TestConvertToTaskFilter_Priority(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkFilter func(*testing.T, repository.TaskFilter)
	}{
		{
			name:        "valid priority",
			query:       "priority:high",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, domain.PriorityHigh, filter.Priority)
			},
		},
		{
			name:        "urgent priority",
			query:       "priority:urgent",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, domain.PriorityUrgent, filter.Priority)
			},
		},
		{
			name:        "invalid priority",
			query:       "priority:critical",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseQuery(tt.query)
			require.NoError(t, err)

			ctx := context.Background()
			converterCtx := &ConverterContext{
				ProjectRepo: newMockProjectRepo(),
			}

			filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFilter != nil {
					tt.checkFilter(t, filter)
				}
			}
		})
	}
}

func TestConvertToTaskFilter_Project(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkFilter func(*testing.T, repository.TaskFilter)
	}{
		{
			name:        "exact project match",
			query:       "@backend",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				require.NotNil(t, filter.ProjectID)
				assert.Equal(t, int64(1), *filter.ProjectID)
			},
		},
		{
			name:        "fuzzy project match",
			query:       "@~back",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				require.NotNil(t, filter.ProjectID)
				assert.Equal(t, int64(1), *filter.ProjectID)
			},
		},
		{
			name:        "project not found",
			query:       "@nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseQuery(tt.query)
			require.NoError(t, err)

			ctx := context.Background()
			converterCtx := &ConverterContext{
				ProjectRepo: newMockProjectRepo(),
			}

			filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFilter != nil {
					tt.checkFilter(t, filter)
				}
			}
		})
	}
}

func TestConvertToTaskFilter_Tags(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkFilter func(*testing.T, repository.TaskFilter)
	}{
		{
			name:        "include single tag",
			query:       "tag:bug",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, []string{"bug"}, filter.Tags)
				assert.Empty(t, filter.ExcludeTags)
			},
		},
		{
			name:        "exclude single tag",
			query:       "-tag:wontfix",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Empty(t, filter.Tags)
				assert.Equal(t, []string{"wontfix"}, filter.ExcludeTags)
			},
		},
		{
			name:        "include and exclude tags",
			query:       "tag:bug -tag:wontfix",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, []string{"bug"}, filter.Tags)
				assert.Equal(t, []string{"wontfix"}, filter.ExcludeTags)
			},
		},
		{
			name:        "multiple include tags",
			query:       "tag:bug tag:urgent",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Equal(t, []string{"bug", "urgent"}, filter.Tags)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseQuery(tt.query)
			require.NoError(t, err)

			ctx := context.Background()
			converterCtx := &ConverterContext{
				ProjectRepo: newMockProjectRepo(),
			}

			filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFilter != nil {
					tt.checkFilter(t, filter)
				}
			}
		})
	}
}

func TestConvertToTaskFilter_DueDates(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkFilter func(*testing.T, repository.TaskFilter)
	}{
		{
			name:        "exact due date",
			query:       "due:2025-01-15",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				require.NotNil(t, filter.DueDateFrom)
				require.NotNil(t, filter.DueDateTo)
				assert.Contains(t, *filter.DueDateFrom, "2025-01-15")
				assert.Contains(t, *filter.DueDateTo, "2025-01-15")
			},
		},
		{
			name:        "due before date",
			query:       "due:<2025-12-31",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.Nil(t, filter.DueDateFrom)
				require.NotNil(t, filter.DueDateTo)
				assert.Contains(t, *filter.DueDateTo, "2025-12-31")
			},
		},
		{
			name:        "due after date",
			query:       "due:>2025-01-01",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				require.NotNil(t, filter.DueDateFrom)
				assert.Nil(t, filter.DueDateTo)
				assert.Contains(t, *filter.DueDateFrom, "2025-01-01")
			},
		},
		{
			name:        "relative due date",
			query:       "due:today",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				assert.NotNil(t, filter.DueDateFrom)
				assert.NotNil(t, filter.DueDateTo)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseQuery(tt.query)
			require.NoError(t, err)

			ctx := context.Background()
			converterCtx := &ConverterContext{
				ProjectRepo: newMockProjectRepo(),
			}

			filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFilter != nil {
					tt.checkFilter(t, filter)
				}
			}
		})
	}
}

func TestConvertToTaskFilter_CreatedDates(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		checkFilter func(*testing.T, repository.TaskFilter)
	}{
		{
			name:        "created after date",
			query:       "created:>2025-01-01",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				require.NotNil(t, filter.CreatedFrom)
				assert.Contains(t, *filter.CreatedFrom, "2025-01-01")
			},
		},
		{
			name:        "created before date",
			query:       "created:<2025-12-31",
			expectError: false,
			checkFilter: func(t *testing.T, filter repository.TaskFilter) {
				require.NotNil(t, filter.CreatedTo)
				assert.Contains(t, *filter.CreatedTo, "2025-12-31")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseQuery(tt.query)
			require.NoError(t, err)

			ctx := context.Background()
			converterCtx := &ConverterContext{
				ProjectRepo: newMockProjectRepo(),
			}

			filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFilter != nil {
					tt.checkFilter(t, filter)
				}
			}
		})
	}
}

func TestConvertToTaskFilter_ComplexQuery(t *testing.T) {
	query := "status:pending priority:high @backend tag:bug -tag:wontfix due:<2025-12-31"

	parsed, err := ParseQuery(query)
	require.NoError(t, err)

	ctx := context.Background()
	converterCtx := &ConverterContext{
		ProjectRepo: newMockProjectRepo(),
	}

	filter, err := ConvertToTaskFilter(ctx, parsed, converterCtx)
	require.NoError(t, err)

	assert.Equal(t, domain.StatusPending, filter.Status)
	assert.Equal(t, domain.PriorityHigh, filter.Priority)
	require.NotNil(t, filter.ProjectID)
	assert.Equal(t, int64(1), *filter.ProjectID)
	assert.Equal(t, []string{"bug"}, filter.Tags)
	assert.Equal(t, []string{"wontfix"}, filter.ExcludeTags)
	require.NotNil(t, filter.DueDateTo)
	assert.Contains(t, *filter.DueDateTo, "2025-12-31")
}
