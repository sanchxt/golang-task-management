package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var (
	// flags
	addPriority    string
	addDescription string
	addProject     string
	addTags        []string
	addDueDate     string
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Long: `Add a new task to your task list.

Examples:
  taskflow add "Implement user authentication"
  taskflow add "Fix login bug" --priority high --project Backend
  taskflow add "Write documentation" --tags docs,important --due-date "2024-12-31"
  taskflow add "Database optimization" --project 1 --priority high`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	// flags
	addCmd.Flags().StringVarP(&addPriority, "priority", "p", "medium", "Task priority (low, medium, high, urgent)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Description of your task")
	addCmd.Flags().StringVarP(&addProject, "project", "P", "", "Project name or ID")
	addCmd.Flags().StringSliceVarP(&addTags, "tags", "t", []string{}, "Comma-separated tags")
	addCmd.Flags().StringVar(&addDueDate, "due-date", "", "Due date (YYYY-MM-DD format)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// get title
	title := strings.Join(args, " ")

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

	styles := theme.NewStyles(themeObj)

	// initialize db
	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	repo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	task := domain.NewTask(title)
	task.Description = addDescription
	task.Priority = domain.Priority(addPriority)
	task.Tags = addTags

	// lookup project
	if addProject != "" {
		projectRepo := sqlite.NewProjectRepository(db)
		projectID, err := lookupProjectID(ctx, projectRepo, addProject)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
		task.ProjectID = projectID
	}

	// parse due date
	if addDueDate != "" {
		dueDate, err := parseDueDate(addDueDate)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Invalid due date format: %v", err)))
			fmt.Println(styles.Info.Render("  Use YYYY-MM-DD format (e.g., 2024-12-31)"))
			return nil
		}
		task.DueDate = dueDate
	}

	if err := repo.Create(ctx, task); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create task: %v", err)))
		return nil
	}

	displayTaskCreated(task, styles)

	return nil
}

func parseDueDate(dateStr string) (*time.Time, error) {
	// supported date formats
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unable to parse date: %s", dateStr)
}

func displayTaskCreated(task *domain.Task, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Task #%d created successfully!", task.ID)))
	fmt.Println()

	// task details
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

	fmt.Println()
}
