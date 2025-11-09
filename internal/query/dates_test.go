package query

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		input       string
		expectError bool
		checkResult func(*testing.T, *time.Time, string)
	}{
		{
			name:        "ISO date YYYY-MM-DD",
			input:       "2025-01-15",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
				if !result.Equal(expected) && result.Format("2006-01-02") != "2025-01-15" {
					t.Errorf("Date = %v, expected 2025-01-15", result)
				}
			},
		},
		{
			name:        "ISO date YYYY/MM/DD",
			input:       "2025/01/15",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				if result.Format("2006-01-02") != "2025-01-15" {
					t.Errorf("Date = %v, expected 2025-01-15", result)
				}
			},
		},
		{
			name:        "today keyword",
			input:       "today",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now)
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (today)", result, expected)
				}
			},
		},
		{
			name:        "tomorrow keyword",
			input:       "tomorrow",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now.AddDate(0, 0, 1))
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (tomorrow)", result, expected)
				}
			},
		},
		{
			name:        "yesterday keyword",
			input:       "yesterday",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now.AddDate(0, 0, -1))
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (yesterday)", result, expected)
				}
			},
		},
		{
			name:        "relative offset +7d",
			input:       "+7d",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now.AddDate(0, 0, 7))
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (+7 days)", result, expected)
				}
			},
		},
		{
			name:        "relative offset -1w",
			input:       "-1w",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now.AddDate(0, 0, -7))
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (-1 week)", result, expected)
				}
			},
		},
		{
			name:        "relative offset +2M",
			input:       "+2M",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now.AddDate(0, 2, 0))
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (+2 months)", result, expected)
				}
			},
		},
		{
			name:        "relative offset 7d (no sign, default positive)",
			input:       "7d",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result == nil {
					t.Fatal("Expected non-nil result")
				}
				expected := startOfDay(now.AddDate(0, 0, 7))
				if !result.Equal(expected) {
					t.Errorf("Date = %v, expected %v (7 days)", result, expected)
				}
			},
		},
		{
			name:        "special: none",
			input:       "none",
			expectError: false,
			checkResult: func(t *testing.T, result *time.Time, special string) {
				if result != nil {
					t.Error("Expected nil result for 'none'")
				}
				if special != "none" {
					t.Errorf("Special = %q, expected 'none'", special)
				}
			},
		},
		{
			name:        "invalid date",
			input:       "invalid-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, special, err := ParseDate(tt.input)

			if tt.expectError && err == nil {
				t.Error("ParseDate() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("ParseDate() unexpected error: %v", err)
			}

			if tt.checkResult != nil && !tt.expectError {
				tt.checkResult(t, result, special)
			}
		})
	}
}

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		operator    string
		expectError bool
		checkResult func(*testing.T, *time.Time, *time.Time)
	}{
		{
			name:        "exact date (single day range)",
			value:       "2025-01-15",
			operator:    ":",
			expectError: false,
			checkResult: func(t *testing.T, start, end *time.Time) {
				if start == nil || end == nil {
					t.Fatal("Expected non-nil start and end")
				}
				if start.Format("2006-01-02") != "2025-01-15" {
					t.Errorf("Start = %v, expected 2025-01-15", start)
				}
				if end.Format("2006-01-02") != "2025-01-15" {
					t.Errorf("End = %v, expected 2025-01-15", end)
				}
				if end.Hour() != 23 || end.Minute() != 59 {
					t.Errorf("End time not end of day: %v", end)
				}
			},
		},
		{
			name:        "less than (before date)",
			value:       "2025-01-15",
			operator:    "<",
			expectError: false,
			checkResult: func(t *testing.T, start, end *time.Time) {
				if start != nil {
					t.Errorf("Expected nil start for <, got %v", start)
				}
				if end == nil {
					t.Fatal("Expected non-nil end")
				}
				if end.Format("2006-01-02") != "2025-01-15" {
					t.Errorf("End = %v, expected 2025-01-15", end)
				}
			},
		},
		{
			name:        "greater than (after date)",
			value:       "2025-01-15",
			operator:    ">",
			expectError: false,
			checkResult: func(t *testing.T, start, end *time.Time) {
				if start == nil {
					t.Fatal("Expected non-nil start")
				}
				if end != nil {
					t.Errorf("Expected nil end for >, got %v", end)
				}
				if start.Format("2006-01-02") != "2025-01-15" {
					t.Errorf("Start = %v, expected 2025-01-15", start)
				}
			},
		},
		{
			name:        "range syntax",
			value:       "2025-01-01..2025-01-31",
			operator:    ":",
			expectError: false,
			checkResult: func(t *testing.T, start, end *time.Time) {
				if start == nil || end == nil {
					t.Fatal("Expected non-nil start and end")
				}
				if start.Format("2006-01-02") != "2025-01-01" {
					t.Errorf("Start = %v, expected 2025-01-01", start)
				}
				if end.Format("2006-01-02") != "2025-01-31" {
					t.Errorf("End = %v, expected 2025-01-31", end)
				}
			},
		},
		{
			name:        "relative date with greater than",
			value:       "today",
			operator:    ">",
			expectError: false,
			checkResult: func(t *testing.T, start, end *time.Time) {
				if start == nil {
					t.Fatal("Expected non-nil start")
				}
				expected := time.Now()
				if start.Format("2006-01-02") != expected.Format("2006-01-02") {
					t.Errorf("Start = %v, expected today (%v)", start, expected)
				}
			},
		},
		{
			name:        "relative offset with less than",
			value:       "+7d",
			operator:    "<",
			expectError: false,
			checkResult: func(t *testing.T, start, end *time.Time) {
				if end == nil {
					t.Fatal("Expected non-nil end")
				}
				expected := time.Now().AddDate(0, 0, 7)
				if end.Format("2006-01-02") != expected.Format("2006-01-02") {
					t.Errorf("End = %v, expected +7d (%v)", end, expected)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := ParseDateRange(tt.value, tt.operator)

			if tt.expectError && err == nil {
				t.Error("ParseDateRange() expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("ParseDateRange() unexpected error: %v", err)
			}

			if tt.checkResult != nil && !tt.expectError {
				tt.checkResult(t, start, end)
			}
		})
	}
}

func TestDateHelpers(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)

	t.Run("startOfDay", func(t *testing.T) {
		result := startOfDay(testTime)
		if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
			t.Errorf("startOfDay() = %v, expected 00:00:00", result)
		}
		if result.Format("2006-01-02") != "2025-01-15" {
			t.Errorf("startOfDay() changed date: %v", result)
		}
	})

	t.Run("endOfDay", func(t *testing.T) {
		result := endOfDay(testTime)
		if result.Hour() != 23 || result.Minute() != 59 || result.Second() != 59 {
			t.Errorf("endOfDay() = %v, expected 23:59:59", result)
		}
		if result.Format("2006-01-02") != "2025-01-15" {
			t.Errorf("endOfDay() changed date: %v", result)
		}
	})

	t.Run("FormatDateForSQL", func(t *testing.T) {
		result := FormatDateForSQL(testTime)
		expected := "2025-01-15 14:30:45"
		if result != expected {
			t.Errorf("FormatDateForSQL() = %q, expected %q", result, expected)
		}
	})

	t.Run("FormatDateForDisplay", func(t *testing.T) {
		result := FormatDateForDisplay(testTime)
		expected := "2025-01-15"
		if result != expected {
			t.Errorf("FormatDateForDisplay() = %q, expected %q", result, expected)
		}
	})
}
