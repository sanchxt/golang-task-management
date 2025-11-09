package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNewSearchHistory(t *testing.T) {
	queryText := "test query"
	searchMode := SearchModeText
	queryType := QueryTypeSimple

	entry := NewSearchHistory(queryText, searchMode, queryType)

	if entry.QueryText != queryText {
		t.Errorf("expected QueryText %s, got %s", queryText, entry.QueryText)
	}
	if entry.SearchMode != searchMode {
		t.Errorf("expected SearchMode %s, got %s", searchMode, entry.SearchMode)
	}
	if entry.QueryType != queryType {
		t.Errorf("expected QueryType %s, got %s", queryType, entry.QueryType)
	}
	if entry.ResultCount != 0 {
		t.Errorf("expected ResultCount 0, got %d", entry.ResultCount)
	}
	if entry.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if entry.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestSearchHistory_Validate(t *testing.T) {
	tests := []struct {
		name      string
		entry     *SearchHistory
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid entry",
			entry: &SearchHistory{
				QueryText:  "test query",
				SearchMode: SearchModeText,
				QueryType:  QueryTypeSimple,
			},
			wantError: false,
		},
		{
			name: "empty query text",
			entry: &SearchHistory{
				QueryText:  "",
				SearchMode: SearchModeText,
				QueryType:  QueryTypeSimple,
			},
			wantError: true,
			errorMsg:  "query text cannot be empty",
		},
		{
			name: "whitespace only query text",
			entry: &SearchHistory{
				QueryText:  "   ",
				SearchMode: SearchModeText,
				QueryType:  QueryTypeSimple,
			},
			wantError: true,
			errorMsg:  "query text cannot be empty",
		},
		{
			name: "invalid search mode",
			entry: &SearchHistory{
				QueryText:  "test",
				SearchMode: "invalid",
				QueryType:  QueryTypeSimple,
			},
			wantError: true,
			errorMsg:  "invalid search mode",
		},
		{
			name: "invalid query type",
			entry: &SearchHistory{
				QueryText:  "test",
				SearchMode: SearchModeText,
				QueryType:  "invalid",
			},
			wantError: true,
			errorMsg:  "invalid query type",
		},
		{
			name: "fuzzy threshold negative",
			entry: &SearchHistory{
				QueryText:      "test",
				SearchMode:     SearchModeFuzzy,
				QueryType:      QueryTypeSimple,
				FuzzyThreshold: intPtr(-1),
			},
			wantError: true,
			errorMsg:  "fuzzy threshold must be between 0 and 100",
		},
		{
			name: "fuzzy threshold too high",
			entry: &SearchHistory{
				QueryText:      "test",
				SearchMode:     SearchModeFuzzy,
				QueryType:      QueryTypeSimple,
				FuzzyThreshold: intPtr(101),
			},
			wantError: true,
			errorMsg:  "fuzzy threshold must be between 0 and 100",
		},
		{
			name: "valid fuzzy threshold",
			entry: &SearchHistory{
				QueryText:      "test",
				SearchMode:     SearchModeFuzzy,
				QueryType:      QueryTypeSimple,
				FuzzyThreshold: intPtr(60),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestSearchHistory_GetModeIndicator(t *testing.T) {
	tests := []struct {
		name     string
		entry    *SearchHistory
		expected string
	}{
		{
			name: "text mode",
			entry: &SearchHistory{
				SearchMode: SearchModeText,
			},
			expected: "",
		},
		{
			name: "regex mode",
			entry: &SearchHistory{
				SearchMode: SearchModeRegex,
			},
			expected: "[RE]",
		},
		{
			name: "fuzzy mode without threshold",
			entry: &SearchHistory{
				SearchMode: SearchModeFuzzy,
			},
			expected: "[F]",
		},
		{
			name: "fuzzy mode with threshold",
			entry: &SearchHistory{
				SearchMode:     SearchModeFuzzy,
				FuzzyThreshold: intPtr(75),
			},
			expected: "[F:75]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetModeIndicator()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSearchHistory_GetRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		updatedAt time.Time
		expected  string
	}{
		{
			name:      "just now",
			updatedAt: now.Add(-30 * time.Second),
			expected:  "just now",
		},
		{
			name:      "1 minute ago",
			updatedAt: now.Add(-1 * time.Minute),
			expected:  "1 minute ago",
		},
		{
			name:      "5 minutes ago",
			updatedAt: now.Add(-5 * time.Minute),
			expected:  "5 minutes ago",
		},
		{
			name:      "1 hour ago",
			updatedAt: now.Add(-1 * time.Hour),
			expected:  "1 hour ago",
		},
		{
			name:      "3 hours ago",
			updatedAt: now.Add(-3 * time.Hour),
			expected:  "3 hours ago",
		},
		{
			name:      "yesterday",
			updatedAt: now.Add(-24 * time.Hour),
			expected:  "yesterday",
		},
		{
			name:      "3 days ago",
			updatedAt: now.Add(-3 * 24 * time.Hour),
			expected:  "3 days ago",
		},
		{
			name:      "1 week ago",
			updatedAt: now.Add(-7 * 24 * time.Hour),
			expected:  "1 week ago",
		},
		{
			name:      "2 weeks ago",
			updatedAt: now.Add(-14 * 24 * time.Hour),
			expected:  "2 weeks ago",
		},
		{
			name:      "1 month ago",
			updatedAt: now.Add(-30 * 24 * time.Hour),
			expected:  "1 month ago",
		},
		{
			name:      "3 months ago",
			updatedAt: now.Add(-90 * 24 * time.Hour),
			expected:  "3 months ago",
		},
		{
			name:      "1 year ago",
			updatedAt: now.Add(-365 * 24 * time.Hour),
			expected:  "1 year ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &SearchHistory{
				UpdatedAt: tt.updatedAt,
			}
			result := entry.GetRelativeTime()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSearchHistory_GetDisplayText(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		entry    *SearchHistory
		contains []string
	}{
		{
			name: "text mode simple query",
			entry: &SearchHistory{
				QueryText:  "test query",
				SearchMode: SearchModeText,
				QueryType:  QueryTypeSimple,
				UpdatedAt:  now.Add(-5 * time.Minute),
			},
			contains: []string{"test query", "5 minutes ago"},
		},
		{
			name: "regex mode query",
			entry: &SearchHistory{
				QueryText:  "api.*error",
				SearchMode: SearchModeRegex,
				QueryType:  QueryTypeSimple,
				UpdatedAt:  now.Add(-1 * time.Hour),
			},
			contains: []string{"[RE]", "api.*error", "1 hour ago"},
		},
		{
			name: "fuzzy mode with threshold",
			entry: &SearchHistory{
				QueryText:      "backend",
				SearchMode:     SearchModeFuzzy,
				FuzzyThreshold: intPtr(70),
				QueryType:      QueryTypeSimple,
				UpdatedAt:      now.Add(-2 * time.Hour),
			},
			contains: []string{"[F:70]", "backend", "2 hours ago"},
		},
		{
			name: "project mention query",
			entry: &SearchHistory{
				QueryText:     "@backend api",
				SearchMode:    SearchModeText,
				QueryType:     QueryTypeProjectMention,
				ProjectFilter: "backend",
				UpdatedAt:     now.Add(-1 * 24 * time.Hour),
			},
			contains: []string{"@backend api", "yesterday"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetDisplayText()
			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

func TestIsValidSearchMode(t *testing.T) {
	tests := []struct {
		mode  SearchMode
		valid bool
	}{
		{SearchModeText, true},
		{SearchModeRegex, true},
		{SearchModeFuzzy, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			result := isValidSearchMode(tt.mode)
			if result != tt.valid {
				t.Errorf("expected %v for mode %q, got %v", tt.valid, tt.mode, result)
			}
		})
	}
}

func TestIsValidQueryType(t *testing.T) {
	tests := []struct {
		qType QueryType
		valid bool
	}{
		{QueryTypeSimple, true},
		{QueryTypeQueryLanguage, true},
		{QueryTypeProjectMention, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.qType), func(t *testing.T) {
			result := isValidQueryType(tt.qType)
			if result != tt.valid {
				t.Errorf("expected %v for query type %q, got %v", tt.valid, tt.qType, result)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
