package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type (
	projectsLoadedMsg struct {
		projects []*domain.Project
		err      error
	}

	projectCreatedMsg struct {
		project *domain.Project
		err     error
	}

	projectUpdatedMsg struct {
		project *domain.Project
		err     error
	}

	projectDeletedMsg struct {
		projectID int64
		err       error
	}

	projectStatsMsg struct {
		projectID int64
		stats     map[domain.Status]int
		taskCount int
		err       error
	}
)

func fetchProjectsCmd(ctx context.Context, repo repository.ProjectRepository, filter repository.ProjectFilter) tea.Cmd {
	return func() tea.Msg {
		projects, err := repo.List(ctx, filter)
		if err != nil {
			return projectsLoadedMsg{err: err}
		}
		return projectsLoadedMsg{projects: projects}
	}
}

func createProjectCmd(ctx context.Context, repo repository.ProjectRepository, project *domain.Project) tea.Cmd {
	return func() tea.Msg {
		err := repo.Create(ctx, project)
		if err != nil {
			return projectCreatedMsg{err: err}
		}
		return projectCreatedMsg{project: project}
	}
}

func updateProjectCmd(ctx context.Context, repo repository.ProjectRepository, project *domain.Project) tea.Cmd {
	return func() tea.Msg {
		err := repo.Update(ctx, project)
		if err != nil {
			return projectUpdatedMsg{err: err}
		}
		return projectUpdatedMsg{project: project}
	}
}

func deleteProjectCmd(ctx context.Context, repo repository.ProjectRepository, projectID int64) tea.Cmd {
	return func() tea.Msg {
		err := repo.Delete(ctx, projectID)
		if err != nil {
			return projectDeletedMsg{err: err}
		}
		return projectDeletedMsg{projectID: projectID}
	}
}

func archiveProjectCmd(ctx context.Context, repo repository.ProjectRepository, projectID int64) tea.Cmd {
	return func() tea.Msg {
		err := repo.Archive(ctx, projectID)
		if err != nil {
			return projectUpdatedMsg{err: err}
		}
		project, err := repo.GetByID(ctx, projectID)
		if err != nil {
			return projectUpdatedMsg{err: err}
		}
		return projectUpdatedMsg{project: project}
	}
}

func fetchProjectStatsCmd(ctx context.Context, repo repository.ProjectRepository, projectID int64) tea.Cmd {
	return func() tea.Msg {
		taskCount, err := repo.GetTaskCount(ctx, projectID)
		if err != nil {
			return projectStatsMsg{projectID: projectID, err: err}
		}

		stats, err := repo.GetTaskCountByStatus(ctx, projectID)
		if err != nil {
			return projectStatsMsg{projectID: projectID, err: err}
		}

		return projectStatsMsg{
			projectID: projectID,
			stats:     stats,
			taskCount: taskCount,
		}
	}
}
