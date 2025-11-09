package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/query"
	"task-management/internal/repository"
)


type tasksLoadedMsg struct {
	tasks      []*domain.Task
	totalCount int64
}

type taskUpdatedMsg struct {
	task *domain.Task
}

type taskCreatedMsg struct {
	task *domain.Task
}

type taskDeletedMsg struct {
	taskID int64
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type queryParsedMsg struct {
	filter   repository.TaskFilter
	queryStr string
	err      error
}


func fetchTasksCmd(ctx context.Context, repo repository.TaskRepository, filter repository.TaskFilter, page int, pageSize int) tea.Cmd {
	return func() tea.Msg {
		filter.Limit = pageSize
		filter.Offset = (page - 1) * pageSize

		tasks, err := repo.List(ctx, filter)
		if err != nil {
			return errMsg{err}
		}

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

func createTaskCmd(ctx context.Context, repo repository.TaskRepository, task *domain.Task) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Create(ctx, task); err != nil {
			return errMsg{err}
		}
		return taskCreatedMsg{task: task}
	}
}

func updateTaskCmd(ctx context.Context, repo repository.TaskRepository, task *domain.Task) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Update(ctx, task); err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{task: task}
	}
}

func deleteTaskCmd(ctx context.Context, repo repository.TaskRepository, taskID int64) tea.Cmd {
	return func() tea.Msg {
		if err := repo.Delete(ctx, taskID); err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{taskID: taskID}
	}
}

func (m *Model) refreshCmd() tea.Cmd {
	return fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize)
}

func parseQueryLanguageCmd(ctx context.Context, queryStr string, converterCtx *query.ConverterContext) tea.Cmd {
	return func() tea.Msg {
		parsed, err := query.ParseQuery(queryStr)
		if err != nil {
			return queryParsedMsg{err: fmt.Errorf("query parse error: %w", err)}
		}

		filter, err := query.ConvertToTaskFilter(ctx, parsed, converterCtx)
		if err != nil {
			return queryParsedMsg{err: fmt.Errorf("query conversion error: %w", err)}
		}

		return queryParsedMsg{filter: filter, queryStr: queryStr}
	}
}
