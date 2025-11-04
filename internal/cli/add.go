package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
)

var (
	// flags
	addPriority    string
	addDescription string
	addProject     string
	addTags        []string
	addDueDate     string

	// styles
	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))
)

var addCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Long: `Add a new task to your task list.

Examples:
  taskflow add "Implement user authentication"
  taskflow add "Fix login bug" --priority high --project backend
  taskflow add "Write documentation" --tags docs,important --due-date "2024-12-31"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	// flags
	addCmd.Flags().StringVarP(&addPriority, "priority", "p", "medium", "Task priority (low, medium, high, urgent)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "Description of your task")
	addCmd.Flags().StringVarP(&addProject, "project", "P", "", "Project name")
	addCmd.Flags().StringSliceVarP(&addTags, "tags", "t", []string{}, "Comma-separated tags")
	addCmd.Flags().StringVar(&addDueDate, "due-date", "", "Due date (YYYY-MM-DD format)")
}

func runAdd(cmd *cobra.Command, args []string) error {
	// get title
	title := strings.Join(args, " ")

	// get config
	cfg, err := config.GetDefaultConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// initialize db
	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	repo := sqlite.NewTaskRepository(db)

	task := domain.NewTask(title)
	task.Description = addDescription
	task.Priority = domain.Priority(addPriority)
	task.Project = addProject
	task.Tags = addTags

	// parse due date
	if addDueDate != "" {
		dueDate, err := parseDueDate(addDueDate)
		if err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("✗ Invalid due date format: %v", err)))
			fmt.Println(infoStyle.Render("  Use YYYY-MM-DD format (e.g., 2024-12-31)"))
			return nil
		}
		task.DueDate = dueDate
	}

	// save to db
	ctx := context.Background()
	if err := repo.Create(ctx, task); err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("✗ Failed to create task: %v", err)))
		return nil
	}

	displayTaskCreated(task)

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

func displayTaskCreated(task *domain.Task) {
	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Task #%d created successfully!", task.ID)))
	fmt.Println()

	// task details
	fmt.Printf("  %s %s\n", infoStyle.Render("Title:"), task.Title)

	if task.Description != "" {
		fmt.Printf("  %s %s\n", infoStyle.Render("Description:"), task.Description)
	}

	fmt.Printf("  %s %s\n", infoStyle.Render("Priority:"), task.Priority)
	fmt.Printf("  %s %s\n", infoStyle.Render("Status:"), task.Status)

	if task.Project != "" {
		fmt.Printf("  %s %s\n", infoStyle.Render("Project:"), task.Project)
	}

	if len(task.Tags) > 0 {
		fmt.Printf("  %s %s\n", infoStyle.Render("Tags:"), strings.Join(task.Tags, ", "))
	}

	if task.DueDate != nil {
		fmt.Printf("  %s %s\n", infoStyle.Render("Due Date:"), task.DueDate.Format("2006-01-02"))
	}

	fmt.Println()
}
