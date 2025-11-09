package sqlite

import (
	"context"
	"testing"

	"task-management/internal/domain"
)

func setupSearchHistoryTestDB(t *testing.T) (*DB, context.Context) {
	t.Helper()
	db, err := NewDB(Config{Path: ":memory:"})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	return db, context.Background()
}

func TestSearchHistoryRepository_RecordSearch(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	t.Run("create new entry", func(t *testing.T) {
		entry := domain.NewSearchHistory("test query", domain.SearchModeText, domain.QueryTypeSimple)

		err := repo.RecordSearch(ctx, entry)
		if err != nil {
			t.Fatalf("failed to record search: %v", err)
		}

		if entry.ID == 0 {
			t.Error("expected ID to be set after insert")
		}

		retrieved, err := repo.GetByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("failed to retrieve entry: %v", err)
		}

		if retrieved.QueryText != entry.QueryText {
			t.Errorf("expected QueryText %q, got %q", entry.QueryText, retrieved.QueryText)
		}
	})

	t.Run("deduplicate identical query", func(t *testing.T) {
		entry1 := domain.NewSearchHistory("duplicate query", domain.SearchModeText, domain.QueryTypeSimple)
		err := repo.RecordSearch(ctx, entry1)
		if err != nil {
			t.Fatalf("failed to record first search: %v", err)
		}

		firstID := entry1.ID
		originalCreatedAt := entry1.CreatedAt

		entry2 := domain.NewSearchHistory("duplicate query", domain.SearchModeText, domain.QueryTypeSimple)
		err = repo.RecordSearch(ctx, entry2)
		if err != nil {
			t.Fatalf("failed to record second search: %v", err)
		}

		if entry2.ID != firstID {
			t.Errorf("expected deduplication to use same ID %d, got %d", firstID, entry2.ID)
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}
		if count != 2 {
			t.Errorf("expected count to be 2 after deduplication, got %d", count)
		}

		retrieved, err := repo.GetByID(ctx, firstID)
		if err != nil {
			t.Fatalf("failed to retrieve entry: %v", err)
		}

		if retrieved.CreatedAt.Unix() != originalCreatedAt.Unix() {
			t.Errorf("expected created_at to remain %v (unix: %d), got %v (unix: %d)",
				originalCreatedAt, originalCreatedAt.Unix(),
				retrieved.CreatedAt, retrieved.CreatedAt.Unix())
		}
	})

	t.Run("different search modes create separate entries", func(t *testing.T) {
		queryText := "same query different mode"

		entry1 := domain.NewSearchHistory(queryText, domain.SearchModeText, domain.QueryTypeSimple)
		err := repo.RecordSearch(ctx, entry1)
		if err != nil {
			t.Fatalf("failed to record text mode search: %v", err)
		}

		entry2 := domain.NewSearchHistory(queryText, domain.SearchModeFuzzy, domain.QueryTypeSimple)
		threshold := 70
		entry2.FuzzyThreshold = &threshold
		err = repo.RecordSearch(ctx, entry2)
		if err != nil {
			t.Fatalf("failed to record fuzzy mode search: %v", err)
		}

		if entry1.ID == entry2.ID {
			t.Error("expected different IDs for different search modes")
		}
	})

	t.Run("validation error on invalid entry", func(t *testing.T) {
		entry := domain.NewSearchHistory("", domain.SearchModeText, domain.QueryTypeSimple)

		err := repo.RecordSearch(ctx, entry)
		if err == nil {
			t.Error("expected validation error for empty query text")
		}
	})
}

func TestSearchHistoryRepository_List(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	entries := []struct {
		query string
		mode  domain.SearchMode
		qType domain.QueryType
	}{
		{"list test query 1", domain.SearchModeText, domain.QueryTypeSimple},
		{"list test query 2", domain.SearchModeRegex, domain.QueryTypeSimple},
		{"list test query 3", domain.SearchModeFuzzy, domain.QueryTypeQueryLanguage},
	}

	for _, e := range entries {
		entry := domain.NewSearchHistory(e.query, e.mode, e.qType)
		if err := repo.RecordSearch(ctx, entry); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
	}

	t.Run("list all entries", func(t *testing.T) {
		results, err := repo.List(ctx, 0)
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("expected 3 entries, got %d", len(results))
		}

		found := make(map[string]bool)
		for _, result := range results {
			found[result.QueryText] = true
		}

		for _, e := range entries {
			if !found[e.query] {
				t.Errorf("expected to find query %q in results", e.query)
			}
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		results, err := repo.List(ctx, 2)
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 entries with limit, got %d", len(results))
		}
	})

	t.Run("list returns empty for no results", func(t *testing.T) {
		if err := repo.Clear(ctx); err != nil {
			t.Fatalf("failed to clear: %v", err)
		}

		results, err := repo.List(ctx, 0)
		if err != nil {
			t.Fatalf("failed to list entries: %v", err)
		}

		if len(results) != 0 {
			t.Errorf("expected 0 entries after clear, got %d", len(results))
		}
	})
}

func TestSearchHistoryRepository_GetByID(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	entry := domain.NewSearchHistory("test query", domain.SearchModeFuzzy, domain.QueryTypeSimple)
	threshold := 65
	entry.FuzzyThreshold = &threshold
	entry.ProjectFilter = "backend"

	if err := repo.RecordSearch(ctx, entry); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	t.Run("get existing entry", func(t *testing.T) {
		retrieved, err := repo.GetByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}

		if retrieved.QueryText != entry.QueryText {
			t.Errorf("expected QueryText %q, got %q", entry.QueryText, retrieved.QueryText)
		}
		if retrieved.SearchMode != entry.SearchMode {
			t.Errorf("expected SearchMode %q, got %q", entry.SearchMode, retrieved.SearchMode)
		}
		if retrieved.FuzzyThreshold == nil || *retrieved.FuzzyThreshold != threshold {
			t.Errorf("expected FuzzyThreshold %d, got %v", threshold, retrieved.FuzzyThreshold)
		}
		if retrieved.ProjectFilter != entry.ProjectFilter {
			t.Errorf("expected ProjectFilter %q, got %q", entry.ProjectFilter, retrieved.ProjectFilter)
		}
	})

	t.Run("get non-existent entry", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 99999)
		if err == nil {
			t.Error("expected error for non-existent entry")
		}
	})
}

func TestSearchHistoryRepository_Delete(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	entry := domain.NewSearchHistory("to delete", domain.SearchModeText, domain.QueryTypeSimple)
	if err := repo.RecordSearch(ctx, entry); err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	t.Run("delete existing entry", func(t *testing.T) {
		err := repo.Delete(ctx, entry.ID)
		if err != nil {
			t.Fatalf("failed to delete entry: %v", err)
		}

		_, err = repo.GetByID(ctx, entry.ID)
		if err == nil {
			t.Error("expected error when getting deleted entry")
		}
	})

	t.Run("delete non-existent entry", func(t *testing.T) {
		err := repo.Delete(ctx, 99999)
		if err == nil {
			t.Error("expected error when deleting non-existent entry")
		}
	})
}

func TestSearchHistoryRepository_Clear(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	for i := 0; i < 5; i++ {
		entry := domain.NewSearchHistory("query "+string(rune('A'+i)), domain.SearchModeText, domain.QueryTypeSimple)
		if err := repo.RecordSearch(ctx, entry); err != nil {
			t.Fatalf("failed to create entry: %v", err)
		}
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 entries before clear, got %d", count)
	}

	err = repo.Clear(ctx)
	if err != nil {
		t.Fatalf("failed to clear: %v", err)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("failed to count after clear: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 entries after clear, got %d", count)
	}
}

func TestSearchHistoryRepository_Count(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	t.Run("count empty repository", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
	})

	t.Run("count after adding entries", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			entry := domain.NewSearchHistory("count query "+string(rune('A'+i)), domain.SearchModeText, domain.QueryTypeSimple)
			if err := repo.RecordSearch(ctx, entry); err != nil {
				t.Fatalf("failed to create entry: %v", err)
			}
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("failed to count: %v", err)
		}
		if count != 3 {
			t.Errorf("expected count 3, got %d", count)
		}
	})
}

func TestSearchHistoryRepository_NullableFields(t *testing.T) {
	db, ctx := setupSearchHistoryTestDB(t)
	defer db.Close()

	repo := NewSearchHistoryRepository(db)

	t.Run("entry without nullable fields", func(t *testing.T) {
		entry := domain.NewSearchHistory("no nulls", domain.SearchModeText, domain.QueryTypeSimple)

		err := repo.RecordSearch(ctx, entry)
		if err != nil {
			t.Fatalf("failed to record search: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}

		if retrieved.FuzzyThreshold != nil {
			t.Errorf("expected nil FuzzyThreshold, got %v", retrieved.FuzzyThreshold)
		}
		if retrieved.ProjectFilter != "" {
			t.Errorf("expected empty ProjectFilter, got %q", retrieved.ProjectFilter)
		}
	})

	t.Run("entry with nullable fields", func(t *testing.T) {
		entry := domain.NewSearchHistory("with nulls", domain.SearchModeFuzzy, domain.QueryTypeProjectMention)
		threshold := 80
		entry.FuzzyThreshold = &threshold
		entry.ProjectFilter = "frontend"

		err := repo.RecordSearch(ctx, entry)
		if err != nil {
			t.Fatalf("failed to record search: %v", err)
		}

		retrieved, err := repo.GetByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("failed to get entry: %v", err)
		}

		if retrieved.FuzzyThreshold == nil {
			t.Error("expected FuzzyThreshold to be set")
		} else if *retrieved.FuzzyThreshold != threshold {
			t.Errorf("expected FuzzyThreshold %d, got %d", threshold, *retrieved.FuzzyThreshold)
		}

		if retrieved.ProjectFilter != "frontend" {
			t.Errorf("expected ProjectFilter 'frontend', got %q", retrieved.ProjectFilter)
		}
	})
}
