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
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
)

var (
	// list command flags
	listStatus   string
	listPriority string
	listProject  string
	listTags     []string

	// table styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(1).
			PaddingRight(1)

	cellStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	urgentRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	highRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8800"))

	mediumRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0088FF"))

	lowRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long: `List all tasks with optional filtering.

Examples:
  taskflow list
  taskflow list --status pending
  taskflow list --priority high --project backend
  taskflow list --tags bug,urgent`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	// flags
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	listCmd.Flags().StringVarP(&listPriority, "priority", "p", "", "Filter by priority (low, medium, high, urgent)")
	listCmd.Flags().StringVarP(&listProject, "project", "P", "", "Filter by project")
	listCmd.Flags().StringSliceVarP(&listTags, "tags", "t", []string{}, "Filter by tags (comma-separated)")
}

func runList(cmd *cobra.Command, args []string) error {
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
		fmt.Println(errorStyle.Render(fmt.Sprintf("✗ Failed to list tasks: %v", err)))
		return nil
	}

	// filter by tags manually (todo: make repo support it)
	if len(listTags) > 0 {
		tasks = filterTasksByTags(tasks, listTags)
	}

	if len(tasks) == 0 {
		fmt.Println()
		fmt.Println(infoStyle.Render("No tasks found."))
		fmt.Println()
		return nil
	}

	displayTasksTable(tasks)

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

func displayTasksTable(tasks []*domain.Task) {
	fmt.Println()

	headers := []string{
		headerStyle.Render("Status"),
		headerStyle.Render("Priority"),
		headerStyle.Render("Title"),
		headerStyle.Render("Project"),
		headerStyle.Render("Tags"),
		headerStyle.Render("Due Date"),
	}
	fmt.Println(strings.Join(headers, " "))

	separator := strings.Repeat("─", 120)
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render(separator))

	for _, task := range tasks {
		printTaskRow(task)
	}

	fmt.Println()
	fmt.Printf("Total: %d task(s)\n", len(tasks))
	fmt.Println()
}

func printTaskRow(task *domain.Task) {
	rowStyle := getRowStyle(task.Priority)

	// status
	statusIcon := getStatusIcon(task.Status)
	status := fmt.Sprintf("%s %s", statusIcon, task.Status)

	// priority
	priorityIcon := getPriorityIcon(task.Priority)
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
		dueDate = formatDueDate(task.DueDate)
	}

	// format cells
	cells := []string{
		rowStyle.Render(cellStyle.Render(fmt.Sprintf("%-15s", status))),
		rowStyle.Render(cellStyle.Render(fmt.Sprintf("%-12s", priority))),
		rowStyle.Render(cellStyle.Render(fmt.Sprintf("%-40s", title))),
		rowStyle.Render(cellStyle.Render(fmt.Sprintf("%-15s", project))),
		rowStyle.Render(cellStyle.Render(fmt.Sprintf("%-20s", tags))),
		rowStyle.Render(cellStyle.Render(fmt.Sprintf("%-12s", dueDate))),
	}

	fmt.Println(strings.Join(cells, " "))
}

func getRowStyle(priority domain.Priority) lipgloss.Style {
	switch priority {
	case domain.PriorityUrgent:
		return urgentRowStyle
	case domain.PriorityHigh:
		return highRowStyle
	case domain.PriorityMedium:
		return mediumRowStyle
	case domain.PriorityLow:
		return lowRowStyle
	default:
		return cellStyle
	}
}

func getStatusIcon(status domain.Status) string {
	switch status {
	case domain.StatusCompleted:
		return "✓"
	case domain.StatusInProgress:
		return "⚡"
	case domain.StatusPending:
		return "○"
	case domain.StatusCancelled:
		return "✗"
	default:
		return "?"
	}
}

func getPriorityIcon(priority domain.Priority) string {
	switch priority {
	case domain.PriorityUrgent:
		return "🔥"
	case domain.PriorityHigh:
		return "⬆"
	case domain.PriorityMedium:
		return "➡"
	case domain.PriorityLow:
		return "⬇"
	default:
		return "?"
	}
}

func formatDueDate(dueDate *time.Time) string {
	if dueDate == nil {
		return "-"
	}

	now := time.Now()
	diff := dueDate.Sub(now)

	// if overdue
	if diff < 0 {
		days := int(-diff.Hours() / 24)
		if days == 0 {
			return "TODAY!"
		}
		return fmt.Sprintf("-%dd", days)
	}

	// if due soon
	days := int(diff.Hours() / 24)
	if days == 0 {
		return "Today"
	} else if days == 1 {
		return "Tomorrow"
	} else if days <= 7 {
		return fmt.Sprintf("%dd", days)
	}

	// return formatted date
	return dueDate.Format("2006-01-02")
}
