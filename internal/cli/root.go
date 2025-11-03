package cli

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	// Styles for the welcome message
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C6C6C")).
			Italic(true)
)

var rootCmd = &cobra.Command{
	Use:   "taskflow",
	Short: "TaskFlow - Your advanced CLI task management system",
	Long: `TaskFlow is an advanced command-line task management system that combines
the simplicity of traditional todo lists with powerful features inspired by
modern project management tools.`,
	Run: func(cmd *cobra.Command, args []string) {
		displayWelcome()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// displayWelcome shows the welcome message
func displayWelcome() {
	title := titleStyle.Render("Welcome to Your Favorite Task Management Tool")
	subtitle := subtitleStyle.Render("✨ TaskFlow - Manage tasks like a pro")

	fmt.Println()
	fmt.Println(title)
	fmt.Println(subtitle)
	fmt.Println()
	fmt.Println("Run 'taskflow --help' to see available commands.")
	fmt.Println()
}
