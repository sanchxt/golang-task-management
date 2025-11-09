package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProject(t *testing.T) {
	name := "Test Project"
	project := NewProject(name)

	assert.Equal(t, name, project.Name)
	assert.Equal(t, ProjectStatusActive, project.Status)
	assert.False(t, project.IsFavorite)
	assert.NotNil(t, project.Aliases)
	assert.Empty(t, project.Aliases)
	assert.Equal(t, "", project.Notes)
	assert.False(t, project.CreatedAt.IsZero())
	assert.False(t, project.UpdatedAt.IsZero())
}

func TestProjectValidate_Aliases(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid project with aliases",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"be", "back"},
			},
			wantErr: false,
		},
		{
			name: "valid project with no aliases",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid alias with hyphens and underscores",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"my-project", "my_project", "proj-123"},
			},
			wantErr: false,
		},
		{
			name: "too many aliases",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10", "a11"},
			},
			wantErr: true,
			errMsg:  "cannot have more than 10 aliases",
		},
		{
			name: "alias too short",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"a"},
			},
			wantErr: true,
			errMsg:  "must be at least 2 characters",
		},
		{
			name: "alias too long",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{strings.Repeat("a", 31)},
			},
			wantErr: true,
			errMsg:  "cannot exceed 30 characters",
		},
		{
			name: "empty alias",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{""},
			},
			wantErr: true,
			errMsg:  "alias cannot be empty",
		},
		{
			name: "whitespace only alias",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"   "},
			},
			wantErr: true,
			errMsg:  "alias cannot be empty",
		},
		{
			name: "duplicate aliases",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"be", "be"},
			},
			wantErr: true,
			errMsg:  "duplicate alias",
		},
		{
			name: "duplicate aliases case-insensitive",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"be", "BE"},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid alias format - uppercase",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"Backend"},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid alias format - special chars",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"back@end"},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "invalid alias format - spaces",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"back end"},
			},
			wantErr: true,
			errMsg:  "lowercase alphanumeric",
		},
		{
			name: "max valid aliases (10)",
			project: &Project{
				Name:    "Backend",
				Aliases: []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProjectValidate_Notes(t *testing.T) {
	tests := []struct {
		name    string
		project *Project
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid project with notes",
			project: &Project{
				Name:  "Backend",
				Notes: "This is a test note",
			},
			wantErr: false,
		},
		{
			name: "valid project with empty notes",
			project: &Project{
				Name:  "Backend",
				Notes: "",
			},
			wantErr: false,
		},
		{
			name: "valid project with long notes",
			project: &Project{
				Name:  "Backend",
				Notes: strings.Repeat("a", 10000),
			},
			wantErr: false,
		},
		{
			name: "notes too long",
			project: &Project{
				Name:  "Backend",
				Notes: strings.Repeat("a", 10001),
			},
			wantErr: true,
			errMsg:  "notes cannot exceed 10,000 characters",
		},
		{
			name: "notes with markdown",
			project: &Project{
				Name:  "Backend",
				Notes: "# Heading\n\n- Item 1\n- Item 2\n\n```go\nfunc main() {}\n```",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProject_HasAlias(t *testing.T) {
	project := &Project{
		Name:    "Backend",
		Aliases: []string{"be", "back", "backend-api"},
	}

	tests := []struct {
		name     string
		alias    string
		expected bool
	}{
		{"exact match", "be", true},
		{"exact match 2", "back", true},
		{"exact match with hyphen", "backend-api", true},
		{"case insensitive match", "BE", true},
		{"case insensitive match 2", "BACK", true},
		{"with whitespace", "  be  ", true},
		{"no match", "frontend", false},
		{"partial match", "bac", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := project.HasAlias(tt.alias)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProject_GetAliases(t *testing.T) {
	t.Run("returns copy of aliases", func(t *testing.T) {
		project := &Project{
			Name:    "Backend",
			Aliases: []string{"be", "back"},
		}

		aliases := project.GetAliases()
		assert.Equal(t, []string{"be", "back"}, aliases)

		aliases[0] = "modified"
		assert.Equal(t, "be", project.Aliases[0])
	})

	t.Run("returns empty slice for nil aliases", func(t *testing.T) {
		project := &Project{
			Name:    "Backend",
			Aliases: nil,
		}

		aliases := project.GetAliases()
		assert.NotNil(t, aliases)
		assert.Empty(t, aliases)
	})

	t.Run("returns empty slice for empty aliases", func(t *testing.T) {
		project := &Project{
			Name:    "Backend",
			Aliases: []string{},
		}

		aliases := project.GetAliases()
		assert.NotNil(t, aliases)
		assert.Empty(t, aliases)
	})
}

func TestProject_FormatAliases(t *testing.T) {
	tests := []struct {
		name     string
		aliases  []string
		expected string
	}{
		{"multiple aliases", []string{"be", "back", "api"}, "be, back, api"},
		{"single alias", []string{"be"}, "be"},
		{"empty aliases", []string{}, ""},
		{"nil aliases", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &Project{
				Name:    "Backend",
				Aliases: tt.aliases,
			}
			result := project.FormatAliases()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProject_HasNotes(t *testing.T) {
	tests := []struct {
		name     string
		notes    string
		expected bool
	}{
		{"has notes", "This is a note", true},
		{"empty notes", "", false},
		{"whitespace only notes", "   ", false},
		{"multiline notes", "Line 1\nLine 2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &Project{
				Name:  "Backend",
				Notes: tt.notes,
			}
			result := project.HasNotes()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidAliasFormat(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		expected bool
	}{
		{"lowercase only", "backend", true},
		{"numbers", "backend123", true},
		{"hyphens", "back-end", true},
		{"underscores", "back_end", true},
		{"mixed valid", "my-project_123", true},
		{"uppercase", "Backend", false},
		{"spaces", "back end", false},
		{"special chars", "back@end", false},
		{"dots", "back.end", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAliasFormatBool(tt.alias)
			assert.Equal(t, tt.expected, result)
		})
	}
}
