package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/repository/sqlite"
	"task-management/internal/theme"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "Manage saved views and filters",
	Long: `Manage saved views and filters for quick access to frequently-used filter combinations.

Views allow you to save filter configurations and quickly access them via hot keys (1-9)
or by name. Each view stores a complete filter configuration for tasks.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
	viewCmd.AddCommand(viewSaveCmd)
	viewCmd.AddCommand(viewListCmd)
	viewCmd.AddCommand(viewShowCmd)
	viewCmd.AddCommand(viewApplyCmd)
	viewCmd.AddCommand(viewDeleteCmd)
	viewCmd.AddCommand(viewUpdateCmd)
	viewCmd.AddCommand(viewHotkeyCmd)
	viewCmd.AddCommand(viewFavoriteCmd)
}


var (
	saveViewDescription string
	saveViewFavorite    bool
	saveViewHotKey      int
	saveViewStatus      string
	saveViewPriority    string
	saveViewProject     string
	saveViewTags        []string
	saveViewSearch      string
)

var viewSaveCmd = &cobra.Command{
	Use:   "save [name]",
	Short: "Save current filter configuration as a view",
	Long: `Save a filter configuration as a named view for quick access.

Examples:
  taskflow view save "High Priority Backend" --status pending --priority high --project Backend
  taskflow view save "Due This Week" --search "due:this-week"
  taskflow view save "My Tasks" --favorite --hotkey 1`,
	Args: cobra.MaximumNArgs(1),
	RunE: runViewSave,
}

func init() {
	viewSaveCmd.Flags().StringVarP(&saveViewDescription, "description", "d", "", "View description")
	viewSaveCmd.Flags().BoolVarP(&saveViewFavorite, "favorite", "f", false, "Mark as favorite")
	viewSaveCmd.Flags().IntVarP(&saveViewHotKey, "hotkey", "k", 0, "Hot key (1-9)")
	viewSaveCmd.Flags().StringVar(&saveViewStatus, "status", "", "Filter by status (pending, in_progress, completed, cancelled)")
	viewSaveCmd.Flags().StringVar(&saveViewPriority, "priority", "", "Filter by priority (low, medium, high, urgent)")
	viewSaveCmd.Flags().StringVar(&saveViewProject, "project", "", "Filter by project (name or ID)")
	viewSaveCmd.Flags().StringSliceVar(&saveViewTags, "tags", nil, "Filter by tags (comma-separated)")
	viewSaveCmd.Flags().StringVar(&saveViewSearch, "search", "", "Search query")
}

func runViewSave(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	projectRepo := sqlite.NewProjectRepository(db)
	ctx := context.Background()

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		var err error
		name, err = promptForInput("View name", "")
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
	}

	filter := domain.SavedViewFilter{}

	if saveViewStatus != "" {
		filter.Status = domain.Status(saveViewStatus)
	}
	if saveViewPriority != "" {
		filter.Priority = domain.Priority(saveViewPriority)
	}
	if saveViewProject != "" {
		projectID, err := lookupProjectID(ctx, projectRepo, saveViewProject)
		if err != nil {
			fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
			return nil
		}
		filter.ProjectID = projectID
	}
	if len(saveViewTags) > 0 {
		filter.Tags = saveViewTags
	}
	if saveViewSearch != "" {
		filter.SearchQuery = saveViewSearch
	}

	view := domain.NewSavedView(name)
	view.Description = saveViewDescription
	view.FilterConfig = filter
	view.IsFavorite = saveViewFavorite

	if saveViewHotKey > 0 {
		view.HotKey = &saveViewHotKey
	}

	if err := view.Validate(); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Validation failed: %v", err)))
		return nil
	}

	if err := viewRepo.Create(ctx, view); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to save view: %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ View '%s' saved successfully!", name)))

	if view.HotKey != nil {
		fmt.Println(styles.Info.Render(fmt.Sprintf("  Press %d in TUI to quick-apply", *view.HotKey)))
	}

	fmt.Println()
	fmt.Println(styles.Subtitle.Render("View Details:"))
	fmt.Println(styles.Info.Render(fmt.Sprintf("  ID: %d", view.ID)))
	fmt.Println(styles.Info.Render(fmt.Sprintf("  Filters: %s", view.GetFilterSummary())))
	fmt.Println()

	return nil
}


var (
	listViewFavorite bool
	listViewHotKey   bool
	listViewSearch   string
)

var viewListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved views",
	Long: `List all saved views with options to filter by favorite or hot key status.

Examples:
  taskflow view list                   # Show all views
  taskflow view list --favorite        # Show only favorites
  taskflow view list --hotkey          # Show views with hot keys assigned`,
	RunE: runViewList,
}

func init() {
	viewListCmd.Flags().BoolVar(&listViewFavorite, "favorite", false, "Show only favorite views")
	viewListCmd.Flags().BoolVar(&listViewHotKey, "hotkey", false, "Show only views with hot keys")
	viewListCmd.Flags().StringVar(&listViewSearch, "search", "", "Search views by name or description")
}

func runViewList(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	filter := repository.ViewFilter{
		SortBy:    "name",
		SortOrder: "asc",
	}

	if listViewFavorite {
		favTrue := true
		filter.IsFavorite = &favTrue
	}

	if listViewHotKey {
		filter.HasHotKey = true
	}

	if listViewSearch != "" {
		filter.SearchQuery = listViewSearch
	}

	views, err := viewRepo.List(ctx, filter)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to list views: %v", err)))
		return nil
	}

	if len(views) == 0 {
		fmt.Println()
		fmt.Println(styles.Info.Render("No views found."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Title.Render("Saved Views"))
	fmt.Println()

	headers := []string{
		styles.Header.Render("ID"),
		styles.Header.Render("Name"),
		styles.Header.Render("Filters"),
		styles.Header.Render("Hotkey"),
		styles.Header.Render("Favorite"),
	}
	fmt.Println(strings.Join(headers, " | "))

	fmt.Println(styles.Separator.Render(strings.Repeat("─", 100)))

	for _, view := range views {
		hotKeyDisplay := "-"
		if view.HotKey != nil {
			hotKeyDisplay = fmt.Sprintf("[%d]", *view.HotKey)
		}

		favoriteDisplay := ""
		if view.IsFavorite {
			favoriteDisplay = "★"
		}

		row := []string{
			fmt.Sprintf("%d", view.ID),
			view.Name,
			view.GetFilterSummary(),
			hotKeyDisplay,
			favoriteDisplay,
		}
		fmt.Println(strings.Join(row, " | "))
	}

	fmt.Println()
	fmt.Printf("Total: %d view(s)\n", len(views))
	fmt.Println()

	return nil
}


var viewShowCmd = &cobra.Command{
	Use:   "show <name|id>",
	Short: "Show detailed view information",
	Long: `Show detailed information about a saved view including its filter configuration.

Examples:
  taskflow view show "My View"
  taskflow view show 1`,
	Args: cobra.ExactArgs(1),
	RunE: runViewShow,
}

func runViewShow(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	viewID, err := lookupViewID(ctx, viewRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	view, err := viewRepo.GetByID(ctx, *viewID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Title.Render(fmt.Sprintf("View: %s (ID: %d)", view.Name, view.ID)))
	fmt.Println()

	if view.Description != "" {
		fmt.Printf("%s %s\n\n", styles.Subtitle.Render("Description:"), view.Description)
	}

	fmt.Printf("%s\n", styles.Subtitle.Render("Filter Configuration:"))
	fmt.Printf("  Status:        %s\n", displayValue(string(view.FilterConfig.Status)))
	fmt.Printf("  Priority:      %s\n", displayValue(string(view.FilterConfig.Priority)))
	fmt.Printf("  Tags:          %s\n", displayTags(view.FilterConfig.Tags))
	fmt.Printf("  Search:        %s\n", displayValue(view.FilterConfig.SearchQuery))
	fmt.Println()

	fmt.Printf("%s\n", styles.Subtitle.Render("Properties:"))
	fmt.Printf("  Favorite:      %s\n", displayBool(view.IsFavorite))
	fmt.Printf("  Hot Key:       %s\n", displayHotKey(view.HotKey))
	fmt.Println()

	fmt.Printf("%s\n", styles.Subtitle.Render("Timestamps:"))
	fmt.Printf("  Created:       %s\n", view.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("  Updated:       %s\n", view.UpdatedAt.Format("2006-01-02 15:04"))
	if view.LastAccessed != nil {
		fmt.Printf("  Last Accessed: %s\n", view.LastAccessed.Format("2006-01-02 15:04"))
	}
	fmt.Println()

	return nil
}


var viewApplyCmd = &cobra.Command{
	Use:   "apply <name|id|hotkey>",
	Short: "Apply a saved view (launch TUI with filter)",
	Long: `Apply a saved view by loading its filter configuration and launching the task list.

Examples:
  taskflow view apply "My View"
  taskflow view apply 1
  taskflow view apply 5    # Using hot key`,
	Args: cobra.ExactArgs(1),
	RunE: runViewApply,
}

func runViewApply(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := sqlite.NewDB(sqlite.Config{Path: cfg.DBPath})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	viewID, err := lookupViewID(ctx, viewRepo, args[0])
	if err != nil {
		if hotKey := parseHotKey(args[0]); hotKey > 0 {
			view, err := viewRepo.GetByHotKey(ctx, hotKey)
			if err == nil {
				viewID = &view.ID
			}
		}
	}

	if viewID == nil {
		fmt.Printf("View not found: %s\n", args[0])
		return nil
	}

	view, err := viewRepo.GetByID(ctx, *viewID)
	if err != nil {
		fmt.Printf("✗ Failed to load view: %v\n", err)
		return nil
	}

	_ = viewRepo.RecordViewAccess(ctx, view.ID)

	taskFilter := repository.TaskFilter{
		Status:       view.FilterConfig.Status,
		Priority:     view.FilterConfig.Priority,
		ProjectID:    view.FilterConfig.ProjectID,
		Tags:         view.FilterConfig.Tags,
		SearchQuery:  view.FilterConfig.SearchQuery,
		SearchMode:   view.FilterConfig.SearchMode,
		SortBy:       view.FilterConfig.SortBy,
		SortOrder:    view.FilterConfig.SortOrder,
		DueDateFrom:  view.FilterConfig.DueDateFrom,
		DueDateTo:    view.FilterConfig.DueDateTo,
	}

	listStatus = string(taskFilter.Status)
	listPriority = string(taskFilter.Priority)
	listTags = taskFilter.Tags
	listSearch = taskFilter.SearchQuery

	return runList(cmd, []string{})
}


var viewDeleteConfirm bool

var viewDeleteCmd = &cobra.Command{
	Use:   "delete <name|id>",
	Short: "Delete a saved view",
	Long: `Delete a saved view. Requires confirmation unless --confirm is provided.

Examples:
  taskflow view delete "Old View"
  taskflow view delete 1
  taskflow view delete "Temp" --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runViewDelete,
}

func init() {
	viewDeleteCmd.Flags().BoolVar(&viewDeleteConfirm, "confirm", false, "Skip confirmation prompt")
}

func runViewDelete(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	viewID, err := lookupViewID(ctx, viewRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	view, err := viewRepo.GetByID(ctx, *viewID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	if !viewDeleteConfirm {
		fmt.Println()
		fmt.Printf("Delete view '%s' (ID: %d)?\n", view.Name, view.ID)
		fmt.Print("Proceed? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := viewRepo.Delete(ctx, view.ID); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to delete view: %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ View '%s' deleted successfully!", view.Name)))
	fmt.Println()

	return nil
}


var (
	updateViewName        string
	updateViewDescription string
	updateViewFavorite    *bool
	updateViewHotKey      int
)

var viewUpdateCmd = &cobra.Command{
	Use:   "update <name|id>",
	Short: "Update view properties",
	Long: `Update properties of an existing saved view.

Examples:
  taskflow view update "My View" --name "New Name"
  taskflow view update 1 --description "Updated description"
  taskflow view update "View" --favorite --hotkey 5`,
	Args: cobra.ExactArgs(1),
	RunE: runViewUpdate,
}

func init() {
	viewUpdateCmd.Flags().StringVar(&updateViewName, "name", "", "New view name")
	viewUpdateCmd.Flags().StringVar(&updateViewDescription, "description", "", "New description")
	viewUpdateCmd.Flags().IntVar(&updateViewHotKey, "hotkey", 0, "New hot key (1-9, or 0 to clear)")

	var favStr string
	viewUpdateCmd.Flags().StringVar(&favStr, "favorite", "", "Set favorite (true/false)")
}

func runViewUpdate(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	viewID, err := lookupViewID(ctx, viewRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	view, err := viewRepo.GetByID(ctx, *viewID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	if updateViewName != "" {
		view.Name = updateViewName
	}
	if updateViewDescription != "" {
		view.Description = updateViewDescription
	}
	if cmd.Flags().Changed("favorite") {
		favStr, _ := cmd.Flags().GetString("favorite")
		if favStr == "true" {
			view.IsFavorite = true
		} else if favStr == "false" {
			view.IsFavorite = false
		}
	}
	if updateViewHotKey > 0 {
		view.HotKey = &updateViewHotKey
	} else if cmd.Flags().Changed("hotkey") && updateViewHotKey == 0 {
		view.HotKey = nil
	}

	if err := view.Validate(); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Validation failed: %v", err)))
		return nil
	}

	if err := viewRepo.Update(ctx, view); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to update view: %v", err)))
		return nil
	}

	fmt.Println()
	fmt.Println(styles.Success.Render(fmt.Sprintf("✓ View '%s' updated successfully!", view.Name)))
	fmt.Println()

	return nil
}


var viewHotkeyCmd = &cobra.Command{
	Use:   "hotkey <name|id> <1-9|clear>",
	Short: "Assign or clear hot key for a view",
	Long: `Assign a hot key (1-9) to a view for quick access, or clear an existing hot key.

Examples:
  taskflow view hotkey "My View" 5
  taskflow view hotkey 1 3
  taskflow view hotkey "View" clear`,
	Args: cobra.ExactArgs(2),
	RunE: runViewHotkey,
}

func runViewHotkey(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	viewID, err := lookupViewID(ctx, viewRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	view, err := viewRepo.GetByID(ctx, *viewID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	var newHotKey *int
	if args[1] != "clear" {
		hotKey := parseHotKey(args[1])
		if hotKey <= 0 || hotKey > 9 {
			fmt.Println(styles.Error.Render("✗ Hot key must be 1-9 or 'clear'"))
			return nil
		}
		newHotKey = &hotKey
	}

	if err := viewRepo.SetHotKey(ctx, view.ID, newHotKey); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to set hot key: %v", err)))
		return nil
	}

	fmt.Println()
	if newHotKey != nil {
		fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Hot key %d assigned to view '%s'", *newHotKey, view.Name)))
	} else {
		fmt.Println(styles.Success.Render(fmt.Sprintf("✓ Hot key cleared for view '%s'", view.Name)))
	}
	fmt.Println()

	return nil
}


var viewFavoriteCmd = &cobra.Command{
	Use:   "favorite <name|id>",
	Short: "Toggle favorite status for a view",
	Long: `Toggle the favorite status for a saved view.

Examples:
  taskflow view favorite "My View"
  taskflow view favorite 1`,
	Args: cobra.ExactArgs(1),
	RunE: runViewFavorite,
}

func runViewFavorite(cmd *cobra.Command, args []string) error {
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

	viewRepo := sqlite.NewViewRepository(db)
	ctx := context.Background()

	viewID, err := lookupViewID(ctx, viewRepo, args[0])
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	view, err := viewRepo.GetByID(ctx, *viewID)
	if err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ %v", err)))
		return nil
	}

	newFavorite := !view.IsFavorite
	if err := viewRepo.SetFavorite(ctx, view.ID, newFavorite); err != nil {
		fmt.Println(styles.Error.Render(fmt.Sprintf("✗ Failed to update favorite: %v", err)))
		return nil
	}

	fmt.Println()
	if newFavorite {
		fmt.Println(styles.Success.Render(fmt.Sprintf("✓ View '%s' marked as favorite (★)", view.Name)))
	} else {
		fmt.Println(styles.Success.Render(fmt.Sprintf("✓ View '%s' unmarked as favorite", view.Name)))
	}
	fmt.Println()

	return nil
}


func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func displayTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ", ")
}

func displayBool(value bool) string {
	if value {
		return "Yes (★)"
	}
	return "No"
}

func displayHotKey(hotKey *int) string {
	if hotKey == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *hotKey)
}

func parseHotKey(s string) int {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
		return i
	}
	return 0
}
