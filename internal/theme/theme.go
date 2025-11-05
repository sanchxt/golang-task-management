package theme

type Theme struct {
	Name string

	// semantic
	Primary   string
	Secondary string
	Success   string
	Error     string
	Warning   string
	Info      string

	// text
	TextPrimary   string
	TextSecondary string
	TextMuted     string

	// background
	BgPrimary   string
	BgSecondary string

	// priority
	PriorityUrgent string
	PriorityHigh   string
	PriorityMedium string
	PriorityLow    string

	// status
	StatusCompleted  string
	StatusInProgress string
	StatusPending    string
	StatusCancelled  string

	// UI element
	BorderColor   string
	SelectedBg    string
	SelectedFg    string
	HeaderBg      string
	HeaderFg      string
	Separator     string
	HelpText      string
	SubtitleText  string
	TableSelected string
}
