package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewWelcome() string {
	width := m.Width
	height := m.Height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
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
		versionColor   lipgloss.Color
	)

	if isDark {
		accentColor = lipgloss.Color("170")    // purple/magenta
		catColor = lipgloss.Color("252")       // bright white for the cat
		dimColor = lipgloss.Color("240")       // dim gray
		subtleColor = lipgloss.Color("245")    // subtle gray
		keyColor = lipgloss.Color("118")       // green for keys
		descFgColor = lipgloss.Color("250")    // light gray
		separatorColor = lipgloss.Color("238") // dark separator
		taglineColor = lipgloss.Color("248")   // soft white
		versionColor = lipgloss.Color("241")   // muted
	} else {
		accentColor = lipgloss.Color("125")    // magenta
		catColor = lipgloss.Color("238")       // dark for the cat
		dimColor = lipgloss.Color("244")       // gray
		subtleColor = lipgloss.Color("240")    // darker gray
		keyColor = lipgloss.Color("22")        // green for keys
		descFgColor = lipgloss.Color("238")    // dark gray
		separatorColor = lipgloss.Color("252") // light separator
		taglineColor = lipgloss.Color("240")   // gray
		versionColor = lipgloss.Color("248")   // muted
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

	versionStyle := lipgloss.NewStyle().
		Foreground(versionColor)

	// ── Leader key ──
	leader := strings.Title(m.Config.LeaderKey)
	if leader == "" {
		leader = "Ctrl"
	}

	// ── ASCII Art ──
	cat := []string{
		`   /\_/\   `,
		`  ( o.o )  `,
		`   > ^ <   `,
	}

	logoText := []string{
		` ██       █████  ██████  ██████  ██    ██`,
		` ██      ██   ██ ██   ██ ██   ██  ██  ██`,
		` ██      ███████ ██████  ██████    ████  `,
		` ██      ██   ██ ██   ██ ██   ██    ██   `,
		` ███████ ██   ██ ██   ██ ██   ██    ██   `,
	}
	// ── Build content ──
	var lines []string

	// Cat on top
	for _, line := range cat {
		lines = append(lines, catStyle.Render(line))
	}

	lines = append(lines, "")

	// Logo below
	for _, line := range logoText {
		lines = append(lines, logoStyle.Render(line))
	}

	lines = append(lines, "")

	// Tagline
	lines = append(lines, taglineStyle.Render("  A minimalist, high-performance text editor"))

	lines = append(lines, "")

	// Separator
	sepWidth := 54
	if width < sepWidth+10 {
		sepWidth = width - 10
	}
	if sepWidth < 20 {
		sepWidth = 20
	}
	sep := separatorStyle.Render(strings.Repeat("─", sepWidth))
	lines = append(lines, sep)

	lines = append(lines, "")

	// Quick Actions section
	lines = append(lines, sectionTitleStyle.Render("  Quick Actions"))
	lines = append(lines, "")

	actions := []struct {
		Key  string
		Desc string
	}{
		{leader + "+o", "Open file"},
		{leader + "+p", "Larry Finder"},
		{leader + "+h", "Help menu"},
		{leader + "+s", "Save file"},
		{leader + "+q", "Quit editor"},
	}

	for _, a := range actions {
		keyRendered := actionKeyStyle.Render(fmt.Sprintf("  %-10s", a.Key))
		descRendered := actionDescStyle.Render(a.Desc)
		lines = append(lines, keyRendered+" "+descRendered)
	}

	lines = append(lines, "")
	lines = append(lines, sep)
	lines = append(lines, "")

	// Hint
	lines = append(lines, hintStyle.Render("  Start typing to begin, or use a shortcut above"))
	lines = append(lines, "")

	// Version
	lines = append(lines, versionStyle.Render("  larry v0.1.0"))

	content := strings.Join(lines, "\n")

	// Center the whole block in the terminal
	return lipgloss.Place(
		width, height-1, // -1 for the status bar
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
