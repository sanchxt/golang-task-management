package cli

import (
	"context"
	"testing"

	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
)

func TestProjectAlias_AddAlias(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Backend Service")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Aliases = []string{"api-service"}
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to add alias: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if len(updated.Aliases) != 1 {
		t.Errorf("expected 1 alias, got %d", len(updated.Aliases))
	}

	if updated.Aliases[0] != "api-service" {
		t.Errorf("expected alias 'api-service', got '%s'", updated.Aliases[0])
	}
}

func TestProjectAlias_DuplicateOnProject(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Backend Service")
	project.Aliases = []string{"api-service"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Aliases = []string{"api-service", "api-service"}
	err = project.Validate()
	if err == nil {
		t.Error("expected validation error for duplicate alias, got nil")
	}
}

func TestProjectAlias_GlobalUniqueness(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project1 := domain.NewProject("Backend Service")
	project1.Aliases = []string{"api-service"}
	err := repo.Create(ctx, project1)
	if err != nil {
		t.Fatalf("failed to create first project: %v", err)
	}

	project2 := domain.NewProject("Other Service")
	err = repo.Create(ctx, project2)
	if err != nil {
		t.Fatalf("failed to create second project: %v", err)
	}

	project2.Aliases = []string{"api-service"}
	err = repo.Update(ctx, project2)
	if err == nil {
		t.Error("expected error for global alias conflict, got nil")
	}
}

func TestProjectAlias_RetrieveByAlias(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Backend Service")
	project.Aliases = []string{"api-service", "backend-api"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	found, err := repo.GetByAlias(ctx, "api-service")
	if err != nil {
		t.Fatalf("failed to find by alias: %v", err)
	}

	if found.ID != project.ID {
		t.Errorf("expected project ID %d, got %d", project.ID, found.ID)
	}

	found, err = repo.GetByAlias(ctx, "backend-api")
	if err != nil {
		t.Fatalf("failed to find by second alias: %v", err)
	}

	if found.ID != project.ID {
		t.Errorf("expected project ID %d, got %d", project.ID, found.ID)
	}
}

func TestProjectAlias_CaseInsensitiveLookup(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Backend Service")
	project.Aliases = []string{"api-service"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	testCases := []string{"api-service", "API-SERVICE", "Api-Service", "API-service"}
	for _, testAlias := range testCases {
		found, err := repo.GetByAlias(ctx, testAlias)
		if err != nil {
			t.Fatalf("failed to find by alias '%s': %v", testAlias, err)
		}

		if found.ID != project.ID {
			t.Errorf("expected project ID %d for alias '%s', got %d", project.ID, testAlias, found.ID)
		}
	}
}

func TestProjectAlias_InvalidFormat(t *testing.T) {
	testCases := []struct {
		name  string
		alias string
	}{
		{"uppercase", "API-Service"},
		{"space", "api service"},
		{"special", "api@service"},
		{"too_short", "a"},
		{"too_long", "a-very-long-alias-that-exceeds-the-maximum-character-limit"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.IsValidAliasFormat(tc.alias)
			if err == nil {
				t.Errorf("expected validation error for alias '%s', got nil", tc.alias)
			}
		})
	}
}

func TestProjectAlias_ValidFormats(t *testing.T) {
	testCases := []string{
		"api-service",
		"api_service",
		"backend-v2",
		"api-v1-2-3",
		"a1",
		"api123",
		"123api",
	}

	for _, alias := range testCases {
		t.Run(alias, func(t *testing.T) {
			err := domain.IsValidAliasFormat(alias)
			if err != nil {
				t.Errorf("expected no error for alias '%s', got: %v", alias, err)
			}
		})
	}
}

func TestProjectAlias_MultipleAliases(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("API Service")
	project.Aliases = []string{"api-v1", "rest-api", "api-service"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if len(updated.Aliases) != 3 {
		t.Errorf("expected 3 aliases, got %d", len(updated.Aliases))
	}

	for _, alias := range project.Aliases {
		found, err := repo.GetByAlias(ctx, alias)
		if err != nil {
			t.Fatalf("failed to find by alias '%s': %v", alias, err)
		}

		if found.ID != project.ID {
			t.Errorf("expected project ID %d for alias '%s', got %d", project.ID, alias, found.ID)
		}
	}
}

func TestProjectAlias_RemoveAlias(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("API Service")
	project.Aliases = []string{"api-v1", "rest-api", "api-service"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Aliases = []string{"api-v1", "rest-api"}
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if len(updated.Aliases) != 2 {
		t.Errorf("expected 2 aliases, got %d", len(updated.Aliases))
	}

	_, err = repo.GetByAlias(ctx, "api-service")
	if err == nil {
		t.Error("expected error when finding removed alias, got nil")
	}
}

func TestProjectAlias_AliasValidation_MaxCount(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("API Service")

	project.Aliases = make([]string, 10)
	for i := 0; i < 10; i++ {
		project.Aliases[i] = "alias-" + string(rune(i+48))
	}

	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project with 10 aliases: %v", err)
	}

	project.Aliases = append(project.Aliases, "alias-11")
	err = project.Validate()
	if err == nil {
		t.Error("expected error for exceeding max aliases, got nil")
	}
}

func TestProjectAlias_UpdatePreservesOtherFields(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Backend Service")
	project.Description = "Main backend API"
	project.Color = "blue"
	project.Icon = "🚀"
	project.IsFavorite = true

	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	originalID := project.ID

	project.Aliases = []string{"api-service"}
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to add alias: %v", err)
	}

	updated, err := repo.GetByID(ctx, originalID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.Description != "Main backend API" {
		t.Errorf("description changed unexpectedly: %s", updated.Description)
	}

	if updated.Color != "blue" {
		t.Errorf("color changed unexpectedly: %s", updated.Color)
	}

	if updated.Icon != "🚀" {
		t.Errorf("icon changed unexpectedly: %s", updated.Icon)
	}

	if !updated.IsFavorite {
		t.Error("favorite status changed unexpectedly")
	}
}
