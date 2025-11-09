package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/export"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var (
	importFile           string
	importParentProject  string
	importConflictMode   string
	importDryRun         bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import projects or restore backups",
	Long: `Import data into TaskFlow from exported files.

Conflict strategies:
  - merge: Keep existing projects, import new ones (default)
  - skip: Skip projects that already exist
  - overwrite: Replace existing projects with imported data

Examples:
  # Import a project
  taskflow import project backend.json

  # Import under a specific parent
  taskflow import project backend.json --parent "Systems"

  # Restore a full backup
  taskflow import restore backup.json --conflict-strategy merge`,
}

var importProjectCmd = &cobra.Command{
	Use:   "project [file]",
	Short: "Import a project from a file",
	Long: `Import a project and optionally its tasks and children from a JSON file.

The file should be in the format created by 'taskflow export project'.

Examples:
  taskflow import project backend.json
  taskflow import project backend.json --parent 1
  taskflow import project backend.json --parent "Systems" --conflict-strategy skip`,
	Args: cobra.ExactArgs(1),
	RunE: runImportProject,
}

var importRestoreCmd = &cobra.Command{
	Use:   "restore [file]",
	Short: "Restore a full system backup",
	Long: `Restore all projects, tasks, templates, and views from a backup file.

The file should be created by 'taskflow export backup'.

WARNING: This will import all data from the backup. Use --conflict-strategy
to control how conflicts are handled.

Examples:
  taskflow import restore backup.json
  taskflow import restore backup.json --conflict-strategy merge
  taskflow import restore backup.json --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runImportRestore,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importProjectCmd, importRestoreCmd)

	// project import
	importProjectCmd.Flags().StringVar(&importParentProject, "parent", "", "Parent project (name or ID)")
	importProjectCmd.Flags().StringVar(&importConflictMode, "conflict-strategy", "merge", "Conflict strategy (merge, skip, overwrite)")

	// restore
	importRestoreCmd.Flags().StringVar(&importConflictMode, "conflict-strategy", "merge", "Conflict strategy (merge, skip, overwrite)")
	importRestoreCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Preview import without making changes")
}

func runImportProject(cmd *cobra.Command, args []string) error {
	importFile = args[0]

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

	projectRepo := sqlite.NewProjectRepository(db)
	taskRepo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	// resolve parent project if specified
	var parentID *int64
	if importParentProject != "" {
		id, err := resolveProjectID(ctx, projectRepo, importParentProject)
		if err != nil {
			return fmt.Errorf("parent project not found: %w", err)
		}
		parentID = id
	}

	strategy := export.ConflictStrategy(importConflictMode)
	switch strategy {
	case export.ConflictStrategyMerge, export.ConflictStrategySkip, export.ConflictStrategyOverwrite:
	default:
		return fmt.Errorf("invalid conflict strategy: %s (use merge, skip, or overwrite)", importConflictMode)
	}

	file, err := os.Open(importFile)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer file.Close()

	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Importing project from %s...", importFile)))

	importer := export.NewImporter(projectRepo, taskRepo)
	project, err := importer.ImportProject(ctx, file, parentID, strategy)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	fmt.Fprintln(os.Stderr, styles.Success.Render(fmt.Sprintf("✓ Project imported: %s (ID: %d)", project.Name, project.ID)))

	if parentID != nil {
		parent, err := projectRepo.GetByID(ctx, *parentID)
		if err == nil {
			fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("  Parent: %s", parent.Name)))
		}
	}

	return nil
}

func runImportRestore(cmd *cobra.Command, args []string) error {
	importFile = args[0]

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	themeObj, err := theme.GetTheme(cfg.ThemeName)
	if err != nil {
		themeObj = theme.GetDefaultTheme()
	}
	styles := theme.NewStyles(themeObj)

	strategy := export.ConflictStrategy(importConflictMode)
	switch strategy {
	case export.ConflictStrategyMerge, export.ConflictStrategySkip, export.ConflictStrategyOverwrite:
		// valid
	default:
		return fmt.Errorf("invalid conflict strategy: %s (use merge, skip, or overwrite)", importConflictMode)
	}

	if importDryRun {
		fmt.Fprintln(os.Stderr, styles.Info.Render("DRY RUN MODE - No changes will be made"))
		fmt.Fprintln(os.Stderr)
		return fmt.Errorf("dry run mode not yet implemented - coming soon")
	}

	fmt.Fprintln(os.Stderr, styles.Error.Render("⚠️  WARNING: This will import all data from the backup file."))
	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Conflict strategy: %s", strategy)))
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Continue? (yes/no): ")

	var response string
	fmt.Scanln(&response)
	if response != "yes" && response != "y" {
		fmt.Fprintln(os.Stderr, styles.Info.Render("Restore cancelled."))
		return nil
	}

	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	projectRepo := sqlite.NewProjectRepository(db)
	taskRepo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	file, err := os.Open(importFile)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Restoring backup from %s...", importFile)))

	importer := export.NewImporter(projectRepo, taskRepo)
	if err := importer.RestoreBackup(ctx, file, strategy); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Fprintln(os.Stderr, styles.Success.Render("✓ Backup restored successfully"))

	projectCount, _ := projectRepo.Count(ctx, repository.ProjectFilter{})
	taskCount, _ := taskRepo.Count(ctx, repository.TaskFilter{})

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Total projects: %d", projectCount)))
	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Total tasks: %d", taskCount)))

	return nil
}
