package export

import (
	"context"
	"fmt"
	"io"
	"strings"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type MarkdownExporter struct {
	projectRepo repository.ProjectRepository
	taskRepo    repository.TaskRepository
}

func NewMarkdownExporter(projectRepo repository.ProjectRepository, taskRepo repository.TaskRepository) *MarkdownExporter {
	return &MarkdownExporter{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
	}
}

func (e *MarkdownExporter) ExportProjectToMarkdown(ctx context.Context, w io.Writer, projectID int64, includeDescendants, includeTasks bool) error {
	project, err := e.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	return e.writeProject(ctx, w, project, 1, includeDescendants, includeTasks)
}

func (e *MarkdownExporter) ExportTasksToMarkdown(ctx context.Context, w io.Writer, filter repository.TaskFilter) error {
	tasks, err := e.taskRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	fmt.Fprintln(w, "# Tasks")
	fmt.Fprintln(w)

	byStatus := make(map[domain.Status][]*domain.Task)
	for _, task := range tasks {
		byStatus[task.Status] = append(byStatus[task.Status], task)
	}

	for _, status := range []domain.Status{
		domain.StatusPending,
		domain.StatusInProgress,
		domain.StatusCompleted,
		domain.StatusCancelled,
	} {
		if tasks, ok := byStatus[status]; ok && len(tasks) > 0 {
			fmt.Fprintf(w, "## %s (%d)\n\n", strings.Title(string(status)), len(tasks))
			for _, task := range tasks {
				e.writeTask(w, task)
			}
			fmt.Fprintln(w)
		}
	}

	return nil
}

func (e *MarkdownExporter) writeProject(ctx context.Context, w io.Writer, project *domain.Project, level int, includeDescendants, includeTasks bool) error {
	heading := strings.Repeat("#", level)
	icon := project.Icon
	if icon == "" {
		icon = "📁"
	}
	fmt.Fprintf(w, "%s %s %s\n\n", heading, icon, project.Name)

	if project.Description != "" {
		fmt.Fprintf(w, "%s\n\n", project.Description)
	}

	fmt.Fprintf(w, "**Status**: %s", project.Status)
	if project.Color != "" {
		fmt.Fprintf(w, " | **Color**: %s", project.Color)
	}
	if project.IsFavorite {
		fmt.Fprintf(w, " | ⭐ **Favorite**")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w)

	if includeTasks {
		tasks, err := e.taskRepo.List(ctx, repository.TaskFilter{
			ProjectID: &project.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to list tasks for project %d: %w", project.ID, err)
		}

		if len(tasks) > 0 {
			completed := 0
			for _, task := range tasks {
				if task.Status == domain.StatusCompleted {
					completed++
				}
			}

			fmt.Fprintf(w, "### Tasks (%d/%d completed)\n\n", completed, len(tasks))

			byStatus := make(map[domain.Status][]*domain.Task)
			for _, task := range tasks {
				byStatus[task.Status] = append(byStatus[task.Status], task)
			}

			for _, status := range []domain.Status{domain.StatusPending, domain.StatusInProgress} {
				if tasks, ok := byStatus[status]; ok {
					for _, task := range tasks {
						e.writeTask(w, task)
					}
				}
			}

			if completed > 0 {
				fmt.Fprintln(w, "\n**Completed:**")
				for _, task := range byStatus[domain.StatusCompleted] {
					e.writeTask(w, task)
				}
			}

			if cancelled := byStatus[domain.StatusCancelled]; len(cancelled) > 0 {
				fmt.Fprintln(w, "\n**Cancelled:**")
				for _, task := range cancelled {
					e.writeTask(w, task)
				}
			}

			fmt.Fprintln(w)
		}
	}

	if includeDescendants {
		children, err := e.projectRepo.GetChildren(ctx, project.ID)
		if err != nil {
			return fmt.Errorf("failed to get children for project %d: %w", project.ID, err)
		}

		if len(children) > 0 {
			fmt.Fprintf(w, "### Subprojects (%d)\n\n", len(children))

			for _, child := range children {
				if err := e.writeProject(ctx, w, child, level+1, true, includeTasks); err != nil {
					return err
				}
			}
		}
	}

	fmt.Fprintln(w, "---")
	fmt.Fprintln(w)

	return nil
}

func (e *MarkdownExporter) writeTask(w io.Writer, task *domain.Task) {
	checkbox := "[ ]"
	if task.Status == domain.StatusCompleted {
		checkbox = "[x]"
	}

	priority := ""
	switch task.Priority {
	case domain.PriorityUrgent:
		priority = "🔴 "
	case domain.PriorityHigh:
		priority = "🟠 "
	case domain.PriorityMedium:
		priority = "🟡 "
	case domain.PriorityLow:
		priority = "🟢 "
	}

	fmt.Fprintf(w, "- %s %s**%s**", checkbox, priority, task.Title)

	metadata := []string{}

	if task.DueDate != nil {
		metadata = append(metadata, fmt.Sprintf("📅 %s", task.DueDate.Format("2006-01-02")))
	}

	if len(task.Tags) > 0 {
		tags := make([]string, len(task.Tags))
		for i, tag := range task.Tags {
			tags[i] = fmt.Sprintf("`%s`", tag)
		}
		metadata = append(metadata, strings.Join(tags, " "))
	}

	if len(metadata) > 0 {
		fmt.Fprintf(w, " (%s)", strings.Join(metadata, ", "))
	}

	fmt.Fprintln(w)

	if task.Description != "" {
		lines := strings.Split(task.Description, "\n")
		for _, line := range lines {
			fmt.Fprintf(w, "  > %s\n", line)
		}
	}
}
