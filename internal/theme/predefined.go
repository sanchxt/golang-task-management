package theme

func GetPredefinedThemes() map[string]*Theme {
	return map[string]*Theme{
		"default":  DefaultTheme(),
		"dark":     DarkTheme(),
		"light":    LightTheme(),
		"dracula":  DraculaTheme(),
		"nord":     NordTheme(),
		"gruvbox":  GruvboxTheme(),
	}
}

func GetThemeNames() []string {
	return []string{
		"default",
		"dark",
		"light",
		"dracula",
		"nord",
		"gruvbox",
	}
}

func DefaultTheme() *Theme {
	return &Theme{
		Name: "default",

		// semantic
		Primary:   "#7D56F4",
		Secondary: "#8aa4eb",
		Success:   "#04B575",
		Error:     "#FF0000",
		Warning:   "#FF8800",
		Info:      "#0088FF",

		// text
		TextPrimary:   "#FAFAFA",
		TextSecondary: "#888888",
		TextMuted:     "#6C6C6C",

		// background
		BgPrimary:   "#000000",
		BgSecondary: "#1a1a1a",

		// priority
		PriorityUrgent: "#FF0000",
		PriorityHigh:   "#FF8800",
		PriorityMedium: "#0088FF",
		PriorityLow:    "#888888",

		// status
		StatusCompleted:  "#04B575",
		StatusInProgress: "#FFD700",
		StatusPending:    "#888888",
		StatusCancelled:  "#FF0000",

		// UI element
		BorderColor:   "#7D56F4",
		SelectedBg:    "#7D56F4",
		SelectedFg:    "#FAFAFA",
		HeaderBg:      "#7D56F4",
		HeaderFg:      "#FAFAFA",
		Separator:     "#444444",
		HelpText:      "#888888",
		SubtitleText:  "#6C6C6C",
		TableSelected: "57",
	}
}

func DarkTheme() *Theme {
	return &Theme{
		Name: "dark",

		// semantic
		Primary:   "#BB9AF7",
		Secondary: "#7AA2F7",
		Success:   "#9ECE6A",
		Error:     "#F7768E",
		Warning:   "#E0AF68",
		Info:      "#7DCFFF",

		// text
		TextPrimary:   "#C0CAF5",
		TextSecondary: "#9AA5CE",
		TextMuted:     "#565F89",

		// background
		BgPrimary:   "#1A1B26",
		BgSecondary: "#24283B",

		// priority
		PriorityUrgent: "#F7768E",
		PriorityHigh:   "#FF9E64",
		PriorityMedium: "#7AA2F7",
		PriorityLow:    "#565F89",

		// status
		StatusCompleted:  "#9ECE6A",
		StatusInProgress: "#E0AF68",
		StatusPending:    "#565F89",
		StatusCancelled:  "#F7768E",

		// UI element
		BorderColor:   "#BB9AF7",
		SelectedBg:    "#BB9AF7",
		SelectedFg:    "#1A1B26",
		HeaderBg:      "#BB9AF7",
		HeaderFg:      "#1A1B26",
		Separator:     "#3B4261",
		HelpText:      "#565F89",
		SubtitleText:  "#565F89",
		TableSelected: "55",
	}
}

func LightTheme() *Theme {
	return &Theme{
		Name: "light",

		// semantic
		Primary:   "#5B3CC4",
		Secondary: "#2563EB",
		Success:   "#059669",
		Error:     "#DC2626",
		Warning:   "#D97706",
		Info:      "#0284C7",

		// text
		TextPrimary:   "#1F2937",
		TextSecondary: "#6B7280",
		TextMuted:     "#9CA3AF",

		// background
		BgPrimary:   "#FFFFFF",
		BgSecondary: "#F3F4F6",

		// priority
		PriorityUrgent: "#DC2626",
		PriorityHigh:   "#EA580C",
		PriorityMedium: "#2563EB",
		PriorityLow:    "#6B7280",

		// status
		StatusCompleted:  "#059669",
		StatusInProgress: "#D97706",
		StatusPending:    "#6B7280",
		StatusCancelled:  "#DC2626",

		// UI element
		BorderColor:   "#5B3CC4",
		SelectedBg:    "#5B3CC4",
		SelectedFg:    "#FFFFFF",
		HeaderBg:      "#5B3CC4",
		HeaderFg:      "#FFFFFF",
		Separator:     "#D1D5DB",
		HelpText:      "#6B7280",
		SubtitleText:  "#9CA3AF",
		TableSelected: "57",
	}
}

func DraculaTheme() *Theme {
	return &Theme{
		Name: "dracula",

		// semantic
		Primary:   "#BD93F9",
		Secondary: "#8BE9FD",
		Success:   "#50FA7B",
		Error:     "#FF5555",
		Warning:   "#FFB86C",
		Info:      "#8BE9FD",

		// text
		TextPrimary:   "#F8F8F2",
		TextSecondary: "#6272A4",
		TextMuted:     "#44475A",

		// background
		BgPrimary:   "#282A36",
		BgSecondary: "#44475A",

		// priority
		PriorityUrgent: "#FF5555",
		PriorityHigh:   "#FFB86C",
		PriorityMedium: "#BD93F9",
		PriorityLow:    "#6272A4",

		// status
		StatusCompleted:  "#50FA7B",
		StatusInProgress: "#F1FA8C",
		StatusPending:    "#6272A4",
		StatusCancelled:  "#FF5555",

		// UI element
		BorderColor:   "#BD93F9",
		SelectedBg:    "#BD93F9",
		SelectedFg:    "#282A36",
		HeaderBg:      "#BD93F9",
		HeaderFg:      "#282A36",
		Separator:     "#44475A",
		HelpText:      "#6272A4",
		SubtitleText:  "#6272A4",
		TableSelected: "141",
	}
}

func NordTheme() *Theme {
	return &Theme{
		Name: "nord",

		// semantic
		Primary:   "#88C0D0",
		Secondary: "#81A1C1",
		Success:   "#A3BE8C",
		Error:     "#BF616A",
		Warning:   "#EBCB8B",
		Info:      "#5E81AC",

		// text
		TextPrimary:   "#ECEFF4",
		TextSecondary: "#D8DEE9",
		TextMuted:     "#4C566A",

		// background
		BgPrimary:   "#2E3440",
		BgSecondary: "#3B4252",

		// priority
		PriorityUrgent: "#BF616A",
		PriorityHigh:   "#D08770",
		PriorityMedium: "#81A1C1",
		PriorityLow:    "#4C566A",

		// status
		StatusCompleted:  "#A3BE8C",
		StatusInProgress: "#EBCB8B",
		StatusPending:    "#4C566A",
		StatusCancelled:  "#BF616A",

		// UI element
		BorderColor:   "#88C0D0",
		SelectedBg:    "#88C0D0",
		SelectedFg:    "#2E3440",
		HeaderBg:      "#88C0D0",
		HeaderFg:      "#2E3440",
		Separator:     "#434C5E",
		HelpText:      "#4C566A",
		SubtitleText:  "#4C566A",
		TableSelected: "73",
	}
}

func GruvboxTheme() *Theme {
	return &Theme{
		Name: "gruvbox",

		// semantic
		Primary:   "#D3869B",
		Secondary: "#83A598",
		Success:   "#B8BB26",
		Error:     "#FB4934",
		Warning:   "#FABD2F",
		Info:      "#83A598",

		// text
		TextPrimary:   "#EBDBB2",
		TextSecondary: "#A89984",
		TextMuted:     "#665C54",

		// background
		BgPrimary:   "#282828",
		BgSecondary: "#3C3836",

		// priority
		PriorityUrgent: "#FB4934",
		PriorityHigh:   "#FE8019",
		PriorityMedium: "#83A598",
		PriorityLow:    "#928374",

		// status
		StatusCompleted:  "#B8BB26",
		StatusInProgress: "#FABD2F",
		StatusPending:    "#928374",
		StatusCancelled:  "#FB4934",

		// UI element
		BorderColor:   "#D3869B",
		SelectedBg:    "#D3869B",
		SelectedFg:    "#282828",
		HeaderBg:      "#D3869B",
		HeaderFg:      "#282828",
		Separator:     "#504945",
		HelpText:      "#928374",
		SubtitleText:  "#665C54",
		TableSelected: "175",
	}
}
