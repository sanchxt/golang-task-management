package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

// bulk operation flags
var (
	// filter
	bulkStatus     string
	bulkPriority   string
	bulkProject    string
	bulkTags       []string
	bulkSearch     string
	bulkSearchMode string

	// update
	bulkSetStatus      string
	bulkSetPriority    string
	bulkSetProject     string
	bulkSetDescription string
	bulkSetDueDate     string
	bulkUnsetProject   bool
	bulkUnsetDueDate   bool

	// tag
	bulkAddTags    []string
	bulkRemoveTags []string

	// move
	bulkToProject string

	// safety
	bulkConfirm bool
	bulkDryRun  bool
)

var bulkCmd = &cobra.Command{
	Use:   "bulk",
	Short: "Perform bulk operations on tasks",
	Long: `Perform bulk operations on multiple tasks at once.

Available operations:
  - update: Update status, priority, or other fields for multiple tasks
  - move: Move tasks between projects
  - tag: Add or remove tags from multiple tasks
  - delete: Delete multiple tasks

Use filters to specify which tasks to operate on.`,
}

var bulkUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update multiple tasks at once",
	Long: `Update status, priority, description, or other fields for multiple tasks.

Examples:
  # Mark all pending tasks as completed
  taskflow bulk update --status pending --set-status completed --confirm

  # Set priority to high for all tasks in project "Backend"
  taskflow bulk update --project Backend --set-priority high --confirm

  # Update multiple fields for tasks with specific tag
  taskflow bulk update --tags urgent --set-status in_progress --set-priority high --confirm

  # Preview changes without applying (dry run)
  taskflow bulk update --status pending --set-status completed --dry-run`,
	RunE: runBulkUpdate,
}

var bulkMoveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move multiple tasks to a different project",
	Long: `Move multiple tasks from one project to another.

Examples:
  # Move all pending tasks from project 1 to project 2
  taskflow bulk move --project 1 --status pending --to-project 2 --confirm

  # Move all tasks with "backend" tag to "Backend API" project
  taskflow bulk move --tags backend --to-project "Backend API" --confirm

  # Unassign tasks from their projects (set to no project)
  taskflow bulk move --project 1 --to-project "" --confirm

  # Preview move without applying
  taskflow bulk move --project 1 --to-project 2 --dry-run`,
	RunE: runBulkMove,
}

var bulkTagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Add or remove tags from multiple tasks",
	Long: `Add or remove tags from multiple tasks at once.

Examples:
  # Add "reviewed" tag to all completed tasks
  taskflow bulk tag --status completed --add-tags reviewed --confirm

  # Remove "urgent" and "wip" tags from all cancelled tasks
  taskflow bulk tag --status cancelled --remove-tags urgent,wip --confirm

  # Add and remove tags in one operation
  taskflow bulk tag --project 1 --add-tags v2.0 --remove-tags v1.0 --confirm

  # Preview changes
  taskflow bulk tag --status pending --add-tags backlog --dry-run`,
	RunE: runBulkTag,
}

var bulkDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete multiple tasks at once",
	Long: `Delete multiple tasks matching the specified filters.

WARNING: This operation is irreversible. Use --dry-run first to preview.

Examples:
  # Delete all cancelled tasks
  taskflow bulk delete --status cancelled --confirm

  # Delete all low priority completed tasks
  taskflow bulk delete --status completed --priority low --confirm

  # Delete all tasks in a project
  taskflow bulk delete --project "Old Project" --confirm

  # Preview deletion without applying
  taskflow bulk delete --status cancelled --dry-run`,
	RunE: runBulkDelete,
}

func init() {
	rootCmd.AddCommand(bulkCmd)
	bulkCmd.AddCommand(bulkUpdateCmd, bulkMoveCmd, bulkTagCmd, bulkDeleteCmd)

	for _, cmd := range []*cobra.Command{bulkUpdateCmd, bulkMoveCmd, bulkTagCmd, bulkDeleteCmd} {
		cmd.Flags().StringVar(&bulkStatus, "status", "", "Filter by status (pending, in_progress, completed, cancelled)")
		cmd.Flags().StringVar(&bulkPriority, "priority", "", "Filter by priority (low, medium, high, urgent)")
		cmd.Flags().StringVar(&bulkProject, "project", "", "Filter by project name or ID")
		cmd.Flags().StringSliceVar(&bulkTags, "tags", []string{}, "Filter by tags (comma-separated)")
		cmd.Flags().StringVar(&bulkSearch, "search", "", "Search query in title/description")
		cmd.Flags().StringVar(&bulkSearchMode, "search-mode", "text", "Search mode (text or regex)")
		cmd.Flags().BoolVar(&bulkDryRun, "dry-run", false, "Preview changes without applying")
	}

	// update
	bulkUpdateCmd.Flags().StringVar(&bulkSetStatus, "set-status", "", "New status to set")
	bulkUpdateCmd.Flags().StringVar(&bulkSetPriority, "set-priority", "", "New priority to set")
	bulkUpdateCmd.Flags().StringVar(&bulkSetProject, "set-project", "", "New project to set (name or ID)")
	bulkUpdateCmd.Flags().StringVar(&bulkSetDescription, "set-description", "", "New description to set")
	bulkUpdateCmd.Flags().StringVar(&bulkSetDueDate, "set-due-date", "", "New due date (YYYY-MM-DD)")
	bulkUpdateCmd.Flags().BoolVar(&bulkUnsetProject, "unset-project", false, "Remove project assignment")
	bulkUpdateCmd.Flags().BoolVar(&bulkUnsetDueDate, "unset-due-date", false, "Remove due date")
	bulkUpdateCmd.Flags().BoolVar(&bulkConfirm, "confirm", false, "Confirm the operation")

	// move
	bulkMoveCmd.Flags().StringVar(&bulkToProject, "to-project", "", "Target project (name, ID, or empty to unassign)")
	bulkMoveCmd.Flags().BoolVar(&bulkConfirm, "confirm", false, "Confirm the operation")
	bulkMoveCmd.MarkFlagRequired("to-project")

	// tag
	bulkTagCmd.Flags().StringSliceVar(&bulkAddTags, "add-tags", []string{}, "Tags to add (comma-separated)")
	bulkTagCmd.Flags().StringSliceVar(&bulkRemoveTags, "remove-tags", []string{}, "Tags to remove (comma-separated)")
	bulkTagCmd.Flags().BoolVar(&bulkConfirm, "confirm", false, "Confirm the operation")

	// delete
	bulkDeleteCmd.Flags().BoolVar(&bulkConfirm, "confirm", false, "Confirm the operation")
}

func runBulkUpdate(cmd *cobra.Command, args []string) error {
	// load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// load theme
	themeObj, err := theme.GetTheme(cfg.ThemeName)
	if err != nil {
		themeObj = theme.GetDefaultTheme()
	}
	styles := theme.NewStyles(themeObj)

	// initialize db
	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	taskRepo := sqlite.NewTaskRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	// build filter
	filter, err := buildTaskFilter(ctx, projectRepo)
	if err != nil {
		return err
	}

	if bulkSetStatus == "" && bulkSetPriority == "" && bulkSetProject == "" &&
		bulkSetDescription == "" && bulkSetDueDate == "" && !bulkUnsetProject && !bulkUnsetDueDate {
		return fmt.Errorf("no update fields specified. Use --set-status, --set-priority, etc.")
	}

	tasks, err := taskRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println(styles.Info.Render("No tasks match the specified filters."))
		return nil
	}

	// show preview
	fmt.Println(styles.Title.Render(fmt.Sprintf("Bulk Update Preview - %d tasks will be updated:", len(tasks))))
	fmt.Println()
	for i, task := range tasks {
		if i >= 10 {
			fmt.Println(styles.Subtitle.Render(fmt.Sprintf("... and %d more tasks", len(tasks)-10)))
			break
		}
		fmt.Printf("  • %s\n", task.Title)
	}
	fmt.Println()

	// show what will be changed
	fmt.Println(styles.Subtitle.Render("Changes to apply:"))
	if bulkSetStatus != "" {
		fmt.Printf("  • Status → %s\n", styles.Success.Render(bulkSetStatus))
	}
	if bulkSetPriority != "" {
		fmt.Printf("  • Priority → %s\n", styles.Success.Render(bulkSetPriority))
	}
	if bulkSetProject != "" {
		fmt.Printf("  • Project → %s\n", styles.Success.Render(bulkSetProject))
	}
	if bulkUnsetProject {
		fmt.Printf("  • Project → %s\n", styles.Info.Render("(unassigned)"))
	}
	if bulkSetDescription != "" {
		fmt.Printf("  • Description → %s\n", styles.Success.Render(bulkSetDescription))
	}
	if bulkSetDueDate != "" {
		fmt.Printf("  • Due Date → %s\n", styles.Success.Render(bulkSetDueDate))
	}
	if bulkUnsetDueDate {
		fmt.Printf("  • Due Date → %s\n", styles.Info.Render("(removed)"))
	}
	fmt.Println()

	if bulkDryRun {
		fmt.Println(styles.Info.Render("Dry run mode - no changes will be applied"))
		return nil
	}

	if !bulkConfirm {
		fmt.Println(styles.Error.Render("Operation not confirmed. Use --confirm to apply changes"))
		return nil
	}

	updates := repository.TaskUpdate{}

	if bulkSetStatus != "" {
		status := domain.Status(bulkSetStatus)
		validStatuses := map[domain.Status]bool{
			domain.StatusPending:    true,
			domain.StatusInProgress: true,
			domain.StatusCompleted:  true,
			domain.StatusCancelled:  true,
		}
		if !validStatuses[status] {
			return fmt.Errorf("invalid status: %s (must be pending, in_progress, completed, or cancelled)", bulkSetStatus)
		}
		updates.Status = &status
	}

	if bulkSetPriority != "" {
		priority := domain.Priority(bulkSetPriority)
		validPriorities := map[domain.Priority]bool{
			domain.PriorityLow:    true,
			domain.PriorityMedium: true,
			domain.PriorityHigh:   true,
			domain.PriorityUrgent: true,
		}
		if !validPriorities[priority] {
			return fmt.Errorf("invalid priority: %s (must be low, medium, high, or urgent)", bulkSetPriority)
		}
		updates.Priority = &priority
	}

	if bulkSetProject != "" {
		projectID, err := resolveProjectID(ctx, projectRepo, bulkSetProject)
		if err != nil {
			return err
		}
		updates.ProjectID = &projectID
	}

	if bulkUnsetProject {
		var nilPtr *int64
		updates.ProjectID = &nilPtr
	}

	if bulkSetDescription != "" {
		updates.Description = &bulkSetDescription
	}

	if bulkSetDueDate != "" {
		dateStr := bulkSetDueDate
		datePtr := &dateStr
		updates.DueDate = &datePtr
	}

	if bulkUnsetDueDate {
		var nilPtr *string
		updates.DueDate = &nilPtr
	}

	count, err := taskRepo.BulkUpdate(ctx, filter, updates)
	if err != nil {
		return fmt.Errorf("failed to update tasks: %w", err)
	}

	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Successfully updated %d tasks", count)))
	return nil
}

func runBulkMove(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	themeObj, err := theme.GetTheme(cfg.ThemeName)
	if err != nil {
		themeObj = theme.GetDefaultTheme()
	}
	styles := theme.NewStyles(themeObj)

	// initialize db
	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	taskRepo := sqlite.NewTaskRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	filter, err := buildTaskFilter(ctx, projectRepo)
	if err != nil {
		return err
	}

	tasks, err := taskRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println(styles.Info.Render("No tasks match the specified filters."))
		return nil
	}

	var targetProjectID *int64
	var targetProjectName string

	if bulkToProject == "" {
		targetProjectName = "(unassigned)"
	} else {
		id, err := resolveProjectID(ctx, projectRepo, bulkToProject)
		if err != nil {
			return err
		}
		targetProjectID = id

		if targetProjectID != nil {
			project, err := projectRepo.GetByID(ctx, *targetProjectID)
			if err != nil {
				return fmt.Errorf("failed to get project: %w", err)
			}
			targetProjectName = project.Name
		} else {
			targetProjectName = "(unassigned)"
		}
	}

	fmt.Println(styles.Title.Render(fmt.Sprintf("Bulk Move Preview - %d tasks will be moved:", len(tasks))))
	fmt.Println()
	for i, task := range tasks {
		if i >= 10 {
			fmt.Println(styles.Subtitle.Render(fmt.Sprintf("... and %d more tasks", len(tasks)-10)))
			break
		}
		fmt.Printf("  • %s\n", task.Title)
	}
	fmt.Println()
	fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Target project: %s", targetProjectName)))
	fmt.Println()

	if bulkDryRun {
		fmt.Println(styles.Info.Render("Dry run mode - no changes will be applied"))
		return nil
	}

	if !bulkConfirm {
		fmt.Println(styles.Error.Render("Operation not confirmed. Use --confirm to apply changes"))
		return nil
	}

	count, err := taskRepo.BulkMove(ctx, filter, targetProjectID)
	if err != nil {
		return fmt.Errorf("failed to move tasks: %w", err)
	}

	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Successfully moved %d tasks to %s", count, targetProjectName)))
	return nil
}

func runBulkTag(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	themeObj, err := theme.GetTheme(cfg.ThemeName)
	if err != nil {
		themeObj = theme.GetDefaultTheme()
	}
	styles := theme.NewStyles(themeObj)

	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	taskRepo := sqlite.NewTaskRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	if len(bulkAddTags) == 0 && len(bulkRemoveTags) == 0 {
		return fmt.Errorf("no tag operation specified. Use --add-tags or --remove-tags")
	}

	filter, err := buildTaskFilter(ctx, projectRepo)
	if err != nil {
		return err
	}

	tasks, err := taskRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println(styles.Info.Render("No tasks match the specified filters."))
		return nil
	}

	fmt.Println(styles.Title.Render(fmt.Sprintf("Bulk Tag Preview - %d tasks will be updated:", len(tasks))))
	fmt.Println()
	for i, task := range tasks {
		if i >= 10 {
			fmt.Println(styles.Subtitle.Render(fmt.Sprintf("... and %d more tasks", len(tasks)-10)))
			break
		}
		fmt.Printf("  • %s\n", task.Title)
	}
	fmt.Println()

	if len(bulkAddTags) > 0 {
		fmt.Printf("  %s %s\n", styles.Success.Render("+"), strings.Join(bulkAddTags, ", "))
	}
	if len(bulkRemoveTags) > 0 {
		fmt.Printf("  %s %s\n", styles.Error.Render("-"), strings.Join(bulkRemoveTags, ", "))
	}
	fmt.Println()

	if bulkDryRun {
		fmt.Println(styles.Info.Render("Dry run mode - no changes will be applied"))
		return nil
	}

	if !bulkConfirm {
		fmt.Println(styles.Error.Render("Operation not confirmed. Use --confirm to apply changes"))
		return nil
	}

	var totalCount int64

	if len(bulkAddTags) > 0 {
		count, err := taskRepo.BulkAddTags(ctx, filter, bulkAddTags)
		if err != nil {
			return fmt.Errorf("failed to add tags: %w", err)
		}
		totalCount = count
	}

	if len(bulkRemoveTags) > 0 {
		count, err := taskRepo.BulkRemoveTags(ctx, filter, bulkRemoveTags)
		if err != nil {
			return fmt.Errorf("failed to remove tags: %w", err)
		}
		if totalCount == 0 {
			totalCount = count
		}
	}

	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Successfully updated tags for %d tasks", totalCount)))
	return nil
}

func runBulkDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	themeObj, err := theme.GetTheme(cfg.ThemeName)
	if err != nil {
		themeObj = theme.GetDefaultTheme()
	}
	styles := theme.NewStyles(themeObj)

	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	taskRepo := sqlite.NewTaskRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	filter, err := buildTaskFilter(ctx, projectRepo)
	if err != nil {
		return err
	}

	tasks, err := taskRepo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to query tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println(styles.Info.Render("No tasks match the specified filters."))
		return nil
	}

	fmt.Println(styles.Error.Render(fmt.Sprintf("Bulk Delete Preview - %d tasks will be PERMANENTLY DELETED:", len(tasks))))
	fmt.Println()
	for i, task := range tasks {
		if i >= 10 {
			fmt.Println(styles.Subtitle.Render(fmt.Sprintf("... and %d more tasks", len(tasks)-10)))
			break
		}
		fmt.Printf("  • %s\n", task.Title)
	}
	fmt.Println()
	fmt.Println(styles.Error.Render("WARNING: This operation is irreversible!"))
	fmt.Println()

	if bulkDryRun {
		fmt.Println(styles.Info.Render("Dry run mode - no changes will be applied"))
		return nil
	}

	if !bulkConfirm {
		fmt.Println(styles.Error.Render("Operation not confirmed. Use --confirm to apply changes"))
		return nil
	}

	count, err := taskRepo.BulkDelete(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete tasks: %w", err)
	}

	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Successfully deleted %d tasks", count)))
	return nil
}

// build a TaskFilter from the global filter flags
func buildTaskFilter(ctx context.Context, projectRepo *sqlite.ProjectRepository) (repository.TaskFilter, error) {
	filter := repository.TaskFilter{
		SearchQuery: bulkSearch,
		SearchMode:  bulkSearchMode,
	}

	if bulkStatus != "" {
		filter.Status = domain.Status(bulkStatus)
	}

	if bulkPriority != "" {
		filter.Priority = domain.Priority(bulkPriority)
	}

	if bulkProject != "" {
		projectID, err := resolveProjectID(ctx, projectRepo, bulkProject)
		if err != nil {
			return filter, err
		}
		filter.ProjectID = projectID
	}

	if len(bulkTags) > 0 {
		filter.Tags = bulkTags
	}

	return filter, nil
}

// resolves project name or ID to an actual project ID
func resolveProjectID(ctx context.Context, projectRepo *sqlite.ProjectRepository, projectStr string) (*int64, error) {
	if projectStr == "" {
		return nil, nil
	}

	if id, err := strconv.ParseInt(projectStr, 10, 64); err == nil {
		_, err := projectRepo.GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("project with ID %d not found", id)
		}
		return &id, nil
	}

	project, err := projectRepo.GetByName(ctx, projectStr)
	if err != nil {
		return nil, fmt.Errorf("project '%s' not found", projectStr)
	}

	return &project.ID, nil
}
