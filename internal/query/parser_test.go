package query

import (
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectError  bool
		filterCount  int
		checkFilters func(*testing.T, *ParsedQuery)
	}{
		{
			name:        "simple status filter",
			input:       "status:pending",
			expectError: false,
			filterCount: 1,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "status" {
					t.Errorf("Filter field = %q, expected 'status'", q.Filters[0].Field)
				}
				if q.Filters[0].Value != "pending" {
					t.Errorf("Filter value = %q, expected 'pending'", q.Filters[0].Value)
				}
				if q.Filters[0].Operator != ":" {
					t.Errorf("Filter operator = %q, expected ':'", q.Filters[0].Operator)
				}
			},
		},
		{
			name:        "multiple filters",
			input:       "status:pending priority:high",
			expectError: false,
			filterCount: 2,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "status" || q.Filters[0].Value != "pending" {
					t.Errorf("First filter incorrect: %+v", q.Filters[0])
				}
				if q.Filters[1].Field != "priority" || q.Filters[1].Value != "high" {
					t.Errorf("Second filter incorrect: %+v", q.Filters[1])
				}
			},
		},
		{
			name:        "@mention syntax",
			input:       "@backend",
			expectError: false,
			filterCount: 1,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "project" {
					t.Errorf("Filter field = %q, expected 'project'", q.Filters[0].Field)
				}
				if q.Filters[0].Value != "backend" {
					t.Errorf("Filter value = %q, expected 'backend'", q.Filters[0].Value)
				}
				if q.Filters[0].IsFuzzy {
					t.Errorf("Filter should not be fuzzy for @mention")
				}
			},
		},
		{
			name:        "@~fuzzy mention",
			input:       "@~backend",
			expectError: false,
			filterCount: 1,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "project" {
					t.Errorf("Filter field = %q, expected 'project'", q.Filters[0].Field)
				}
				if q.Filters[0].Value != "backend" {
					t.Errorf("Filter value = %q, expected 'backend'", q.Filters[0].Value)
				}
				if !q.Filters[0].IsFuzzy {
					t.Errorf("Filter should be fuzzy for @~mention")
				}
			},
		},
		{
			name:        "negated filter",
			input:       "-tag:wontfix",
			expectError: false,
			filterCount: 1,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "tag" {
					t.Errorf("Filter field = %q, expected 'tag'", q.Filters[0].Field)
				}
				if q.Filters[0].Value != "wontfix" {
					t.Errorf("Filter value = %q, expected 'wontfix'", q.Filters[0].Value)
				}
				if !q.Filters[0].IsNot {
					t.Errorf("Filter should be negated")
				}
			},
		},
		{
			name:        "complex query",
			input:       "status:pending priority:high @backend tag:bug -tag:wontfix",
			expectError: false,
			filterCount: 5,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				expected := []struct {
					field string
					value string
					isNot bool
				}{
					{"status", "pending", false},
					{"priority", "high", false},
					{"project", "backend", false},
					{"tag", "bug", false},
					{"tag", "wontfix", true},
				}

				for i, exp := range expected {
					if i >= len(q.Filters) {
						t.Errorf("Missing filter at index %d", i)
						continue
					}
					f := q.Filters[i]
					if f.Field != exp.field || f.Value != exp.value || f.IsNot != exp.isNot {
						t.Errorf("Filter[%d] = {%s:%s, isNot:%v}, expected {%s:%s, isNot:%v}",
							i, f.Field, f.Value, f.IsNot, exp.field, exp.value, exp.isNot)
					}
				}
			},
		},
		{
			name:        "date filter",
			input:       "due:2025-01-15",
			expectError: false,
			filterCount: 1,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "due" {
					t.Errorf("Filter field = %q, expected 'due'", q.Filters[0].Field)
				}
				if q.Filters[0].Value != "2025-01-15" {
					t.Errorf("Filter value = %q, expected '2025-01-15'", q.Filters[0].Value)
				}
			},
		},
		{
			name:        "relative date filter",
			input:       "due:+7d",
			expectError: false,
			filterCount: 1,
			checkFilters: func(t *testing.T, q *ParsedQuery) {
				if q.Filters[0].Field != "due" {
					t.Errorf("Filter field = %q, expected 'due'", q.Filters[0].Field)
				}
				if q.Filters[0].Value != "+7d" {
					t.Errorf("Filter value = %q, expected '+7d'", q.Filters[0].Value)
				}
			},
		},
		{
			name:        "empty query",
			input:       "",
			expectError: false,
			filterCount: 0,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: false,
			filterCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := ParseQuery(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("ParseQuery() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("ParseQuery() unexpected error: %v", err)
			}

			if query == nil {
				if !tt.expectError {
					t.Fatalf("ParseQuery() returned nil query without error")
				}
				return
			}

			if len(query.Filters) != tt.filterCount {
				t.Errorf("ParseQuery() filter count = %d, expected %d", len(query.Filters), tt.filterCount)
			}

			if tt.checkFilters != nil {
				tt.checkFilters(t, query)
			}
		})
	}
}

func TestParsedQueryHelpers(t *testing.T) {
	query, err := ParseQuery("status:pending priority:high tag:bug -tag:wontfix @backend")
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	t.Run("HasField", func(t *testing.T) {
		if !query.HasField("status") {
			t.Error("HasField('status') = false, expected true")
		}
		if !query.HasField("priority") {
			t.Error("HasField('priority') = false, expected true")
		}
		if query.HasField("due") {
			t.Error("HasField('due') = true, expected false")
		}
	}	)

	t.Run("GetField", func(t *testing.T) {
		statusFilter := query.GetField("status")
		if statusFilter == nil {
			t.Fatal("GetField('status') returned nil")
		}
		if statusFilter.Value != "pending" {
			t.Errorf("GetField('status').Value = %q, expected 'pending'", statusFilter.Value)
		}

		dueFilter := query.GetField("due")
		if dueFilter != nil {
			t.Error("GetField('due') should return nil for non-existent field")
		}
	})

	t.Run("GetAllFields", func(t *testing.T) {
		tagFilters := query.GetAllFields("tag")
		if len(tagFilters) != 2 {
			t.Errorf("GetAllFields('tag') returned %d filters, expected 2", len(tagFilters))
		}

		hasPositive := false
		hasNegative := false
		for _, f := range tagFilters {
			if f.Value == "bug" && !f.IsNot {
				hasPositive = true
			}
			if f.Value == "wontfix" && f.IsNot {
				hasNegative = true
			}
		}

		if !hasPositive {
			t.Error("GetAllFields('tag') missing positive filter")
		}
		if !hasNegative {
			t.Error("GetAllFields('tag') missing negative filter")
		}
	})
}

func TestParseQueryComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		operator string
	}{
		{
			name:     "less than",
			input:    "due:<2025-01-15",
			operator: "<",
		},
		{
			name:     "greater than",
			input:    "created:>2025-01-01",
			operator: ">",
		},
		{
			name:     "equals",
			input:    "priority=high",
			operator: "=",
		},
		{
			name:     "not equals",
			input:    "status!=completed",
			operator: "!=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := ParseQuery(tt.input)
			if err != nil {
				t.Fatalf("ParseQuery() error = %v", err)
			}

			if len(query.Filters) != 1 {
				t.Fatalf("Expected 1 filter, got %d", len(query.Filters))
			}

			if query.Filters[0].Operator != tt.operator {
				t.Errorf("Operator = %q, expected %q", query.Filters[0].Operator, tt.operator)
			}
		})
	}
}

func TestQueryFilterString(t *testing.T) {
	tests := []struct {
		name     string
		filter   QueryFilter
		expected string
	}{
		{
			name: "simple filter",
			filter: QueryFilter{
				Field:    "status",
				Operator: ":",
				Value:    "pending",
			},
			expected: "status:pending",
		},
		{
			name: "negated filter",
			filter: QueryFilter{
				Field:    "tag",
				Operator: ":",
				Value:    "wontfix",
				IsNot:    true,
			},
			expected: "-tag:wontfix",
		},
		{
			name: "fuzzy filter",
			filter: QueryFilter{
				Field:    "project",
				Operator: ":",
				Value:    "backend",
				IsFuzzy:  true,
			},
			expected: "~project:backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.String()
			if result != tt.expected {
				t.Errorf("String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
