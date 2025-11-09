package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"task-management/internal/repository"
)

type CSVExporter struct {
	projectRepo repository.ProjectRepository
	taskRepo    repository.TaskRepository
}

func NewCSVExporter(projectRepo repository.ProjectRepository, taskRepo repository.TaskRepository) *CSVExporter {
	return &CSVExporter{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
	}
}

func (e *CSVExporter) ExportTasksToCSV(ctx context.Context, w io.Writer, filter repository.TaskFilter) error {
	tasks, err := e.taskRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"ID", "Title", "Description", "Priority", "Status", "Tags", "Project", "Due Date", "Created At", "Updated At"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, task := range tasks {
		row := []string{
			strconv.FormatInt(task.ID, 10),
			task.Title,
			task.Description,
			string(task.Priority),
			string(task.Status),
			strings.Join(task.Tags, ";"),
			task.ProjectName,
			"",
			task.CreatedAt.Format("2006-01-02 15:04:05"),
			task.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		if task.DueDate != nil {
			row[7] = task.DueDate.Format("2006-01-02")
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

func (e *CSVExporter) ExportProjectsToCSV(ctx context.Context, w io.Writer, filter repository.ProjectFilter) error {
	projects, err := e.projectRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"ID", "Name", "Description", "Parent Path", "Status", "Color", "Icon", "Task Count", "Created At", "Updated At"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	for _, project := range projects {
		var parentPath string
		if project.ParentID != nil {
			path, err := e.projectRepo.GetPath(ctx, project.ID)
			if err == nil && len(path) > 1 {
				names := make([]string, 0, len(path)-1)
				for i := 0; i < len(path)-1; i++ {
					names = append(names, path[i].Name)
				}
				parentPath = strings.Join(names, " > ")
			}
		}

		taskCount, err := e.taskRepo.Count(ctx, repository.TaskFilter{
			ProjectID: &project.ID,
		})
		if err != nil {
			taskCount = 0
		}

		row := []string{
			strconv.FormatInt(project.ID, 10),
			project.Name,
			project.Description,
			parentPath,
			string(project.Status),
			project.Color,
			project.Icon,
			strconv.FormatInt(taskCount, 10),
			project.CreatedAt.Format("2006-01-02 15:04:05"),
			project.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}
