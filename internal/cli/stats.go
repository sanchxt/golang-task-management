package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show global system statistics",
	Long: `Display comprehensive statistics across all projects and tasks.

Provides an overview of:
  - Project counts and status distribution
  - Task counts by status and priority
  - Completion rates and progress metrics
  - Time-based analytics (overdue, due soon, recent)
  - Top projects by task count

Examples:
  taskflow stats                  # Show all global statistics
  taskflow stats --top 10         # Show top 10 projects`,
	RunE: runStats,
}

var (
	statsTopLimit int
)

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsTopLimit, "top", 5, "Number of top projects to show")
}

func runStats(cmd *cobra.Command, args []string) error {
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

	statsRepo := sqlite.NewStatisticsRepository(db)
	ctx := context.Background()

	stats, err := statsRepo.GetGlobalStatistics(ctx)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to get statistics: %v", err)))
		return nil
	}

	if statsTopLimit > 0 {
		topProjects, err := statsRepo.GetTopProjectsByTaskCount(ctx, statsTopLimit)
		if err == nil {
			stats.TopProjectsByTaskCount = topProjects
		}
	}

	displayGlobalStatistics(stats, styles)

	return nil
}

var projectStatsCmd = &cobra.Command{
	Use:   "stats <id|name>",
	Short: "Show project statistics",
	Long: `Display comprehensive statistics for a specific project.

Provides detailed analytics including:
  - Task counts by status and priority
  - Completion rate and progress metrics
  - Time-based metrics (overdue, due soon, recent activity)
  - Project health score and status
  - Optional: Include statistics for child projects

Examples:
  taskflow project stats "Backend"          # Stats for Backend project
  taskflow project stats 1                  # Stats for project ID 1
  taskflow project stats "Web App" --descendants  # Include child projects`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectStats,
}

var (
	projectStatsDescendants bool
)

func init() {
	projectCmd.AddCommand(projectStatsCmd)
	projectStatsCmd.Flags().BoolVar(&projectStatsDescendants, "descendants", false, "Include statistics for child projects")
}

func runProjectStats(cmd *cobra.Command, args []string) error {
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

	projectRepo := sqlite.NewProjectRepository(db)
	statsRepo := sqlite.NewStatisticsRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, projectRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	stats, err := statsRepo.GetProjectStatistics(ctx, *projectID, projectStatsDescendants)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to get statistics: %v", err)))
		return nil
	}

	project, err := projectRepo.GetByID(ctx, *projectID)
	if err == nil {
		stats.ProjectPath = project.BuildPath()
	}

	displayProjectStatistics(stats, styles)

	return nil
}


func displayGlobalStatistics(stats *domain.GlobalStats, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Title.Render("📊 Global Statistics"))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Projects"))
	fmt.Printf("  Total:      %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.TotalProjects)))
	fmt.Printf("  Active:     %s\n", styles.Success.Render(fmt.Sprintf("%d", stats.ActiveProjects)))
	fmt.Printf("  Archived:   %s\n", styles.Cell.Render(fmt.Sprintf("%d", stats.ArchivedProjects)))
	fmt.Printf("  Completed:  %s\n", styles.Success.Render(fmt.Sprintf("%d", stats.CompletedProjects)))
	fmt.Printf("  Favorites:  %s ★\n", styles.Info.Render(fmt.Sprintf("%d", stats.FavoriteProjects)))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Tasks"))
	fmt.Printf("  Total:        %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.TotalTasks)))
	fmt.Printf("  Pending:      %s\n", styles.Cell.Render(fmt.Sprintf("%d", stats.PendingTasks)))
	fmt.Printf("  In Progress:  %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.InProgressTasks)))
	fmt.Printf("  Completed:    %s\n", styles.Success.Render(fmt.Sprintf("%d", stats.CompletedTasks)))
	fmt.Printf("  Cancelled:    %s\n", styles.Cell.Render(fmt.Sprintf("%d", stats.CancelledTasks)))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Priority Distribution"))
	totalActive := stats.PendingTasks + stats.InProgressTasks
	if totalActive > 0 {
		lowPct := float64(stats.LowPriorityTasks) / float64(totalActive) * 100
		medPct := float64(stats.MediumPriorityTasks) / float64(totalActive) * 100
		highPct := float64(stats.HighPriorityTasks) / float64(totalActive) * 100
		urgentPct := float64(stats.UrgentPriorityTasks) / float64(totalActive) * 100

		fmt.Printf("  Low:    %s %s\n", renderBar(int(lowPct/5), 20, "░"), fmt.Sprintf("%d (%.1f%%)", stats.LowPriorityTasks, lowPct))
		fmt.Printf("  Medium: %s %s\n", renderBar(int(medPct/5), 20, "▒"), fmt.Sprintf("%d (%.1f%%)", stats.MediumPriorityTasks, medPct))
		fmt.Printf("  High:   %s %s\n", renderBar(int(highPct/5), 20, "▓"), fmt.Sprintf("%d (%.1f%%)", stats.HighPriorityTasks, highPct))
		fmt.Printf("  Urgent: %s %s\n", renderBar(int(urgentPct/5), 20, "█"), fmt.Sprintf("%d (%.1f%%)", stats.UrgentPriorityTasks, urgentPct))
	} else {
		fmt.Println("  No active tasks")
	}
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Time-Based Metrics"))
	if stats.OverdueTasks > 0 {
		fmt.Printf("  Overdue:      %s\n", styles.Error.Render(fmt.Sprintf("%d", stats.OverdueTasks)))
	} else {
		fmt.Printf("  Overdue:      %s\n", styles.Success.Render("0"))
	}
	fmt.Printf("  Due Soon:     %s (within 7 days)\n", styles.Info.Render(fmt.Sprintf("%d", stats.DueSoonTasks)))
	fmt.Printf("  Recent:       %s (last 7 days)\n", styles.Info.Render(fmt.Sprintf("%d", stats.RecentTasks)))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Completion Metrics"))
	fmt.Printf("  Completion Rate:  %s\n", renderCompletionRate(stats.OverallCompletionRate, styles))
	if stats.TotalProjects > 0 {
		avgTasks := stats.GetAverageTasksPerProject()
		fmt.Printf("  Avg Tasks/Project:  %s\n", styles.Info.Render(fmt.Sprintf("%.1f", avgTasks)))
	}
	fmt.Println()

	if len(stats.TopProjectsByTaskCount) > 0 {
		fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Top %d Projects by Task Count", len(stats.TopProjectsByTaskCount))))
		for i, proj := range stats.TopProjectsByTaskCount {
			icon := proj.Icon
			if icon == "" {
				icon = "📦"
			}
			bar := renderBar(proj.TaskCount, 20, "█")
			fmt.Printf("  %d. %s %s %s %s\n", i+1, icon, styles.Info.Render(proj.ProjectName), bar, styles.Cell.Render(fmt.Sprintf("(%d tasks)", proj.TaskCount)))
		}
		fmt.Println()
	}

	if stats.TotalViews > 0 || stats.TotalTemplates > 0 {
		fmt.Println(styles.Subtitle.Render("Additional Statistics"))
		if stats.TotalViews > 0 {
			fmt.Printf("  Saved Views:    %s", styles.Info.Render(fmt.Sprintf("%d", stats.TotalViews)))
			if stats.FavoriteViews > 0 {
				fmt.Printf(" (%d favorites ★)", stats.FavoriteViews)
			}
			fmt.Println()
		}
		if stats.TotalTemplates > 0 {
			fmt.Printf("  Templates:      %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.TotalTemplates)))
		}
		fmt.Println()
	}

	fmt.Printf("Calculated at: %s\n", stats.CalculatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
}

func displayProjectStatistics(stats *domain.ProjectStats, styles *theme.Styles) {
	fmt.Println()

	title := fmt.Sprintf("📊 Project Statistics: %s", stats.ProjectName)
	if stats.ProjectPath != "" && stats.ProjectPath != stats.ProjectName {
		title = fmt.Sprintf("📊 Project Statistics: %s", stats.ProjectPath)
	}
	fmt.Println(styles.Title.Render(title))
	fmt.Println()

	if stats.IncludeDescendants {
		fmt.Println(styles.Info.Render(fmt.Sprintf("Including %d descendant project(s)", stats.DescendantCount)))
		fmt.Println()
	}

	healthScore := stats.GetHealthScore()
	healthStatus := stats.GetHealthStatus()
	healthColor := getHealthColor(healthScore, styles)
	fmt.Println(styles.Subtitle.Render("Project Health"))
	fmt.Printf("  Score:  %s (%s)\n", healthColor.Render(fmt.Sprintf("%d/100", healthScore)), healthStatus)
	fmt.Printf("  Status: %s\n", renderHealthBar(healthScore))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Task Overview"))
	fmt.Printf("  Total:        %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.TotalTasks)))
	fmt.Printf("  Pending:      %s\n", styles.Cell.Render(fmt.Sprintf("%d", stats.PendingTasks)))
	fmt.Printf("  In Progress:  %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.InProgressTasks)))
	fmt.Printf("  Completed:    %s\n", styles.Success.Render(fmt.Sprintf("%d", stats.CompletedTasks)))
	fmt.Printf("  Cancelled:    %s\n", styles.Cell.Render(fmt.Sprintf("%d", stats.CancelledTasks)))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Priority Distribution"))
	if stats.TotalTasks > 0 {
		lowPct := float64(stats.LowPriorityTasks) / float64(stats.TotalTasks) * 100
		medPct := float64(stats.MediumPriorityTasks) / float64(stats.TotalTasks) * 100
		highPct := float64(stats.HighPriorityTasks) / float64(stats.TotalTasks) * 100
		urgentPct := float64(stats.UrgentPriorityTasks) / float64(stats.TotalTasks) * 100

		fmt.Printf("  Low:    %s %s\n", renderBar(int(lowPct/5), 20, "░"), fmt.Sprintf("%d (%.1f%%)", stats.LowPriorityTasks, lowPct))
		fmt.Printf("  Medium: %s %s\n", renderBar(int(medPct/5), 20, "▒"), fmt.Sprintf("%d (%.1f%%)", stats.MediumPriorityTasks, medPct))
		fmt.Printf("  High:   %s %s\n", renderBar(int(highPct/5), 20, "▓"), fmt.Sprintf("%d (%.1f%%)", stats.HighPriorityTasks, highPct))
		fmt.Printf("  Urgent: %s %s\n", renderBar(int(urgentPct/5), 20, "█"), fmt.Sprintf("%d (%.1f%%)", stats.UrgentPriorityTasks, urgentPct))
	} else {
		fmt.Println("  No tasks")
	}
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Time-Based Metrics"))
	if stats.OverdueTasks > 0 {
		fmt.Printf("  Overdue:          %s\n", styles.Error.Render(fmt.Sprintf("%d", stats.OverdueTasks)))
	} else {
		fmt.Printf("  Overdue:          %s\n", styles.Success.Render("0"))
	}
	fmt.Printf("  Due Soon:         %s (within 7 days)\n", styles.Info.Render(fmt.Sprintf("%d", stats.DueSoonTasks)))
	fmt.Printf("  Recent:           %s (created in last 7 days)\n", styles.Info.Render(fmt.Sprintf("%d", stats.RecentTasks)))
	fmt.Printf("  Recently Updated: %s (updated in last 7 days)\n", styles.Info.Render(fmt.Sprintf("%d", stats.RecentlyUpdated)))
	fmt.Println()

	fmt.Println(styles.Subtitle.Render("Completion Metrics"))
	fmt.Printf("  Completion Rate:  %s\n", renderCompletionRate(stats.CompletionRate, styles))
	fmt.Printf("  Active Tasks:     %s\n", styles.Info.Render(fmt.Sprintf("%d", stats.GetActiveTasks())))
	fmt.Println()

	fmt.Printf("Calculated at: %s\n", stats.CalculatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
}


func renderBar(value, maxWidth int, char string) string {
	if value > maxWidth {
		value = maxWidth
	}
	if value < 0 {
		value = 0
	}
	return strings.Repeat(char, value)
}

func renderHealthBar(score int) string {
	bar := ""
	filled := score / 5

	for i := 0; i < 20; i++ {
		if i < filled {
			if score >= 80 {
				bar += "█"
			} else if score >= 60 {
				bar += "▓"
			} else if score >= 40 {
				bar += "▒"
			} else {
				bar += "░"
			}
		} else {
			bar += "░"
		}
	}

	return bar
}

func renderCompletionRate(rate float64, styles *theme.Styles) string {
	bar := renderBar(int(rate/5), 20, "█")
	rateStr := fmt.Sprintf("%.1f%%", rate)

	if rate >= 80 {
		return fmt.Sprintf("%s %s", bar, styles.Success.Render(rateStr))
	} else if rate >= 50 {
		return fmt.Sprintf("%s %s", bar, styles.Info.Render(rateStr))
	} else if rate >= 25 {
		return fmt.Sprintf("%s %s", bar, styles.Cell.Render(rateStr))
	}
	return fmt.Sprintf("%s %s", bar, styles.Error.Render(rateStr))
}

func getHealthColor(score int, styles *theme.Styles) lipgloss.Style {
	if score >= 80 {
		return styles.Success
	} else if score >= 60 {
		return styles.Info
	} else if score >= 40 {
		return styles.Cell
	}
	return styles.Error
}
