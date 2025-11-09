package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ProjectStatus string

const (
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusArchived  ProjectStatus = "archived"
	ProjectStatusCompleted ProjectStatus = "completed"
)

type Project struct {
	ID          int64          `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description"`
	ParentID    *int64         `db:"parent_id" json:"parent_id,omitempty"`
	Color       string         `db:"color" json:"color"`
	Icon        string         `db:"icon" json:"icon"`
	Status      ProjectStatus  `db:"status" json:"status"`
	IsFavorite  bool           `db:"is_favorite" json:"is_favorite"`
	Aliases     []string       `db:"aliases" json:"aliases,omitempty"`
	Notes       string         `db:"notes" json:"notes,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`

	Parent      *Project  `db:"-" json:"parent,omitempty"`
	Children    []*Project `db:"-" json:"children,omitempty"`
	TaskCount   int       `db:"-" json:"task_count,omitempty"`
	Path        string    `db:"-" json:"path,omitempty"`
}

func (p *Project) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("project name cannot be empty")
	}

	if len(p.Name) > 100 {
		return errors.New("project name cannot exceed 100 characters")
	}

	if len(p.Description) > 500 {
		return errors.New("project description cannot exceed 500 characters")
	}

	if p.Status != "" && !isValidProjectStatus(p.Status) {
		return errors.New("invalid status: must be active, archived, or completed")
	}

	if p.Color != "" && !isValidColor(p.Color) {
		return errors.New("invalid color: must be a valid terminal color name")
	}

	if p.ParentID != nil && *p.ParentID == p.ID {
		return errors.New("project cannot be its own parent")
	}

	if len(p.Aliases) > 10 {
		return errors.New("project cannot have more than 10 aliases")
	}

	aliasMap := make(map[string]bool)
	for _, alias := range p.Aliases {
		aliasTrimmed := strings.TrimSpace(alias)
		aliasLower := strings.ToLower(aliasTrimmed)

		if aliasTrimmed == "" {
			return errors.New("alias cannot be empty")
		}
		if len(aliasTrimmed) < 2 {
			return errors.New("alias must be at least 2 characters")
		}
		if len(aliasTrimmed) > 30 {
			return errors.New("alias cannot exceed 30 characters")
		}
		if !isValidAliasFormatBool(aliasTrimmed) {
			return errors.New("alias must contain only lowercase alphanumeric characters, hyphens, and underscores")
		}
		if aliasMap[aliasLower] {
			return errors.New("duplicate alias: " + alias)
		}
		aliasMap[aliasLower] = true
	}

	if len(p.Notes) > 10000 {
		return errors.New("notes cannot exceed 10,000 characters")
	}

	return nil
}

func NewProject(name string) *Project {
	now := time.Now()
	return &Project{
		Name:       name,
		Status:     ProjectStatusActive,
		IsFavorite: false,
		Aliases:    []string{},
		Notes:      "",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (p *Project) IsRoot() bool {
	return p.ParentID == nil
}

func (p *Project) HasChildren() bool {
	return len(p.Children) > 0
}

func (p *Project) GetDepth() int {
	depth := 0
	current := p.Parent
	for current != nil {
		depth++
		current = current.Parent
	}
	return depth
}

func (p *Project) BuildPath() string {
	if p.Parent == nil {
		return p.Name
	}

	var parts []string
	current := p
	for current != nil {
		parts = append([]string{current.Name}, parts...)
		current = current.Parent
	}

	return strings.Join(parts, " > ")
}

func isValidProjectStatus(s ProjectStatus) bool {
	switch s {
	case ProjectStatusActive, ProjectStatusArchived, ProjectStatusCompleted:
		return true
	default:
		return false
	}
}

var validColors = map[string]bool{
	"black":   true,
	"red":     true,
	"green":   true,
	"yellow":  true,
	"blue":    true,
	"magenta": true,
	"cyan":    true,
	"white":   true,
	"gray":    true,
	"bright-red":     true,
	"bright-green":   true,
	"bright-yellow":  true,
	"bright-blue":    true,
	"bright-magenta": true,
	"bright-cyan":    true,
	"bright-white":   true,
}

func isValidColor(c string) bool {
	return validColors[strings.ToLower(c)]
}

func GetValidColors() []string {
	colors := make([]string, 0, len(validColors))
	for color := range validColors {
		colors = append(colors, color)
	}
	return colors
}

var commonIcons = []string{
	"📦", "🚀", "💼", "🔧", "⚙️", "🎯", "📊", "🌟",
	"🔨", "💻", "📱", "🌐", "🔐", "🎨", "📝", "🏠",
}

func GetCommonIcons() []string {
	return commonIcons
}

func IsValidAliasFormat(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if len(alias) < 2 {
		return fmt.Errorf("alias must be at least 2 characters long")
	}
	if len(alias) > 30 {
		return fmt.Errorf("alias must be at most 30 characters long")
	}
	for _, r := range alias {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("alias can only contain lowercase alphanumeric characters, hyphens, and underscores")
		}
	}
	return nil
}

func isValidAliasFormatBool(alias string) bool {
	return IsValidAliasFormat(alias) == nil
}

func (p *Project) HasAlias(alias string) bool {
	aliasLower := strings.ToLower(strings.TrimSpace(alias))
	for _, a := range p.Aliases {
		if strings.ToLower(a) == aliasLower {
			return true
		}
	}
	return false
}

func (p *Project) GetAliases() []string {
	if p.Aliases == nil {
		return []string{}
	}
	aliases := make([]string, len(p.Aliases))
	copy(aliases, p.Aliases)
	return aliases
}

func (p *Project) FormatAliases() string {
	if len(p.Aliases) == 0 {
		return ""
	}
	return strings.Join(p.Aliases, ", ")
}

func (p *Project) HasNotes() bool {
	return strings.TrimSpace(p.Notes) != ""
}
