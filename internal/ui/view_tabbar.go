package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

func (m Model) showTabBar() bool {
	return !m.showWelcome && len(m.tabs) > 0
}

func (m Model) tabBarHeight() int {
	if m.showTabBar() {
		return 1
	}
	return 0
}

func (m Model) workspaceHeight() int {
	h := m.Height - m.statusBarHeight()
	if m.saving || m.goToLine || m.searching || m.replacing {
		h -= 2
	}
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) renderTabBar(width int) string {
	if width <= 0 || !m.showTabBar() {
		return ""
	}

	p := activePalette
	labels := m.tabDisplayLabels()

	barBg := p.Surface
	activeStyle := lipgloss.NewStyle().
		Background(p.Accent).
		Foreground(p.AccentFg).
		Bold(true)
	inactiveStyle := lipgloss.NewStyle().
		Background(barBg).
		Foreground(p.SubtleFg)
	sepStyle := lipgloss.NewStyle().
		Background(barBg).
		Foreground(p.Border)
	counterStyle := lipgloss.NewStyle().
		Background(barBg).
		Foreground(p.MutedFg)
	overflowStyle := lipgloss.NewStyle().
		Background(barBg).
		Foreground(p.Accent).
		Bold(true)
	fillStyle := lipgloss.NewStyle().
		Background(barBg)

	type renderedTab struct {
		text  string
		width int
	}

	maxTabTextW := 24
	if width < 40 {
		maxTabTextW = 14
	}

	items := make([]renderedTab, len(m.tabs))
	for i, tab := range m.tabs {
		modified := tab.Modified
		if i == m.activeTab {
			modified = m.Modified
		}
		label := truncateDisplay(labels[i], maxTabTextW)
		raw := " " + label + " "
		if modified {
			raw = " " + label + " ● "
		}
		var styled string
		if i == m.activeTab {
			styled = activeStyle.Render(raw)
		} else {
			styled = inactiveStyle.Render(raw)
		}
		items[i] = renderedTab{
			text:  styled,
			width: runewidth.StringWidth(raw),
		}
	}

	// Optional compact counter on the right when multiple tabs are open
	counterText := ""
	counterW := 0
	if len(m.tabs) > 1 && width >= 30 {
		counterRaw := fmt.Sprintf(" %d/%d ", m.activeTab+1, len(m.tabs))
		counterText = counterStyle.Render(counterRaw)
		counterW = runewidth.StringWidth(counterRaw)
	}

	availW := width - counterW
	if availW < 1 {
		availW = width
		counterText = ""
		counterW = 0
	}

	// Determine visible slice of tabs keeping m.activeTab always in view
	start := m.activeTab
	end := m.activeTab
	used := items[m.activeTab].width

	// Reserve room for potential left/right overflow arrows (" ‹ " / " › ")
	for {
		expanded := false
		// Try adding next tab to the right
		if end+1 < len(items) {
			extra := 1 + items[end+1].width // 1 for divider
			reserve := 0
			if start > 0 {
				reserve += 2
			}
			if end+2 < len(items) {
				reserve += 2
			}
			if used+extra+reserve <= availW {
				end++
				used += extra
				expanded = true
			}
		}
		// Try adding previous tab to the left
		if start > 0 {
			extra := 1 + items[start-1].width
			reserve := 0
			if start-1 > 0 {
				reserve += 2
			}
			if end+1 < len(items) {
				reserve += 2
			}
			if used+extra+reserve <= availW {
				start--
				used += extra
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}

	var b strings.Builder
	renderedW := 0

	if start > 0 {
		b.WriteString(overflowStyle.Render("‹ "))
		renderedW += 2
	}

	sep := sepStyle.Render("│")
	for i := start; i <= end; i++ {
		if i > start {
			b.WriteString(sep)
			renderedW++
		}
		// Guard against a single tab wider than availW on very narrow terminals
		if renderedW+items[i].width > availW && i == start && start == end {
			clipW := availW - renderedW
			if clipW < 1 {
				clipW = 1
			}
			raw := padOrTruncate(" "+labels[i]+" ", clipW)
			b.WriteString(activeStyle.Render(raw))
			renderedW += clipW
			continue
		}
		b.WriteString(items[i].text)
		renderedW += items[i].width
	}

	if end+1 < len(items) && renderedW+2 <= availW {
		b.WriteString(overflowStyle.Render(" ›"))
		renderedW += 2
	}

	gap := width - renderedW - counterW
	if gap > 0 {
		b.WriteString(fillStyle.Render(strings.Repeat(" ", gap)))
	}
	if counterText != "" {
		b.WriteString(counterText)
	}

	return b.String()
}
