package export

import (
	"task-management/internal/domain"
	"time"
)

type ProjectExport struct {
	Version string           `json:"version"`
	Project *ProjectData     `json:"project"`
}

type ProjectData struct {
	ID          int64           `json:"id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	ParentID    *int64          `json:"parent_id,omitempty"`
	Color       string          `json:"color,omitempty"`
	Icon        string          `json:"icon,omitempty"`
	Status      string          `json:"status"`
	IsFavorite  bool            `json:"is_favorite"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Tasks       []*TaskData     `json:"tasks,omitempty"`
	Children    []*ProjectData  `json:"children,omitempty"`
}

type TaskData struct {
	ID          int64     `json:"id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority"`
	Status      string    `json:"status"`
	Tags        []string  `json:"tags,omitempty"`
	DueDate     *string   `json:"due_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BackupData struct {
	Version   string                   `json:"version"`
	Timestamp time.Time                `json:"timestamp"`
	Projects  []*ProjectData           `json:"projects"`
	Tasks     []*TaskData              `json:"tasks"`
	Templates []*domain.ProjectTemplate `json:"templates,omitempty"`
	Views     []*domain.SavedView       `json:"views,omitempty"`
}

type ConflictStrategy string

const (
	ConflictStrategyMerge     ConflictStrategy = "merge"
	ConflictStrategySkip      ConflictStrategy = "skip"
	ConflictStrategyOverwrite ConflictStrategy = "overwrite"
)

type ExportFormat string

const (
	FormatJSON     ExportFormat = "json"
	FormatCSV      ExportFormat = "csv"
	FormatMarkdown ExportFormat = "markdown"
)

type TaskCSVRow struct {
	ID          string
	Title       string
	Description string
	Priority    string
	Status      string
	Tags        string
	ProjectName string
	DueDate     string
	CreatedAt   string
	UpdatedAt   string
}

type ProjectCSVRow struct {
	ID          string
	Name        string
	Description string
	ParentPath  string
	Status      string
	Color       string
	Icon        string
	TaskCount   string
	CreatedAt   string
	UpdatedAt   string
}
