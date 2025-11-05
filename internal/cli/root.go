package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"task-management/internal/config"
	"task-management/internal/theme"
	"task-management/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "taskflow",
	Short: "TaskFlow - Your advanced CLI task management system",
	Long: `TaskFlow is an advanced command-line task management system that combines
the simplicity of traditional todo lists with powerful features inspired by
modern project management tools.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// check if we need to run initial setup
		return checkAndRunSetup()
	},
	Run: func(cmd *cobra.Command, args []string) {
		displayWelcome()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func displayWelcome() {
	// load theme
	cfg, err := config.LoadConfig()
	if err != nil {
		// fallback to default
		cfg = config.GetDefaultConfig()
	}

	themeName := cfg.ThemeName
	if themeName == "" {
		themeName = "default"
	}

	themeObj, err := theme.GetTheme(themeName)
	if err != nil {
		themeObj = theme.GetDefaultTheme()
	}

	styles := theme.NewStyles(themeObj)

	title := styles.Title.Render(`
		------------------------------------------------------

		                T A S K F L O W

		------------------------------------------------------
	`)
	subtitle := styles.Subtitle.Render("Manage your days like never before <3")

	fmt.Println()
	fmt.Println(title)
	fmt.Println(subtitle)
	fmt.Println()
	fmt.Println("Run 'taskflow --help' to see available commands.")
	fmt.Println()
}

// checks if initial setup is needed and runs it
func checkAndRunSetup() error {
	// load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// if theme not set then run initial setup
	if cfg.ThemeName == "" {
		fmt.Println()
		fmt.Println("Welcome to TaskFlow! Let's set up your theme.")
		fmt.Println()

		model := tui.NewSetupModel()
		p := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("failed to run setup: %w", err)
		}

		// read config to see which theme was selected
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config after setup: %w", err)
		}

		fmt.Println()
		if cfg.ThemeName != "" {
			fmt.Printf("✓ Theme configured: '%s'\n", cfg.ThemeName)
		} else {
			fmt.Println("Theme configuration complete!")
		}
		fmt.Println()
	}

	return nil
}
