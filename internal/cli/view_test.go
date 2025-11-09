package cli

import (
	"context"
	"testing"

	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
)

func TestLookupViewID_ByNumericID(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "Test View",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	id, err := lookupViewID(ctx, viewRepo, "1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if id == nil || *id != view.ID {
		t.Errorf("expected ID %d, got %v", view.ID, id)
	}
}

func TestLookupViewID_ByName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "My Custom View",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusCompleted,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	id, err := lookupViewID(ctx, viewRepo, "My Custom View")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if id == nil || *id != view.ID {
		t.Errorf("expected ID %d, got %v", view.ID, id)
	}
}

func TestLookupViewID_EmptyString(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	id, err := lookupViewID(ctx, viewRepo, "")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if id != nil {
		t.Errorf("expected ID to be nil, got %v", id)
	}
}

func TestLookupViewID_NotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	_, err := lookupViewID(ctx, viewRepo, "999")
	if err == nil {
		t.Error("expected error for nonexistent view, got nil")
	}
}

func TestLookupViewID_InvalidNumericID(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	_, err := lookupViewID(ctx, viewRepo, "not-a-number")
	if err == nil {
		t.Error("expected error for invalid view name, got nil")
	}
}

func TestViewCreate_WithBasicFilter(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "Pending Tasks",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}

	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	if view.ID == 0 {
		t.Error("expected view ID to be set")
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.Name != "Pending Tasks" {
		t.Errorf("expected name 'Pending Tasks', got '%s'", retrieved.Name)
	}
}

func TestViewCreate_WithDescription(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name:        "Urgent Tasks",
		Description: "All urgent and high priority tasks",
		FilterConfig: domain.SavedViewFilter{
			Priority: domain.PriorityUrgent,
		},
	}

	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.Description != "All urgent and high priority tasks" {
		t.Errorf("expected description, got '%s'", retrieved.Description)
	}
}

func TestViewCreate_WithHotKey(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	hotKey := 1
	view := &domain.SavedView{
		Name:   "Quick View",
		HotKey: &hotKey,
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}

	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.HotKey == nil || *retrieved.HotKey != 1 {
		t.Errorf("expected hot key 1, got %v", retrieved.HotKey)
	}
}

func TestViewCreate_WithFavorite(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name:        "Favorite View",
		IsFavorite:  true,
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}

	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if !retrieved.IsFavorite {
		t.Error("expected view to be favorite")
	}
}

func TestViewUpdate_UpdateName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "Original Name",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	view.Name = "Updated Name"
	err = viewRepo.Update(ctx, view)
	if err != nil {
		t.Fatalf("failed to update view: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", retrieved.Name)
	}
}

func TestViewUpdate_UpdateDescription(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name:        "Test View",
		Description: "Original",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	view.Description = "Updated description"
	err = viewRepo.Update(ctx, view)
	if err != nil {
		t.Fatalf("failed to update view: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", retrieved.Description)
	}
}

func TestViewList_MultipleViews(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		view := &domain.SavedView{
			Name: "View " + string(rune(48+i)),
			FilterConfig: domain.SavedViewFilter{
				Status: domain.StatusPending,
			},
		}
		err := viewRepo.Create(ctx, view)
		if err != nil {
			t.Fatalf("failed to create view %d: %v", i, err)
		}
	}

	views, err := viewRepo.List(ctx, struct {
		IsFavorite  *bool
		HasHotKey   bool
		SearchQuery string
		SortBy      string
		SortOrder   string
		Limit       int
		Offset      int
	}{})

	if err != nil {
		t.Fatalf("failed to list views: %v", err)
	}

	if len(views) != 3 {
		t.Errorf("expected 3 views, got %d", len(views))
	}
}

func TestViewDelete_DeleteView(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "View to Delete",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	viewID := view.ID

	err = viewRepo.Delete(ctx, viewID)
	if err != nil {
		t.Fatalf("failed to delete view: %v", err)
	}

	_, err = viewRepo.GetByID(ctx, viewID)
	if err == nil {
		t.Error("expected error when retrieving deleted view, got nil")
	}
}

func TestViewFavorite_ToggleFavorite(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "Test View",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	err = viewRepo.SetFavorite(ctx, view.ID, true)
	if err != nil {
		t.Fatalf("failed to set favorite: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if !retrieved.IsFavorite {
		t.Error("expected view to be favorite")
	}

	err = viewRepo.SetFavorite(ctx, view.ID, false)
	if err != nil {
		t.Fatalf("failed to unset favorite: %v", err)
	}

	retrieved, err = viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.IsFavorite {
		t.Error("expected view to not be favorite")
	}
}

func TestViewHotKey_AssignHotKey(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "Test View",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	hotKey := 5
	err = viewRepo.SetHotKey(ctx, view.ID, &hotKey)
	if err != nil {
		t.Fatalf("failed to set hot key: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.HotKey == nil || *retrieved.HotKey != 5 {
		t.Errorf("expected hot key 5, got %v", retrieved.HotKey)
	}
}

func TestViewHotKey_ClearHotKey(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	hotKey := 3
	view := &domain.SavedView{
		Name:   "Test View",
		HotKey: &hotKey,
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	err = viewRepo.SetHotKey(ctx, view.ID, nil)
	if err != nil {
		t.Fatalf("failed to clear hot key: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.HotKey != nil {
		t.Errorf("expected hot key to be nil, got %v", retrieved.HotKey)
	}
}

func TestViewValidation_EmptyName(t *testing.T) {
	view := &domain.SavedView{
		Name:         "",
		FilterConfig: domain.SavedViewFilter{},
	}

	err := view.Validate()
	if err == nil {
		t.Error("expected error for empty name, got nil")
	}
}

func TestViewValidation_LongName(t *testing.T) {
	longName := string(make([]byte, 101))
	view := &domain.SavedView{
		Name:         longName,
		FilterConfig: domain.SavedViewFilter{},
	}

	err := view.Validate()
	if err == nil {
		t.Error("expected error for long name, got nil")
	}
}

func TestViewValidation_DuplicateName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view1 := &domain.SavedView{
		Name: "Duplicate Name",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view1)
	if err != nil {
		t.Fatalf("failed to create view1: %v", err)
	}

	view2 := &domain.SavedView{
		Name: "Duplicate Name",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err = viewRepo.Create(ctx, view2)
	if err == nil {
		t.Error("expected error for duplicate name, got nil")
	}
}

func TestViewSearch_SearchByName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	views := []string{"Urgent Tasks", "Pending Review", "High Priority"}
	for _, name := range views {
		view := &domain.SavedView{
			Name: name,
			FilterConfig: domain.SavedViewFilter{
				Status: domain.StatusPending,
			},
		}
		err := viewRepo.Create(ctx, view)
		if err != nil {
			t.Fatalf("failed to create view: %v", err)
		}
	}

	results, err := viewRepo.Search(ctx, "Urgent", 10)
	if err != nil {
		t.Fatalf("failed to search views: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[0].Name != "Urgent Tasks" {
		t.Errorf("expected 'Urgent Tasks', got '%s'", results[0].Name)
	}
}

func TestViewGetFavorites(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view1 := &domain.SavedView{
		Name:       "Favorite 1",
		IsFavorite: true,
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	view2 := &domain.SavedView{
		Name:       "Not Favorite",
		IsFavorite: false,
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	view3 := &domain.SavedView{
		Name:       "Favorite 2",
		IsFavorite: true,
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}

	for _, v := range []*domain.SavedView{view1, view2, view3} {
		err := viewRepo.Create(ctx, v)
		if err != nil {
			t.Fatalf("failed to create view: %v", err)
		}
	}

	favorites, err := viewRepo.GetFavorites(ctx)
	if err != nil {
		t.Fatalf("failed to get favorites: %v", err)
	}

	if len(favorites) != 2 {
		t.Errorf("expected 2 favorites, got %d", len(favorites))
	}

	for _, fav := range favorites {
		if !fav.IsFavorite {
			t.Errorf("expected favorite view, got %s", fav.Name)
		}
	}
}

func TestViewRecordAccess(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	view := &domain.SavedView{
		Name: "Test View",
		FilterConfig: domain.SavedViewFilter{
			Status: domain.StatusPending,
		},
	}
	err := viewRepo.Create(ctx, view)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}

	err = viewRepo.RecordViewAccess(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to record access: %v", err)
	}

	retrieved, err := viewRepo.GetByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("failed to retrieve view: %v", err)
	}

	if retrieved.LastAccessed == nil {
		t.Error("expected LastAccessed to be set")
	}
}

func TestViewGetRecentViews(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		view := &domain.SavedView{
			Name: "View " + string(rune(48+i)),
			FilterConfig: domain.SavedViewFilter{
				Status: domain.StatusPending,
			},
		}
		err := viewRepo.Create(ctx, view)
		if err != nil {
			t.Fatalf("failed to create view: %v", err)
		}

		err = viewRepo.RecordViewAccess(ctx, view.ID)
		if err != nil {
			t.Fatalf("failed to record access: %v", err)
		}
	}

	recent, err := viewRepo.GetRecentViews(ctx, 5)
	if err != nil {
		t.Fatalf("failed to get recent views: %v", err)
	}

	if len(recent) != 3 {
		t.Errorf("expected 3 recent views, got %d", len(recent))
	}

	for _, v := range recent {
		if v.LastAccessed == nil {
			t.Errorf("expected LastAccessed to be set for %s", v.Name)
		}
	}
}
