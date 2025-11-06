package sqlite

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// create temp db file
	tmpFile, err := os.CreateTemp("", "taskflow_test_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	dbPath := tmpFile.Name()

	// initialize db
	db, err := NewDB(Config{Path: dbPath})
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestTaskRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	t.Run("create valid task", func(t *testing.T) {
		task := domain.NewTask("Test Task")
		task.Description = "This is a test task"
		task.Priority = domain.PriorityHigh
		task.Tags = []string{"test", "important"}
		task.Project = "Test Project"

		err := repo.Create(ctx, task)
		require.NoError(t, err)
		assert.NotZero(t, task.ID)
	})

	t.Run("create task with invalid data", func(t *testing.T) {
		task := &domain.Task{
			Title: "",
		}

		err := repo.Create(ctx, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("create task with due date", func(t *testing.T) {
		dueDate := time.Now().Add(24 * time.Hour)
		task := domain.NewTask("Task with due date")
		task.DueDate = &dueDate

		err := repo.Create(ctx, task)
		require.NoError(t, err)
		assert.NotZero(t, task.ID)
	})
}

func TestTaskRepository_GetByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	originalTask := domain.NewTask("Test Task")
	originalTask.Description = "Test Description"
	originalTask.Priority = domain.PriorityUrgent
	originalTask.Tags = []string{"tag1", "tag2"}
	originalTask.Project = "Project Alpha"

	err := repo.Create(ctx, originalTask)
	require.NoError(t, err)

	t.Run("get existing task", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, originalTask.ID)
		require.NoError(t, err)

		assert.Equal(t, originalTask.ID, retrieved.ID)
		assert.Equal(t, originalTask.Title, retrieved.Title)
		assert.Equal(t, originalTask.Description, retrieved.Description)
		assert.Equal(t, originalTask.Priority, retrieved.Priority)
		assert.Equal(t, originalTask.Status, retrieved.Status)
		assert.Equal(t, originalTask.Tags, retrieved.Tags)
		assert.Equal(t, originalTask.Project, retrieved.Project)
	})

	t.Run("get non-existent task", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})
}

func TestTaskRepository_List(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	tasks := []*domain.Task{
		{Title: "Task 1", Priority: domain.PriorityHigh, Status: domain.StatusPending, Project: "Project A"},
		{Title: "Task 2", Priority: domain.PriorityLow, Status: domain.StatusCompleted, Project: "Project A"},
		{Title: "Task 3", Priority: domain.PriorityHigh, Status: domain.StatusPending, Project: "Project B"},
	}

	for _, task := range tasks {
		err := repo.Create(ctx, task)
		require.NoError(t, err)
	}

	t.Run("list all tasks", func(t *testing.T) {
		retrieved, err := repo.List(ctx, repository.TaskFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 3)
	})

	t.Run("filter by status", func(t *testing.T) {
		retrieved, err := repo.List(ctx, repository.TaskFilter{
			Status: domain.StatusPending,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 2)

		for _, task := range retrieved {
			assert.Equal(t, domain.StatusPending, task.Status)
		}
	})

	t.Run("filter by priority", func(t *testing.T) {
		retrieved, err := repo.List(ctx, repository.TaskFilter{
			Priority: domain.PriorityHigh,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 2)

		for _, task := range retrieved {
			assert.Equal(t, domain.PriorityHigh, task.Priority)
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		retrieved, err := repo.List(ctx, repository.TaskFilter{
			Project: "Project A",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 2)

		for _, task := range retrieved {
			assert.Equal(t, "Project A", task.Project)
		}
	})
}

func TestTaskRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	task := domain.NewTask("Original Task")
	err := repo.Create(ctx, task)
	require.NoError(t, err)

	t.Run("update existing task", func(t *testing.T) {
		task.Title = "Updated Task"
		task.Description = "Updated Description"
		task.Status = domain.StatusInProgress
		task.Priority = domain.PriorityUrgent

		err := repo.Update(ctx, task)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Task", retrieved.Title)
		assert.Equal(t, "Updated Description", retrieved.Description)
		assert.Equal(t, domain.StatusInProgress, retrieved.Status)
		assert.Equal(t, domain.PriorityUrgent, retrieved.Priority)
	})

	t.Run("update non-existent task", func(t *testing.T) {
		nonExistent := domain.NewTask("Non-existent")
		nonExistent.ID = 99999

		err := repo.Update(ctx, nonExistent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})

	t.Run("update with invalid data", func(t *testing.T) {
		task.Title = ""

		err := repo.Update(ctx, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

func TestTaskRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	t.Run("delete existing task", func(t *testing.T) {
		task := domain.NewTask("Task to delete")
		err := repo.Create(ctx, task)
		require.NoError(t, err)

		err = repo.Delete(ctx, task.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, task.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})

	t.Run("delete non-existent task", func(t *testing.T) {
		err := repo.Delete(ctx, 99999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})
}

func TestTaskRepository_Count(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	tasks := []*domain.Task{
		{Title: "Task 1", Priority: domain.PriorityHigh, Status: domain.StatusPending, Project: "Project A"},
		{Title: "Task 2", Priority: domain.PriorityLow, Status: domain.StatusCompleted, Project: "Project A"},
		{Title: "Task 3", Priority: domain.PriorityHigh, Status: domain.StatusPending, Project: "Project B"},
		{Title: "Task 4", Priority: domain.PriorityMedium, Status: domain.StatusInProgress, Project: "Project B"},
		{Title: "Task 5", Priority: domain.PriorityUrgent, Status: domain.StatusPending, Project: "Project C"},
	}

	for _, task := range tasks {
		err := repo.Create(ctx, task)
		require.NoError(t, err)
	}

	t.Run("count all tasks", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.TaskFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(5))
	})

	t.Run("count by status", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.TaskFilter{
			Status: domain.StatusPending,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3))
	})

	t.Run("count by priority", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.TaskFilter{
			Priority: domain.PriorityHigh,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
	})

	t.Run("count by project", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.TaskFilter{
			Project: "Project A",
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
	})

	t.Run("count with multiple filters", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.TaskFilter{
			Status:   domain.StatusPending,
			Priority: domain.PriorityHigh,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
	})

	t.Run("count with no matches", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.TaskFilter{
			Project: "Non-existent Project",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestTaskRepository_Pagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	for i := 1; i <= 15; i++ {
		task := &domain.Task{
			Title:    fmt.Sprintf("Task %d", i),
			Priority: domain.PriorityMedium,
			Status:   domain.StatusPending,
		}
		err := repo.Create(ctx, task)
		require.NoError(t, err)
		time.Sleep(time.Millisecond)
	}

	t.Run("list first page", func(t *testing.T) {
		filter := repository.TaskFilter{
			Limit:  5,
			Offset: 0,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
	})

	t.Run("list second page", func(t *testing.T) {
		filter := repository.TaskFilter{
			Limit:  5,
			Offset: 5,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
	})

	t.Run("list third page", func(t *testing.T) {
		filter := repository.TaskFilter{
			Limit:  5,
			Offset: 10,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 5)
	})

	t.Run("list page beyond results", func(t *testing.T) {
		filter := repository.TaskFilter{
			Limit:  5,
			Offset: 100,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, tasks, 0)
	})

	t.Run("list all with no limit", func(t *testing.T) {
		filter := repository.TaskFilter{
			Limit:  0,
			Offset: 0,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 15)
	})

	t.Run("pagination with filter", func(t *testing.T) {
		for i := 1; i <= 7; i++ {
			task := &domain.Task{
				Title:    fmt.Sprintf("High Priority Task %d", i),
				Priority: domain.PriorityHigh,
				Status:   domain.StatusPending,
			}
			err := repo.Create(ctx, task)
			require.NoError(t, err)
		}

		filter := repository.TaskFilter{
			Priority: domain.PriorityHigh,
			Limit:    3,
			Offset:   0,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, tasks, 3)

		for _, task := range tasks {
			assert.Equal(t, domain.PriorityHigh, task.Priority)
		}
	})
}

func TestTaskRepository_Search(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	testTasks := []*domain.Task{
		{
			Title:       "Implement user authentication",
			Description: "Add login and signup functionality",
			Priority:    domain.PriorityHigh,
			Status:      domain.StatusPending,
			Project:     "Backend",
			Tags:        []string{"auth", "security"},
		},
		{
			Title:       "Fix login bug",
			Description: "Users cannot log in after password reset",
			Priority:    domain.PriorityUrgent,
			Status:      domain.StatusInProgress,
			Project:     "Backend",
			Tags:        []string{"bug", "auth"},
		},
		{
			Title:       "Write documentation",
			Description: "Document the API endpoints",
			Priority:    domain.PriorityMedium,
			Status:      domain.StatusPending,
			Project:     "Documentation",
			Tags:        []string{"docs"},
		},
		{
			Title:       "Refactor authentication service",
			Description: "Clean up code in auth module",
			Priority:    domain.PriorityLow,
			Status:      domain.StatusPending,
			Project:     "Backend",
			Tags:        []string{"refactor", "auth"},
		},
	}

	for _, task := range testTasks {
		err := repo.Create(ctx, task)
		require.NoError(t, err)
	}

	t.Run("search in title - text mode", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "authentication",
			SearchMode:  "text",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 2)
	})

	t.Run("search in description - text mode", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "password",
			SearchMode:  "text",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("search in project - text mode", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "Documentation",
			SearchMode:  "text",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("search in tags - text mode", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "auth",
			SearchMode:  "text",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 3) 
	})

	t.Run("search case insensitive", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "AUTHENTICATION",
			SearchMode:  "text",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 2)
	})

	t.Run("search with regex mode - basic pattern", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "auth.*service",
			SearchMode:  "regex",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("search with regex mode - word boundary", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "\\blog\\b",
			SearchMode:  "regex",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("search with no results", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "nonexistent",
			SearchMode:  "text",
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, tasks, 0)
	})

	t.Run("search combined with filters", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "auth",
			SearchMode:  "text",
			Priority:    domain.PriorityHigh,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(tasks), 1)

		for _, task := range tasks {
			assert.Equal(t, domain.PriorityHigh, task.Priority)
		}
	})

	t.Run("search with pagination", func(t *testing.T) {
		filter := repository.TaskFilter{
			SearchQuery: "auth",
			SearchMode:  "text",
			Limit:       2,
			Offset:      0,
		}
		tasks, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(tasks), 2)
	})
}

func TestTaskRepository_Sorting(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	now := time.Now()
	tasks := []*domain.Task{
		{
			Title:    "Zebra Task",
			Priority: domain.PriorityLow,
			Status:   domain.StatusPending,
			DueDate:  &[]time.Time{now.Add(48 * time.Hour)}[0],
		},
		{
			Title:    "Alpha Task",
			Priority: domain.PriorityUrgent,
			Status:   domain.StatusPending,
			DueDate:  &[]time.Time{now.Add(24 * time.Hour)}[0],
		},
		{
			Title:    "Beta Task",
			Priority: domain.PriorityHigh,
			Status:   domain.StatusInProgress,
			DueDate:  &[]time.Time{now.Add(72 * time.Hour)}[0],
		},
		{
			Title:    "Gamma Task",
			Priority: domain.PriorityMedium,
			Status:   domain.StatusCompleted,
			DueDate:  nil,
		},
	}

	for _, task := range tasks {
		err := repo.Create(ctx, task)
		require.NoError(t, err)
		time.Sleep(2 * time.Millisecond)
	}

	t.Run("sort by created_at desc (default)", func(t *testing.T) {
		filter := repository.TaskFilter{
			SortBy:    "created_at",
			SortOrder: "desc",
		}
		retrieved, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)

		assert.Equal(t, "Gamma Task", retrieved[len(retrieved)-4].Title)
	})

	t.Run("sort by created_at asc", func(t *testing.T) {
		filter := repository.TaskFilter{
			SortBy:    "created_at",
			SortOrder: "asc",
		}
		retrieved, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)
	})

	t.Run("sort by title asc", func(t *testing.T) {
		filter := repository.TaskFilter{
			SortBy:    "title",
			SortOrder: "asc",
		}
		retrieved, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)

		var testResults []*domain.Task
		for _, task := range retrieved {
			if task.Title == "Alpha Task" || task.Title == "Beta Task" ||
				task.Title == "Gamma Task" || task.Title == "Zebra Task" {
				testResults = append(testResults, task)
			}
		}

		assert.GreaterOrEqual(t, len(testResults), 4)
		var alphaIdx, zebraIdx int
		for i, task := range testResults {
			if task.Title == "Alpha Task" {
				alphaIdx = i
			}
			if task.Title == "Zebra Task" {
				zebraIdx = i
			}
		}
		assert.Less(t, alphaIdx, zebraIdx)
	})

	t.Run("sort by priority desc", func(t *testing.T) {
		filter := repository.TaskFilter{
			SortBy:    "priority",
			SortOrder: "desc",
		}
		retrieved, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)

		for i, task := range retrieved {
			if task.Title == "Alpha Task" {
				assert.Equal(t, domain.PriorityUrgent, task.Priority)
				assert.Less(t, i, 5)
				break
			}
		}
	})

	t.Run("sort by due_date asc", func(t *testing.T) {
		filter := repository.TaskFilter{
			SortBy:    "due_date",
			SortOrder: "asc",
		}
		retrieved, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)

		var withDueDates []*domain.Task
		for _, task := range retrieved {
			if task.DueDate != nil &&
				(task.Title == "Alpha Task" || task.Title == "Beta Task" || task.Title == "Zebra Task") {
				withDueDates = append(withDueDates, task)
			}
		}

		if len(withDueDates) >= 2 {
			assert.True(t, withDueDates[0].DueDate.Before(*withDueDates[1].DueDate) ||
				withDueDates[0].DueDate.Equal(*withDueDates[1].DueDate))
		}
	})

	t.Run("default sort when not specified", func(t *testing.T) {
		filter := repository.TaskFilter{}
		retrieved, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrieved), 4)
	})
}
