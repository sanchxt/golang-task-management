package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"task-management/internal/domain"
)

// renders the UI
func (m Model) View() string {
	if m.loading {
		return m.styles.TUITitle.Render("Loading...") + "\n"
	}

	var b strings.Builder

	// title
	title := m.styles.TUITitle.Render("  TaskFlow TUI  ")
	b.WriteString(title)
	b.WriteString("\n")

	// confirmation dialog takes precedence
	if m.confirm.active {
		b.WriteString("\n")
		b.WriteString(m.renderConfirmDialog())
		b.WriteString("\n")
		return b.String()
	}

	// edit form
	if m.editForm.active {
		b.WriteString(m.renderEditForm())
		b.WriteString("\n")
		b.WriteString(m.renderEditHelp())
		return b.String()
	}

	// search mode
	if m.uiMode == searchingMode {
		b.WriteString(m.renderSearchMode())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		b.WriteString("\n")
		b.WriteString(m.renderHelp())
		return b.String()
	}

	// filter mode
	if m.uiMode == filteringMode {
		b.WriteString(m.renderFilterPanel())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		b.WriteString("\n")
		b.WriteString(m.renderHelp())
		return b.String()
	}

	// normal mode
	switch m.viewMode {
	case tableView:
		b.WriteString(m.renderTableView())
	case detailView:
		b.WriteString(m.renderDetailView())
	case editView:
		b.WriteString(m.renderEditForm())
	}

	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// renders table view
func (m Model) renderTableView() string {
	var b strings.Builder

	// filter summary
	if m.hasActiveFilters() {
		b.WriteString(m.renderFilterSummary())
		b.WriteString("\n")
	}

	// message (success/error)
	if m.message != "" {
		b.WriteString(m.styles.Success.Render(m.message))
		b.WriteString("\n\n")
	}
	if m.err != nil {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if len(m.tasks) == 0 {
		if m.hasActiveFilters() {
			b.WriteString(m.styles.Info.Render("No tasks found matching the filters."))
		} else {
			b.WriteString(m.styles.Info.Render("No tasks found."))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(m.table.View())
	}

	return b.String()
}

// renders detail view
func (m Model) renderDetailView() string {
	if m.selectedTask == nil {
		return m.styles.Info.Render("No task selected.")
	}

	task := m.selectedTask
	var b strings.Builder

	// message (success/error)
	if m.message != "" {
		b.WriteString(m.styles.Success.Render(m.message))
		b.WriteString("\n\n")
	}
	if m.err != nil {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	// task detail card
	content := []string{}

	// ID and title
	content = append(content, m.renderDetailRow("ID:", fmt.Sprintf("#%d", task.ID)))
	content = append(content, m.renderDetailRow("Title:", task.Title))

	// description
	if task.Description != "" {
		content = append(content, m.renderDetailRow("Description:", wrapText(task.Description, 60)))
	}

	// status
	statusStyle := m.styles.GetStatusStyle(task.Status)
	statusText := statusStyle.Render(string(task.Status))
	content = append(content, m.renderDetailRow("Status:", statusText))

	// priority
	priorityStyle := m.styles.GetPriorityTextStyle(task.Priority)
	priorityText := priorityStyle.Render(string(task.Priority))
	content = append(content, m.renderDetailRow("Priority:", priorityText))

	// project
	if task.Project != "" {
		content = append(content, m.renderDetailRow("Project:", task.Project))
	}

	// tags
	if len(task.Tags) > 0 {
		tagsText := strings.Join(task.Tags, ", ")
		content = append(content, m.renderDetailRow("Tags:", tagsText))
	}

	// due date
	if task.DueDate != nil {
		dueText := formatDetailDueDate(task.DueDate)
		content = append(content, m.renderDetailRow("Due Date:", dueText))
	}

	// timestamps
	content = append(content, m.renderDetailRow("Created:", task.CreatedAt.Format("2006-01-02 15:04:05")))
	content = append(content, m.renderDetailRow("Updated:", task.UpdatedAt.Format("2006-01-02 15:04:05")))

	cardContent := strings.Join(content, "\n")
	card := m.styles.DetailContainer.Render(cardContent)

	b.WriteString(card)

	return b.String()
}

// renders search mode UI
func (m Model) renderSearchMode() string {
	var b strings.Builder

	label := m.styles.TUISubtitle.Render("Search Tasks:")
	b.WriteString(label)
	b.WriteString("\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	hint := m.styles.TUIHelp.Render("Tip: Use 're:' prefix for regex search (e.g., 're:bug.*urgent')")
	b.WriteString(hint)

	return b.String()
}

// renders filter panel
func (m Model) renderFilterPanel() string {
	var b strings.Builder

	label := m.styles.TUISubtitle.Render("Filter Options:")
	b.WriteString(label)
	b.WriteString("\n\n")

	for i, item := range m.filterPanel.items {
		if item.label == "" {
			b.WriteString("\n")
			continue
		}

		line := item.label

		// check if this filter is currently active
		isActive := false
		if item.filterType == "status" && item.value == string(m.filter.Status) {
			isActive = true
		} else if item.filterType == "priority" && item.value == string(m.filter.Priority) {
			isActive = true
		} else if item.filterType == "duedate" {
			// check date filter match
			isActive = m.isDateFilterActive(item.value)
		}

		// highlight selected item
		if i == m.filterPanel.selectedItem {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.SelectedFg)).
				Background(lipgloss.Color(m.theme.SelectedBg)).
				Bold(true).
				Render(" " + line + " ")
		} else if isActive {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.Success)).
				Render(line + " ✓")
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// checks if a date filter is currently active
func (m Model) isDateFilterActive(value string) bool {
	// no date filter
	if m.filter.DueDateFrom == nil && m.filter.DueDateTo == nil {
		return value == ""
	}

	// special case: no due date filter
	if m.filter.DueDateFrom != nil && *m.filter.DueDateFrom == "none" {
		return value == "none"
	}

	// calculate expected date ranges for each filter type
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	switch value {
	case "overdue":
		yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
		return m.filter.DueDateFrom == nil && m.filter.DueDateTo != nil && *m.filter.DueDateTo == yesterday
	case "today":
		todayStr := today.Format("2006-01-02")
		tomorrowStr := today.AddDate(0, 0, 1).Format("2006-01-02")
		return m.filter.DueDateFrom != nil && *m.filter.DueDateFrom == todayStr &&
			m.filter.DueDateTo != nil && *m.filter.DueDateTo == tomorrowStr
	case "week":
		todayStr := today.Format("2006-01-02")
		weekStr := today.AddDate(0, 0, 7).Format("2006-01-02")
		return m.filter.DueDateFrom != nil && *m.filter.DueDateFrom == todayStr &&
			m.filter.DueDateTo != nil && *m.filter.DueDateTo == weekStr
	case "month":
		todayStr := today.Format("2006-01-02")
		monthStr := today.AddDate(0, 0, 30).Format("2006-01-02")
		return m.filter.DueDateFrom != nil && *m.filter.DueDateFrom == todayStr &&
			m.filter.DueDateTo != nil && *m.filter.DueDateTo == monthStr
	}

	return false
}

// renders confirmation dialog
func (m Model) renderConfirmDialog() string {
	var b strings.Builder

	// message
	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Warning)).
		Bold(true).
		Render(m.confirm.message)

	// prompt
	prompt := m.styles.TUISubtitle.Render("Are you sure? (y/n)")

	// container
	content := lipgloss.JoinVertical(lipgloss.Left, message, "", prompt)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.Warning)).
		Padding(1, 2).
		Render(content)

	b.WriteString(box)

	return b.String()
}

// renders status bar
func (m Model) renderStatusBar() string {
	var items []string

	// multi-select mode indicator
	if m.multiSelect.enabled {
		selectedCount := len(m.multiSelect.selectedTasks)
		multiInfo := m.styles.Success.Render(fmt.Sprintf("✓ Multi-select: %d selected", selectedCount))
		items = append(items, multiInfo)
	}

	// total and page info
	if m.totalCount > 0 {
		totalPages := m.calculateTotalPages()
		if totalPages > 1 {
			startIdx := (m.currentPage-1)*m.pageSize + 1
			endIdx := startIdx + len(m.tasks) - 1
			pageInfo := fmt.Sprintf("Page %d/%d (%d-%d of %d)",
				m.currentPage, totalPages, startIdx, endIdx, m.totalCount)
			items = append(items, pageInfo)
		} else {
			items = append(items, fmt.Sprintf("Total: %d task(s)", m.totalCount))
		}
	}

	// sort info
	sortIcon := "↓"
	if m.filter.SortOrder == "asc" {
		sortIcon = "↑"
	}
	sortInfo := fmt.Sprintf("Sort: %s %s", m.filter.SortBy, sortIcon)
	items = append(items, sortInfo)

	// active filters count
	filterCount := m.countActiveFilters()
	if filterCount > 0 {
		items = append(items, fmt.Sprintf("Filters: %d active", filterCount))
	}

	statusText := strings.Join(items, " • ")
	return m.styles.TUISubtitle.Render(statusText)
}

// renders filter summary (compact)
func (m Model) renderFilterSummary() string {
	var filters []string

	if m.filter.Status != "" {
		filters = append(filters, fmt.Sprintf("Status: %s", m.filter.Status))
	}
	if m.filter.Priority != "" {
		filters = append(filters, fmt.Sprintf("Priority: %s", m.filter.Priority))
	}
	if m.filter.Project != "" {
		filters = append(filters, fmt.Sprintf("Project: %s", m.filter.Project))
	}
	if len(m.filter.Tags) > 0 {
		filters = append(filters, fmt.Sprintf("Tags: %s", strings.Join(m.filter.Tags, ", ")))
	}
	if m.filter.SearchQuery != "" {
		searchLabel := "Search"
		if m.filter.SearchMode == "regex" {
			searchLabel = "Search (regex)"
		}
		filters = append(filters, fmt.Sprintf("%s: %s", searchLabel, m.filter.SearchQuery))
	}

	summary := strings.Join(filters, " | ")
	return m.styles.Info.Render("Active filters: " + summary)
}

// help text
func (m Model) renderHelp() string {
	if m.showHelp {
		return m.renderFullHelp()
	}

	return m.renderQuickHelp()
}

// renders full help screen
func (m Model) renderFullHelp() string {
	var help []string

	if m.uiMode == searchingMode {
		help = []string{
			"Search Mode:",
			"  Type to search",
			"  Enter       Apply search",
			"  Esc         Cancel",
		}
	} else if m.uiMode == filteringMode {
		help = []string{
			"Filter Mode:",
			"  ↑/k         Move up",
			"  ↓/j         Move down",
			"  Enter       Select filter",
			"  Esc         Cancel",
		}
	} else if m.viewMode == tableView {
		help = []string{
			"Table View:",
			"  ↑/k         Move up",
			"  ↓/j         Move down",
			"  Enter       View details",
			"  n           New task",
			"  e           Edit task",
			"  f           Open filters",
			"  F           Clear filters",
			"  /           Search",
			"  s           Cycle sort",
			"  S           Toggle sort order",
			"  [/]         Prev/Next page",
			"  r           Refresh",
			"",
			"Quick Actions:",
			"  c           Mark complete",
			"  p           Cycle priority",
			"  x           Toggle status",
			"  d           Delete task",
			"",
			"Multi-select:",
			"  v           Toggle multi-select mode",
			"  Space       Toggle selection",
			"  Ctrl+A      Select all",
			"  Ctrl+D      Deselect all",
			"",
			"General:",
			"  q/Ctrl+C    Quit",
			"  ?           Toggle help",
		}
	} else {
		help = []string{
			"Detail View:",
			"  ↑/k         Previous task",
			"  ↓/j         Next task",
			"  Esc         Back to list",
			"  e           Edit task",
			"",
			"Quick Actions:",
			"  c           Mark complete",
			"  p           Cycle priority",
			"  x           Toggle status",
			"  d           Delete task",
			"",
			"General:",
			"  q/Ctrl+C    Quit",
			"  ?           Toggle help",
		}
	}

	return m.styles.TUIHelp.Render(strings.Join(help, "\n"))
}

// renders quick help hints
func (m Model) renderQuickHelp() string {
	var hints []string

	if m.uiMode == searchingMode {
		hints = []string{"Type: search", "Enter: apply", "Esc: cancel", "?: help"}
	} else if m.uiMode == filteringMode {
		hints = []string{"↑/↓: navigate", "Enter: select", "Esc: cancel", "?: help"}
	} else if m.viewMode == tableView {
		if m.multiSelect.enabled {
			hints = []string{
				"Space: toggle",
				fmt.Sprintf("Selected: %d", len(m.multiSelect.selectedTasks)),
				"v: exit multi-select",
				"c/p/x/d: bulk ops",
				"?: help",
			}
		} else {
			hints = []string{
				"↑/↓: navigate",
				"n: new",
				"e: edit",
				"v: multi-select",
				"f: filter",
				"/: search",
				"c/p/x/d: actions",
				"?: help",
			}
		}
	} else {
		hints = []string{
			"↑/↓: prev/next",
			"e: edit",
			"Esc: back",
			"c/p/x/d: actions",
			"?: help",
		}
	}

	return m.styles.TUIHelp.Render(strings.Join(hints, "  •  "))
}

// detail row
func (m Model) renderDetailRow(label, value string) string {
	return m.styles.DetailLabel.Render(label) + " " + m.styles.DetailValue.Render(value)
}

// Helper methods

func (m *Model) hasActiveFilters() bool {
	return m.filter.Status != "" ||
		m.filter.Priority != "" ||
		m.filter.Project != "" ||
		len(m.filter.Tags) > 0 ||
		m.filter.SearchQuery != ""
}

func (m *Model) countActiveFilters() int {
	count := 0
	if m.filter.Status != "" {
		count++
	}
	if m.filter.Priority != "" {
		count++
	}
	if m.filter.Project != "" {
		count++
	}
	if len(m.filter.Tags) > 0 {
		count++
	}
	if m.filter.SearchQuery != "" {
		count++
	}
	return count
}

// Edit form rendering

func (m Model) renderEditForm() string {
	var b strings.Builder

	// title
	formTitle := "Edit Task"
	if m.editForm.isNewTask {
		formTitle = "New Task"
	}
	b.WriteString(m.styles.TUISubtitle.Render(formTitle))
	b.WriteString("\n\n")

	// error message
	if m.editForm.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.editForm.err))
		b.WriteString("\n\n")
	}

	// form fields
	priorities := []string{"low", "medium", "high", "urgent"}
	statuses := []string{"pending", "in_progress", "completed", "cancelled"}

	// Title field
	fieldLabel := "Title:"
	if m.editForm.focusedField == 0 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.editForm.titleInput.View())
	b.WriteString("\n\n")

	// Description field
	fieldLabel = "Description:"
	if m.editForm.focusedField == 1 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.editForm.descInput.View())
	b.WriteString("\n\n")

	// Project field
	fieldLabel = "Project:"
	if m.editForm.focusedField == 2 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.editForm.projectInput.View())
	b.WriteString("\n\n")

	// Tags field
	fieldLabel = "Tags:"
	if m.editForm.focusedField == 3 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.editForm.tagsInput.View())
	b.WriteString("\n\n")

	// Due Date field
	fieldLabel = "Due Date:"
	if m.editForm.focusedField == 4 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.editForm.dueDateInput.View())
	b.WriteString("\n\n")

	// Priority field (selector)
	b.WriteString(m.styles.DetailLabel.Render("  Priority:"))
	b.WriteString(" ")
	priorityValue := domain.Priority(priorities[m.editForm.priorityIdx])
	priorityStyle := m.styles.GetPriorityTextStyle(priorityValue)
	b.WriteString(priorityStyle.Render(priorities[m.editForm.priorityIdx]))
	b.WriteString(m.styles.TUIHelp.Render(" (Ctrl+P to cycle)"))
	b.WriteString("\n\n")

	// Status field (selector)
	b.WriteString(m.styles.DetailLabel.Render("  Status:"))
	b.WriteString(" ")
	statusValue := domain.Status(statuses[m.editForm.statusIdx])
	statusStyle := m.styles.GetStatusStyle(statusValue)
	b.WriteString(statusStyle.Render(statuses[m.editForm.statusIdx]))
	b.WriteString(m.styles.TUIHelp.Render(" (Ctrl+T to cycle)"))
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderEditHelp() string {
	hints := []string{
		"Tab/Shift+Tab: navigate fields",
		"Ctrl+S: save",
		"Ctrl+P: cycle priority",
		"Ctrl+T: cycle status",
		"Esc: cancel",
	}
	return m.styles.TUIHelp.Render(strings.Join(hints, "  •  "))
}
