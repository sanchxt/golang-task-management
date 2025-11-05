package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"task-management/internal/config"
	"task-management/internal/display"
	"task-management/internal/domain"
	"task-management/internal/theme"
)

// SetupModel is the model for the initial setup TUI
type SetupModel struct {
	themes        []string
	selectedIndex int
	currentTheme  *theme.Theme
	width         int
	height        int
	quitting      bool
	confirmed     bool
}

// NewSetupModel creates a new setup model
func NewSetupModel() SetupModel {
	themes := theme.ListThemes()
	currentTheme, _ := theme.GetTheme(themes[0])

	return SetupModel{
		themes:        themes,
		selectedIndex: 0,
		currentTheme:  currentTheme,
		width:         100, // default width
		height:        30,  // default height
	}
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.selectedIndex > 0 {
				m.selectedIndex--
				t, _ := theme.GetTheme(m.themes[m.selectedIndex])
				m.currentTheme = t
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.selectedIndex < len(m.themes)-1 {
				m.selectedIndex++
				t, _ := theme.GetTheme(m.themes[m.selectedIndex])
				m.currentTheme = t
			}
			return m, nil

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// save theme selection
			selectedTheme := m.themes[m.selectedIndex]
			if err := config.UpdateTheme(selectedTheme); err != nil {
				// if saving fails, just continue
				fmt.Printf("Warning: failed to save theme: %v\n", err)
			}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m SetupModel) View() string {
	if m.quitting {
		if m.confirmed {
			return ""
		}
		return "Setup cancelled.\n"
	}

	// create styles for the current theme
	styles := theme.NewStyles(m.currentTheme)

	// calculate dimensions with safety checks
	leftWidth := m.width / 3
	if leftWidth < 30 {
		leftWidth = 30
	}
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 30 {
		rightWidth = 30
	}

	// ensure minimum dimensions
	if m.width < 60 || m.height < 10 {
		return "Terminal too small. Please resize and try again.\n"
	}

	// render left side (theme list)
	leftContent := m.renderThemeList(styles, leftWidth)

	// render right side (preview)
	rightContent := m.renderPreview(styles, rightWidth)

	// combine left and right with lipgloss
	left := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.height - 4).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.currentTheme.BorderColor)).
		Padding(1).
		Render(leftContent)

	right := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.height - 4).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.currentTheme.BorderColor)).
		Padding(1).
		Render(rightContent)

	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	// header
	header := styles.TUITitle.Render("TaskFlow Initial Setup")
	subtitle := styles.TUISubtitle.Render("Select a theme to get started")

	// footer
	help := styles.TUIHelp.Render("↑/k: up • ↓/j: down • enter: confirm • q: quit")

	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s", header, subtitle, main, help)
}

func (m SetupModel) renderThemeList(styles *theme.Styles, width int) string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.currentTheme.Primary)).
		Render("Available Themes")

	b.WriteString(title)
	b.WriteString("\n\n")

	for i, themeName := range m.themes {
		prefix := "  "
		if i == m.selectedIndex {
			prefix = "▶ "
		}

		line := fmt.Sprintf("%s%s", prefix, themeName)

		if i == m.selectedIndex {
			// highlight selected theme
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.currentTheme.SelectedFg)).
				Background(lipgloss.Color(m.currentTheme.SelectedBg)).
				Bold(true).
				Width(width - 4).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.currentTheme.TextSecondary)).
				Width(width - 4).
				Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func (m SetupModel) renderPreview(styles *theme.Styles, width int) string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.currentTheme.Primary)).
		Render("Preview")

	b.WriteString(title)
	b.WriteString("\n\n")

	// create sample tasks
	sampleTasks := []*domain.Task{
		{
			ID:          1,
			Title:       "Implement user authentication",
			Description: "Add JWT-based authentication",
			Priority:    domain.PriorityUrgent,
			Status:      domain.StatusInProgress,
			Project:     "backend",
			Tags:        []string{"security", "auth"},
			CreatedAt:   time.Now(),
			DueDate:     timePtr(time.Now().Add(2 * 24 * time.Hour)),
		},
		{
			ID:          2,
			Title:       "Write documentation",
			Description: "Update API docs",
			Priority:    domain.PriorityMedium,
			Status:      domain.StatusPending,
			Project:     "docs",
			Tags:        []string{"documentation"},
			CreatedAt:   time.Now(),
		},
		{
			ID:          3,
			Title:       "Fix login bug",
			Description: "Users can't login after password reset",
			Priority:    domain.PriorityHigh,
			Status:      domain.StatusCompleted,
			Project:     "frontend",
			Tags:        []string{"bug", "urgent"},
			CreatedAt:   time.Now(),
		},
	}

	// render sample tasks
	for i, task := range sampleTasks {
		if i > 0 {
			// add separator with safety check
			sepWidth := width - 4
			if sepWidth < 1 {
				sepWidth = 1
			}
			sep := strings.Repeat("─", sepWidth)
			b.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.currentTheme.Separator)).
				Render(sep))
			b.WriteString("\n")
		}

		b.WriteString(m.renderTaskPreview(styles, task, width))
	}

	return b.String()
}

func (m SetupModel) renderTaskPreview(styles *theme.Styles, task *domain.Task, width int) string {
	var b strings.Builder

	// title with status icon
	statusIcon := display.GetStatusIcon(task.Status)
	titleLine := fmt.Sprintf("%s %s", statusIcon, task.Title)
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.currentTheme.TextPrimary)).
		Bold(true)
	b.WriteString(titleStyle.Render(titleLine))
	b.WriteString("\n")

	// priority and status
	priorityIcon := display.GetPriorityIcon(task.Priority)
	priorityStyle := styles.GetPriorityTextStyle(task.Priority)
	statusStyle := styles.GetStatusStyle(task.Status)

	infoLine := fmt.Sprintf("  %s %s | %s",
		priorityIcon,
		priorityStyle.Render(string(task.Priority)),
		statusStyle.Render(string(task.Status)),
	)
	b.WriteString(infoLine)
	b.WriteString("\n")

	// project and tags if present
	if task.Project != "" || len(task.Tags) > 0 {
		detailLine := "  "
		if task.Project != "" {
			detailLine += lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.currentTheme.Info)).
				Render(fmt.Sprintf("📁 %s", task.Project))
		}
		if len(task.Tags) > 0 {
			if task.Project != "" {
				detailLine += " "
			}
			detailLine += lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.currentTheme.TextMuted)).
				Render(fmt.Sprintf("🏷  %s", strings.Join(task.Tags, ", ")))
		}
		b.WriteString(detailLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	return b.String()
}

func timePtr(t time.Time) *time.Time {
	return &t
}
