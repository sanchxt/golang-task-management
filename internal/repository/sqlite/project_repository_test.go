package sqlite

import (
	"context"
	"testing"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

func setupProjectTestDB(t *testing.T) *DB {
	db, err := NewDB(Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	return db
}

func TestProjectRepository_Create(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("create root project", func(t *testing.T) {
		project := domain.NewProject("Test Project")
		project.Description = "Test description"
		project.Color = "blue"
		project.Icon = "🔧"

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		if project.ID == 0 {
			t.Error("expected project ID to be set")
		}

		if project.Status != domain.ProjectStatusActive {
			t.Errorf("expected status 'active', got '%s'", project.Status)
		}
	})

	t.Run("create project with parent", func(t *testing.T) {
		parent := domain.NewProject("Parent Project")
		err := repo.Create(ctx, parent)
		if err != nil {
			t.Fatalf("failed to create parent project: %v", err)
		}

		child := domain.NewProject("Child Project")
		child.ParentID = &parent.ID

		err = repo.Create(ctx, child)
		if err != nil {
			t.Fatalf("failed to create child project: %v", err)
		}

		if child.ParentID == nil || *child.ParentID != parent.ID {
			t.Error("expected child to have parent ID set")
		}
	})

	t.Run("fail on duplicate name", func(t *testing.T) {
		project1 := domain.NewProject("Duplicate")
		err := repo.Create(ctx, project1)
		if err != nil {
			t.Fatalf("failed to create first project: %v", err)
		}

		project2 := domain.NewProject("Duplicate")
		err = repo.Create(ctx, project2)
		if err == nil {
			t.Error("expected error for duplicate name")
		}
	})

	t.Run("fail on empty name", func(t *testing.T) {
		project := domain.NewProject("")
		err := repo.Create(ctx, project)
		if err == nil {
			t.Error("expected error for empty name")
		}
	})
}

func TestProjectRepository_GetByID(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	project.Description = "Test description"
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("get existing project", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to get project: %v", err)
		}

		if retrieved.Name != project.Name {
			t.Errorf("expected name '%s', got '%s'", project.Name, retrieved.Name)
		}

		if retrieved.Description != project.Description {
			t.Errorf("expected description '%s', got '%s'", project.Description, retrieved.Description)
		}
	})

	t.Run("get non-existent project", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		if err == nil {
			t.Error("expected error for non-existent project")
		}
	})
}

func TestProjectRepository_GetByName(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Unique Name")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("get by name", func(t *testing.T) {
		retrieved, err := repo.GetByName(ctx, "Unique Name")
		if err != nil {
			t.Fatalf("failed to get project by name: %v", err)
		}

		if retrieved.ID != project.ID {
			t.Errorf("expected ID %d, got %d", project.ID, retrieved.ID)
		}
	})

	t.Run("get non-existent name", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "Non Existent")
		if err == nil {
			t.Error("expected error for non-existent name")
		}
	})
}

func TestProjectRepository_Update(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Original Name")
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("update project fields", func(t *testing.T) {
		project.Name = "Updated Name"
		project.Description = "Updated description"
		project.Color = "red"
		project.Icon = "🚀"
		project.IsFavorite = true

		err := repo.Update(ctx, project)
		if err != nil {
			t.Fatalf("failed to update project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to get updated project: %v", err)
		}

		if retrieved.Name != "Updated Name" {
			t.Errorf("expected name 'Updated Name', got '%s'", retrieved.Name)
		}

		if retrieved.Description != "Updated description" {
			t.Errorf("expected description 'Updated description', got '%s'", retrieved.Description)
		}

		if !retrieved.IsFavorite {
			t.Error("expected IsFavorite to be true")
		}
	})

	t.Run("update timestamp", func(t *testing.T) {
		original, _ := repo.GetByID(ctx, project.ID)
		time.Sleep(1100 * time.Millisecond)

		project.Description = "Another update"
		err := repo.Update(ctx, project)
		if err != nil {
			t.Fatalf("failed to update project: %v", err)
		}

		updated, _ := repo.GetByID(ctx, project.ID)
		if !updated.UpdatedAt.After(original.UpdatedAt) {
			t.Errorf("expected UpdatedAt (%v) to be later than (%v)", updated.UpdatedAt, original.UpdatedAt)
		}
	})
}

func TestProjectRepository_Delete(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("delete project", func(t *testing.T) {
		project := domain.NewProject("To Delete")
		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		err = repo.Delete(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to delete project: %v", err)
		}

		_, err = repo.GetByID(ctx, project.ID)
		if err == nil {
			t.Error("expected error when getting deleted project")
		}
	})

	t.Run("delete with children cascades", func(t *testing.T) {
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

		err = repo.Delete(ctx, parent.ID)
		if err != nil {
			t.Fatalf("failed to delete parent: %v", err)
		}

		_, err = repo.GetByID(ctx, child.ID)
		if err == nil {
			t.Error("expected child to be cascade deleted")
		}
	})
}

func TestProjectRepository_List(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	root1 := domain.NewProject("Root 1")
	repo.Create(ctx, root1)

	root2 := domain.NewProject("Root 2")
	root2.Status = domain.ProjectStatusArchived
	repo.Create(ctx, root2)

	child := domain.NewProject("Child 1")
	child.ParentID = &root1.ID
	child.IsFavorite = true
	repo.Create(ctx, child)

	t.Run("list all projects", func(t *testing.T) {
		projects, err := repo.List(ctx, repository.ProjectFilter{})
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}

		if len(projects) != 3 {
			t.Errorf("expected 3 projects, got %d", len(projects))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		projects, err := repo.List(ctx, repository.ProjectFilter{
			Status: domain.ProjectStatusActive,
		})
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}

		if len(projects) != 2 {
			t.Errorf("expected 2 active projects, got %d", len(projects))
		}
	})

	t.Run("exclude archived", func(t *testing.T) {
		projects, err := repo.List(ctx, repository.ProjectFilter{
			ExcludeArchived: true,
		})
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}

		if len(projects) != 2 {
			t.Errorf("expected 2 non-archived projects, got %d", len(projects))
		}
	})

	t.Run("filter by favorite", func(t *testing.T) {
		isFav := true
		projects, err := repo.List(ctx, repository.ProjectFilter{
			IsFavorite: &isFav,
		})
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}

		if len(projects) != 1 {
			t.Errorf("expected 1 favorite project, got %d", len(projects))
		}
	})

	t.Run("sort by name", func(t *testing.T) {
		projects, err := repo.List(ctx, repository.ProjectFilter{
			SortBy:    "name",
			SortOrder: "asc",
		})
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}

		if len(projects) >= 2 && projects[0].Name > projects[1].Name {
			t.Error("expected projects to be sorted by name ascending")
		}
	})
}

func TestProjectRepository_GetChildren(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	parent := domain.NewProject("Parent")
	repo.Create(ctx, parent)

	child1 := domain.NewProject("Child 1")
	child1.ParentID = &parent.ID
	repo.Create(ctx, child1)

	child2 := domain.NewProject("Child 2")
	child2.ParentID = &parent.ID
	repo.Create(ctx, child2)

	t.Run("get children", func(t *testing.T) {
		children, err := repo.GetChildren(ctx, parent.ID)
		if err != nil {
			t.Fatalf("failed to get children: %v", err)
		}

		if len(children) != 2 {
			t.Errorf("expected 2 children, got %d", len(children))
		}
	})

	t.Run("get children of project with no children", func(t *testing.T) {
		children, err := repo.GetChildren(ctx, child1.ID)
		if err != nil {
			t.Fatalf("failed to get children: %v", err)
		}

		if len(children) != 0 {
			t.Errorf("expected 0 children, got %d", len(children))
		}
	})
}

func TestProjectRepository_GetDescendants(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	grandparent := domain.NewProject("Grandparent")
	repo.Create(ctx, grandparent)

	parent := domain.NewProject("Parent")
	parent.ParentID = &grandparent.ID
	repo.Create(ctx, parent)

	child := domain.NewProject("Child")
	child.ParentID = &parent.ID
	repo.Create(ctx, child)

	t.Run("get all descendants", func(t *testing.T) {
		descendants, err := repo.GetDescendants(ctx, grandparent.ID)
		if err != nil {
			t.Fatalf("failed to get descendants: %v", err)
		}

		if len(descendants) != 2 {
			t.Errorf("expected 2 descendants (parent + child), got %d", len(descendants))
		}
	})

	t.Run("get descendants one level", func(t *testing.T) {
		descendants, err := repo.GetDescendants(ctx, parent.ID)
		if err != nil {
			t.Fatalf("failed to get descendants: %v", err)
		}

		if len(descendants) != 1 {
			t.Errorf("expected 1 descendant, got %d", len(descendants))
		}
	})
}

func TestProjectRepository_GetPath(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	root := domain.NewProject("Root")
	repo.Create(ctx, root)

	middle := domain.NewProject("Middle")
	middle.ParentID = &root.ID
	repo.Create(ctx, middle)

	leaf := domain.NewProject("Leaf")
	leaf.ParentID = &middle.ID
	repo.Create(ctx, leaf)

	t.Run("get path to leaf", func(t *testing.T) {
		path, err := repo.GetPath(ctx, leaf.ID)
		if err != nil {
			t.Fatalf("failed to get path: %v", err)
		}

		if len(path) != 3 {
			t.Errorf("expected path length 3, got %d", len(path))
		}

		if path[0].Name != "Root" {
			t.Errorf("expected first element 'Root', got '%s'", path[0].Name)
		}

		if path[len(path)-1].Name != "Leaf" {
			t.Errorf("expected last element 'Leaf', got '%s'", path[len(path)-1].Name)
		}
	})

	t.Run("get path to root", func(t *testing.T) {
		path, err := repo.GetPath(ctx, root.ID)
		if err != nil {
			t.Fatalf("failed to get path: %v", err)
		}

		if len(path) != 1 {
			t.Errorf("expected path length 1, got %d", len(path))
		}
	})
}

func TestProjectRepository_ValidateHierarchy(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	parent := domain.NewProject("Parent")
	repo.Create(ctx, parent)

	child := domain.NewProject("Child")
	child.ParentID = &parent.ID
	repo.Create(ctx, child)

	t.Run("prevent self-reference", func(t *testing.T) {
		err := repo.ValidateHierarchy(ctx, parent.ID, parent.ID)
		if err == nil {
			t.Error("expected error for self-reference")
		}
	})

	t.Run("prevent cycle", func(t *testing.T) {
		err := repo.ValidateHierarchy(ctx, parent.ID, child.ID)
		if err == nil {
			t.Error("expected error for cycle")
		}
	})

	t.Run("allow valid parent", func(t *testing.T) {
		newChild := domain.NewProject("New Child")
		repo.Create(ctx, newChild)

		err := repo.ValidateHierarchy(ctx, newChild.ID, parent.ID)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
}

func TestProjectRepository_Archive(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("To Archive")
	repo.Create(ctx, project)

	t.Run("archive project", func(t *testing.T) {
		err := repo.Archive(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to archive project: %v", err)
		}

		retrieved, _ := repo.GetByID(ctx, project.ID)
		if retrieved.Status != domain.ProjectStatusArchived {
			t.Errorf("expected status 'archived', got '%s'", retrieved.Status)
		}
	})

	t.Run("unarchive project", func(t *testing.T) {
		err := repo.Unarchive(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to unarchive project: %v", err)
		}

		retrieved, _ := repo.GetByID(ctx, project.ID)
		if retrieved.Status != domain.ProjectStatusActive {
			t.Errorf("expected status 'active', got '%s'", retrieved.Status)
		}
	})
}

func TestProjectRepository_Favorite(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	repo.Create(ctx, project)

	t.Run("set favorite", func(t *testing.T) {
		err := repo.SetFavorite(ctx, project.ID, true)
		if err != nil {
			t.Fatalf("failed to set favorite: %v", err)
		}

		retrieved, _ := repo.GetByID(ctx, project.ID)
		if !retrieved.IsFavorite {
			t.Error("expected IsFavorite to be true")
		}
	})

	t.Run("unset favorite", func(t *testing.T) {
		err := repo.SetFavorite(ctx, project.ID, false)
		if err != nil {
			t.Fatalf("failed to unset favorite: %v", err)
		}

		retrieved, _ := repo.GetByID(ctx, project.ID)
		if retrieved.IsFavorite {
			t.Error("expected IsFavorite to be false")
		}
	})

	t.Run("get favorites", func(t *testing.T) {
		repo.SetFavorite(ctx, project.ID, true)

		favorites, err := repo.GetFavorites(ctx)
		if err != nil {
			t.Fatalf("failed to get favorites: %v", err)
		}

		if len(favorites) == 0 {
			t.Error("expected at least one favorite")
		}
	})
}

func TestProjectRepository_Count(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		project := domain.NewProject("Project " + string(rune('A'+i)))
		if i%2 == 0 {
			project.Status = domain.ProjectStatusArchived
		}
		repo.Create(ctx, project)
	}

	t.Run("count all", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.ProjectFilter{})
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}

		if count != 5 {
			t.Errorf("expected count 5, got %d", count)
		}
	})

	t.Run("count active", func(t *testing.T) {
		count, err := repo.Count(ctx, repository.ProjectFilter{
			Status: domain.ProjectStatusActive,
		})
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}

		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
	})
}

func TestProjectRepository_Aliases(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("create project with aliases", func(t *testing.T) {
		project := domain.NewProject("Backend API")
		project.Aliases = []string{"be", "back", "api"}

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project with aliases: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if len(retrieved.Aliases) != 3 {
			t.Errorf("expected 3 aliases, got %d", len(retrieved.Aliases))
		}

		if retrieved.Aliases[0] != "be" || retrieved.Aliases[1] != "back" || retrieved.Aliases[2] != "api" {
			t.Errorf("aliases don't match: %v", retrieved.Aliases)
		}
	})

	t.Run("create project with empty aliases", func(t *testing.T) {
		project := domain.NewProject("Frontend")
		project.Aliases = []string{}

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if len(retrieved.Aliases) != 0 {
			t.Errorf("expected 0 aliases, got %d", len(retrieved.Aliases))
		}
	})

	t.Run("update project aliases", func(t *testing.T) {
		project := domain.NewProject("DevOps")
		project.Aliases = []string{"ops"}

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		project.Aliases = []string{"ops", "infra", "devops"}
		err = repo.Update(ctx, project)
		if err != nil {
			t.Fatalf("failed to update project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if len(retrieved.Aliases) != 3 {
			t.Errorf("expected 3 aliases, got %d", len(retrieved.Aliases))
		}
	})

	t.Run("fail on duplicate alias in same project", func(t *testing.T) {
		project := domain.NewProject("Duplicate Alias Test")
		project.Aliases = []string{"test", "test"}

		err := repo.Create(ctx, project)
		if err == nil {
			t.Error("expected error for duplicate alias in same project")
		}
	})

	t.Run("fail on duplicate alias across projects", func(t *testing.T) {
		project1 := domain.NewProject("Project 1")
		project1.Aliases = []string{"shared"}
		err := repo.Create(ctx, project1)
		if err != nil {
			t.Fatalf("failed to create project1: %v", err)
		}

		project2 := domain.NewProject("Project 2")
		project2.Aliases = []string{"shared"}
		err = repo.Create(ctx, project2)
		if err == nil {
			t.Error("expected error for duplicate alias across projects")
		}
	})
}

func TestProjectRepository_GetByAlias(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Test Project")
	project.Aliases = []string{"test", "proj", "tp"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("find by exact alias", func(t *testing.T) {
		found, err := repo.GetByAlias(ctx, "test")
		if err != nil {
			t.Fatalf("failed to get by alias: %v", err)
		}

		if found.ID != project.ID {
			t.Errorf("expected project ID %d, got %d", project.ID, found.ID)
		}

		if found.Name != "Test Project" {
			t.Errorf("expected name 'Test Project', got '%s'", found.Name)
		}
	})

	t.Run("find by alias case-insensitive", func(t *testing.T) {
		found, err := repo.GetByAlias(ctx, "TEST")
		if err != nil {
			t.Fatalf("failed to get by alias: %v", err)
		}

		if found.ID != project.ID {
			t.Errorf("expected project ID %d, got %d", project.ID, found.ID)
		}
	})

	t.Run("find by another alias", func(t *testing.T) {
		found, err := repo.GetByAlias(ctx, "proj")
		if err != nil {
			t.Fatalf("failed to get by alias: %v", err)
		}

		if found.ID != project.ID {
			t.Errorf("expected project ID %d, got %d", project.ID, found.ID)
		}
	})

	t.Run("not found with invalid alias", func(t *testing.T) {
		_, err := repo.GetByAlias(ctx, "nonexistent")
		if err == nil {
			t.Error("expected error for non-existent alias")
		}
	})
}

func TestProjectRepository_ValidateAliasUniqueness(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	project := domain.NewProject("Existing Project")
	project.Aliases = []string{"existing"}
	err := repo.Create(ctx, project)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	t.Run("validate new unique alias", func(t *testing.T) {
		err := repo.ValidateAliasUniqueness(ctx, "newunique", nil)
		if err != nil {
			t.Errorf("expected no error for unique alias, got: %v", err)
		}
	})

	t.Run("fail validation for existing alias", func(t *testing.T) {
		err := repo.ValidateAliasUniqueness(ctx, "existing", nil)
		if err == nil {
			t.Error("expected error for existing alias")
		}
	})

	t.Run("fail validation case-insensitive", func(t *testing.T) {
		err := repo.ValidateAliasUniqueness(ctx, "EXISTING", nil)
		if err == nil {
			t.Error("expected error for existing alias (case-insensitive)")
		}
	})

	t.Run("allow alias for same project when updating", func(t *testing.T) {
		err := repo.ValidateAliasUniqueness(ctx, "existing", &project.ID)
		if err != nil {
			t.Errorf("expected no error when validating own alias, got: %v", err)
		}
	})

	t.Run("fail validation for other project's alias even when excluding current", func(t *testing.T) {
		otherID := int64(999)
		err := repo.ValidateAliasUniqueness(ctx, "existing", &otherID)
		if err == nil {
			t.Error("expected error for another project's alias")
		}
	})
}

func TestProjectRepository_Notes(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("create project with notes", func(t *testing.T) {
		project := domain.NewProject("Project with Notes")
		project.Notes = "# Project Notes\n\nThis is a test note with **markdown**."

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project with notes: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if retrieved.Notes != project.Notes {
			t.Errorf("notes don't match.\nExpected: %s\nGot: %s", project.Notes, retrieved.Notes)
		}
	})

	t.Run("create project with empty notes", func(t *testing.T) {
		project := domain.NewProject("Project without Notes")
		project.Notes = ""

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if retrieved.Notes != "" {
			t.Errorf("expected empty notes, got: %s", retrieved.Notes)
		}
	})

	t.Run("update project notes", func(t *testing.T) {
		project := domain.NewProject("Update Notes Test")
		project.Notes = "Initial notes"

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		project.Notes = "Updated notes with more content"
		err = repo.Update(ctx, project)
		if err != nil {
			t.Fatalf("failed to update project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if retrieved.Notes != "Updated notes with more content" {
			t.Errorf("notes not updated correctly, got: %s", retrieved.Notes)
		}
	})

	t.Run("clear project notes", func(t *testing.T) {
		project := domain.NewProject("Clear Notes Test")
		project.Notes = "Some notes to be cleared"

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		project.Notes = ""
		err = repo.Update(ctx, project)
		if err != nil {
			t.Fatalf("failed to update project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if retrieved.Notes != "" {
			t.Errorf("expected empty notes, got: %s", retrieved.Notes)
		}
	})
}

func TestProjectRepository_AliasesAndNotesTogether(t *testing.T) {
	db := setupProjectTestDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("create project with both aliases and notes", func(t *testing.T) {
		project := domain.NewProject("Full Featured Project")
		project.Aliases = []string{"full", "featured"}
		project.Notes = "# Full Project\n\nWith aliases and notes"
		project.Description = "A complete project"
		project.Color = "blue"

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("failed to retrieve project: %v", err)
		}

		if len(retrieved.Aliases) != 2 {
			t.Errorf("expected 2 aliases, got %d", len(retrieved.Aliases))
		}

		if retrieved.Notes != project.Notes {
			t.Errorf("notes don't match")
		}

		if retrieved.Description != project.Description {
			t.Errorf("description doesn't match")
		}

		if retrieved.Color != project.Color {
			t.Errorf("color doesn't match")
		}
	})

	t.Run("retrieve by alias includes all fields including notes", func(t *testing.T) {
		project := domain.NewProject("Alias Retrieval Test")
		project.Aliases = []string{"retrieve"}
		project.Notes = "Test notes for alias retrieval"

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		retrieved, err := repo.GetByAlias(ctx, "retrieve")
		if err != nil {
			t.Fatalf("failed to get by alias: %v", err)
		}

		if len(retrieved.Aliases) != 1 {
			t.Errorf("expected 1 alias, got %d", len(retrieved.Aliases))
		}

		if retrieved.Notes != project.Notes {
			t.Errorf("notes don't match when retrieved by alias")
		}
	})

	t.Run("list projects includes aliases and notes", func(t *testing.T) {
		project := domain.NewProject("List Test Project")
		project.Aliases = []string{"list", "test"}
		project.Notes = "Notes for list test"

		err := repo.Create(ctx, project)
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		projects, err := repo.List(ctx, repository.ProjectFilter{})
		if err != nil {
			t.Fatalf("failed to list projects: %v", err)
		}

		var found *domain.Project
		for _, p := range projects {
			if p.ID == project.ID {
				found = p
				break
			}
		}

		if found == nil {
			t.Fatal("project not found in list")
		}

		if len(found.Aliases) != 2 {
			t.Errorf("expected 2 aliases, got %d", len(found.Aliases))
		}

		if found.Notes != project.Notes {
			t.Errorf("notes don't match in list")
		}
	})
}
