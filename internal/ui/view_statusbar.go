package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderStatusBar builds a styled status bar with file info on the left
// and shortcut hints on the right side.
func (m Model) renderStatusBar() string {
	width := m.Width
	if width < 20 {
		width = 20
	}

	isDark := lipgloss.HasDarkBackground()

	// ── Colors ──
	var (
		barBg        lipgloss.Color
		fileBg       lipgloss.Color
		fileFg       lipgloss.Color
		shortcutKey  lipgloss.Color
		shortcutSep  lipgloss.Color
		shortcutDesc lipgloss.Color
		modifiedFg   lipgloss.Color
	)

	if isDark {
		barBg = lipgloss.Color("236")
		fileBg = lipgloss.Color("62")
		fileFg = lipgloss.Color("255")
		shortcutKey = lipgloss.Color("170")
		shortcutSep = lipgloss.Color("240")
		shortcutDesc = lipgloss.Color("252")
		modifiedFg = lipgloss.Color("215")
	} else {
		barBg = lipgloss.Color("254")
		fileBg = lipgloss.Color("62")
		fileFg = lipgloss.Color("255")
		shortcutKey = lipgloss.Color("125")
		shortcutSep = lipgloss.Color("249")
		shortcutDesc = lipgloss.Color("236")
		modifiedFg = lipgloss.Color("166")
	}

	// ── Styles ──
	fileTabStyle := lipgloss.NewStyle().
		Background(fileBg).
		Foreground(fileFg).
		Bold(true).
		Padding(0, 1)

	modifiedStyle := lipgloss.NewStyle().
		Background(fileBg).
		Foreground(modifiedFg).
		Bold(true)

	barStyle := lipgloss.NewStyle().
		Background(barBg).
		Width(width)

	keyStyle := lipgloss.NewStyle().
		Foreground(shortcutKey).
		Background(barBg).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(shortcutDesc).
		Background(barBg)

	sepStyle := lipgloss.NewStyle().
		Foreground(shortcutSep).
		Background(barBg)

	spacerStyle := lipgloss.NewStyle().
		Background(barBg)

	// ── File section ──
	fileStatus := m.FileName
	if fileStatus == "" {
		fileStatus = "[No Name]"
	}

	fileSection := fileTabStyle.Render(" " + fileStatus + " ")
	if m.Modified {
		fileSection += modifiedStyle.Render(" ● ")
	}

	// ── If there's a custom status message, show it simply ──
	if m.statusMsg != "" {
		msgStyle := lipgloss.NewStyle().
			Foreground(shortcutDesc).
			Background(barBg).
			Padding(0, 1)
		msgRendered := msgStyle.Render(m.statusMsg)
		gap := width - lipgloss.Width(fileSection) - lipgloss.Width(msgRendered)
		if gap < 0 {
			gap = 0
		}
		return barStyle.Render(fileSection + spacerStyle.Render(strings.Repeat(" ", gap)) + msgRendered)
	}

	// ── Shortcuts section ──
	leader := strings.Title(m.Config.LeaderKey)
	if leader == "" {
		leader = "Ctrl"
	}

	type shortcut struct {
		Key  string
		Desc string
	}

	var shortcuts []shortcut

	if len(m.searchResults) > 0 {
		shortcuts = []shortcut{
			{leader + "+f", "Search"},
			{leader + "+h", "Help"},
			{leader + "+s", "Save"},
			{leader + "+q", "Quit"},
		}
	} else if isMarkdownFile(m.FileName) {
		previewLabel := "Preview"
		if m.viewMode == ViewModeSplit {
			previewLabel = "Close Preview"
		}
		shortcuts = []shortcut{
			{leader + "+u", previewLabel},
			{leader + "+h", "Help"},
			{leader + "+s", "Save"},
			{leader + "+q", "Quit"},
		}
	} else {
		shortcuts = []shortcut{
			{leader + "+o", "Open"},
			{leader + "+h", "Help"},
			{leader + "+s", "Save"},
			{leader + "+f", "Search"},
			{leader + "+p", "Finder"},
			{leader + "+q", "Quit"},
		}
	}

	sep := sepStyle.Render(" │ ")

	var parts []string
	for _, s := range shortcuts {
		part := keyStyle.Render(s.Key) + descStyle.Render(" "+s.Desc)
		parts = append(parts, part)
	}
	shortcutBar := strings.Join(parts, sep)

	// Search results indicator
	if len(m.searchResults) > 0 {
		searchInfo := descStyle.Render(fmt.Sprintf(" Search: %s (%d/%d)",
			m.searchQuery, m.currentResultIndex+1, len(m.searchResults)))
		shortcutBar = searchInfo + sep + shortcutBar
	}

	// ── Compose the bar ──
	leftWidth := lipgloss.Width(fileSection)
	rightWidth := lipgloss.Width(shortcutBar)
	gap := width - leftWidth - rightWidth - 1 // -1 for breathing room
	if gap < 1 {
		gap = 1
	}

	fullBar := fileSection + spacerStyle.Render(strings.Repeat(" ", gap)) + shortcutBar

	return barStyle.Render(fullBar)
}
