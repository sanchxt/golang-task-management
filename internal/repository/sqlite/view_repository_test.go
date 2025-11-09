package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

func setupViewRepo(t *testing.T) *ViewRepository {
	tempFile, err := os.CreateTemp("", "test_views_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tempFile.Close()

	db, err := NewDB(Config{Path: tempFile.Name()})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		os.Remove(tempFile.Name())
	})

	return NewViewRepository(db)
}

func TestViewRepositoryCreate_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Test View")
	view.Description = "A test view"
	view.IsFavorite = true

	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	if view.ID == 0 {
		t.Error("expected view ID to be set")
	}

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.Name != "Test View" {
		t.Errorf("expected name 'Test View', got %q", retrieved.Name)
	}
	if retrieved.Description != "A test view" {
		t.Errorf("expected description 'A test view', got %q", retrieved.Description)
	}
	if !retrieved.IsFavorite {
		t.Error("expected IsFavorite to be true")
	}
}

func TestViewRepositoryCreate_DuplicateName(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view1 := domain.NewSavedView("Duplicate")
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create first view: %v", err)
	}

	view2 := domain.NewSavedView("Duplicate")
	err = repo.Create(ctx, view2)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestViewRepositoryCreate_WithHotKey(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Hotkey View")
	hotKey := 5
	view.HotKey = &hotKey

	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view with hot key: %v", err)
	}

	retrieved, err := repo.GetByHotKey(ctx, 5)
	if err != nil {
		t.Fatalf("failed to get view by hot key: %v", err)
	}

	if retrieved.Name != "Hotkey View" {
		t.Errorf("expected 'Hotkey View', got %q", retrieved.Name)
	}
}

func TestViewRepositoryCreate_DuplicateHotKey(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	hotKey := 3
	view1 := domain.NewSavedView("View 1")
	view1.HotKey = &hotKey
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create first view: %v", err)
	}

	view2 := domain.NewSavedView("View 2")
	view2.HotKey = &hotKey
	err = repo.Create(ctx, view2)
	if err == nil {
		t.Error("expected error for duplicate hot key")
	}
}

func TestViewRepositoryGetByID_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Get By ID Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to get view: %v", err)
	}

	if retrieved.Name != "Get By ID Test" {
		t.Errorf("expected 'Get By ID Test', got %q", retrieved.Name)
	}
}

func TestViewRepositoryGetByID_NotFound(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	if err == nil {
		t.Error("expected error for non-existent view")
	}
}

func TestViewRepositoryGetByName_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Get By Name Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	retrieved, err := repo.GetByName(ctx, "Get By Name Test")
	if err != nil {
		t.Fatalf("failed to get view by name: %v", err)
	}

	if retrieved.ID != view.ID {
		t.Errorf("expected ID %d, got %d", view.ID, retrieved.ID)
	}
}

func TestViewRepositoryUpdate_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Update Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	view.Description = "Updated description"
	view.IsFavorite = true
	err = repo.Update(ctx, view)
	if err != nil {
		t.Fatalf("failed to update view: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve updated view: %v", err)
	}

	if retrieved.Description != "Updated description" {
		t.Errorf("expected 'Updated description', got %q", retrieved.Description)
	}
	if !retrieved.IsFavorite {
		t.Error("expected IsFavorite to be true")
	}
}

func TestViewRepositoryDelete_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Delete Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	err = repo.Delete(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to delete view: %v", err)
	}

	_, err = repo.GetByID(ctx, view.ID)
	if err == nil {
		t.Error("expected error when retrieving deleted view")
	}
}

func TestViewRepositoryList_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		view := domain.NewSavedView("View " + string(rune(i)))
		err := repo.Create(ctx, view)
		if err != nil {
			t.Fatalf("failed to create view %d: %v", i, err)
		}
	}

	views, err := repo.List(ctx, repository.ViewFilter{})
	if err != nil {
		t.Fatalf("failed to list views: %v", err)
	}

	if len(views) != 3 {
		t.Errorf("expected 3 views, got %d", len(views))
	}
}

func TestViewRepositoryList_WithFavoriteFilter(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view1 := domain.NewSavedView("Favorite 1")
	view1.IsFavorite = true
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create favorite view: %v", err)
	}

	view2 := domain.NewSavedView("Not Favorite")
	err = repo.Create(ctx, view2)
	if err != nil {
		t.Fatalf("failed to create non-favorite view: %v", err)
	}

	isFav := true
	views, err := repo.List(ctx, repository.ViewFilter{IsFavorite: &isFav})
	if err != nil {
		t.Fatalf("failed to list favorite views: %v", err)
	}

	if len(views) != 1 {
		t.Errorf("expected 1 favorite view, got %d", len(views))
	}
	if views[0].Name != "Favorite 1" {
		t.Errorf("expected 'Favorite 1', got %q", views[0].Name)
	}
}

func TestViewRepositoryList_WithHotKeyFilter(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	hotKey := 1
	view1 := domain.NewSavedView("With Hot Key")
	view1.HotKey = &hotKey
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create view with hot key: %v", err)
	}

	view2 := domain.NewSavedView("Without Hot Key")
	err = repo.Create(ctx, view2)
	if err != nil {
		t.Fatalf("failed to create view without hot key: %v", err)
	}

	views, err := repo.List(ctx, repository.ViewFilter{HasHotKey: true})
	if err != nil {
		t.Fatalf("failed to list views with hot keys: %v", err)
	}

	if len(views) != 1 {
		t.Errorf("expected 1 view with hot key, got %d", len(views))
	}
	if views[0].Name != "With Hot Key" {
		t.Errorf("expected 'With Hot Key', got %q", views[0].Name)
	}
}

func TestViewRepositoryCount_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		view := domain.NewSavedView("View " + string(rune(i)))
		err := repo.Create(ctx, view)
		if err != nil {
			t.Fatalf("failed to create view: %v", err)
		}
	}

	count, err := repo.Count(ctx, repository.ViewFilter{})
	if err != nil {
		t.Fatalf("failed to count views: %v", err)
	}

	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestViewRepositorySearch_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view1 := domain.NewSavedView("Backend API")
	view1.Description = "For backend operations"
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create view 1: %v", err)
	}

	view2 := domain.NewSavedView("Frontend Tasks")
	err = repo.Create(ctx, view2)
	if err != nil {
		t.Fatalf("failed to create view 2: %v", err)
	}

	results, err := repo.Search(ctx, "Backend", 10)
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "Backend API" {
		t.Errorf("expected 'Backend API', got %q", results[0].Name)
	}
}

func TestViewRepositoryGetByHotKey_NotFound(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	_, err := repo.GetByHotKey(ctx, 5)
	if err == nil {
		t.Error("expected error for non-existent hot key")
	}
}

func TestViewRepositorySetHotKey_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Set Hot Key Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	hotKey := 7
	err = repo.SetHotKey(ctx, view.ID, &hotKey)
	if err != nil {
		t.Fatalf("failed to set hot key: %v", err)
	}

	retrieved, err := repo.GetByHotKey(ctx, 7)
	if err != nil {
		t.Fatalf("failed to get view by hot key: %v", err)
	}

	if retrieved.Name != "Set Hot Key Test" {
		t.Errorf("expected 'Set Hot Key Test', got %q", retrieved.Name)
	}
}

func TestViewRepositorySetHotKey_Clear(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	hotKey := 2
	view := domain.NewSavedView("Clear Hot Key Test")
	view.HotKey = &hotKey
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	err = repo.SetHotKey(ctx, view.ID, nil)
	if err != nil {
		t.Fatalf("failed to clear hot key: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.HotKey != nil {
		t.Errorf("expected HotKey to be nil, got %v", *retrieved.HotKey)
	}
}

func TestViewRepositoryGetFavorites_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view1 := domain.NewSavedView("Favorite 1")
	view1.IsFavorite = true
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create favorite 1: %v", err)
	}

	view2 := domain.NewSavedView("Favorite 2")
	view2.IsFavorite = true
	err = repo.Create(ctx, view2)
	if err != nil {
		t.Fatalf("failed to create favorite 2: %v", err)
	}

	view3 := domain.NewSavedView("Not Favorite")
	err = repo.Create(ctx, view3)
	if err != nil {
		t.Fatalf("failed to create non-favorite: %v", err)
	}

	favorites, err := repo.GetFavorites(ctx)
	if err != nil {
		t.Fatalf("failed to get favorites: %v", err)
	}

	if len(favorites) != 2 {
		t.Errorf("expected 2 favorites, got %d", len(favorites))
	}
}

func TestViewRepositorySetFavorite_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Set Favorite Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	err = repo.SetFavorite(ctx, view.ID, true)
	if err != nil {
		t.Fatalf("failed to set favorite: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if !retrieved.IsFavorite {
		t.Error("expected IsFavorite to be true")
	}
}

func TestViewRepositoryGetRecentViews_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view1 := domain.NewSavedView("Recent 1")
	err := repo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create view 1: %v", err)
	}

	view2 := domain.NewSavedView("Recent 2")
	err = repo.Create(ctx, view2)
	if err != nil {
		t.Fatalf("failed to create view 2: %v", err)
	}

	err = repo.RecordViewAccess(ctx, view1.ID)
	if err != nil {
		t.Fatalf("failed to record access: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	err = repo.RecordViewAccess(ctx, view2.ID)
	if err != nil {
		t.Fatalf("failed to record access: %v", err)
	}

	recent, err := repo.GetRecentViews(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get recent views: %v", err)
	}

	if len(recent) != 2 {
		t.Errorf("expected 2 recent views, got %d", len(recent))
	}

	if recent[0].Name != "Recent 2" {
		t.Errorf("expected 'Recent 2', got %q", recent[0].Name)
	}
}

func TestViewRepositoryRecordViewAccess_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("Access Test")
	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	beforeAccess := time.Now()
	err = repo.RecordViewAccess(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to record access: %v", err)
	}
	afterAccess := time.Now()

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.LastAccessed == nil {
		t.Error("expected LastAccessed to be set")
	} else {
		if retrieved.LastAccessed.Before(beforeAccess) || retrieved.LastAccessed.After(afterAccess.Add(time.Second)) {
			t.Errorf("unexpected LastAccessed time: %v (expected between %v and %v)", retrieved.LastAccessed, beforeAccess, afterAccess)
		}
	}
}

func TestViewRepositoryWithFilterConfig_Success(t *testing.T) {
	repo := setupViewRepo(t)
	ctx := context.Background()

	view := domain.NewSavedView("With Filter")
	view.FilterConfig.Status = domain.StatusPending
	view.FilterConfig.Priority = domain.PriorityHigh
	view.FilterConfig.Tags = []string{"tag1", "tag2"}
	view.FilterConfig.SearchQuery = "test"

	err := repo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.FilterConfig.Status != domain.StatusPending {
		t.Errorf("expected status pending, got %v", retrieved.FilterConfig.Status)
	}
	if retrieved.FilterConfig.Priority != domain.PriorityHigh {
		t.Errorf("expected priority high, got %v", retrieved.FilterConfig.Priority)
	}
	if len(retrieved.FilterConfig.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(retrieved.FilterConfig.Tags))
	}
	if retrieved.FilterConfig.SearchQuery != "test" {
		t.Errorf("expected search query 'test', got %q", retrieved.FilterConfig.SearchQuery)
	}
}
