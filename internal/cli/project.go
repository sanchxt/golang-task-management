package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects and hierarchies",
	Long: `Manage projects and project hierarchies.

Projects allow you to organize tasks into hierarchical structures with parent-child
relationships. Each project can have multiple children and belongs to one parent.

Features:
  - Hierarchical organization (unlimited depth)
  - Color coding for visual distinction
  - Icons for quick identification
  - Task statistics and completion tracking
  - Archive/favorite support`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectAddCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectViewCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectArchiveCmd)
	projectCmd.AddCommand(projectUnarchiveCmd)
	projectCmd.AddCommand(projectAliasCmd)
	projectCmd.AddCommand(projectAliasesCmd)
	projectCmd.AddCommand(projectUnaliasCmd)
	projectCmd.AddCommand(projectNoteCmd)
}


var (
	addProjectParent      string
	addProjectDescription string
	addProjectColor       string
	addProjectIcon        string
	addProjectFavorite    bool
	addProjectTemplate    string
	addProjectNotes       string
)

var projectAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Create a new project",
	Long: `Create a new project with optional parent hierarchy.

Examples:
  taskflow project add "Backend"
  taskflow project add "API Service" --parent "Backend"
  taskflow project add "Web App" --color blue --icon 🚀
  taskflow project add "Frontend" --description "UI tasks" --favorite`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProjectAdd,
}

func init() {
	projectAddCmd.Flags().StringVarP(&addProjectParent, "parent", "p", "", "Parent project name or ID")
	projectAddCmd.Flags().StringVarP(&addProjectDescription, "description", "d", "", "Project description")
	projectAddCmd.Flags().StringVarP(&addProjectColor, "color", "c", "", "Project color")
	projectAddCmd.Flags().StringVarP(&addProjectIcon, "icon", "i", "", "Project icon (emoji)")
	projectAddCmd.Flags().BoolVarP(&addProjectFavorite, "favorite", "f", false, "Mark as favorite")
	projectAddCmd.Flags().StringVarP(&addProjectTemplate, "template", "t", "", "Template name or ID to apply")
	projectAddCmd.Flags().StringVar(&addProjectNotes, "notes", "", "Project notes (markdown supported)")
}

func runProjectAdd(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	templateRepo := sqlite.NewTemplateRepository(db)
	taskRepo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	var template *domain.ProjectTemplate
	if addProjectTemplate != "" {
		template, err = lookupTemplate(ctx, templateRepo, addProjectTemplate)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		name, err = promptForInput("Project name", "")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		if name == "" {
			return fmt.Errorf("project name is required")
		}
	}

	project := domain.NewProject(name)
	project.Description = addProjectDescription
	project.IsFavorite = addProjectFavorite
	project.Notes = addProjectNotes

	if addProjectParent != "" {
		parentID, err := lookupProjectID(ctx, repo, addProjectParent)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
		project.ParentID = parentID
	} else if !cmd.Flags().Changed("parent") {
		fmt.Println()
		fmt.Println(styles.Info.Render("Select parent project (optional):"))
		parentID, err := selectParentProject(repo, ctx, styles, 0)
		if err != nil {
			return fmt.Errorf("failed to select parent: %w", err)
		}
		project.ParentID = parentID
	}

	if template != nil && template.ProjectDefaults != nil {
		if addProjectColor == "" && !cmd.Flags().Changed("color") {
			project.Color = template.ProjectDefaults.Color
		}
		if addProjectIcon == "" && !cmd.Flags().Changed("icon") {
			project.Icon = template.ProjectDefaults.Icon
		}
	}

	if addProjectColor != "" {
		project.Color = addProjectColor
	} else if !cmd.Flags().Changed("color") && project.Color == "" {
		color, err := promptForColor(styles)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
		project.Color = color
	}

	if addProjectIcon != "" {
		project.Icon = addProjectIcon
	} else if !cmd.Flags().Changed("icon") && project.Icon == "" {
		icon, err := promptForIcon(styles)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
		project.Icon = icon
	}

	if err := project.Validate(); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Validation failed: %v", err)))
		return nil
	}

	if project.ParentID != nil {
		if err := repo.ValidateHierarchy(ctx, 0, *project.ParentID); err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Invalid parent: %v", err)))
			return nil
		}
	}

	if err := repo.Create(ctx, project); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create project: %v", err)))
		return nil
	}

	tasksCreated := 0
	if template != nil && len(template.TaskDefinitions) > 0 {
		for _, taskDef := range template.TaskDefinitions {
			task := domain.NewTask(taskDef.Title)
			task.Description = taskDef.Description
			task.Priority = domain.Priority(taskDef.Priority)
			task.Tags = taskDef.Tags
			task.ProjectID = &project.ID

			if err := taskRepo.Create(ctx, task); err != nil {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create task '%s': %v", taskDef.Title, err)))
			} else {
				tasksCreated++
			}
		}
	}

	project, err = repo.GetByID(ctx, project.ID)
	if err == nil {
		displayProjectCreated(project, styles)
		if tasksCreated > 0 {
			fmt.Printf("  %s %d task(s) created from template '%s'\n",
				styles.Info.Render("Tasks:"), tasksCreated, template.Name)
			fmt.Println()
		}
	} else {
		fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Project #%d created successfully!", project.ID)))
		if tasksCreated > 0 {
			fmt.Printf("  %d task(s) created from template\n", tasksCreated)
		}
	}

	return nil
}

func displayProjectCreated(project *domain.Project, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Project #%d created successfully!", project.ID)))
	fmt.Println()

	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}

	fmt.Printf("  %s %s %s\n", styles.Info.Render("Name:"), icon, project.Name)

	if project.Description != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Description:"), project.Description)
	}

	path := project.BuildPath()
	fmt.Printf("  %s %s\n", styles.Info.Render("Path:"), path)

	if project.Color != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Color:"), project.Color)
	}

	fmt.Printf("  %s %s\n", styles.Info.Render("Status:"), project.Status)

	if project.IsFavorite {
		fmt.Printf("  %s ★\n", styles.Info.Render("Favorite:"))
	}

	if len(project.Aliases) > 0 {
		fmt.Printf("  %s %s\n", styles.Info.Render("Aliases:"), project.FormatAliases())
	}

	fmt.Println()
}


var (
	listProjectAll      bool
	listProjectArchived bool
	listProjectFavorites bool
	listProjectStats    bool
)

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects in tree view",
	Long: `List all projects in hierarchical tree view.

Examples:
  taskflow project list
  taskflow project list --stats
  taskflow project list --all
  taskflow project list --archived
  taskflow project list --favorites`,
	RunE: runProjectList,
}

func init() {
	projectListCmd.Flags().BoolVar(&listProjectAll, "all", false, "Show all projects including archived")
	projectListCmd.Flags().BoolVar(&listProjectArchived, "archived", false, "Show only archived projects")
	projectListCmd.Flags().BoolVar(&listProjectFavorites, "favorites", false, "Show only favorite projects")
	projectListCmd.Flags().BoolVar(&listProjectStats, "stats", false, "Include task statistics")
}

func runProjectList(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	filter := repository.ProjectFilter{
		SortBy:    "name",
		SortOrder: "asc",
	}

	if listProjectArchived {
		filter.Status = domain.ProjectStatusArchived
	} else if !listProjectAll {
		filter.ExcludeArchived = true
	}

	if listProjectFavorites {
		isFav := true
		filter.IsFavorite = &isFav
	}

	projects, err := repo.List(ctx, filter)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list projects: %v", err)))
		return nil
	}

	displayProjectTree(projects, listProjectStats, repo, ctx, styles)

	return nil
}


var projectViewCmd = &cobra.Command{
	Use:   "view <id|name>",
	Short: "View detailed project information",
	Long: `View detailed information about a project including statistics.

Examples:
  taskflow project view 1
  taskflow project view "Backend"`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectView,
}

func runProjectView(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load project: %v", err)))
		return nil
	}

	stats, err := repo.GetTaskCountByStatus(ctx, project.ID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load statistics: %v", err)))
		return nil
	}

	children, err := repo.GetChildren(ctx, project.ID)
	if err != nil {
		children = []*domain.Project{}
	}

	displayProjectDetails(project, stats, children, styles)

	return nil
}

func displayProjectDetails(project *domain.Project, stats map[domain.Status]int, children []*domain.Project, styles *theme.Styles) {
	fmt.Println()

	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}

	title := fmt.Sprintf("%s %s (ID: %d)", icon, project.Name, project.ID)
	fmt.Println(styles.Title.Render(title))
	fmt.Println()

	fmt.Printf("  %s %s", styles.Info.Render("Status:"), project.Status)
	if project.IsFavorite {
		fmt.Print(" ★")
	}
	fmt.Println()

	path := project.BuildPath()
	if path != project.Name {
		fmt.Printf("  %s %s\n", styles.Info.Render("Path:"), path)
	}

	fmt.Printf("  %s %s\n", styles.Info.Render("Created:"), project.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("  %s %s\n", styles.Info.Render("Updated:"), project.UpdatedAt.Format("2006-01-02 15:04"))

	if project.Color != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Color:"), project.Color)
	}

	if len(project.Aliases) > 0 {
		fmt.Println()
		fmt.Println(styles.Subtitle.Render("Aliases:"))
		for i, alias := range project.Aliases {
			fmt.Printf("  %d. %s\n", i+1, alias)
		}
	}

	if project.Description != "" {
		fmt.Println()
		fmt.Println(styles.Subtitle.Render("Description:"))
		fmt.Printf("  %s\n", project.Description)
	}

	if project.HasNotes() {
		fmt.Println()
		fmt.Println(styles.Subtitle.Render("Notes:"))

		notesPreview := strings.TrimSpace(project.Notes)
		lines := strings.Split(notesPreview, "\n")

		previewLines := 3
		if len(lines) > previewLines {
			for i := 0; i < previewLines; i++ {
				fmt.Printf("  %s\n", lines[i])
			}
			fmt.Printf("  %s\n", styles.Info.Render(fmt.Sprintf("... (%d more lines, use 'project note %d' to view/edit)", len(lines)-previewLines, project.ID)))
		} else {
			for _, line := range lines {
				fmt.Printf("  %s\n", line)
			}
		}
	}

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Task Statistics:"))
	fmt.Printf("  %s\n", formatProjectStats(stats, styles))

	if len(children) > 0 {
		fmt.Println()
		fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Child Projects (%d):", len(children))))
		for _, child := range children {
			childIcon := child.Icon
			if childIcon == "" {
				childIcon = "📦"
			}
			fmt.Printf("  %s %s\n", childIcon, child.Name)
		}
	}

	fmt.Println()
}


var (
	deleteProjectConfirm bool
)

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete a project",
	Long: `Delete a project and optionally its children.

By default, deleting a project:
  - Cascades to delete all child projects
  - Preserves tasks (sets project_id to NULL)

Examples:
  taskflow project delete 1
  taskflow project delete "Backend"
  taskflow project delete 1 --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectDelete,
}

func init() {
	projectDeleteCmd.Flags().BoolVar(&deleteProjectConfirm, "confirm", false, "Skip confirmation prompt")
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load project: %v", err)))
		return nil
	}

	descendants, err := repo.GetDescendants(ctx, project.ID)
	if err != nil {
		descendants = []*domain.Project{}
	}

	taskCount, err := repo.GetTaskCount(ctx, project.ID)
	if err != nil {
		taskCount = 0
	}

	if !deleteProjectConfirm {
		fmt.Println()
		fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Delete project '%s' (ID: %d)?", project.Name, project.ID)))

		if len(descendants) > 0 {
			fmt.Printf("  - %d child project(s) will be deleted\n", len(descendants))
		}
		if taskCount > 0 {
			fmt.Printf("  - %d task(s) will be orphaned (project_id set to NULL)\n", taskCount)
		}

		fmt.Println()
		if !promptForConfirmation("Proceed?") {
			fmt.Println(styles.Info.Render("Cancelled."))
			return nil
		}
	}

	if err := repo.Delete(ctx, project.ID); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to delete project: %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Project '%s' deleted successfully!", project.Name)))
	if len(descendants) > 0 {
		fmt.Println(styles.Info.Render(fmt.Sprintf("  %d child project(s) deleted", len(descendants))))
	}
	if taskCount > 0 {
		fmt.Println(styles.Info.Render(fmt.Sprintf("  %d task(s) preserved", taskCount)))
	}
	fmt.Println()

	return nil
}


func promptForInput(prompt string, defaultVal string) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" && defaultVal != "" {
		return defaultVal, nil
	}

	return input, nil
}

func promptForConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s (y/N): ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func buildTreeView(projects []*domain.Project, parentID *int64, prefix string, isLast bool, includeStats bool, taskCounts map[int64]int, styles *theme.Styles) []string {
	lines := []string{}

	children := []*domain.Project{}
	for _, p := range projects {
		if parentID == nil && p.ParentID == nil {
			children = append(children, p)
		} else if parentID != nil && p.ParentID != nil && *p.ParentID == *parentID {
			children = append(children, p)
		}
	}

	for i, project := range children {
		isLastChild := i == len(children)-1

		var connector, extension string
		if isLastChild {
			connector = "└── "
			extension = "    "
		} else {
			connector = "├── "
			extension = "│   "
		}

		icon := project.Icon
		if icon == "" {
			icon = "📦"
		}

		name := project.Name
		favorite := ""
		if project.IsFavorite {
			favorite = " ★"
		}

		statsStr := ""
		if includeStats {
			taskCount := taskCounts[project.ID]
			if taskCount > 0 {
				statsStr = fmt.Sprintf(" (%d tasks)", taskCount)
			}
		}

		line := fmt.Sprintf("%s%s%s %s%s%s", prefix, connector, icon, name, statsStr, favorite)
		lines = append(lines, line)

		childLines := buildTreeView(projects, &project.ID, prefix+extension, isLastChild, includeStats, taskCounts, styles)
		lines = append(lines, childLines...)
	}

	return lines
}

func displayProjectTree(projects []*domain.Project, includeStats bool, repo repository.ProjectRepository, ctx context.Context, styles *theme.Styles) {
	if len(projects) == 0 {
		fmt.Println()
		fmt.Println(styles.Info.Render("No projects found."))
		fmt.Println()
		return
	}

	taskCounts := make(map[int64]int)
	if includeStats {
		for _, p := range projects {
			count, err := repo.GetTaskCount(ctx, p.ID)
			if err == nil {
				taskCounts[p.ID] = count
			}
		}
	}

	fmt.Println()
	fmt.Println(styles.Title.Render("Projects"))
	fmt.Println()

	lines := buildTreeView(projects, nil, "", false, includeStats, taskCounts, styles)
	for _, line := range lines {
		fmt.Println(line)
	}

	fmt.Println()
	fmt.Printf("Total: %d project(s)\n", len(projects))
	fmt.Println()
}

func promptForColor(styles *theme.Styles) (string, error) {
	colors := domain.GetValidColors()

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Available colors:"))
	fmt.Println()

	for i, color := range colors {
		fmt.Printf("  %2d. %s", i+1, color)
		if (i+1)%4 == 0 {
			fmt.Println()
		} else {
			fmt.Print("\t")
		}
	}
	fmt.Println()
	fmt.Println()

	input, err := promptForInput("Select color number or name", "")
	if err != nil {
		return "", err
	}

	if input == "" {
		return "", nil
	}

	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(colors) {
			return colors[num-1], nil
		}
		return "", fmt.Errorf("invalid color number: %d", num)
	}

	for _, color := range colors {
		if strings.EqualFold(color, input) {
			return color, nil
		}
	}

	return "", fmt.Errorf("invalid color: %s", input)
}

func promptForIcon(styles *theme.Styles) (string, error) {
	icons := domain.GetCommonIcons()

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Common icons:"))
	fmt.Println()

	for i, icon := range icons {
		fmt.Printf("  %2d. %s", i+1, icon)
		if (i+1)%8 == 0 {
			fmt.Println()
		} else {
			fmt.Print("  ")
		}
	}
	fmt.Println()
	fmt.Println()

	input, err := promptForInput("Select icon number or enter custom", "")
	if err != nil {
		return "", err
	}

	if input == "" {
		return "", nil
	}

	if num, err := strconv.Atoi(input); err == nil {
		if num >= 1 && num <= len(icons) {
			return icons[num-1], nil
		}
		return "", fmt.Errorf("invalid icon number: %d", num)
	}

	return input, nil
}

func selectParentProject(repo repository.ProjectRepository, ctx context.Context, styles *theme.Styles, excludeID int64) (*int64, error) {
	filter := repository.ProjectFilter{
		ExcludeArchived: true,
		SortBy:          "name",
		SortOrder:       "asc",
	}

	projects, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	filtered := []*domain.Project{}
	for _, p := range projects {
		if p.ID != excludeID {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Available parent projects:"))
	fmt.Println()
	fmt.Println("  0. <None - Root Project>")

	for i, p := range filtered {
		icon := p.Icon
		if icon == "" {
			icon = "📦"
		}
		path := p.BuildPath()
		fmt.Printf("  %d. %s %s\n", i+1, icon, path)
	}
	fmt.Println()

	input, err := promptForInput("Select parent number (0 for root)", "0")
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	if num == 0 {
		return nil, nil
	}

	if num < 1 || num > len(filtered) {
		return nil, fmt.Errorf("invalid selection: %d", num)
	}

	parentID := filtered[num-1].ID
	return &parentID, nil
}

func formatProjectStats(stats map[domain.Status]int, styles *theme.Styles) string {
	total := 0
	for _, count := range stats {
		total += count
	}

	if total == 0 {
		return "No tasks"
	}

	completed := stats[domain.StatusCompleted]
	percentage := 0
	if total > 0 {
		percentage = (completed * 100) / total
	}

	parts := []string{}
	if stats[domain.StatusPending] > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", stats[domain.StatusPending]))
	}
	if stats[domain.StatusInProgress] > 0 {
		parts = append(parts, fmt.Sprintf("%d in progress", stats[domain.StatusInProgress]))
	}
	if completed > 0 {
		parts = append(parts, fmt.Sprintf("%d completed (%d%%)", completed, percentage))
	}
	if stats[domain.StatusCancelled] > 0 {
		parts = append(parts, fmt.Sprintf("%d cancelled", stats[domain.StatusCancelled]))
	}

	return fmt.Sprintf("Total: %d tasks (%s)", total, strings.Join(parts, ", "))
}


var (
	updateProjectName        string
	updateProjectDescription string
	updateProjectParent      string
	updateProjectColor       string
	updateProjectIcon        string
	updateProjectNoParent    bool
	updateProjectFavorite    bool
	updateProjectNoFavorite  bool
	updateProjectAddAlias    string
	updateProjectRemoveAlias string
	updateProjectNotes       string
)

var projectUpdateCmd = &cobra.Command{
	Use:   "update [project-id-or-name]",
	Short: "Update an existing project",
	Long: `Update an existing project with new values.
Only the fields you specify will be updated; all other fields remain unchanged.

You can update project properties like name, description, parent, color, icon, favorite status, and aliases.
Use --parent "" or --no-parent to make a project a root project (remove parent).

Examples:
  taskflow project update 1 --name "New Name"
  taskflow project update "Backend" --description "Backend services"
  taskflow project update 2 --parent "Development" --color green
  taskflow project update "API" --no-parent  # Make root project
  taskflow project update 3 --favorite       # Mark as favorite
  taskflow project update 4 --no-favorite    # Remove favorite
  taskflow project update 5 --icon 🚀 --color blue
  taskflow project update 1 --add-alias api-backend    # Add alias
  taskflow project update "Backend" --remove-alias old-name  # Remove alias`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectUpdate,
}

func init() {
	projectUpdateCmd.Flags().StringVarP(&updateProjectName, "name", "n", "", "Update project name")
	projectUpdateCmd.Flags().StringVarP(&updateProjectDescription, "description", "d", "", "Update description")
	projectUpdateCmd.Flags().StringVarP(&updateProjectParent, "parent", "p", "", "Update parent (ID or name, \"\" to clear)")
	projectUpdateCmd.Flags().BoolVar(&updateProjectNoParent, "no-parent", false, "Remove parent (make root project)")
	projectUpdateCmd.Flags().StringVarP(&updateProjectColor, "color", "c", "", "Update color")
	projectUpdateCmd.Flags().StringVarP(&updateProjectIcon, "icon", "i", "", "Update icon")
	projectUpdateCmd.Flags().BoolVar(&updateProjectFavorite, "favorite", false, "Mark as favorite")
	projectUpdateCmd.Flags().BoolVar(&updateProjectNoFavorite, "no-favorite", false, "Remove favorite status")
	projectUpdateCmd.Flags().StringVar(&updateProjectAddAlias, "add-alias", "", "Add a new alias to the project")
	projectUpdateCmd.Flags().StringVar(&updateProjectRemoveAlias, "remove-alias", "", "Remove an alias from the project")
	projectUpdateCmd.Flags().StringVar(&updateProjectNotes, "notes", "", "Update project notes (markdown supported)")
}

func runProjectUpdate(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Project not found: %v", err)))
		return nil
	}

	nameSet := cmd.Flags().Changed("name")
	descriptionSet := cmd.Flags().Changed("description")
	parentSet := cmd.Flags().Changed("parent")
	colorSet := cmd.Flags().Changed("color")
	iconSet := cmd.Flags().Changed("icon")
	favoriteSet := cmd.Flags().Changed("favorite")
	noFavoriteSet := cmd.Flags().Changed("no-favorite")
	addAliasSet := cmd.Flags().Changed("add-alias")
	removeAliasSet := cmd.Flags().Changed("remove-alias")
	notesSet := cmd.Flags().Changed("notes")

	if !nameSet && !descriptionSet && !parentSet && !updateProjectNoParent && !colorSet && !iconSet && !favoriteSet && !noFavoriteSet && !addAliasSet && !removeAliasSet && !notesSet {
		fmt.Println(styles.Info.Render("No updates specified. Use --help to see available flags."))
		return nil
	}

	modified := false

	if nameSet {
		project.Name = updateProjectName
		modified = true
	}

	if descriptionSet {
		project.Description = updateProjectDescription
		modified = true
	}

	if parentSet || updateProjectNoParent {
		if updateProjectNoParent || updateProjectParent == "" {
			project.ParentID = nil
		} else {
			newParentID, err := lookupProjectID(ctx, repo, updateProjectParent)
			if err != nil {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
				return nil
			}
			project.ParentID = newParentID
		}
		modified = true
	}

	if colorSet {
		if updateProjectColor == "" {
			project.Color = ""
		} else {
			project.Color = updateProjectColor
		}
		modified = true
	}

	if iconSet {
		if updateProjectIcon == "" {
			project.Icon = ""
		} else {
			project.Icon = updateProjectIcon
		}
		modified = true
	}

	if favoriteSet || noFavoriteSet {
		if favoriteSet {
			project.IsFavorite = true
		} else if noFavoriteSet {
			project.IsFavorite = false
		}
		modified = true
	}

	if addAliasSet {
		newAlias := updateProjectAddAlias

		if err := domain.IsValidAliasFormat(newAlias); err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Invalid alias format: %v", err)))
			return nil
		}

		if project.HasAlias(newAlias) {
			fmt.Println(styles.Info.Render(fmt.Sprintf("Alias '%s' is already assigned to this project.", newAlias)))
			return nil
		}

		if len(project.Aliases) >= 10 {
			fmt.Println(styles.Error.Render("✗ Project cannot have more than 10 aliases."))
			return nil
		}

		if err := repo.ValidateAliasUniqueness(ctx, newAlias, &project.ID); err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}

		project.Aliases = append(project.Aliases, newAlias)
		modified = true
	}

	if removeAliasSet {
		aliasToRemove := updateProjectRemoveAlias

		foundIdx := -1
		for i, alias := range project.Aliases {
			if strings.EqualFold(alias, aliasToRemove) {
				foundIdx = i
				break
			}
		}

		if foundIdx == -1 {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Alias '%s' not found in project.", aliasToRemove)))
			return nil
		}

		project.Aliases = append(project.Aliases[:foundIdx], project.Aliases[foundIdx+1:]...)
		modified = true
	}

	if notesSet {
		project.Notes = updateProjectNotes
		modified = true
	}

	if !modified {
		fmt.Println(styles.Info.Render("No updates specified."))
		return nil
	}

	if err := repo.Update(ctx, project); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to update project: %v", err)))
		return nil
	}

	updated, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load updated project: %v", err)))
		return nil
	}

	displayProjectUpdated(updated, styles)

	return nil
}

func displayProjectUpdated(project *domain.Project, styles *theme.Styles) {
	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ %s %s (ID: %d) updated successfully!", icon, project.Name, project.ID)))
	fmt.Println()

	fmt.Printf("  %s %s\n", styles.Info.Render("Status:"), project.Status)

	if project.ParentID != nil {
		path := project.BuildPath()
		fmt.Printf("  %s %s\n", styles.Info.Render("Path:"), path)
	} else {
		fmt.Printf("  %s %s\n", styles.Info.Render("Type:"), "Root Project")
	}

	if project.Description != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Description:"), project.Description)
	}

	if project.Color != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Color:"), project.Color)
	}

	if project.IsFavorite {
		fmt.Printf("  %s %s\n", styles.Info.Render("Favorite:"), "★ Yes")
	}

	if len(project.Aliases) > 0 {
		fmt.Printf("  %s %s\n", styles.Info.Render("Aliases:"), project.FormatAliases())
	}

	fmt.Printf("  %s %s\n", styles.Info.Render("Updated:"), project.UpdatedAt.Format("2006-01-02 15:04"))

	fmt.Println()
}


var (
	archiveNoRecursive bool
	archiveConfirm     bool
)

var projectArchiveCmd = &cobra.Command{
	Use:   "archive [project-id-or-name]",
	Short: "Archive a project",
	Long: `Archive a project, marking it as inactive and hiding it from default listings.

By default, archiving a project also archives all its child projects (recursive).
Use --no-recursive to archive only the specified project.

Archived projects can be viewed with 'taskflow project list --all' and can be
restored using 'taskflow project unarchive'.

Examples:
  taskflow project archive "Backend"           # Archive with children (default)
  taskflow project archive 1 --no-recursive    # Archive only this project
  taskflow project archive 2 --confirm         # Skip confirmation prompt`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectArchive,
}

func init() {
	projectArchiveCmd.Flags().BoolVar(&archiveNoRecursive, "no-recursive", false, "Archive only this project, not children")
	projectArchiveCmd.Flags().BoolVarP(&archiveConfirm, "confirm", "y", false, "Skip confirmation prompt")
}

func runProjectArchive(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Project not found: %v", err)))
		return nil
	}

	if project.Status == domain.ProjectStatusArchived {
		fmt.Println(styles.Info.Render(fmt.Sprintf("Project '%s' is already archived.", project.Name)))
		return nil
	}

	if project.Status == domain.ProjectStatusCompleted {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Cannot archive completed project '%s'. Completed is a terminal state.", project.Name)))
		return nil
	}

	descendants := []*domain.Project{}
	if !archiveNoRecursive {
		descendants, err = repo.GetDescendants(ctx, project.ID)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to get child projects: %v", err)))
			return nil
		}
	}

	taskCount, err := repo.GetTaskCount(ctx, project.ID)
	if err != nil {
		taskCount = 0
	}

	if !archiveConfirm {
		fmt.Println()
		icon := project.Icon
		if icon == "" {
			icon = "📦"
		}
		fmt.Printf("Archive project '%s %s' (ID: %d)?\n", icon, project.Name, project.ID)
		fmt.Printf("  - Status will change: %s → %s\n", project.Status, domain.ProjectStatusArchived)

		if !archiveNoRecursive && len(descendants) > 0 {
			fmt.Printf("  - %d child project(s) will also be archived\n", len(descendants))
		} else if archiveNoRecursive && len(descendants) > 0 {
			fmt.Printf("  - %d child project(s) will remain active (--no-recursive)\n", len(descendants))
		}

		if taskCount > 0 {
			fmt.Printf("  - %d task(s) will remain accessible\n", taskCount)
		}

		fmt.Println()
		if !promptForConfirmation("Proceed?") {
			fmt.Println(styles.Info.Render("Archive cancelled."))
			return nil
		}
	}

	if err := repo.Archive(ctx, project.ID); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to archive project: %v", err)))
		return nil
	}

	archivedCount := 1

	if !archiveNoRecursive && len(descendants) > 0 {
		for _, desc := range descendants {
			if err := repo.Archive(ctx, desc.ID); err != nil {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to archive child project '%s': %v", desc.Name, err)))
			} else {
				archivedCount++
			}
		}
	}

	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ %s %s archived successfully!", icon, project.Name)))
	if archivedCount > 1 {
		fmt.Printf("  %d project(s) archived in total\n", archivedCount)
	}
	if taskCount > 0 {
		fmt.Printf("  %d task(s) preserved and remain accessible\n", taskCount)
	}
	fmt.Println()

	return nil
}


var (
	unarchiveRecursive bool
)

var projectUnarchiveCmd = &cobra.Command{
	Use:   "unarchive [project-id-or-name]",
	Short: "Unarchive a project",
	Long: `Restore an archived project to active status.

By default, only the specified project is unarchived. Use --recursive to also
unarchive all child projects.

Examples:
  taskflow project unarchive "Backend"        # Unarchive only this project
  taskflow project unarchive 1 --recursive    # Unarchive with all children`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectUnarchive,
}

func init() {
	projectUnarchiveCmd.Flags().BoolVarP(&unarchiveRecursive, "recursive", "r", false, "Also unarchive child projects")
}

func runProjectUnarchive(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Project not found: %v", err)))
		return nil
	}

	if project.Status != domain.ProjectStatusArchived {
		fmt.Println(styles.Info.Render(fmt.Sprintf("Project '%s' is not archived (status: %s).", project.Name, project.Status)))
		return nil
	}

	descendants := []*domain.Project{}
	if unarchiveRecursive {
		descendants, err = repo.GetDescendants(ctx, project.ID)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to get child projects: %v", err)))
			return nil
		}
	}

	if err := repo.Unarchive(ctx, project.ID); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to unarchive project: %v", err)))
		return nil
	}

	unarchivedCount := 1

	if unarchiveRecursive && len(descendants) > 0 {
		for _, desc := range descendants {
			if desc.Status == domain.ProjectStatusArchived {
				if err := repo.Unarchive(ctx, desc.ID); err != nil {
					fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to unarchive child project '%s': %v", desc.Name, err)))
				} else {
					unarchivedCount++
				}
			}
		}
	}

	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ %s %s unarchived successfully!", icon, project.Name)))
	if unarchivedCount > 1 {
		fmt.Printf("  %d project(s) restored to active status\n", unarchivedCount)
	}
	fmt.Println()

	return nil
}


var projectAliasCmd = &cobra.Command{
	Use:   "alias <project-id-or-name> <new-alias>",
	Short: "Add an alias to a project",
	Long: `Add a new alias to an existing project.

Aliases provide alternative ways to reference a project. Alias requirements:
  - 2-30 characters long
  - Lowercase alphanumeric characters, hyphens, and underscores only
  - Must be unique across all projects
  - Case-insensitive matching (e.g., 'api-v2' and 'API-V2' are the same)

Examples:
  taskflow project alias "Backend" api-service
  taskflow project alias 1 "web-app"
  taskflow project alias "MyProject" proj-alias`,
	Args: cobra.ExactArgs(2),
	RunE: runProjectAlias,
}

func runProjectAlias(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load project: %v", err)))
		return nil
	}

	newAlias := args[1]

	if err := domain.IsValidAliasFormat(newAlias); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Invalid alias format: %v", err)))
		return nil
	}

	for _, existingAlias := range project.Aliases {
		if strings.EqualFold(existingAlias, newAlias) {
			fmt.Println(styles.Info.Render(fmt.Sprintf("Alias '%s' is already assigned to this project.", newAlias)))
			return nil
		}
	}

	if err := repo.ValidateAliasUniqueness(ctx, newAlias, projectID); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project.Aliases = append(project.Aliases, newAlias)

	if len(project.Aliases) > 10 {
		fmt.Println(styles.Error.Render("✗ Project cannot have more than 10 aliases."))
		return nil
	}

	if err := project.Validate(); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Validation failed: %v", err)))
		return nil
	}

	if err := repo.Update(ctx, project); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to add alias: %v", err)))
		return nil
	}

	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Alias '%s' added to project %s %s", newAlias, icon, project.Name)))
	fmt.Printf("  Project now has %d alias(es)\n", len(project.Aliases))
	fmt.Println()

	return nil
}


var projectAliasesCmd = &cobra.Command{
	Use:   "aliases <project-id-or-name>",
	Short: "List all aliases for a project",
	Long: `Display all aliases assigned to a project.

Examples:
  taskflow project aliases "Backend"
  taskflow project aliases 1
  taskflow project aliases api-service  # Can use an alias to find the project`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectAliases,
}

func runProjectAliases(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load project: %v", err)))
		return nil
	}

	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Title.Render(fmt.Sprintf("%s %s (ID: %d) - Aliases", icon, project.Name, project.ID)))
	fmt.Println()

	if len(project.Aliases) == 0 {
		fmt.Println(styles.Info.Render("  No aliases assigned."))
	} else {
		for i, alias := range project.Aliases {
			fmt.Printf("  %d. %s\n", i+1, alias)
		}
	}

	fmt.Printf("\nTotal: %d alias(es)\n", len(project.Aliases))
	fmt.Println()

	return nil
}


var projectUnaliasCmd = &cobra.Command{
	Use:   "unalias <alias>",
	Short: "Remove an alias from a project",
	Long: `Remove an alias from a project.

The alias argument can be the alias itself. The command will find the project
that owns this alias and remove it.

Examples:
  taskflow project unalias api-service
  taskflow project unalias web-app`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectUnalias,
}

func runProjectUnalias(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	aliasToRemove := args[0]

	project, err := repo.GetByAlias(ctx, aliasToRemove)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Alias '%s' not found: %v", aliasToRemove, err)))
		return nil
	}

	foundIdx := -1
	for i, alias := range project.Aliases {
		if strings.EqualFold(alias, aliasToRemove) {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Alias '%s' not found in project.", aliasToRemove)))
		return nil
	}

	project.Aliases = append(project.Aliases[:foundIdx], project.Aliases[foundIdx+1:]...)

	if err := repo.Update(ctx, project); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to remove alias: %v", err)))
		return nil
	}

	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Alias '%s' removed from project %s %s", aliasToRemove, icon, project.Name)))
	if len(project.Aliases) > 0 {
		fmt.Printf("  Project now has %d alias(es)\n", len(project.Aliases))
	} else {
		fmt.Println("  Project has no remaining aliases.")
	}
	fmt.Println()

	return nil
}


var projectNoteCmd = &cobra.Command{
	Use:   "note <project-id-or-name>",
	Short: "Edit project notes using $EDITOR",
	Long: `Open project notes in your default editor for viewing or editing.

The command uses the $EDITOR environment variable to determine which editor to use.
If $EDITOR is not set, it will try to use: nano, vim, or vi (in that order).

Notes support markdown formatting and can be up to 10,000 characters.

Examples:
  taskflow project note "Backend"
  taskflow project note 1
  taskflow project note api-service  # Using alias`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectNote,
}

func runProjectNote(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	projectID, err := lookupProjectID(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project, err := repo.GetByID(ctx, *projectID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to load project: %v", err)))
		return nil
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = detectAvailableEditor()
		if editor == "" {
			fmt.Println(styles.Error.Render("✗ No editor found. Set $EDITOR or install nano/vim/vi"))
			return nil
		}
	}

	tmpFile, err := os.CreateTemp("", fmt.Sprintf("taskflow-notes-%d-*.md", project.ID))
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create temp file: %v", err)))
		return nil
	}
	tmpFilePath := tmpFile.Name()
	defer os.Remove(tmpFilePath)

	if _, err := tmpFile.WriteString(project.Notes); err != nil {
		tmpFile.Close()
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to write notes: %v", err)))
		return nil
	}
	tmpFile.Close()

	fmt.Println(styles.Info.Render(fmt.Sprintf("Opening notes for project '%s' in %s...", project.Name, editor)))

	editorCmd := exec.Command(editor, tmpFilePath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Editor failed: %v", err)))
		return nil
	}

	modifiedContent, err := os.ReadFile(tmpFilePath)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to read modified notes: %v", err)))
		return nil
	}

	newNotes := string(modifiedContent)

	if newNotes == project.Notes {
		fmt.Println(styles.Info.Render("No changes made to notes."))
		return nil
	}

	if len(newNotes) > 10000 {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Notes too long: %d characters (max 10,000)", len(newNotes))))
		return nil
	}

	project.Notes = newNotes
	if err := repo.Update(ctx, project); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to update notes: %v", err)))
		return nil
	}

	fmt.Println()
	icon := project.Icon
	if icon == "" {
		icon = "📦"
	}
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Notes updated for project %s %s", icon, project.Name)))

	notesLen := len(strings.TrimSpace(newNotes))
	if notesLen > 0 {
		fmt.Printf("  Notes length: %d characters\n", notesLen)
	} else {
		fmt.Println("  Notes cleared")
	}
	fmt.Println()

	return nil
}

func detectAvailableEditor() string {
	editors := []string{"nano", "vim", "vi"}
	for _, editor := range editors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}
	return ""
}
