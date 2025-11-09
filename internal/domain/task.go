package domain

import (
	"errors"
	"strings"
	"time"
)

// task priority
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
	PriorityUrgent Priority = "urgent"
)

// task status
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

type Task struct {
	ID          int64      `db:"id" json:"id"`
	Title       string     `db:"title" json:"title"`
	Description string     `db:"description" json:"description"`
	Priority    Priority   `db:"priority" json:"priority"`
	Status      Status     `db:"status" json:"status"`
	Tags        []string   `db:"tags" json:"tags"`
	ProjectID   *int64     `db:"project_id" json:"project_id,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	DueDate     *time.Time `db:"due_date" json:"due_date,omitempty"`

	ProjectName string `db:"-" json:"project_name,omitempty"`
}

func (t *Task) Validate() error {
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("task title cannot be empty")
	}

	if len(t.Title) > 200 {
		return errors.New("task title cannot exceed 200 characters")
	}

	if len(t.Description) > 1000 {
		return errors.New("task description cannot exceed 1000 characters")
	}

	if t.Priority != "" && !isValidPriority(t.Priority) {
		return errors.New("invalid priority: must be low, medium, high, or urgent")
	}

	if t.Status != "" && !isValidStatus(t.Status) {
		return errors.New("invalid status: must be pending, in_progress, completed, or cancelled")
	}

	return nil
}

// create a new task
func NewTask(title string) *Task {
	now := time.Now()
	return &Task{
		Title:     title,
		Priority:  PriorityMedium,
		Status:    StatusPending,
		Tags:      make([]string, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func isValidPriority(p Priority) bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}

func isValidStatus(s Status) bool {
	switch s {
	case StatusPending, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

// parses a date string in various formats
func ParseDueDate(dateStr string) (*time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("unable to parse date: " + dateStr)
}
