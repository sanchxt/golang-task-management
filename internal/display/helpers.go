package display

import (
	"fmt"
	"time"

	"task-management/internal/domain"
)

func GetStatusIcon(status domain.Status) string {
	switch status {
	case domain.StatusCompleted:
		return "✓"
	case domain.StatusInProgress:
		return "⚡"
	case domain.StatusPending:
		return "○"
	case domain.StatusCancelled:
		return "✗"
	default:
		return "?"
	}
}

func GetPriorityIcon(priority domain.Priority) string {
	switch priority {
	case domain.PriorityUrgent:
		return "🔥"
	case domain.PriorityHigh:
		return "⬆"
	case domain.PriorityMedium:
		return "➡"
	case domain.PriorityLow:
		return "⬇"
	default:
		return "?"
	}
}

func FormatDueDate(dueDate *time.Time) string {
	if dueDate == nil {
		return "-"
	}

	now := time.Now()
	diff := dueDate.Sub(now)

	// overdue
	if diff < 0 {
		days := int(-diff.Hours() / 24)
		if days == 0 {
			return "TODAY!"
		}
		return fmt.Sprintf("-%dd", days)
	}

	// due soon
	days := int(diff.Hours() / 24)
	if days == 0 {
		return "Today"
	} else if days == 1 {
		return "Tomorrow"
	} else if days <= 7 {
		return fmt.Sprintf("%dd", days)
	}

	return dueDate.Format("2006-01-02")
}
