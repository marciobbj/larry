package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var AppVersion = "v1.2.0"

func (m Model) viewWelcome() string {
	width := m.mainContentWidth()
	height := m.editorViewportHeight()
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 23
	}

	p := activePalette

	// ── Color palette ──
	accentColor := p.AccentAlt
	catColor := p.Foreground
	dimColor := p.MutedFg
	subtleColor := p.SubtleFg
	keyColor := p.Green
	descFgColor := p.Foreground
	separatorColor := p.Border
	taglineColor := p.SubtleFg
	versionBg := p.Surface
	versionFg := p.AccentAlt
	if omarchyActive {
		accentColor = p.Accent
		versionFg = p.Accent
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
	leader := displayLeaderKey(m.Config.LeaderKey)

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
