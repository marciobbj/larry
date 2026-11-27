package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"larry/internal/search"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type globalReplaceFocus int

const (
	grFocusFind globalReplaceFocus = iota
	grFocusReplace
	grFocusList
)

type globalReplacePreviewMsg struct {
	Query  string
	Groups []search.FileReplaceGroup
	Err    error
}

// GlobalReplaceModel manages the project-scoped Global Replace modal.
type GlobalReplaceModel struct {
	root         string
	findInput    textinput.Model
	replaceInput textinput.Model
	focus        globalReplaceFocus
	groups       []search.FileReplaceGroup
	selected     map[string]bool
	cursor       int
	scrollOffset int
	width        int
	height       int
	loading      bool
	confirming   bool
	errMsg       string
	replacer     *search.GlobalReplacer
}

// NewGlobalReplaceModel creates a GlobalReplaceModel scoped to root.
func NewGlobalReplaceModel(root string, width, height int) GlobalReplaceModel {
	if root == "" {
		root = "."
	}

	findIn := textinput.New()
	findIn.Placeholder = "Text to find across project..."
	findIn.Prompt = " » "
	findIn.Focus()

	repIn := textinput.New()
	repIn.Placeholder = "Replacement text (empty to delete)..."
	repIn.Prompt = " » "

	return GlobalReplaceModel{
		root:         root,
		findInput:    findIn,
		replaceInput: repIn,
		focus:        grFocusFind,
		selected:     make(map[string]bool),
		width:        width,
		height:       height,
		replacer:     search.NewGlobalReplacer(),
	}
}

func (m GlobalReplaceModel) Root() string {
	return m.root
}

func (m GlobalReplaceModel) FindValue() string {
	return m.findInput.Value()
}

func (m GlobalReplaceModel) ReplaceValue() string {
	return m.replaceInput.Value()
}

func (m GlobalReplaceModel) SelectedCounts() (files int, matches int) {
	for _, g := range m.groups {
		if m.selected[g.Path] {
			files++
			matches += g.TotalCount
		}
	}
	return files, matches
}

func (m GlobalReplaceModel) TotalCounts() (files int, matches int) {
	files = len(m.groups)
	for _, g := range m.groups {
		matches += g.TotalCount
	}
	return files, matches
}

func (m *GlobalReplaceModel) setFocus(f globalReplaceFocus) {
	m.focus = f
	switch f {
	case grFocusFind:
		m.findInput.Focus()
		m.replaceInput.Blur()
	case grFocusReplace:
		m.findInput.Blur()
		m.replaceInput.Focus()
	case grFocusList:
		m.findInput.Blur()
		m.replaceInput.Blur()
	}
}

func (m *GlobalReplaceModel) performPreview() tea.Cmd {
	query := m.findInput.Value()
	if query == "" {
		m.groups = nil
		m.selected = make(map[string]bool)
		m.cursor = 0
		m.scrollOffset = 0
		m.loading = false
		m.errMsg = ""
		return nil
	}

	m.loading = true
	m.errMsg = ""
	root := m.root
	replacer := m.replacer

	return func() tea.Msg {
		groups, err := replacer.Preview(root, query)
		return globalReplacePreviewMsg{
			Query:  query,
			Groups: groups,
			Err:    err,
		}
	}
}

func (m *GlobalReplaceModel) applyPreview(msg globalReplacePreviewMsg) {
	if msg.Query != m.findInput.Value() {
		return
	}
	m.loading = false
	if msg.Err != nil {
		m.errMsg = msg.Err.Error()
		m.groups = nil
		return
	}

	prevSelected := m.selected
	nextSelected := make(map[string]bool, len(msg.Groups))
	for _, g := range msg.Groups {
		if val, existed := prevSelected[g.Path]; existed {
			nextSelected[g.Path] = val
		} else {
			nextSelected[g.Path] = true
		}
	}
	m.groups = msg.Groups
	m.selected = nextSelected
	if m.cursor >= len(m.groups) {
		m.cursor = 0
	}
	m.scrollOffset = 0
}

func (m *GlobalReplaceModel) toggleCurrentFile() {
	if m.cursor < 0 || m.cursor >= len(m.groups) {
		return
	}
	p := m.groups[m.cursor].Path
	m.selected[p] = !m.selected[p]
}

func (m *GlobalReplaceModel) toggleAllFiles() {
	if len(m.groups) == 0 {
		return
	}
	allSelected := true
	for _, g := range m.groups {
		if !m.selected[g.Path] {
			allSelected = false
			break
		}
	}
	target := !allSelected
	for _, g := range m.groups {
		m.selected[g.Path] = target
	}
}

func (m *GlobalReplaceModel) maxVisibleGroups() int {
	// Each group takes 2 lines (file header + first match preview)
	availLines := m.height - 11
	if availLines < 4 {
		availLines = 4
	}
	maxGroups := availLines / 2
	if maxGroups < 2 {
		maxGroups = 2
	}
	return maxGroups
}

func (m *GlobalReplaceModel) moveCursor(delta int) {
	if len(m.groups) == 0 {
		m.cursor = 0
		m.scrollOffset = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.groups) {
		m.cursor = len(m.groups) - 1
	}
	maxVis := m.maxVisibleGroups()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+maxVis {
		m.scrollOffset = m.cursor - maxVis + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// Execute applies the replacement to all selected files within the project root.
func (m GlobalReplaceModel) Execute() (search.GlobalReplaceResult, error) {
	return m.replacer.Apply(m.root, m.findInput.Value(), m.replaceInput.Value(), m.selected)
}

// Update handles messages for the GlobalReplaceModel.
// Returns (updated model, cmd, executed, closed).
func (m GlobalReplaceModel) Update(msg tea.Msg) (GlobalReplaceModel, tea.Cmd, bool, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil, false, false

	case globalReplacePreviewMsg:
		m.applyPreview(msg)
		return m, nil, false, false

	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "enter", "y", "Y":
				m.confirming = false
				return m, nil, true, false
			case "esc", "n", "N":
				m.confirming = false
				return m, nil, false, false
			}
			return m, nil, false, false
		}

		switch msg.Type {
		case tea.KeyEsc:
			return m, nil, false, true

		case tea.KeyTab:
			switch m.focus {
			case grFocusFind:
				m.setFocus(grFocusReplace)
			case grFocusReplace:
				if len(m.groups) > 0 {
					m.setFocus(grFocusList)
				} else {
					m.setFocus(grFocusFind)
				}
			case grFocusList:
				m.setFocus(grFocusFind)
			}
			return m, nil, false, false

		case tea.KeyShiftTab:
			switch m.focus {
			case grFocusFind:
				if len(m.groups) > 0 {
					m.setFocus(grFocusList)
				} else {
					m.setFocus(grFocusReplace)
				}
			case grFocusReplace:
				m.setFocus(grFocusFind)
			case grFocusList:
				m.setFocus(grFocusReplace)
			}
			return m, nil, false, false

		case tea.KeyUp:
			m.moveCursor(-1)
			return m, nil, false, false

		case tea.KeyDown:
			m.moveCursor(1)
			return m, nil, false, false

		case tea.KeyCtrlA:
			m.toggleAllFiles()
			return m, nil, false, false

		case tea.KeyEnter:
			if m.focus == grFocusFind {
				m.setFocus(grFocusReplace)
				return m, nil, false, false
			}
			selFiles, selMatches := m.SelectedCounts()
			if m.findInput.Value() != "" && m.findInput.Value() != m.replaceInput.Value() && selFiles > 0 && selMatches > 0 {
				m.confirming = true
			}
			return m, nil, false, false
		}

		if m.focus == grFocusList {
			switch msg.String() {
			case "j":
				m.moveCursor(1)
			case "k":
				m.moveCursor(-1)
			case " ":
				m.toggleCurrentFile()
			case "a":
				m.toggleAllFiles()
			}
			return m, nil, false, false
		}

		var cmd tea.Cmd
		if m.focus == grFocusFind {
			oldQuery := m.findInput.Value()
			m.findInput, cmd = m.findInput.Update(msg)
			if m.findInput.Value() != oldQuery {
				return m, m.performPreview(), false, false
			}
			return m, cmd, false, false
		}

		if m.focus == grFocusReplace {
			m.replaceInput, cmd = m.replaceInput.Update(msg)
			return m, cmd, false, false
		}
	}

	return m, nil, false, false
}

func (m GlobalReplaceModel) View(innerWidth int) string {
	if innerWidth < 30 {
		innerWidth = 30
	}

	p := activePalette
	bg := modalStyle.GetBackground()

	badgeActiveBg := p.Accent
	badgeInactiveBg := p.Surface
	badgeFg := p.AccentFg
	badgeInactiveFg := p.SubtleFg
	mutedFg := p.MutedFg
	highlightBg := p.Selection
	highlightFg := p.SelectionFg
	oldMatchFg := p.Red
	newMatchFg := p.Green
	confirmFg := p.Yellow
	fileFg := p.Cyan

	findBadgeStyle := lipgloss.NewStyle().Background(badgeInactiveBg).Foreground(badgeInactiveFg).Bold(true).Width(9).Align(lipgloss.Center)
	repBadgeStyle := lipgloss.NewStyle().Background(badgeInactiveBg).Foreground(badgeInactiveFg).Bold(true).Width(9).Align(lipgloss.Center)
	if m.focus == grFocusFind {
		findBadgeStyle = findBadgeStyle.Background(badgeActiveBg).Foreground(badgeFg)
	} else if m.focus == grFocusReplace {
		repBadgeStyle = repBadgeStyle.Background(badgeActiveBg).Foreground(badgeFg)
	}

	mutedStyle := lipgloss.NewStyle().Foreground(mutedFg).Background(bg)
	fileStyle := lipgloss.NewStyle().Foreground(fileFg).Background(bg).Bold(true)
	oldStyle := lipgloss.NewStyle().Foreground(oldMatchFg).Background(bg)
	newStyle := lipgloss.NewStyle().Foreground(newMatchFg).Background(bg)
	confirmStyle := lipgloss.NewStyle().Foreground(confirmFg).Background(bg).Bold(true)

	inputW := innerWidth - 14
	if inputW < 10 {
		inputW = 10
	}
	m.findInput.Width = inputW
	m.replaceInput.Width = inputW

	rootName := filepath.Base(m.root)
	if rootName == "" || rootName == "." {
		rootName = m.root
	}

	var lines []string

	// Input rows
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, findBadgeStyle.Render("FIND"), m.findInput.View()))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, repBadgeStyle.Render("REPLACE"), m.replaceInput.View()))
	lines = append(lines, "")

	// Summary line
	totalFiles, totalMatches := m.TotalCounts()
	selFiles, selMatches := m.SelectedCounts()
	var summary string
	switch {
	case m.errMsg != "":
		summary = oldStyle.Render("Error: " + m.errMsg)
	case m.findInput.Value() == "":
		summary = mutedStyle.Render(fmt.Sprintf("Scope: %s/  ·  Type to search across project", rootName))
	case m.loading:
		summary = mutedStyle.Render("Searching project files...")
	case totalFiles == 0:
		summary = mutedStyle.Render("No matches found in project.")
	default:
		summary = mutedStyle.Render(fmt.Sprintf(
			"Scope: %s/  ·  %d matches in %d files (%d/%d selected · %d to replace)",
			rootName, totalMatches, totalFiles, selFiles, totalFiles, selMatches,
		))
	}
	lines = append(lines, summary)
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", innerWidth)))

	// Results list
	maxVis := m.maxVisibleGroups()
	renderedRows := 0
	query := m.findInput.Value()
	replacement := m.replaceInput.Value()

	if len(m.groups) > 0 {
		end := m.scrollOffset + maxVis
		if end > len(m.groups) {
			end = len(m.groups)
		}
		for i := m.scrollOffset; i < end; i++ {
			g := m.groups[i]
			isCurr := i == m.cursor

			check := "[ ]"
			if m.selected[g.Path] {
				check = "[x]"
			}
			pointer := "  "
			if isCurr {
				pointer = "» "
			}

			headerText := fmt.Sprintf("%s%s %s (%d)", pointer, check, g.DisplayPath, g.TotalCount)
			headerText = truncateDisplay(headerText, innerWidth)
			if isCurr {
				rowStyle := lipgloss.NewStyle().Foreground(highlightFg).Background(highlightBg).Bold(true)
				lines = append(lines, rowStyle.Width(innerWidth).Render(headerText))
			} else {
				lines = append(lines, fileStyle.Render(headerText))
			}
			renderedRows++

			// First matching line preview with before -> after
			if len(g.Matches) > 0 {
				first := g.Matches[0]
				prefix := fmt.Sprintf("      L%d: ", first.Line)
				avail := innerWidth - runewidth.StringWidth(prefix)
				if avail < 10 {
					avail = 10
				}

				if query != "" && query != replacement {
					previewLine := renderDiffLine(first.Content, query, replacement, avail, oldStyle, newStyle, mutedStyle)
					lines = append(lines, mutedStyle.Render(prefix)+previewLine)
				} else {
					lines = append(lines, mutedStyle.Render(prefix+truncateDisplay(first.Content, avail)))
				}
				renderedRows++
			}
		}
	}

	targetListRows := maxVis * 2
	for renderedRows < targetListRows {
		lines = append(lines, "")
		renderedRows++
	}

	// Footer / confirmation bar
	lines = append(lines, mutedStyle.Render(strings.Repeat("─", innerWidth)))
	if m.confirming {
		lines = append(lines, confirmStyle.Render(fmt.Sprintf(
			"Replace %d occurrences in %d files?  [Enter/y] Confirm  ·  [Esc/n] Cancel",
			selMatches, selFiles,
		)))
	} else if m.focus == grFocusList {
		lines = append(lines, mutedStyle.Render("↑/↓ Navigate · Space Toggle File · a Toggle All · Enter Replace · Tab Fields · Esc Close"))
	} else {
		lines = append(lines, mutedStyle.Render("Tab Switch Field/List · ↑/↓ Preview · Ctrl+A Toggle All · Enter Continue/Replace · Esc Close"))
	}

	return strings.Join(lines, "\n")
}

func renderDiffLine(line, query, replacement string, maxWidth int, oldStyle, newStyle, mutedStyle lipgloss.Style) string {
	idx := strings.Index(line, query)
	if idx < 0 {
		return mutedStyle.Render(truncateDisplay(line, maxWidth))
	}

	before := line[:idx]
	after := line[idx+len(query):]

	repDisplay := replacement
	if repDisplay == "" {
		repDisplay = "∅"
	}

	// Keep context tight around the match so the change is always visible
	if runewidth.StringWidth(before) > 18 {
		runes := []rune(before)
		if len(runes) > 15 {
			before = "…" + string(runes[len(runes)-15:])
		}
	}

	plainCore := before + query + " → " + repDisplay + after
	if runewidth.StringWidth(plainCore) > maxWidth {
		coreUsed := runewidth.StringWidth(before) + runewidth.StringWidth(query) + 3 + runewidth.StringWidth(repDisplay)
		rem := maxWidth - coreUsed
		if rem > 0 {
			after = truncateDisplay(after, rem)
		} else {
			after = ""
		}
	}

	return mutedStyle.Render(before) +
		oldStyle.Render(query) +
		mutedStyle.Render(" → ") +
		newStyle.Render(repDisplay) +
		mutedStyle.Render(after)
}
