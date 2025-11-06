package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"task-management/internal/display"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/theme"
)

type viewMode int

const (
	tableView viewMode = iota
	detailView
	filterView
	searchView
	confirmView
)

// UI mode for different interactions
type uiMode int

const (
	normalMode uiMode = iota
	filteringMode
	searchingMode
	confirmingMode
)

// confirmation dialog state
type confirmDialog struct {
	message   string
	onConfirm func(m *Model) tea.Cmd
	active    bool
}

// main bubble tea model for the TUI
type Model struct {
	// repository and data
	repo         repository.TaskRepository
	tasks        []*domain.Task
	totalCount   int64

	// filter and pagination state
	filter       repository.TaskFilter
	currentPage  int
	pageSize     int

	// UI components
	table        table.Model
	searchInput  textinput.Model
	keys         keyMap

	// view state
	viewMode     viewMode
	uiMode       uiMode
	selectedTask *domain.Task

	// filter panel state
	filterPanel  filterPanel

	// confirmation dialog
	confirm      confirmDialog

	// UI state
	err          error
	width        int
	height       int
	showHelp     bool
	loading      bool
	message      string

	// theme
	theme        *theme.Theme
	styles       *theme.Styles

	// context
	ctx          context.Context
}

// filter panel for interactive filtering
type filterPanel struct {
	active       bool
	selectedItem int
	items        []filterItem
	tempStatus   string
	tempPriority string
	tempProject  string
	tempTags     []string
}

type filterItem struct {
	label       string
	value       string
	filterType  string // "status", "priority", "project", "tags", "clear"
}

// creates a new TUI model
func NewModel(repo repository.TaskRepository, initialFilter repository.TaskFilter, pageSize int, themeObj *theme.Theme, styles *theme.Styles) Model {
	// table columns
	columns := []table.Column{
		{Title: "Status", Width: 15},
		{Title: "Priority", Width: 12},
		{Title: "Title", Width: 45},
		{Title: "Project", Width: 15},
		{Title: "Tags", Width: 20},
		{Title: "Due", Width: 12},
	}

	// table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	// table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(themeObj.BorderColor)).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(themeObj.SelectedFg)).
		Background(lipgloss.Color(themeObj.SelectedBg)).
		Bold(true)
	t.SetStyles(s)

	// search input
	si := textinput.New()
	si.Placeholder = "Search tasks..."
	si.CharLimit = 100
	si.Width = 50

	// initialize filter with defaults
	if initialFilter.SortBy == "" {
		initialFilter.SortBy = "created_at"
	}
	if initialFilter.SortOrder == "" {
		initialFilter.SortOrder = "desc"
	}

	// set page size
	if pageSize == 0 {
		pageSize = 20
	}

	return Model{
		repo:        repo,
		tasks:       []*domain.Task{},
		filter:      initialFilter,
		currentPage: 1,
		pageSize:    pageSize,
		table:       t,
		searchInput: si,
		keys:        defaultKeyMap(),
		viewMode:    tableView,
		uiMode:      normalMode,
		theme:       themeObj,
		styles:      styles,
		ctx:         context.Background(),
	}
}

// initializes model - fetches initial tasks
func (m Model) Init() tea.Cmd {
	return fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize)
}

// convert task -> table row
func taskToRow(task *domain.Task) table.Row {
	// status
	statusIcon := display.GetStatusIcon(task.Status)
	status := fmt.Sprintf("%s %s", statusIcon, task.Status)

	// priority
	priorityIcon := display.GetPriorityIcon(task.Priority)
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
		dueDate = display.FormatDueDate(task.DueDate)
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

// wraps text for display
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

// formats a due date for detail view
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
