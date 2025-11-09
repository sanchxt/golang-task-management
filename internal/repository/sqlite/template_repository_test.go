package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

func setupTestTemplateDB(t *testing.T) *DB {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDB(Config{Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return db
}

func TestTemplateCreate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template := domain.NewTemplate("Web Application")
	template.Description = "Standard web app template"
	template.AddTaskDefinition(domain.TaskDefinition{
		Title:    "Setup repository",
		Priority: "high",
		Tags:     []string{"setup", "devops"},
	})

	err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	if template.ID == 0 {
		t.Error("expected ID to be set after create")
	}

	if template.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if template.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestTemplateCreateDuplicate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template1 := domain.NewTemplate("Duplicate")
	template1.AddTaskDefinition(domain.TaskDefinition{Title: "Task 1", Priority: "medium"})

	err := repo.Create(ctx, template1)
	if err != nil {
		t.Fatalf("failed to create first template: %v", err)
	}

	template2 := domain.NewTemplate("Duplicate")
	template2.AddTaskDefinition(domain.TaskDefinition{Title: "Task 2", Priority: "medium"})

	err = repo.Create(ctx, template2)
	if err == nil {
		t.Error("expected error for duplicate template name, got nil")
	}
}

func TestTemplateGetByID(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template := domain.NewTemplate("Backend Service")
	template.Description = "Microservice template"
	template.ProjectDefaults = &domain.ProjectDefaults{
		Color: "blue",
		Icon:  "🔧",
	}
	template.AddTaskDefinition(domain.TaskDefinition{
		Title:       "Setup project",
		Description: "Initialize project structure",
		Priority:    "high",
		Tags:        []string{"setup"},
	})
	template.AddTaskDefinition(domain.TaskDefinition{
		Title:    "Implement API",
		Priority: "medium",
	})

	err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, template.ID)
	if err != nil {
		t.Fatalf("failed to get template by ID: %v", err)
	}

	if retrieved.Name != template.Name {
		t.Errorf("expected name %q, got %q", template.Name, retrieved.Name)
	}

	if retrieved.Description != template.Description {
		t.Errorf("expected description %q, got %q", template.Description, retrieved.Description)
	}

	if len(retrieved.TaskDefinitions) != 2 {
		t.Errorf("expected 2 task definitions, got %d", len(retrieved.TaskDefinitions))
	}

	if retrieved.TaskDefinitions[0].Title != "Setup project" {
		t.Errorf("expected first task title 'Setup project', got %q", retrieved.TaskDefinitions[0].Title)
	}

	if retrieved.ProjectDefaults == nil {
		t.Fatal("expected project defaults to be set")
	}

	if retrieved.ProjectDefaults.Color != "blue" {
		t.Errorf("expected color 'blue', got %q", retrieved.ProjectDefaults.Color)
	}

	if retrieved.ProjectDefaults.Icon != "🔧" {
		t.Errorf("expected icon '🔧', got %q", retrieved.ProjectDefaults.Icon)
	}
}

func TestTemplateGetByIDNotFound(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent template, got nil")
	}
}

func TestTemplateGetByName(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template := domain.NewTemplate("Frontend App")
	template.AddTaskDefinition(domain.TaskDefinition{Title: "Setup", Priority: "high"})

	err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	retrieved, err := repo.GetByName(ctx, "Frontend App")
	if err != nil {
		t.Fatalf("failed to get template by name: %v", err)
	}

	if retrieved.ID != template.ID {
		t.Errorf("expected ID %d, got %d", template.ID, retrieved.ID)
	}

	if retrieved.Name != template.Name {
		t.Errorf("expected name %q, got %q", template.Name, retrieved.Name)
	}
}

func TestTemplateGetByNameNotFound(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	_, err := repo.GetByName(ctx, "NonExistent")
	if err == nil {
		t.Error("expected error for non-existent template, got nil")
	}
}

func TestTemplateUpdate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template := domain.NewTemplate("Original")
	template.AddTaskDefinition(domain.TaskDefinition{Title: "Task 1", Priority: "medium"})

	err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	originalUpdatedAt := template.UpdatedAt

	template.Name = "Updated"
	template.Description = "New description"
	template.AddTaskDefinition(domain.TaskDefinition{Title: "Task 2", Priority: "high"})
	template.ProjectDefaults = &domain.ProjectDefaults{
		Color: "green",
		Icon:  "🚀",
	}

	err = repo.Update(ctx, template)
	if err != nil {
		t.Fatalf("failed to update template: %v", err)
	}

	updated, err := repo.GetByID(ctx, template.ID)
	if err != nil {
		t.Fatalf("failed to get updated template: %v", err)
	}

	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}

	if updated.Description != "New description" {
		t.Errorf("expected description 'New description', got %q", updated.Description)
	}

	if len(updated.TaskDefinitions) != 2 {
		t.Errorf("expected 2 task definitions, got %d", len(updated.TaskDefinitions))
	}

	if updated.ProjectDefaults == nil || updated.ProjectDefaults.Color != "green" {
		t.Error("expected project defaults to be updated")
	}

	_ = originalUpdatedAt
}

func TestTemplateDelete(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template := domain.NewTemplate("ToDelete")
	template.AddTaskDefinition(domain.TaskDefinition{Title: "Task", Priority: "medium"})

	err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	err = repo.Delete(ctx, template.ID)
	if err != nil {
		t.Fatalf("failed to delete template: %v", err)
	}

	_, err = repo.GetByID(ctx, template.ID)
	if err == nil {
		t.Error("expected error for deleted template, got nil")
	}
}

func TestTemplateDeleteNotFound(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent template, got nil")
	}
}

func TestTemplateList(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	templates := []*domain.ProjectTemplate{
		{Name: "Template A", TaskDefinitions: []domain.TaskDefinition{{Title: "Task 1", Priority: "medium"}}},
		{Name: "Template B", TaskDefinitions: []domain.TaskDefinition{{Title: "Task 2", Priority: "high"}}},
		{Name: "Template C", TaskDefinitions: []domain.TaskDefinition{{Title: "Task 3", Priority: "low"}}},
	}

	for _, tmpl := range templates {
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	filter := repository.TemplateFilter{}
	list, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 templates, got %d", len(list))
	}
}

func TestTemplateListWithSearch(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	templates := []*domain.ProjectTemplate{
		{Name: "Web Application", Description: "Frontend app", TaskDefinitions: []domain.TaskDefinition{{Title: "Setup", Priority: "medium"}}},
		{Name: "Backend Service", Description: "API service", TaskDefinitions: []domain.TaskDefinition{{Title: "Setup", Priority: "medium"}}},
		{Name: "Mobile App", Description: "iOS application", TaskDefinitions: []domain.TaskDefinition{{Title: "Setup", Priority: "medium"}}},
	}

	for _, tmpl := range templates {
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	filter := repository.TemplateFilter{
		SearchQuery: "application",
	}

	list, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list templates with search: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 templates matching 'application', got %d", len(list))
	}
}

func TestTemplateListWithSorting(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	templates := []*domain.ProjectTemplate{
		{Name: "C Template", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "A Template", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "B Template", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
	}

	for _, tmpl := range templates {
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	filter := repository.TemplateFilter{
		SortBy:    "name",
		SortOrder: "asc",
	}

	list, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list templates: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(list))
	}

	if list[0].Name != "A Template" {
		t.Errorf("expected first template 'A Template', got %q", list[0].Name)
	}

	if list[2].Name != "C Template" {
		t.Errorf("expected last template 'C Template', got %q", list[2].Name)
	}
}

func TestTemplateListWithPagination(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	for i := 1; i <= 5; i++ {
		tmpl := &domain.ProjectTemplate{
			Name:            string(rune('A' + i - 1)) + " Template",
			TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}},
		}
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	filter := repository.TemplateFilter{
		SortBy:    "name",
		SortOrder: "asc",
		Limit:     2,
		Offset:    0,
	}

	page1, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get page 1: %v", err)
	}

	if len(page1) != 2 {
		t.Errorf("expected 2 templates in page 1, got %d", len(page1))
	}

	filter.Offset = 2

	page2, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to get page 2: %v", err)
	}

	if len(page2) != 2 {
		t.Errorf("expected 2 templates in page 2, got %d", len(page2))
	}

	if page1[0].ID == page2[0].ID {
		t.Error("pages should not overlap")
	}
}

func TestTemplateCount(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx, repository.TemplateFilter{})
	if err != nil {
		t.Fatalf("failed to count templates: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 templates, got %d", count)
	}

	for i := 1; i <= 3; i++ {
		tmpl := domain.NewTemplate(string(rune('A'+i-1)) + " Template")
		tmpl.AddTaskDefinition(domain.TaskDefinition{Title: "Task", Priority: "medium"})
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	count, err = repo.Count(ctx, repository.TemplateFilter{})
	if err != nil {
		t.Fatalf("failed to count templates: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 templates, got %d", count)
	}
}

func TestTemplateCountWithFilter(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	templates := []*domain.ProjectTemplate{
		{Name: "Web Application", Description: "Frontend", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "Backend API", Description: "Backend service", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "Mobile App", Description: "iOS app", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
	}

	for _, tmpl := range templates {
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	count, err := repo.Count(ctx, repository.TemplateFilter{SearchQuery: "app"})
	if err != nil {
		t.Fatalf("failed to count with filter: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 templates matching 'app', got %d", count)
	}
}

func TestTemplateSearch(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	templates := []*domain.ProjectTemplate{
		{Name: "Web Application", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "Backend Service", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "Mobile App", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
		{Name: "Desktop Application", TaskDefinitions: []domain.TaskDefinition{{Title: "Task", Priority: "medium"}}},
	}

	for _, tmpl := range templates {
		if err := repo.Create(ctx, tmpl); err != nil {
			t.Fatalf("failed to create template: %v", err)
		}
	}

	results, err := repo.Search(ctx, "application", 10)
	if err != nil {
		t.Fatalf("failed to search templates: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	results, err = repo.Search(ctx, "app", 1)
	if err != nil {
		t.Fatalf("failed to search with limit: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result with limit, got %d", len(results))
	}
}

func TestTemplateJSONSerialization(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	repo := NewTemplateRepository(db)
	ctx := context.Background()

	template := domain.NewTemplate("Complex Template")
	template.AddTaskDefinition(domain.TaskDefinition{
		Title:       "Task with tags",
		Description: "Detailed description",
		Priority:    "high",
		Tags:        []string{"tag1", "tag2", "tag3"},
	})
	template.AddTaskDefinition(domain.TaskDefinition{
		Title:    "Task without tags",
		Priority: "low",
	})

	err := repo.Create(ctx, template)
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, template.ID)
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}

	if len(retrieved.TaskDefinitions) != 2 {
		t.Errorf("expected 2 task definitions, got %d", len(retrieved.TaskDefinitions))
	}

	firstTask := retrieved.TaskDefinitions[0]
	if len(firstTask.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(firstTask.Tags))
	}

	if firstTask.Tags[1] != "tag2" {
		t.Errorf("expected tag 'tag2', got %q", firstTask.Tags[1])
	}

	secondTask := retrieved.TaskDefinitions[1]
	if len(secondTask.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(secondTask.Tags))
	}
}
