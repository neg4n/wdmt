package ui

import "github.com/charmbracelet/lipgloss"

var Colors = struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color

	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	TextPrimary   lipgloss.Color
	TextSecondary lipgloss.Color
	TextMuted     lipgloss.Color
	TextDim       lipgloss.Color

	BgPrimary   lipgloss.Color
	BgSecondary lipgloss.Color
	BgMuted     lipgloss.Color

	BorderPrimary   lipgloss.Color
	BorderSecondary lipgloss.Color

	ProgressStart lipgloss.Color
	ProgressEnd   lipgloss.Color
}{

	Primary:   lipgloss.Color("#7C3AED"),
	Secondary: lipgloss.Color("#4ECDC4"),

	Success: lipgloss.Color("#10B981"),
	Warning: lipgloss.Color("#F59E0B"),
	Error:   lipgloss.Color("#EF4444"),
	Info:    lipgloss.Color("#3B82F6"),

	TextPrimary:   lipgloss.Color("#E5E7EB"),
	TextSecondary: lipgloss.Color("#9CA3AF"),
	TextMuted:     lipgloss.Color("#6B7280"),
	TextDim:       lipgloss.Color("#4B5563"),

	BgPrimary:   lipgloss.Color("#1F2937"),
	BgSecondary: lipgloss.Color("#374151"),
	BgMuted:     lipgloss.Color("#065F46"),

	BorderPrimary:   lipgloss.Color("#374151"),
	BorderSecondary: lipgloss.Color("#6B7280"),

	ProgressStart: lipgloss.Color("#FF6B6B"),
	ProgressEnd:   lipgloss.Color("#4ECDC4"),
}

func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.Success).Bold(true)
}

func ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.Error).Bold(true)
}

func WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.Warning).Bold(true)
}

func InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.Info).Bold(true)
}

func HeaderContainerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Colors.BorderPrimary).
		Padding(0, 1).
		MarginBottom(1)
}

func WarningContainerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(Colors.Warning).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Colors.Warning).
		Padding(0, 1).
		MarginBottom(1)
}

func PrimaryTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.TextPrimary)
}

func SecondaryTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.TextSecondary)
}

func MutedTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Colors.TextMuted)
}
