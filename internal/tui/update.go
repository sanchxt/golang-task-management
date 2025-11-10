package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"task-management/internal/domain"
	"task-management/internal/fuzzy"
	"task-management/internal/query"
	"task-management/internal/repository"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		return m.updateConfirmDialog(msg)
	}

	if m.viewPicker.active {
		return m.updateViewPicker(msg)
	}

	if m.projectPicker.active {
		return m.updateProjectPicker(msg)
	}

	if m.editForm.active {
		return m.updateEditMode(msg)
	}

	if m.projectForm.active {
		return m.updateProjectFormMode(msg)
	}

	if m.uiMode == searchingMode {
		return m.updateSearchMode(msg)
	}

	if m.uiMode == filteringMode {
		return m.updateFilterMode(msg)
	}

	return m.updateNormalMode(msg)
}

func (m Model) updateConfirmDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirm.active = false
			cmd := m.confirm.onConfirm(&m)
			return m, cmd

		case "n", "N", "esc":
			m.confirm.active = false
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateProjectPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.projectPicker.active = false
			return m, nil

		case "up", "k":
			if m.projectPicker.cursor > 0 {
				m.projectPicker.cursor--
			}
			return m, nil

		case "down", "j":
			visibleProjects := m.getVisiblePickerProjects()
			if m.projectPicker.cursor < len(visibleProjects)-1 {
				m.projectPicker.cursor++
			}
			return m, nil

		case "enter":
			visibleProjects := m.getVisiblePickerProjects()
			if m.projectPicker.cursor < len(visibleProjects) {
				selectedProject := visibleProjects[m.projectPicker.cursor]
				m.projectPicker.selected = selectedProject

				m.editForm.projectInput.SetValue(selectedProject.Name)

				m.projectPicker.active = false
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) getVisiblePickerProjects() []*domain.Project {
	return m.projectPicker.projects
}

func (m *Model) lookupProjectByFuzzyName(ctx context.Context, searchName string, threshold int) (*int64, error) {
	if strings.TrimSpace(searchName) == "" {
		return nil, fmt.Errorf("search name cannot be empty")
	}

	filter := repository.ProjectFilter{
		ExcludeArchived: true,
	}

	projects, err := m.projectRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no matching project found for '%s'", searchName)
	}

	type projectWithScore struct {
		project *domain.Project
		score   int
	}

	scoredProjects := make([]projectWithScore, 0, len(projects))
	for _, proj := range projects {
		score := fuzzy.Match(searchName, proj.Name)
		if score >= threshold {
			scoredProjects = append(scoredProjects, projectWithScore{
				project: proj,
				score:   score,
			})
		}
	}

	if len(scoredProjects) == 0 {
		return nil, fmt.Errorf("no matching project found for '%s' (threshold: %d)", searchName, threshold)
	}

	sort.Slice(scoredProjects, func(i, j int) bool {
		return scoredProjects[i].score > scoredProjects[j].score
	})

	bestMatch := scoredProjects[0].project
	return &bestMatch.ID, nil
}

func (m *Model) lookupProjectID(ctx context.Context, projectStr string) (*int64, error) {
	if strings.TrimSpace(projectStr) == "" {
		return nil, nil
	}

	if id, err := strconv.ParseInt(projectStr, 10, 64); err == nil {
		project, err := m.projectRepo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("project with ID %d not found: %w", id, err)
		}
		return &project.ID, nil
	}

	project, err := m.projectRepo.GetByName(ctx, projectStr)
	if err != nil {
		return nil, fmt.Errorf("project '%s' not found: %w", projectStr, err)
	}

	return &project.ID, nil
}

func (m Model) updateSearchMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.historyDropdown.active {
			switch msg.String() {
			case "esc":
				m.historyDropdown.active = false
				m.historyDropdown.cursor = 0
				return m, nil

			case "up", "k":
				if m.historyDropdown.cursor > 0 {
					m.historyDropdown.cursor--
				}
				return m, nil

			case "down", "j":
				if m.historyDropdown.cursor < len(m.searchHistory)-1 {
					m.historyDropdown.cursor++
				}
				return m, nil

			case "enter":
				if m.historyDropdown.cursor < len(m.searchHistory) {
					selected := m.searchHistory[m.historyDropdown.cursor]
					m.searchInput.SetValue(selected.QueryText)

					if selected.SearchMode == domain.SearchModeFuzzy {
						m.fuzzyMode = true
						if selected.FuzzyThreshold != nil {
							m.fuzzyThreshold = *selected.FuzzyThreshold
						}
					} else {
						m.fuzzyMode = false
					}

					m.historyDropdown.active = false
					m.historyDropdown.cursor = 0
				}
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, m.keys.Back):
			if m.showQueryHelp {
				m.showQueryHelp = false
				return m, nil
			}
			if m.historyDropdown.active {
				m.historyDropdown.active = false
				m.historyDropdown.cursor = 0
				return m, nil
			}
			m.uiMode = normalMode
			m.searchInput.Blur()
			return m, nil

		case msg.String() == "up":
			if m.searchInput.Value() == "" && len(m.searchHistory) > 0 {
				m.historyDropdown.active = true
				m.historyDropdown.cursor = 0
				return m, nil
			}
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd

		case msg.String() == "f" || msg.String() == "F":
			m.fuzzyMode = !m.fuzzyMode
			return m, nil

		case msg.String() == "?":
			m.showQueryHelp = !m.showQueryHelp
			return m, nil

		case msg.Type == tea.KeyEnter:
			searchQuery := m.searchInput.Value()

			if query.IsQueryLanguage(searchQuery) {
				m.queryMode = true
				m.queryString = searchQuery

				converterCtx := &query.ConverterContext{
					ProjectRepo: m.projectRepo,
				}
				m.uiMode = normalMode
				m.searchInput.Blur()
				m.loading = true

				historyEntry := &domain.SearchHistory{
					QueryText:  searchQuery,
					SearchMode: domain.SearchModeText,
					QueryType:  domain.QueryTypeQueryLanguage,
				}
				recordCmd := recordSearchCmd(m.ctx, m.searchHistoryRepo, historyEntry)

				return m, tea.Batch(parseQueryLanguageCmd(m.ctx, searchQuery, converterCtx), recordCmd)
			}

			m.queryMode = false
			m.queryString = ""

			parsedQuery, err := query.ParseProjectMentions(searchQuery)
			if err != nil {
				m.err = fmt.Errorf("failed to parse query: %w", err)
				m.uiMode = normalMode
				m.searchInput.Blur()
				return m, nil
			}

			if parsedQuery.HasProjectFilter() {
				mention := parsedQuery.ProjectMentions[0]
				ctx := context.Background()

				if mention.Fuzzy {
					fuzzyThreshold := 60
					projectID, err := m.lookupProjectByFuzzyName(ctx, mention.Name, fuzzyThreshold)
					if err != nil {
						m.err = err
						m.uiMode = normalMode
						m.searchInput.Blur()
						return m, nil
					}
					m.filter.ProjectID = projectID
				} else {
					projectID, err := m.lookupProjectID(ctx, mention.Name)
					if err != nil {
						m.err = err
						m.uiMode = normalMode
						m.searchInput.Blur()
						return m, nil
					}
					m.filter.ProjectID = projectID
				}

				searchQuery = parsedQuery.BaseQuery
			}

			if searchQuery != "" {
				if strings.HasPrefix(searchQuery, "re:") {
					m.filter.SearchMode = "regex"
					m.filter.SearchQuery = strings.TrimPrefix(searchQuery, "re:")
				} else if m.fuzzyMode {
					m.filter.SearchMode = "fuzzy"
					m.filter.SearchQuery = searchQuery
					m.filter.FuzzyThreshold = m.fuzzyThreshold
				} else if strings.HasPrefix(searchQuery, "/") {
					m.filter.SearchMode = "text"
					m.filter.SearchQuery = strings.TrimPrefix(searchQuery, "/")
				} else {
					m.filter.SearchMode = "text"
					m.filter.SearchQuery = searchQuery
				}
			} else {
				m.filter.SearchQuery = ""
				m.filter.SearchMode = ""
			}

			m.currentPage = 1
			m.uiMode = normalMode
			m.searchInput.Blur()
			m.loading = true

			if m.filter.SearchQuery != "" {
				queryType := domain.QueryTypeSimple
				if parsedQuery.HasProjectFilter() {
					queryType = domain.QueryTypeProjectMention
				}

				var searchMode domain.SearchMode
				switch m.filter.SearchMode {
				case "regex":
					searchMode = domain.SearchModeRegex
				case "fuzzy":
					searchMode = domain.SearchModeFuzzy
				default:
					searchMode = domain.SearchModeText
				}

				historyEntry := &domain.SearchHistory{
					QueryText:  m.filter.SearchQuery,
					SearchMode: searchMode,
					QueryType:  queryType,
				}

				if searchMode == domain.SearchModeFuzzy {
					historyEntry.FuzzyThreshold = &m.fuzzyThreshold
				}

				if parsedQuery.HasProjectFilter() {
					mention := parsedQuery.ProjectMentions[0]
					historyEntry.ProjectFilter = mention.Name
				}

				recordCmd := recordSearchCmd(m.ctx, m.searchHistoryRepo, historyEntry)
				return m, tea.Batch(m.refreshCmd(), recordCmd)
			}

			return m, m.refreshCmd()
		}
	}

	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) updateFilterMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
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
			return m.applyFilterSelection()
		}
	}
	return m, nil
}

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

	case "project":
		if item.value == "" {
			m.filter.ProjectID = nil
		} else {
			var projectID int64
			fmt.Sscanf(item.value, "%d", &projectID)
			m.filter.ProjectID = &projectID
		}

	case "duedate":
		m.filter.DueDateFrom = nil
		m.filter.DueDateTo = nil

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		switch item.value {
		case "":
		case "overdue":
			yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
			m.filter.DueDateTo = &yesterday
		case "today":
			todayStr := today.Format("2006-01-02")
			tomorrowStr := today.AddDate(0, 0, 1).Format("2006-01-02")
			m.filter.DueDateFrom = &todayStr
			m.filter.DueDateTo = &tomorrowStr
		case "week":
			todayStr := today.Format("2006-01-02")
			weekStr := today.AddDate(0, 0, 7).Format("2006-01-02")
			m.filter.DueDateFrom = &todayStr
			m.filter.DueDateTo = &weekStr
		case "month":
			todayStr := today.Format("2006-01-02")
			monthStr := today.AddDate(0, 0, 30).Format("2006-01-02")
			m.filter.DueDateFrom = &todayStr
			m.filter.DueDateTo = &monthStr
		case "none":
			noneMarker := "none"
			m.filter.DueDateFrom = &noneMarker
		}

	case "clear":
		m.filter.Status = ""
		m.filter.Priority = ""
		m.filter.ProjectID = nil
		m.filter.Tags = []string{}
		m.filter.SearchQuery = ""
		m.filter.SearchMode = ""
		m.filter.DueDateFrom = nil
		m.filter.DueDateTo = nil

	case "sort":
		return m, nil
	}

	m.currentPage = 1
	m.uiMode = normalMode
	m.filterPanel.active = false
	m.loading = true
	return m, m.refreshCmd()
}

func (m Model) updateNormalMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.viewMode == projectView {
		} else {
			m.table.SetHeight(msg.Height - 12)
		}
		return m, nil

	case tasksLoadedMsg:
		m.tasks = msg.tasks
		m.totalCount = msg.totalCount
		m.loading = false
		m.message = ""
		m.err = nil
		m.updateTableRows()
		return m, nil

	case queryParsedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.queryMode = false
			return m, nil
		}

		m.filter = msg.filter
		m.currentPage = 1
		m.message = fmt.Sprintf("🔍 Query: %s", msg.queryStr)
		return m, fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize)

	case taskUpdatedMsg:
		m.message = "Task updated successfully"
		m.loading = false
		return m, m.refreshCmd()

	case taskDeletedMsg:
		m.message = "Task deleted successfully"
		m.loading = false
		m.selectedTask = nil
		m.viewMode = tableView
		return m, m.refreshCmd()

	case projectsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
			return m, nil
		}
		m.projects = msg.projects
		m.projectTree = buildProjectTree(msg.projects)
		m.loading = false
		return m, nil

	case projectCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
			return m, nil
		}
		m.message = "Project created successfully"
		m.loading = false
		projectFilter := repository.ProjectFilter{ExcludeArchived: true}
		return m, fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter)

	case projectUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
			return m, nil
		}
		m.message = "Project updated successfully"
		m.loading = false
		projectFilter := repository.ProjectFilter{ExcludeArchived: true}
		return m, fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter)

	case projectDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
			return m, nil
		}
		m.message = "Project deleted successfully"
		m.loading = false
		m.selectedProject = nil
		projectFilter := repository.ProjectFilter{ExcludeArchived: true}
		return m, fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter)

	case projectStatsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.projectStats[msg.projectID] = projectStatsData{
			taskCount: msg.taskCount,
			stats:     msg.stats,
		}
		if m.selectedProject != nil && m.selectedProject.ID == msg.projectID {
			m.selectedProject.TaskCount = msg.taskCount
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
	}

	if m.viewMode == tableView && m.uiMode == normalMode {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.viewMode == projectView {
		return m.handleProjectViewKeyPress(msg)
	}

	if m.viewMode == notesView {
		return m.handleNotesViewKeyPress(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case msg.String() == "?":
		if m.queryMode {
			m.showQueryHelp = !m.showQueryHelp
			return m, nil
		}

	case key.Matches(msg, m.keys.ToggleProjects):
		m.viewMode = projectView
		m.projectCursor = 0
		visibleNodes := m.getVisibleProjectNodes()
		if len(visibleNodes) > 0 {
			m.selectedProject = visibleNodes[0].project
		}
		m.message = "Project View (Press P or Esc to go back)"
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		m.uiMode = filteringMode
		m.filterPanel.active = true
		m.filterPanel.selectedItem = 0
		m.filterPanel.items = m.buildFilterItems()
		return m, nil

	case key.Matches(msg, m.keys.ClearFilters):
		m.filter.Status = ""
		m.filter.Priority = ""
		m.filter.ProjectID = nil
		m.filter.Tags = []string{}
		m.filter.SearchQuery = ""
		m.filter.SearchMode = ""
		m.currentPage = 1
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.Search):
		m.uiMode = searchingMode
		m.searchInput.Focus()
		m.searchInput.SetValue(m.filter.SearchQuery)
		return m, nil

	case key.Matches(msg, m.keys.Sort):
		m.cycleSortMode()
		m.currentPage = 1
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.SortOrder):
		if m.filter.SortOrder == "asc" {
			m.filter.SortOrder = "desc"
		} else {
			m.filter.SortOrder = "asc"
		}
		m.currentPage = 1
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.NextPage):
		totalPages := m.calculateTotalPages()
		if m.currentPage < totalPages {
			m.currentPage++
			m.loading = true
			return m, m.refreshCmd()
		}
		return m, nil

	case key.Matches(msg, m.keys.PrevPage):
		if m.currentPage > 1 {
			m.currentPage--
			m.loading = true
			return m, m.refreshCmd()
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, m.refreshCmd()

	case key.Matches(msg, m.keys.Enter):
		if m.viewMode == tableView && len(m.tasks) > 0 {
			selectedRow := m.table.Cursor()
			if selectedRow < len(m.tasks) {
				m.selectedTask = m.tasks[selectedRow]
				m.viewMode = detailView
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Back):
		if m.viewMode == detailView {
			m.viewMode = tableView
			m.selectedTask = nil
			m.message = ""
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		switch m.viewMode {
			case detailView:
				m.navigateToPreviousTask()
				return m, nil
			case tableView:
				var cmd tea.Cmd
				m.table, cmd = m.table.Update(msg)
				return m, cmd
		}

	case key.Matches(msg, m.keys.Down):
		switch m.viewMode {
			case detailView:
				m.navigateToNextTask()
				return m, nil
			case tableView:
				var cmd tea.Cmd
				m.table, cmd = m.table.Update(msg)
				return m, cmd
		}

	case key.Matches(msg, m.keys.New):
		return m.handleNewTask()

	case key.Matches(msg, m.keys.Edit):
		return m.handleEditTask()

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


func (m Model) handleMarkComplete() (tea.Model, tea.Cmd) {
	task := m.getSelectedTask()
	if task == nil {
		return m, nil
	}

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

	switch task.Status {
		case domain.StatusPending:
			task.Status = domain.StatusInProgress
		case domain.StatusInProgress:
			task.Status = domain.StatusPending
		default:
			task.Status = domain.StatusInProgress
	}

	m.loading = true
	return m, updateTaskCmd(m.ctx, m.repo, task)
}


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
		{label: "Filter by Project", value: "", filterType: "project"},
		{label: "  ○ All", value: "", filterType: "project"},
	}

	for _, proj := range m.projects {
		items = append(items, filterItem{
			label:      fmt.Sprintf("  ○ %s", proj.Name),
			value:      fmt.Sprintf("%d", proj.ID),
			filterType: "project",
		})
	}

	items = append(items, []filterItem{
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
	}...)
	return items
}

func (m *Model) navigateToPreviousTask() {
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

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(m.tasks) - 1
	}

	m.selectedTask = m.tasks[prevIndex]
	m.table.SetCursor(prevIndex)
}

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


func (m Model) updateEditMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.editForm.active = false
			m.editForm.err = ""
			m.viewMode = tableView
			return m, nil

		case "ctrl+s", "ctrl+enter":
			return m.handleSaveTask()

		case "tab":
			m.editForm.focusedField++
			if m.editForm.focusedField > 4 {
				m.editForm.focusedField = 0
			}
			m.updateFormFocus()
			return m, nil

		case "shift+tab":
			m.editForm.focusedField--
			if m.editForm.focusedField < 0 {
				m.editForm.focusedField = 4
			}
			m.updateFormFocus()
			return m, nil

		case "ctrl+p":
			if m.editForm.focusedField == 2 {
				m.projectPicker.active = true
				m.projectPicker.projects = m.projects
				m.projectPicker.cursor = 0
				m.projectPicker.searchQuery = ""
				m.projectPicker.selected = nil
				m.projectPicker.tree = buildProjectTree(m.projects)
				return m, nil
			} else {
				priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
				m.editForm.priorityIdx = (m.editForm.priorityIdx + 1) % len(priorities)
				return m, nil
			}

		case "ctrl+t":
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

func (m Model) updateProjectFormMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.resetProjectForm()
			m.viewMode = projectView
			return m, nil

		case "ctrl+s", "ctrl+enter":
			return m.handleSaveProject()

		case "tab":
			m.projectForm.focusedField++
			if m.projectForm.focusedField > 4 {
				m.projectForm.focusedField = 0
			}
			m.updateProjectFormFocus()
			return m, nil

		case "shift+tab":
			m.projectForm.focusedField--
			if m.projectForm.focusedField < 0 {
				m.projectForm.focusedField = 4
			}
			m.updateProjectFormFocus()
			return m, nil

		case "ctrl+t":
			if m.projectForm.mode == editProjectMode {
				statuses := []domain.ProjectStatus{domain.ProjectStatusActive, domain.ProjectStatusArchived, domain.ProjectStatusCompleted}
				m.projectForm.statusIdx = (m.projectForm.statusIdx + 1) % len(statuses)
			}
			return m, nil
		}

	case projectCreatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.projectForm.errors["general"] = msg.err.Error()
			return m, nil
		}
		m.message = fmt.Sprintf("Project '%s' created successfully", msg.project.Name)
		m.resetProjectForm()
		m.viewMode = projectView
		projectFilter := repository.ProjectFilter{ExcludeArchived: true}
		return m, fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter)

	case projectUpdatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.projectForm.errors["general"] = msg.err.Error()
			return m, nil
		}
		m.message = fmt.Sprintf("Project '%s' updated successfully", msg.project.Name)
		m.resetProjectForm()
		m.viewMode = projectView
		projectFilter := repository.ProjectFilter{ExcludeArchived: true}
		return m, fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter)
	}

	var cmd tea.Cmd
	switch m.projectForm.focusedField {
	case 0:
		m.projectForm.nameInput, cmd = m.projectForm.nameInput.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		m.projectForm.descInput, cmd = m.projectForm.descInput.Update(msg)
		cmds = append(cmds, cmd)
	case 2:
		m.projectForm.parentInput, cmd = m.projectForm.parentInput.Update(msg)
		cmds = append(cmds, cmd)
	case 3:
		m.projectForm.colorInput, cmd = m.projectForm.colorInput.Update(msg)
		cmds = append(cmds, cmd)
	case 4:
		m.projectForm.iconInput, cmd = m.projectForm.iconInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateProjectFormFocus() {
	m.projectForm.nameInput.Blur()
	m.projectForm.descInput.Blur()
	m.projectForm.parentInput.Blur()
	m.projectForm.colorInput.Blur()
	m.projectForm.iconInput.Blur()

	switch m.projectForm.focusedField {
	case 0:
		m.projectForm.nameInput.Focus()
	case 1:
		m.projectForm.descInput.Focus()
	case 2:
		m.projectForm.parentInput.Focus()
	case 3:
		m.projectForm.colorInput.Focus()
	case 4:
		m.projectForm.iconInput.Focus()
	}
}

func (m Model) handleSaveProject() (tea.Model, tea.Cmd) {
	if !m.validateProjectForm() {
		return m, nil
	}

	name := strings.TrimSpace(m.projectForm.nameInput.Value())
	desc := strings.TrimSpace(m.projectForm.descInput.Value())
	parentInput := strings.TrimSpace(m.projectForm.parentInput.Value())
	color := strings.TrimSpace(m.projectForm.colorInput.Value())
	icon := strings.TrimSpace(m.projectForm.iconInput.Value())

	var parentID *int64
	if parentInput != "" {
		parent := m.lookupProjectForForm(parentInput)
		if parent != nil {
			parentID = &parent.ID
		} else {
			m.projectForm.errors["parent"] = "Parent project not found"
			return m, nil
		}
	}

	if m.projectForm.mode == createProjectMode {
		project := domain.NewProject(name)
		project.Description = desc
		project.ParentID = parentID
		project.Color = color
		project.Icon = icon

		if err := project.Validate(); err != nil {
			m.projectForm.errors["general"] = err.Error()
			return m, nil
		}

		m.loading = true
		return m, createProjectCmd(m.ctx, m.projectRepo, project)
	} else {
		project := m.projectForm.editingProject
		if project == nil {
			m.projectForm.errors["general"] = "No project to edit"
			return m, nil
		}

		project.Name = name
		project.Description = desc
		project.ParentID = parentID
		project.Color = color
		project.Icon = icon

		statuses := []domain.ProjectStatus{domain.ProjectStatusActive, domain.ProjectStatusArchived, domain.ProjectStatusCompleted}
		if m.projectForm.statusIdx >= 0 && m.projectForm.statusIdx < len(statuses) {
			project.Status = statuses[m.projectForm.statusIdx]
		}

		if err := project.Validate(); err != nil {
			m.projectForm.errors["general"] = err.Error()
			return m, nil
		}

		m.loading = true
		return m, updateProjectCmd(m.ctx, m.projectRepo, project)
	}
}


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
		m.editForm.editingTask = task
		m.editForm.titleInput.SetValue(task.Title)
		m.editForm.descInput.SetValue(task.Description)
		m.editForm.projectInput.SetValue(task.ProjectName)
		if len(task.Tags) > 0 {
			m.editForm.tagsInput.SetValue(strings.Join(task.Tags, ", "))
		}
		if task.DueDate != nil {
			m.editForm.dueDateInput.SetValue(task.DueDate.Format("2006-01-02"))
		}

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
		m.editForm.editingTask = nil
		m.editForm.priorityIdx = 1
		m.editForm.statusIdx = 0
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
	title := strings.TrimSpace(m.editForm.titleInput.Value())
	if title == "" {
		m.editForm.err = "Title is required"
		return m, nil
	}

	description := strings.TrimSpace(m.editForm.descInput.Value())

	projectName := strings.TrimSpace(m.editForm.projectInput.Value())
	var projectID *int64
	if projectName != "" {
		for _, proj := range m.projects {
			if strings.EqualFold(proj.Name, projectName) {
				projectID = &proj.ID
				break
			}
		}
	}

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

	dueDateStr := strings.TrimSpace(m.editForm.dueDateInput.Value())

	priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
	statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}

	if m.editForm.isNewTask {
		task := domain.NewTask(title)
		task.Description = description
		task.ProjectID = projectID
		task.Tags = tags
		task.Priority = priorities[m.editForm.priorityIdx]
		task.Status = statuses[m.editForm.statusIdx]

		if dueDateStr != "" {
			dueTime, err := domain.ParseDueDate(dueDateStr)
			if err == nil {
				task.DueDate = dueTime
			}
		}

		m.loading = true
		return m, createTaskCmd(m.ctx, m.repo, task)
	} else {
		task := m.editForm.editingTask
		task.Title = title
		task.Description = description
		task.ProjectID = projectID
		task.Tags = tags
		task.Priority = priorities[m.editForm.priorityIdx]
		task.Status = statuses[m.editForm.statusIdx]

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


func (m Model) handleToggleMultiSelect() (tea.Model, tea.Cmd) {
	m.multiSelect.enabled = !m.multiSelect.enabled
	if !m.multiSelect.enabled {
		m.multiSelect.selectedTasks = make(map[int64]bool)
	}
	m.updateTableRows()
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

	m.updateTableRows()
	return m, nil
}

func (m Model) handleSelectAll() (tea.Model, tea.Cmd) {
	for _, task := range m.tasks {
		m.multiSelect.selectedTasks[task.ID] = true
	}
	m.updateTableRows()
	return m, nil
}

func (m Model) handleDeselectAll() (tea.Model, tea.Cmd) {
	m.multiSelect.selectedTasks = make(map[int64]bool)
	m.updateTableRows()
	return m, nil
}


func (m Model) handleBulkMarkComplete() (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	for _, task := range m.tasks {
		if m.multiSelect.selectedTasks[task.ID] {
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
			switch task.Status {
				case domain.StatusPending:
					task.Status = domain.StatusInProgress
				case domain.StatusInProgress:
					task.Status = domain.StatusPending
				default:
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


func (m Model) handleProjectViewKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visibleNodes := m.getVisibleProjectNodes()
	if len(visibleNodes) == 0 {
		return m, nil
	}

	if m.projectCursor < 0 {
		m.projectCursor = 0
	}
	if m.projectCursor >= len(visibleNodes) {
		m.projectCursor = len(visibleNodes) - 1
	}

	switch {
	case key.Matches(msg, m.keys.ToggleProjects):
		m.viewMode = tableView
		m.message = ""
		return m, nil

	case key.Matches(msg, m.keys.Back):
		m.viewMode = tableView
		m.selectedProject = nil
		m.message = ""
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.projectCursor > 0 {
			m.projectCursor--
			m.selectedProject = visibleNodes[m.projectCursor].project
			return m, fetchProjectStatsCmd(m.ctx, m.projectRepo, m.selectedProject.ID)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.projectCursor < len(visibleNodes)-1 {
			m.projectCursor++
			m.selectedProject = visibleNodes[m.projectCursor].project
			return m, fetchProjectStatsCmd(m.ctx, m.projectRepo, m.selectedProject.ID)
		}
		return m, nil

	case key.Matches(msg, m.keys.ExpandProject):
		if m.projectCursor < len(visibleNodes) {
			node := visibleNodes[m.projectCursor]
			if len(node.children) > 0 {
				m.projectExpanded[node.project.ID] = true
				m.message = fmt.Sprintf("Expanded: %s", node.project.Name)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.CollapseProject):
		if m.projectCursor < len(visibleNodes) {
			node := visibleNodes[m.projectCursor]
			if len(node.children) > 0 && m.projectExpanded[node.project.ID] {
				m.projectExpanded[node.project.ID] = false
				m.message = fmt.Sprintf("Collapsed: %s", node.project.Name)
			} else if node.parent != nil {
				for i, n := range visibleNodes {
					if n.project.ID == node.parent.project.ID {
						m.projectCursor = i
						m.selectedProject = n.project
						return m, fetchProjectStatsCmd(m.ctx, m.projectRepo, n.project.ID)
					}
				}
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.ViewProject):
		if m.projectCursor < len(visibleNodes) {
			node := visibleNodes[m.projectCursor]
			m.selectedProject = node.project
			m.message = fmt.Sprintf("Selected: %s", node.project.Name)
			return m, fetchProjectStatsCmd(m.ctx, m.projectRepo, node.project.ID)
		}
		return m, nil

	case key.Matches(msg, m.keys.NewProject):
		m.initNewProjectForm()
		m.viewMode = projectFormView
		m.message = "Creating new project"
		return m, nil

	case key.Matches(msg, m.keys.EditProject):
		if m.projectCursor < len(visibleNodes) {
			node := visibleNodes[m.projectCursor]
			m.initEditProjectForm(node.project)
			m.viewMode = projectFormView
			m.message = fmt.Sprintf("Editing project: %s", node.project.Name)
		}
		return m, nil

	case key.Matches(msg, m.keys.DeleteProject):
		if m.projectCursor < len(visibleNodes) {
			node := visibleNodes[m.projectCursor]
			project := node.project

			childCount := len(node.children)

			confirmMsg := fmt.Sprintf("Delete project '%s' (ID: %d)?", project.Name, project.ID)
			if childCount > 0 {
				confirmMsg += fmt.Sprintf("\n  - %d child project(s) will be deleted", childCount)
			}
			confirmMsg += "\n  - Associated tasks will be orphaned (project_id set to NULL)"

			m.confirm = confirmDialog{
				message: confirmMsg,
				active:  true,
				onConfirm: func(model *Model) tea.Cmd {
					return deleteProjectCmd(model.ctx, model.projectRepo, project.ID)
				},
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.ArchiveProject):
		if m.projectCursor < len(visibleNodes) {
			node := visibleNodes[m.projectCursor]
			project := node.project

			isArchived := project.Status == domain.ProjectStatusArchived
			action := "Archive"
			if isArchived {
				action = "Unarchive"
			}

			childCount := len(node.children)
			confirmMsg := fmt.Sprintf("%s project '%s' (ID: %d)?", action, project.Name, project.ID)
			if !isArchived && childCount > 0 {
				confirmMsg += fmt.Sprintf("\n  - %d child project(s) will also be archived", childCount)
			}

			m.confirm = confirmDialog{
				message: confirmMsg,
				active:  true,
				onConfirm: func(model *Model) tea.Cmd {
					if isArchived {
						updatedProject := *project
						updatedProject.Status = domain.ProjectStatusActive
						return updateProjectCmd(model.ctx, model.projectRepo, &updatedProject)
					}
					return archiveProjectCmd(model.ctx, model.projectRepo, project.ID)
				},
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.ViewNotes):
		if m.selectedProject != nil && m.selectedProject.HasNotes() {
			m.initNotesViewer(m.selectedProject)
			m.viewMode = notesView
			m.message = "Viewing notes"
		} else if m.selectedProject != nil {
			m.message = "No notes for this project. Use 'project note' command to add notes."
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		projectFilter := repository.ProjectFilter{ExcludeArchived: true}
		return m, fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter)

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.ViewPicker):
		m.viewPicker.active = true
		m.viewPicker.cursor = 0
		m.viewPicker.views = m.savedViews
		m.message = "View Picker (↑/↓: navigate, enter: select, esc: cancel)"
		return m, nil

	case key.Matches(msg, m.keys.FavoriteViews):
		if len(m.favoriteViews) > 0 {
			m.viewPicker.active = true
			m.viewPicker.cursor = 0
			m.viewPicker.views = m.favoriteViews
			m.message = "Favorite Views (↑/↓: navigate, enter: select, esc: cancel)"
		} else {
			m.message = "No favorite views"
		}
		return m, nil

	case key.Matches(msg, m.keys.QuickAccess1):
		return m.applyQuickAccessView(1)
	case key.Matches(msg, m.keys.QuickAccess2):
		return m.applyQuickAccessView(2)
	case key.Matches(msg, m.keys.QuickAccess3):
		return m.applyQuickAccessView(3)
	case key.Matches(msg, m.keys.QuickAccess4):
		return m.applyQuickAccessView(4)
	case key.Matches(msg, m.keys.QuickAccess5):
		return m.applyQuickAccessView(5)
	case key.Matches(msg, m.keys.QuickAccess6):
		return m.applyQuickAccessView(6)
	case key.Matches(msg, m.keys.QuickAccess7):
		return m.applyQuickAccessView(7)
	case key.Matches(msg, m.keys.QuickAccess8):
		return m.applyQuickAccessView(8)
	case key.Matches(msg, m.keys.QuickAccess9):
		return m.applyQuickAccessView(9)
	}

	return m, nil
}

func (m Model) handleNotesViewKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Back):
		m.notesViewer.active = false
		m.viewMode = projectView
		m.message = "Closed notes viewer"
		return m, nil

	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		m.notesViewer.viewport.LineUp(1)
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.notesViewer.viewport.LineDown(1)
		return m, nil

	case msg.String() == "pgup":
		m.notesViewer.viewport.ViewUp()
		return m, cmd

	case msg.String() == "pgdown":
		m.notesViewer.viewport.ViewDown()
		return m, cmd
	}

	return m, nil
}

func (m Model) updateViewPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.viewPicker.active = false
			return m, nil

		case "up", "k":
			if m.viewPicker.cursor > 0 {
				m.viewPicker.cursor--
			}
			return m, nil

		case "down", "j":
			if m.viewPicker.cursor < len(m.viewPicker.views)-1 {
				m.viewPicker.cursor++
			}
			return m, nil

		case "enter":
			if m.viewPicker.cursor < len(m.viewPicker.views) {
				selectedView := m.viewPicker.views[m.viewPicker.cursor]
				m.viewPicker.selected = selectedView

				m.viewPicker.active = false

				return m, applyViewCmd(m.ctx, m.viewRepo, selectedView.ID)
			}
			return m, nil
		}
	case viewsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to load views: %v", msg.err)
			return m, nil
		}
		m.savedViews = msg.views

		m.favoriteViews = []*domain.SavedView{}
		m.quickAccessViews = make(map[int]*domain.SavedView)

		for _, v := range m.savedViews {
			if v.HotKey != nil && *v.HotKey >= 1 && *v.HotKey <= 9 {
				m.quickAccessViews[*v.HotKey] = v
			}
			if v.IsFavorite {
				m.favoriteViews = append(m.favoriteViews, v)
			}
		}

		return m, nil

	case viewAppliedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to apply view: %v", msg.err)
			return m, nil
		}

		m.selectedView = msg.view
		m.filter = m.convertViewFilterToTaskFilter(msg.view.FilterConfig)
		m.currentPage = 1
		m.message = fmt.Sprintf("Applied view: %s", msg.view.Name)

		m.loading = true
		return m, fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize)

	case viewCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to create view: %v", msg.err)
			return m, nil
		}
		m.savedViews = append(m.savedViews, msg.view)
		m.message = fmt.Sprintf("Created view: %s", msg.view.Name)
		return m, nil

	case viewUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to update view: %v", msg.err)
			return m, nil
		}
		for i, v := range m.savedViews {
			if v.ID == msg.view.ID {
				m.savedViews[i] = msg.view
				break
			}
		}
		m.message = fmt.Sprintf("Updated view: %s", msg.view.Name)
		return m, nil

	case viewDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.message = fmt.Sprintf("Failed to delete view: %v", msg.err)
			return m, nil
		}
		m.savedViews = sliceRemoveByID(m.savedViews, msg.viewID)
		m.message = "View deleted"
		return m, nil

	case searchHistoryLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.searchHistory = msg.history
		return m, nil

	case searchRecordedMsg:
		if msg.err != nil {
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m Model) applyQuickAccessView(hotKey int) (tea.Model, tea.Cmd) {
	view, exists := m.quickAccessViews[hotKey]
	if !exists {
		m.message = fmt.Sprintf("No view assigned to key %d", hotKey)
		return m, nil
	}

	m.selectedView = view
	m.filter = m.convertViewFilterToTaskFilter(view.FilterConfig)
	m.currentPage = 1
	m.message = fmt.Sprintf("Applied view: %s", view.Name)

	_ = m.viewRepo.RecordViewAccess(m.ctx, view.ID)

	m.loading = true
	return m, fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize)
}

func (m *Model) convertViewFilterToTaskFilter(vf domain.SavedViewFilter) repository.TaskFilter {
	return repository.TaskFilter{
		Status:      vf.Status,
		Priority:    vf.Priority,
		ProjectID:   vf.ProjectID,
		Tags:        vf.Tags,
		SearchQuery: vf.SearchQuery,
		SearchMode:  vf.SearchMode,
		SortBy:      vf.SortBy,
		SortOrder:   vf.SortOrder,
		DueDateFrom: vf.DueDateFrom,
		DueDateTo:   vf.DueDateTo,
	}
}

func sliceRemoveByID(views []*domain.SavedView, id int64) []*domain.SavedView {
	result := make([]*domain.SavedView, 0, len(views))
	for _, v := range views {
		if v.ID != id {
			result = append(result, v)
		}
	}
	return result
}
