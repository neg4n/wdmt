package cmd

import (
	"fmt"
	"os"
	"time"

	"wdmt/internal/cleaner"
	"wdmt/internal/scanner"
	"wdmt/internal/ui"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	Version = "1.0.0"
)

type scanProgressMsg float64
type scanCompleteMsg struct{}

type scanModel struct {
	progress progress.Model
	done     bool
}

func newScanModel() scanModel {
	return scanModel{
		progress: progress.New(
			progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
			progress.WithWidth(20),
		),
	}
}

func (m scanModel) Init() tea.Cmd {
	return nil
}

func (m scanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case scanProgressMsg:
		return m, m.progress.SetPercent(float64(msg))
	case scanCompleteMsg:
		m.done = true
		return m, tea.Quit
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m scanModel) View() string {
	if m.done {
		return ""
	}
	return "\n" + "wdmt scanning... " + m.progress.View() + "\n"
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

		err = s.ScanWithProgress(func(progress float64) {
			p.Send(scanProgressMsg(progress))
		})

		if err != nil {
			scanErr = err
			p.Send(scanCompleteMsg{})
			return
		}

		p.Send(scanProgressMsg(1.0))

		time.Sleep(1 * time.Second)
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
		fmt.Println("\n✨ no cleanup targets found! your directory is already clean.\n")
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

	interactiveUI := ui.New(validTargets)
	interactiveUI.SetCleaner(cleanerInstance)
	p := tea.NewProgram(interactiveUI.GetModel(), tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return fmt.Errorf("failed to run interactive interface: %w", err)
	}

	return nil
}
