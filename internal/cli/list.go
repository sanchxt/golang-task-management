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
	"task-management/internal/query"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
	"task-management/internal/tui"
)

var (
	// list command
	listStatus   string
	listPriority string
	listProject  string
	listTags     []string
	listCLI      bool

	// pagination
	listPage     int
	listPageSize int
	listAll      bool

	// search
	listSearch         string
	listRegex          bool
	listFuzzy          bool
	listFuzzyThreshold int
	listSortBy         string
	listSortOrder      string

	// query language
	listQuery string
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
  taskflow list --cli --status pending             # Text table with filter

  # Query language examples (use 'taskflow query help' for full syntax reference):
  taskflow list --query "status:pending priority:high"      # Combine status + priority
  taskflow list --query "@backend tag:bug -tag:wontfix"     # Project + include/exclude tags
  taskflow list --query "due:+7d priority:urgent"           # Due in next 7 days, urgent
  taskflow list --query "due:-3d status:pending"            # Overdue by 3 days, pending
  taskflow list --query "@~front status:pending"            # Fuzzy project match + status
  taskflow list --query "-status:completed -status:cancelled" # Multiple exclusions

  # Fuzzy search examples:
  taskflow list --search back --fuzzy              # Fuzzy search for "back" (finds backend, backup, etc.)
  taskflow list --search api --fuzzy --fuzzy-threshold 70  # Higher threshold for stricter matching
  taskflow list --cli --search bcknd --fuzzy       # Typo-tolerant search in CLI mode`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	// display
	listCmd.Flags().BoolVar(&listCLI, "cli", false, "Display as text table instead of TUI")

	// filter
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	listCmd.Flags().StringVarP(&listPriority, "priority", "p", "", "Filter by priority (low, medium, high, urgent)")
	listCmd.Flags().StringVarP(&listProject, "project", "P", "", "Filter by project (name or ID)")
	listCmd.Flags().StringSliceVarP(&listTags, "tags", "t", []string{}, "Filter by tags (comma-separated)")

	// pagination
	listCmd.Flags().IntVar(&listPage, "page", 1, "Page number (starts at 1)")
	listCmd.Flags().IntVar(&listPageSize, "page-size", 0, "Number of tasks per page (0 = use config default)")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all tasks (disable pagination)")

	// search
	listCmd.Flags().StringVar(&listSearch, "search", "", "Search query (searches in title, description, project, tags)")
	listCmd.Flags().BoolVar(&listRegex, "regex", false, "Use regex mode for search")
	listCmd.Flags().BoolVar(&listFuzzy, "fuzzy", false, "Use fuzzy search mode (typo-tolerant, abbreviation-friendly)")
	listCmd.Flags().IntVar(&listFuzzyThreshold, "fuzzy-threshold", 60, "Minimum fuzzy match score (0-100, default 60)")
	listCmd.Flags().StringVar(&listSortBy, "sort-by", "created_at", "Sort by field (created_at, updated_at, priority, due_date, title)")
	listCmd.Flags().StringVar(&listSortOrder, "sort-order", "desc", "Sort order (asc, desc)")

	// query language
	listCmd.Flags().StringVarP(&listQuery, "query", "q", "", "Query language filter (e.g., 'status:pending @backend tag:bug')")
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
	projectRepo := sqlite.NewProjectRepository(db)
	viewRepo := sqlite.NewViewRepository(db)
	searchHistoryRepo := sqlite.NewSearchHistoryRepository(db)
	ctx := context.Background()

	pageSize := listPageSize
	if pageSize == 0 {
		pageSize = cfg.DefaultPageSize
	}
	if pageSize > cfg.MaxPageSize {
		pageSize = cfg.MaxPageSize
	}

	if listQuery != "" {
		return runListWithQueryLanguage(ctx, repo, projectRepo, viewRepo, searchHistoryRepo, cfg, themeObj, styles, pageSize)
	}

	var parsedQuery *query.ProjectMentionQuery
	if listSearch != "" {
		var err error
		parsedQuery, err = query.ParseProjectMentions(listSearch)
		if err != nil {
			if listCLI {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to parse query: %v", err)))
				return nil
			}
			return fmt.Errorf("failed to parse query: %w", err)
		}
	}

	var projectID *int64
	var projectSource string

	if parsedQuery != nil && parsedQuery.HasProjectFilter() {
		mention := parsedQuery.ProjectMentions[0]

		if mention.Fuzzy {
			fuzzyThreshold := 60
			var err error
			projectID, err = lookupProjectByFuzzyName(ctx, projectRepo, mention.Name, fuzzyThreshold)
			if err != nil {
				if listCLI {
					fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
					return nil
				}
				return fmt.Errorf("%v", err)
			}
			projectSource = "@~" + mention.Name
		} else {
			var err error
			projectID, err = lookupProjectID(ctx, projectRepo, mention.Name)
			if err != nil {
				if listCLI {
					fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
					return nil
				}
				return fmt.Errorf("%v", err)
			}
			projectSource = "@" + mention.Name
		}

		listSearch = parsedQuery.BaseQuery
	} else if listProject != "" {
		var err error
		projectID, err = lookupProjectID(ctx, projectRepo, listProject)
		if err != nil {
			if listCLI {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
				return nil
			}
			return fmt.Errorf("%v", err)
		}
		projectSource = "--project=" + listProject
	}

	_ = projectSource

	filter := repository.TaskFilter{
		Status:      domain.Status(listStatus),
		Priority:    domain.Priority(listPriority),
		ProjectID:   projectID,
		Tags:        listTags,
		SearchQuery: listSearch,
		SortBy:      listSortBy,
		SortOrder:   listSortOrder,
	}

	if listFuzzy && listSearch != "" {
		filter.SearchMode = "fuzzy"
		filter.FuzzyThreshold = listFuzzyThreshold
		if filter.FuzzyThreshold < 0 || filter.FuzzyThreshold > 100 {
			if listCLI {
				fmt.Println(styles.Error.Render("✗ Fuzzy threshold must be between 0 and 100"))
				return nil
			}
			return fmt.Errorf("fuzzy threshold must be between 0 and 100")
		}
	} else if listRegex {
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

	totalCount, err := repo.Count(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to count tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to count tasks: %w", err)
	}

	tasks, err := repo.List(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to list tasks: %w", err)
	}

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

	if listCLI {
		totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
		if listAll {
			listPage = 1
			totalPages = 1
		}

		displayTasksTable(tasks, styles, filter, listPage, totalPages, totalCount)
	} else {
		model := tui.NewModel(repo, projectRepo, viewRepo, searchHistoryRepo, filter, pageSize, themeObj, styles)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
	}

	return nil
}

// handles --query flag using the query language parser
func runListWithQueryLanguage(
	ctx context.Context,
	repo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	viewRepo repository.ViewRepository,
	searchHistoryRepo repository.SearchHistoryRepository,
	cfg *config.Config,
	themeObj *theme.Theme,
	styles *theme.Styles,
	pageSize int,
) error {
	parsed, err := query.ParseQuery(listQuery)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Query parse error: %v", err)))
			return nil
		}
		return fmt.Errorf("query parse error: %w", err)
	}

	converterCtx := &query.ConverterContext{
		ProjectRepo: projectRepo,
	}

	filter, err := query.ConvertToTaskFilter(ctx, parsed, converterCtx)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Query conversion error: %v", err)))
			return nil
		}
		return fmt.Errorf("query conversion error: %w", err)
	}

	filter.SortBy = listSortBy
	filter.SortOrder = listSortOrder

	if !listAll && listCLI {
		if listPage < 1 {
			listPage = 1
		}
		filter.Limit = pageSize
		filter.Offset = (listPage - 1) * pageSize
	}

	totalCount, err := repo.Count(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to count tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to count tasks: %w", err)
	}

	tasks, err := repo.List(ctx, filter)
	if err != nil {
		if listCLI {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list tasks: %v", err)))
			return nil
		}
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 && listCLI {
		fmt.Println()
		fmt.Println(styles.Info.Render("No tasks found matching the query."))
		fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Query: %s", listQuery)))
		fmt.Println()
		return nil
	}

	if listCLI {
		totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
		if listAll {
			listPage = 1
			totalPages = 1
		}

		fmt.Println()
		fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Query: %s", listQuery)))
		displayTasksTable(tasks, styles, filter, listPage, totalPages, totalCount)
	} else {
		model := tui.NewModel(repo, projectRepo, viewRepo, searchHistoryRepo, filter, pageSize, themeObj, styles)
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
	}

	return nil
}

func hasActiveFilters(filter repository.TaskFilter) bool {
	return filter.Status != "" ||
		filter.Priority != "" ||
		filter.ProjectID != nil ||
		len(filter.Tags) > 0 ||
		filter.SearchQuery != ""
}

func displayActiveFilters(filter repository.TaskFilter, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Active filters:"))

	if filter.Status != "" {
		fmt.Printf("  Status: %s\n", filter.Status)
	}
	if filter.Priority != "" {
		fmt.Printf("  Priority: %s\n", filter.Priority)
	}
	if filter.ProjectID != nil {
		fmt.Printf("  Project ID: %d\n", *filter.ProjectID)
	}
	if len(filter.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(filter.Tags, ", "))
	}
	if filter.SearchQuery != "" {
		mode := "text"
		if filter.SearchMode == "regex" {
			mode = "regex"
		} else if filter.SearchMode == "fuzzy" {
			mode = fmt.Sprintf("fuzzy, threshold=%d", filter.FuzzyThreshold)
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

	// truncate title
	title := task.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	// format project
	project := task.ProjectName
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
