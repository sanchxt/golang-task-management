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

type Importer struct {
	projectRepo repository.ProjectRepository
	taskRepo    repository.TaskRepository
}

func NewImporter(projectRepo repository.ProjectRepository, taskRepo repository.TaskRepository) *Importer {
	return &Importer{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
	}
}

func (i *Importer) ImportProject(ctx context.Context, r io.Reader, parentID *int64, strategy ConflictStrategy) (*domain.Project, error) {
	var export ProjectExport
	if err := json.NewDecoder(r).Decode(&export); err != nil {
		return nil, fmt.Errorf("failed to decode project export: %w", err)
	}

	if export.Project == nil {
		return nil, fmt.Errorf("no project data in export")
	}

	return i.importProjectData(ctx, export.Project, parentID, strategy)
}

func (i *Importer) RestoreBackup(ctx context.Context, r io.Reader, strategy ConflictStrategy) error {
	var backup BackupData
	if err := json.NewDecoder(r).Decode(&backup); err != nil {
		return fmt.Errorf("failed to decode backup: %w", err)
	}

	projectIDMap := make(map[int64]int64)

	for _, projectData := range backup.Projects {
		if projectData.ParentID != nil {
			if _, ok := projectIDMap[*projectData.ParentID]; !ok {
				continue
			}
		}

		project, err := i.importProjectData(ctx, projectData, nil, strategy)
		if err != nil {
			return fmt.Errorf("failed to import project %s: %w", projectData.Name, err)
		}
		projectIDMap[projectData.ID] = project.ID
	}

	for _, projectData := range backup.Projects {
		if projectData.ParentID != nil {
			if newParentID, ok := projectIDMap[*projectData.ParentID]; ok {
				projectData.ParentID = &newParentID

				if _, ok := projectIDMap[projectData.ID]; !ok {
					project, err := i.importProjectData(ctx, projectData, projectData.ParentID, strategy)
					if err != nil {
						return fmt.Errorf("failed to import project %s: %w", projectData.Name, err)
					}
					projectIDMap[projectData.ID] = project.ID
				}
			}
		}
	}

	for _, taskData := range backup.Tasks {
		var newProjectID *int64
		if taskData.ID != 0 {
			for _, pd := range backup.Projects {
				for _, td := range pd.Tasks {
					if td.ID == taskData.ID {
						if newID, ok := projectIDMap[pd.ID]; ok {
							newProjectID = &newID
						}
						break
					}
				}
			}
		}

		if err := i.importTaskData(ctx, taskData, newProjectID); err != nil {
			return fmt.Errorf("failed to import task %s: %w", taskData.Title, err)
		}
	}

	return nil
}

func (i *Importer) importProjectData(ctx context.Context, data *ProjectData, parentID *int64, strategy ConflictStrategy) (*domain.Project, error) {
	existing, err := i.projectRepo.GetByName(ctx, data.Name)
	if err == nil && existing != nil {
		switch strategy {
		case ConflictStrategySkip:
			return existing, nil
		case ConflictStrategyOverwrite:
			existing.Description = data.Description
			existing.Color = data.Color
			existing.Icon = data.Icon
			existing.Status = domain.ProjectStatus(data.Status)
			existing.IsFavorite = data.IsFavorite
			if err := i.projectRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("failed to update existing project: %w", err)
			}
			return existing, nil
		case ConflictStrategyMerge:
			return existing, nil
		}
	}

	project := &domain.Project{
		Name:        data.Name,
		Description: data.Description,
		ParentID:    parentID,
		Color:       data.Color,
		Icon:        data.Icon,
		Status:      domain.ProjectStatus(data.Status),
		IsFavorite:  data.IsFavorite,
		CreatedAt:   data.CreatedAt,
		UpdatedAt:   data.UpdatedAt,
	}

	if err := i.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	for _, taskData := range data.Tasks {
		if err := i.importTaskData(ctx, taskData, &project.ID); err != nil {
			return nil, fmt.Errorf("failed to import task: %w", err)
		}
	}

	for _, childData := range data.Children {
		if _, err := i.importProjectData(ctx, childData, &project.ID, strategy); err != nil {
			return nil, fmt.Errorf("failed to import child project: %w", err)
		}
	}

	return project, nil
}

func (i *Importer) importTaskData(ctx context.Context, data *TaskData, projectID *int64) error {
	task := &domain.Task{
		Title:       data.Title,
		Description: data.Description,
		Priority:    domain.Priority(data.Priority),
		Status:      domain.Status(data.Status),
		Tags:        data.Tags,
		ProjectID:   projectID,
		CreatedAt:   data.CreatedAt,
		UpdatedAt:   data.UpdatedAt,
	}

	if data.DueDate != nil {
		dueDate, err := time.Parse("2006-01-02", *data.DueDate)
		if err == nil {
			task.DueDate = &dueDate
		}
	}

	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task data: %w", err)
	}

	return i.taskRepo.Create(ctx, task)
}
