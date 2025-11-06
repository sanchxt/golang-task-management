package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

// Message types for async operations

// tasksLoadedMsg is sent when tasks are successfully loaded
type tasksLoadedMsg struct {
	tasks      []*domain.Task
	totalCount int64
}

// taskUpdatedMsg is sent when a task is successfully updated
type taskUpdatedMsg struct {
	task *domain.Task
}

// taskDeletedMsg is sent when a task is successfully deleted
type taskDeletedMsg struct {
	taskID int64
}

// errMsg wraps errors from async operations
type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

// Bubble Tea commands for async operations

// fetchTasksCmd fetches tasks based on current filter and pagination
func fetchTasksCmd(ctx context.Context, repo repository.TaskRepository, filter repository.TaskFilter, page int, pageSize int) tea.Cmd {
	return func() tea.Msg {
		// calculate offset
		filter.Limit = pageSize
		filter.Offset = (page - 1) * pageSize

		// fetch tasks
		tasks, err := repo.List(ctx, filter)
		if err != nil {
			return errMsg{err}
		}

		// get total count
		totalCount, err := repo.Count(ctx, filter)
		if err != nil {
			return errMsg{err}
		}

		return tasksLoadedMsg{
			tasks:      tasks,
			totalCount: totalCount,
		}
	}
}

// updateTaskCmd updates a task in the database
func updateTaskCmd(ctx context.Context, repo repository.TaskRepository, task *domain.Task) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Update(ctx, task); err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{task: task}
	}
}

// deleteTaskCmd deletes a task from the database
func deleteTaskCmd(ctx context.Context, repo repository.TaskRepository, taskID int64) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Delete(ctx, taskID); err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{taskID: taskID}
	}
}

// refreshCmd is a convenience wrapper for fetching tasks after an operation
func (m *Model) refreshCmd() tea.Cmd {
	return fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize)
}
