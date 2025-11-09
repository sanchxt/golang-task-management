package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProjectMentions(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedBase     string
		expectedMentions []ProjectMention
		expectError      bool
	}{
		{
			name:             "no mentions",
			input:            "simple search query",
			expectedBase:     "simple search query",
			expectedMentions: []ProjectMention{},
			expectError:      false,
		},
		{
			name:         "single exact mention",
			input:        "tasks @backend",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "single fuzzy mention",
			input:        "tasks @~backend",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: true},
			},
			expectError: false,
		},
		{
			name:         "multiple mentions",
			input:        "@backend @frontend API tasks",
			expectedBase: "API tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
				{Name: "frontend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "mixed exact and fuzzy mentions",
			input:        "@backend @~frontend urgent",
			expectedBase: "urgent",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
				{Name: "frontend", Fuzzy: true},
			},
			expectError: false,
		},
		{
			name:         "mention at start",
			input:        "@backend API development",
			expectedBase: "API development",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "mention at end",
			input:        "API development @backend",
			expectedBase: "API development",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "mention in middle",
			input:        "urgent @backend tasks",
			expectedBase: "urgent tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:             "only mention, no query",
			input:            "@backend",
			expectedBase:     "",
			expectedMentions: []ProjectMention{{Name: "backend", Fuzzy: false}},
			expectError:      false,
		},
		{
			name:             "only fuzzy mention, no query",
			input:            "@~backend",
			expectedBase:     "",
			expectedMentions: []ProjectMention{{Name: "backend", Fuzzy: true}},
			expectError:      false,
		},
		{
			name:             "empty input",
			input:            "",
			expectedBase:     "",
			expectedMentions: []ProjectMention{},
			expectError:      false,
		},
		{
			name:             "whitespace only",
			input:            "   ",
			expectedBase:     "",
			expectedMentions: []ProjectMention{},
			expectError:      false,
		},
		{
			name:         "project name with hyphen",
			input:        "@mobile-app tasks",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "mobile-app", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "project name with underscore",
			input:        "@web_frontend tasks",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "web_frontend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "project name with numbers",
			input:        "@v2-backend tasks",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "v2-backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "camel case project name",
			input:        "@BackendAPI tasks",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "BackendAPI", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "duplicate mentions",
			input:        "@backend tasks @backend",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "same project exact and fuzzy",
			input:        "@backend @~backend tasks",
			expectedBase: "tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
				{Name: "backend", Fuzzy: true},
			},
			expectError: false,
		},
		{
			name:         "multiple spaces between tokens",
			input:        "urgent    @backend    tasks",
			expectedBase: "urgent tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "tabs and spaces",
			input:        "urgent\t@backend\ttasks",
			expectedBase: "urgent tasks",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
		{
			name:         "preserve case in query",
			input:        "API @backend Development",
			expectedBase: "API Development",
			expectedMentions: []ProjectMention{
				{Name: "backend", Fuzzy: false},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseProjectMentions(tt.input)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedBase, result.BaseQuery, "base query mismatch")
			assert.Equal(t, len(tt.expectedMentions), len(result.ProjectMentions), "mention count mismatch")

			for i, expected := range tt.expectedMentions {
				assert.Equal(t, expected.Name, result.ProjectMentions[i].Name, "mention %d name mismatch", i)
				assert.Equal(t, expected.Fuzzy, result.ProjectMentions[i].Fuzzy, "mention %d fuzzy flag mismatch", i)
			}
		})
	}
}

func TestProjectMention_String(t *testing.T) {
	tests := []struct {
		name     string
		mention  ProjectMention
		expected string
	}{
		{
			name:     "exact mention",
			mention:  ProjectMention{Name: "backend", Fuzzy: false},
			expected: "@backend",
		},
		{
			name:     "fuzzy mention",
			mention:  ProjectMention{Name: "backend", Fuzzy: true},
			expected: "@~backend",
		},
		{
			name:     "mention with special chars",
			mention:  ProjectMention{Name: "web-app_v2", Fuzzy: false},
			expected: "@web-app_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mention.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsedQuery_HasProjectFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    ProjectMentionQuery
		expected bool
	}{
		{
			name: "has project mentions",
			query: ProjectMentionQuery{
				BaseQuery:       "tasks",
				ProjectMentions: []ProjectMention{{Name: "backend", Fuzzy: false}},
			},
			expected: true,
		},
		{
			name: "no project mentions",
			query: ProjectMentionQuery{
				BaseQuery:       "tasks",
				ProjectMentions: []ProjectMention{},
			},
			expected: false,
		},
		{
			name: "nil project mentions",
			query: ProjectMentionQuery{
				BaseQuery:       "tasks",
				ProjectMentions: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.query.HasProjectFilter()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsedQuery_GetProjectNames(t *testing.T) {
	tests := []struct {
		name     string
		query    ProjectMentionQuery
		expected []string
	}{
		{
			name: "single mention",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{{Name: "backend", Fuzzy: false}},
			},
			expected: []string{"backend"},
		},
		{
			name: "multiple mentions",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{
					{Name: "backend", Fuzzy: false},
					{Name: "frontend", Fuzzy: true},
				},
			},
			expected: []string{"backend", "frontend"},
		},
		{
			name: "duplicate mentions",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{
					{Name: "backend", Fuzzy: false},
					{Name: "backend", Fuzzy: false},
				},
			},
			expected: []string{"backend", "backend"},
		},
		{
			name: "no mentions",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.query.GetProjectNames()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsedQuery_HasFuzzyProjectFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    ProjectMentionQuery
		expected bool
	}{
		{
			name: "has fuzzy mention",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{
					{Name: "backend", Fuzzy: true},
				},
			},
			expected: true,
		},
		{
			name: "has only exact mentions",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{
					{Name: "backend", Fuzzy: false},
				},
			},
			expected: false,
		},
		{
			name: "has mixed mentions",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{
					{Name: "backend", Fuzzy: false},
					{Name: "frontend", Fuzzy: true},
				},
			},
			expected: true,
		},
		{
			name: "no mentions",
			query: ProjectMentionQuery{
				ProjectMentions: []ProjectMention{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.query.HasFuzzyProjectFilter()
			assert.Equal(t, tt.expected, result)
		})
	}
}
