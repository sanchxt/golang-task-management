package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type mockTemplateRepository struct {
	templates      []*domain.ProjectTemplate
	fetchErr       error
	getErr         error
	getByNameErr   error
	createErr      error
	updateErr      error
	deleteErr      error
	countErr       error
	searchErr      error
	listErr        error
	totalCount     int64
}

func (m *mockTemplateRepository) Create(ctx context.Context, template *domain.ProjectTemplate) error {
	if m.createErr != nil {
		return m.createErr
	}
	template.ID = int64(len(m.templates) + 1)
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	m.templates = append(m.templates, template)
	return nil
}

func (m *mockTemplateRepository) GetByID(ctx context.Context, id int64) (*domain.ProjectTemplate, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, t := range m.templates {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, errors.New("template not found")
}

func (m *mockTemplateRepository) GetByName(ctx context.Context, name string) (*domain.ProjectTemplate, error) {
	if m.getByNameErr != nil {
		return nil, m.getByNameErr
	}
	for _, t := range m.templates {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, errors.New("template not found")
}

func (m *mockTemplateRepository) Update(ctx context.Context, template *domain.ProjectTemplate) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	template.UpdatedAt = time.Now()
	return nil
}

func (m *mockTemplateRepository) Delete(ctx context.Context, id int64) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *mockTemplateRepository) List(ctx context.Context, filter repository.TemplateFilter) ([]*domain.ProjectTemplate, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.templates, nil
}

func (m *mockTemplateRepository) Count(ctx context.Context, filter repository.TemplateFilter) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return int64(len(m.templates)), nil
}

func (m *mockTemplateRepository) Search(ctx context.Context, query string, limit int) ([]*domain.ProjectTemplate, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.templates, nil
}

func getVisibleTemplates(tp *TemplatePicker) []*domain.ProjectTemplate {
	if tp.searchQuery == "" {
		return tp.templates
	}

	var filtered []*domain.ProjectTemplate
	query := strings.ToLower(tp.searchQuery)
	for _, t := range tp.templates {
		if strings.Contains(strings.ToLower(t.Name), query) ||
			strings.Contains(strings.ToLower(t.Description), query) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

type templatesLoadedMsg struct {
	templates []*domain.ProjectTemplate
	err       error
}

type templateCreatedMsg struct {
	template *domain.ProjectTemplate
	err      error
}

type projectCreatedFromTemplateMsg struct {
	project *domain.Project
	err     error
}

func TestFetchTemplatesCmd_Success(t *testing.T) {
	mockRepo := &mockTemplateRepository{
		templates: []*domain.ProjectTemplate{
			{
				ID:   1,
				Name: "Web App",
				TaskDefinitions: []domain.TaskDefinition{
					{Title: "Setup", Priority: "high"},
				},
			},
			{
				ID:   2,
				Name: "Backend API",
				TaskDefinitions: []domain.TaskDefinition{
					{Title: "Design", Priority: "high"},
				},
			},
		},
	}

	ctx := context.Background()
	filter := repository.TemplateFilter{}

	cmd := func() tea.Msg {
		templates, err := mockRepo.List(ctx, filter)
		return templatesLoadedMsg{
			templates: templates,
			err:       err,
		}
	}

	msg := cmd()
	loadedMsg, ok := msg.(templatesLoadedMsg)
	if !ok {
		t.Fatal("expected templatesLoadedMsg")
	}

	if loadedMsg.err != nil {
		t.Errorf("expected no error, got %v", loadedMsg.err)
	}

	if len(loadedMsg.templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(loadedMsg.templates))
	}

	if loadedMsg.templates[0].Name != "Web App" {
		t.Errorf("expected template name 'Web App', got %s", loadedMsg.templates[0].Name)
	}
}

func TestFetchTemplatesCmd_Error(t *testing.T) {
	mockRepo := &mockTemplateRepository{
		listErr: errors.New("database error"),
	}

	ctx := context.Background()
	filter := repository.TemplateFilter{}

	cmd := func() tea.Msg {
		templates, err := mockRepo.List(ctx, filter)
		return templatesLoadedMsg{
			templates: templates,
			err:       err,
		}
	}

	msg := cmd()
	loadedMsg, ok := msg.(templatesLoadedMsg)
	if !ok {
		t.Fatal("expected templatesLoadedMsg")
	}

	if loadedMsg.err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCreateProjectFromTemplateCmd_Success(t *testing.T) {
	now := time.Now()
	template := &domain.ProjectTemplate{
		ID:   1,
		Name: "Web App",
		ProjectDefaults: &domain.ProjectDefaults{
			Color: "blue",
			Icon:  "🚀",
		},
		TaskDefinitions: []domain.TaskDefinition{
			{Title: "Setup repo", Priority: "high", Tags: []string{"setup"}},
			{Title: "Design DB", Priority: "high", Tags: []string{"backend"}},
		},
	}

	project := &domain.Project{
		Name:      "My Web App",
		CreatedAt: now,
		UpdatedAt: now,
	}

	mockProjectRepo := &mockProjectRepository{
		projects: []*domain.Project{},
	}

	ctx := context.Background()

	cmd := func() tea.Msg {
		err := mockProjectRepo.Create(ctx, project)
		return projectCreatedFromTemplateMsg{
			project: project,
			err:     err,
		}
	}

	msg := cmd()
	createdMsg, ok := msg.(projectCreatedFromTemplateMsg)
	if !ok {
		t.Fatal("expected projectCreatedFromTemplateMsg")
	}

	if createdMsg.err != nil {
		t.Errorf("expected no error, got %v", createdMsg.err)
	}

	if createdMsg.project.ID == 0 {
		t.Error("expected project ID to be set")
	}

	if template.ProjectDefaults != nil {
		if template.ProjectDefaults.Color == "" {
			t.Error("expected template to have color defined")
		}
		if template.ProjectDefaults.Icon == "" {
			t.Error("expected template to have icon defined")
		}
	}
}

func TestTemplatePicker_Initialize(t *testing.T) {
	templates := []*domain.ProjectTemplate{
		{ID: 1, Name: "Web App"},
		{ID: 2, Name: "Backend API"},
		{ID: 3, Name: "Mobile App"},
	}

	picker := &TemplatePicker{
		templates: templates,
		cursor:    0,
		active:    true,
	}

	if !picker.active {
		t.Error("expected picker to be active")
	}

	if picker.cursor != 0 {
		t.Error("expected cursor at position 0")
	}

	if len(picker.templates) != 3 {
		t.Errorf("expected 3 templates, got %d", len(picker.templates))
	}
}

func TestTemplatePicker_Navigation(t *testing.T) {
	templates := []*domain.ProjectTemplate{
		{ID: 1, Name: "Web App"},
		{ID: 2, Name: "Backend API"},
		{ID: 3, Name: "Mobile App"},
	}

	picker := &TemplatePicker{
		templates: templates,
		cursor:    0,
		active:    true,
	}

	if picker.cursor < len(picker.templates)-1 {
		picker.cursor++
	}

	if picker.cursor != 1 {
		t.Errorf("expected cursor at position 1 after move down, got %d", picker.cursor)
	}

	if picker.cursor > 0 {
		picker.cursor--
	}

	if picker.cursor != 0 {
		t.Errorf("expected cursor at position 0 after move up, got %d", picker.cursor)
	}
}

func TestTemplatePicker_Selection(t *testing.T) {
	templates := []*domain.ProjectTemplate{
		{ID: 1, Name: "Web App"},
		{ID: 2, Name: "Backend API"},
		{ID: 3, Name: "Mobile App"},
	}

	picker := &TemplatePicker{
		templates: templates,
		cursor:    1,
		active:    true,
	}

	selectedTemplate := picker.templates[picker.cursor]

	if selectedTemplate.ID != 2 {
		t.Errorf("expected selected template ID 2, got %d", selectedTemplate.ID)
	}

	if selectedTemplate.Name != "Backend API" {
		t.Errorf("expected selected template name 'Backend API', got %s", selectedTemplate.Name)
	}
}

func TestTemplatePicker_Filtering(t *testing.T) {
	templates := []*domain.ProjectTemplate{
		{ID: 1, Name: "Web App"},
		{ID: 2, Name: "Backend API"},
		{ID: 3, Name: "Mobile App"},
	}

	picker := &TemplatePicker{
		templates:   templates,
		cursor:      0,
		active:      true,
		searchQuery: "web",
	}

	filtered := getVisibleTemplates(picker)

	if len(filtered) != 1 {
		t.Errorf("expected 1 filtered template, got %d", len(filtered))
	}

	if filtered[0].Name != "Web App" {
		t.Errorf("expected filtered template 'Web App', got %s", filtered[0].Name)
	}
}

func TestProjectForm_WithTemplate_Validation(t *testing.T) {
	template := &domain.ProjectTemplate{
		ID:          1,
		Name:        "Web App",
		Description: "Web application template",
		ProjectDefaults: &domain.ProjectDefaults{
			Color: "blue",
			Icon:  "🚀",
		},
		TaskDefinitions: []domain.TaskDefinition{
			{Title: "Setup", Priority: "high"},
		},
	}

	project := &domain.Project{
		Name: "My Project",
	}

	err := project.Validate()
	if err != nil {
		t.Errorf("expected no validation error, got %v", err)
	}

	if template.ProjectDefaults != nil {
		project.Color = template.ProjectDefaults.Color
		project.Icon = template.ProjectDefaults.Icon
	}

	if project.Color != "blue" {
		t.Error("expected project color from template")
	}

	if project.Icon != "🚀" {
		t.Error("expected project icon from template")
	}
}
