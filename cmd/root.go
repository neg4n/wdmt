package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/neg4n/wdmt/internal/cleaner"
	"github.com/neg4n/wdmt/internal/scanner"
	"github.com/neg4n/wdmt/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	Version = "1.0.0"
)

type scanTickMsg struct{}
type scanCompleteMsg struct{}

type scanModel struct {
	done          bool
	animFrame     int
	barWidth      int
	ballPosition  int
	ballDirection int
	scanStartTime time.Time
	messages      []string
	messageIndex  int
}

var loadingMessages = []string{
	"Hunting for node_modules monsters...",
	"Chasing build artifacts in the wild...",
	"Detecting cache creatures...",
	"Searching for forgotten dependencies...",
	"Tracking down temporary files...",
	"Discovering hidden build outputs...",
	"Scanning for development debris...",
	"Finding orphaned test coverage...",
	"Locating stray distribution files...",
	"Investigating suspicious .next folders...",
}

func newScanModel() scanModel {
	return scanModel{
		done:          false,
		animFrame:     0,
		barWidth:      20,
		ballPosition:  0,
		ballDirection: 1,
		scanStartTime: time.Now(),
		messages:      loadingMessages,
		messageIndex:  0,
	}
}

func (m scanModel) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return scanTickMsg{}
	})
}

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case scanTickMsg:
		if !m.done {
			m.animFrame++

			m.ballPosition += m.ballDirection

			if m.ballPosition >= m.barWidth-1 {
				m.ballDirection = -1
			} else if m.ballPosition <= 0 {
				m.ballDirection = 1
			}

			if m.animFrame%37 == 0 && len(m.messages) > 0 {
				m.messageIndex = (m.messageIndex + 1) % len(m.messages)
			}

			return m, tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
				return scanTickMsg{}
			})
		}
		return m, nil

	case scanCompleteMsg:
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m scanModel) View() string {
	if m.done {
		return ""
	}

	var bar strings.Builder

	ballColors := []string{"#ff006e", "#fb5607", "#ffbe0b", "#8338ec", "#3a86ff"}

	ballColor := ballColors[m.animFrame%len(ballColors)]

	for i := 0; i < m.barWidth; i++ {
		if i == m.ballPosition {

			styled := lipgloss.NewStyle().Foreground(lipgloss.Color(ballColor)).Render("█")
			bar.WriteString(styled)
		} else {

			distance := abs(i - m.ballPosition)
			var char string
			var color string

			if distance <= 1 {

				char = "▓"
				color = "#4a5568"
			} else if distance <= 2 {

				char = "▒"
				color = "#2d3748"
			} else {

				char = "░"
				color = "#1a202c"
			}

			styled := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(char)
			bar.WriteString(styled)
		}
	}

	currentMessage := "scanning directories..."
	if len(m.messages) > 0 {
		currentMessage = m.messages[m.messageIndex]
	}

	return fmt.Sprintf("\nWDMT %s\n\n%s\n\n", bar.String(), currentMessage)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var rootCmd = &cobra.Command{
	Use:   "wdmt",
	Short: "Web Developer Maintenance Tool - Clean up your development directories",
	Long: `WDMT is a CLI tool for web developers to safely clean up common development
directories like node_modules, .next, dist, build, and more.

It provides an interactive interface to select which directories to remove,
with built-in safety features to prevent deletion outside the current
working directory.`,
	Version: Version,
	Run:     runCleanup,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func runCleanup(cmd *cobra.Command, args []string) {
	model := newScanModel()
	p := tea.NewProgram(model)

	var scannerInstance *scanner.Scanner
	var scanErr error

	go func() {
		s, err := scanner.New()
		if err != nil {
			scanErr = err
			p.Send(scanCompleteMsg{})
			return
		}
		scannerInstance = s

		err = s.Scan()

		if err != nil {
			scanErr = err
			p.Send(scanCompleteMsg{})
			return
		}

		time.Sleep(500 * time.Millisecond)
		p.Send(scanCompleteMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if scanErr != nil {
		fmt.Printf("Error during scanning: %v\n", scanErr)
		os.Exit(1)
	}

	if err := performCleanupWithScanner(scannerInstance); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func performCleanupWithScanner(s *scanner.Scanner) error {
	targets := s.GetTargets()

	if len(targets) == 0 {
		fmt.Println("✨ no cleanup targets found! your directory is already clean.")
		return nil
	}

	cleanerInstance, err := cleaner.New(s.GetWorkingDir())
	if err != nil {
		return fmt.Errorf("failed to initialize cleaner: %w", err)
	}

	validTargets, err := cleanerInstance.ValidateTargets(targets)
	if err != nil {
		return fmt.Errorf("failed to validate targets: %w", err)
	}

	if len(validTargets) == 0 {
		fmt.Println("⚠️  No valid targets remain after validation.")
		return nil
	}

	interactiveUI := ui.NewWithScanner(validTargets, s)
	interactiveUI.SetCleaner(cleanerInstance)
	p := tea.NewProgram(interactiveUI.GetModel(), tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run interactive interface: %w", err)
	}

	return nil
}
