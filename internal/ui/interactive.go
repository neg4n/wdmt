package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wdmt/internal/scanner"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type State int

const (
	StateScanning State = iota
	StateSelectingTargets
	StateConfirming
	StateDeleting
	StateCompletionDelay
)

type PathDisplayMode int

const (
	PathDisplaySmart PathDisplayMode = iota
	PathDisplayCondensed
	PathDisplayFull
)

func (pdm PathDisplayMode) String() string {
	switch pdm {
	case PathDisplaySmart:
		return "smart"
	case PathDisplayCondensed:
		return "condensed"
	case PathDisplayFull:
		return "full"
	default:
		return "unknown"
	}
}

type DeleteProgress struct {
	Target        scanner.CleanupTarget
	Progress      float64
	Done          bool
	Error         error
	OriginalIndex int
}

type InteractiveUI struct {
	model *Model
}

type Model struct {
	state           State
	targets         []scanner.CleanupTarget
	selectedItems   map[int]bool
	cursor          int
	width           int
	height          int
	err             error
	spinner         spinner.Model
	list            list.Model
	progress        progress.Model
	deleteProgress  map[int]*DeleteProgress
	totalFreed      int64
	deletedCount    int
	showingHelp     bool
	cleaner         interface{} 
	pathDisplayMode PathDisplayMode
	workingDir      string
	scrollOffset    int
	scanDuration    string
}

type CleanupItem struct {
	target   scanner.CleanupTarget
	index    int
	selected bool
	model    *Model
}

func (i CleanupItem) FilterValue() string { return i.target.Name }
func (i CleanupItem) Title() string       { return i.formatTitle() }
func (i CleanupItem) Description() string { return i.formatDescription() }

type ItemDelegate struct {
	selectedItems map[int]bool
}

func (d ItemDelegate) Height() int                             { return 2 }
func (d ItemDelegate) Spacing() int                            { return 1 }
func (d ItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	if i, ok := listItem.(CleanupItem); ok {
		var style lipgloss.Style
		isSelected := d.selectedItems[i.index]
		isFocused := index == m.Index()

		if isFocused && isSelected {
			style = selectedFocusedStyle
		} else if isFocused {
			style = focusedStyle
		} else if isSelected {
			style = selectedStyle
		} else {
			style = normalStyle
		}

		checkbox := "☐"
		if isSelected {
			checkbox = "☑"
		}

		title := style.Render(fmt.Sprintf("%s %s", checkbox, i.Title()))
		desc := dimStyle.Render(i.Description())

		fmt.Fprintf(w, "%s\n%s", title, desc)
	}
}

func (i CleanupItem) formatTitle() string {
	path := i.target.Path

	if i.model == nil {
		return path
	}

	switch i.model.pathDisplayMode {
	case PathDisplayFull:
		return path

	case PathDisplayCondensed:
		return i.formatCondensedPath(path)

	case PathDisplaySmart:
		return i.formatSmartPath(path)

	default:
		return path
	}
}

func (i CleanupItem) formatCondensedPath(path string) string {
	parts := strings.Split(path, string(filepath.Separator))

	if len(parts) <= 3 {
		return path
	}

	var shortened []string
	for j, part := range parts {
		if j == len(parts)-1 {
			shortened = append(shortened, part) 
		} else if part != "" {
			if len(part) > 0 {
				shortened = append(shortened, string(part[0]))
			}
		}
	}

	return strings.Join(shortened, "/")
}

func (i CleanupItem) formatSmartPath(path string) string {
	if i.model.workingDir == "" {
		return path
	}

	relPath, err := filepath.Rel(i.model.workingDir, path)
	if err != nil {
		return path
	}

	if strings.HasPrefix(relPath, "..") {
		return i.formatCondensedPath(path)
	}

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) <= 3 {
		return relPath
	}

	var result []string
	for j, part := range parts {
		if j <= 2 { 
			result = append(result, part)
		} else if j == len(parts)-1 {
			result = append(result, part)
		} else if part != "" {
			if len(part) > 0 {
				result = append(result, string(part[0]))
			}
		}
	}

	return strings.Join(result, "/")
}

func (i CleanupItem) formatDescription() string {
	return fmt.Sprintf("%s • %s", i.target.Type, formatSize(i.target.Size))
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Primary).
			Background(Colors.BgPrimary).
			Padding(0, 2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Colors.Success).
			Padding(0, 1).
			MarginBottom(1)

	containerStyle = HeaderContainerStyle()


	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			PaddingLeft(2)

	focusedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Background(lipgloss.Color("#374151")).
			PaddingLeft(2).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			PaddingLeft(2)

	selectedFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981")).
				Background(lipgloss.Color("#065F46")).
				PaddingLeft(2).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			PaddingLeft(4)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true)

	// Enhanced warning container style
	warningContainerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B")).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#F59E0B")).
				Padding(0, 1).
				MarginBottom(1)


	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			MarginTop(1).
			Italic(true)
)

type errMsg error
type deleteFinishedMsg struct{ index int }
type deleteProgressMsg struct {
	index    int
	progress float64
}
type progressTickMsg struct{ index int }
type completionDelayMsg struct{}
type exitAfterDelayMsg struct{}

func New(targets []scanner.CleanupTarget) *InteractiveUI {
	return NewWithScanner(targets, nil)
}

func NewWithScanner(targets []scanner.CleanupTarget, scannerInstance *scanner.Scanner) *InteractiveUI {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(Colors.Primary)

	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}

	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.PercentageStyle = lipgloss.NewStyle().Foreground(Colors.Success)

	scanDuration := ""
	if scannerInstance != nil {
		scanDuration = scannerInstance.GetScanDurationString()
	}

	model := &Model{
		state:           StateSelectingTargets,
		targets:         targets,
		selectedItems:   make(map[int]bool),
		spinner:         s,
		progress:        progressBar,
		deleteProgress:  make(map[int]*DeleteProgress),
		pathDisplayMode: PathDisplaySmart,
		workingDir:      workingDir,
		scanDuration:    scanDuration,
	}

	items := make([]list.Item, len(targets))
	for i, target := range targets {
		items[i] = CleanupItem{target: target, index: i, model: model}
	}

	l := list.New(items, ItemDelegate{selectedItems: make(map[int]bool)}, 80, 20)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = headerStyle

	model.list = l

	return &InteractiveUI{model: model}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width - 4)
		m.list.SetHeight(msg.Height - 8)

	case tea.KeyMsg:
		switch m.state {
		case StateSelectingTargets:
			return m.updateSelecting(msg)
		case StateConfirming:
			return m.updateConfirming(msg)
		case StateDeleting:
			return m.updateDeleting(msg)
		case StateCompletionDelay:
			return m.updateCompletionDelay(msg)
		}

	case errMsg:
		m.err = msg
		return m, nil

	case deleteProgressMsg:
		if dp, exists := m.deleteProgress[msg.index]; exists {
			dp.Progress = msg.progress
		}
		return m, nil

	case progressTickMsg:
		if dp, exists := m.deleteProgress[msg.index]; exists && !dp.Done {
			newProgress := dp.Progress + 0.03
			if newProgress > 0.95 {
				newProgress = 0.95 // Cap at 95% until deletion completes
			}
			dp.Progress = newProgress
			// Continue animation
			return m, m.animateProgress(msg.index)
		}
		return m, nil

	case exitAfterDelayMsg:
		// Print summary to terminal and exit
		m.printSummaryAndExit()
		return m, tea.Quit

	case deleteFinishedMsg:
		if dp, exists := m.deleteProgress[msg.index]; exists {
			dp.Done = true
			dp.Progress = 1.0
			m.deletedCount++
			m.totalFreed += dp.Target.Size
		}

		// Check if all deletions are done
		allDone := true
		for _, dp := range m.deleteProgress {
			if !dp.Done {
				allDone = false
				break
			}
		}

		if allDone {
			// Move to completion delay state
			m.state = StateCompletionDelay
			return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
				return exitAfterDelayMsg{}
			})
		}

		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	if m.state == StateSelectingTargets {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSelecting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case " ":
		index := m.list.Index()
		if index < len(m.targets) {
			m.selectedItems[index] = !m.selectedItems[index]
			delegate := ItemDelegate{selectedItems: m.selectedItems}
			m.list.SetDelegate(delegate)
		}
		return m, nil
	case "enter":
		if len(m.getSelectedTargets()) > 0 {
			m.state = StateConfirming
			m.scrollOffset = 0 // Reset scroll when changing states
		}
		return m, nil
	case "a":
		for i := range m.targets {
			m.selectedItems[i] = true
		}
		delegate := ItemDelegate{selectedItems: m.selectedItems}
		m.list.SetDelegate(delegate)
		return m, nil
	case "A":
		m.selectedItems = make(map[int]bool)
		delegate := ItemDelegate{selectedItems: m.selectedItems}
		m.list.SetDelegate(delegate)
		return m, nil
	case "p":
		switch m.pathDisplayMode {
		case PathDisplaySmart:
			m.pathDisplayMode = PathDisplayCondensed
		case PathDisplayCondensed:
			m.pathDisplayMode = PathDisplayFull
		case PathDisplayFull:
			m.pathDisplayMode = PathDisplaySmart
		}
		items := make([]list.Item, len(m.targets))
		for i, target := range m.targets {
			items[i] = CleanupItem{target: target, index: i, model: m}
		}
		m.list.SetItems(items)
		return m, nil
	case "?":
		m.showingHelp = !m.showingHelp
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) updateConfirming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.state = StateDeleting
		m.scrollOffset = 0 // Reset scroll when changing states
		return m, m.startDeletion()
	case "n", "N", "q", "ctrl+c", "esc":
		m.state = StateSelectingTargets
		m.scrollOffset = 0 // Reset scroll when changing states
		return m, nil
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil
	case "down", "j":
		selected, _ := m.getSelectedTargetsWithIndices()
		maxScroll := len(selected) - (m.height - 8) // Adjust for header and help
		if maxScroll > 0 && m.scrollOffset < maxScroll {
			m.scrollOffset++
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateDeleting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
		return m, nil
	case "down", "j":
		maxScroll := len(m.deleteProgress) - (m.height - 8) // Adjust for header and help
		if maxScroll > 0 && m.scrollOffset < maxScroll {
			m.scrollOffset++
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateCompletionDelay(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key press exits immediately
	m.printSummaryAndExit()
	return m, tea.Quit
}

func (m *Model) getSelectedTargets() []scanner.CleanupTarget {
	var selected []scanner.CleanupTarget
	for i, target := range m.targets {
		if m.selectedItems[i] {
			selected = append(selected, target)
		}
	}
	return selected
}

func (m *Model) getSelectedTargetsWithIndices() ([]scanner.CleanupTarget, []int) {
	var selected []scanner.CleanupTarget
	var indices []int
	for i, target := range m.targets {
		if m.selectedItems[i] {
			selected = append(selected, target)
			indices = append(indices, i)
		}
	}
	return selected, indices
}

func (m *Model) getSortedProgressIndices() []int {
	var indices []int
	for i := range m.deleteProgress {
		indices = append(indices, i)
	}

	for i := 0; i < len(indices); i++ {
		for j := i + 1; j < len(indices); j++ {
			if m.deleteProgress[indices[i]].OriginalIndex > m.deleteProgress[indices[j]].OriginalIndex {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}

	return indices
}

func (m *Model) startDeletion() tea.Cmd {
	selected, originalIndices := m.getSelectedTargetsWithIndices()

	for i, target := range selected {
		originalIndex := originalIndices[i]
		m.deleteProgress[originalIndex] = &DeleteProgress{
			Target:        target,
			Progress:      0.0,
			Done:          false,
			OriginalIndex: originalIndex,
		}
	}

	var cmds []tea.Cmd
	for i, target := range selected {
		originalIndex := originalIndices[i]
		cmds = append(cmds, m.deleteDirectory(originalIndex, target))
	}

	return tea.Batch(cmds...)
}

func (m *Model) deleteDirectory(index int, target scanner.CleanupTarget) tea.Cmd {
	return tea.Batch(
		m.animateProgress(index),
		func() tea.Msg {
			err := os.RemoveAll(target.Path)
			if err != nil {
				return errMsg(err)
			}
			return deleteFinishedMsg{index: index}
		},
	)
}

func (m *Model) animateProgress(index int) tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return progressTickMsg{index: index}
	})
}

func (m *Model) printSummaryAndExit() {
	fmt.Println() // Add a newline after TUI closes
	
	if m.deletedCount == 0 {
		fmt.Println("🚫 No directories deleted")
	} else {
		fmt.Printf("✅ Deleted %d directories • %s freed\n", m.deletedCount, formatSize(m.totalFreed))
		
		// List deleted directories
		sortedIndices := m.getSortedProgressIndices()
		for _, i := range sortedIndices {
			dp := m.deleteProgress[i]
			if dp.Done {
				shortPath := CleanupItem{target: dp.Target, index: i, model: m}.formatTitle()
				fmt.Printf("  ✗ %s (%s)\n", shortPath, formatSize(dp.Target.Size))
			}
		}
	}
	fmt.Println()
}

func (m *Model) View() string {
	var content strings.Builder

	switch m.state {
	case StateScanning:
		content.WriteString(m.viewScanning())
	case StateSelectingTargets:
		content.WriteString(m.viewSelecting())
	case StateConfirming:
		content.WriteString(m.viewConfirming())
	case StateDeleting:
		content.WriteString(m.viewDeleting())
	case StateCompletionDelay:
		content.WriteString(m.viewCompletionDelay())
	}

	if m.err != nil {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", m.err)))
	}

	return content.String()
}

func (m *Model) viewScanning() string {
	return fmt.Sprintf("%s Scanning for cleanup targets...", m.spinner.View())
}

func (m *Model) viewSelecting() string {
	var content strings.Builder

	if len(m.targets) == 0 {
		content.WriteString(successStyle.Render("✨ No cleanup targets found! Your directory is already clean."))
		content.WriteString("\n\n")
		content.WriteString(helpStyle.Render("Press 'q' to quit"))
		return content.String()
	}

	var allTargetsSize int64
	for _, target := range m.targets {
		allTargetsSize += target.Size
	}

	// Calculate selected targets more efficiently
	selectedTargets := m.getSelectedTargets()
	selectedCount := len(selectedTargets)
	var selectedSize int64
	for _, target := range selectedTargets {
		selectedSize += target.Size
	}

	var statsContent strings.Builder

	mainStats := fmt.Sprintf("💾 %s available", formatSize(allTargetsSize))
	statsContent.WriteString(lipgloss.NewStyle().
		Foreground(Colors.Success).
		Bold(true).
		Render(mainStats))

	statsContent.WriteString(" • ")

	selectionInfo := fmt.Sprintf("%d selected", selectedCount)
	selectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
	if selectedCount > 0 {
		selectionStyle = selectionStyle.Bold(true)
		selectionInfo = fmt.Sprintf("%d selected (%s)", selectedCount, formatSize(selectedSize))
	}
	statsContent.WriteString(selectionStyle.Render(selectionInfo))

	statsContent.WriteString(" • ")

	if m.scanDuration != "" {
		scanInfo := fmt.Sprintf("Scanned in %s", m.scanDuration)
		statsContent.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Render(scanInfo))
		statsContent.WriteString(" • ")
	}

	pathInfo := fmt.Sprintf("Path: %s", m.pathDisplayMode)
	statsContent.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render(pathInfo))

	styledStats := containerStyle.Render(statsContent.String())
	content.WriteString(styledStats)
	content.WriteString("\n")

	m.list.Title = fmt.Sprintf("📁 %d directories found", len(m.targets))

	content.WriteString(m.list.View())
	content.WriteString("\n")

	if m.showingHelp {
		help := `Commands:
  ↑/↓, j/k    Navigate    space    Toggle selection    a/A    Select/deselect all
  p           Path mode   enter    Proceed             ?      Toggle help    q    Quit`
		content.WriteString(helpStyle.Render(help))
	} else {
		help := "? help • space select • p path mode • enter proceed • q quit"
		content.WriteString(helpStyle.Render(help))
	}

	return content.String()
}


func (m *Model) viewConfirming() string {
	var content strings.Builder

	selected, originalIndices := m.getSelectedTargetsWithIndices()
	var totalSize int64
	for _, target := range selected {
		totalSize += target.Size
	}

	// Header box with confirmation question (consistent with selection screen)
	confirmationHeader := fmt.Sprintf("⚠️  Confirm deletion of %d directories (%s)?", len(selected), formatSize(totalSize))
	styledHeader := warningContainerStyle.Render(confirmationHeader)
	content.WriteString(styledHeader)
	content.WriteString("\n")

	// Calculate available height for directory list with scrolling support
	reservedLines := 5 // Header + help + padding
	availableHeight := m.height - reservedLines
	maxVisibleItems := availableHeight - 1 // Leave buffer

	// Scrollable list of directories to be deleted
	startIdx := m.scrollOffset
	endIdx := startIdx + maxVisibleItems
	if endIdx > len(selected) {
		endIdx = len(selected)
	}

	// Show items within the current scroll window
	for i := startIdx; i < endIdx; i++ {
		target := selected[i]
		originalIndex := originalIndices[i]
		shortPath := CleanupItem{target: target, index: originalIndex, model: m}.formatTitle()
		
		// Ensure paths fit within viewport width
		maxPathWidth := m.width - 12 // Account for icon and size
		if len(shortPath) > maxPathWidth {
			shortPath = shortPath[:maxPathWidth-3] + "..."
		}
		
		itemStyle := lipgloss.NewStyle().Foreground(Colors.Error).PaddingLeft(2)
		content.WriteString(itemStyle.Render(fmt.Sprintf("🗑  %s (%s)", shortPath, formatSize(target.Size))))
		content.WriteString("\n")
	}

	// Show scroll indicators if needed
	if len(selected) > maxVisibleItems {
		scrollInfo := ""
		if m.scrollOffset > 0 {
			scrollInfo += "↑ "
		}
		scrollInfo += fmt.Sprintf("%d-%d of %d", startIdx+1, endIdx, len(selected))
		if endIdx < len(selected) {
			scrollInfo += " ↓"
		}
		
		scrollStyle := lipgloss.NewStyle().Foreground(Colors.TextMuted).PaddingLeft(2)
		content.WriteString(scrollStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	
	// Help text that fits in viewport
	helpText := "Y/y confirm • N/n cancel • ESC go back"
	if len(selected) > maxVisibleItems {
		helpText += " • ↑/↓ scroll"
	}
	helpStyle := lipgloss.NewStyle().
		Foreground(Colors.TextSecondary).
		Italic(true).
		MaxWidth(m.width - 4)
	content.WriteString(helpStyle.Render(helpText))

	return content.String()
}


func (m *Model) viewDeleting() string {
	var content strings.Builder

	// Calculate progress and size information
	totalItems := len(m.deleteProgress)
	completedItems := 0
	var totalSizeToDelete int64
	var deletedSize int64
	
	for _, dp := range m.deleteProgress {
		totalSizeToDelete += dp.Target.Size
		if dp.Done {
			completedItems++
			deletedSize += dp.Target.Size
		}
	}
	
	// Header box with deletion metadata (consistent with selection screen)
	progressPercent := float64(completedItems) / float64(totalItems) * 100
	deletionHeader := fmt.Sprintf("🗑️  Deleting %d directories • %.0f%% complete • %s of %s freed", 
		totalItems, progressPercent, formatSize(deletedSize), formatSize(totalSizeToDelete))
	styledHeader := HeaderContainerStyle().Render(deletionHeader)
	content.WriteString(styledHeader)
	content.WriteString("\n")

	// Calculate available height for directory list with scrolling support
	reservedLines := 5 // Header + help + padding
	availableHeight := m.height - reservedLines
	maxVisibleItems := availableHeight / 2 // Each item takes ~2 lines (status + progress)
	if maxVisibleItems < 1 {
		maxVisibleItems = 1
	}

	// Individual item progress with scrolling
	sortedIndices := m.getSortedProgressIndices()
	startIdx := m.scrollOffset
	endIdx := startIdx + maxVisibleItems
	if endIdx > len(sortedIndices) {
		endIdx = len(sortedIndices)
	}

	// Show items within the current scroll window
	for idx := startIdx; idx < endIdx; idx++ {
		i := sortedIndices[idx]
		dp := m.deleteProgress[i]

		status := "⏳"
		statusColor := Colors.Warning // Yellow for in progress
		if dp.Done {
			status = "✅"
			statusColor = Colors.Success // Green for done
		} else if dp.Error != nil {
			status = "❌"
			statusColor = Colors.Error // Red for error
		}

		shortPath := CleanupItem{target: dp.Target, index: i, model: m}.formatTitle()
		
		// Ensure paths fit within viewport width
		maxPathWidth := m.width - 12 // Account for icon and size
		if len(shortPath) > maxPathWidth {
			shortPath = shortPath[:maxPathWidth-3] + "..."
		}
		
		// Status and file info
		statusStyle := lipgloss.NewStyle().Foreground(statusColor).Bold(true)
		pathStyle := lipgloss.NewStyle().Foreground(Colors.TextPrimary)
		sizeStyle := lipgloss.NewStyle().Foreground(Colors.TextSecondary)
		
		content.WriteString(statusStyle.Render(status))
		content.WriteString(" ")
		content.WriteString(pathStyle.Render(shortPath))
		content.WriteString(" ")
		content.WriteString(sizeStyle.Render(fmt.Sprintf("(%s)", formatSize(dp.Target.Size))))
		content.WriteString("\n")

		// Progress bar for active deletions
		if !dp.Done && dp.Error == nil {
			progressBar := progress.New(
				progress.WithScaledGradient(string(Colors.ProgressStart), string(Colors.ProgressEnd)),
				progress.WithWidth(40),
			)
			progressBar.PercentageStyle = lipgloss.NewStyle().Foreground(Colors.Success)
			
			content.WriteString("  ")
			content.WriteString(progressBar.ViewAs(dp.Progress))
			content.WriteString("\n")
		} else {
			// Add spacing for completed items
			content.WriteString("\n")
		}
	}

	// Show scroll indicators if needed
	if len(sortedIndices) > maxVisibleItems {
		scrollInfo := ""
		if m.scrollOffset > 0 {
			scrollInfo += "↑ "
		}
		scrollInfo += fmt.Sprintf("%d-%d of %d", startIdx+1, endIdx, len(sortedIndices))
		if endIdx < len(sortedIndices) {
			scrollInfo += " ↓"
		}
		
		scrollStyle := lipgloss.NewStyle().Foreground(Colors.TextMuted).PaddingLeft(2)
		content.WriteString(scrollStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	// Help text
	content.WriteString("\n")
	if completedItems < totalItems {
		helpText := "Press Ctrl+C to cancel (not recommended during deletion)"
		if len(sortedIndices) > maxVisibleItems {
			helpText += " • ↑/↓ scroll"
		}
		content.WriteString(helpStyle.Render(helpText))
	} else {
		content.WriteString(helpStyle.Render("Cleanup completed! Exiting..."))
	}

	return content.String()
}

func (m *Model) viewCompletionDelay() string {
	var content strings.Builder

	// Success header
	header := successStyle.Render("✅ Cleanup completed successfully!")
	content.WriteString(header)
	content.WriteString("\n\n")

	// Progress overview
	totalItems := len(m.deleteProgress)
	progressInfo := fmt.Sprintf("Cleaned %d directories • %s freed", totalItems, formatSize(m.totalFreed))
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
	content.WriteString(progressStyle.Render(progressInfo))
	content.WriteString("\n\n")

	// Show completed items
	sortedIndices := m.getSortedProgressIndices()
	for _, i := range sortedIndices {
		dp := m.deleteProgress[i]
		if dp.Done {
			shortPath := CleanupItem{target: dp.Target, index: i, model: m}.formatTitle()
			
			statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Bold(true)
			pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB"))
			sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
			
			content.WriteString(statusStyle.Render("✅"))
			content.WriteString(" ")
			content.WriteString(pathStyle.Render(shortPath))
			content.WriteString(" ")
			content.WriteString(sizeStyle.Render(fmt.Sprintf("(%s)", formatSize(dp.Target.Size))))
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	
	// Auto-exit message
	exitMessage := "Closing in 5 seconds or press any key to exit immediately"
	exitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		Italic(true).
		Bold(true)
	content.WriteString(exitStyle.Render(exitMessage))

	return content.String()
}


func (ui *InteractiveUI) SelectTargets() ([]scanner.CleanupTarget, error) {
	if len(ui.model.targets) == 0 {
		fmt.Println("✨ No cleanup targets found! Your directory is already clean.")
		return nil, nil
	}

	p := tea.NewProgram(ui.model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run interactive interface: %w", err)
	}

	if model, ok := finalModel.(*Model); ok {
		return model.getSelectedTargets(), nil
	}

	return nil, fmt.Errorf("unexpected model type")
}


func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (ui *InteractiveUI) GetModel() *Model {
	return ui.model
}

func (ui *InteractiveUI) SetCleaner(cleaner interface{}) {
	ui.model.cleaner = cleaner
}
