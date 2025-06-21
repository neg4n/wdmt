package ui

import "github.com/charmbracelet/lipgloss"

// Color palette for consistent theming across the application
var Colors = struct {
	// Primary brand colors
	Primary    lipgloss.Color // Purple - main accent
	Secondary  lipgloss.Color // Teal - secondary accent
	
	// Status colors
	Success    lipgloss.Color // Green - success states
	Warning    lipgloss.Color // Yellow/Orange - warnings
	Error      lipgloss.Color // Red - errors
	Info       lipgloss.Color // Blue - informational
	
	// Text colors
	TextPrimary   lipgloss.Color // Primary text
	TextSecondary lipgloss.Color // Secondary text
	TextMuted     lipgloss.Color // Muted text
	TextDim       lipgloss.Color // Dimmed text
	
	// Background colors
	BgPrimary   lipgloss.Color // Primary background
	BgSecondary lipgloss.Color // Secondary background
	BgMuted     lipgloss.Color // Muted background
	
	// Border colors
	BorderPrimary   lipgloss.Color // Primary borders
	BorderSecondary lipgloss.Color // Secondary borders
	
	// Progress bar gradient colors
	ProgressStart lipgloss.Color // Progress bar start color
	ProgressEnd   lipgloss.Color // Progress bar end color
}{
	// Primary brand colors
	Primary:   lipgloss.Color("#7C3AED"), // Purple
	Secondary: lipgloss.Color("#4ECDC4"), // Teal
	
	// Status colors
	Success: lipgloss.Color("#10B981"), // Green
	Warning: lipgloss.Color("#F59E0B"), // Yellow/Orange
	Error:   lipgloss.Color("#EF4444"), // Red
	Info:    lipgloss.Color("#3B82F6"), // Blue
	
	// Text colors
	TextPrimary:   lipgloss.Color("#E5E7EB"), // Light gray
	TextSecondary: lipgloss.Color("#9CA3AF"), // Medium gray
	TextMuted:     lipgloss.Color("#6B7280"), // Muted gray
	TextDim:       lipgloss.Color("#4B5563"), // Dim gray
	
	// Background colors
	BgPrimary:   lipgloss.Color("#1F2937"), // Dark gray
	BgSecondary: lipgloss.Color("#374151"), // Medium dark gray
	BgMuted:     lipgloss.Color("#065F46"), // Dark green
	
	// Border colors
	BorderPrimary:   lipgloss.Color("#374151"), // Medium dark gray
	BorderSecondary: lipgloss.Color("#6B7280"), // Muted gray
	
	// Progress bar gradient colors
	ProgressStart: lipgloss.Color("#FF6B6B"), // Red
	ProgressEnd:   lipgloss.Color("#4ECDC4"), // Teal
}

// Helper functions for creating common styles
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