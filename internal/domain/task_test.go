package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTask(t *testing.T) {
	title := "Test Task"
	task := NewTask(title)

	assert.Equal(t, title, task.Title)
	assert.Equal(t, PriorityMedium, task.Priority)
	assert.Equal(t, StatusPending, task.Status)
	assert.NotNil(t, task.Tags)
	assert.Empty(t, task.Tags)
	assert.False(t, task.CreatedAt.IsZero())
	assert.False(t, task.UpdatedAt.IsZero())
	assert.Nil(t, task.DueDate)
}

func TestTaskValidate(t *testing.T) {
	tests := []struct {
		name    string
		task    *Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid task",
			task: &Task{
				Title:    "Valid Task",
				Priority: PriorityHigh,
				Status:   StatusPending,
			},
			wantErr: false,
		},
		{
			name: "empty title",
			task: &Task{
				Title: "",
			},
			wantErr: true,
			errMsg:  "task title cannot be empty",
		},
		{
			name: "whitespace only title",
			task: &Task{
				Title: "   ",
			},
			wantErr: true,
			errMsg:  "task title cannot be empty",
		},
		{
			name: "title too long",
			task: &Task{
				Title: strings.Repeat("a", 201),
			},
			wantErr: true,
			errMsg:  "task title cannot exceed 200 characters",
		},
		{
			name: "description too long",
			task: &Task{
				Title:       "Valid Task",
				Description: strings.Repeat("a", 1001),
			},
			wantErr: true,
			errMsg:  "task description cannot exceed 1000 characters",
		},
		{
			name: "invalid priority",
			task: &Task{
				Title:    "Valid Task",
				Priority: Priority("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid priority",
		},
		{
			name: "invalid status",
			task: &Task{
				Title:  "Valid Task",
				Status: Status("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid status",
		},
		{
			name: "valid with all fields",
			task: &Task{
				Title:       "Complete Task",
				Description: "This is a complete task description",
				Priority:    PriorityUrgent,
				Status:      StatusInProgress,
				Tags:        []string{"tag1", "tag2"},
				Project:     "Project Alpha",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidPriority(t *testing.T) {
	tests := []struct {
		priority Priority
		want     bool
	}{
		{PriorityLow, true},
		{PriorityMedium, true},
		{PriorityHigh, true},
		{PriorityUrgent, true},
		{Priority("invalid"), false},
		{Priority(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			got := isValidPriority(tt.priority)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPending, true},
		{StatusInProgress, true},
		{StatusCompleted, true},
		{StatusCancelled, true},
		{Status("invalid"), false},
		{Status(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := isValidStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTaskWithDueDate(t *testing.T) {
	task := NewTask("Task with due date")
	dueDate := time.Now().Add(24 * time.Hour)
	task.DueDate = &dueDate

	err := task.Validate()
	assert.NoError(t, err)
	assert.NotNil(t, task.DueDate)
}
