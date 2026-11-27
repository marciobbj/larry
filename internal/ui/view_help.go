package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type helpItem struct {
	Key  string
	Desc string
}

type helpSection struct {
	Title string
	Color lipgloss.Color
	Items []helpItem
}

func (m Model) viewHelpMenu(base string) string {
	width := m.Width
	height := m.Height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	p := activePalette
	bg := helpStyle.GetBackground()

	leaderKey := normalizeLeaderKey(m.Config.LeaderKey)
	leader := displayLeaderKey(leaderKey)
	secondary := "Alt"
	if leaderKey == "alt" {
		secondary = "Ctrl"
	}

	leftSections := []helpSection{
		{
			Title: "Files & Tabs",
			Color: p.Cyan,
			Items: []helpItem{
				{leader + "+s", "Save file"},
				{leader + "+q", "Quit Larry"},
				{leader + "+n", "New tab"},
				{leader + "+d", "Close tab"},
				{secondary + "+← / →", "Prev/next (Alt+[])"},
				{leader + "+l", "Toggle word wrap"},
				{leader + "+u", "Markdown preview"},
				{secondary + "+p / F1", "Command palette"},
				{leader + "+h", "Toggle help"},
			},
		},
		{
			Title: "Search & Replace",
			Color: p.Yellow,
			Items: []helpItem{
				{leader + "+f", "Find in file"},
				{leader + "+t", "Replace in file"},
				{leader + "+p", "Global finder"},
				{leader + "+w", "Global replace"},
				{leader + "+g", "Go to line"},
			},
		},
		{
			Title: "File Tree",
			Color: p.Blue,
			Items: []helpItem{
				{leader + "+o", "Toggle file tree"},
				{leader + "+e", "Focus file tree"},
				{"↑↓ / jk", "Move in tree"},
				{"Enter / →←", "Open / fold dir"},
				{". / - / []", "Hidden / up / size"},
			},
		},
	}

	rightSections := []helpSection{
		{
			Title: "Editing",
			Color: p.Green,
			Items: []helpItem{
				{leader + "+z", "Undo"},
				{leader + "+r", "Redo"},
				{leader + "+c", "Copy"},
				{leader + "+x", "Cut"},
				{leader + "+v", "Paste"},
				{leader + "+a", "Select all"},
			},
		},
		{
			Title: "Multi-Cursor",
			Color: p.Magenta,
			Items: []helpItem{
				{leader + "+k / Alt+i", "Select next below"},
				{leader + "+b", "Select all below"},
				{secondary + "+↑ / ↓", "Add cursor up/down"},
				{"Esc", "Clear extra cursors"},
			},
		},
		{
			Title: "Navigation & Selection",
			Color: p.Cyan,
			Items: []helpItem{
				{"← ↑ ↓ →", "Move cursor"},
				{"Shift+Arrow", "Select text"},
				{leader + "+← / →", "Jump word"},
				{leader + "+↑ / ↓", "Jump 5 lines"},
				{leader + "+Sh+←/→", "Select word"},
				{leader + "+Sh+↑/↓", "Select lines"},
				{"Home / End", "Line start / end"},
				{leader + "+Home/End", "File start / end"},
			},
		},
	}

	if m.plugins != nil {
		if kms := m.plugins.Keymaps(); len(kms) > 0 {
			items := make([]helpItem, 0, len(kms))
			for _, km := range kms {
				desc := km.Desc
				if desc == "" {
					desc = km.Plugin
				}
				items = append(items, helpItem{Key: km.Key, Desc: desc})
			}
			leftSections = append(leftSections, helpSection{
				Title: "Plugins",
				Color: p.Magenta,
				Items: items,
			})
		}
	}

	// Responsive card dimensions
	outerW := width - 4
	if outerW > 92 {
		outerW = 92
	}
	if outerW < 36 {
		outerW = width
	}
	innerW := outerW - 6 // 2 border + 4 horizontal padding
	if innerW < 28 {
		innerW = 28
	}

	vPad := 1
	if height <= 26 {
		vPad = 0
	}
	outerH := height - 2
	if outerH < 10 {
		outerH = height
	}
	innerH := outerH - 2 - (vPad * 2) // border + vertical padding
	if innerH < 6 {
		innerH = 6
	}

	bgFill := lipgloss.NewStyle().Background(bg)
	ruleStyle := lipgloss.NewStyle().Foreground(p.Border).Background(bg)
	badgeStyle := lipgloss.NewStyle().Background(p.Accent).Foreground(p.AccentFg).Bold(true).Padding(0, 1)
	titleTextStyle := lipgloss.NewStyle().Foreground(p.Foreground).Background(bg).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)
	leaderBadgeStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)
	keyCapStyle := lipgloss.NewStyle().Foreground(p.Green).Background(p.Background).Bold(true)
	descTextStyle := lipgloss.NewStyle().Foreground(p.SubtleFg).Background(bg)
	hintKeyStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)

	// Build body rows (2 columns when innerW >= 62, single column on narrow terminals)
	var bodyRows []string
	if innerW >= 62 {
		colGap := 3 // " │ "
		colW := (innerW - colGap) / 2
		rightColW := innerW - colGap - colW

		leftLines := renderHelpColumn(leftSections, colW, bg, keyCapStyle, descTextStyle)
		rightLines := renderHelpColumn(rightSections, rightColW, bg, keyCapStyle, descTextStyle)

		maxColRows := len(leftLines)
		if len(rightLines) > maxColRows {
			maxColRows = len(rightLines)
		}
		emptyLeft := bgFill.Render(strings.Repeat(" ", colW))
		emptyRight := bgFill.Render(strings.Repeat(" ", rightColW))
		colSep := bgFill.Render(" ") + ruleStyle.Render("│") + bgFill.Render(" ")

		bodyRows = make([]string, 0, maxColRows)
		for i := 0; i < maxColRows; i++ {
			l := emptyLeft
			if i < len(leftLines) {
				l = leftLines[i]
			}
			r := emptyRight
			if i < len(rightLines) {
				r = rightLines[i]
			}
			bodyRows = append(bodyRows, l+colSep+r)
		}
	} else {
		allSections := append(leftSections, rightSections...)
		bodyRows = renderHelpColumn(allSections, innerW, bg, keyCapStyle, descTextStyle)
	}

	// Chrome rows: header (1) + divider (1) + divider (1) + footer (1) = 4
	maxBodyRows := innerH - 4
	if maxBodyRows < 3 {
		maxBodyRows = 3
	}

	totalBodyRows := len(bodyRows)
	scrollable := totalBodyRows > maxBodyRows
	scroll := m.helpScroll
	maxScroll := totalBodyRows - maxBodyRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	visibleBody := bodyRows
	if scrollable {
		visibleBody = bodyRows[scroll : scroll+maxBodyRows]
	}

	// Header row
	leftHeader := badgeStyle.Render("LARRY") + bgFill.Render(" ") + titleTextStyle.Render("Keyboard Shortcuts")
	rightHeader := metaStyle.Render("Leader: ") + leaderBadgeStyle.Render(leader) + metaStyle.Render("  "+AppVersion)
	hGap := innerW - lipgloss.Width(leftHeader) - lipgloss.Width(rightHeader)
	if hGap < 1 {
		rightHeader = metaStyle.Render(AppVersion)
		hGap = innerW - lipgloss.Width(leftHeader) - lipgloss.Width(rightHeader)
		if hGap < 0 {
			hGap = 0
		}
	}
	headerRow := leftHeader + bgFill.Render(strings.Repeat(" ", hGap)) + rightHeader

	dividerRow := ruleStyle.Render(strings.Repeat("─", innerW))

	// Footer row
	var leftFooter string
	if scrollable {
		leftFooter = hintKeyStyle.Render("↑/↓") + metaStyle.Render(" Scroll  ·  ") +
			hintKeyStyle.Render("Esc") + metaStyle.Render(" / ") +
			hintKeyStyle.Render(leader+"+h") + metaStyle.Render(" Close")
	} else {
		leftFooter = hintKeyStyle.Render("Esc") + metaStyle.Render(" / ") +
			hintKeyStyle.Render(leader+"+h") + metaStyle.Render(" Close")
	}

	var rightFooter string
	if scrollable {
		arrows := ""
		if scroll > 0 {
			arrows += "▲"
		}
		if scroll < maxScroll {
			arrows += "▼"
		}
		if arrows != "" {
			arrows = " " + arrows
		}
		rightFooter = metaStyle.Render(fmt.Sprintf("%d–%d / %d%s", scroll+1, scroll+len(visibleBody), totalBodyRows, arrows))
	} else {
		rightFooter = metaStyle.Render("~/.config/larry/config.json")
	}

	fGap := innerW - lipgloss.Width(leftFooter) - lipgloss.Width(rightFooter)
	if fGap < 1 {
		rightFooter = ""
		fGap = innerW - lipgloss.Width(leftFooter)
		if fGap < 0 {
			fGap = 0
		}
	}
	footerRow := leftFooter + bgFill.Render(strings.Repeat(" ", fGap)) + rightFooter

	allRows := make([]string, 0, len(visibleBody)+4)
	allRows = append(allRows, headerRow, dividerRow)
	allRows = append(allRows, visibleBody...)
	allRows = append(allRows, dividerRow, footerRow)

	cardStyle := helpStyle.Copy().Padding(vPad, 2)
	helpMenu := cardStyle.Render(strings.Join(allRows, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, helpMenu)
}

func renderHelpColumn(
	sections []helpSection,
	colW int,
	bg lipgloss.TerminalColor,
	keyCapStyle, descTextStyle lipgloss.Style,
) []string {
	bgFill := lipgloss.NewStyle().Background(bg)

	// Measure widest key in this column for crisp alignment
	maxKeyW := 0
	for _, sec := range sections {
		for _, item := range sec.Items {
			if w := runewidth.StringWidth(item.Key); w > maxKeyW {
				maxKeyW = w
			}
		}
	}
	// Reserve inner 1-space padding on each side of the keycap
	keyColW := maxKeyW + 2
	maxAllowedKey := colW / 2
	if keyColW > maxAllowedKey {
		keyColW = maxAllowedKey
	}
	if keyColW < 8 {
		keyColW = 8
	}

	descColW := colW - keyColW - 2 // 2-space gutter between keycap and desc
	if descColW < 6 {
		descColW = 6
	}

	var rows []string
	for sIdx, sec := range sections {
		if sIdx > 0 {
			rows = append(rows, bgFill.Render(strings.Repeat(" ", colW)))
		}
		bar := lipgloss.NewStyle().Foreground(sec.Color).Background(bg).Bold(true).Render("▍ ")
		secTitle := lipgloss.NewStyle().Foreground(sec.Color).Background(bg).Bold(true).Render(truncateDisplay(sec.Title, colW-2))
		secUsed := 2 + runewidth.StringWidth(truncateDisplay(sec.Title, colW-2))
		secPad := colW - secUsed
		if secPad < 0 {
			secPad = 0
		}
		rows = append(rows, bar+secTitle+bgFill.Render(strings.Repeat(" ", secPad)))

		for _, item := range sec.Items {
			keyText := truncateDisplay(item.Key, keyColW-2)
			keyW := runewidth.StringWidth(keyText)
			pillInner := " " + keyText + " "
			pillW := keyW + 2
			pillPad := keyColW - pillW
			if pillPad < 0 {
				pillPad = 0
			}

			descText := truncateDisplay(item.Desc, descColW)
			descW := runewidth.StringWidth(descText)
			rowUsed := pillW + pillPad + 2 + descW
			tailPad := colW - rowUsed
			if tailPad < 0 {
				tailPad = 0
			}

			row := keyCapStyle.Render(pillInner) +
				bgFill.Render(strings.Repeat(" ", pillPad+2)) +
				descTextStyle.Render(descText) +
				bgFill.Render(strings.Repeat(" ", tailPad))
			rows = append(rows, row)
		}
	}
	return rows
}
