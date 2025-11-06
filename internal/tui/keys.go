package tui

import "github.com/charmbracelet/bubbles/key"

// keybinds
type keyMap struct {
	// navigation
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding

	// filtering and search
	Filter       key.Binding
	ClearFilters key.Binding
	Search       key.Binding

	// sorting
	Sort      key.Binding
	SortOrder key.Binding

	// pagination
	NextPage key.Binding
	PrevPage key.Binding

	// actions
	New           key.Binding
	Edit          key.Binding
	MarkComplete  key.Binding
	CyclePriority key.Binding
	ToggleStatus  key.Binding
	Delete        key.Binding
	Refresh       key.Binding

	// multi-select
	ToggleMultiSelect key.Binding
	ToggleSelection   key.Binding
	SelectAll         key.Binding
	DeselectAll       key.Binding

	// general
	Quit key.Binding
	Help key.Binding
}

// default keybinds
func defaultKeyMap() keyMap {
	return keyMap{
		// navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view details"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to list"),
		),

		// filtering and search
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "open filters"),
		),
		ClearFilters: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "clear all filters"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search tasks"),
		),

		// sorting
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "cycle sort mode"),
		),
		SortOrder: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle sort order"),
		),

		// pagination
		NextPage: key.NewBinding(
			key.WithKeys("]", "pgdown"),
			key.WithHelp("]", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("[", "pgup"),
			key.WithHelp("[", "previous page"),
		),

		// actions
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new task"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit task"),
		),
		MarkComplete: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "mark complete"),
		),
		CyclePriority: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "cycle priority"),
		),
		ToggleStatus: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "toggle status"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete task"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),

		// multi-select
		ToggleMultiSelect: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "multi-select mode"),
		),
		ToggleSelection: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle selection"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "select all"),
		),
		DeselectAll: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "deselect all"),
		),

		// general
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.New, k.Edit, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Back},
		{k.New, k.Edit, k.Delete, k.Refresh},
		{k.MarkComplete, k.CyclePriority, k.ToggleStatus},
		{k.Filter, k.ClearFilters, k.Search},
		{k.Sort, k.SortOrder, k.NextPage, k.PrevPage},
		{k.ToggleMultiSelect, k.ToggleSelection, k.SelectAll, k.DeselectAll},
		{k.Quit, k.Help},
	}
}
