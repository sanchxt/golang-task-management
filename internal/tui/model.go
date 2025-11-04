package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"task-management/internal/domain"
	"task-management/internal/repository"
)

type viewMode int

const (
	tableView viewMode = iota
	detailView
)

// main bubble tea model for the TUI
type Model struct {
	repo           repository.TaskRepository
	tasks          []*domain.Task
	table          table.Model
	keys           keyMap
	viewMode       viewMode
	selectedTask   *domain.Task
	err            error
	width          int
	height         int
	showHelp       bool
}

// creates a new TUI model
func NewModel(repo repository.TaskRepository, tasks []*domain.Task) Model {
	// table columns
	columns := []table.Column{
		{Title: "Status", Width: 15},
		{Title: "Priority", Width: 12},
		{Title: "Title", Width: 45},
		{Title: "Project", Width: 15},
		{Title: "Tags", Width: 20},
		{Title: "Due", Width: 12},
	}

	// table rows from tasks
	rows := make([]table.Row, len(tasks))
	for i, task := range tasks {
		rows[i] = taskToRow(task)
	}

	// table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	// table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	t.SetStyles(s)

	return Model{
		repo:     repo,
		tasks:    tasks,
		table:    t,
		keys:     defaultKeyMap(),
		viewMode: tableView,
	}
}

// initializes model
func (m Model) Init() tea.Cmd {
	return nil
}

// updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if m.viewMode == tableView && len(m.tasks) > 0 {
				// switch to detail view
				selectedRow := m.table.Cursor()
				if selectedRow < len(m.tasks) {
					m.selectedTask = m.tasks[selectedRow]
					m.viewMode = detailView
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Back):
			if m.viewMode == detailView {
				// back to table view
				m.viewMode = tableView
				m.selectedTask = nil
			}
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.viewMode == detailView {
				// prev task in detail view
				m.navigateToPreviousTask()
				return m, nil
			}

		case key.Matches(msg, m.keys.Down):
			if m.viewMode == detailView {
				// next task in detail view
				m.navigateToNextTask()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 10)
	}

	// update table if in table view
	if m.viewMode == tableView {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

// renders the UI
func (m Model) View() string {
	var b strings.Builder

	// title
	title := titleStyle.Render("  TaskFlow TUI  ")
	b.WriteString(title)
	b.WriteString("\n")

	switch m.viewMode {
	case tableView:
		b.WriteString(m.renderTableView())
	case detailView:
		b.WriteString(m.renderDetailView())
	}

	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// renders table view
func (m Model) renderTableView() string {
	var b strings.Builder

	subtitle := subtitleStyle.Render(fmt.Sprintf("Total: %d task(s)", len(m.tasks)))
	b.WriteString(subtitle)
	b.WriteString("\n\n")

	if len(m.tasks) == 0 {
		b.WriteString("No tasks found.\n")
	} else {
		b.WriteString(m.table.View())
	}

	return b.String()
}

// renders detail view
func (m Model) renderDetailView() string {
	if m.selectedTask == nil {
		return "No task selected."
	}

	task := m.selectedTask
	var b strings.Builder

	// task detail card
	content := []string{}

	// ID and title
	content = append(content, renderDetailRow("ID:", fmt.Sprintf("#%d", task.ID)))
	content = append(content, renderDetailRow("Title:", task.Title))

	// description
	if task.Description != "" {
		content = append(content, renderDetailRow("Description:", wrapText(task.Description, 60)))
	}

	// status
	statusStyle := getStatusStyle(string(task.Status))
	statusText := statusStyle.Render(string(task.Status))
	content = append(content, renderDetailRow("Status:", statusText))

	// priority
	priorityStyle := getPriorityStyle(string(task.Priority))
	priorityText := priorityStyle.Render(string(task.Priority))
	content = append(content, renderDetailRow("Priority:", priorityText))

	// project
	if task.Project != "" {
		content = append(content, renderDetailRow("Project:", task.Project))
	}

	// tags
	if len(task.Tags) > 0 {
		tagsText := strings.Join(task.Tags, ", ")
		content = append(content, renderDetailRow("Tags:", tagsText))
	}

	// due date
	if task.DueDate != nil {
		dueText := formatDetailDueDate(task.DueDate)
		content = append(content, renderDetailRow("Due Date:", dueText))
	}

	// timestamps
	content = append(content, renderDetailRow("Created:", task.CreatedAt.Format("2006-01-02 15:04:05")))
	content = append(content, renderDetailRow("Updated:", task.UpdatedAt.Format("2006-01-02 15:04:05")))

	cardContent := strings.Join(content, "\n")
	card := detailContainerStyle.Render(cardContent)

	b.WriteString(card)

	return b.String()
}

// help text
func (m Model) renderHelp() string {
	if m.showHelp {
		var help []string
		if m.viewMode == tableView {
			help = []string{
				"Keybindings:",
				"  ↑/k         Move up",
				"  ↓/j         Move down",
				"  Enter       View task details",
				"  q/Ctrl+C    Quit",
				"  ?           Toggle help",
			}
		} else {
			help = []string{
				"Keybindings:",
				"  ↑/k         Previous task",
				"  ↓/j         Next task",
				"  Esc         Back to list",
				"  q/Ctrl+C    Quit",
				"  ?           Toggle help",
			}
		}
		return helpStyle.Render(strings.Join(help, "\n"))
	}

	var hints []string
	if m.viewMode == tableView {
		hints = []string{
			"↑/↓: navigate",
			"Enter: view",
			"q: quit",
			"?: help",
		}
	} else {
		hints = []string{
			"↑/↓: prev/next task",
			"Esc: back",
			"q: quit",
			"?: help",
		}
	}
	return helpStyle.Render(strings.Join(hints, "  •  "))
}

// convert task -> table row
func taskToRow(task *domain.Task) table.Row {
	// status
	statusIcon := getStatusIcon(task.Status)
	status := fmt.Sprintf("%s %s", statusIcon, task.Status)

	// priority
	priorityIcon := getPriorityIcon(task.Priority)
	priority := fmt.Sprintf("%s %s", priorityIcon, task.Priority)

	// truncate title
	title := task.Title
	if len(title) > 37 {
		title = title[:37] + "..."
	}

	// project
	project := task.Project
	if project == "" {
		project = "-"
	}

	// tags
	tags := strings.Join(task.Tags, ", ")
	if tags == "" {
		tags = "-"
	}
	if len(tags) > 17 {
		tags = tags[:17] + "..."
	}

	// due date
	dueDate := "-"
	if task.DueDate != nil {
		dueDate = formatDueDate(task.DueDate)
	}

	return table.Row{
		status,
		priority,
		title,
		project,
		tags,
		dueDate,
	}
}

// detail row
func renderDetailRow(label, value string) string {
	return detailLabelStyle.Render(label) + " " + detailValueStyle.Render(value)
}

// wraps text
func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var wrapped []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				wrapped = append(wrapped, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		wrapped = append(wrapped, currentLine)
	}

	return strings.Join(wrapped, "\n"+strings.Repeat(" ", 16))
}

// formats a due date
func formatDetailDueDate(dueDate *time.Time) string {
	if dueDate == nil {
		return "-"
	}

	now := time.Now()
	diff := dueDate.Sub(now)

	dateStr := dueDate.Format("2006-01-02 (Mon)")

	if diff < 0 {
		days := int(-diff.Hours() / 24)
		return fmt.Sprintf("%s - OVERDUE by %d day(s)", dateStr, days)
	}

	days := int(diff.Hours() / 24)
	if days == 0 {
		return fmt.Sprintf("%s - DUE TODAY", dateStr)
	} else if days == 1 {
		return fmt.Sprintf("%s - Due tomorrow", dateStr)
	} else if days <= 7 {
		return fmt.Sprintf("%s - Due in %d days", dateStr, days)
	}

	return dateStr
}

// navigates to prev task
func (m *Model) navigateToPreviousTask() {
	if m.selectedTask == nil || len(m.tasks) == 0 {
		return
	}

	// find current task index
	currentIndex := -1
	for i, task := range m.tasks {
		if task.ID == m.selectedTask.ID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return
	}

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(m.tasks) - 1
	}

	m.selectedTask = m.tasks[prevIndex]
	m.table.SetCursor(prevIndex)
}

// navigates to next task
func (m *Model) navigateToNextTask() {
	if m.selectedTask == nil || len(m.tasks) == 0 {
		return
	}

	currentIndex := -1
	for i, task := range m.tasks {
		if task.ID == m.selectedTask.ID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return
	}

	nextIndex := currentIndex + 1
	if nextIndex >= len(m.tasks) {
		nextIndex = 0
	}

	m.selectedTask = m.tasks[nextIndex]
	m.table.SetCursor(nextIndex)
}

// helper functions
func getStatusIcon(status domain.Status) string {
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

func getPriorityIcon(priority domain.Priority) string {
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

func formatDueDate(dueDate *time.Time) string {
	if dueDate == nil {
		return "-"
	}

	now := time.Now()
	diff := dueDate.Sub(now)

	if diff < 0 {
		days := int(-diff.Hours() / 24)
		if days == 0 {
			return "TODAY!"
		}
		return fmt.Sprintf("-%dd", days)
	}

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
