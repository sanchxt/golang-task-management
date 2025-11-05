package cli

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/display"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
	"task-management/internal/tui"
)

var (
	// list command flags
	listStatus   string
	listPriority string
	listProject  string
	listTags     []string
	listCLI      bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks (interactive TUI by default)",
	Long: `List all tasks in an interactive TUI with optional filtering.
Use the --cli flag to display tasks as a text table instead.

The TUI provides:
  - Interactive table view with navigation
  - Detailed task view with arrow key navigation
  - Keyboard shortcuts for quick operations

Keyboard shortcuts (TUI mode):
  Table view:
    ↑/k     Move up
    ↓/j     Move down
    Enter   View task details

  Detail view:
    ↑/k     Previous task
    ↓/j     Next task
    Esc     Back to table

  Global:
    q       Quit
    ?       Toggle help

Examples:
  taskflow list                                    # Launch TUI
  taskflow list --status pending                   # TUI with filter
  taskflow list --priority high --project backend  # TUI with filters
  taskflow list --tags bug,urgent                  # TUI with tags
  taskflow list --cli                              # Text table mode
  taskflow list --cli --status pending             # Text table with filter`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	// flags
	listCmd.Flags().BoolVar(&listCLI, "cli", false, "Display as text table instead of TUI")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	listCmd.Flags().StringVarP(&listPriority, "priority", "p", "", "Filter by priority (low, medium, high, urgent)")
	listCmd.Flags().StringVarP(&listProject, "project", "P", "", "Filter by project")
	listCmd.Flags().StringSliceVarP(&listTags, "tags", "t", []string{}, "Filter by tags (comma-separated)")
}

func runList(cmd *cobra.Command, args []string) error {
	// get config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// load theme
	themeName := cfg.ThemeName
	if themeName == "" {
		themeName = "default"
	}
	themeObj, err := theme.GetTheme(themeName)
	if err != nil {
		return fmt.Errorf("failed to load theme: %w", err)
	}

	// create styles from theme
	styles := theme.NewStyles(themeObj)

	// initialize db
	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	repo := sqlite.NewTaskRepository(db)

	// build filter
	filter := repository.TaskFilter{
		Status:   domain.Status(listStatus),
		Priority: domain.Priority(listPriority),
		Project:  listProject,
		Tags:     listTags,
	}

	// fetch tasks
	ctx := context.Background()
	tasks, err := repo.List(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// filter by tags manually (todo: make repo support it)
	if len(listTags) > 0 {
		tasks = filterTasksByTags(tasks, listTags)
	}

	// handle empty tasks
	if len(tasks) == 0 && listCLI {
		fmt.Println()
		fmt.Println(styles.Info.Render("No tasks found."))
		fmt.Println()
		return nil
	}

	// mode to use
	if listCLI {
		displayTasksTable(tasks, styles)
	} else {
		model := tui.NewModel(repo, tasks, themeObj, styles)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
	}

	return nil
}

func filterTasksByTags(tasks []*domain.Task, filterTags []string) []*domain.Task {
	if len(filterTags) == 0 {
		return tasks
	}

	filtered := make([]*domain.Task, 0)
	for _, task := range tasks {
		if hasAllTags(task.Tags, filterTags) {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func hasAllTags(taskTags, filterTags []string) bool {
	for _, filterTag := range filterTags {
		found := false
		for _, taskTag := range taskTags {
			if taskTag == filterTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func displayTasksTable(tasks []*domain.Task, styles *theme.Styles) {
	fmt.Println()

	headers := []string{
		styles.Header.Render("Status"),
		styles.Header.Render("Priority"),
		styles.Header.Render("Title"),
		styles.Header.Render("Project"),
		styles.Header.Render("Tags"),
		styles.Header.Render("Due Date"),
	}
	fmt.Println(strings.Join(headers, " "))

	separator := strings.Repeat("─", 120)
	fmt.Println(styles.Separator.Render(separator))

	for _, task := range tasks {
		printTaskRow(task, styles)
	}

	fmt.Println()
	fmt.Printf("Total: %d task(s)\n", len(tasks))
	fmt.Println()
}

func printTaskRow(task *domain.Task, styles *theme.Styles) {
	rowStyle := styles.GetPriorityStyle(task.Priority)

	// status
	statusIcon := display.GetStatusIcon(task.Status)
	status := fmt.Sprintf("%s %s", statusIcon, task.Status)

	// priority
	priorityIcon := display.GetPriorityIcon(task.Priority)
	priority := fmt.Sprintf("%s %s", priorityIcon, task.Priority)

	// truncate title if too long
	title := task.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	// format project
	project := task.Project
	if project == "" {
		project = "-"
	}

	// format tags
	tags := strings.Join(task.Tags, ", ")
	if tags == "" {
		tags = "-"
	}
	if len(tags) > 20 {
		tags = tags[:17] + "..."
	}

	// format due date
	dueDate := "-"
	if task.DueDate != nil {
		dueDate = display.FormatDueDate(task.DueDate)
	}

	// format cells
	cells := []string{
		rowStyle.Render(styles.Cell.Render(fmt.Sprintf("%-15s", status))),
		rowStyle.Render(styles.Cell.Render(fmt.Sprintf("%-12s", priority))),
		rowStyle.Render(styles.Cell.Render(fmt.Sprintf("%-40s", title))),
		rowStyle.Render(styles.Cell.Render(fmt.Sprintf("%-15s", project))),
		rowStyle.Render(styles.Cell.Render(fmt.Sprintf("%-20s", tags))),
		rowStyle.Render(styles.Cell.Render(fmt.Sprintf("%-12s", dueDate))),
	}

	fmt.Println(strings.Join(cells, " "))
}
