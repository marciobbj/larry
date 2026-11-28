package ui

import (
	"fmt"
	"strings"

	"larry/internal/plugin"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// PickerEntry is an entry displayed inside Larry's Command Palette or a plugin picker.
type PickerEntry struct {
	ID       string
	Label    string
	Detail   string
	Category string
	Shortcut string
	Path     string
	Line     int
	Col      int
	Item     plugin.PickerItem
}

// PickerModal renders a centered, fuzzy-searchable list modal (used for both
// the built-in Command Palette and plugin-opened pickers via larry.ui.show_picker).
type PickerModal struct {
	Title      string
	Placeholder string
	Input      textinput.Model
	AllItems   []PickerEntry
	Filtered   []PickerEntry
	Cursor     int
	Scroll     int
	OnSelect   func(PickerEntry)
	IsPalette  bool
}

func NewPickerModal(title, placeholder string, items []PickerEntry, isPalette bool, onSelect func(PickerEntry)) PickerModal {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = "❯ "
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(activePalette.AccentAlt).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(activePalette.Foreground)

	pm := PickerModal{
		Title:       title,
		Placeholder: placeholder,
		Input:       ti,
		AllItems:    items,
		OnSelect:    onSelect,
		IsPalette:   isPalette,
	}
	pm.applyFilter()
	return pm
}

func (p *PickerModal) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(p.Input.Value()))
	if query == "" {
		p.Filtered = append([]PickerEntry(nil), p.AllItems...)
	} else {
		terms := strings.Fields(query)
		out := make([]PickerEntry, 0, len(p.AllItems))
		for _, item := range p.AllItems {
			haystack := strings.ToLower(item.Category + " " + item.Label + " " + item.Detail + " " + item.ID + " " + item.Shortcut)
			matched := true
			for _, term := range terms {
				if !strings.Contains(haystack, term) {
					matched = false
					break
				}
			}
			if matched {
				out = append(out, item)
			}
		}
		p.Filtered = out
	}
	if p.Cursor >= len(p.Filtered) {
		p.Cursor = len(p.Filtered) - 1
	}
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	p.Scroll = 0
}

func (p PickerModal) Update(msg tea.Msg) (PickerModal, tea.Cmd, *PickerEntry, bool) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyEsc:
			return p, nil, nil, true
		case tea.KeyEnter:
			if len(p.Filtered) > 0 && p.Cursor >= 0 && p.Cursor < len(p.Filtered) {
				chosen := p.Filtered[p.Cursor]
				return p, nil, &chosen, true
			}
			return p, nil, nil, true
		case tea.KeyUp:
			if p.Cursor > 0 {
				p.Cursor--
			}
			return p, nil, nil, false
		case tea.KeyDown:
			if p.Cursor < len(p.Filtered)-1 {
				p.Cursor++
			}
			return p, nil, nil, false
		case tea.KeyPgUp:
			p.Cursor -= 6
			if p.Cursor < 0 {
				p.Cursor = 0
			}
			return p, nil, nil, false
		case tea.KeyPgDown:
			p.Cursor += 6
			if p.Cursor >= len(p.Filtered) {
				p.Cursor = len(p.Filtered) - 1
			}
			if p.Cursor < 0 {
				p.Cursor = 0
			}
			return p, nil, nil, false
		}
	}

	prevVal := p.Input.Value()
	var cmd tea.Cmd
	p.Input, cmd = p.Input.Update(msg)
	if p.Input.Value() != prevVal {
		p.applyFilter()
	}
	return p, cmd, nil, false
}

func (p *PickerModal) Render(totalW, totalH int) string {
	outerW := totalW - 6
	if outerW > 92 {
		outerW = 92
	}
	if outerW < 40 {
		outerW = totalW - 2
	}
	innerW := outerW - 6
	if innerW < 30 {
		innerW = 30
	}

	outerH := totalH - 4
	if outerH > 24 {
		outerH = 24
	}
	if outerH < 10 {
		outerH = totalH - 2
	}
	maxListRows := outerH - 8
	if maxListRows < 3 {
		maxListRows = 3
	}

	pal := activePalette
	bg := modalStyle.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	ruleStyle := lipgloss.NewStyle().Foreground(pal.Border).Background(bg)
	catStyle := lipgloss.NewStyle().Foreground(pal.Cyan).Background(bg).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(pal.Foreground).Background(bg)
	detailStyle := lipgloss.NewStyle().Foreground(pal.MutedFg).Background(bg)
	keyPillStyle := lipgloss.NewStyle().Foreground(pal.Green).Background(pal.Surface).Bold(true)
	selBg := pal.Accent
	selFg := pal.AccentFg
	selStyle := lipgloss.NewStyle().Background(selBg).Foreground(selFg).Bold(true)
	selDetailStyle := lipgloss.NewStyle().Background(selBg).Foreground(selFg)

	// Header
	countStr := fmt.Sprintf("%d/%d", len(p.Filtered), len(p.AllItems))
	titleRendered := modalTitleStyle.Align(lipgloss.Left).Render(p.Title)
	countRendered := detailStyle.Render(countStr)
	hGap := innerW - lipgloss.Width(titleRendered) - lipgloss.Width(countRendered)
	if hGap < 1 {
		hGap = 1
	}
	header := titleRendered + bgFill.Render(strings.Repeat(" ", hGap)) + countRendered

	// Input row
	p.Input.Width = innerW - 4
	inputRow := p.Input.View()
	divider := ruleStyle.Render(strings.Repeat("─", innerW))

	// Adjust scroll window
	if p.Cursor < p.Scroll {
		p.Scroll = p.Cursor
	}
	if p.Cursor >= p.Scroll+maxListRows {
		p.Scroll = p.Cursor - maxListRows + 1
	}
	if p.Scroll < 0 {
		p.Scroll = 0
	}

	var listRows []string
	if len(p.Filtered) == 0 {
		listRows = append(listRows, detailStyle.Render("  No matching items"))
	} else {
		end := p.Scroll + maxListRows
		if end > len(p.Filtered) {
			end = len(p.Filtered)
		}
		for i := p.Scroll; i < end; i++ {
			item := p.Filtered[i]
			selected := i == p.Cursor

			var leftText strings.Builder
			if item.Category != "" {
				leftText.WriteString("[" + item.Category + "] ")
			}
			leftText.WriteString(item.Label)
			if item.Detail != "" {
				leftText.WriteString("  " + item.Detail)
			}

			shortcutText := ""
			if item.Shortcut != "" {
				shortcutText = " " + item.Shortcut + " "
			}
			rightW := runewidth.StringWidth(shortcutText)
			availLeft := innerW - rightW - 2
			if availLeft < 8 {
				availLeft = 8
			}

			if selected {
				txt := truncateDisplay("▸ "+leftText.String(), availLeft)
				pad := innerW - runewidth.StringWidth(txt) - rightW
				if pad < 0 {
					pad = 0
				}
				row := selStyle.Render(txt) + selDetailStyle.Render(strings.Repeat(" ", pad))
				if shortcutText != "" {
					row += selStyle.Render(shortcutText)
				}
				listRows = append(listRows, row)
			} else {
				var parts string
				usedW := 2
				parts += bgFill.Render("  ")
				if item.Category != "" {
					cat := "[" + item.Category + "] "
					parts += catStyle.Render(cat)
					usedW += runewidth.StringWidth(cat)
				}
				rem := availLeft - usedW
				if rem < 4 {
					rem = 4
				}
				lbl := truncateDisplay(item.Label, rem)
				parts += labelStyle.Render(lbl)
				usedW += runewidth.StringWidth(lbl)

				if item.Detail != "" && usedW+3 < availLeft {
					det := truncateDisplay("  "+item.Detail, availLeft-usedW)
					parts += detailStyle.Render(det)
					usedW += runewidth.StringWidth(det)
				}

				pad := innerW - usedW - rightW
				if pad < 0 {
					pad = 0
				}
				row := parts + bgFill.Render(strings.Repeat(" ", pad))
				if shortcutText != "" {
					row += keyPillStyle.Render(shortcutText)
				}
				listRows = append(listRows, row)
			}
		}
	}

	footer := detailStyle.Render("↑/↓ Navigate · Enter Select · Esc Close")

	rows := make([]string, 0, len(listRows)+5)
	rows = append(rows, header, inputRow, divider)
	rows = append(rows, listRows...)
	rows = append(rows, divider, footer)

	card := modalStyle.Render(strings.Join(rows, "\n"))
	return placeCentered(totalW, totalH, card)
}

// HoverState stores an active floating documentation / diagnostic popup.
type HoverState struct {
	Visible bool
	Title   string
	Lines   []string
	Scroll  int
}

func (h HoverState) Render(totalW, totalH int) string {
	outerW := totalW - 8
	if outerW > 84 {
		outerW = 84
	}
	if outerW < 34 {
		outerW = totalW - 2
	}
	innerW := outerW - 6
	if innerW < 24 {
		innerW = 24
	}
	maxLines := totalH - 8
	if maxLines > 18 {
		maxLines = 18
	}
	if maxLines < 4 {
		maxLines = 4
	}

	pal := activePalette
	bg := modalStyle.GetBackground()
	titleStyle := lipgloss.NewStyle().Foreground(pal.AccentAlt).Background(bg).Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(pal.Foreground).Background(bg)
	ruleStyle := lipgloss.NewStyle().Foreground(pal.Border).Background(bg)
	footerStyleLocal := lipgloss.NewStyle().Foreground(pal.MutedFg).Background(bg)

	// Wrap long hover lines to innerW
	var wrapped []string
	for _, raw := range h.Lines {
		if raw == "" {
			wrapped = append(wrapped, "")
			continue
		}
		runes := []rune(raw)
		for len(runes) > 0 {
			if len(runes) <= innerW {
				wrapped = append(wrapped, string(runes))
				break
			}
			wrapped = append(wrapped, string(runes[:innerW]))
			runes = runes[innerW:]
		}
	}

	start := h.Scroll
	if start < 0 {
		start = 0
	}
	if start > len(wrapped) {
		start = len(wrapped)
	}
	end := start + maxLines
	if end > len(wrapped) {
		end = len(wrapped)
	}

	var rows []string
	if h.Title != "" {
		rows = append(rows, titleStyle.Render(truncateDisplay(h.Title, innerW)))
		rows = append(rows, ruleStyle.Render(strings.Repeat("─", innerW)))
	}
	for _, l := range wrapped[start:end] {
		rows = append(rows, bodyStyle.Render(l))
	}
	rows = append(rows, ruleStyle.Render(strings.Repeat("─", innerW)))
	rows = append(rows, footerStyleLocal.Render("↑/↓ Scroll · Esc Close"))

	card := modalStyle.Render(strings.Join(rows, "\n"))
	return placeCentered(totalW, totalH, card)
}

// PanelState stores a plugin-controlled split output panel (right side of editor).
type PanelState struct {
	Visible bool
	Title   string
	Lines   []string
	Scroll  int
}

func (m Model) viewWithPluginPanel() string {
	totalWidth := m.mainContentWidth()
	totalHeight := m.editorViewportHeight()

	editorWidth := totalWidth * 60 / 100
	if editorWidth < 20 {
		editorWidth = totalWidth / 2
	}
	panelWidth := totalWidth - editorWidth - 1
	if panelWidth < 12 {
		panelWidth = 12
	}

	editorView := m.viewEditor(editorViewConfig{
		width:          editorWidth,
		height:         totalHeight,
		showSearchUI:   m.searching,
		hideLeftBorder: m.sidebarWidth() > 0,
	})

	pal := activePalette
	headerStyle := lipgloss.NewStyle().Foreground(pal.AccentAlt).Bold(true)
	ruleStyle := lipgloss.NewStyle().Foreground(pal.Border)
	bodyStyle := lipgloss.NewStyle().Foreground(pal.Foreground)
	hintStyle := lipgloss.NewStyle().Foreground(pal.MutedFg)

	innerW := panelWidth - 2
	if innerW < 8 {
		innerW = 8
	}

	var panelRows []string
	title := m.pluginPanel.Title
	if title == "" {
		title = "Plugin Output"
	}
	panelRows = append(panelRows, " "+headerStyle.Render(truncateDisplay(title, innerW)))
	panelRows = append(panelRows, " "+ruleStyle.Render(strings.Repeat("─", innerW)))

	bodyMax := totalHeight - 3
	if bodyMax < 1 {
		bodyMax = 1
	}
	start := m.pluginPanel.Scroll
	if start < 0 {
		start = 0
	}
	if start > len(m.pluginPanel.Lines) {
		start = len(m.pluginPanel.Lines)
	}
	end := start + bodyMax
	if end > len(m.pluginPanel.Lines) {
		end = len(m.pluginPanel.Lines)
	}
	for _, line := range m.pluginPanel.Lines[start:end] {
		panelRows = append(panelRows, " "+bodyStyle.Render(truncateDisplay(line, innerW)))
	}
	for len(panelRows) < totalHeight-1 {
		panelRows = append(panelRows, "")
	}
	panelRows = append(panelRows, " "+hintStyle.Render(truncateDisplay("Esc close panel", innerW)))

	var dividerBuilder strings.Builder
	for i := 0; i < totalHeight; i++ {
		dividerBuilder.WriteString(borderStyle.Render("│"))
		if i < totalHeight-1 {
			dividerBuilder.WriteByte('\n')
		}
	}

	editorBox := lipgloss.NewStyle().Width(editorWidth).Height(totalHeight).Render(editorView)
	panelBox := lipgloss.NewStyle().Width(panelWidth).Height(totalHeight).Render(strings.Join(panelRows, "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, editorBox, dividerBuilder.String(), panelBox)
}
