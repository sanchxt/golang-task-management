package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/tui"
)

var (
	// tui command flags
	tuiStatus   string
	tuiPriority string
	tuiProject  string
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI",
	Long: `Launch the interactive Text User Interface for managing tasks.

The TUI provides:
  - Interactive table view with navigation
  - Detailed task view with arrow key navigation
  - Keyboard shortcuts for quick operations

Keyboard shortcuts:
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
  taskflow tui
  taskflow tui --status pending
  taskflow tui --priority high --project backend`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)

	// flags for filtering
	tuiCmd.Flags().StringVarP(&tuiStatus, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	tuiCmd.Flags().StringVarP(&tuiPriority, "priority", "p", "", "Filter by priority (low, medium, high, urgent)")
	tuiCmd.Flags().StringVarP(&tuiProject, "project", "P", "", "Filter by project")
}

func runTUI(cmd *cobra.Command, args []string) error {
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
		Status:   domain.Status(tuiStatus),
		Priority: domain.Priority(tuiPriority),
		Project:  tuiProject,
	}

	// fetch tasks
	ctx := context.Background()
	tasks, err := repo.List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// create, run tui
	model := tui.NewModel(repo, tasks)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
