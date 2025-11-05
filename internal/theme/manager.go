package theme

import (
	"errors"
	"fmt"
)

var (
	ErrThemeNotFound = errors.New("theme not found")
)

type Manager struct {
	themes map[string]*Theme
}

func NewManager() *Manager {
	return &Manager{
		themes: GetPredefinedThemes(),
	}
}

// returns theme name
func (m *Manager) GetTheme(name string) (*Theme, error) {
	theme, exists := m.themes[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrThemeNotFound, name)
	}
	return theme, nil
}

// returns all available theme names
func (m *Manager) ListThemes() []string {
	return GetThemeNames()
}

// checks if a theme exists
func (m *Manager) ThemeExists(name string) bool {
	_, exists := m.themes[name]
	return exists
}

// returns default theme
func (m *Manager) GetDefaultTheme() *Theme {
	return DefaultTheme()
}

var globalManager = NewManager()

// returns theme by name using the global manager
func GetTheme(name string) (*Theme, error) {
	return globalManager.GetTheme(name)
}

// returns all available theme names using the global manager
func ListThemes() []string {
	return globalManager.ListThemes()
}

// checks if a theme exists using the global manager
func ThemeExists(name string) bool {
	return globalManager.ThemeExists(name)
}

// returns the default theme using the global manager
func GetDefaultTheme() *Theme {
	return globalManager.GetDefaultTheme()
}
