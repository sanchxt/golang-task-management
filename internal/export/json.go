package export

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type JSONExporter struct {
	projectRepo repository.ProjectRepository
	taskRepo    repository.TaskRepository
}

func NewJSONExporter(projectRepo repository.ProjectRepository, taskRepo repository.TaskRepository) *JSONExporter {
	return &JSONExporter{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
	}
}

func (e *JSONExporter) ExportProject(ctx context.Context, projectID int64, includeDescendants, includeTasks bool) (*ProjectExport, error) {
	project, err := e.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	projectData, err := e.convertProject(ctx, project, includeDescendants, includeTasks)
	if err != nil {
		return nil, err
	}

	return &ProjectExport{
		Version: "1.0",
		Project: projectData,
	}, nil
}

func (e *JSONExporter) ExportProjectToWriter(ctx context.Context, w io.Writer, projectID int64, includeDescendants, includeTasks bool) error {
	export, err := e.ExportProject(ctx, projectID, includeDescendants, includeTasks)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(export)
}

func (e *JSONExporter) ExportTasks(ctx context.Context, filter repository.TaskFilter) ([]*TaskData, error) {
	tasks, err := e.taskRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	taskData := make([]*TaskData, 0, len(tasks))
	for _, task := range tasks {
		taskData = append(taskData, e.convertTask(task))
	}

	return taskData, nil
}

func (e *JSONExporter) ExportTasksToWriter(ctx context.Context, w io.Writer, filter repository.TaskFilter) error {
	tasks, err := e.ExportTasks(ctx, filter)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"version": "1.0",
		"tasks":   tasks,
	})
}

func (e *JSONExporter) CreateFullBackup(ctx context.Context) (*BackupData, error) {
	projects, err := e.projectRepo.List(ctx, repository.ProjectFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	projectData := make([]*ProjectData, 0, len(projects))
	for _, project := range projects {
		pd, err := e.convertProject(ctx, project, false, false)
		if err != nil {
			return nil, err
		}
		projectData = append(projectData, pd)
	}

	tasks, err := e.taskRepo.List(ctx, repository.TaskFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	taskData := make([]*TaskData, 0, len(tasks))
	for _, task := range tasks {
		taskData = append(taskData, e.convertTask(task))
	}

	return &BackupData{
		Version:   "1.0",
		Timestamp: time.Now(),
		Projects:  projectData,
		Tasks:     taskData,
	}, nil
}

func (e *JSONExporter) CreateFullBackupToWriter(ctx context.Context, w io.Writer) error {
	backup, err := e.CreateFullBackup(ctx)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backup)
}

func (e *JSONExporter) convertProject(ctx context.Context, project *domain.Project, includeDescendants, includeTasks bool) (*ProjectData, error) {
	pd := &ProjectData{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		ParentID:    project.ParentID,
		Color:       project.Color,
		Icon:        project.Icon,
		Status:      string(project.Status),
		IsFavorite:  project.IsFavorite,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}

	if includeTasks {
		tasks, err := e.taskRepo.List(ctx, repository.TaskFilter{
			ProjectID: &project.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks for project %d: %w", project.ID, err)
		}

		pd.Tasks = make([]*TaskData, 0, len(tasks))
		for _, task := range tasks {
			pd.Tasks = append(pd.Tasks, e.convertTask(task))
		}
	}

	if includeDescendants {
		children, err := e.projectRepo.GetChildren(ctx, project.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get children for project %d: %w", project.ID, err)
		}

		pd.Children = make([]*ProjectData, 0, len(children))
		for _, child := range children {
			childData, err := e.convertProject(ctx, child, true, includeTasks)
			if err != nil {
				return nil, err
			}
			pd.Children = append(pd.Children, childData)
		}
	}

	return pd, nil
}

func (e *JSONExporter) convertTask(task *domain.Task) *TaskData {
	td := &TaskData{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Priority:    string(task.Priority),
		Status:      string(task.Status),
		Tags:        task.Tags,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}

	if task.DueDate != nil {
		dueDate := task.DueDate.Format("2006-01-02")
		td.DueDate = &dueDate
	}

	return td
}
