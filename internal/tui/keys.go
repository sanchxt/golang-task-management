package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Back   key.Binding

	Filter       key.Binding
	ClearFilters key.Binding
	Search       key.Binding

	Sort      key.Binding
	SortOrder key.Binding

	NextPage key.Binding
	PrevPage key.Binding

	New           key.Binding
	Edit          key.Binding
	MarkComplete  key.Binding
	CyclePriority key.Binding
	ToggleStatus  key.Binding
	Delete        key.Binding
	Refresh       key.Binding

	ToggleMultiSelect key.Binding
	ToggleSelection   key.Binding
	SelectAll         key.Binding
	DeselectAll       key.Binding

	ToggleProjects   key.Binding
	ExpandProject    key.Binding
	CollapseProject  key.Binding
	ViewProject      key.Binding
	NewProject       key.Binding
	EditProject      key.Binding
	DeleteProject    key.Binding
	ArchiveProject   key.Binding
	ProjectPicker    key.Binding
	FilterByProject  key.Binding
	ViewNotes        key.Binding

	ViewPicker      key.Binding
	FavoriteViews   key.Binding
	QuickAccess1    key.Binding
	QuickAccess2    key.Binding
	QuickAccess3    key.Binding
	QuickAccess4    key.Binding
	QuickAccess5    key.Binding
	QuickAccess6    key.Binding
	QuickAccess7    key.Binding
	QuickAccess8    key.Binding
	QuickAccess9    key.Binding

	Quit key.Binding
	Help key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
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

		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "cycle sort mode"),
		),
		SortOrder: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "toggle sort order"),
		),

		NextPage: key.NewBinding(
			key.WithKeys("]", "pgdown"),
			key.WithHelp("]", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("[", "pgup"),
			key.WithHelp("[", "previous page"),
		),

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

		ToggleProjects: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle projects view"),
		),
		ExpandProject: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand project"),
		),
		CollapseProject: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse project"),
		),
		ViewProject: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select project"),
		),
		NewProject: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "new project"),
		),
		EditProject: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "edit project"),
		),
		DeleteProject: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "delete project"),
		),
		ArchiveProject: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "archive/unarchive"),
		),
		ProjectPicker: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "project picker"),
		),
		FilterByProject: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "filter by project"),
		),
		ViewNotes: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "view/edit notes"),
		),

		ViewPicker: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "view picker"),
		),
		FavoriteViews: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "favorite views"),
		),
		QuickAccess1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "quick access view 1"),
		),
		QuickAccess2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "quick access view 2"),
		),
		QuickAccess3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "quick access view 3"),
		),
		QuickAccess4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "quick access view 4"),
		),
		QuickAccess5: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "quick access view 5"),
		),
		QuickAccess6: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "quick access view 6"),
		),
		QuickAccess7: key.NewBinding(
			key.WithKeys("7"),
			key.WithHelp("7", "quick access view 7"),
		),
		QuickAccess8: key.NewBinding(
			key.WithKeys("8"),
			key.WithHelp("8", "quick access view 8"),
		),
		QuickAccess9: key.NewBinding(
			key.WithKeys("9"),
			key.WithHelp("9", "quick access view 9"),
		),

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
		{k.ToggleProjects, k.ViewProject, k.ProjectPicker},
		{k.ViewPicker, k.FavoriteViews},
		{k.QuickAccess1, k.QuickAccess2, k.QuickAccess3, k.QuickAccess4},
		{k.QuickAccess5, k.QuickAccess6, k.QuickAccess7, k.QuickAccess8},
		{k.QuickAccess9, k.Quit, k.Help},
	}
}
