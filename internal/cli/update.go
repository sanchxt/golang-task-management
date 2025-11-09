package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var (
	updateTitle       string
	updateDescription string
	updatePriority    string
	updateStatus      string
	updateProject     string
	updateTags        []string
	updateDueDate     string
	updateClearDue    bool

	titleSet       bool
	descriptionSet bool
	prioritySet    bool
	statusSet      bool
	projectSet     bool
	tagsSet        bool
	dueDateSet     bool
)

var updateCmd = &cobra.Command{
	Use:   "update [task-id]",
	Short: "Update an existing task",
	Long: `Update an existing task with new values.
Only the fields you specify will be updated; all other fields remain unchanged.

Examples:
  taskflow update 1 --title "New title"
  taskflow update 2 --priority urgent --status in_progress
  taskflow update 3 --description "Updated description" --tags bug,critical
  taskflow update 4 --project frontend --due-date "2024-12-31"
  taskflow update 5 --clear-due-date`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&updateTitle, "title", "", "Update task title")
	updateCmd.Flags().StringVar(&updateDescription, "description", "", "Update task description")
	updateCmd.Flags().StringVar(&updatePriority, "priority", "", "Update priority (low, medium, high, urgent)")
	updateCmd.Flags().StringVar(&updateStatus, "status", "", "Update status (pending, in_progress, completed, cancelled)")
	updateCmd.Flags().StringVar(&updateProject, "project", "", "Update project (name or ID, empty to remove)")
	updateCmd.Flags().StringSliceVar(&updateTags, "tags", nil, "Update tags (comma-separated)")
	updateCmd.Flags().StringVar(&updateDueDate, "due-date", "", "Update due date (YYYY-MM-DD format)")
	updateCmd.Flags().BoolVar(&updateClearDue, "clear-due-date", false, "Clear the due date")

	updateCmd.Flags().Lookup("title").Changed = false
	updateCmd.Flags().Lookup("description").Changed = false
	updateCmd.Flags().Lookup("priority").Changed = false
	updateCmd.Flags().Lookup("status").Changed = false
	updateCmd.Flags().Lookup("project").Changed = false
	updateCmd.Flags().Lookup("tags").Changed = false
	updateCmd.Flags().Lookup("due-date").Changed = false
}

func runUpdate(cmd *cobra.Command, args []string) error {
	taskID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", args[0])
	}

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

	task, err := repo.GetByID(ctx, taskID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Task not found: %v", err)))
		return nil
	}

	titleSet = cmd.Flags().Changed("title")
	descriptionSet = cmd.Flags().Changed("description")
	prioritySet = cmd.Flags().Changed("priority")
	statusSet = cmd.Flags().Changed("status")
	projectSet = cmd.Flags().Changed("project")
	tagsSet = cmd.Flags().Changed("tags")
	dueDateSet = cmd.Flags().Changed("due-date")

	if !titleSet && !descriptionSet && !prioritySet && !statusSet && !projectSet && !tagsSet && !dueDateSet && !updateClearDue {
		fmt.Println(styles.Info.Render("No updates specified. Use --help to see available flags."))
		return nil
	}

	if titleSet {
		task.Title = updateTitle
	}
	if descriptionSet {
		task.Description = updateDescription
	}
	if prioritySet {
		task.Priority = domain.Priority(updatePriority)
	}
	if statusSet {
		task.Status = domain.Status(updateStatus)
	}
	if projectSet {
		if updateProject == "" {
			task.ProjectID = nil
		} else {
			projectRepo := sqlite.NewProjectRepository(db)
			projectID, err := lookupProjectID(ctx, projectRepo, updateProject)
			if err != nil {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
				return nil
			}
			task.ProjectID = projectID
		}
	}
	if tagsSet {
		task.Tags = updateTags
	}
	if dueDateSet {
		dueDate, err := parseDueDate(updateDueDate)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Invalid due date format: %v", err)))
			fmt.Println(styles.Info.Render("  Use YYYY-MM-DD format (e.g., 2024-12-31)"))
			return nil
		}
		task.DueDate = dueDate
	}
	if updateClearDue {
		task.DueDate = nil
	}

	if err := repo.Update(ctx, task); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to update task: %v", err)))
		return nil
	}

	displayTaskUpdated(task, styles)

	return nil
}

func displayTaskUpdated(task *domain.Task, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Task #%d updated successfully!", task.ID)))
	fmt.Println()

	fmt.Printf("  %s %s\n", styles.Info.Render("Title:"), task.Title)

	if task.Description != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Description:"), task.Description)
	}

	fmt.Printf("  %s %s\n", styles.Info.Render("Priority:"), task.Priority)
	fmt.Printf("  %s %s\n", styles.Info.Render("Status:"), task.Status)

	if task.ProjectName != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Project:"), task.ProjectName)
	}

	if len(task.Tags) > 0 {
		fmt.Printf("  %s %s\n", styles.Info.Render("Tags:"), strings.Join(task.Tags, ", "))
	}

	if task.DueDate != nil {
		fmt.Printf("  %s %s\n", styles.Info.Render("Due Date:"), task.DueDate.Format("2006-01-02"))
	}

	fmt.Printf("  %s %s\n", styles.Info.Render("Updated:"), task.UpdatedAt.Format("2006-01-02 15:04:05"))

	fmt.Println()
}
