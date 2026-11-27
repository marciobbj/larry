package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const AppVersion = "v0.1.0"

func (m Model) viewWelcome() string {
	width := m.mainContentWidth()
	height := m.editorViewportHeight()
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 23
	}

	isDark := lipgloss.HasDarkBackground()

	// ── Color palette ──
	var (
		accentColor    lipgloss.Color
		catColor       lipgloss.Color
		dimColor       lipgloss.Color
		subtleColor    lipgloss.Color
		keyColor       lipgloss.Color
		descFgColor    lipgloss.Color
		separatorColor lipgloss.Color
		taglineColor   lipgloss.Color
		versionBg      lipgloss.Color
		versionFg      lipgloss.Color
	)

	if isDark {
		accentColor = lipgloss.Color("170")    // purple/magenta
		catColor = lipgloss.Color("252")       // bright white for the cat
		dimColor = lipgloss.Color("242")       // dim gray
		subtleColor = lipgloss.Color("246")    // subtle gray
		keyColor = lipgloss.Color("118")       // green for keys
		descFgColor = lipgloss.Color("252")    // light gray
		separatorColor = lipgloss.Color("238") // dark separator
		taglineColor = lipgloss.Color("249")   // soft white
		versionBg = lipgloss.Color("237")      // subtle pill bg
		versionFg = lipgloss.Color("170")      // accent version text
	} else {
		accentColor = lipgloss.Color("125")    // magenta
		catColor = lipgloss.Color("238")       // dark for the cat
		dimColor = lipgloss.Color("244")       // gray
		subtleColor = lipgloss.Color("240")    // darker gray
		keyColor = lipgloss.Color("22")        // green for keys
		descFgColor = lipgloss.Color("236")    // dark gray
		separatorColor = lipgloss.Color("251") // light separator
		taglineColor = lipgloss.Color("240")   // gray
		versionBg = lipgloss.Color("252")      // subtle pill bg
		versionFg = lipgloss.Color("125")      // accent version text
	}

	// ── Styles ──
	logoStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)

	catStyle := lipgloss.NewStyle().
		Foreground(catColor)

	taglineStyle := lipgloss.NewStyle().
		Foreground(taglineColor).
		Italic(true)

	versionStyle := lipgloss.NewStyle().
		Foreground(versionFg).
		Background(versionBg).
		Bold(true).
		Padding(0, 1)

	separatorStyle := lipgloss.NewStyle().
		Foreground(separatorColor)

	sectionTitleStyle := lipgloss.NewStyle().
		Foreground(subtleColor).
		Bold(true)

	actionKeyStyle := lipgloss.NewStyle().
		Foreground(keyColor).
		Bold(true)

	actionDescStyle := lipgloss.NewStyle().
		Foreground(descFgColor)

	hintStyle := lipgloss.NewStyle().
		Foreground(dimColor).
		Italic(true)

	// ── Leader key ──
	leader := strings.Title(m.Config.LeaderKey)
	if leader == "" {
		leader = "Ctrl"
	}

	// ── Uniform block width ──
	blockWidth := 50
	if width-4 < blockWidth {
		blockWidth = width - 4
	}
	if blockWidth < 20 {
		blockWidth = width
	}

	centerLine := func(s string) string {
		return lipgloss.NewStyle().Width(blockWidth).Align(lipgloss.Center).Render(s)
	}

	// ── Symmetrical Cat ASCII Art (exact 7-char lines) ──
	catLines := []string{
		` /\_/\ `,
		`( o.o )`,
		` > ^ < `,
	}

	// ── Exact 40-char LARRY Logo ASCII Art ──
	logoLines := []string{
		`██       █████  ██████  ██████  ██    ██`,
		`██      ██   ██ ██   ██ ██   ██  ██  ██ `,
		`██      ███████ ██████  ██████    ████  `,
		`██      ██   ██ ██   ██ ██   ██    ██   `,
		`███████ ██   ██ ██   ██ ██   ██    ██   `,
	}

	// ── Aligned Quick Actions Table ──
	actions := []struct {
		Key  string
		Desc string
	}{
		{leader + "+o", "Toggle file tree"},
		{leader + "+p", "Larry Finder"},
		{leader + "+h", "Help menu"},
		{leader + "+s", "Save file"},
		{leader + "+q", "Quit editor"},
	}

	maxKeyW := 0
	maxDescW := 0
	for _, a := range actions {
		if kw := runewidth.StringWidth(a.Key); kw > maxKeyW {
			maxKeyW = kw
		}
		if dw := runewidth.StringWidth(a.Desc); dw > maxDescW {
			maxDescW = dw
		}
	}

	const colGap = "   "
	actionRows := make([]string, 0, len(actions))
	for _, a := range actions {
		keyPadded := a.Key + strings.Repeat(" ", maxKeyW-runewidth.StringWidth(a.Key))
		descPadded := a.Desc + strings.Repeat(" ", maxDescW-runewidth.StringWidth(a.Desc))
		row := actionKeyStyle.Render(keyPadded) + colGap + actionDescStyle.Render(descPadded)
		actionRows = append(actionRows, centerLine(row))
	}

	// ── Separator ──
	sepWidth := blockWidth - 4
	if sepWidth < 16 {
		sepWidth = blockWidth
	}
	sep := centerLine(separatorStyle.Render(strings.Repeat("─", sepWidth)))

	compact := height < 27
	var lines []string

	// Cat on top
	for _, line := range catLines {
		lines = append(lines, centerLine(catStyle.Render(line)))
	}

	if !compact {
		lines = append(lines, centerLine(""))
	}

	// LARRY logo always centered (falls back only if pane is narrower than the 40-col art)
	if blockWidth >= 40 {
		for _, line := range logoLines {
			lines = append(lines, centerLine(logoStyle.Render(line)))
		}
	} else {
		lines = append(lines, centerLine(logoStyle.Render("L A R R Y")))
	}

	lines = append(lines, centerLine(""))

	// Tagline & Version badge
	tagline := "A minimalist, high-performance text editor"
	if runewidth.StringWidth(tagline) > blockWidth {
		tagline = "Minimalist TUI text editor"
	}
	lines = append(lines, centerLine(taglineStyle.Render(tagline)))
	lines = append(lines, centerLine(versionStyle.Render("larry "+AppVersion)))

	if !compact {
		lines = append(lines, centerLine(""))
	}
	lines = append(lines, sep)
	if !compact {
		lines = append(lines, centerLine(""))
	}

	// Quick Actions section
	lines = append(lines, centerLine(sectionTitleStyle.Render("Quick Actions")))
	if !compact {
		lines = append(lines, centerLine(""))
	}
	lines = append(lines, actionRows...)

	if !compact {
		lines = append(lines, centerLine(""))
	}
	lines = append(lines, sep)
	if !compact {
		lines = append(lines, centerLine(""))
	}

	// Hint
	hint := "Start typing to begin, or use a shortcut above"
	if runewidth.StringWidth(hint) > blockWidth {
		hint = "Start typing or use a shortcut"
	}
	lines = append(lines, centerLine(hintStyle.Render(hint)))

	if len(lines) > height {
		lines = lines[:height]
	}

	content := strings.Join(lines, "\n")

	// Center the uniform-width block cleanly in the available pane
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
