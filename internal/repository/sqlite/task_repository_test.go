package sqlite

import (
	"context"
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

	// return cleanup function
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
			Title: "", // should fail validation
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

	// create a task first
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

	// create multiple tasks
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

	// create a task first
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

		// verify update
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
		task.Title = "" // should fail

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

		// verify deletion
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
