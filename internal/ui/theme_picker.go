package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// ThemePickerModal is the built-in interactive colorscheme switcher UI with
// fuzzy search, category/mode filters, truecolor palette swatches, and live preview.
type ThemePickerModal struct {
	Input         textinput.Model
	AllSchemes    []ColorScheme
	Filtered      []ColorScheme
	CategoryIdx   int // index into ColorSchemeCategories
	Cursor        int
	Scroll        int
	ActiveThemeID string // currently persisted theme ID
	OriginalTheme string // theme active before opening modal (for Esc revert)
}

func NewThemePickerModal(currentTheme string) ThemePickerModal {
	ti := textinput.New()
	ti.Placeholder = "Search colorschemes (e.g. catppuccin, nord, light, retro)..."
	ti.Prompt = "❯ "
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(activePalette.AccentAlt).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(activePalette.Foreground)

	activeID := strings.ToLower(strings.TrimSpace(currentTheme))
	if cs, ok := LookupColorScheme(activeID); ok {
		activeID = cs.ID
	}

	all := BuiltinColorSchemes()
	tp := ThemePickerModal{
		Input:         ti,
		AllSchemes:    all,
		CategoryIdx:   0,
		ActiveThemeID: activeID,
		OriginalTheme: currentTheme,
	}
	tp.applyFilter()

	// Start cursor on the currently active scheme if present.
	for i, s := range tp.Filtered {
		if s.ID == activeID {
			tp.Cursor = i
			break
		}
	}
	return tp
}

func (tp *ThemePickerModal) applyFilter() {
	cat := ColorSchemeCategories[tp.CategoryIdx]
	query := strings.ToLower(strings.TrimSpace(tp.Input.Value()))
	terms := strings.Fields(query)

	out := make([]ColorScheme, 0, len(tp.AllSchemes))
	for _, s := range tp.AllSchemes {
		switch cat {
		case "All":
		case "Dark":
			if s.Mode != "dark" {
				continue
			}
		case "Light":
			if s.Mode != "light" {
				continue
			}
		default:
			if s.Category != cat {
				continue
			}
		}

		if len(terms) > 0 {
			haystack := strings.ToLower(s.Name + " " + s.Family + " " + s.ID + " " + s.Category + " " + s.Mode + " " + s.Description)
			matched := true
			for _, term := range terms {
				if !strings.Contains(haystack, term) {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, s)
	}

	tp.Filtered = out
	if tp.Cursor >= len(tp.Filtered) {
		tp.Cursor = len(tp.Filtered) - 1
	}
	if tp.Cursor < 0 {
		tp.Cursor = 0
	}
	tp.Scroll = 0
}

// Hovered returns the currently highlighted ColorScheme, if any.
func (tp *ThemePickerModal) Hovered() (ColorScheme, bool) {
	if len(tp.Filtered) == 0 || tp.Cursor < 0 || tp.Cursor >= len(tp.Filtered) {
		return ColorScheme{}, false
	}
	return tp.Filtered[tp.Cursor], true
}

// Update processes input inside the Colorscheme Picker.
// Returns (updated, cmd, chosenScheme, closed, hoveredChanged).
func (tp ThemePickerModal) Update(msg tea.Msg) (ThemePickerModal, tea.Cmd, *ColorScheme, bool, bool) {
	prevID := ""
	if cur, ok := tp.Hovered(); ok {
		prevID = cur.ID
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.Type {
		case tea.KeyEsc:
			return tp, nil, nil, true, false
		case tea.KeyEnter:
			if chosen, ok := tp.Hovered(); ok {
				return tp, nil, &chosen, true, false
			}
			return tp, nil, nil, true, false
		case tea.KeyTab:
			tp.CategoryIdx = (tp.CategoryIdx + 1) % len(ColorSchemeCategories)
			tp.Cursor = 0
			tp.applyFilter()
			curID := ""
			if cur, ok := tp.Hovered(); ok {
				curID = cur.ID
			}
			return tp, nil, nil, false, curID != prevID
		case tea.KeyShiftTab:
			tp.CategoryIdx = (tp.CategoryIdx - 1 + len(ColorSchemeCategories)) % len(ColorSchemeCategories)
			tp.Cursor = 0
			tp.applyFilter()
			curID := ""
			if cur, ok := tp.Hovered(); ok {
				curID = cur.ID
			}
			return tp, nil, nil, false, curID != prevID
		case tea.KeyUp, tea.KeyCtrlK:
			if tp.Cursor > 0 {
				tp.Cursor--
			} else if len(tp.Filtered) > 0 {
				tp.Cursor = len(tp.Filtered) - 1
			}
			curID := ""
			if cur, ok := tp.Hovered(); ok {
				curID = cur.ID
			}
			return tp, nil, nil, false, curID != prevID
		case tea.KeyDown, tea.KeyCtrlJ:
			if len(tp.Filtered) > 0 {
				tp.Cursor = (tp.Cursor + 1) % len(tp.Filtered)
			}
			curID := ""
			if cur, ok := tp.Hovered(); ok {
				curID = cur.ID
			}
			return tp, nil, nil, false, curID != prevID
		case tea.KeyPgUp:
			tp.Cursor -= 6
			if tp.Cursor < 0 {
				tp.Cursor = 0
			}
			curID := ""
			if cur, ok := tp.Hovered(); ok {
				curID = cur.ID
			}
			return tp, nil, nil, false, curID != prevID
		case tea.KeyPgDown:
			tp.Cursor += 6
			if tp.Cursor >= len(tp.Filtered) {
				tp.Cursor = len(tp.Filtered) - 1
			}
			if tp.Cursor < 0 {
				tp.Cursor = 0
			}
			curID := ""
			if cur, ok := tp.Hovered(); ok {
				curID = cur.ID
			}
			return tp, nil, nil, false, curID != prevID
		}
	}

	prevVal := tp.Input.Value()
	var cmd tea.Cmd
	tp.Input, cmd = tp.Input.Update(msg)
	if tp.Input.Value() != prevVal {
		tp.Cursor = 0
		tp.applyFilter()
	}
	curID := ""
	if cur, ok := tp.Hovered(); ok {
		curID = cur.ID
	}
	return tp, cmd, nil, false, curID != prevID
}

// Render builds the centered two-pane Colorscheme Picker modal guaranteed to fit
// within (totalW, totalH).
func (tp *ThemePickerModal) Render(totalW, totalH int) string {
	if totalW < 36 {
		totalW = 36
	}
	if totalH < 12 {
		totalH = 12
	}

	outerW := totalW - 4
	if outerW > 106 {
		outerW = 106
	}
	if outerW < 36 {
		outerW = totalW
	}
	innerW := outerW - 6 // border (2) + horizontal padding (4)
	if innerW < 30 {
		innerW = 30
	}

	vPad := 1
	if totalH <= 24 {
		vPad = 0
	}
	outerH := totalH - 2
	if outerH > 29 {
		outerH = 29
	}
	if outerH < 12 {
		outerH = totalH
	}
	innerH := outerH - 2 - (vPad * 2)
	if innerH < 8 {
		innerH = 8
	}

	pal := activePalette
	bg := modalStyle.GetBackground()
	bgFill := lipgloss.NewStyle().Background(bg)
	ruleStyle := lipgloss.NewStyle().Foreground(pal.Border).Background(bg)
	badgeStyle := lipgloss.NewStyle().Background(pal.Accent).Foreground(pal.AccentFg).Bold(true).Padding(0, 1)
	titleStyle := lipgloss.NewStyle().Foreground(pal.Foreground).Background(bg).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(pal.MutedFg).Background(bg)
	subtleStyle := lipgloss.NewStyle().Foreground(pal.SubtleFg).Background(bg)
	hintKeyStyle := lipgloss.NewStyle().Foreground(pal.AccentAlt).Background(bg).Bold(true)

	// Refresh textinput colors to follow live-previewed theme
	tp.Input.PromptStyle = lipgloss.NewStyle().Foreground(pal.AccentAlt).Background(bg).Bold(true)
	tp.Input.TextStyle = lipgloss.NewStyle().Foreground(pal.Foreground).Background(bg)
	tp.Input.PlaceholderStyle = lipgloss.NewStyle().Foreground(pal.MutedFg).Background(bg)

	// ── Header row ──
	leftHeader := badgeStyle.Render("LARRY") + bgFill.Render(" ") + titleStyle.Render("Colorschemes")
	activeLabel := tp.ActiveThemeID
	if cs, ok := LookupColorScheme(tp.ActiveThemeID); ok {
		activeLabel = cs.Name
	}
	countStr := fmt.Sprintf("%d/%d", len(tp.Filtered), len(tp.AllSchemes))
	rightHeader := mutedStyle.Render("Active: ") +
		lipgloss.NewStyle().Foreground(pal.Cyan).Background(bg).Bold(true).Render(activeLabel) +
		mutedStyle.Render("  "+countStr)
	hGap := innerW - lipgloss.Width(leftHeader) - lipgloss.Width(rightHeader)
	if hGap < 1 {
		rightHeader = mutedStyle.Render(countStr)
		hGap = innerW - lipgloss.Width(leftHeader) - lipgloss.Width(rightHeader)
		if hGap < 0 {
			hGap = 0
		}
	}
	headerRow := leftHeader + bgFill.Render(strings.Repeat(" ", hGap)) + rightHeader

	// ── Search input row ──
	tp.Input.Width = innerW - 4
	if tp.Input.Width < 12 {
		tp.Input.Width = 12
	}
	inputView := tp.Input.View()
	inputPad := innerW - lipgloss.Width(inputView)
	if inputPad < 0 {
		inputPad = 0
	}
	inputRow := inputView + bgFill.Render(strings.Repeat(" ", inputPad))

	// ── Category filter pills row ──
	catRow := tp.renderCategoryBar(innerW, pal, bg)

	dividerRow := ruleStyle.Render(strings.Repeat("─", innerW))

	// Body height: header (1) + input (1) + cats (1) + top divider (1) + bottom divider (1) + footer (1) = 6
	bodyH := innerH - 6
	if bodyH < 4 {
		bodyH = 4
	}

	// Adjust list scroll
	if tp.Cursor < tp.Scroll {
		tp.Scroll = tp.Cursor
	}
	if tp.Cursor >= tp.Scroll+bodyH {
		tp.Scroll = tp.Cursor - bodyH + 1
	}
	if tp.Scroll < 0 {
		tp.Scroll = 0
	}

	var bodyRows []string
	if innerW >= 68 {
		leftW := innerW * 44 / 100
		if leftW < 28 {
			leftW = 28
		}
		if leftW > 42 {
			leftW = 42
		}
		rightW := innerW - leftW - 3 // " │ "
		leftLines := tp.renderListPane(leftW, bodyH, pal, bg)
		rightLines := tp.renderPreviewPane(rightW, bodyH, pal, bg)
		colSep := bgFill.Render(" ") + ruleStyle.Render("│") + bgFill.Render(" ")

		bodyRows = make([]string, bodyH)
		for i := 0; i < bodyH; i++ {
			bodyRows[i] = leftLines[i] + colSep + rightLines[i]
		}
	} else {
		bodyRows = tp.renderListPane(innerW, bodyH, pal, bg)
	}

	// ── Footer row ──
	leftFooter := hintKeyStyle.Render("↑/↓") + mutedStyle.Render(" Preview  ·  ") +
		hintKeyStyle.Render("Tab") + mutedStyle.Render(" Filter  ·  ") +
		hintKeyStyle.Render("Enter") + mutedStyle.Render(" Apply  ·  ") +
		hintKeyStyle.Render("Esc") + mutedStyle.Render(" Revert")
	if lipgloss.Width(leftFooter) > innerW {
		leftFooter = hintKeyStyle.Render("↑/↓") + mutedStyle.Render(" Move · ") +
			hintKeyStyle.Render("Tab") + mutedStyle.Render(" Cat · ") +
			hintKeyStyle.Render("Enter") + mutedStyle.Render(" Save · ") +
			hintKeyStyle.Render("Esc") + mutedStyle.Render(" Cancel")
	}
	if lipgloss.Width(leftFooter) > innerW {
		leftFooter = subtleStyle.Render(truncateDisplay("↑/↓ Move · Tab Filter · Enter Save · Esc", innerW))
	}
	fPad := innerW - lipgloss.Width(leftFooter)
	if fPad < 0 {
		fPad = 0
	}
	footerRow := leftFooter + bgFill.Render(strings.Repeat(" ", fPad))

	rows := make([]string, 0, bodyH+6)
	rows = append(rows, headerRow, inputRow, catRow, dividerRow)
	rows = append(rows, bodyRows...)
	rows = append(rows, dividerRow, footerRow)

	cardStyle := modalStyle.Copy().Padding(vPad, 2)
	card := cardStyle.Render(strings.Join(rows, "\n"))
	return lipgloss.Place(totalW, totalH, lipgloss.Center, lipgloss.Center, card)
}

func (tp *ThemePickerModal) renderCategoryBar(innerW int, pal UIPalette, bg lipgloss.TerminalColor) string {
	bgFill := lipgloss.NewStyle().Background(bg)
	activePill := lipgloss.NewStyle().Background(pal.Accent).Foreground(pal.AccentFg).Bold(true)
	inactivePill := lipgloss.NewStyle().Background(pal.Surface).Foreground(pal.SubtleFg)

	var b strings.Builder
	used := 0
	for i, cat := range ColorSchemeCategories {
		label := " " + cat + " "
		if cat == "Dark" {
			label = " ☾ Dark "
		} else if cat == "Light" {
			label = " ☀ Light "
		}
		lw := runewidth.StringWidth(label)
		sepW := 0
		if i > 0 {
			sepW = 1
		}
		if used+sepW+lw > innerW {
			break
		}
		if sepW > 0 {
			b.WriteString(bgFill.Render(" "))
			used++
		}
		if i == tp.CategoryIdx {
			b.WriteString(activePill.Render(label))
		} else {
			b.WriteString(inactivePill.Render(label))
		}
		used += lw
	}
	if used < innerW {
		b.WriteString(bgFill.Render(strings.Repeat(" ", innerW-used)))
	}
	return b.String()
}

func (tp *ThemePickerModal) renderListPane(width, height int, pal UIPalette, bg lipgloss.TerminalColor) []string {
	bgFill := lipgloss.NewStyle().Background(bg)
	emptyRow := bgFill.Render(strings.Repeat(" ", width))
	mutedStyle := lipgloss.NewStyle().Foreground(pal.MutedFg).Background(bg)

	rows := make([]string, 0, height)
	if len(tp.Filtered) == 0 {
		msg := truncateDisplay("  No matching colorschemes", width)
		pad := width - runewidth.StringWidth(msg)
		if pad < 0 {
			pad = 0
		}
		rows = append(rows, mutedStyle.Render(msg)+bgFill.Render(strings.Repeat(" ", pad)))
		for len(rows) < height {
			rows = append(rows, emptyRow)
		}
		return rows
	}

	end := tp.Scroll + height
	if end > len(tp.Filtered) {
		end = len(tp.Filtered)
	}

	selBg := pal.Accent
	selFg := pal.AccentFg

	for i := tp.Scroll; i < end; i++ {
		s := tp.Filtered[i]
		selected := i == tp.Cursor
		isActive := s.ID == tp.ActiveThemeID

		modeIcon := "☾"
		if s.Mode == "light" {
			modeIcon = "☀"
		}

		// Right-hand mini signature swatch (3 colored dots + mode icon = 6 cols)
		showSwatches := width >= 26
		rightW := 2 // " " + modeIcon
		if showSwatches {
			rightW = 6 // " ●●● ☾"
		}
		if isActive {
			rightW += 2 // " ✓"
		}

		prefix := "  "
		if selected {
			prefix = "▸ "
		}
		availName := width - 2 - rightW
		if availName < 4 {
			availName = 4
		}
		nameTxt := truncateDisplay(s.Name, availName)
		nameW := runewidth.StringWidth(nameTxt)
		pad := width - 2 - nameW - rightW
		if pad < 0 {
			pad = 0
		}

		rowBg := bg
		if selected {
			rowBg = selBg
		}
		rowFill := lipgloss.NewStyle().Background(rowBg)

		var line strings.Builder
		if selected {
			st := lipgloss.NewStyle().Background(selBg).Foreground(selFg).Bold(true)
			line.WriteString(st.Render(prefix + nameTxt))
			line.WriteString(rowFill.Render(strings.Repeat(" ", pad)))
			if isActive {
				line.WriteString(st.Render(" ✓"))
			}
			if showSwatches {
				line.WriteString(rowFill.Render(" "))
				line.WriteString(lipgloss.NewStyle().Background(selBg).Foreground(lipgloss.Color(s.Colors.Red)).Render("●"))
				line.WriteString(lipgloss.NewStyle().Background(selBg).Foreground(lipgloss.Color(s.Colors.Green)).Render("●"))
				line.WriteString(lipgloss.NewStyle().Background(selBg).Foreground(lipgloss.Color(s.Colors.Blue)).Render("●"))
			}
			line.WriteString(lipgloss.NewStyle().Background(selBg).Foreground(selFg).Render(" " + modeIcon))
		} else {
			nameSt := lipgloss.NewStyle().Background(bg).Foreground(pal.Foreground)
			if isActive {
				nameSt = nameSt.Bold(true)
			}
			line.WriteString(bgFill.Render(prefix))
			line.WriteString(nameSt.Render(nameTxt))
			line.WriteString(bgFill.Render(strings.Repeat(" ", pad)))
			if isActive {
				line.WriteString(lipgloss.NewStyle().Background(bg).Foreground(pal.Green).Bold(true).Render(" ✓"))
			}
			if showSwatches {
				line.WriteString(bgFill.Render(" "))
				line.WriteString(lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color(s.Colors.Accent)).Render("●"))
				line.WriteString(lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color(s.Colors.Green)).Render("●"))
				line.WriteString(lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color(s.Colors.Yellow)).Render("●"))
			}
			iconColor := pal.MutedFg
			if s.Mode == "light" {
				iconColor = pal.Yellow
			}
			line.WriteString(lipgloss.NewStyle().Background(bg).Foreground(iconColor).Render(" " + modeIcon))
		}

		rows = append(rows, line.String())
	}

	for len(rows) < height {
		rows = append(rows, emptyRow)
	}
	return rows
}

func (tp *ThemePickerModal) renderPreviewPane(width, height int, pal UIPalette, bg lipgloss.TerminalColor) []string {
	bgFill := lipgloss.NewStyle().Background(bg)
	emptyRow := bgFill.Render(strings.Repeat(" ", width))

	scheme, ok := tp.Hovered()
	if !ok {
		rows := make([]string, height)
		for i := range rows {
			rows[i] = emptyRow
		}
		return rows
	}

	c := scheme.Colors
	prevPal := paletteFromOmarchy(c)

	padLine := func(rendered string) string {
		w := lipgloss.Width(rendered)
		if w >= width {
			return rendered
		}
		return rendered + bgFill.Render(strings.Repeat(" ", width-w))
	}

	var rows []string

	// 1. Title + Mode pill
	modeBadgeText := " DARK "
	if scheme.Mode == "light" {
		modeBadgeText = " LIGHT "
	}
	modeBadge := lipgloss.NewStyle().
		Background(prevPal.Accent).
		Foreground(prevPal.AccentFg).
		Bold(true).
		Render(modeBadgeText)
	titleMax := width - lipgloss.Width(modeBadge) - 1
	if titleMax < 8 {
		titleMax = 8
	}
	titleTxt := lipgloss.NewStyle().
		Background(bg).
		Foreground(pal.Foreground).
		Bold(true).
		Render(truncateDisplay(scheme.Name, titleMax))
	gap := width - lipgloss.Width(titleTxt) - lipgloss.Width(modeBadge)
	if gap < 1 {
		gap = 1
	}
	rows = append(rows, titleTxt+bgFill.Render(strings.Repeat(" ", gap))+modeBadge)

	// 2. Family · Category + description
	metaTxt := scheme.Family + " · " + scheme.Category
	if scheme.ID == tp.ActiveThemeID {
		metaTxt += " · Active"
	}
	rows = append(rows, padLine(
		lipgloss.NewStyle().Background(bg).Foreground(pal.Cyan).Render(truncateDisplay(metaTxt, width)),
	))
	rows = append(rows, padLine(
		lipgloss.NewStyle().Background(bg).Foreground(pal.SubtleFg).Italic(true).Render(truncateDisplay(scheme.Description, width)),
	))
	rows = append(rows, emptyRow)

	// 3. Truecolor Palette Swatches
	swatches := []string{
		c.Background, c.LighterBackground, c.Accent,
		c.Red, c.Green, c.Yellow, c.Orange, c.Blue, c.Cyan, c.Magenta,
	}
	var sw strings.Builder
	swW := 0
	for i, hex := range swatches {
		if swW+3 > width {
			break
		}
		if i > 0 {
			sw.WriteString(bgFill.Render(" "))
			swW++
		}
		sw.WriteString(lipgloss.NewStyle().Background(lipgloss.Color(hex)).Render("  "))
		swW += 2
	}
	rows = append(rows, padLine(sw.String()))

	hexSummary := fmt.Sprintf("bg %s · fg %s · accent %s", c.Background, c.Foreground, c.Accent)
	rows = append(rows, padLine(
		lipgloss.NewStyle().Background(bg).Foreground(pal.MutedFg).Render(truncateDisplay(hexSummary, width)),
	))
	rows = append(rows, emptyRow)

	// 4. Live Syntax Code Preview Card rendered on the hovered scheme's true background!
	codeBg := lipgloss.Color(c.Background)
	codeFill := lipgloss.NewStyle().Background(codeBg)
	kw := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Magenta)).Bold(true)
	fn := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Blue))
	typ := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Cyan))
	str := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Green))
	num := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Orange))
	cmt := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.DarkForeground)).Italic(true)
	fg := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Foreground))
	lnSt := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Muted))
	gitAdd := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Green)).Bold(true)
	gitMod := lipgloss.NewStyle().Background(codeBg).Foreground(lipgloss.Color(c.Yellow)).Bold(true)

	type codeLine struct {
		ln    string
		sign  string
		parts []struct {
			st  lipgloss.Style
			txt string
		}
	}

	sampleLines := []codeLine{
		{ln: " 1 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{kw, "package"}, {fg, " larry"}}},
		{ln: " 2 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{cmt, "// Live syntax preview"}}},
		{ln: " 3 ", sign: "▎", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{kw, "func"}, {fg, " "}, {fn, "Apply"}, {fg, "(name "}, {typ, "string"}, {fg, ", n "}, {typ, "int"}, {fg, ") "}, {typ, "bool"}, {fg, " {"}}},
		{ln: " 4 ", sign: "▎", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{fg, "    "}, {kw, "const"}, {fg, " tag = "}, {str, `"` + truncateDisplay(scheme.ID, 14) + `"`}}},
		{ln: " 5 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{fg, "    "}, {kw, "if"}, {fg, " n > "}, {num, "0"}, {fg, " && name != "}, {str, `""`}, {fg, " {"}}},
		{ln: " 6 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{fg, "        "}, {kw, "return"}, {fg, " "}, {num, "true"}}},
		{ln: " 7 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{fg, "    }"}}},
		{ln: " 8 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{fg, "    "}, {kw, "return"}, {fg, " "}, {num, "false"}}},
		{ln: " 9 ", sign: " ", parts: []struct {
			st  lipgloss.Style
			txt string
		}{{fg, "}"}}},
	}

	// Reserve 2 rows at bottom for mini status bar preview if room allows
	reserveBottom := 0
	if height-len(rows) >= 6 {
		reserveBottom = 2
	}
	maxCodeRows := height - len(rows) - reserveBottom
	if maxCodeRows < 0 {
		maxCodeRows = 0
	}
	if maxCodeRows > len(sampleLines) {
		maxCodeRows = len(sampleLines)
	}

	for i := 0; i < maxCodeRows; i++ {
		cl := sampleLines[i]
		var lb strings.Builder
		used := 0
		signRendered := codeFill.Render(" ")
		if cl.sign == "▎" {
			if i == 2 {
				signRendered = gitAdd.Render("▎")
			} else {
				signRendered = gitMod.Render("▎")
			}
		}
		lb.WriteString(signRendered)
		used++
		lb.WriteString(lnSt.Render(cl.ln))
		used += runewidth.StringWidth(cl.ln)

		for _, part := range cl.parts {
			if used >= width {
				break
			}
			rem := width - used
			txt := truncateDisplay(part.txt, rem)
			lb.WriteString(part.st.Render(txt))
			used += runewidth.StringWidth(txt)
		}
		if used < width {
			lb.WriteString(codeFill.Render(strings.Repeat(" ", width-used)))
		}
		rows = append(rows, lb.String())
	}

	// 5. Mini Status Bar Preview in the hovered scheme's palette
	if reserveBottom == 2 && len(rows)+2 <= height {
		rows = append(rows, emptyRow)
		sbBg := prevPal.Surface
		sbFill := lipgloss.NewStyle().Background(sbBg)
		mPill := lipgloss.NewStyle().Background(prevPal.Accent).Foreground(prevPal.AccentFg).Bold(true).Render(" EDIT ")
		bPill := lipgloss.NewStyle().Background(prevPal.Background).Foreground(prevPal.Cyan).Bold(true).Render(" ⎇ main ")
		fPart := lipgloss.NewStyle().Background(sbBg).Foreground(prevPal.Foreground).Bold(true).Render(" app.go") +
			lipgloss.NewStyle().Background(sbBg).Foreground(prevPal.Yellow).Bold(true).Render(" ●")
		rPart := lipgloss.NewStyle().Background(sbBg).Foreground(prevPal.SubtleFg).Render("Go · 4:12 ")

		sbLeft := mPill + bPill + fPart
		if lipgloss.Width(sbLeft)+lipgloss.Width(rPart)+1 <= width {
			sbGap := width - lipgloss.Width(sbLeft) - lipgloss.Width(rPart)
			rows = append(rows, sbLeft+sbFill.Render(strings.Repeat(" ", sbGap))+rPart)
		} else {
			sbLeft = mPill + fPart
			sbGap := width - lipgloss.Width(sbLeft)
			if sbGap < 0 {
				sbGap = 0
			}
			rows = append(rows, sbLeft+sbFill.Render(strings.Repeat(" ", sbGap)))
		}
	}

	for len(rows) < height {
		rows = append(rows, emptyRow)
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return rows
}
