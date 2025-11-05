package cli

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/theme"
	"task-management/internal/tui"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage application theme",
	Long: `Manage application theme settings.

Run without arguments to launch the interactive theme selector TUI.
Use subcommands for direct theme management.

Examples:
  taskflow theme              # Launch interactive TUI
  taskflow theme set dracula  # Set theme directly
  taskflow theme list         # List available themes
  taskflow theme show         # Show current theme`,
	RunE: runThemeTUI,
}

var themeSetCmd = &cobra.Command{
	Use:   "set [theme-name]",
	Short: "Set application theme",
	Long: `Set the application theme.

Available themes:
  - default
  - dark
  - light
  - dracula
  - nord
  - gruvbox

Examples:
  taskflow theme set dracula
  taskflow theme set nord`,
	Args: cobra.ExactArgs(1),
	RunE: runThemeSet,
}

var themeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available themes",
	Long:  `List all available themes.`,
	RunE:  runThemeList,
}

var themeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current theme",
	Long:  `Display the currently selected theme and its color palette.`,
	RunE:  runThemeShow,
}

func init() {
	rootCmd.AddCommand(themeCmd)
	themeCmd.AddCommand(themeSetCmd)
	themeCmd.AddCommand(themeListCmd)
	themeCmd.AddCommand(themeShowCmd)
}

// launches theme selector
func runThemeTUI(cmd *cobra.Command, args []string) error {
	model := tui.NewSetupModel()
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run theme TUI: %w", err)
	}

	// read config to see which theme was selected
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.ThemeName != "" {
		fmt.Println()
		fmt.Printf("✓ Theme set to '%s'\n", cfg.ThemeName)
		fmt.Println()
	}

	return nil
}

// sets the theme directly
func runThemeSet(cmd *cobra.Command, args []string) error {
	themeName := args[0]

	if !theme.ThemeExists(themeName) {
		return fmt.Errorf("theme '%s' not found. Run 'taskflow theme list' to see available themes", themeName)
	}

	if err := config.UpdateTheme(themeName); err != nil {
		return fmt.Errorf("failed to update theme: %w", err)
	}

	fmt.Printf("✓ Theme set to '%s'\n", themeName)
	return nil
}

// lists all available themes
func runThemeList(cmd *cobra.Command, args []string) error {
	// load current theme
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.GetDefaultConfig()
	}

	themeName := cfg.ThemeName
	if themeName == "" {
		themeName = "default"
	}

	currentTheme, err := theme.GetTheme(themeName)
	if err != nil {
		currentTheme = theme.GetDefaultTheme()
	}

	styles := theme.NewStyles(currentTheme)

	// get all themes
	themes := theme.ListThemes()

	fmt.Println()
	fmt.Println(styles.Header.Render(" Available Themes "))
	fmt.Println()

	for _, name := range themes {
		prefix := "  "
		if name == themeName {
			prefix = "▶ "
			name = styles.Success.Render(name + " (current)")
		}
		fmt.Printf("%s%s\n", prefix, name)
	}

	fmt.Println()
	return nil
}

// displays current theme details
func runThemeShow(cmd *cobra.Command, args []string) error {
	// load current config and theme
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

	fmt.Println()
	fmt.Println(styles.Header.Render(fmt.Sprintf(" Current Theme: %s ", themeName)))
	fmt.Println()

	fmt.Println(styles.Info.Render("Color Palette:"))
	fmt.Println()

	colors := map[string]string{
		"Primary":     themeObj.Primary,
		"Success":     themeObj.Success,
		"Error":       themeObj.Error,
		"Warning":     themeObj.Warning,
		"Info":        themeObj.Info,
		"Text":        themeObj.TextPrimary,
		"Border":      themeObj.BorderColor,
	}

	for name, color := range colors {
		colorSample := styles.Cell.Copy().
			Background(lipgloss.Color(color)).
			Foreground(lipgloss.Color(color)).
			Render("  ████  ")
		fmt.Printf("  %-12s %s %s\n", name+":", colorSample, color)
	}

	fmt.Println()
	return nil
}
