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

	// pagination flags
	listPage     int
	listPageSize int
	listAll      bool

	// search flags
	listSearch    string
	listRegex     bool
	listSortBy    string
	listSortOrder string
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

	// display flags
	listCmd.Flags().BoolVar(&listCLI, "cli", false, "Display as text table instead of TUI")

	// filter flags
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	listCmd.Flags().StringVarP(&listPriority, "priority", "p", "", "Filter by priority (low, medium, high, urgent)")
	listCmd.Flags().StringVarP(&listProject, "project", "P", "", "Filter by project")
	listCmd.Flags().StringSliceVarP(&listTags, "tags", "t", []string{}, "Filter by tags (comma-separated)")

	// pagination flags
	listCmd.Flags().IntVar(&listPage, "page", 1, "Page number (starts at 1)")
	listCmd.Flags().IntVar(&listPageSize, "page-size", 0, "Number of tasks per page (0 = use config default)")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all tasks (disable pagination)")

	// search flags
	listCmd.Flags().StringVar(&listSearch, "search", "", "Search query (searches in title, description, project, tags)")
	listCmd.Flags().BoolVar(&listRegex, "regex", false, "Use regex mode for search")
	listCmd.Flags().StringVar(&listSortBy, "sort-by", "created_at", "Sort by field (created_at, updated_at, priority, due_date, title)")
	listCmd.Flags().StringVar(&listSortOrder, "sort-order", "desc", "Sort order (asc, desc)")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	themeName := cfg.ThemeName
	if themeName == "" {
		themeName = "default"
	}
	themeObj, err := theme.GetTheme(themeName)
	if err != nil {
		return fmt.Errorf("failed to load theme: %w", err)
	}

	styles := theme.NewStyles(themeObj)

	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	repo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	// determine page size
	pageSize := listPageSize
	if pageSize == 0 {
		pageSize = cfg.DefaultPageSize
	}
	if pageSize > cfg.MaxPageSize {
		pageSize = cfg.MaxPageSize
	}

	// build filter
	filter := repository.TaskFilter{
		Status:      domain.Status(listStatus),
		Priority:    domain.Priority(listPriority),
		Project:     listProject,
		Tags:        listTags,
		SearchQuery: listSearch,
		SortBy:      listSortBy,
		SortOrder:   listSortOrder,
	}

	// search mode
	if listRegex {
		filter.SearchMode = "regex"
	} else if listSearch != "" {
		filter.SearchMode = "text"
	}

	if !listAll && listCLI {
		if listPage < 1 {
			listPage = 1
		}
		filter.Limit = pageSize
		filter.Offset = (listPage - 1) * pageSize
	}

	// total count for pagination info
	totalCount, err := repo.Count(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to count tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to count tasks: %w", err)
	}

	// fetch tasks
	tasks, err := repo.List(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// handle empty tasks
	if len(tasks) == 0 && listCLI {
		fmt.Println()
		if hasActiveFilters(filter) {
			fmt.Println(styles.Info.Render("No tasks found matching the filters."))
			displayActiveFilters(filter, styles)
		} else {
			fmt.Println(styles.Info.Render("No tasks found."))
		}
		fmt.Println()
		return nil
	}

	// mode to use
	if listCLI {
		totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
		if listAll {
			listPage = 1
			totalPages = 1
		}

		displayTasksTable(tasks, styles, filter, listPage, totalPages, totalCount)
	} else {
		model := tui.NewModel(repo, filter, pageSize, themeObj, styles)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
	}

	return nil
}

// checks if any filters are active
func hasActiveFilters(filter repository.TaskFilter) bool {
	return filter.Status != "" ||
		filter.Priority != "" ||
		filter.Project != "" ||
		len(filter.Tags) > 0 ||
		filter.SearchQuery != ""
}

// displays active filters summary
func displayActiveFilters(filter repository.TaskFilter, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Active filters:"))

	if filter.Status != "" {
		fmt.Printf("  Status: %s\n", filter.Status)
	}
	if filter.Priority != "" {
		fmt.Printf("  Priority: %s\n", filter.Priority)
	}
	if filter.Project != "" {
		fmt.Printf("  Project: %s\n", filter.Project)
	}
	if len(filter.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(filter.Tags, ", "))
	}
	if filter.SearchQuery != "" {
		mode := "text"
		if filter.SearchMode == "regex" {
			mode = "regex"
		}
		fmt.Printf("  Search (%s): %s\n", mode, filter.SearchQuery)
	}
	if filter.SortBy != "created_at" || filter.SortOrder != "desc" {
		fmt.Printf("  Sort: %s %s\n", filter.SortBy, filter.SortOrder)
	}
}

func displayTasksTable(tasks []*domain.Task, styles *theme.Styles, filter repository.TaskFilter, currentPage, totalPages int, totalCount int64) {
	fmt.Println()

	if hasActiveFilters(filter) {
		displayActiveFilters(filter, styles)
		fmt.Println()
	}

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

	// pagination info
	if filter.Limit > 0 {
		startIdx := filter.Offset + 1
		endIdx := filter.Offset + len(tasks)

		paginationInfo := fmt.Sprintf("Showing %d-%d of %d tasks (Page %d of %d)",
			startIdx, endIdx, totalCount, currentPage, totalPages)

		fmt.Println(styles.Subtitle.Render(paginationInfo))

		if currentPage < totalPages {
			nextPageHint := fmt.Sprintf("Use --page %d to see the next page", currentPage+1)
			fmt.Println(styles.Info.Render(nextPageHint))
		}
	} else {
		fmt.Printf("Total: %d task(s)\n", totalCount)
	}

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
