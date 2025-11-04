package tui

import "github.com/charmbracelet/lipgloss"

// styles
var (
	// general
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(1, 0)

	// table
	normalRowStyle = lipgloss.NewStyle()

	// detail view
	detailContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(1, 2).
				MarginTop(1).
				MarginBottom(1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true).
				Width(15)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA"))

	// priority
	urgentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	highStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8800"))

	mediumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0088FF"))

	lowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	// status
	completedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	inProgressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	pendingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	cancelledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Strikethrough(true)
)

func getPriorityStyle(priority string) lipgloss.Style {
	switch priority {
	case "urgent":
		return urgentStyle
	case "high":
		return highStyle
	case "medium":
		return mediumStyle
	case "low":
		return lowStyle
	default:
		return normalRowStyle
	}
}

func getStatusStyle(status string) lipgloss.Style {
	switch status {
	case "completed":
		return completedStyle
	case "in_progress":
		return inProgressStyle
	case "pending":
		return pendingStyle
	case "cancelled":
		return cancelledStyle
	default:
		return normalRowStyle
	}
}
