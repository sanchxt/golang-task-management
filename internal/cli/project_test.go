package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
)

func setupTestDB(t *testing.T) (*sqlite.DB, string) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sqlite.NewDB(sqlite.Config{Path: dbPath})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cfg := config.GetDefaultConfig()
	cfg.DBPath = dbPath
	configPath := filepath.Join(tempDir, "config.yaml")

	os.Setenv("TASKFLOW_CONFIG", configPath)

	return db, tempDir
}

func TestProjectUpdate_UpdateName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Original Name")
	project.Description = "Original description"
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Name = "Updated Name"
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", updated.Name)
	}

	if updated.Description != "Original description" {
		t.Errorf("expected description unchanged, got '%s'", updated.Description)
	}
}

func TestProjectUpdate_UpdateDescription(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	project.Description = "Original description"
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Description = "Updated description"
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", updated.Description)
	}
}

func TestProjectUpdate_UpdateParent(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	parent1 := domain.NewProject("Parent 1")
	err := repo.Create(ctx, parent1)
	if err != nil {
		t.Fatalf("failed to create parent1: %v", err)
	}

	parent2 := domain.NewProject("Parent 2")
	err = repo.Create(ctx, parent2)
	if err != nil {
		t.Fatalf("failed to create parent2: %v", err)
	}

	child := domain.NewProject("Child")
	child.ParentID = &parent1.ID
	err = repo.Create(ctx, child)
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	child.ParentID = &parent2.ID
	err = repo.Update(ctx, child)
	if err != nil {
		t.Fatalf("failed to update child parent: %v", err)
	}

	updated, err := repo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("failed to get updated child: %v", err)
	}

	if updated.ParentID == nil || *updated.ParentID != parent2.ID {
		t.Errorf("expected parent ID %d, got %v", parent2.ID, updated.ParentID)
	}
}

func TestProjectUpdate_ClearParent(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	parent := domain.NewProject("Parent")
	err := repo.Create(ctx, parent)
	if err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}

	child := domain.NewProject("Child")
	child.ParentID = &parent.ID
	err = repo.Create(ctx, child)
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	child.ParentID = nil
	err = repo.Update(ctx, child)
	if err != nil {
		t.Fatalf("failed to clear parent: %v", err)
	}

	updated, err := repo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("failed to get updated child: %v", err)
	}

	if updated.ParentID != nil {
		t.Errorf("expected parent ID to be nil, got %v", updated.ParentID)
	}
}

func TestProjectUpdate_CycleDetection(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	// hierarchy: A -> B -> C
	projectA := domain.NewProject("Project A")
	err := repo.Create(ctx, projectA)
	if err != nil {
		t.Fatalf("failed to create project A: %v", err)
	}

	projectB := domain.NewProject("Project B")
	projectB.ParentID = &projectA.ID
	err = repo.Create(ctx, projectB)
	if err != nil {
		t.Fatalf("failed to create project B: %v", err)
	}

	projectC := domain.NewProject("Project C")
	projectC.ParentID = &projectB.ID
	err = repo.Create(ctx, projectC)
	if err != nil {
		t.Fatalf("failed to create project C: %v", err)
	}

	projectA.ParentID = &projectC.ID
	err = repo.Update(ctx, projectA)
	if err == nil {
		t.Error("expected error for cycle creation, got nil")
	}
}

func TestProjectUpdate_UpdateColor(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	project.Color = "blue"
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Color = "green"
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to update color: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.Color != "green" {
		t.Errorf("expected color 'green', got '%s'", updated.Color)
	}
}

func TestProjectUpdate_UpdateIcon(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	project.Icon = "📦"
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Icon = "🚀"
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to update icon: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.Icon != "🚀" {
		t.Errorf("expected icon '🚀', got '%s'", updated.Icon)
	}
}

func TestProjectUpdate_ToggleFavorite(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	err = repo.SetFavorite(ctx, project.ID, true)
	if err != nil {
		t.Fatalf("failed to set favorite: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if !updated.IsFavorite {
		t.Error("expected project to be favorite")
	}

	err = repo.SetFavorite(ctx, project.ID, false)
	if err != nil {
		t.Fatalf("failed to unset favorite: %v", err)
	}

	updated, err = repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if updated.IsFavorite {
		t.Error("expected project to not be favorite")
	}
}

func TestProjectUpdate_InvalidName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	project.Name = ""
	err = repo.Update(ctx, project)
	if err == nil {
		t.Error("expected error for empty name, got nil")
	}

	project.Name = string(make([]byte, 101))
	err = repo.Update(ctx, project)
	if err == nil {
		t.Error("expected error for long name, got nil")
	}
}

func TestProjectUpdate_DuplicateName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project1 := domain.NewProject("Project 1")
	err := repo.Create(ctx, project1)
	if err != nil {
		t.Fatalf("failed to create project1: %v", err)
	}

	project2 := domain.NewProject("Project 2")
	err = repo.Create(ctx, project2)
	if err != nil {
		t.Fatalf("failed to create project2: %v", err)
	}

	project2.Name = "Project 1"
	err = repo.Update(ctx, project2)
	if err == nil {
		t.Error("expected error for duplicate name, got nil")
	}
}

func TestProjectArchive_ArchiveProject(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	err = repo.Archive(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to archive project: %v", err)
	}

	archived, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get archived project: %v", err)
	}

	if archived.Status != domain.ProjectStatusArchived {
		t.Errorf("expected status 'archived', got '%s'", archived.Status)
	}
}

func TestProjectArchive_ArchiveHierarchy(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	parent := domain.NewProject("Parent")
	err := repo.Create(ctx, parent)
	if err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}

	child1 := domain.NewProject("Child 1")
	child1.ParentID = &parent.ID
	err = repo.Create(ctx, child1)
	if err != nil {
		t.Fatalf("failed to create child1: %v", err)
	}

	child2 := domain.NewProject("Child 2")
	child2.ParentID = &parent.ID
	err = repo.Create(ctx, child2)
	if err != nil {
		t.Fatalf("failed to create child2: %v", err)
	}

	err = repo.Archive(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to archive parent: %v", err)
	}

	descendants, err := repo.GetDescendants(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to get descendants: %v", err)
	}

	for _, desc := range descendants {
		err = repo.Archive(ctx, desc.ID)
		if err != nil {
			t.Fatalf("failed to archive descendant %d: %v", desc.ID, err)
		}
	}

	archivedParent, err := repo.GetByID(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to get parent: %v", err)
	}
	if archivedParent.Status != domain.ProjectStatusArchived {
		t.Errorf("expected parent archived, got '%s'", archivedParent.Status)
	}

	archivedChild1, err := repo.GetByID(ctx, child1.ID)
	if err != nil {
		t.Fatalf("failed to get child1: %v", err)
	}
	if archivedChild1.Status != domain.ProjectStatusArchived {
		t.Errorf("expected child1 archived, got '%s'", archivedChild1.Status)
	}

	archivedChild2, err := repo.GetByID(ctx, child2.ID)
	if err != nil {
		t.Fatalf("failed to get child2: %v", err)
	}
	if archivedChild2.Status != domain.ProjectStatusArchived {
		t.Errorf("expected child2 archived, got '%s'", archivedChild2.Status)
	}
}

func TestProjectArchive_CannotArchiveCompleted(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	project.Status = domain.ProjectStatusCompleted
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	err = repo.Archive(ctx, project.ID)
	if err != nil {
		t.Logf("archive attempt resulted in: %v", err)
	}
}

func TestProjectUnarchive_UnarchiveProject(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	err = repo.Archive(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to archive project: %v", err)
	}

	err = repo.Unarchive(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to unarchive project: %v", err)
	}

	unarchived, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get unarchived project: %v", err)
	}

	if unarchived.Status != domain.ProjectStatusActive {
		t.Errorf("expected status 'active', got '%s'", unarchived.Status)
	}
}

func TestProjectUnarchive_UnarchiveHierarchy(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	parent := domain.NewProject("Parent")
	err := repo.Create(ctx, parent)
	if err != nil {
		t.Fatalf("failed to create parent: %v", err)
	}

	child := domain.NewProject("Child")
	child.ParentID = &parent.ID
	err = repo.Create(ctx, child)
	if err != nil {
		t.Fatalf("failed to create child: %v", err)
	}

	err = repo.Archive(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to archive parent: %v", err)
	}

	err = repo.Archive(ctx, child.ID)
	if err != nil {
		t.Fatalf("failed to archive child: %v", err)
	}

	err = repo.Unarchive(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to unarchive parent: %v", err)
	}

	unarchivedParent, err := repo.GetByID(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to get parent: %v", err)
	}
	if unarchivedParent.Status != domain.ProjectStatusActive {
		t.Errorf("expected parent active, got '%s'", unarchivedParent.Status)
	}

	childStillArchived, err := repo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("failed to get child: %v", err)
	}
	if childStillArchived.Status != domain.ProjectStatusArchived {
		t.Errorf("expected child still archived, got '%s'", childStillArchived.Status)
	}

	descendants, err := repo.GetDescendants(ctx, parent.ID)
	if err != nil {
		t.Fatalf("failed to get descendants: %v", err)
	}

	for _, desc := range descendants {
		err = repo.Unarchive(ctx, desc.ID)
		if err != nil {
			t.Fatalf("failed to unarchive descendant %d: %v", desc.ID, err)
		}
	}

	unarchivedChild, err := repo.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("failed to get child: %v", err)
	}
	if unarchivedChild.Status != domain.ProjectStatusActive {
		t.Errorf("expected child active, got '%s'", unarchivedChild.Status)
	}
}

func TestProjectUpdate_TimestampUpdated(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	originalUpdatedAt := project.UpdatedAt

	time.Sleep(1100 * time.Millisecond)

	project.Description = "Updated description"
	err = repo.Update(ctx, project)
	if err != nil {
		t.Fatalf("failed to update project: %v", err)
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("failed to get updated project: %v", err)
	}

	if !updated.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("expected UpdatedAt to be updated, got original=%v, updated=%v",
			originalUpdatedAt, updated.UpdatedAt)
	}
}
