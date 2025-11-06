package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
		hints = []string{
			"↑/↓: navigate",
			"Enter: view",
			"f: filter",
			"/: search",
			"s: sort",
			"c: complete",
			"p: priority",
			"d: delete",
			"q: quit",
			"?: help",
		}
	} else {
		hints = []string{
			"↑/↓: prev/next",
			"Esc: back",
			"c: complete",
			"p: priority",
			"d: delete",
			"q: quit",
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
