package ui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	StateSummary
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
			Foreground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")).
			Padding(0, 1).
			MarginBottom(1)

	containerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1).
			MarginBottom(1)

	listHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(0, 1).
			MarginBottom(1)

	sectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FBBF24")).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#374151")).
				Padding(0, 1).
				MarginBottom(1)

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

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981"))

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

func New(targets []scanner.CleanupTarget) *InteractiveUI {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = ""
	}

	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.PercentageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))

	model := &Model{
		state:           StateSelectingTargets,
		targets:         targets,
		selectedItems:   make(map[int]bool),
		spinner:         s,
		progress:        progressBar,
		deleteProgress:  make(map[int]*DeleteProgress),
		pathDisplayMode: PathDisplaySmart,
		workingDir:      workingDir,
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
		case StateSummary:
			return m.updateSummary(msg)
		}

	case errMsg:
		m.err = msg
		return m, nil

	case deleteProgressMsg:
		if dp, exists := m.deleteProgress[msg.index]; exists {
			dp.Progress = msg.progress
		}
		return m, nil

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
			m.state = StateSummary
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
		return m, m.startDeletion()
	case "n", "N", "q", "ctrl+c", "esc":
		m.state = StateSelectingTargets
		return m, nil
	}
	return m, nil
}

func (m *Model) updateDeleting(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "enter", "esc":
		return m, tea.Quit
	}
	return m, nil
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
	return func() tea.Msg {
		err := os.RemoveAll(target.Path)
		if err != nil {
			return errMsg(err)
		}

		return deleteFinishedMsg{index: index}
	}
}

func (m *Model) View() string {
	var content strings.Builder

	if m.state == StateSelectingTargets {
		title := titleStyle.Render("🧹 Web Developer Maintenance Tool")
		content.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title))
		content.WriteString("\n\n")
	}

	switch m.state {
	case StateScanning:
		content.WriteString(m.viewScanning())
	case StateSelectingTargets:
		content.WriteString(m.viewSelecting())
	case StateConfirming:
		content.WriteString(m.viewConfirming())
	case StateDeleting:
		content.WriteString(m.viewDeleting())
	case StateSummary:
		content.WriteString(m.viewSummary())
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

	selectedCount := len(m.getSelectedTargets())
	var selectedSize int64
	for _, target := range m.getSelectedTargets() {
		selectedSize += target.Size
	}

	var statsContent strings.Builder

	mainStats := fmt.Sprintf("💾 %s available", formatSize(allTargetsSize))
	statsContent.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
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

	confirmationHeader := fmt.Sprintf("⚠️  Delete %d directories (%s)?", len(selected), formatSize(totalSize))
	header := warningContainerStyle.Render(confirmationHeader)
	content.WriteString(header)
	content.WriteString("\n\n")

	for i, target := range selected {
		originalIndex := originalIndices[i]
		shortPath := CleanupItem{target: target, index: originalIndex, model: m}.formatTitle()
		content.WriteString(fmt.Sprintf("  🗑  %s (%s)\n", shortPath, formatSize(target.Size)))
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("y confirm • n cancel"))

	return content.String()
}

func (m *Model) viewDeleting() string {
	var content strings.Builder

	header := sectionHeaderStyle.Render("🗑️  Deleting...")
	content.WriteString(header)
	content.WriteString("\n\n")

	sortedIndices := m.getSortedProgressIndices()
	for _, i := range sortedIndices {
		dp := m.deleteProgress[i]

		status := "⏳"
		if dp.Done {
			status = "✅"
		} else if dp.Error != nil {
			status = "❌"
		}

		shortPath := CleanupItem{target: dp.Target, index: i, model: m}.formatTitle()
		content.WriteString(fmt.Sprintf("%s %s (%s)\n", status, shortPath, formatSize(dp.Target.Size)))

		if dp.Done || dp.Error != nil {
			content.WriteString("    " + strings.Repeat(" ", 48) + "\n") 
		} else {
			progressBar := progress.New(
				progress.WithScaledGradient("#FF6B6B", "#4ECDC4"),
				progress.WithWidth(40),
			)
			progressBar.PercentageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
			content.WriteString("    ")
			content.WriteString(progressBar.ViewAs(dp.Progress))
			content.WriteString("\n")
		}

		content.WriteString("\n")
	}

	content.WriteString(helpStyle.Render("q cancel"))

	return content.String()
}

func (m *Model) viewSummary() string {
	var content strings.Builder

	if m.deletedCount == 0 {
		header := warningContainerStyle.Render("🚫 No directories deleted")
		content.WriteString(header)
	} else {
		resultText := fmt.Sprintf("✅ Deleted %d directories • %s freed", m.deletedCount, formatSize(m.totalFreed))
		header := containerStyle.Render(resultText)
		content.WriteString(header)
	}

	content.WriteString("\n\n")

	if m.deletedCount > 0 {
		sortedIndices := m.getSortedProgressIndices()
		for _, i := range sortedIndices {
			dp := m.deleteProgress[i]
			if dp.Done {
				shortPath := CleanupItem{target: dp.Target, index: i, model: m}.formatTitle()
				content.WriteString(fmt.Sprintf("  ✗ %s (%s)\n", shortPath, formatSize(dp.Target.Size)))
			}
		}
		content.WriteString("\n")
	}

	content.WriteString(helpStyle.Render("any key to exit"))

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

func (ui *InteractiveUI) ConfirmDeletion(targets []scanner.CleanupTarget) (bool, error) {
	return true, nil
}

func (ui *InteractiveUI) ShowSummary(deletedTargets []scanner.CleanupTarget, totalFreed int64) {
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
