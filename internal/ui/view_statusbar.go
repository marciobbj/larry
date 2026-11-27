package ui

import (
	"fmt"
	"strings"

	"larry/internal/plugin"

	"github.com/charmbracelet/lipgloss"
)

// renderStatusBar builds a styled status bar with file info on the left
// and shortcut hints on the right side.
func (m Model) renderStatusBar() string {
	width := m.Width
	if width < 20 {
		width = 20
	}

	p := activePalette

	// ── Colors ──
	barBg := p.Surface
	fileBg := p.Accent
	fileFg := p.AccentFg
	shortcutKey := p.AccentAlt
	shortcutSep := p.Border
	shortcutDesc := p.Foreground
	modifiedFg := p.Yellow

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
	if m.fileTree.GitBranch != "" && width >= 50 {
		branchBadge := lipgloss.NewStyle().
			Background(barBg).
			Foreground(p.Cyan).
			Bold(true).
			Padding(0, 1).
			Render("⎇ " + m.fileTree.GitBranch)
		fileSection += branchBadge
	}
	if m.Config.WordWrap {
		wrapBadge := lipgloss.NewStyle().
			Background(barBg).
			Foreground(shortcutKey).
			Bold(true).
			Padding(0, 1).
			Render("WRAP")
		fileSection += wrapBadge
	}
	if m.hasMultiCursor() {
		cursorBadge := lipgloss.NewStyle().
			Background(barBg).
			Foreground(p.Green).
			Bold(true).
			Padding(0, 1).
			Render(fmt.Sprintf("%d cursors", len(m.allCursors())))
		fileSection += cursorBadge
	}
	if m.plugins != nil {
		if m.FileName != "" {
			errs, warns := m.plugins.DiagnosticCounts(normalizeAbsPath(m.FileName))
			if errs > 0 {
				fileSection += lipgloss.NewStyle().
					Background(barBg).
					Foreground(p.Red).
					Bold(true).
					Padding(0, 1).
					Render(fmt.Sprintf("● %d", errs))
			}
			if warns > 0 {
				fileSection += lipgloss.NewStyle().
					Background(barBg).
					Foreground(p.Yellow).
					Bold(true).
					Padding(0, 1).
					Render(fmt.Sprintf("▲ %d", warns))
			}
		}
		for _, badge := range m.plugins.Badges() {
			fileSection += lipgloss.NewStyle().
				Background(barBg).
				Foreground(paletteColorByName(badge.Color)).
				Bold(true).
				Padding(0, 1).
				Render(badge.Text)
		}
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

	// ── If cursor is on a line with a diagnostic, surface its message cleanly ──
	if m.plugins != nil && m.FileName != "" {
		if lineDiags := m.plugins.DiagnosticsByLine(normalizeAbsPath(m.FileName))[m.CursorRow]; len(lineDiags) > 0 {
			d := lineDiags[0]
			diagColor := p.Cyan
			prefix := "ℹ "
			switch d.Severity {
			case plugin.SeverityError:
				diagColor = p.Red
				prefix = "● "
			case plugin.SeverityWarning:
				diagColor = p.Yellow
				prefix = "▲ "
			}
			label := prefix + d.Message
			if d.Source != "" {
				label += " [" + d.Source + "]"
			}
			avail := width - lipgloss.Width(fileSection) - 3
			if avail > 10 {
				msgRendered := lipgloss.NewStyle().
					Foreground(diagColor).
					Background(barBg).
					Padding(0, 1).
					Render(truncateDisplay(label, avail))
				gap := width - lipgloss.Width(fileSection) - lipgloss.Width(msgRendered)
				if gap < 0 {
					gap = 0
				}
				return barStyle.Render(fileSection + spacerStyle.Render(strings.Repeat(" ", gap)) + msgRendered)
			}
		}
	}

	// ── Shortcuts section ──
	leader := displayLeaderKey(m.Config.LeaderKey)

	type shortcut struct {
		Key  string
		Desc string
	}

	var shortcuts []shortcut

	if m.fileTree.Visible && m.fileTree.Focused {
		shortcuts = []shortcut{
			{"Enter", "Open/Fold"},
			{"←/→", "Collapse/Expand"},
			{".", "Hidden"},
			{"Esc", "Editor"},
			{leader + "+o", "Hide"},
		}
	} else if len(m.searchResults) > 0 {
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
			{leader + "+o", "Tree"},
			{leader + "+h", "Help"},
			{leader + "+s", "Save"},
			{leader + "+q", "Quit"},
		}
	} else if m.fileTree.Visible {
		shortcuts = []shortcut{
			{leader + "+o", "Hide Tree"},
			{leader + "+e", "Focus Tree"},
			{leader + "+s", "Save"},
			{leader + "+f", "Search"},
			{leader + "+p", "Finder"},
			{leader + "+q", "Quit"},
		}
	} else {
		shortcuts = []shortcut{
			{leader + "+o", "Tree"},
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

func (m Model) statusBarHeight() int {
	height := lipgloss.Height(m.renderStatusBar())
	if height < 1 {
		return 1
	}
	return height
}

func (m Model) editorViewportHeight() int {
	viewportHeight := m.workspaceHeight() - m.tabBarHeight()
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	return viewportHeight
}
