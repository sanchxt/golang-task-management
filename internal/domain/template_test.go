package domain

import (
	"strings"
	"testing"
)

func TestTemplateValidation(t *testing.T) {
	tests := []struct {
		name    string
		template *ProjectTemplate
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid template with basic task",
			template: &ProjectTemplate{
				Name:        "Web App",
				Description: "Standard web application template",
				TaskDefinitions: []TaskDefinition{
					{Title: "Setup repository", Priority: "high"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid template with multiple tasks",
			template: &ProjectTemplate{
				Name:        "Backend Service",
				Description: "Backend microservice template",
				TaskDefinitions: []TaskDefinition{
					{Title: "Setup project", Priority: "high", Tags: []string{"setup"}},
					{Title: "Implement API", Priority: "medium", Description: "Create REST API"},
					{Title: "Write tests", Priority: "low"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid template with project defaults",
			template: &ProjectTemplate{
				Name: "Frontend App",
				TaskDefinitions: []TaskDefinition{
					{Title: "Setup", Priority: "medium"},
				},
				ProjectDefaults: &ProjectDefaults{
					Color: "blue",
					Icon:  "🚀",
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			template: &ProjectTemplate{
				Name: "",
				TaskDefinitions: []TaskDefinition{
					{Title: "Task", Priority: "medium"},
				},
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "name too long",
			template: &ProjectTemplate{
				Name: strings.Repeat("a", 101),
				TaskDefinitions: []TaskDefinition{
					{Title: "Task", Priority: "medium"},
				},
			},
			wantErr: true,
			errMsg:  "name cannot exceed 100 characters",
		},
		{
			name: "description too long",
			template: &ProjectTemplate{
				Name:        "Test",
				Description: strings.Repeat("a", 501),
				TaskDefinitions: []TaskDefinition{
					{Title: "Task", Priority: "medium"},
				},
			},
			wantErr: true,
			errMsg:  "description cannot exceed 500 characters",
		},
		{
			name: "no task definitions",
			template: &ProjectTemplate{
				Name:            "Test",
				TaskDefinitions: []TaskDefinition{},
			},
			wantErr: true,
			errMsg:  "at least one task definition",
		},
		{
			name: "too many task definitions",
			template: &ProjectTemplate{
				Name:            "Test",
				TaskDefinitions: make([]TaskDefinition, 101),
			},
			wantErr: true,
			errMsg:  "cannot have more than 100 task definitions",
		},
		{
			name: "invalid task definition",
			template: &ProjectTemplate{
				Name: "Test",
				TaskDefinitions: []TaskDefinition{
					{Title: "", Priority: "medium"},
				},
			},
			wantErr: true,
			errMsg:  "task title cannot be empty",
		},
		{
			name: "invalid project defaults color",
			template: &ProjectTemplate{
				Name: "Test",
				TaskDefinitions: []TaskDefinition{
					{Title: "Task", Priority: "medium"},
				},
				ProjectDefaults: &ProjectDefaults{
					Color: "invalid-color",
				},
			},
			wantErr: true,
			errMsg:  "invalid color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestTaskDefinitionValidation(t *testing.T) {
	tests := []struct {
		name    string
		taskDef TaskDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid basic task",
			taskDef: TaskDefinition{Title: "Setup project", Priority: "medium"},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			taskDef: TaskDefinition{
				Title:       "Implement feature",
				Description: "Detailed description",
				Priority:    "high",
				Tags:        []string{"backend", "api"},
			},
			wantErr: false,
		},
		{
			name:    "valid without priority",
			taskDef: TaskDefinition{Title: "Task", Priority: ""},
			wantErr: false,
		},
		{
			name:    "empty title",
			taskDef: TaskDefinition{Title: "", Priority: "medium"},
			wantErr: true,
			errMsg:  "task title cannot be empty",
		},
		{
			name:    "title too long",
			taskDef: TaskDefinition{Title: strings.Repeat("a", 201), Priority: "medium"},
			wantErr: true,
			errMsg:  "task title cannot exceed 200 characters",
		},
		{
			name:    "description too long",
			taskDef: TaskDefinition{Title: "Task", Description: strings.Repeat("a", 1001)},
			wantErr: true,
			errMsg:  "task description cannot exceed 1000 characters",
		},
		{
			name:    "invalid priority",
			taskDef: TaskDefinition{Title: "Task", Priority: "invalid"},
			wantErr: true,
			errMsg:  "invalid priority",
		},
		{
			name:    "empty tag",
			taskDef: TaskDefinition{Title: "Task", Tags: []string{"valid", ""}},
			wantErr: true,
			errMsg:  "tag cannot be empty",
		},
		{
			name:    "tag too long",
			taskDef: TaskDefinition{Title: "Task", Tags: []string{strings.Repeat("a", 51)}},
			wantErr: true,
			errMsg:  "tag cannot exceed 50 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.taskDef.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestProjectDefaultsValidation(t *testing.T) {
	tests := []struct {
		name     string
		defaults ProjectDefaults
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid with color and icon",
			defaults: ProjectDefaults{Color: "blue", Icon: "🚀"},
			wantErr:  false,
		},
		{
			name:     "valid with only color",
			defaults: ProjectDefaults{Color: "green"},
			wantErr:  false,
		},
		{
			name:     "valid with only icon",
			defaults: ProjectDefaults{Icon: "📦"},
			wantErr:  false,
		},
		{
			name:     "empty defaults",
			defaults: ProjectDefaults{},
			wantErr:  false,
		},
		{
			name:     "invalid color",
			defaults: ProjectDefaults{Color: "invalid"},
			wantErr:  true,
			errMsg:   "invalid color",
		},
		{
			name:     "icon too long",
			defaults: ProjectDefaults{Icon: "12345678901"},
			wantErr:  true,
			errMsg:   "icon cannot exceed 10 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.defaults.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewTemplate(t *testing.T) {
	name := "Test Template"
	template := NewTemplate(name)

	if template.Name != name {
		t.Errorf("expected name %q, got %q", name, template.Name)
	}

	if template.TaskDefinitions == nil {
		t.Error("expected TaskDefinitions to be initialized, got nil")
	}

	if len(template.TaskDefinitions) != 0 {
		t.Errorf("expected empty TaskDefinitions, got %d items", len(template.TaskDefinitions))
	}

	if template.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set, got zero time")
	}

	if template.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set, got zero time")
	}
}

func TestNewTaskDefinition(t *testing.T) {
	title := "Test Task"
	taskDef := NewTaskDefinition(title)

	if taskDef.Title != title {
		t.Errorf("expected title %q, got %q", title, taskDef.Title)
	}

	if taskDef.Priority != "medium" {
		t.Errorf("expected default priority 'medium', got %q", taskDef.Priority)
	}

	if taskDef.Tags == nil {
		t.Error("expected Tags to be initialized, got nil")
	}

	if len(taskDef.Tags) != 0 {
		t.Errorf("expected empty Tags, got %d items", len(taskDef.Tags))
	}
}

func TestTemplateGetTaskCount(t *testing.T) {
	template := &ProjectTemplate{
		Name: "Test",
		TaskDefinitions: []TaskDefinition{
			{Title: "Task 1", Priority: "medium"},
			{Title: "Task 2", Priority: "high"},
			{Title: "Task 3", Priority: "low"},
		},
	}

	count := template.GetTaskCount()
	if count != 3 {
		t.Errorf("expected task count 3, got %d", count)
	}
}

func TestTemplateHasProjectDefaults(t *testing.T) {
	tests := []struct {
		name     string
		template *ProjectTemplate
		want     bool
	}{
		{
			name: "has defaults",
			template: &ProjectTemplate{
				ProjectDefaults: &ProjectDefaults{Color: "blue"},
			},
			want: true,
		},
		{
			name:     "no defaults",
			template: &ProjectTemplate{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.template.HasProjectDefaults()
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestTemplateAddTaskDefinition(t *testing.T) {
	template := NewTemplate("Test")

	err := template.AddTaskDefinition(TaskDefinition{
		Title:    "New Task",
		Priority: "high",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(template.TaskDefinitions) != 1 {
		t.Errorf("expected 1 task definition, got %d", len(template.TaskDefinitions))
	}

	err = template.AddTaskDefinition(TaskDefinition{
		Title:    "",
		Priority: "medium",
	})
	if err == nil {
		t.Error("expected error for invalid task definition, got nil")
	}

	if len(template.TaskDefinitions) != 1 {
		t.Errorf("expected 1 task definition after failed add, got %d", len(template.TaskDefinitions))
	}
}

func TestTemplateRemoveTaskDefinition(t *testing.T) {
	template := &ProjectTemplate{
		Name: "Test",
		TaskDefinitions: []TaskDefinition{
			{Title: "Task 1", Priority: "medium"},
			{Title: "Task 2", Priority: "high"},
			{Title: "Task 3", Priority: "low"},
		},
	}

	err := template.RemoveTaskDefinition(1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(template.TaskDefinitions) != 2 {
		t.Errorf("expected 2 task definitions, got %d", len(template.TaskDefinitions))
	}

	if template.TaskDefinitions[1].Title != "Task 3" {
		t.Errorf("expected second task to be 'Task 3', got %q", template.TaskDefinitions[1].Title)
	}

	err = template.RemoveTaskDefinition(5)
	if err == nil {
		t.Error("expected error for invalid index, got nil")
	}

	err = template.RemoveTaskDefinition(-1)
	if err == nil {
		t.Error("expected error for negative index, got nil")
	}
}

func TestTemplateUpdateTaskDefinition(t *testing.T) {
	template := &ProjectTemplate{
		Name: "Test",
		TaskDefinitions: []TaskDefinition{
			{Title: "Task 1", Priority: "medium"},
			{Title: "Task 2", Priority: "high"},
		},
	}

	newTaskDef := TaskDefinition{
		Title:       "Updated Task",
		Priority:    "urgent",
		Description: "New description",
	}

	err := template.UpdateTaskDefinition(0, newTaskDef)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if template.TaskDefinitions[0].Title != "Updated Task" {
		t.Errorf("expected title 'Updated Task', got %q", template.TaskDefinitions[0].Title)
	}

	err = template.UpdateTaskDefinition(1, TaskDefinition{Title: ""})
	if err == nil {
		t.Error("expected error for invalid task definition, got nil")
	}

	err = template.UpdateTaskDefinition(5, newTaskDef)
	if err == nil {
		t.Error("expected error for invalid index, got nil")
	}
}

func TestIsValidTaskPriority(t *testing.T) {
	tests := []struct {
		priority string
		want     bool
	}{
		{"low", true},
		{"medium", true},
		{"high", true},
		{"urgent", true},
		{"invalid", false},
		{"", false},
		{"LOW", false},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			got := isValidTaskPriority(tt.priority)
			if got != tt.want {
				t.Errorf("isValidTaskPriority(%q) = %v, want %v", tt.priority, got, tt.want)
			}
		})
	}
}
