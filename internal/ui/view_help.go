package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewHelpMenu(base string) string {
	width := m.Width
	height := m.Height

	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	leader := strings.Title(m.Config.LeaderKey)
	if leader == "" {
		leader = "Leader"
	}

	generalShortcuts := []struct {
		Key  string
		Desc string
	}{
		{leader + "+q", "Quit"},
		{leader + "+s", "Save"},
		{leader + "+o", "Open File"},
		{leader + "+g", "Go to Line"},
		{leader + "+f", "Search"},
		{leader + "+t", "Replace"},
		{leader + "+p", "Global Finder"},
		{leader + "+h", "Toggle Help"},
		{leader + "+z", "Undo"},
		{leader + "+r", "Redo"},
		{leader + "+c", "Copy"},
		{leader + "+v", "Paste"},
		{leader + "+x", "Cut"},
		{leader + "+a", "Select All"},
		{leader + "+u", "Markdown Preview"},
	}

	navShortcuts := []struct {
		Key  string
		Desc string
	}{
		{"←/→/↑/↓", "Move Cursor"},
		{"Shift+Arrow", "Select Text"},
		{leader + "+←/→", "Jump Word"},
		{leader + "+↑/↓", "Jump 5 Lines"},
		{leader + "+Shift+←/→", "Select Word"},
		{leader + "+Shift+↑/↓", "Select Lines"},
		{"Home", "Line Start"},
		{"End", "Line End"},
		{"Shift+Home/End", "Select to Start/End"},
		{leader + "+Home", "File Start"},
		{leader + "+End", "File End"},
	}

	bg := helpStyle.GetBackground()
	spacerStyle := lipgloss.NewStyle().Background(bg)

	colWidth := (width - 20) / 2
	if colWidth < 25 {
		colWidth = 25
	}

	var leftColLines []string
	leftColLines = append(leftColLines, categoryStyle.Render("General"))
	for _, s := range generalShortcuts {
		keyStr := keyStyle.Width(14).Render(s.Key)
		descStr := descStyle.Width(colWidth - 14).Render(s.Desc)
		leftColLines = append(leftColLines, lipgloss.JoinHorizontal(lipgloss.Top, keyStr, descStr))
	}

	var rightColLines []string
	rightColLines = append(rightColLines, categoryStyle.Render("Navigation"))
	for _, s := range navShortcuts {
		keyStr := keyStyle.Width(18).Render(s.Key)
		descStr := descStyle.Width(colWidth - 18).Render(s.Desc)
		rightColLines = append(rightColLines, lipgloss.JoinHorizontal(lipgloss.Top, keyStr, descStr))
	}

	maxLines := len(leftColLines)
	if len(rightColLines) > maxLines {
		maxLines = len(rightColLines)
	}

	for len(leftColLines) < maxLines {
		leftColLines = append(leftColLines, spacerStyle.Render(""))
	}
	for len(rightColLines) < maxLines {
		rightColLines = append(rightColLines, spacerStyle.Render(""))
	}

	leftColStyled := lipgloss.NewStyle().
		PaddingRight(3).
		Background(bg).
		Render(strings.Join(leftColLines, "\n"))
	rightColStyled := lipgloss.NewStyle().
		Background(bg).
		Render(strings.Join(rightColLines, "\n"))
	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftColStyled, rightColStyled)

	contentWidth := lipgloss.Width(columns)

	title := "Larry - Help Menu"
	titleGap := contentWidth - lipgloss.Width(title)
	titleLeft := titleGap / 2
	titleRight := titleGap - titleLeft
	paddedTitle := helpTitleStyle.Render(strings.Repeat(" ", titleLeft) + title + strings.Repeat(" ", titleRight))

	footer := "Press Esc or " + leader + "+h to close"
	footerGap := contentWidth - lipgloss.Width(footer)
	footerLeft := footerGap / 2
	footerRight := footerGap - footerLeft
	paddedFooter := footerStyle.Render(strings.Repeat(" ", footerLeft) + footer + strings.Repeat(" ", footerRight))

	var sb strings.Builder
	sb.WriteString(paddedTitle)
	sb.WriteString("\n")
	sb.WriteString(spacerStyle.Render(strings.Repeat(" ", contentWidth)))
	sb.WriteString("\n")
	sb.WriteString(columns)
	sb.WriteString("\n")
	sb.WriteString(spacerStyle.Render(strings.Repeat(" ", contentWidth)))
	sb.WriteString("\n")
	sb.WriteString(paddedFooter)

	rawContent := sb.String()
	lines := strings.Split(rawContent, "\n")

	maxWidth := 0
	for _, line := range lines {
		lw := lipgloss.Width(line)
		if lw > maxWidth {
			maxWidth = lw
		}
	}

	for i, line := range lines {
		styledLine := strings.ReplaceAll(line, " ", spacerStyle.Render(" "))
		lines[i] = spacerStyle.Width(maxWidth).Render(styledLine)
	}
	helpMenu := helpStyle.Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, helpMenu)
}
