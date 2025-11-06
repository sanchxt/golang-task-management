package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
)

// updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle confirmation dialog first
	if m.confirm.active {
		return m.updateConfirmDialog(msg)
	}

	// handle edit form
	if m.editForm.active {
		return m.updateEditMode(msg)
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

	case "duedate":
		// clear existing date filters first
		m.filter.DueDateFrom = nil
		m.filter.DueDateTo = nil

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		switch item.value {
		case "":
			// All - no date filter
		case "overdue":
			// Tasks with due date before today
			yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
			m.filter.DueDateTo = &yesterday
		case "today":
			// Tasks due today
			todayStr := today.Format("2006-01-02")
			tomorrowStr := today.AddDate(0, 0, 1).Format("2006-01-02")
			m.filter.DueDateFrom = &todayStr
			m.filter.DueDateTo = &tomorrowStr
		case "week":
			// Tasks due within the next 7 days
			todayStr := today.Format("2006-01-02")
			weekStr := today.AddDate(0, 0, 7).Format("2006-01-02")
			m.filter.DueDateFrom = &todayStr
			m.filter.DueDateTo = &weekStr
		case "month":
			// Tasks due within the next 30 days
			todayStr := today.Format("2006-01-02")
			monthStr := today.AddDate(0, 0, 30).Format("2006-01-02")
			m.filter.DueDateFrom = &todayStr
			m.filter.DueDateTo = &monthStr
		case "none":
			// Tasks with no due date - this requires a different approach
			// We'll use a special marker value
			noneMarker := "none"
			m.filter.DueDateFrom = &noneMarker
		}

	case "clear":
		// clear all filters
		m.filter.Status = ""
		m.filter.Priority = ""
		m.filter.Project = ""
		m.filter.Tags = []string{}
		m.filter.SearchQuery = ""
		m.filter.SearchMode = ""
		m.filter.DueDateFrom = nil
		m.filter.DueDateTo = nil

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

	// Task creation and editing
	case key.Matches(msg, m.keys.New):
		return m.handleNewTask()

	case key.Matches(msg, m.keys.Edit):
		return m.handleEditTask()

	// Multi-select
	case key.Matches(msg, m.keys.ToggleMultiSelect):
		return m.handleToggleMultiSelect()

	case key.Matches(msg, m.keys.ToggleSelection):
		if m.multiSelect.enabled {
			return m.handleToggleSelection()
		}

	case key.Matches(msg, m.keys.SelectAll):
		if m.multiSelect.enabled {
			return m.handleSelectAll()
		}

	case key.Matches(msg, m.keys.DeselectAll):
		if m.multiSelect.enabled {
			return m.handleDeselectAll()
		}

	// Quick actions
	case key.Matches(msg, m.keys.MarkComplete):
		if m.multiSelect.enabled && len(m.multiSelect.selectedTasks) > 0 {
			return m.handleBulkMarkComplete()
		}
		return m.handleMarkComplete()

	case key.Matches(msg, m.keys.CyclePriority):
		if m.multiSelect.enabled && len(m.multiSelect.selectedTasks) > 0 {
			return m.handleBulkCyclePriority()
		}
		return m.handleCyclePriority()

	case key.Matches(msg, m.keys.Delete):
		if m.multiSelect.enabled && len(m.multiSelect.selectedTasks) > 0 {
			return m.handleBulkDelete()
		}
		return m.handleDelete()

	case key.Matches(msg, m.keys.ToggleStatus):
		if m.multiSelect.enabled && len(m.multiSelect.selectedTasks) > 0 {
			return m.handleBulkToggleStatus()
		}
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
		rows[i] = m.taskToRow(task)
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
		{label: "Filter by Due Date", value: "", filterType: "duedate"},
		{label: "  ○ All", value: "", filterType: "duedate"},
		{label: "  ○ Overdue", value: "overdue", filterType: "duedate"},
		{label: "  ○ Due Today", value: "today", filterType: "duedate"},
		{label: "  ○ Due This Week", value: "week", filterType: "duedate"},
		{label: "  ○ Due This Month", value: "month", filterType: "duedate"},
		{label: "  ○ No Due Date", value: "none", filterType: "duedate"},
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

// Edit mode handler

func (m Model) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// cancel edit
			m.editForm.active = false
			m.editForm.err = ""
			m.viewMode = tableView
			return m, nil

		case "ctrl+s", "ctrl+enter":
			// save task
			return m.handleSaveTask()

		case "tab":
			// next field
			m.editForm.focusedField++
			if m.editForm.focusedField > 4 {
				m.editForm.focusedField = 0
			}
			m.updateFormFocus()
			return m, nil

		case "shift+tab":
			// previous field
			m.editForm.focusedField--
			if m.editForm.focusedField < 0 {
				m.editForm.focusedField = 4
			}
			m.updateFormFocus()
			return m, nil

		case "ctrl+p":
			// cycle priority
			priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
			m.editForm.priorityIdx = (m.editForm.priorityIdx + 1) % len(priorities)
			return m, nil

		case "ctrl+t":
			// cycle status
			statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}
			m.editForm.statusIdx = (m.editForm.statusIdx + 1) % len(statuses)
			return m, nil
		}

	case taskCreatedMsg:
		m.message = "Task created successfully"
		m.loading = false
		m.editForm.active = false
		m.editForm.err = ""
		m.viewMode = tableView
		return m, m.refreshCmd()

	case taskUpdatedMsg:
		m.message = "Task updated successfully"
		m.loading = false
		m.editForm.active = false
		m.editForm.err = ""
		m.viewMode = tableView
		return m, m.refreshCmd()

	case errMsg:
		m.err = msg.err
		m.editForm.err = msg.err.Error()
		m.loading = false
		return m, nil
	}

	// update focused field
	var cmd tea.Cmd
	switch m.editForm.focusedField {
	case 0:
		m.editForm.titleInput, cmd = m.editForm.titleInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		m.editForm.descInput, cmd = m.editForm.descInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		m.editForm.projectInput, cmd = m.editForm.projectInput.Update(msg)
		cmds = append(cmds, cmd)
	case 3:
		m.editForm.tagsInput, cmd = m.editForm.tagsInput.Update(msg)
		cmds = append(cmds, cmd)
	case 4:
		m.editForm.dueDateInput, cmd = m.editForm.dueDateInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// New task and edit handlers

func (m Model) handleNewTask() (tea.Model, tea.Cmd) {
	m.initEditForm(nil)
	m.editForm.active = true
	m.editForm.isNewTask = true
	m.viewMode = editView
	return m, nil
}

func (m Model) handleEditTask() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

	m.initEditForm(task)
	m.editForm.active = true
	m.editForm.isNewTask = false
	m.viewMode = editView
	return m, nil
}

func (m *Model) initEditForm(task *domain.Task) {
	// initialize text inputs
	titleInput := textinput.New()
	titleInput.Placeholder = "Task title"
	titleInput.CharLimit = 200
	titleInput.Width = 60

	descInput := textarea.New()
	descInput.Placeholder = "Task description (optional)"
	descInput.CharLimit = 1000
	descInput.SetWidth(60)
	descInput.SetHeight(5)

	projectInput := textinput.New()
	projectInput.Placeholder = "Project (optional)"
	projectInput.CharLimit = 100
	projectInput.Width = 40

	tagsInput := textinput.New()
	tagsInput.Placeholder = "Tags (comma-separated, optional)"
	tagsInput.CharLimit = 200
	tagsInput.Width = 60

	dueDateInput := textinput.New()
	dueDateInput.Placeholder = "Due date (YYYY-MM-DD, optional)"
	dueDateInput.CharLimit = 10
	dueDateInput.Width = 20

	m.editForm.titleInput = titleInput
	m.editForm.descInput = descInput
	m.editForm.projectInput = projectInput
	m.editForm.tagsInput = tagsInput
	m.editForm.dueDateInput = dueDateInput
	m.editForm.focusedField = 0
	m.editForm.err = ""

	if task != nil {
		// editing existing task
		m.editForm.editingTask = task
		m.editForm.titleInput.SetValue(task.Title)
		m.editForm.descInput.SetValue(task.Description)
		m.editForm.projectInput.SetValue(task.Project)
		if len(task.Tags) > 0 {
			m.editForm.tagsInput.SetValue(strings.Join(task.Tags, ", "))
		}
		if task.DueDate != nil {
			m.editForm.dueDateInput.SetValue(task.DueDate.Format("2006-01-02"))
		}

		// set priority and status indices
		priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
		for i, p := range priorities {
			if p == task.Priority {
				m.editForm.priorityIdx = i
				break
			}
		}

		statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}
		for i, s := range statuses {
			if s == task.Status {
				m.editForm.statusIdx = i
				break
			}
		}
	} else {
		// new task
		m.editForm.editingTask = nil
		m.editForm.priorityIdx = 1 // default to medium
		m.editForm.statusIdx = 0    // default to pending
	}

	m.editForm.titleInput.Focus()
}

func (m *Model) updateFormFocus() {
	m.editForm.titleInput.Blur()
	m.editForm.descInput.Blur()
	m.editForm.projectInput.Blur()
	m.editForm.tagsInput.Blur()
	m.editForm.dueDateInput.Blur()

	switch m.editForm.focusedField {
	case 0:
		m.editForm.titleInput.Focus()
	case 1:
		m.editForm.descInput.Focus()
	case 2:
		m.editForm.projectInput.Focus()
	case 3:
		m.editForm.tagsInput.Focus()
	case 4:
		m.editForm.dueDateInput.Focus()
	}
}

func (m Model) handleSaveTask() (tea.Model, tea.Cmd) {
	// build task from form
	title := strings.TrimSpace(m.editForm.titleInput.Value())
	if title == "" {
		m.editForm.err = "Title is required"
		return m, nil
	}

	description := strings.TrimSpace(m.editForm.descInput.Value())
	project := strings.TrimSpace(m.editForm.projectInput.Value())

	// parse tags
	var tags []string
	tagsStr := strings.TrimSpace(m.editForm.tagsInput.Value())
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// parse due date
	dueDateStr := strings.TrimSpace(m.editForm.dueDateInput.Value())

	priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
	statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}

	if m.editForm.isNewTask {
		// create new task
		task := domain.NewTask(title)
		task.Description = description
		task.Project = project
		task.Tags = tags
		task.Priority = priorities[m.editForm.priorityIdx]
		task.Status = statuses[m.editForm.statusIdx]

		// parse due date if provided
		if dueDateStr != "" {
			// Simple date parsing (you can enhance this)
			dueTime, err := domain.ParseDueDate(dueDateStr)
			if err == nil {
				task.DueDate = dueTime
			}
		}

		m.loading = true
		return m, createTaskCmd(m.ctx, m.repo, task)
	} else {
		// update existing task
		task := m.editForm.editingTask
		task.Title = title
		task.Description = description
		task.Project = project
		task.Tags = tags
		task.Priority = priorities[m.editForm.priorityIdx]
		task.Status = statuses[m.editForm.statusIdx]

		// parse due date if provided
		if dueDateStr != "" {
			dueTime, err := domain.ParseDueDate(dueDateStr)
			if err == nil {
				task.DueDate = dueTime
			}
		} else {
			task.DueDate = nil
		}

		m.loading = true
		return m, updateTaskCmd(m.ctx, m.repo, task)
	}
}

// Multi-select handlers

func (m Model) handleToggleMultiSelect() (tea.Model, tea.Cmd) {
	m.multiSelect.enabled = !m.multiSelect.enabled
	if !m.multiSelect.enabled {
		// clear selections when disabling
		m.multiSelect.selectedTasks = make(map[int64]bool)
	}
	return m, nil
}

func (m Model) handleToggleSelection() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

	if m.multiSelect.selectedTasks[task.ID] {
		delete(m.multiSelect.selectedTasks, task.ID)
	} else {
		m.multiSelect.selectedTasks[task.ID] = true
	}

	return m, nil
}

func (m Model) handleSelectAll() (tea.Model, tea.Cmd) {
	for _, task := range m.tasks {
		m.multiSelect.selectedTasks[task.ID] = true
	}
	return m, nil
}

func (m Model) handleDeselectAll() (tea.Model, tea.Cmd) {
	m.multiSelect.selectedTasks = make(map[int64]bool)
	return m, nil
}

// Bulk operation handlers

func (m Model) handleBulkMarkComplete() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	for _, task := range m.tasks {
		if m.multiSelect.selectedTasks[task.ID] {
			// toggle between pending/completed
			if task.Status == domain.StatusCompleted {
				task.Status = domain.StatusPending
			} else {
				task.Status = domain.StatusCompleted
			}
			cmds = append(cmds, updateTaskCmd(m.ctx, m.repo, task))
		}
	}

	m.multiSelect.selectedTasks = make(map[int64]bool)
	m.loading = true
	return m, tea.Batch(cmds...)
}

func (m Model) handleBulkCyclePriority() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	for _, task := range m.tasks {
		if m.multiSelect.selectedTasks[task.ID] {
			// cycle priority
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
			cmds = append(cmds, updateTaskCmd(m.ctx, m.repo, task))
		}
	}

	m.multiSelect.selectedTasks = make(map[int64]bool)
	m.loading = true
	return m, tea.Batch(cmds...)
}

func (m Model) handleBulkToggleStatus() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	for _, task := range m.tasks {
		if m.multiSelect.selectedTasks[task.ID] {
			// toggle between pending/in_progress
			if task.Status == domain.StatusPending {
				task.Status = domain.StatusInProgress
			} else if task.Status == domain.StatusInProgress {
				task.Status = domain.StatusPending
			} else {
				task.Status = domain.StatusInProgress
			}
			cmds = append(cmds, updateTaskCmd(m.ctx, m.repo, task))
		}
	}

	m.multiSelect.selectedTasks = make(map[int64]bool)
	m.loading = true
	return m, tea.Batch(cmds...)
}

func (m Model) handleBulkDelete() (tea.Model, tea.Cmd) {
	count := len(m.multiSelect.selectedTasks)
	if count == 0 {
		return m, nil
	}

	// show confirmation dialog
	m.confirm = confirmDialog{
		message: fmt.Sprintf("Delete %d task(s)?", count),
		active:  true,
		onConfirm: func(model *Model) tea.Cmd {
			var cmds []tea.Cmd
			for taskID := range model.multiSelect.selectedTasks {
				cmds = append(cmds, deleteTaskCmd(model.ctx, model.repo, taskID))
			}
			model.multiSelect.selectedTasks = make(map[int64]bool)
			return tea.Batch(cmds...)
		},
	}

	return m, nil
}
