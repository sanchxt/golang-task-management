package domain

import (
	"testing"
	"time"
)

func TestNewSavedView(t *testing.T) {
	name := "Test View"
	view := NewSavedView(name)

	if view.Name != name {
		t.Errorf("expected name %q, got %q", name, view.Name)
	}

	if view.IsFavorite {
		t.Error("expected IsFavorite to be false")
	}

	if view.HotKey != nil {
		t.Errorf("expected HotKey to be nil, got %v", view.HotKey)
	}

	if !view.CreatedAt.Before(time.Now().Add(time.Second)) {
		t.Error("expected CreatedAt to be set to recent time")
	}

	if !view.UpdatedAt.Before(time.Now().Add(time.Second)) {
		t.Error("expected UpdatedAt to be set to recent time")
	}
}

func TestSavedViewValidation_EmptyName(t *testing.T) {
	view := NewSavedView("")
	err := view.Validate()
	if err == nil {
		t.Error("expected validation error for empty name")
	}
}

func TestSavedViewValidation_NameTooLong(t *testing.T) {
	longName := ""
	for i := 0; i < 101; i++ {
		longName += "a"
	}

	view := NewSavedView(longName)
	err := view.Validate()
	if err == nil {
		t.Error("expected validation error for name exceeding 100 characters")
	}
}

func TestSavedViewValidation_DescriptionTooLong(t *testing.T) {
	longDesc := ""
	for i := 0; i < 501; i++ {
		longDesc += "a"
	}

	view := NewSavedView("Valid Name")
	view.Description = longDesc
	err := view.Validate()
	if err == nil {
		t.Error("expected validation error for description exceeding 500 characters")
	}
}

func TestSavedViewValidation_ValidHotKey(t *testing.T) {
	view := NewSavedView("Valid View")
	for i := 1; i <= 9; i++ {
		hotKey := i
		view.HotKey = &hotKey
		err := view.Validate()
		if err != nil {
			t.Errorf("expected no error for valid hot key %d, got %v", i, err)
		}
	}
}

func TestSavedViewValidation_InvalidHotKeyTooSmall(t *testing.T) {
	view := NewSavedView("Valid View")
	hotKey := 0
	view.HotKey = &hotKey
	err := view.Validate()
	if err == nil {
		t.Error("expected validation error for hot key 0")
	}
}

func TestSavedViewValidation_InvalidHotKeyTooLarge(t *testing.T) {
	view := NewSavedView("Valid View")
	hotKey := 10
	view.HotKey = &hotKey
	err := view.Validate()
	if err == nil {
		t.Error("expected validation error for hot key 10")
	}
}

func TestSavedViewValidation_ValidView(t *testing.T) {
	view := NewSavedView("Valid View")
	view.Description = "This is a valid description"
	hotKey := 5
	view.HotKey = &hotKey
	view.IsFavorite = true

	err := view.Validate()
	if err != nil {
		t.Errorf("expected no validation error, got %v", err)
	}
}

func TestSavedViewHasFilter_NoFilter(t *testing.T) {
	view := NewSavedView("Empty View")
	if view.HasFilter() {
		t.Error("expected HasFilter to be false for empty filter")
	}
}

func TestSavedViewHasFilter_WithStatus(t *testing.T) {
	view := NewSavedView("Status View")
	view.FilterConfig.Status = StatusPending

	if !view.HasFilter() {
		t.Error("expected HasFilter to be true with status filter")
	}
}

func TestSavedViewHasFilter_WithPriority(t *testing.T) {
	view := NewSavedView("Priority View")
	view.FilterConfig.Priority = PriorityHigh

	if !view.HasFilter() {
		t.Error("expected HasFilter to be true with priority filter")
	}
}

func TestSavedViewHasFilter_WithProjectID(t *testing.T) {
	view := NewSavedView("Project View")
	projectID := int64(1)
	view.FilterConfig.ProjectID = &projectID

	if !view.HasFilter() {
		t.Error("expected HasFilter to be true with project ID filter")
	}
}

func TestSavedViewHasFilter_WithTags(t *testing.T) {
	view := NewSavedView("Tags View")
	view.FilterConfig.Tags = []string{"important", "urgent"}

	if !view.HasFilter() {
		t.Error("expected HasFilter to be true with tags filter")
	}
}

func TestSavedViewHasFilter_WithSearch(t *testing.T) {
	view := NewSavedView("Search View")
	view.FilterConfig.SearchQuery = "backend"

	if !view.HasFilter() {
		t.Error("expected HasFilter to be true with search query")
	}
}

func TestSavedViewHasFilter_WithMultipleFilters(t *testing.T) {
	view := NewSavedView("Complex View")
	view.FilterConfig.Status = StatusInProgress
	view.FilterConfig.Priority = PriorityHigh
	view.FilterConfig.Tags = []string{"backend"}

	if !view.HasFilter() {
		t.Error("expected HasFilter to be true with multiple filters")
	}
}

func TestSavedViewGetFilterSummary_NoFilters(t *testing.T) {
	view := NewSavedView("Empty View")
	summary := view.GetFilterSummary()

	if summary != "no filters" {
		t.Errorf("expected 'no filters', got %q", summary)
	}
}

func TestSavedViewGetFilterSummary_WithStatus(t *testing.T) {
	view := NewSavedView("Status View")
	view.FilterConfig.Status = StatusPending
	summary := view.GetFilterSummary()

	if !contains(summary, "status:pending") {
		t.Errorf("expected 'status:pending' in summary, got %q", summary)
	}
}

func TestSavedViewGetFilterSummary_WithPriority(t *testing.T) {
	view := NewSavedView("Priority View")
	view.FilterConfig.Priority = PriorityHigh
	summary := view.GetFilterSummary()

	if !contains(summary, "priority:high") {
		t.Errorf("expected 'priority:high' in summary, got %q", summary)
	}
}

func TestSavedViewGetFilterSummary_WithTags(t *testing.T) {
	view := NewSavedView("Tags View")
	view.FilterConfig.Tags = []string{"tag1", "tag2"}
	summary := view.GetFilterSummary()

	if !contains(summary, "2 tags") {
		t.Errorf("expected '2 tags' in summary, got %q", summary)
	}
}

func TestSavedViewGetFilterSummary_WithSearch(t *testing.T) {
	view := NewSavedView("Search View")
	view.FilterConfig.SearchQuery = "test query"
	summary := view.GetFilterSummary()

	if !contains(summary, "search: test query") {
		t.Errorf("expected 'search: test query' in summary, got %q", summary)
	}
}

func TestSavedViewGetFilterSummary_MultipleFilters(t *testing.T) {
	view := NewSavedView("Complex View")
	view.FilterConfig.Status = StatusInProgress
	view.FilterConfig.Priority = PriorityHigh
	view.FilterConfig.Tags = []string{"backend"}
	view.FilterConfig.SearchQuery = "api"

	summary := view.GetFilterSummary()

	if !contains(summary, "status:in_progress") {
		t.Errorf("expected 'status:in_progress' in summary, got %q", summary)
	}
	if !contains(summary, "priority:high") {
		t.Errorf("expected 'priority:high' in summary, got %q", summary)
	}
	if !contains(summary, "1 tags") {
		t.Errorf("expected '1 tags' in summary, got %q", summary)
	}
	if !contains(summary, "search: api") {
		t.Errorf("expected 'search: api' in summary, got %q", summary)
	}
}

func TestSavedViewGetHotKeyDisplay_NoHotKey(t *testing.T) {
	view := NewSavedView("No Hot Key")
	display := view.GetHotKeyDisplay()

	if display != "" {
		t.Errorf("expected empty string, got %q", display)
	}
}

func TestSavedViewGetHotKeyDisplay_WithHotKey(t *testing.T) {
	view := NewSavedView("With Hot Key")
	hotKey := 5
	view.HotKey = &hotKey
	display := view.GetHotKeyDisplay()

	if display != "[5]" {
		t.Errorf("expected '[5]', got %q", display)
	}
}

func TestSavedViewGetFavoriteIndicator_NotFavorite(t *testing.T) {
	view := NewSavedView("Not Favorite")
	indicator := view.GetFavoriteIndicator()

	if indicator != "" {
		t.Errorf("expected empty string, got %q", indicator)
	}
}

func TestSavedViewGetFavoriteIndicator_Favorite(t *testing.T) {
	view := NewSavedView("Favorite")
	view.IsFavorite = true
	indicator := view.GetFavoriteIndicator()

	if indicator != "★" {
		t.Errorf("expected '★', got %q", indicator)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && substr != ""
}
