package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
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
	editView
	projectView
	projectFormView
	templateView
	viewPickerView
	notesView
)

type uiMode int

const (
	normalMode uiMode = iota
	filteringMode
	searchingMode
	confirmingMode
)

type confirmDialog struct {
	message   string
	onConfirm func(m *Model) tea.Cmd
	active    bool
}

type ProjectTree struct {
	roots   []*ProjectTreeNode
	flatMap map[int64]*ProjectTreeNode
}

type ProjectTreeNode struct {
	project  *domain.Project
	parent   *ProjectTreeNode
	children []*ProjectTreeNode
	depth    int
	expanded bool
}

type projectStatsData struct {
	taskCount int
	stats     map[domain.Status]int
}

type ProjectPicker struct {
	active      bool
	projects    []*domain.Project
	tree        *ProjectTree
	cursor      int
	searchQuery string
	selected    *domain.Project
}

type TemplatePicker struct {
	active      bool
	templates   []*domain.ProjectTemplate
	cursor      int
	searchQuery string
	selected    *domain.ProjectTemplate
}

type ViewPicker struct {
	active      bool
	views       []*domain.SavedView
	cursor      int
	searchQuery string
	selected    *domain.SavedView
}

type notesViewer struct {
	active   bool
	project  *domain.Project
	viewport viewport.Model
}

type Model struct {
	repo         repository.TaskRepository
	projectRepo  repository.ProjectRepository
	templateRepo repository.TemplateRepository
	tasks        []*domain.Task
	totalCount   int64

	projects         []*domain.Project
	projectTree      *ProjectTree
	selectedProject  *domain.Project
	projectExpanded  map[int64]bool
	projectCursor    int
	projectPicker    ProjectPicker
	projectStats     map[int64]projectStatsData

	templates        []*domain.ProjectTemplate
	templatePicker   TemplatePicker
	selectedTemplate *domain.ProjectTemplate

	viewRepo         repository.ViewRepository
	savedViews       []*domain.SavedView
	viewPicker       ViewPicker
	selectedView     *domain.SavedView
	favoriteViews    []*domain.SavedView
	quickAccessViews map[int]*domain.SavedView

	searchHistoryRepo repository.SearchHistoryRepository
	searchHistory     []*domain.SearchHistory
	historyDropdown   struct {
		active bool
		cursor int
		height int
	}

	filter          repository.TaskFilter
	currentPage     int
	pageSize        int
	fuzzyMode       bool
	fuzzyThreshold  int

	queryMode       bool
	queryString     string
	showQueryHelp   bool

	table        table.Model
	searchInput  textinput.Model
	keys         keyMap

	viewMode     viewMode
	uiMode       uiMode
	selectedTask *domain.Task

	filterPanel  filterPanel

	editForm     editForm

	projectForm  projectForm

	multiSelect  multiSelectState

	confirm      confirmDialog

	notesViewer  notesViewer

	err          error
	width        int
	height       int
	showHelp     bool
	loading      bool
	message      string

	theme        *theme.Theme
	styles       *theme.Styles

	ctx          context.Context
}

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
	filterType  string
}

type editForm struct {
	active         bool
	isNewTask      bool
	editingTask    *domain.Task
	titleInput     textinput.Model
	descInput      textarea.Model
	projectInput   textinput.Model
	tagsInput      textinput.Model
	dueDateInput   textinput.Model
	focusedField   int
	priorityIdx    int
	statusIdx      int
	err            string
}

type multiSelectState struct {
	enabled       bool
	selectedTasks map[int64]bool
}

type projectFormMode int

const (
	createProjectMode projectFormMode = iota
	editProjectMode
)

type projectForm struct {
	active        bool
	mode          projectFormMode
	editingProject *domain.Project
	nameInput     textinput.Model
	descInput     textarea.Model
	parentInput   textinput.Model
	colorInput    textinput.Model
	iconInput     textinput.Model
	focusedField  int
	statusIdx     int
	errors        map[string]string
}

func NewModel(repo repository.TaskRepository, projectRepo repository.ProjectRepository, viewRepo repository.ViewRepository, searchHistoryRepo repository.SearchHistoryRepository, initialFilter repository.TaskFilter, pageSize int, themeObj *theme.Theme, styles *theme.Styles) Model {
	columns := []table.Column{
		{Title: "Status", Width: 15},
		{Title: "Priority", Width: 12},
		{Title: "Title", Width: 45},
		{Title: "Project", Width: 15},
		{Title: "Tags", Width: 20},
		{Title: "Due", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(20),
	)

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

	si := textinput.New()
	si.Placeholder = "Search tasks..."
	si.CharLimit = 100
	si.Width = 50

	if initialFilter.SortBy == "" {
		initialFilter.SortBy = "created_at"
	}
	if initialFilter.SortOrder == "" {
		initialFilter.SortOrder = "desc"
	}

	if pageSize == 0 {
		pageSize = 20
	}

	return Model{
		repo:              repo,
		projectRepo:       projectRepo,
		viewRepo:          viewRepo,
		searchHistoryRepo: searchHistoryRepo,
		tasks:             []*domain.Task{},
		projects:          []*domain.Project{},
		projectExpanded:   make(map[int64]bool),
		projectStats:      make(map[int64]projectStatsData),
		savedViews:        []*domain.SavedView{},
		favoriteViews:     []*domain.SavedView{},
		quickAccessViews:  make(map[int]*domain.SavedView),
		searchHistory:     []*domain.SearchHistory{},
		filter:            initialFilter,
		currentPage:       1,
		pageSize:          pageSize,
		fuzzyMode:         false,
		fuzzyThreshold:    60,
		table:             t,
		searchInput:       si,
		keys:              defaultKeyMap(),
		viewMode:          tableView,
		uiMode:            normalMode,
		multiSelect: multiSelectState{
			enabled:       false,
			selectedTasks: make(map[int64]bool),
		},
		historyDropdown: struct {
			active bool
			cursor int
			height int
		}{
			active: false,
			cursor: 0,
			height: 5,
		},
		theme:  themeObj,
		styles: styles,
		ctx:    context.Background(),
	}
}

func (m Model) Init() tea.Cmd {
	projectFilter := repository.ProjectFilter{
		ExcludeArchived: true,
	}
	return tea.Batch(
		fetchTasksCmd(m.ctx, m.repo, m.filter, m.currentPage, m.pageSize),
		fetchProjectsCmd(m.ctx, m.projectRepo, projectFilter),
		fetchViewsCmd(m.ctx, m.viewRepo),
		fetchSearchHistoryCmd(m.ctx, m.searchHistoryRepo, 50),
	)
}

func (m *Model) taskToRow(task *domain.Task) table.Row {
	checkbox := "  "
	if m.multiSelect.enabled {
		if m.multiSelect.selectedTasks[task.ID] {
			checkbox = "☑ "
		} else {
			checkbox = "☐ "
		}
	}

	// status
	statusIcon := display.GetStatusIcon(task.Status)
	status := fmt.Sprintf("%s%s %s", checkbox, statusIcon, task.Status)

	// priority
	priorityIcon := display.GetPriorityIcon(task.Priority)
	priority := fmt.Sprintf("%s %s", priorityIcon, task.Priority)

	// truncate title
	title := task.Title
	if len(title) > 37 {
		title = title[:37] + "..."
	}

	// project
	project := task.ProjectName
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

	var rowStyle lipgloss.Style
	hasColor := false
	if task.ProjectID != nil {
		for _, p := range m.projects {
			if p.ID == *task.ProjectID {
				if p.Color != "" {
					rowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Color))
					hasColor = true
				}
				break
			}
		}
	}

	if hasColor {
		status = rowStyle.Render(status)
		priority = rowStyle.Render(priority)
		title = rowStyle.Render(title)
		project = rowStyle.Render(project)
		tags = rowStyle.Render(tags)
		dueDate = rowStyle.Render(dueDate)
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

func buildProjectTree(projects []*domain.Project) *ProjectTree {
	if len(projects) == 0 {
		return &ProjectTree{
			roots:   []*ProjectTreeNode{},
			flatMap: make(map[int64]*ProjectTreeNode),
		}
	}

	tree := &ProjectTree{
		roots:   []*ProjectTreeNode{},
		flatMap: make(map[int64]*ProjectTreeNode),
	}

	// create all nodes
	for _, project := range projects {
		node := &ProjectTreeNode{
			project:  project,
			parent:   nil,
			children: []*ProjectTreeNode{},
			depth:    0,
			expanded: false,
		}
		tree.flatMap[project.ID] = node
	}

	// build relationships
	var roots []*ProjectTreeNode
	for _, node := range tree.flatMap {
		if node.project.ParentID == nil {
			roots = append(roots, node)
		} else {
			parent, exists := tree.flatMap[*node.project.ParentID]
			if exists {
				node.parent = parent
				parent.children = append(parent.children, node)
			} else {
				roots = append(roots, node)
			}
		}
	}

	for _, root := range roots {
		calculateDepth(root, 0)
	}

	tree.roots = roots
	return tree
}

func calculateDepth(node *ProjectTreeNode, depth int) {
	if node == nil {
		return
	}
	node.depth = depth
	for _, child := range node.children {
		calculateDepth(child, depth+1)
	}
}

func (m *Model) getVisibleProjectNodes() []*ProjectTreeNode {
	if m.projectTree == nil {
		return []*ProjectTreeNode{}
	}

	var visible []*ProjectTreeNode
	var collectVisible func(nodes []*ProjectTreeNode)
	collectVisible = func(nodes []*ProjectTreeNode) {
		for _, node := range nodes {
			visible = append(visible, node)
			if m.projectExpanded[node.project.ID] && len(node.children) > 0 {
				collectVisible(node.children)
			}
		}
	}

	collectVisible(m.projectTree.roots)
	return visible
}

func (m *Model) initNewProjectForm() {
	m.projectForm.active = true
	m.projectForm.mode = createProjectMode
	m.projectForm.editingProject = nil
	m.projectForm.focusedField = 0
	m.projectForm.statusIdx = 0
	m.projectForm.errors = make(map[string]string)

	m.projectForm.nameInput = textinput.New()
	m.projectForm.nameInput.Placeholder = "Project name"
	m.projectForm.nameInput.CharLimit = 100
	m.projectForm.nameInput.Width = 50
	m.projectForm.nameInput.Focus()

	m.projectForm.descInput = textarea.New()
	m.projectForm.descInput.Placeholder = "Project description (optional)"
	m.projectForm.descInput.CharLimit = 500
	m.projectForm.descInput.SetWidth(50)
	m.projectForm.descInput.SetHeight(3)

	m.projectForm.parentInput = textinput.New()
	m.projectForm.parentInput.Placeholder = "Parent project name or ID (optional)"
	m.projectForm.parentInput.CharLimit = 100
	m.projectForm.parentInput.Width = 50

	m.projectForm.colorInput = textinput.New()
	m.projectForm.colorInput.Placeholder = "Color (e.g., blue, red, green)"
	m.projectForm.colorInput.CharLimit = 20
	m.projectForm.colorInput.Width = 50

	m.projectForm.iconInput = textinput.New()
	m.projectForm.iconInput.Placeholder = "Icon emoji (e.g., 📦, 🚀)"
	m.projectForm.iconInput.CharLimit = 10
	m.projectForm.iconInput.Width = 50
}

func (m *Model) initEditProjectForm(project *domain.Project) {
	m.projectForm.active = true
	m.projectForm.mode = editProjectMode
	m.projectForm.editingProject = project
	m.projectForm.focusedField = 0
	m.projectForm.errors = make(map[string]string)

	m.projectForm.nameInput = textinput.New()
	m.projectForm.nameInput.SetValue(project.Name)
	m.projectForm.nameInput.Placeholder = "Project name"
	m.projectForm.nameInput.CharLimit = 100
	m.projectForm.nameInput.Width = 50
	m.projectForm.nameInput.Focus()

	m.projectForm.descInput = textarea.New()
	m.projectForm.descInput.SetValue(project.Description)
	m.projectForm.descInput.Placeholder = "Project description (optional)"
	m.projectForm.descInput.CharLimit = 500
	m.projectForm.descInput.SetWidth(50)
	m.projectForm.descInput.SetHeight(3)

	m.projectForm.parentInput = textinput.New()
	if project.ParentID != nil {
		for _, p := range m.projects {
			if p.ID == *project.ParentID {
				m.projectForm.parentInput.SetValue(p.Name)
				break
			}
		}
	}
	m.projectForm.parentInput.Placeholder = "Parent project name or ID (optional)"
	m.projectForm.parentInput.CharLimit = 100
	m.projectForm.parentInput.Width = 50

	m.projectForm.colorInput = textinput.New()
	m.projectForm.colorInput.SetValue(project.Color)
	m.projectForm.colorInput.Placeholder = "Color (e.g., blue, red, green)"
	m.projectForm.colorInput.CharLimit = 20
	m.projectForm.colorInput.Width = 50

	m.projectForm.iconInput = textinput.New()
	m.projectForm.iconInput.SetValue(project.Icon)
	m.projectForm.iconInput.Placeholder = "Icon emoji (e.g., 📦, 🚀)"
	m.projectForm.iconInput.CharLimit = 10
	m.projectForm.iconInput.Width = 50

	switch project.Status {
	case domain.ProjectStatusActive:
		m.projectForm.statusIdx = 0
	case domain.ProjectStatusArchived:
		m.projectForm.statusIdx = 1
	case domain.ProjectStatusCompleted:
		m.projectForm.statusIdx = 2
	default:
		m.projectForm.statusIdx = 0
	}
}

func (m *Model) resetProjectForm() {
	m.projectForm = projectForm{
		active:        false,
		mode:          createProjectMode,
		editingProject: nil,
		focusedField:  0,
		statusIdx:     0,
		errors:        make(map[string]string),
	}
}

func (m *Model) initNotesViewer(project *domain.Project) {
	m.notesViewer.active = true
	m.notesViewer.project = project

	vp := viewport.New(m.width-6, m.height-10)
	vp.SetContent(project.Notes)
	m.notesViewer.viewport = vp
}

func (m *Model) validateProjectForm() bool {
	m.projectForm.errors = make(map[string]string)

	name := strings.TrimSpace(m.projectForm.nameInput.Value())
	if name == "" {
		m.projectForm.errors["name"] = "Project name is required"
		return false
	}
	if len(name) > 100 {
		m.projectForm.errors["name"] = "Project name cannot exceed 100 characters"
		return false
	}

	desc := strings.TrimSpace(m.projectForm.descInput.Value())
	if len(desc) > 500 {
		m.projectForm.errors["description"] = "Description cannot exceed 500 characters"
		return false
	}

	return len(m.projectForm.errors) == 0
}

func (m *Model) lookupProjectForForm(nameOrID string) *domain.Project {
	nameOrID = strings.TrimSpace(nameOrID)
	if nameOrID == "" {
		return nil
	}

	for _, p := range m.projects {
		if strings.EqualFold(p.Name, nameOrID) {
			return p
		}
	}

	return nil
}

// build breadcrumb path for a task's project
func (m *Model) buildBreadcrumb(task *domain.Task) string {
	if task == nil || task.ProjectID == nil {
		return ""
	}

	var project *domain.Project
	for _, p := range m.projects {
		if p.ID == *task.ProjectID {
			project = p
			break
		}
	}

	if project == nil {
		return ""
	}

	var path []string
	current := project

	for i := 0; i < 20 && current != nil; i++ {
		displayName := current.Name
		if current.Icon != "" {
			displayName = current.Icon + " " + displayName
		}
		path = append([]string{displayName}, path...) // prepend to build root-to-leaf path

		if current.ParentID == nil {
			break
		}

		var parent *domain.Project
		for _, p := range m.projects {
			if p.ID == *current.ParentID {
				parent = p
				break
			}
		}
		current = parent
	}

	if len(path) == 0 {
		return ""
	}

	return strings.Join(path, " > ")
}
