package tui

import (
	"task-management/internal/domain"
	"task-management/internal/theme"
)

// Note: All styles are now managed through theme.Styles
// This file is kept for backward compatibility and utility functions

// Helper functions that delegate to theme.Styles methods
func getPriorityStyleFromTheme(styles *theme.Styles, priority domain.Priority) theme.Styles {
	// Return the appropriate style from the theme
	return *styles
}

func getStatusStyleFromTheme(styles *theme.Styles, status domain.Status) theme.Styles {
	// Return the appropriate style from the theme
	return *styles
}
