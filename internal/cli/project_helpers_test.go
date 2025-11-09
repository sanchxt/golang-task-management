package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
)

func TestLookupProjectID(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	backend := domain.NewProject("backend")
	require.NoError(t, repo.Create(ctx, backend))

	frontend := domain.NewProject("frontend")
	require.NoError(t, repo.Create(ctx, frontend))

	tests := []struct {
		name        string
		projectStr  string
		expectedID  *int64
		expectError bool
	}{
		{
			name:        "empty string returns nil",
			projectStr:  "",
			expectedID:  nil,
			expectError: false,
		},
		{
			name:        "whitespace only returns nil",
			projectStr:  "   ",
			expectedID:  nil,
			expectError: false,
		},
		{
			name:        "lookup by numeric ID",
			projectStr:  "1",
			expectedID:  &backend.ID,
			expectError: false,
		},
		{
			name:        "lookup by name",
			projectStr:  "backend",
			expectedID:  &backend.ID,
			expectError: false,
		},
		{
			name:        "lookup by name case sensitive",
			projectStr:  "Backend",
			expectedID:  nil,
			expectError: true,
		},
		{
			name:        "non-existent numeric ID",
			projectStr:  "999",
			expectedID:  nil,
			expectError: true,
		},
		{
			name:        "non-existent name",
			projectStr:  "nonexistent",
			expectedID:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lookupProjectID(ctx, repo, tt.projectStr)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.expectedID == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expectedID, *result)
			}
		})
	}
}

func TestLookupProjectByFuzzyName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	backend := domain.NewProject("backend-api")
	require.NoError(t, repo.Create(ctx, backend))

	frontend := domain.NewProject("frontend-web")
	require.NoError(t, repo.Create(ctx, frontend))

	mobile := domain.NewProject("mobile-app")
	require.NoError(t, repo.Create(ctx, mobile))

	tests := []struct {
		name        string
		searchName  string
		threshold   int
		expectedID  *int64
		expectError bool
	}{
		{
			name:        "exact match",
			searchName:  "backend-api",
			threshold:   60,
			expectedID:  &backend.ID,
			expectError: false,
		},
		{
			name:        "fuzzy match - abbreviation",
			searchName:  "back",
			threshold:   50,
			expectedID:  &backend.ID,
			expectError: false,
		},
		{
			name:        "fuzzy match - typo",
			searchName:  "backnd",
			threshold:   50,
			expectedID:  &backend.ID,
			expectError: false,
		},
		{
			name:        "fuzzy match - partial",
			searchName:  "front",
			threshold:   50,
			expectedID:  &frontend.ID,
			expectError: false,
		},
		{
			name:        "fuzzy match - mobile",
			searchName:  "mob",
			threshold:   50,
			expectedID:  &mobile.ID,
			expectError: false,
		},
		{
			name:        "no match - threshold too high",
			searchName:  "xyz123",
			threshold:   60,
			expectedID:  nil,
			expectError: true,
		},
		{
			name:        "no match - completely different",
			searchName:  "xyz",
			threshold:   60,
			expectedID:  nil,
			expectError: true,
		},
		{
			name:        "empty search returns error",
			searchName:  "",
			threshold:   60,
			expectedID:  nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lookupProjectByFuzzyName(ctx, repo, tt.searchName, tt.threshold)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, *tt.expectedID, *result)
		})
	}
}

func TestLookupProjectByFuzzyName_PrefersBetterMatch(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	backend := domain.NewProject("backend")
	require.NoError(t, repo.Create(ctx, backend))

	backendAPI := domain.NewProject("backend-api")
	require.NoError(t, repo.Create(ctx, backendAPI))

	backendAuth := domain.NewProject("backend-auth")
	require.NoError(t, repo.Create(ctx, backendAuth))

	result, err := lookupProjectByFuzzyName(ctx, repo, "backend", 60)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.ID, *result, "should prefer exact match 'backend'")

	result, err = lookupProjectByFuzzyName(ctx, repo, "backapi", 50)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backendAPI.ID, *result, "should match 'backend-api' best")
}

func TestLookupProjectByFuzzyName_NoProjects(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	result, err := lookupProjectByFuzzyName(ctx, repo, "backend", 60)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no matching project found")
}
