package theme

import (
	"task-management/internal/domain"

	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	// cli
	Success   lipgloss.Style
	Error     lipgloss.Style
	Info      lipgloss.Style
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Header    lipgloss.Style
	Cell      lipgloss.Style
	Separator lipgloss.Style

	// priority
	UrgentRow lipgloss.Style
	HighRow   lipgloss.Style
	MediumRow lipgloss.Style
	LowRow    lipgloss.Style

	// tui
	TUITitle          lipgloss.Style
	TUISubtitle       lipgloss.Style
	TUIHelp           lipgloss.Style
	DetailContainer   lipgloss.Style
	DetailLabel       lipgloss.Style
	DetailValue       lipgloss.Style
	UrgentText        lipgloss.Style
	HighText          lipgloss.Style
	MediumText        lipgloss.Style
	LowText           lipgloss.Style
	CompletedText     lipgloss.Style
	InProgressText    lipgloss.Style
	PendingText       lipgloss.Style
	CancelledText     lipgloss.Style
}

// creates all styles based on the given theme
func NewStyles(t *Theme) *Styles {
	return &Styles{
		// cli
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Success)).
			Bold(true),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)).
			Bold(true),

		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Secondary)).
			PaddingTop(1).
			PaddingBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SubtitleText)).
			Italic(true),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.HeaderFg)).
			Background(lipgloss.Color(t.HeaderBg)).
			PaddingLeft(1).
			PaddingRight(1),

		Cell: lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1),

		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Separator)),

		// priority row
		UrgentRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityUrgent)),

		HighRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityHigh)),

		MediumRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityMedium)),

		LowRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityLow)),

		// tui
		TUITitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.TextPrimary)).
			Background(lipgloss.Color(t.HeaderBg)).
			Padding(0, 1),

		TUISubtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextSecondary)),

		TUIHelp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.HelpText)),

		DetailContainer: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderColor)).
			Padding(1, 2),

		DetailLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)).
			Bold(true),

		DetailValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextPrimary)),

		// priority
		UrgentText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityUrgent)).
			Bold(true),

		HighText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityHigh)),

		MediumText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityMedium)),

		LowText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.PriorityLow)),

		// status
		CompletedText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusCompleted)),

		InProgressText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusInProgress)),

		PendingText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusPending)),

		CancelledText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.StatusCancelled)),
	}
}

func (s *Styles) GetPriorityStyle(priority domain.Priority) lipgloss.Style {
	switch priority {
	case domain.PriorityUrgent:
		return s.UrgentRow
	case domain.PriorityHigh:
		return s.HighRow
	case domain.PriorityMedium:
		return s.MediumRow
	case domain.PriorityLow:
		return s.LowRow
	default:
		return s.Cell
	}
}

func (s *Styles) GetPriorityTextStyle(priority domain.Priority) lipgloss.Style {
	switch priority {
	case domain.PriorityUrgent:
		return s.UrgentText
	case domain.PriorityHigh:
		return s.HighText
	case domain.PriorityMedium:
		return s.MediumText
	case domain.PriorityLow:
		return s.LowText
	default:
		return s.DetailValue
	}
}

func (s *Styles) GetStatusStyle(status domain.Status) lipgloss.Style {
	switch status {
	case domain.StatusCompleted:
		return s.CompletedText
	case domain.StatusInProgress:
		return s.InProgressText
	case domain.StatusPending:
		return s.PendingText
	case domain.StatusCancelled:
		return s.CancelledText
	default:
		return s.DetailValue
	}
}
