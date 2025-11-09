package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"task-management/internal/domain"
	"task-management/internal/query"
)

func (m Model) View() string {
	if m.loading {
		return m.styles.TUITitle.Render("Loading...") + "\n"
	}

	var b strings.Builder

	title := m.styles.TUITitle.Render("  TaskFlow TUI  ")
	b.WriteString(title)
	b.WriteString("\n")

	if m.confirm.active {
		b.WriteString("\n")
		b.WriteString(m.renderConfirmDialog())
		b.WriteString("\n")
		return b.String()
	}

	if m.projectPicker.active {
		b.WriteString("\n")
		b.WriteString(m.renderProjectPicker())
		b.WriteString("\n")
		return b.String()
	}

	if m.viewPicker.active {
		b.WriteString("\n")
		b.WriteString(m.renderViewPicker())
		b.WriteString("\n")
		return b.String()
	}

	if m.notesViewer.active {
		b.WriteString("\n")
		b.WriteString(m.renderNotesViewer())
		b.WriteString("\n")
		return b.String()
	}

	if m.editForm.active {
		b.WriteString(m.renderEditForm())
		b.WriteString("\n")
		b.WriteString(m.renderEditHelp())
		return b.String()
	}

	if m.projectForm.active {
		b.WriteString(m.renderProjectForm())
		b.WriteString("\n")
		b.WriteString(m.renderProjectFormHelp())
		return b.String()
	}

	if m.uiMode == searchingMode {
		b.WriteString(m.renderSearchMode())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		b.WriteString("\n")
		b.WriteString(m.renderHelp())

		if m.showQueryHelp {
			b.WriteString("\n\n")
			b.WriteString(m.renderQueryHelpModal())
		}

		return b.String()
	}

	if m.uiMode == filteringMode {
		b.WriteString(m.renderFilterPanel())
		b.WriteString("\n")
		b.WriteString(m.renderStatusBar())
		b.WriteString("\n")
		b.WriteString(m.renderHelp())
		return b.String()
	}

	switch m.viewMode {
	case tableView:
		b.WriteString(m.renderTableView())
	case detailView:
		b.WriteString(m.renderDetailView())
	case editView:
		b.WriteString(m.renderEditForm())
	case projectView:
		b.WriteString(m.renderProjectView())
	}

	b.WriteString("\n")

	if (m.viewMode == tableView || m.viewMode == detailView) && len(m.quickAccessViews) > 0 {
		b.WriteString(m.renderQuickAccessWidget())
		b.WriteString("\n")
	}

	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	if m.showQueryHelp {
		b.WriteString("\n\n")
		b.WriteString(m.renderQueryHelpModal())
	}

	return b.String()
}

func (m Model) renderTableView() string {
	var b strings.Builder

	if queryIndicator := m.renderQueryModeIndicator(); queryIndicator != "" {
		b.WriteString(queryIndicator)
		b.WriteString("\n")
	}

	if m.hasActiveFilters() {
		b.WriteString(m.renderFilterSummary())
		b.WriteString("\n")
	}

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

func (m Model) renderDetailView() string {
	if m.selectedTask == nil {
		return m.styles.Info.Render("No task selected.")
	}

	task := m.selectedTask
	var b strings.Builder

	if m.message != "" {
		b.WriteString(m.styles.Success.Render(m.message))
		b.WriteString("\n\n")
	}
	if m.err != nil {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if breadcrumb := m.buildBreadcrumb(task); breadcrumb != "" {
		breadcrumbStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Info)).
			Italic(true)
		b.WriteString(breadcrumbStyle.Render("📍 " + breadcrumb))
		b.WriteString("\n\n")
	}

	content := []string{}

	content = append(content, m.renderDetailRow("ID:", fmt.Sprintf("#%d", task.ID)))
	content = append(content, m.renderDetailRow("Title:", task.Title))

	if task.Description != "" {
		content = append(content, m.renderDetailRow("Description:", wrapText(task.Description, 60)))
	}

	statusStyle := m.styles.GetStatusStyle(task.Status)
	statusText := statusStyle.Render(string(task.Status))
	content = append(content, m.renderDetailRow("Status:", statusText))

	priorityStyle := m.styles.GetPriorityTextStyle(task.Priority)
	priorityText := priorityStyle.Render(string(task.Priority))
	content = append(content, m.renderDetailRow("Priority:", priorityText))

	if task.ProjectName != "" {
		content = append(content, m.renderDetailRow("Project:", task.ProjectName))
	}

	if len(task.Tags) > 0 {
		tagsText := strings.Join(task.Tags, ", ")
		content = append(content, m.renderDetailRow("Tags:", tagsText))
	}

	if task.DueDate != nil {
		dueText := formatDetailDueDate(task.DueDate)
		content = append(content, m.renderDetailRow("Due Date:", dueText))
	}

	content = append(content, m.renderDetailRow("Created:", task.CreatedAt.Format("2006-01-02 15:04:05")))
	content = append(content, m.renderDetailRow("Updated:", task.UpdatedAt.Format("2006-01-02 15:04:05")))

	cardContent := strings.Join(content, "\n")
	card := m.styles.DetailContainer.Render(cardContent)

	b.WriteString(card)

	return b.String()
}

func (m Model) renderSearchMode() string {
	var b strings.Builder

	label := m.styles.TUISubtitle.Render("Search Tasks:")
	b.WriteString(label)
	b.WriteString("\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n")

	if m.historyDropdown.active && len(m.searchHistory) > 0 {
		b.WriteString(m.renderSearchHistoryDropdown())
	}
	b.WriteString("\n")

	searchQuery := m.searchInput.Value()
	if searchQuery != "" {
		parsedQuery, err := query.ParseProjectMentions(searchQuery)
		if err == nil && parsedQuery.HasProjectFilter() {
			mentionsStr := "Project filters: "
			mentionParts := make([]string, 0, len(parsedQuery.ProjectMentions))
			for _, mention := range parsedQuery.ProjectMentions {
				if mention.Fuzzy {
					mentionParts = append(mentionParts, m.styles.Success.Render("@~"+mention.Name))
				} else {
					mentionParts = append(mentionParts, m.styles.Info.Render("@"+mention.Name))
				}
			}
			mentionsStr += strings.Join(mentionParts, " ")
			b.WriteString(mentionsStr)
			b.WriteString("\n")
		}
	}

	if m.fuzzyMode {
		fuzzyStatus := m.styles.Success.Render(fmt.Sprintf("● Fuzzy Mode ON (threshold: %d)", m.fuzzyThreshold))
		b.WriteString(fuzzyStatus)
		b.WriteString("\n")
	} else {
		fuzzyStatus := m.styles.TUIHelp.Render("○ Fuzzy Mode OFF")
		b.WriteString(fuzzyStatus)
		b.WriteString("\n")
	}

	hint := m.styles.TUIHelp.Render("Tip: Press 'f' for fuzzy • Use 're:' for regex • Use '@project' to filter by project")
	b.WriteString(hint)

	return b.String()
}

func (m Model) renderSearchHistoryDropdown() string {
	var b strings.Builder

	header := m.styles.TUISubtitle.Render("Recent Searches:")
	b.WriteString(header)
	b.WriteString("\n")

	maxItems := m.historyDropdown.height
	if maxItems > len(m.searchHistory) {
		maxItems = len(m.searchHistory)
	}

	for i := 0; i < maxItems; i++ {
		entry := m.searchHistory[i]

		var line string
		if i == m.historyDropdown.cursor {
			displayText := fmt.Sprintf("▶ %s %s  %s",
				entry.GetModeIndicator(),
				entry.QueryText,
				m.styles.TUIHelp.Render(entry.GetRelativeTime()))
			line = m.styles.Info.Bold(true).Render(displayText)
		} else {
			displayText := fmt.Sprintf("  %s %s  %s",
				entry.GetModeIndicator(),
				entry.QueryText,
				m.styles.TUIHelp.Render(entry.GetRelativeTime()))
			line = displayText
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	footer := m.styles.TUIHelp.Render("↑↓ navigate • Enter select • Esc close")
	b.WriteString(footer)
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderQueryModeIndicator() string {
	if !m.queryMode || m.queryString == "" {
		return ""
	}

	var b strings.Builder

	indicator := m.styles.Success.Render("🔍 Query Language Mode")
	b.WriteString(indicator)
	b.WriteString("\n")

	queryDisplay := m.styles.Info.Render(fmt.Sprintf("Active Query: %s", m.queryString))
	b.WriteString(queryDisplay)
	b.WriteString("\n")

	helpHint := m.styles.TUIHelp.Render("Press '?' for query syntax help")
	b.WriteString(helpHint)
	b.WriteString("\n")

	return b.String()
}

func getQueryHelpContent() string {
	content := `
Query Language Syntax Reference

FIELD FILTERS:
  status:<value>       Filter by status (pending, in_progress, completed, cancelled)
  priority:<value>     Filter by priority (low, medium, high, urgent)
  tag:<value>          Filter by tag
  project:<name>       Filter by project name

NEGATION:
  -tag:<value>         Exclude tasks with tag
  -status:<value>      Exclude tasks with status

PROJECT MENTIONS:
  @<name>              Exact project name match
  @~<name>             Fuzzy project name match

DATE FILTERS:
  due:<date>           Due on specific date (YYYY-MM-DD)
  due:+<N>d            Due in next N days
  due:-<N>d            Due in last N days (overdue)
  due:today            Due today
  due:tomorrow         Due tomorrow
  due:none             No due date

COMBINING FILTERS:
  Use spaces to combine multiple filters
  Example: status:pending priority:high @backend -tag:wontfix

EXAMPLES:
  status:pending @frontend
    → Show pending tasks in frontend project

  priority:high due:+7d
    → Show high priority tasks due in next 7 days

  @~back tag:bug -status:completed
    → Show bug tasks in projects matching "back", excluding completed
`
	return content
}

func (m Model) renderQueryHelpModal() string {
	if !m.showQueryHelp {
		return ""
	}

	helpContent := getQueryHelpContent()

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.Primary)).
		Padding(1, 2).
		Width(80)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Primary)).
		Bold(true)

	title := titleStyle.Render("Query Language Help")
	closeHint := m.styles.TUIHelp.Render("Press '?' or ESC to close")

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(helpContent)
	b.WriteString("\n\n")
	b.WriteString(closeHint)

	return modalStyle.Render(b.String())
}

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

		isActive := false
		if item.filterType == "status" && item.value == string(m.filter.Status) {
			isActive = true
		} else if item.filterType == "priority" && item.value == string(m.filter.Priority) {
			isActive = true
		} else if item.filterType == "duedate" {
			isActive = m.isDateFilterActive(item.value)
		}

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

func (m Model) isDateFilterActive(value string) bool {
	if m.filter.DueDateFrom == nil && m.filter.DueDateTo == nil {
		return value == ""
	}

	if m.filter.DueDateFrom != nil && *m.filter.DueDateFrom == "none" {
		return value == "none"
	}

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

func (m Model) renderConfirmDialog() string {
	var b strings.Builder

	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Warning)).
		Bold(true).
		Render(m.confirm.message)

	prompt := m.styles.TUISubtitle.Render("Are you sure? (y/n)")

	content := lipgloss.JoinVertical(lipgloss.Left, message, "", prompt)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.Warning)).
		Padding(1, 2).
		Render(content)

	b.WriteString(box)

	return b.String()
}

func (m Model) renderProjectPicker() string {
	var b strings.Builder

	title := m.styles.TUISubtitle.Render("Select Project")
	b.WriteString(title)
	b.WriteString("\n\n")

	visibleProjects := m.getVisiblePickerProjects()
	if len(visibleProjects) == 0 {
		b.WriteString(m.styles.Info.Render("No projects available."))
	} else {
		for i, project := range visibleProjects {
			icon := project.Icon
			if icon == "" {
				icon = "📦"
			}

			line := fmt.Sprintf("%s %s", icon, project.Name)

			switch project.Status {
				case domain.ProjectStatusArchived:
					line += m.styles.Info.Render(" [archived]")
				case domain.ProjectStatusCompleted:
					line += m.styles.Success.Render(" [✓]")
			}

			if i == m.projectPicker.cursor {
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.theme.SelectedFg)).
					Background(lipgloss.Color(m.theme.SelectedBg)).
					Bold(true).
					Render("▶ " + line)
			} else {
				line = "  " + line
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	help := m.styles.TUIHelp.Render("↑/↓: navigate  •  Enter: select  •  Esc: cancel")
	b.WriteString(help)

	content := b.String()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(1, 2).
		Width(60).
		Render(content)

	return box
}

func (m Model) renderViewPicker() string {
	var b strings.Builder

	title := m.styles.TUISubtitle.Render("Select View")
	b.WriteString(title)
	b.WriteString("\n\n")

	if len(m.viewPicker.views) == 0 {
		b.WriteString(m.styles.Info.Render("No views available."))
	} else {
		for i, view := range m.viewPicker.views {
			line := fmt.Sprintf("%s", view.Name)

			if view.IsFavorite {
				line += m.styles.Success.Render(" ★")
			}
			if view.HotKey != nil && *view.HotKey >= 1 && *view.HotKey <= 9 {
				line += m.styles.Info.Render(fmt.Sprintf(" [%d]", *view.HotKey))
			}

			if i == m.viewPicker.cursor {
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color(m.theme.SelectedFg)).
					Background(lipgloss.Color(m.theme.SelectedBg)).
					Bold(true).
					Render("▶ " + line)
			} else {
				line = "  " + line
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	help := m.styles.TUIHelp.Render("↑/↓: navigate  •  Enter: select  •  Esc: cancel")
	b.WriteString(help)

	content := b.String()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(1, 2).
		Width(60).
		Render(content)

	return box
}

func (m Model) renderQuickAccessWidget() string {
	var b strings.Builder

	title := m.styles.TUIHelp.Render("Quick Access Views (1-9):")
	b.WriteString(title)
	b.WriteString(" ")

	if len(m.quickAccessViews) == 0 {
		b.WriteString(m.styles.Info.Render("None configured"))
	} else {
		var quickAccessLines []string
		for i := 1; i <= 9; i++ {
			if view, exists := m.quickAccessViews[i]; exists {
				viewInfo := fmt.Sprintf("%d: %s", i, view.Name)
				quickAccessLines = append(quickAccessLines, m.styles.Success.Render(viewInfo))
			}
		}
		b.WriteString(strings.Join(quickAccessLines, " • "))
	}

	return b.String()
}

func (m Model) renderStatusBar() string {
	var items []string

	if m.multiSelect.enabled {
		selectedCount := len(m.multiSelect.selectedTasks)
		multiInfo := m.styles.Success.Render(fmt.Sprintf("✓ Multi-select: %d selected", selectedCount))
		items = append(items, multiInfo)
	}

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

	sortIcon := "↓"
	if m.filter.SortOrder == "asc" {
		sortIcon = "↑"
	}
	sortInfo := fmt.Sprintf("Sort: %s %s", m.filter.SortBy, sortIcon)
	items = append(items, sortInfo)

	filterCount := m.countActiveFilters()
	if filterCount > 0 {
		items = append(items, fmt.Sprintf("Filters: %d active", filterCount))
	}

	statusText := strings.Join(items, " • ")
	return m.styles.TUISubtitle.Render(statusText)
}

func (m Model) renderFilterSummary() string {
	var filters []string

	if m.filter.Status != "" {
		filters = append(filters, fmt.Sprintf("Status: %s", m.filter.Status))
	}
	if m.filter.Priority != "" {
		filters = append(filters, fmt.Sprintf("Priority: %s", m.filter.Priority))
	}
	if m.filter.ProjectID != nil {
		filters = append(filters, fmt.Sprintf("Project ID: %d", *m.filter.ProjectID))
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

func (m Model) renderHelp() string {
	if m.showHelp {
		return m.renderFullHelp()
	}

	return m.renderQuickHelp()
}

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

func (m Model) renderDetailRow(label, value string) string {
	return m.styles.DetailLabel.Render(label) + " " + m.styles.DetailValue.Render(value)
}


func (m *Model) hasActiveFilters() bool {
	return m.filter.Status != "" ||
		m.filter.Priority != "" ||
		m.filter.ProjectID != nil ||
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
	if m.filter.ProjectID != nil {
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


func (m Model) renderEditForm() string {
	var b strings.Builder

	formTitle := "Edit Task"
	if m.editForm.isNewTask {
		formTitle = "New Task"
	}
	b.WriteString(m.styles.TUISubtitle.Render(formTitle))
	b.WriteString("\n\n")

	if m.editForm.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.editForm.err))
		b.WriteString("\n\n")
	}

	priorities := []string{"low", "medium", "high", "urgent"}
	statuses := []string{"pending", "in_progress", "completed", "cancelled"}

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

	b.WriteString(m.styles.DetailLabel.Render("  Priority:"))
	b.WriteString(" ")
	priorityValue := domain.Priority(priorities[m.editForm.priorityIdx])
	priorityStyle := m.styles.GetPriorityTextStyle(priorityValue)
	b.WriteString(priorityStyle.Render(priorities[m.editForm.priorityIdx]))
	b.WriteString(m.styles.TUIHelp.Render(" (Ctrl+P to cycle)"))
	b.WriteString("\n\n")

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
	}

	if m.editForm.focusedField == 2 {
		hints = append(hints, "Ctrl+P: project picker")
	} else {
		hints = append(hints, "Ctrl+P: cycle priority")
	}

	hints = append(hints, "Ctrl+T: cycle status", "Esc: cancel")

	return m.styles.TUIHelp.Render(strings.Join(hints, "  •  "))
}

func (m Model) renderProjectForm() string {
	var b strings.Builder

	formTitle := "Edit Project"
	if m.projectForm.mode == createProjectMode {
		formTitle = "New Project"
	}
	b.WriteString(m.styles.TUISubtitle.Render(formTitle))
	b.WriteString("\n\n")

	if errMsg, ok := m.projectForm.errors["general"]; ok {
		b.WriteString(m.styles.Error.Render("Error: " + errMsg))
		b.WriteString("\n\n")
	}

	fieldLabel := "Name:"
	if m.projectForm.focusedField == 0 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.projectForm.nameInput.View())
	if errMsg, ok := m.projectForm.errors["name"]; ok {
		b.WriteString("\n  ")
		b.WriteString(m.styles.Error.Render(errMsg))
	}
	b.WriteString("\n\n")

	fieldLabel = "Description:"
	if m.projectForm.focusedField == 1 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.projectForm.descInput.View())
	if errMsg, ok := m.projectForm.errors["description"]; ok {
		b.WriteString("\n  ")
		b.WriteString(m.styles.Error.Render(errMsg))
	}
	b.WriteString("\n\n")

	fieldLabel = "Parent:"
	if m.projectForm.focusedField == 2 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.projectForm.parentInput.View())
	if errMsg, ok := m.projectForm.errors["parent"]; ok {
		b.WriteString("\n  ")
		b.WriteString(m.styles.Error.Render(errMsg))
	}
	b.WriteString("\n\n")

	fieldLabel = "Color:"
	if m.projectForm.focusedField == 3 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.projectForm.colorInput.View())
	b.WriteString(m.styles.TUIHelp.Render("  (blue, red, green, yellow, cyan, magenta, etc.)"))
	b.WriteString("\n\n")

	fieldLabel = "Icon:"
	if m.projectForm.focusedField == 4 {
		fieldLabel = m.styles.Success.Render("▶ " + fieldLabel)
	} else {
		fieldLabel = "  " + fieldLabel
	}
	b.WriteString(m.styles.DetailLabel.Render(fieldLabel))
	b.WriteString("\n  ")
	b.WriteString(m.projectForm.iconInput.View())
	b.WriteString(m.styles.TUIHelp.Render("  (emoji, e.g., 📦 🚀 💼 🔧)"))
	b.WriteString("\n\n")

	if m.projectForm.mode == editProjectMode {
		statuses := []string{"active", "archived", "completed"}
		b.WriteString(m.styles.DetailLabel.Render("  Status:"))
		b.WriteString(" ")
		statusValue := domain.ProjectStatus(statuses[m.projectForm.statusIdx])
		var statusStyle lipgloss.Style
		switch statusValue {
		case domain.ProjectStatusActive:
			statusStyle = m.styles.Success
		case domain.ProjectStatusArchived:
			statusStyle = m.styles.Info
		case domain.ProjectStatusCompleted:
			statusStyle = m.styles.Success.Bold(true)
		default:
			statusStyle = m.styles.DetailValue
		}
		b.WriteString(statusStyle.Render(statuses[m.projectForm.statusIdx]))
		b.WriteString(m.styles.TUIHelp.Render(" (Ctrl+T to cycle)"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderProjectFormHelp() string {
	hints := []string{
		"Tab/Shift+Tab: navigate fields",
		"Ctrl+S: save",
		"Esc: cancel",
	}
	if m.projectForm.mode == editProjectMode {
		hints = append(hints, "Ctrl+T: cycle status")
	}
	return m.styles.TUIHelp.Render(strings.Join(hints, "  •  "))
}

func (m Model) renderNotesViewer() string {
	if m.notesViewer.project == nil {
		return ""
	}

	project := m.notesViewer.project
	var b strings.Builder

	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	title := m.styles.Title.Render(fmt.Sprintf("%s %s - Notes", icon, project.Name))
	b.WriteString(title)
	b.WriteString("\n\n")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(1, 2)

	viewportContent := m.notesViewer.viewport.View()
	b.WriteString(border.Render(viewportContent))
	b.WriteString("\n\n")

	help := m.styles.TUIHelp.Render("↑/↓: scroll  •  PgUp/PgDn: page  •  Esc: close  •  (Read-only, use 'project note' to edit)")
	b.WriteString(help)

	return b.String()
}
