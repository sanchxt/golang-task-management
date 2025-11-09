package fuzzy

import (
	"testing"
)

func TestExactMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		want    int
	}{
		{
			name:    "exact match lowercase",
			pattern: "backend",
			text:    "backend",
			want:    100,
		},
		{
			name:    "exact match mixed case",
			pattern: "Backend",
			text:    "backend",
			want:    100,
		},
		{
			name:    "exact match with spaces",
			pattern: "backend api",
			text:    "backend api",
			want:    100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Match(tt.pattern, tt.text)
			if score != tt.want {
				t.Errorf("Match(%q, %q) = %d, want %d", tt.pattern, tt.text, score, tt.want)
			}
		})
	}
}

func TestPartialMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		minScore int
		maxScore int
	}{
		{
			name:     "prefix match",
			pattern:  "back",
			text:     "backend",
			minScore: 85,
			maxScore: 100,
		},
		{
			name:     "suffix match",
			pattern:  "end",
			text:     "backend",
			minScore: 65,
			maxScore: 90,
		},
		{
			name:     "middle match",
			pattern:  "ckend",
			text:     "backend",
			minScore: 70,
			maxScore: 95,
		},
		{
			name:     "scattered characters",
			pattern:  "bnd",
			text:     "backend",
			minScore: 40,
			maxScore: 95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Match(tt.pattern, tt.text)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Match(%q, %q) = %d, want between %d and %d",
					tt.pattern, tt.text, score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestCamelCaseMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		minScore int
	}{
		{
			name:     "camel case initials",
			pattern:  "BC",
			text:     "BackendController",
			minScore: 55,
		},
		{
			name:     "camel case partial",
			pattern:  "BaCo",
			text:     "BackendController",
			minScore: 70,
		},
		{
			name:     "snake case match",
			pattern:  "bt",
			text:     "backend_task",
			minScore: 70,
		},
		{
			name:     "kebab case match",
			pattern:  "ba",
			text:     "backend-api",
			minScore: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Match(tt.pattern, tt.text)
			if score < tt.minScore {
				t.Errorf("Match(%q, %q) = %d, want at least %d",
					tt.pattern, tt.text, score, tt.minScore)
			}
		})
	}
}

func TestConsecutiveBonus(t *testing.T) {
	consecutiveScore := Match("backend", "backend-api")
	scatteredScore := Match("backend", "b-a-c-k-e-n-d")

	if consecutiveScore <= scatteredScore {
		t.Errorf("Consecutive match should score higher: consecutive=%d, scattered=%d",
			consecutiveScore, scatteredScore)
	}
}

func TestFirstCharacterBonus(t *testing.T) {
	firstCharScore := Match("back", "backend")
	notFirstCharScore := Match("ack", "backend")

	if firstCharScore <= notFirstCharScore {
		t.Errorf("First character match should score higher: first=%d, notFirst=%d",
			firstCharScore, notFirstCharScore)
	}
}

func TestNoMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
	}{
		{
			name:    "completely different",
			pattern: "xyz",
			text:    "backend",
		},
		{
			name:    "pattern longer than text",
			pattern: "backend-api-service",
			text:    "api",
		},
		{
			name:    "wrong order",
			pattern: "dne",
			text:    "backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Match(tt.pattern, tt.text)
			if score > 0 {
				t.Errorf("Match(%q, %q) = %d, want 0 (no match)",
					tt.pattern, tt.text, score)
			}
		})
	}
}

func TestCaseInsensitivity(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		texts    []string
	}{
		{
			name:    "different cases",
			pattern: "backend",
			texts:   []string{"backend", "Backend", "BACKEND", "BaCkEnD"},
		},
		{
			name:    "mixed case pattern",
			pattern: "BaCk",
			texts:   []string{"backend", "Backend", "BACKEND"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scores := make([]int, len(tt.texts))
			for i, text := range tt.texts {
				scores[i] = Match(tt.pattern, text)
			}

			firstScore := scores[0]
			for i, score := range scores {
				if score != firstScore {
					t.Errorf("Case-insensitive match failed: Match(%q, %q) = %d, expected %d",
						tt.pattern, tt.texts[i], score, firstScore)
				}
			}
		})
	}
}

func TestEmptyInputs(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		want    int
	}{
		{
			name:    "empty pattern",
			pattern: "",
			text:    "backend",
			want:    0,
		},
		{
			name:    "empty text",
			pattern: "backend",
			text:    "",
			want:    0,
		},
		{
			name:    "both empty",
			pattern: "",
			text:    "",
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Match(tt.pattern, tt.text)
			if score != tt.want {
				t.Errorf("Match(%q, %q) = %d, want %d", tt.pattern, tt.text, score, tt.want)
			}
		})
	}
}

func TestRealWorldExamples(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		text     string
		minScore int
	}{
		{
			name:     "project name typo",
			pattern:  "bcknd",
			text:     "backend",
			minScore: 50,
		},
		{
			name:     "project name abbreviation",
			pattern:  "be",
			text:     "backend",
			minScore: 70,
		},
		{
			name:     "task title fuzzy",
			pattern:  "fix api",
			text:     "Fix API authentication bug",
			minScore: 70,
		},
		{
			name:     "partial project name",
			pattern:  "front",
			text:     "frontend-dashboard",
			minScore: 75,
		},
		{
			name:     "misspelling",
			pattern:  "databse",
			text:     "database",
			minScore: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := Match(tt.pattern, tt.text)
			if score < tt.minScore {
				t.Errorf("Match(%q, %q) = %d, want at least %d",
					tt.pattern, tt.text, score, tt.minScore)
			}
		})
	}
}

func TestMatchMany(t *testing.T) {
	pattern := "back"
	texts := []string{
		"backend",
		"frontend",
		"database",
		"backup",
		"cache",
	}

	results := MatchMany(pattern, texts, 50)

	if len(results) < 2 {
		t.Errorf("MatchMany returned %d results, want at least 2", len(results))
	}

	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Errorf("Results not sorted by score: results[%d].Score=%d < results[%d].Score=%d",
				i, results[i].Score, i+1, results[i+1].Score)
		}
	}

	if len(results) > 0 && results[0].Text != "backend" {
		t.Errorf("Best match = %q, want %q", results[0].Text, "backend")
	}
}

func TestScoreOrdering(t *testing.T) {
	pattern := "api"

	tests := []struct {
		better       string
		worse        string
		allowEqual   bool
	}{
		{
			better:     "api",           
			worse:      "api-service",   
			allowEqual: true, 
		},
		{
			better:     "api-service",   
			worse:      "backend-api",   
			allowEqual: false,
		},
	}

	for _, tt := range tests {
		betterScore := Match(pattern, tt.better)
		worseScore := Match(pattern, tt.worse)

		if tt.allowEqual {
			if betterScore < worseScore {
				t.Errorf("Match(%q, %q)=%d should score at least as high as Match(%q, %q)=%d",
					pattern, tt.better, betterScore, pattern, tt.worse, worseScore)
			}
		} else {
			if betterScore <= worseScore {
				t.Errorf("Match(%q, %q)=%d should score higher than Match(%q, %q)=%d",
					pattern, tt.better, betterScore, pattern, tt.worse, worseScore)
			}
		}
	}
}

func BenchmarkMatch(b *testing.B) {
	pattern := "backend"
	text := "backend-api-service-controller"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Match(pattern, text)
	}
}

func BenchmarkMatchMany(b *testing.B) {
	pattern := "api"
	texts := []string{
		"backend-api-service",
		"frontend-dashboard",
		"database-connector",
		"api-gateway",
		"authentication-service",
		"authorization-middleware",
		"cache-manager",
		"api-documentation",
		"logging-utility",
		"monitoring-agent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MatchMany(pattern, texts, 50)
	}
}
