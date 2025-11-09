package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage project templates",
	Long: `Manage project templates for creating projects with pre-defined tasks.

Templates allow you to quickly create new projects with a standard set of tasks,
making it easy to replicate project structures and workflows.`,
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateShowCmd)
	templateCmd.AddCommand(templateEditCmd)
	templateCmd.AddCommand(templateDeleteCmd)
	templateCmd.AddCommand(templateApplyCmd)
}

var (
	createTemplateDescription string
	createTemplateColor       string
	createTemplateIcon        string
)

var templateCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new project template",
	Long: `Create a new project template with pre-defined tasks.

Examples:
  taskflow template create "Web Application"
  taskflow template create "Backend Service" --description "Microservice template"
  taskflow template create "Frontend App" --color blue --icon 🚀`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTemplateCreate,
}

func init() {
	templateCreateCmd.Flags().StringVarP(&createTemplateDescription, "description", "d", "", "Template description")
	templateCreateCmd.Flags().StringVar(&createTemplateColor, "color", "", "Default project color")
	templateCreateCmd.Flags().StringVar(&createTemplateIcon, "icon", "", "Default project icon")
}

func runTemplateCreate(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewTemplateRepository(db)
	ctx := context.Background()

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		fmt.Print("Template name: ")
		fmt.Scanln(&name)
	}

	if strings.TrimSpace(name) == "" {
		fmt.Println(styles.Error.Render("✗ Template name cannot be empty"))
		return nil
	}

	template := domain.NewTemplate(name)

	if createTemplateDescription != "" {
		template.Description = createTemplateDescription
	} else {
		fmt.Print("Description (optional): ")
		fmt.Scanln(&template.Description)
	}

	hasDefaults := false
	fmt.Print("Add default project properties? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) == "y" {
		hasDefaults = true
		template.ProjectDefaults = &domain.ProjectDefaults{}

		if createTemplateColor != "" {
			template.ProjectDefaults.Color = createTemplateColor
		} else {
			color, err := promptForColor(styles)
			if err != nil {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			} else if color != "" {
				template.ProjectDefaults.Color = color
			}
		}

		if createTemplateIcon != "" {
			template.ProjectDefaults.Icon = createTemplateIcon
		} else {
			icon, err := promptForIcon(styles)
			if err != nil {
				fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			} else if icon != "" {
				template.ProjectDefaults.Icon = icon
			}
		}

		if template.ProjectDefaults.Color == "" && template.ProjectDefaults.Icon == "" {
			template.ProjectDefaults = nil
			hasDefaults = false
		}
	}

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Add tasks to template:"))
	fmt.Println()

	taskNum := 1
	for {
		fmt.Printf("%s Task %d %s\n", styles.Info.Render("───"), taskNum, styles.Info.Render("───"))

		fmt.Print("  Title (or press Enter to finish): ")
		var title string
		fmt.Scanln(&title)

		if strings.TrimSpace(title) == "" {
			if taskNum == 1 {
				fmt.Println(styles.Error.Render("✗ Template must have at least one task"))
				continue
			}
			break
		}

		taskDef := domain.NewTaskDefinition(title)

		fmt.Print("  Description (optional): ")
		var desc string
		fmt.Scanln(&desc)
		taskDef.Description = desc

		fmt.Print("  Priority (low/medium/high/urgent) [medium]: ")
		var priority string
		fmt.Scanln(&priority)
		if priority != "" {
			taskDef.Priority = priority
		}

		fmt.Print("  Tags (comma-separated, optional): ")
		var tagsInput string
		fmt.Scanln(&tagsInput)
		if tagsInput != "" {
			tags := strings.Split(tagsInput, ",")
			for i, tag := range tags {
				tags[i] = strings.TrimSpace(tag)
			}
			taskDef.Tags = tags
		}

		if err := template.AddTaskDefinition(taskDef); err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to add task: %v", err)))
			continue
		}

		fmt.Println(styles.Success.Render("  ✓ Task added"))
		fmt.Println()
		taskNum++
	}

	if err := repo.Create(ctx, template); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create template: %v", err)))
		return nil
	}

	displayTemplateCreated(template, hasDefaults, styles)

	return nil
}

func displayTemplateCreated(template *domain.ProjectTemplate, hasDefaults bool, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Template '%s' created successfully!", template.Name)))
	fmt.Println()

	fmt.Printf("  %s %d\n", styles.Info.Render("ID:"), template.ID)
	fmt.Printf("  %s %s\n", styles.Info.Render("Name:"), template.Name)

	if template.Description != "" {
		fmt.Printf("  %s %s\n", styles.Info.Render("Description:"), template.Description)
	}

	if hasDefaults && template.ProjectDefaults != nil {
		fmt.Printf("  %s", styles.Info.Render("Defaults:"))
		if template.ProjectDefaults.Color != "" {
			fmt.Printf(" color=%s", template.ProjectDefaults.Color)
		}
		if template.ProjectDefaults.Icon != "" {
			fmt.Printf(" icon=%s", template.ProjectDefaults.Icon)
		}
		fmt.Println()
	}

	fmt.Printf("  %s %d\n", styles.Info.Render("Tasks:"), len(template.TaskDefinitions))

	fmt.Println()
}

var (
	listTemplatesSearch string
	listTemplatesPage   int
	listTemplatesLimit  int
	listTemplatesAll    bool
)

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all templates",
	Long: `List all project templates with optional filtering and pagination.

Examples:
  taskflow template list
  taskflow template list --search "web"
  taskflow template list --page 2 --page-size 10
  taskflow template list --all`,
	RunE: runTemplateList,
}

func init() {
	templateListCmd.Flags().StringVar(&listTemplatesSearch, "search", "", "Search query")
	templateListCmd.Flags().IntVar(&listTemplatesPage, "page", 1, "Page number")
	templateListCmd.Flags().IntVar(&listTemplatesLimit, "page-size", 0, "Number of templates per page")
	templateListCmd.Flags().BoolVar(&listTemplatesAll, "all", false, "Show all templates (disable pagination)")
}

func runTemplateList(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewTemplateRepository(db)
	ctx := context.Background()

	pageSize := listTemplatesLimit
	if pageSize == 0 {
		pageSize = cfg.DefaultPageSize
	}
	if pageSize > cfg.MaxPageSize {
		pageSize = cfg.MaxPageSize
	}

	filter := repository.TemplateFilter{
		SearchQuery: listTemplatesSearch,
		SortBy:      "created_at",
		SortOrder:   "desc",
	}

	if !listTemplatesAll {
		if listTemplatesPage < 1 {
			listTemplatesPage = 1
		}
		filter.Limit = pageSize
		filter.Offset = (listTemplatesPage - 1) * pageSize
	}

	totalCount, err := repo.Count(ctx, filter)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to count templates: %v", err)))
		return nil
	}

	templates, err := repo.List(ctx, filter)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list templates: %v", err)))
		return nil
	}

	if len(templates) == 0 {
		fmt.Println()
		if listTemplatesSearch != "" {
			fmt.Println(styles.Info.Render(fmt.Sprintf("No templates found matching '%s'", listTemplatesSearch)))
		} else {
			fmt.Println(styles.Info.Render("No templates found"))
		}
		fmt.Println()
		return nil
	}

	displayTemplatesTable(templates, styles, filter, listTemplatesPage, pageSize, totalCount)

	return nil
}

func displayTemplatesTable(templates []*domain.ProjectTemplate, styles *theme.Styles, filter repository.TemplateFilter, currentPage, pageSize int, totalCount int64) {
	fmt.Println()
	fmt.Println(styles.Title.Render("Templates"))
	fmt.Println()

	if filter.SearchQuery != "" {
		fmt.Printf("  Search: %s\n\n", filter.SearchQuery)
	}

	headers := []string{
		styles.Header.Render("ID"),
		styles.Header.Render("Name"),
		styles.Header.Render("Description"),
		styles.Header.Render("Tasks"),
		styles.Header.Render("Created"),
	}
	fmt.Println(strings.Join(headers, " │ "))

	separator := strings.Repeat("─", 100)
	fmt.Println(styles.Separator.Render(separator))

	for _, tmpl := range templates {
		desc := tmpl.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		if desc == "" {
			desc = "-"
		}

		created := tmpl.CreatedAt.Format("2006-01-02")

		cells := []string{
			styles.Cell.Render(fmt.Sprintf("%-4d", tmpl.ID)),
			styles.Cell.Render(fmt.Sprintf("%-25s", truncate(tmpl.Name, 25))),
			styles.Cell.Render(fmt.Sprintf("%-40s", desc)),
			styles.Cell.Render(fmt.Sprintf("%-6d", len(tmpl.TaskDefinitions))),
			styles.Cell.Render(fmt.Sprintf("%-12s", created)),
		}

		fmt.Println(strings.Join(cells, " │ "))
	}

	fmt.Println()

	if filter.Limit > 0 {
		totalPages := int((totalCount + int64(pageSize) - 1) / int64(pageSize))
		startIdx := filter.Offset + 1
		endIdx := filter.Offset + len(templates)

		paginationInfo := fmt.Sprintf("Showing %d-%d of %d templates (Page %d of %d)",
			startIdx, endIdx, totalCount, currentPage, totalPages)

		fmt.Println(styles.Subtitle.Render(paginationInfo))

		if currentPage < totalPages {
			nextPageHint := fmt.Sprintf("Use --page %d to see the next page", currentPage+1)
			fmt.Println(styles.Info.Render(nextPageHint))
		}
	} else {
		fmt.Printf("Total: %d template(s)\n", totalCount)
	}

	fmt.Println()
}

var templateShowCmd = &cobra.Command{
	Use:   "show <name|id>",
	Short: "Show template details",
	Long: `Display detailed information about a template including all task definitions.

Examples:
  taskflow template show 1
  taskflow template show "Web Application"`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateShow,
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewTemplateRepository(db)
	ctx := context.Background()

	template, err := lookupTemplate(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	displayTemplateDetails(template, styles)

	return nil
}

func displayTemplateDetails(template *domain.ProjectTemplate, styles *theme.Styles) {
	fmt.Println()
	fmt.Println(styles.Title.Render(fmt.Sprintf("%s (ID: %d)", template.Name, template.ID)))
	fmt.Println()

	if template.Description != "" {
		fmt.Printf("  %s\n", template.Description)
		fmt.Println()
	}

	if template.ProjectDefaults != nil {
		fmt.Println(styles.Subtitle.Render("Project Defaults:"))
		if template.ProjectDefaults.Color != "" {
			fmt.Printf("  Color: %s\n", template.ProjectDefaults.Color)
		}
		if template.ProjectDefaults.Icon != "" {
			fmt.Printf("  Icon: %s\n", template.ProjectDefaults.Icon)
		}
		fmt.Println()
	}

	fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Tasks (%d):", len(template.TaskDefinitions))))
	fmt.Println()

	for i, taskDef := range template.TaskDefinitions {
		fmt.Printf("  %s %s\n", styles.Info.Render(fmt.Sprintf("%d.", i+1)), taskDef.Title)

		if taskDef.Description != "" {
			fmt.Printf("     Description: %s\n", taskDef.Description)
		}

		fmt.Printf("     Priority: %s\n", taskDef.Priority)

		if len(taskDef.Tags) > 0 {
			fmt.Printf("     Tags: %s\n", strings.Join(taskDef.Tags, ", "))
		}

		fmt.Println()
	}

	fmt.Printf("Created: %s\n", template.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Updated: %s\n", template.UpdatedAt.Format("2006-01-02 15:04"))
	fmt.Println()
}

var templateEditCmd = &cobra.Command{
	Use:   "edit <name|id>",
	Short: "Edit a template",
	Long: `Edit an existing template using an interactive wizard.

Examples:
  taskflow template edit 1
  taskflow template edit "Web Application"`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateEdit,
}

func runTemplateEdit(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewTemplateRepository(db)
	ctx := context.Background()

	template, err := lookupTemplate(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Subtitle.Render(fmt.Sprintf("Editing template: %s", template.Name)))
	fmt.Println()

	fmt.Printf("Name [%s]: ", template.Name)
	var newName string
	fmt.Scanln(&newName)
	if newName != "" {
		template.Name = newName
	}

	fmt.Printf("Description [%s]: ", template.Description)
	var newDesc string
	fmt.Scanln(&newDesc)
	if newDesc != "" {
		template.Description = newDesc
	}

	fmt.Printf("Edit project defaults? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) == "y" {
		if template.ProjectDefaults == nil {
			template.ProjectDefaults = &domain.ProjectDefaults{}
		}

		currentColor := ""
		currentIcon := ""
		if template.ProjectDefaults != nil {
			currentColor = template.ProjectDefaults.Color
			currentIcon = template.ProjectDefaults.Icon
		}

		color, err := promptForColorWithCurrent(currentColor, styles)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			color = currentColor
		}

		icon, err := promptForIconWithCurrent(currentIcon, styles)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			icon = currentIcon
		}

		if color != "" || icon != "" {
			template.ProjectDefaults = &domain.ProjectDefaults{
				Color: color,
				Icon:  icon,
			}
		} else {
			template.ProjectDefaults = nil
		}
	}

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("Task Management:"))
	fmt.Println("  1. Add new task")
	fmt.Println("  2. Remove task")
	fmt.Println("  3. Keep tasks as-is")
	fmt.Print("Choice [3]: ")

	var choice string
	fmt.Scanln(&choice)

	if choice == "1" {
		for {
			fmt.Print("Task title (or press Enter to finish): ")
			var title string
			fmt.Scanln(&title)

			if strings.TrimSpace(title) == "" {
				break
			}

			taskDef := domain.NewTaskDefinition(title)

			fmt.Print("  Description (optional): ")
			var desc string
			fmt.Scanln(&desc)
			taskDef.Description = desc

			fmt.Print("  Priority (low/medium/high/urgent) [medium]: ")
			var priority string
			fmt.Scanln(&priority)
			if priority != "" {
				taskDef.Priority = priority
			}

			fmt.Print("  Tags (comma-separated): ")
			var tagsInput string
			fmt.Scanln(&tagsInput)
			if tagsInput != "" {
				tags := strings.Split(tagsInput, ",")
				for i, tag := range tags {
					tags[i] = strings.TrimSpace(tag)
				}
				taskDef.Tags = tags
			}

			template.AddTaskDefinition(taskDef)
			fmt.Println(styles.Success.Render("  ✓ Task added"))
		}
	} else if choice == "2" {
		for i, taskDef := range template.TaskDefinitions {
			fmt.Printf("  %d. %s\n", i+1, taskDef.Title)
		}

		fmt.Print("Enter task number to remove (or 0 to skip): ")
		var taskNum int
		fmt.Scanln(&taskNum)

		if taskNum > 0 && taskNum <= len(template.TaskDefinitions) {
			template.RemoveTaskDefinition(taskNum - 1)
			fmt.Println(styles.Success.Render("  ✓ Task removed"))
		}
	}

	if err := repo.Update(ctx, template); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to update template: %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Template '%s' updated successfully!", template.Name)))
	fmt.Println()

	return nil
}

var deleteTemplateConfirm bool

var templateDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a template",
	Long: `Delete a project template.

Examples:
  taskflow template delete 1
  taskflow template delete "Web Application"
  taskflow template delete 1 --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateDelete,
}

func init() {
	templateDeleteCmd.Flags().BoolVar(&deleteTemplateConfirm, "confirm", false, "Skip confirmation prompt")
}

func runTemplateDelete(cmd *cobra.Command, args []string) error {
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

	repo := sqlite.NewTemplateRepository(db)
	ctx := context.Background()

	template, err := lookupTemplate(ctx, repo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	if !deleteTemplateConfirm {
		fmt.Println()
		fmt.Printf("Delete template '%s' (ID: %d)?\n", template.Name, template.ID)
		fmt.Printf("  - %d task definition(s) will be deleted\n", len(template.TaskDefinitions))
		fmt.Println()
		fmt.Print("Proceed? (y/N): ")

		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) != "y" {
			fmt.Println(styles.Info.Render("Cancelled"))
			return nil
		}
	}

	if err := repo.Delete(ctx, template.ID); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to delete template: %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Template '%s' deleted successfully!", template.Name)))
	fmt.Println()

	return nil
}

var (
	applyTemplateName    string
	applyTemplateParent  string
	applyTemplateColor   string
	applyTemplateIcon    string
	applyNoDefaults      bool
)

var templateApplyCmd = &cobra.Command{
	Use:   "apply <template-name|id> --name <project-name>",
	Short: "Apply a template to create a new project",
	Long: `Create a new project from a template, including all task definitions.

Examples:
  taskflow template apply "Web Application" --name "My Website"
  taskflow template apply 1 --name "Backend API" --parent "Development"
  taskflow template apply 2 --name "Mobile App" --no-defaults --color green`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateApply,
}

func init() {
	templateApplyCmd.Flags().StringVar(&applyTemplateName, "name", "", "Name for the new project (required)")
	templateApplyCmd.Flags().StringVar(&applyTemplateParent, "parent", "", "Parent project name or ID")
	templateApplyCmd.Flags().StringVar(&applyTemplateColor, "color", "", "Override project color")
	templateApplyCmd.Flags().StringVar(&applyTemplateIcon, "icon", "", "Override project icon")
	templateApplyCmd.Flags().BoolVar(&applyNoDefaults, "no-defaults", false, "Don't use template defaults")

	templateApplyCmd.MarkFlagRequired("name")
}

func runTemplateApply(cmd *cobra.Command, args []string) error {
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

	templateRepo := sqlite.NewTemplateRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	taskRepo := sqlite.NewTaskRepository(db)
	ctx := context.Background()

	template, err := lookupTemplate(ctx, templateRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	project := domain.NewProject(applyTemplateName)

	if !applyNoDefaults && template.ProjectDefaults != nil {
		if applyTemplateColor == "" && template.ProjectDefaults.Color != "" {
			project.Color = template.ProjectDefaults.Color
		}
		if applyTemplateIcon == "" && template.ProjectDefaults.Icon != "" {
			project.Icon = template.ProjectDefaults.Icon
		}
	}

	if applyTemplateColor != "" {
		project.Color = applyTemplateColor
	}
	if applyTemplateIcon != "" {
		project.Icon = applyTemplateIcon
	}

	if applyTemplateParent != "" {
		parentID, err := lookupProjectID(ctx, projectRepo, applyTemplateParent)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
		project.ParentID = parentID
	}

	if err := projectRepo.Create(ctx, project); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create project: %v", err)))
		return nil
	}

	tasksCreated := 0
	for _, taskDef := range template.TaskDefinitions {
		task := domain.NewTask(taskDef.Title)
		task.Description = taskDef.Description
		task.Priority = domain.Priority(taskDef.Priority)
		task.Tags = taskDef.Tags
		task.ProjectID = &project.ID

		if err := taskRepo.Create(ctx, task); err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to create task '%s': %v", taskDef.Title, err)))
			continue
		}

		tasksCreated++
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Project '%s' created from template '%s'!", project.Name, template.Name)))
	fmt.Println()
	fmt.Printf("  %s %d\n", styles.Info.Render("Project ID:"), project.ID)
	fmt.Printf("  %s %d tasks created\n", styles.Info.Render("Tasks:"), tasksCreated)
	fmt.Println()

	return nil
}


func lookupTemplate(ctx context.Context, repo *sqlite.TemplateRepository, nameOrID string) (*domain.ProjectTemplate, error) {
	if id, err := strconv.ParseInt(nameOrID, 10, 64); err == nil {
		return repo.GetByID(ctx, id)
	}

	return repo.GetByName(ctx, nameOrID)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func promptForColorWithCurrent(current string, styles *theme.Styles) (string, error) {
	if current != "" {
		fmt.Printf("Color (current: %s, press Enter to keep): ", current)
	} else {
		return promptForColor(styles)
	}

	var color string
	fmt.Scanln(&color)

	if color == "" {
		return current, nil
	}

	return color, nil
}

func promptForIconWithCurrent(current string, styles *theme.Styles) (string, error) {
	if current != "" {
		fmt.Printf("Icon (current: %s, press Enter to keep): ", current)
	} else {
		return promptForIcon(styles)
	}

	var icon string
	fmt.Scanln(&icon)

	if icon == "" {
		return current, nil
	}

	return icon, nil
}
