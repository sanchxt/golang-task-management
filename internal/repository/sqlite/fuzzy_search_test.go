package sqlite

import (
	"context"
	"testing"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

func TestFuzzySearch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	tasks := []*domain.Task{
		{Title: "Backend API Development", Priority: domain.PriorityHigh, Status: domain.StatusPending},
		{Title: "Frontend Dashboard", Priority: domain.PriorityMedium, Status: domain.StatusPending},
		{Title: "Database Optimization", Priority: domain.PriorityHigh, Status: domain.StatusPending},
		{Title: "Backend Authentication", Priority: domain.PriorityUrgent, Status: domain.StatusInProgress},
		{Title: "API Documentation", Priority: domain.PriorityLow, Status: domain.StatusPending},
		{Title: "User Interface Design", Priority: domain.PriorityMedium, Status: domain.StatusPending},
		{Title: "Backup System", Priority: domain.PriorityLow, Status: domain.StatusPending},
	}

	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	tests := []struct {
		name           string
		searchQuery    string
		fuzzyThreshold int
		expectedTitles []string
		minResults     int
	}{
		{
			name:           "fuzzy search for 'back' - should find backend and backup",
			searchQuery:    "back",
			fuzzyThreshold: 70,
			expectedTitles: []string{"Backend API Development", "Backend Authentication", "Backup System"},
			minResults:     3,
		},
		{
			name:           "fuzzy search for 'api' - exact and partial matches",
			searchQuery:    "api",
			fuzzyThreshold: 50,
			expectedTitles: []string{"API Documentation", "Backend API Development"},
			minResults:     2,
		},
		{
			name:           "fuzzy search for 'be' - abbreviation match",
			searchQuery:    "be",
			fuzzyThreshold: 50,
			expectedTitles: []string{"Backend API Development", "Backend Authentication"},
			minResults:     2,
		},
		{
			name:           "fuzzy search for 'bcknd' - typo tolerance",
			searchQuery:    "bcknd",
			fuzzyThreshold: 50,
			expectedTitles: []string{"Backend API Development", "Backend Authentication"},
			minResults:     2,
		},
		{
			name:           "fuzzy search with high threshold - only best matches",
			searchQuery:    "backend",
			fuzzyThreshold: 85,
			expectedTitles: []string{"Backend API Development", "Backend Authentication"},
			minResults:     2,
		},
		{
			name:           "fuzzy search with low threshold - more results",
			searchQuery:    "data",
			fuzzyThreshold: 60,
			expectedTitles: []string{"Database Optimization"},
			minResults:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := repository.TaskFilter{
				SearchQuery:    tt.searchQuery,
				SearchMode:     "fuzzy",
				FuzzyThreshold: tt.fuzzyThreshold,
			}

			results, err := repo.List(ctx, filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(results) < tt.minResults {
				t.Errorf("Expected at least %d results, got %d", tt.minResults, len(results))
			}

			resultTitles := make(map[string]bool)
			for _, task := range results {
				resultTitles[task.Title] = true
			}

			for _, expectedTitle := range tt.expectedTitles {
				if !resultTitles[expectedTitle] {
					t.Errorf("Expected result '%s' not found in results", expectedTitle)
				}
			}

			if len(results) > 0 && len(tt.expectedTitles) > 0 {
				found := false
				for i := 0; i < len(tt.expectedTitles) && i < len(results); i++ {
					if results[0].Title == tt.expectedTitles[i] {
						found = true
						break
					}
				}
				if !found {
					t.Logf("Warning: First result '%s' may not be the most relevant", results[0].Title)
				}
			}
		})
	}
}

func TestFuzzySearchWithOtherFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	tasks := []*domain.Task{
		{Title: "Backend API Development", Priority: domain.PriorityHigh, Status: domain.StatusPending, Tags: []string{"backend", "api"}},
		{Title: "Backend Testing", Priority: domain.PriorityMedium, Status: domain.StatusCompleted, Tags: []string{"backend", "testing"}},
		{Title: "Backend Authentication", Priority: domain.PriorityUrgent, Status: domain.StatusInProgress, Tags: []string{"backend", "security"}},
		{Title: "Frontend Backend Integration", Priority: domain.PriorityHigh, Status: domain.StatusPending, Tags: []string{"frontend", "backend"}},
	}

	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	tests := []struct {
		name           string
		filter         repository.TaskFilter
		expectedTitles []string
	}{
		{
			name: "fuzzy search + status filter",
			filter: repository.TaskFilter{
				SearchQuery:    "backend",
				SearchMode:     "fuzzy",
				FuzzyThreshold: 70,
				Status:         domain.StatusPending,
			},
			expectedTitles: []string{"Backend API Development", "Frontend Backend Integration"},
		},
		{
			name: "fuzzy search + priority filter",
			filter: repository.TaskFilter{
				SearchQuery:    "backend",
				SearchMode:     "fuzzy",
				FuzzyThreshold: 70,
				Priority:       domain.PriorityHigh,
			},
			expectedTitles: []string{"Backend API Development", "Frontend Backend Integration"},
		},
		{
			name: "fuzzy search + tag filter",
			filter: repository.TaskFilter{
				SearchQuery:    "backend",
				SearchMode:     "fuzzy",
				FuzzyThreshold: 70,
				Tags:           []string{"api"},
			},
			expectedTitles: []string{"Backend API Development"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repo.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(results) != len(tt.expectedTitles) {
				t.Errorf("Expected %d results, got %d", len(tt.expectedTitles), len(results))
			}

			resultTitles := make(map[string]bool)
			for _, task := range results {
				resultTitles[task.Title] = true
			}

			for _, expectedTitle := range tt.expectedTitles {
				if !resultTitles[expectedTitle] {
					t.Errorf("Expected result '%s' not found", expectedTitle)
				}
			}
		})
	}
}

func TestFuzzySearchEmpty(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	tasks := []*domain.Task{
		{Title: "Backend API", Priority: domain.PriorityHigh, Status: domain.StatusPending},
		{Title: "Frontend Dashboard", Priority: domain.PriorityMedium, Status: domain.StatusPending},
	}

	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	filter := repository.TaskFilter{
		SearchQuery:    "xyz",
		SearchMode:     "fuzzy",
		FuzzyThreshold: 70,
	}

	results, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for non-matching query, got %d", len(results))
	}
}

func TestFuzzySearchThreshold(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	task := &domain.Task{
		Title:    "Backend API Development",
		Priority: domain.PriorityHigh,
		Status:   domain.StatusPending,
	}

	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	tests := []struct {
		name           string
		searchQuery    string
		threshold      int
		expectResults  bool
	}{
		{
			name:          "high threshold - exact match passes",
			searchQuery:   "backend",
			threshold:     90,
			expectResults: true,
		},
		{
			name:          "medium threshold - partial match passes",
			searchQuery:   "back",
			threshold:     70,
			expectResults: true,
		},
		{
			name:          "low threshold - loose match passes",
			searchQuery:   "bknd",
			threshold:     40,
			expectResults: true,
		},
		{
			name:          "very high threshold - partial match fails",
			searchQuery:   "ba",
			threshold:     95,
			expectResults: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := repository.TaskFilter{
				SearchQuery:    tt.searchQuery,
				SearchMode:     "fuzzy",
				FuzzyThreshold: tt.threshold,
			}

			results, err := repo.List(ctx, filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			hasResults := len(results) > 0
			if hasResults != tt.expectResults {
				t.Errorf("Expected results=%v, got results=%v (count=%d)", tt.expectResults, hasResults, len(results))
			}
		})
	}
}

func TestFuzzySearchCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	tasks := []*domain.Task{
		{Title: "Backend API Development", Priority: domain.PriorityHigh, Status: domain.StatusPending},
		{Title: "Backend Authentication", Priority: domain.PriorityUrgent, Status: domain.StatusInProgress},
		{Title: "Frontend Dashboard", Priority: domain.PriorityMedium, Status: domain.StatusPending},
	}

	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	filter := repository.TaskFilter{
		SearchQuery:    "backend",
		SearchMode:     "fuzzy",
		FuzzyThreshold: 70,
	}

	count, err := repo.Count(ctx, filter)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	results, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if count != int64(len(results)) {
		t.Errorf("Count() = %d, but List() returned %d results", count, len(results))
	}

	if count < 2 {
		t.Errorf("Expected at least 2 matches for 'backend', got %d", count)
	}
}

func TestFuzzySearchPagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		task := &domain.Task{
			Title:    "Backend Task " + string(rune('A'+i)),
			Priority: domain.PriorityMedium,
			Status:   domain.StatusPending,
		}
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	filter := repository.TaskFilter{
		SearchQuery:    "backend",
		SearchMode:     "fuzzy",
		FuzzyThreshold: 70,
		Limit:          5,
		Offset:         0,
	}

	results1, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(results1) != 5 {
		t.Errorf("Expected 5 results on first page, got %d", len(results1))
	}

	filter.Offset = 5
	results2, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(results2) != 5 {
		t.Errorf("Expected 5 results on second page, got %d", len(results2))
	}

	ids1 := make(map[int64]bool)
	for _, task := range results1 {
		ids1[task.ID] = true
	}

	for _, task := range results2 {
		if ids1[task.ID] {
			t.Errorf("Found duplicate task %d across pages", task.ID)
		}
	}
}

func TestFuzzySearchDescriptionAndTags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(db)
	projectRepo := NewProjectRepository(db)
	ctx := context.Background()

	project := &domain.Project{
		Name:        "Backend Services",
		Description: "Backend microservices project",
	}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	tasks := []*domain.Task{
		{
			Title:       "Implement authentication",
			Description: "Backend authentication system with JWT",
			Priority:    domain.PriorityHigh,
			Status:      domain.StatusPending,
			Tags:        []string{"backend", "security"},
		},
		{
			Title:       "Setup database",
			Description: "Configure database connections",
			Priority:    domain.PriorityMedium,
			Status:      domain.StatusPending,
			ProjectID:   &project.ID,
		},
		{
			Title:       "API endpoints",
			Description: "Create REST API endpoints",
			Priority:    domain.PriorityHigh,
			Status:      domain.StatusPending,
			Tags:        []string{"api", "restful"},
		},
	}

	for _, task := range tasks {
		if err := repo.Create(ctx, task); err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}
	}

	tests := []struct {
		name           string
		searchQuery    string
		expectedTitles []string
	}{
		{
			name:           "fuzzy search in description",
			searchQuery:    "backend",
			expectedTitles: []string{"Implement authentication", "Setup database"},
		},
		{
			name:           "fuzzy search in tags",
			searchQuery:    "api",
			expectedTitles: []string{"API endpoints"},
		},
		{
			name:           "fuzzy search in project name",
			searchQuery:    "backend services",
			expectedTitles: []string{"Setup database"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := repository.TaskFilter{
				SearchQuery:    tt.searchQuery,
				SearchMode:     "fuzzy",
				FuzzyThreshold: 60,
			}

			results, err := repo.List(ctx, filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(results) == 0 {
				t.Errorf("Expected some results, got none")
			}

			resultTitles := make(map[string]bool)
			for _, task := range results {
				resultTitles[task.Title] = true
			}

			foundAny := false
			for _, expectedTitle := range tt.expectedTitles {
				if resultTitles[expectedTitle] {
					foundAny = true
					break
				}
			}

			if !foundAny {
				t.Errorf("None of the expected results were found")
			}
		})
	}
}
