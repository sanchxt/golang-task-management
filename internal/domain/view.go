package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type SavedViewFilter struct {
	Status       Status    `json:"status,omitempty"`
	Priority     Priority  `json:"priority,omitempty"`
	ProjectID    *int64    `json:"project_id,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	SearchQuery  string    `json:"search_query,omitempty"`
	SearchMode   string    `json:"search_mode,omitempty"`
	SortBy       string    `json:"sort_by,omitempty"`
	SortOrder    string    `json:"sort_order,omitempty"`
	DueDateFrom  *string   `json:"due_date_from,omitempty"`
	DueDateTo    *string   `json:"due_date_to,omitempty"`
}

type SavedView struct {
	ID           int64            `db:"id" json:"id"`
	Name         string           `db:"name" json:"name"`
	Description  string           `db:"description" json:"description"`
	FilterConfig SavedViewFilter  `db:"filter_config" json:"filter_config"`
	IsFavorite   bool             `db:"is_favorite" json:"is_favorite"`
	HotKey       *int             `db:"hot_key" json:"hot_key,omitempty"`
	LastAccessed *time.Time       `db:"last_accessed" json:"last_accessed,omitempty"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updated_at"`

	TaskCount int `db:"-" json:"task_count,omitempty"`
}

func (v *SavedView) Validate() error {
	if strings.TrimSpace(v.Name) == "" {
		return errors.New("view name cannot be empty")
	}

	if len(v.Name) > 100 {
		return errors.New("view name cannot exceed 100 characters")
	}

	if len(v.Description) > 500 {
		return errors.New("view description cannot exceed 500 characters")
	}

	if v.HotKey != nil {
		if *v.HotKey < 1 || *v.HotKey > 9 {
			return errors.New("hot key must be between 1 and 9")
		}
	}

	return nil
}

func NewSavedView(name string) *SavedView {
	now := time.Now()
	return &SavedView{
		Name:         name,
		FilterConfig: SavedViewFilter{},
		IsFavorite:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (v *SavedView) HasFilter() bool {
	return v.FilterConfig.Status != "" ||
		v.FilterConfig.Priority != "" ||
		v.FilterConfig.ProjectID != nil ||
		len(v.FilterConfig.Tags) > 0 ||
		v.FilterConfig.SearchQuery != "" ||
		v.FilterConfig.DueDateFrom != nil ||
		v.FilterConfig.DueDateTo != nil
}

func (v *SavedView) GetFilterSummary() string {
	var parts []string

	if v.FilterConfig.Status != "" {
		parts = append(parts, "status:"+string(v.FilterConfig.Status))
	}
	if v.FilterConfig.Priority != "" {
		parts = append(parts, "priority:"+string(v.FilterConfig.Priority))
	}
	if v.FilterConfig.ProjectID != nil {
		parts = append(parts, "project filtered")
	}
	if len(v.FilterConfig.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("%d tags", len(v.FilterConfig.Tags)))
	}
	if v.FilterConfig.SearchQuery != "" {
		parts = append(parts, "search: "+v.FilterConfig.SearchQuery)
	}

	if len(parts) == 0 {
		return "no filters"
	}

	return strings.Join(parts, ", ")
}

func (v *SavedView) GetHotKeyDisplay() string {
	if v.HotKey == nil {
		return ""
	}
	return fmt.Sprintf("[%d]", *v.HotKey)
}

func (v *SavedView) GetFavoriteIndicator() string {
	if v.IsFavorite {
		return "★"
	}
	return ""
}
