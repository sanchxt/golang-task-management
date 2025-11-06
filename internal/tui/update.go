package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
)

// updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle confirmation dialog first
	if m.confirm.active {
		return m.updateConfirmDialog(msg)
	}

	// handle search mode
	if m.uiMode == searchingMode {
		return m.updateSearchMode(msg)
	}

	// handle filter mode
	if m.uiMode == filteringMode {
		return m.updateFilterMode(msg)
	}

	// normal mode updates
	return m.updateNormalMode(msg)
}

// updates confirmation dialog
func (m Model) updateConfirmDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			// confirm action
			m.confirm.active = false
			cmd := m.confirm.onConfirm(&m)
			return m, cmd

		case "n", "N", "esc":
			// cancel
			m.confirm.active = false
			return m, nil
		}
	}
	return m, nil
}

// updates search mode
func (m Model) updateSearchMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			// exit search mode
			m.uiMode = normalMode
			m.searchInput.Blur()
			return m, nil

		case msg.Type == tea.KeyEnter:
			// apply search
			searchQuery := m.searchInput.Value()

			// check for regex mode (re: prefix)
			if strings.HasPrefix(searchQuery, "re:") {
				m.filter.SearchMode = "regex"
				m.filter.SearchQuery = strings.TrimPrefix(searchQuery, "re:")
			} else if strings.HasPrefix(searchQuery, "/") {
				m.filter.SearchMode = "text"
				m.filter.SearchQuery = strings.TrimPrefix(searchQuery, "/")
			} else {
				m.filter.SearchMode = "text"
				m.filter.SearchQuery = searchQuery
			}

			m.currentPage = 1
			m.uiMode = normalMode
			m.searchInput.Blur()
			m.loading = true
			return m, m.refreshCmd()
		}
	}

	// update search input
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// updates filter mode
func (m Model) updateFilterMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			// exit filter mode
			m.uiMode = normalMode
			m.filterPanel.active = false
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.filterPanel.selectedItem > 0 {
				m.filterPanel.selectedItem--
			}
			return m, nil

		case key.Matches(msg, m.keys.Down):
			if m.filterPanel.selectedItem < len(m.filterPanel.items)-1 {
				m.filterPanel.selectedItem++
			}
			return m, nil

		case msg.Type == tea.KeyEnter:
			// apply selected filter
			return m.applyFilterSelection()
		}
	}
	return m, nil
}

// applies filter selection
func (m Model) applyFilterSelection() (tea.Model, tea.Cmd) {
	if m.filterPanel.selectedItem < 0 || m.filterPanel.selectedItem >= len(m.filterPanel.items) {
		return m, nil
	}

	item := m.filterPanel.items[m.filterPanel.selectedItem]

	switch item.filterType {
	case "status":
		if item.value == "" {
			m.filter.Status = ""
		} else {
			m.filter.Status = domain.Status(item.value)
		}

	case "priority":
		if item.value == "" {
			m.filter.Priority = ""
		} else {
			m.filter.Priority = domain.Priority(item.value)
		}

	case "clear":
		// clear all filters
		m.filter.Status = ""
		m.filter.Priority = ""
		m.filter.Project = ""
		m.filter.Tags = []string{}
		m.filter.SearchQuery = ""
		m.filter.SearchMode = ""

	case "sort":
		// handled separately via keybindings
		return m, nil
	}

	m.currentPage = 1
	m.uiMode = normalMode
	m.filterPanel.active = false
	m.loading = true
	return m, m.refreshCmd()
}

// updates normal mode
func (m Model) updateNormalMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 12) // leave room for header, status bar, help
		return m, nil

	case tasksLoadedMsg:
		// tasks loaded successfully
		m.tasks = msg.tasks
		m.totalCount = msg.totalCount
		m.loading = false
		m.message = ""
		m.err = nil
		m.updateTableRows()
		return m, nil

	case taskUpdatedMsg:
		// task updated successfully
		m.message = "Task updated successfully"
		m.loading = false
		return m, m.refreshCmd()

	case taskDeletedMsg:
		// task deleted successfully
		m.message = "Task deleted successfully"
		m.loading = false
		m.selectedTask = nil
		m.viewMode = tableView
		return m, m.refreshCmd()

	case errMsg:
		// error occurred
		m.err = msg.err
		m.loading = false
		return m, nil
	}

	// update table if in table view
	if m.viewMode == tableView && m.uiMode == normalMode {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

// handles key presses in normal mode
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		// open filter panel
		m.uiMode = filteringMode
		m.filterPanel.active = true
		m.filterPanel.selectedItem = 0
		m.filterPanel.items = m.buildFilterItems()
		return m, nil

	case key.Matches(msg, m.keys.ClearFilters):
		// clear all filters
		m.filter.Status = ""
		m.filter.Priority = ""
		m.filter.Project = ""
		m.filter.Tags = []string{}
		m.filter.SearchQuery = ""
		m.filter.SearchMode = ""
		m.currentPage = 1
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.Search):
		// enter search mode
		m.uiMode = searchingMode
		m.searchInput.Focus()
		m.searchInput.SetValue(m.filter.SearchQuery)
		return m, nil

	case key.Matches(msg, m.keys.Sort):
		// cycle sort mode
		m.cycleSortMode()
		m.currentPage = 1
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.SortOrder):
		// toggle sort order
		if m.filter.SortOrder == "asc" {
			m.filter.SortOrder = "desc"
		} else {
			m.filter.SortOrder = "asc"
		}
		m.currentPage = 1
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.NextPage):
		// next page
		totalPages := m.calculateTotalPages()
		if m.currentPage < totalPages {
			m.currentPage++
			m.loading = true
			return m, m.refreshCmd()
		}
		return m, nil

	case key.Matches(msg, m.keys.PrevPage):
		// previous page
		if m.currentPage > 1 {
			m.currentPage--
			m.loading = true
			return m, m.refreshCmd()
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		// refresh tasks
		m.loading = true
		return m, m.refreshCmd()

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
			m.message = ""
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.viewMode == detailView {
			// prev task in detail view
			m.navigateToPreviousTask()
			return m, nil
		} else if m.viewMode == tableView {
			// let table handle navigation
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	case key.Matches(msg, m.keys.Down):
		if m.viewMode == detailView {
			// next task in detail view
			m.navigateToNextTask()
			return m, nil
		} else if m.viewMode == tableView {
			// let table handle navigation
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	// Quick actions
	case key.Matches(msg, m.keys.MarkComplete):
		return m.handleMarkComplete()

	case key.Matches(msg, m.keys.CyclePriority):
		return m.handleCyclePriority()

	case key.Matches(msg, m.keys.Delete):
		return m.handleDelete()

	case key.Matches(msg, m.keys.ToggleStatus):
		return m.handleToggleStatus()
	}

	return m, nil
}

// Quick action handlers

func (m Model) handleMarkComplete() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

	// toggle between pending/completed
	if task.Status == domain.StatusCompleted {
		task.Status = domain.StatusPending
	} else {
		task.Status = domain.StatusCompleted
	}

	m.loading = true
	return m, updateTaskCmd(m.ctx, m.repo, task)
}

func (m Model) handleCyclePriority() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

	// cycle: low -> medium -> high -> urgent -> low
	switch task.Priority {
	case domain.PriorityLow:
		task.Priority = domain.PriorityMedium
	case domain.PriorityMedium:
		task.Priority = domain.PriorityHigh
	case domain.PriorityHigh:
		task.Priority = domain.PriorityUrgent
	case domain.PriorityUrgent:
		task.Priority = domain.PriorityLow
	default:
		task.Priority = domain.PriorityMedium
	}

	m.loading = true
	return m, updateTaskCmd(m.ctx, m.repo, task)
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

	// show confirmation dialog
	m.confirm = confirmDialog{
		message: "Delete task: " + task.Title + "?",
		active:  true,
		onConfirm: func(model *Model) tea.Cmd {
			return deleteTaskCmd(model.ctx, model.repo, task.ID)
		},
	}

	return m, nil
}

func (m Model) handleToggleStatus() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

	// toggle between pending/in_progress
	if task.Status == domain.StatusPending {
		task.Status = domain.StatusInProgress
	} else if task.Status == domain.StatusInProgress {
		task.Status = domain.StatusPending
	} else {
		// if other status, set to in_progress
		task.Status = domain.StatusInProgress
	}

	m.loading = true
	return m, updateTaskCmd(m.ctx, m.repo, task)
}

// Helper methods

func (m *Model) getSelectedTask() *domain.Task {
	if m.viewMode == detailView {
		return m.selectedTask
	}
	if m.viewMode == tableView && len(m.tasks) > 0 {
		selectedRow := m.table.Cursor()
		if selectedRow < len(m.tasks) {
			return m.tasks[selectedRow]
		}
	}
	return nil
}

func (m *Model) cycleSortMode() {
	// cycle: created_at -> updated_at -> priority -> due_date -> title -> created_at
	switch m.filter.SortBy {
	case "created_at":
		m.filter.SortBy = "updated_at"
	case "updated_at":
		m.filter.SortBy = "priority"
	case "priority":
		m.filter.SortBy = "due_date"
	case "due_date":
		m.filter.SortBy = "title"
	case "title":
		m.filter.SortBy = "created_at"
	default:
		m.filter.SortBy = "created_at"
	}
}

func (m *Model) calculateTotalPages() int {
	if m.pageSize == 0 {
		return 1
	}
	return int((m.totalCount + int64(m.pageSize) - 1) / int64(m.pageSize))
}

func (m *Model) updateTableRows() {
	rows := make([]table.Row, len(m.tasks))
	for i, task := range m.tasks {
		rows[i] = taskToRow(task)
	}
	m.table.SetRows(rows)
}

func (m *Model) buildFilterItems() []filterItem {
	items := []filterItem{
		{label: "Filter by Status", value: "", filterType: "status"},
		{label: "  ○ All", value: "", filterType: "status"},
		{label: "  ○ Pending", value: "pending", filterType: "status"},
		{label: "  ○ In Progress", value: "in_progress", filterType: "status"},
		{label: "  ○ Completed", value: "completed", filterType: "status"},
		{label: "  ○ Cancelled", value: "cancelled", filterType: "status"},
		{label: "", value: "", filterType: ""},
		{label: "Filter by Priority", value: "", filterType: "priority"},
		{label: "  ○ All", value: "", filterType: "priority"},
		{label: "  ○ Low", value: "low", filterType: "priority"},
		{label: "  ○ Medium", value: "medium", filterType: "priority"},
		{label: "  ○ High", value: "high", filterType: "priority"},
		{label: "  ○ Urgent", value: "urgent", filterType: "priority"},
		{label: "", value: "", filterType: ""},
		{label: "Clear All Filters", value: "", filterType: "clear"},
	}
	return items
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
