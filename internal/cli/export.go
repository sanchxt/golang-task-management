package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/export"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var (
	exportOutput          string
	exportFormat          string
	exportIncludeTasks    bool
	exportIncludeChildren bool
	exportProjectID       string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export projects, tasks, or create backups",
	Long: `Export data from TaskFlow in various formats.

Supported formats:
  - json: Structured JSON format (default)
  - csv: Comma-separated values for spreadsheets
  - markdown: Human-readable markdown format

Examples:
  # Export project to JSON
  taskflow export project 1 --output backend.json

  # Export project with tasks and children
  taskflow export project "Backend API" --include-tasks --include-children

  # Export tasks to CSV
  taskflow export tasks --format csv --output tasks.csv

  # Create full backup
  taskflow export backup --output backup.json`,
}

var exportProjectCmd = &cobra.Command{
	Use:   "project [id|name]",
	Short: "Export a project",
	Long: `Export a single project to a file.

By default, only the project metadata is exported. Use --include-tasks
and --include-children to export related data.

Examples:
  taskflow export project 1
  taskflow export project "Backend API" --include-tasks --include-children --output backend.json
  taskflow export project 1 --format markdown --output project.md`,
	Args: cobra.ExactArgs(1),
	RunE: runExportProject,
}

var exportTasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Export tasks",
	Long: `Export tasks matching filters to a file.

Use filter flags to specify which tasks to export.

Examples:
  taskflow export tasks --output all-tasks.json
  taskflow export tasks --project 1 --format csv --output project-tasks.csv
  taskflow export tasks --status pending --priority high --format markdown`,
	RunE: runExportTasks,
}

var exportBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a full system backup",
	Long: `Create a complete backup of all projects, tasks, templates, and views.

The backup file can be used with the 'taskflow import restore' command
to restore your data.

Examples:
  taskflow export backup --output backup.json
  taskflow export backup --output backup_$(date +%Y%m%d).json`,
	RunE: runExportBackup,
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportProjectCmd, exportTasksCmd, exportBackupCmd)

	// project export
	exportProjectCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
	exportProjectCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format (json, csv, markdown)")
	exportProjectCmd.Flags().BoolVar(&exportIncludeTasks, "include-tasks", false, "Include tasks in export")
	exportProjectCmd.Flags().BoolVar(&exportIncludeChildren, "include-children", false, "Include child projects")

	// tasks export
	exportTasksCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
	exportTasksCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "Export format (json, csv, markdown)")
	exportTasksCmd.Flags().StringVar(&exportProjectID, "project", "", "Filter by project (name or ID)")
	exportTasksCmd.Flags().StringVar(&bulkStatus, "status", "", "Filter by status")
	exportTasksCmd.Flags().StringVar(&bulkPriority, "priority", "", "Filter by priority")
	exportTasksCmd.Flags().StringSliceVar(&bulkTags, "tags", []string{}, "Filter by tags")

	// backup
	exportBackupCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (required)")
	exportBackupCmd.MarkFlagRequired("output")
}

func runExportProject(cmd *cobra.Command, args []string) error {
	projectIDOrName := args[0]

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

	var projectID int64
	if id, err := strconv.ParseInt(projectIDOrName, 10, 64); err == nil {
		projectID = id
	} else {
		project, err := projectRepo.GetByName(ctx, projectIDOrName)
		if err != nil {
			return fmt.Errorf("project not found: %s", projectIDOrName)
		}
		projectID = project.ID
	}

	project, err := projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	var output *os.File
	if exportOutput == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Exporting project: %s (ID: %d)", project.Name, project.ID)))

	switch exportFormat {
	case "json":
		exporter := export.NewJSONExporter(projectRepo, taskRepo)
		if err := exporter.ExportProjectToWriter(ctx, output, projectID, exportIncludeChildren, exportIncludeTasks); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

	case "markdown":
		exporter := export.NewMarkdownExporter(projectRepo, taskRepo)
		if err := exporter.ExportProjectToMarkdown(ctx, output, projectID, exportIncludeChildren, exportIncludeTasks); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

	case "csv":
		return fmt.Errorf("CSV format is not supported for single project export. Use JSON or Markdown instead.")

	default:
		return fmt.Errorf("unsupported format: %s (use json, csv, or markdown)", exportFormat)
	}

	if exportOutput != "" {
		fmt.Fprintln(os.Stderr, styles.Success.Render(fmt.Sprintf("✓ Project exported to %s", exportOutput)))
	}

	return nil
}

func runExportTasks(cmd *cobra.Command, args []string) error {
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

	filter, err := buildTaskFilter(ctx, projectRepo)
	if err != nil {
		return err
	}

	// override with exportProjectID if set
	if exportProjectID != "" {
		projectID, err := resolveProjectID(ctx, projectRepo, exportProjectID)
		if err != nil {
			return err
		}
		filter.ProjectID = projectID
	}

	count, err := taskRepo.Count(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to count tasks: %w", err)
	}

	if count == 0 {
		fmt.Fprintln(os.Stderr, styles.Info.Render("No tasks match the specified filters."))
		return nil
	}

	var output *os.File
	if exportOutput == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	}

	fmt.Fprintln(os.Stderr, styles.Info.Render(fmt.Sprintf("Exporting %d tasks...", count)))

	switch exportFormat {
	case "json":
		exporter := export.NewJSONExporter(projectRepo, taskRepo)
		if err := exporter.ExportTasksToWriter(ctx, output, filter); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

	case "csv":
		exporter := export.NewCSVExporter(projectRepo, taskRepo)
		if err := exporter.ExportTasksToCSV(ctx, output, filter); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

	case "markdown":
		exporter := export.NewMarkdownExporter(projectRepo, taskRepo)
		if err := exporter.ExportTasksToMarkdown(ctx, output, filter); err != nil {
			return fmt.Errorf("export failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported format: %s (use json, csv, or markdown)", exportFormat)
	}

	if exportOutput != "" {
		fmt.Fprintln(os.Stderr, styles.Success.Render(fmt.Sprintf("✓ Tasks exported to %s", exportOutput)))
	}

	return nil
}

func runExportBackup(cmd *cobra.Command, args []string) error {
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

	outputFile := exportOutput
	if outputFile == "" {
		outputFile = fmt.Sprintf("taskflow_backup_%s.json", time.Now().Format("20060102_150405"))
	}

	dir := filepath.Dir(outputFile)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer output.Close()

	fmt.Fprintln(os.Stderr, styles.Info.Render("Creating full system backup..."))

	exporter := export.NewJSONExporter(projectRepo, taskRepo)
	if err := exporter.CreateFullBackupToWriter(ctx, output); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Fprintln(os.Stderr, styles.Success.Render(fmt.Sprintf("✓ Backup created: %s", outputFile)))
	return nil
}
