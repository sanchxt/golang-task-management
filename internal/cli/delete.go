package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var (
	// delete flags
	deleteForce bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete [task-id...]",
	Short: "Delete one or more tasks",
	Long: `Delete one or more tasks by their IDs.
You will be prompted for confirmation unless you use the --force flag.

Examples:
  taskflow delete 1
  taskflow delete 1 2 3
  taskflow delete 5 --force`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// flags
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	// parse task IDs
	var taskIDs []int64
	for _, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", arg)
		}
		taskIDs = append(taskIDs, id)
	}

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

	// confirmation prompt
	if !deleteForce {
		var taskWord string
		if len(taskIDs) == 1 {
			taskWord = "task"
		} else {
			taskWord = "tasks"
		}

		fmt.Println()
		fmt.Println(styles.Error.Render(fmt.Sprintf("⚠  You are about to delete %d %s:", len(taskIDs), taskWord)))
		fmt.Println(styles.Info.Render(fmt.Sprintf("   IDs: %v", taskIDs)))
		fmt.Println()
		fmt.Print(styles.Subtitle.Render("   Are you sure? (y/N): "))

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println()
			fmt.Println(styles.Info.Render("Deletion cancelled."))
			fmt.Println()
			return nil
		}
	}

	// initialize db
	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	repo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	// delete tasks
	var deleted []int64
	var failed []string

	for _, id := range taskIDs {
		if err := repo.Delete(ctx, id); err != nil {
			failed = append(failed, fmt.Sprintf("#%d (%v)", id, err))
		} else {
			deleted = append(deleted, id)
		}
	}

	// display results
	fmt.Println()

	if len(deleted) > 0 {
		var taskWord string
		if len(deleted) == 1 {
			taskWord = "task"
		} else {
			taskWord = "tasks"
		}
		fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Successfully deleted %d %s", len(deleted), taskWord)))
		if len(deleted) <= 10 {
			fmt.Println(styles.Info.Render(fmt.Sprintf("  IDs: %v", deleted)))
		}
	}

	if len(failed) > 0 {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to delete %d task(s):", len(failed))))
		for _, f := range failed {
			fmt.Println(styles.Error.Render(fmt.Sprintf("  %s", f)))
		}
	}

	fmt.Println()

	return nil
}
