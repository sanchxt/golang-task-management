package domain

import (
	"errors"
	"strings"
	"time"
)

type ProjectTemplate struct {
	ID              int64             `db:"id" json:"id"`
	Name            string            `db:"name" json:"name"`
	Description     string            `db:"description" json:"description"`
	TaskDefinitions []TaskDefinition  `db:"task_definitions" json:"task_definitions"`
	ProjectDefaults *ProjectDefaults  `db:"project_defaults" json:"project_defaults,omitempty"`
	CreatedAt       time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at" json:"updated_at"`
}

type TaskDefinition struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    string   `json:"priority"`
	Tags        []string `json:"tags,omitempty"`
}

type ProjectDefaults struct {
	Color string `json:"color,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

func (t *ProjectTemplate) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("template name cannot be empty")
	}

	if len(t.Name) > 100 {
		return errors.New("template name cannot exceed 100 characters")
	}

	if len(t.Description) > 500 {
		return errors.New("template description cannot exceed 500 characters")
	}

	if len(t.TaskDefinitions) == 0 {
		return errors.New("template must have at least one task definition")
	}

	if len(t.TaskDefinitions) > 100 {
		return errors.New("template cannot have more than 100 task definitions")
	}

	for i, taskDef := range t.TaskDefinitions {
		if err := taskDef.Validate(); err != nil {
			return errors.New("task definition " + string(rune(i+1)) + ": " + err.Error())
		}
	}

	if t.ProjectDefaults != nil {
		if err := t.ProjectDefaults.Validate(); err != nil {
			return errors.New("project defaults: " + err.Error())
		}
	}

	return nil
}

func (td *TaskDefinition) Validate() error {
	if strings.TrimSpace(td.Title) == "" {
		return errors.New("task title cannot be empty")
	}

	if len(td.Title) > 200 {
		return errors.New("task title cannot exceed 200 characters")
	}

	if len(td.Description) > 1000 {
		return errors.New("task description cannot exceed 1000 characters")
	}

	if td.Priority != "" && !isValidTaskPriority(td.Priority) {
		return errors.New("invalid priority: must be low, medium, high, or urgent")
	}

	for _, tag := range td.Tags {
		if strings.TrimSpace(tag) == "" {
			return errors.New("tag cannot be empty")
		}
		if len(tag) > 50 {
			return errors.New("tag cannot exceed 50 characters")
		}
	}

	return nil
}

func (pd *ProjectDefaults) Validate() error {
	if pd.Color != "" && !isValidColor(pd.Color) {
		return errors.New("invalid color: must be a valid terminal color name")
	}

	if len(pd.Icon) > 10 {
		return errors.New("icon cannot exceed 10 characters")
	}

	return nil
}

func NewTemplate(name string) *ProjectTemplate {
	now := time.Now()
	return &ProjectTemplate{
		Name:            name,
		TaskDefinitions: make([]TaskDefinition, 0),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func NewTaskDefinition(title string) TaskDefinition {
	return TaskDefinition{
		Title:    title,
		Priority: "medium",
		Tags:     make([]string, 0),
	}
}

func (t *ProjectTemplate) GetTaskCount() int {
	return len(t.TaskDefinitions)
}

func (t *ProjectTemplate) HasProjectDefaults() bool {
	return t.ProjectDefaults != nil
}

func (t *ProjectTemplate) AddTaskDefinition(taskDef TaskDefinition) error {
	if err := taskDef.Validate(); err != nil {
		return err
	}

	t.TaskDefinitions = append(t.TaskDefinitions, taskDef)
	return nil
}

func (t *ProjectTemplate) RemoveTaskDefinition(index int) error {
	if index < 0 || index >= len(t.TaskDefinitions) {
		return errors.New("invalid task definition index")
	}

	t.TaskDefinitions = append(t.TaskDefinitions[:index], t.TaskDefinitions[index+1:]...)
	return nil
}

func (t *ProjectTemplate) UpdateTaskDefinition(index int, taskDef TaskDefinition) error {
	if index < 0 || index >= len(t.TaskDefinitions) {
		return errors.New("invalid task definition index")
	}

	if err := taskDef.Validate(); err != nil {
		return err
	}

	t.TaskDefinitions[index] = taskDef
	return nil
}

func isValidTaskPriority(priority string) bool {
	switch priority {
	case "low", "medium", "high", "urgent":
		return true
	default:
		return false
	}
}
