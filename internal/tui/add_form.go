package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/theme"
)

type AddFormModel struct {
	ctx         context.Context
	projectRepo repository.ProjectRepository
	theme       *theme.Theme
	styles      *theme.Styles

	// form inputs
	titleInput    textinput.Model
	descInput     textarea.Model
	projectInput  textinput.Model
	tagsInput     textinput.Model
	dueDateInput  textinput.Model
	focusedField  int
	priorityIdx   int
	statusIdx     int

	// project picker
	projectPicker struct {
		active   bool
		projects []*domain.Project
		cursor   int
	}

	// state
	width    int
	height   int
	err      string
	quitting bool
	saved    bool

	// result
	createdTask *domain.Task
}

func NewAddFormModel(ctx context.Context, projectRepo repository.ProjectRepository, themeObj *theme.Theme, styles *theme.Styles) AddFormModel {
	titleInput := textinput.New()
	titleInput.Placeholder = "Task title (required)"
	titleInput.CharLimit = 200
	titleInput.Width = 60
	titleInput.Focus()

	descInput := textarea.New()
	descInput.Placeholder = "Task description (optional)"
	descInput.CharLimit = 1000
	descInput.SetWidth(60)
	descInput.SetHeight(5)

	projectInput := textinput.New()
	projectInput.Placeholder = "Project name or ID (optional)"
	projectInput.CharLimit = 100
	projectInput.Width = 40

	tagsInput := textinput.New()
	tagsInput.Placeholder = "Tags (comma-separated, optional)"
	tagsInput.CharLimit = 200
	tagsInput.Width = 60

	dueDateInput := textinput.New()
	dueDateInput.Placeholder = "YYYY-MM-DD (optional)"
	dueDateInput.CharLimit = 10
	dueDateInput.Width = 20

	return AddFormModel{
		ctx:          ctx,
		projectRepo:  projectRepo,
		theme:        themeObj,
		styles:       styles,
		titleInput:   titleInput,
		descInput:    descInput,
		projectInput: projectInput,
		tagsInput:    tagsInput,
		dueDateInput: dueDateInput,
		focusedField: 0,
		priorityIdx:  1,
		statusIdx:    0,
		width:        100,
		height:       30,
	}
}

func (m AddFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AddFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.projectPicker.active {
		return m.updateProjectPicker(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.err != "" {
				m.err = ""
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s", "ctrl+enter":
			if m.validateForm() {
				task, err := m.createTask()
				if err != nil {
					m.err = err.Error()
					return m, nil
				}
				m.createdTask = task
				m.saved = true
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case "tab":
			m.focusedField++
			if m.focusedField > 4 {
				m.focusedField = 0
			}
			m.updateFormFocus()
			return m, nil

		case "shift+tab":
			m.focusedField--
			if m.focusedField < 0 {
				m.focusedField = 4
			}
			m.updateFormFocus()
			return m, nil

		case "ctrl+p":
			if m.focusedField == 2 {
				projects, err := m.projectRepo.List(m.ctx, repository.ProjectFilter{ExcludeArchived: true})
				if err == nil && len(projects) > 0 {
					m.projectPicker.active = true
					m.projectPicker.projects = projects
					m.projectPicker.cursor = 0
					return m, nil
				}
			} else {
				priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
				m.priorityIdx = (m.priorityIdx + 1) % len(priorities)
			}
			return m, nil

		case "ctrl+t":
			statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}
			m.statusIdx = (m.statusIdx + 1) % len(statuses)
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.focusedField {
	case 0:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case 1:
		m.descInput, cmd = m.descInput.Update(msg)
	case 2:
		m.projectInput, cmd = m.projectInput.Update(msg)
	case 3:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	case 4:
		m.dueDateInput, cmd = m.dueDateInput.Update(msg)
	}

	return m, cmd
}

func (m *AddFormModel) updateProjectPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.projectPicker.cursor < len(m.projectPicker.projects)-1 {
				m.projectPicker.cursor++
			}
			return m, nil

		case "enter":
			if m.projectPicker.cursor < len(m.projectPicker.projects) {
				selected := m.projectPicker.projects[m.projectPicker.cursor]
				m.projectInput.SetValue(selected.Name)
				m.projectPicker.active = false
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *AddFormModel) updateFormFocus() {
	m.titleInput.Blur()
	m.descInput.Blur()
	m.projectInput.Blur()
	m.tagsInput.Blur()
	m.dueDateInput.Blur()

	switch m.focusedField {
	case 0:
		m.titleInput.Focus()
	case 1:
		m.descInput.Focus()
	case 2:
		m.projectInput.Focus()
	case 3:
		m.tagsInput.Focus()
	case 4:
		m.dueDateInput.Focus()
	}
}

func (m *AddFormModel) validateForm() bool {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.err = "Title is required"
		return false
	}
	m.err = ""
	return true
}

func (m *AddFormModel) createTask() (*domain.Task, error) {
	title := strings.TrimSpace(m.titleInput.Value())
	description := strings.TrimSpace(m.descInput.Value())

	priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
	statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}

	task := domain.NewTask(title)
	task.Description = description
	task.Priority = priorities[m.priorityIdx]
	task.Status = statuses[m.statusIdx]

	projectName := strings.TrimSpace(m.projectInput.Value())
	if projectName != "" {
		projects, err := m.projectRepo.List(m.ctx, repository.ProjectFilter{ExcludeArchived: true})
		if err == nil {
			for _, proj := range projects {
				if strings.EqualFold(proj.Name, projectName) {
					task.ProjectID = &proj.ID
					break
				}
			}
		}
	}

	tagsStr := strings.TrimSpace(m.tagsInput.Value())
	if tagsStr != "" {
		var tags []string
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		task.Tags = tags
	} else {
		task.Tags = []string{}
	}

	dueDateStr := strings.TrimSpace(m.dueDateInput.Value())
	if dueDateStr != "" {
		dueDate, err := domain.ParseDueDate(dueDateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid due date format: %w", err)
		}
		task.DueDate = dueDate
	}

	return task, nil
}

func (m AddFormModel) View() string {
	if m.quitting {
		if m.saved {
			return ""
		}
		return m.styles.Info.Render("Task creation cancelled.\n")
	}

	if m.projectPicker.active {
		return m.renderProjectPicker()
	}

	return m.renderForm()
}

func (m AddFormModel) renderForm() string {
	priorities := []domain.Priority{domain.PriorityLow, domain.PriorityMedium, domain.PriorityHigh, domain.PriorityUrgent}
	statuses := []domain.Status{domain.StatusPending, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}

	var b strings.Builder

	header := m.styles.TUITitle.Render("Create New Task")
	b.WriteString(header)
	b.WriteString("\n\n")

	titleLabel := m.styles.DetailLabel.Render("Title:")
	if m.focusedField == 0 {
		titleLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Primary)).
			Bold(true).
			Render("▶ Title:")
	}
	b.WriteString(titleLabel)
	b.WriteString("\n")
	b.WriteString(m.titleInput.View())
	b.WriteString("\n\n")

	descLabel := m.styles.DetailLabel.Render("Description:")
	if m.focusedField == 1 {
		descLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Primary)).
			Bold(true).
			Render("▶ Description:")
	}
	b.WriteString(descLabel)
	b.WriteString("\n")
	b.WriteString(m.descInput.View())
	b.WriteString("\n\n")

	projectLabel := m.styles.DetailLabel.Render("Project:")
	if m.focusedField == 2 {
		projectLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Primary)).
			Bold(true).
			Render("▶ Project:")
	}
	projectHint := m.styles.TUISubtitle.Render(" (Ctrl+P: picker)")
	b.WriteString(projectLabel)
	b.WriteString(projectHint)
	b.WriteString("\n")
	b.WriteString(m.projectInput.View())
	b.WriteString("\n\n")

	tagsLabel := m.styles.DetailLabel.Render("Tags:")
	if m.focusedField == 3 {
		tagsLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Primary)).
			Bold(true).
			Render("▶ Tags:")
	}
	b.WriteString(tagsLabel)
	b.WriteString("\n")
	b.WriteString(m.tagsInput.View())
	b.WriteString("\n\n")

	dueDateLabel := m.styles.DetailLabel.Render("Due Date:")
	if m.focusedField == 4 {
		dueDateLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Primary)).
			Bold(true).
			Render("▶ Due Date:")
	}
	b.WriteString(dueDateLabel)
	b.WriteString("\n")
	b.WriteString(m.dueDateInput.View())
	b.WriteString("\n\n")

	priorityLabel := m.styles.DetailLabel.Render("Priority:")
	priorityValue := m.styles.GetPriorityTextStyle(priorities[m.priorityIdx]).Render(string(priorities[m.priorityIdx]))
	priorityHint := m.styles.TUISubtitle.Render(" (Ctrl+P to cycle)")
	b.WriteString(priorityLabel)
	b.WriteString(" ")
	b.WriteString(priorityValue)
	b.WriteString(priorityHint)
	b.WriteString("\n")

	statusLabel := m.styles.DetailLabel.Render("Status:")
	statusValue := m.styles.GetStatusStyle(statuses[m.statusIdx]).Render(string(statuses[m.statusIdx]))
	statusHint := m.styles.TUISubtitle.Render(" (Ctrl+T to cycle)")
	b.WriteString(statusLabel)
	b.WriteString(" ")
	b.WriteString(statusValue)
	b.WriteString(statusHint)
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}

	sepWidth := 60
	sep := strings.Repeat("─", sepWidth)
	b.WriteString(m.styles.Separator.Render(sep))
	b.WriteString("\n")

	help := m.styles.TUIHelp.Render("Tab/Shift+Tab: navigate • Ctrl+S: save • Esc: cancel")
	b.WriteString(help)

	contentStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(1, 2).
		Width(70)

	return contentStyle.Render(b.String())
}

func (m AddFormModel) renderProjectPicker() string {
	var b strings.Builder

	header := m.styles.TUITitle.Render("Select Project")
	b.WriteString(header)
	b.WriteString("\n\n")

	for i, proj := range m.projectPicker.projects {
		prefix := "  "
		if i == m.projectPicker.cursor {
			prefix = "▶ "
		}

		line := fmt.Sprintf("%s%s", prefix, proj.Name)

		if i == m.projectPicker.cursor {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.SelectedFg)).
				Background(lipgloss.Color(m.theme.SelectedBg)).
				Bold(true).
				Width(50).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.TextSecondary)).
				Width(50).
				Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := m.styles.TUIHelp.Render("↑/k: up • ↓/j: down • Enter: select • Esc: cancel")
	b.WriteString(help)

	contentStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.BorderColor)).
		Padding(1, 2).
		Width(56)

	return contentStyle.Render(b.String())
}

func (m AddFormModel) GetCreatedTask() *domain.Task {
	return m.createdTask
}
