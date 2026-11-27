package ui

import (
	"fmt"
	"larry/internal/search"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type FinderMode int

const (
	FinderModeFile FinderMode = iota
	FinderModeGrep
)

type FinderModel struct {
	textInput textinput.Model
	mode      FinderMode
	results   []search.FinderResult
	cursor    int
	width     int
	height    int
	matcher   *search.FuzzyMatcher
	grep      *search.LiveGrep
	scanner   *search.DirectoryScanner
	allFiles  []string
	loading   bool
	root      string
}

func NewFinderModel(width, height int) FinderModel {
	return NewFinderModelInRoot(".", width, height)
}

func NewFinderModelInRoot(root string, width, height int) FinderModel {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	ti := textinput.New()
	ti.Placeholder = "Search files by name..."
	ti.Prompt = ""
	ti.Focus()

	return FinderModel{
		textInput: ti,
		mode:      FinderModeFile,
		matcher:   search.NewFuzzyMatcher(),
		grep:      search.NewLiveGrep(),
		scanner:   search.NewDirectoryScanner(),
		root:      root,
		width:     width,
		height:    height,
	}
}

func (m FinderModel) Init() tea.Cmd {
	return nil
}

type searchMsg []search.FinderResult

func (m FinderModel) Update(msg tea.Msg) (FinderModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.mode == FinderModeFile {
				m.mode = FinderModeGrep
				m.textInput.Placeholder = "Search text across project..."
			} else {
				m.mode = FinderModeFile
				m.textInput.Placeholder = "Search files by name..."
			}
			m.cursor = 0
			return m, m.performSearch()

		case "up", "ctrl+k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "ctrl+j":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
			return m, nil

		case "pgup":
			step := 8
			m.cursor -= step
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil

		case "pgdown":
			step := 8
			m.cursor += step
			if m.cursor >= len(m.results) {
				m.cursor = len(m.results) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			return m, nil
		}

	case searchMsg:
		m.results = msg
		m.loading = false
		if m.cursor >= len(m.results) {
			m.cursor = 0
		}
		return m, nil
	}

	oldQuery := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	if m.textInput.Value() != oldQuery {
		m.cursor = 0
		return m, m.performSearch()
	}

	return m, cmd
}

func (m *FinderModel) performSearch() tea.Cmd {
	query := m.textInput.Value()
	m.loading = true

	return func() tea.Msg {
		if m.mode == FinderModeFile {
			if m.allFiles == nil {
				files, _ := m.scanner.Scan(m.root)
				m.allFiles = files
			}

			var results []search.FinderResult
			for _, f := range m.allFiles {
				if matched, _ := m.matcher.Match(query, f); matched {
					results = append(results, search.FinderResult{
						File: &search.FileResult{Path: f},
						Mode: search.ModeFiles,
					})
					if len(results) >= 100 {
						break
					}
				}
			}
			return searchMsg(results)
		}

		results, _ := m.grep.Search(m.root, query)
		var finderResults []search.FinderResult
		for i := range results {
			finderResults = append(finderResults, search.FinderResult{
				Grep: &results[i],
				Mode: search.ModeGrep,
			})
			if len(finderResults) >= 100 {
				break
			}
		}
		return searchMsg(finderResults)
	}
}

func (m FinderModel) View() string {
	w := m.width
	if w < 36 {
		w = 36
	}
	h := m.height
	if h < 10 {
		h = 10
	}
	return m.Render(w, h)
}

// Render builds the finder modal interior with exact innerWidth and innerHeight.
func (m FinderModel) Render(innerWidth, innerHeight int) string {
	if innerWidth < 34 {
		innerWidth = 34
	}
	if innerHeight < 9 {
		innerHeight = 9
	}

	p := activePalette
	bg := modalStyle.GetBackground()
	inputBg := p.Surface

	bgFill := lipgloss.NewStyle().Background(bg)
	inputFill := lipgloss.NewStyle().Background(inputBg)
	titleStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)
	scopeStyle := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)
	ruleStyle := lipgloss.NewStyle().Foreground(p.Border).Background(bg)
	mutedBg := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)
	hintKeyStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)
	hintDescStyle := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)
	hintSepStyle := lipgloss.NewStyle().Foreground(p.Border).Background(bg)

	// 1. Header bar: Title + Scope on left, segmented FILES / GREP pills on right
	rootLabel := m.displayRootName()
	leftHeader := titleStyle.Render("Global Finder") + scopeStyle.Render("  "+rootLabel+"/")

	filesPill := lipgloss.NewStyle().Background(p.Surface).Foreground(p.MutedFg).Padding(0, 1).Render("FILES")
	grepPill := lipgloss.NewStyle().Background(p.Surface).Foreground(p.MutedFg).Padding(0, 1).Render("GREP")
	if m.mode == FinderModeFile {
		filesPill = lipgloss.NewStyle().Background(p.Accent).Foreground(p.AccentFg).Bold(true).Padding(0, 1).Render("FILES")
	} else {
		grepPill = lipgloss.NewStyle().Background(p.Magenta).Foreground(p.AccentFg).Bold(true).Padding(0, 1).Render("GREP")
	}
	rightHeader := filesPill + bgFill.Render(" ") + grepPill

	headerGap := innerWidth - lipgloss.Width(leftHeader) - lipgloss.Width(rightHeader)
	if headerGap < 1 {
		leftHeader = titleStyle.Render(truncateDisplay("Global Finder", innerWidth-lipgloss.Width(rightHeader)-1))
		headerGap = innerWidth - lipgloss.Width(leftHeader) - lipgloss.Width(rightHeader)
		if headerGap < 0 {
			headerGap = 0
		}
	}
	headerRow := leftHeader + bgFill.Render(strings.Repeat(" ", headerGap)) + rightHeader

	// 2. Search input bar on elevated p.Surface background
	var countStr string
	var countStyle lipgloss.Style
	switch {
	case m.loading:
		countStr = "searching… "
		countStyle = lipgloss.NewStyle().Foreground(p.Yellow).Background(inputBg)
	case len(m.results) == 0:
		countStr = "0 results "
		countStyle = lipgloss.NewStyle().Foreground(p.MutedFg).Background(inputBg)
	default:
		countStr = fmt.Sprintf("%d/%d ", m.cursor+1, len(m.results))
		countStyle = lipgloss.NewStyle().Foreground(p.Cyan).Background(inputBg).Bold(true)
	}
	countRendered := countStyle.Render(countStr)
	countW := lipgloss.Width(countRendered)

	promptIcon := " ❯ "
	if m.mode == FinderModeGrep {
		promptIcon = " ⌕ "
	}
	promptRendered := lipgloss.NewStyle().Foreground(p.Accent).Background(inputBg).Bold(true).Render(promptIcon)
	promptW := lipgloss.Width(promptRendered)

	inputAvail := innerWidth - promptW - countW - 1
	if inputAvail < 8 {
		inputAvail = 8
	}

	ti := m.textInput
	ti.Width = inputAvail
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(p.Foreground).Background(inputBg)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(p.MutedFg).Background(inputBg)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(p.AccentFg).Background(p.Accent)
	ti.Cursor.TextStyle = lipgloss.NewStyle().Foreground(p.Foreground).Background(inputBg)
	inputView := ti.View()
	inputW := lipgloss.Width(inputView)

	inGap := innerWidth - promptW - inputW - countW
	if inGap < 0 {
		inGap = 0
	}
	searchRow := promptRendered + inputView + inputFill.Render(strings.Repeat(" ", inGap)) + countRendered

	// 3. Divider rule
	dividerRow := ruleStyle.Render(strings.Repeat("─", innerWidth))

	// 4. Results list area (header + blank + search + divider + divider + footer = 6 chrome rows)
	maxResults := innerHeight - 6
	if maxResults < 3 {
		maxResults = 3
	}

	start := 0
	if m.cursor >= maxResults {
		start = m.cursor - maxResults + 1
	}

	query := m.textInput.Value()
	rows := make([]string, 0, innerHeight)
	rows = append(rows, headerRow)
	rows = append(rows, bgFill.Render(strings.Repeat(" ", innerWidth)))
	rows = append(rows, searchRow)
	rows = append(rows, dividerRow)

	renderedCount := 0
	if len(m.results) == 0 {
		var emptyMsg string
		switch {
		case m.loading:
			emptyMsg = "Searching project files…"
		case m.mode == FinderModeGrep && strings.TrimSpace(query) == "":
			emptyMsg = "Type a query to grep across project files"
		case query != "":
			emptyMsg = fmt.Sprintf("No matches found for %q", truncateDisplay(query, innerWidth-24))
		default:
			emptyMsg = "No files found in workspace"
		}
		padTop := (maxResults - 1) / 2
		for i := 0; i < padTop; i++ {
			rows = append(rows, bgFill.Render(strings.Repeat(" ", innerWidth)))
			renderedCount++
		}
		emptyW := runewidth.StringWidth(emptyMsg)
		leftPad := (innerWidth - emptyW) / 2
		if leftPad < 0 {
			leftPad = 0
			emptyMsg = truncateDisplay(emptyMsg, innerWidth)
			emptyW = runewidth.StringWidth(emptyMsg)
		}
		rightPad := innerWidth - leftPad - emptyW
		if rightPad < 0 {
			rightPad = 0
		}
		rows = append(rows,
			bgFill.Render(strings.Repeat(" ", leftPad))+
				mutedBg.Italic(true).Render(emptyMsg)+
				bgFill.Render(strings.Repeat(" ", rightPad)),
		)
		renderedCount++
	} else {
		for i := start; i < len(m.results) && renderedCount < maxResults; i++ {
			res := m.results[i]
			selected := i == m.cursor
			if res.Mode == search.ModeFiles && res.File != nil {
				rows = append(rows, renderFinderFileRow(res.File.Path, query, selected, innerWidth, p, bg))
			} else if res.Grep != nil {
				rows = append(rows, renderFinderGrepRow(res.Grep, query, selected, innerWidth, p, bg))
			} else {
				rows = append(rows, bgFill.Render(strings.Repeat(" ", innerWidth)))
			}
			renderedCount++
		}
	}

	for renderedCount < maxResults {
		rows = append(rows, bgFill.Render(strings.Repeat(" ", innerWidth)))
		renderedCount++
	}

	// 5. Footer divider + shortcut hints
	rows = append(rows, dividerRow)

	hints := []struct{ k, d string }{
		{"Tab", "Mode"},
		{"↑/↓", "Navigate"},
		{"Enter", "Open"},
		{"Esc", "Close"},
	}
	var footerParts []string
	for _, h := range hints {
		footerParts = append(footerParts, hintKeyStyle.Render(h.k)+hintDescStyle.Render(" "+h.d))
	}
	footerText := strings.Join(footerParts, hintSepStyle.Render("  ·  "))
	footerW := lipgloss.Width(footerText)
	if footerW > innerWidth {
		footerText = hintDescStyle.Render(truncateDisplay("Tab Mode · ↑/↓ Move · Enter Open · Esc Close", innerWidth))
		footerW = lipgloss.Width(footerText)
	}
	fPad := innerWidth - footerW
	if fPad < 0 {
		fPad = 0
	}
	rows = append(rows, footerText+bgFill.Render(strings.Repeat(" ", fPad)))

	return strings.Join(rows, "\n")
}

func (m FinderModel) displayRootName() string {
	root := m.root
	if root == "" || root == "." {
		if abs, err := filepath.Abs("."); err == nil {
			root = abs
		}
	}
	base := filepath.Base(root)
	if base == "" || base == "." {
		return "workspace"
	}
	return base
}

func renderFinderFileRow(path, query string, selected bool, width int, p UIPalette, bg lipgloss.TerminalColor) string {
	cleanPath := sanitizeFinderText(filepath.ToSlash(strings.TrimPrefix(path, "./")))
	avail := width - 3 // prefix " ▸ " or "   "
	if avail < 6 {
		avail = 6
	}
	display := truncatePathLeft(cleanPath, avail)
	usedW := 3 + runewidth.StringWidth(display)
	padRight := width - usedW
	if padRight < 0 {
		padRight = 0
	}

	dir, base := splitDisplayPath(display)

	if selected {
		rowBg := p.Selection
		ptrStyle := lipgloss.NewStyle().Foreground(p.SelectionFg).Background(rowBg).Bold(true)
		dirStyle := lipgloss.NewStyle().Foreground(p.SelectionFg).Background(rowBg).Faint(true)
		baseStyle := lipgloss.NewStyle().Foreground(p.SelectionFg).Background(rowBg).Bold(true)
		padStyle := lipgloss.NewStyle().Background(rowBg)
		return ptrStyle.Render(" ▸ ") +
			dirStyle.Render(dir) +
			baseStyle.Render(base) +
			padStyle.Render(strings.Repeat(" ", padRight))
	}

	padStyle := lipgloss.NewStyle().Background(bg)
	dirStyle := lipgloss.NewStyle().Foreground(p.MutedFg).Background(bg)
	baseStyle := lipgloss.NewStyle().Foreground(p.Foreground).Background(bg)
	matchStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)

	highlighted := renderFuzzyHighlight(dir, base, query, dirStyle, baseStyle, matchStyle)
	return padStyle.Render("   ") + highlighted + padStyle.Render(strings.Repeat(" ", padRight))
}

func renderFinderGrepRow(g *search.GrepResult, query string, selected bool, width int, p UIPalette, bg lipgloss.TerminalColor) string {
	cleanPath := sanitizeFinderText(filepath.ToSlash(strings.TrimPrefix(g.Path, "./")))
	lineNumStr := fmt.Sprintf(":%d", g.Line)
	snippet := strings.TrimSpace(sanitizeFinderText(g.Content))

	avail := width - 3 // prefix " ▸ " or "   "
	if avail < 10 {
		avail = 10
	}

	// Allocate up to 42% of row width to path:line so the matched code snippet has plenty of room
	maxLocW := avail * 42 / 100
	if maxLocW < 14 {
		maxLocW = 14
	}
	if maxLocW > avail-6 {
		maxLocW = avail - 6
	}
	if maxLocW < 4 {
		maxLocW = 4
	}

	pathAvail := maxLocW - runewidth.StringWidth(lineNumStr)
	if pathAvail < 3 {
		pathAvail = 3
	}
	locPath := truncatePathLeft(cleanPath, pathAvail)
	locW := runewidth.StringWidth(locPath) + runewidth.StringWidth(lineNumStr)

	sep := "  "
	snippetAvail := avail - locW - len(sep)
	if snippetAvail < 0 {
		snippetAvail = 0
	}
	snippetDisplay := truncateDisplay(snippet, snippetAvail)
	usedW := 3 + locW + len(sep) + runewidth.StringWidth(snippetDisplay)
	padRight := width - usedW
	if padRight < 0 {
		padRight = 0
	}

	if selected {
		rowBg := p.Selection
		ptrStyle := lipgloss.NewStyle().Foreground(p.SelectionFg).Background(rowBg).Bold(true)
		pathStyle := lipgloss.NewStyle().Foreground(p.SelectionFg).Background(rowBg).Bold(true)
		lineStyle := lipgloss.NewStyle().Foreground(p.Yellow).Background(rowBg).Bold(true)
		snipStyle := lipgloss.NewStyle().Foreground(p.SelectionFg).Background(rowBg)
		matchStyle := lipgloss.NewStyle().Foreground(p.Yellow).Background(rowBg).Bold(true).Underline(true)
		padStyle := lipgloss.NewStyle().Background(rowBg)
		return ptrStyle.Render(" ▸ ") +
			pathStyle.Render(locPath) +
			lineStyle.Render(lineNumStr) +
			padStyle.Render(sep) +
			renderSubstringHighlight(snippetDisplay, query, snipStyle, matchStyle) +
			padStyle.Render(strings.Repeat(" ", padRight))
	}

	padStyle := lipgloss.NewStyle().Background(bg)
	pathStyle := lipgloss.NewStyle().Foreground(p.Cyan).Background(bg)
	lineStyle := lipgloss.NewStyle().Foreground(p.Yellow).Background(bg)
	snipStyle := lipgloss.NewStyle().Foreground(p.SubtleFg).Background(bg)
	matchStyle := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(bg).Bold(true)

	return padStyle.Render("   ") +
		pathStyle.Render(locPath) +
		lineStyle.Render(lineNumStr) +
		padStyle.Render(sep) +
		renderSubstringHighlight(snippetDisplay, query, snipStyle, matchStyle) +
		padStyle.Render(strings.Repeat(" ", padRight))
}

func splitDisplayPath(p string) (dir, base string) {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return "", p
	}
	return p[:idx+1], p[idx+1:]
}

func truncatePathLeft(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth == 1 {
		return "…"
	}
	runes := []rune(s)
	target := maxWidth - 1 // leave 1 cell for leading "…"
	w := 0
	start := len(runes)
	for start > 0 {
		rw := runewidth.RuneWidth(runes[start-1])
		if rw < 0 {
			rw = 0
		}
		if w+rw > target {
			break
		}
		w += rw
		start--
	}
	return "…" + string(runes[start:])
}

func sanitizeFinderText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteByte(' ')
		case isUnsafeTerminalRune(r):
			b.WriteRune('·')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func renderFuzzyHighlight(dir, base, query string, dirStyle, baseStyle, matchStyle lipgloss.Style) string {
	if strings.TrimSpace(query) == "" {
		return dirStyle.Render(dir) + baseStyle.Render(base)
	}
	full := []rune(dir + base)
	dirLen := len([]rune(dir))
	qRunes := []rune(strings.ToLower(query))
	matched := make([]bool, len(full))
	qi := 0
	for i, r := range full {
		if qi < len(qRunes) && unicode.ToLower(r) == qRunes[qi] {
			matched[i] = true
			qi++
		}
	}
	if qi < len(qRunes) {
		return dirStyle.Render(dir) + baseStyle.Render(base)
	}

	var b strings.Builder
	for i, r := range full {
		s := string(r)
		if matched[i] {
			b.WriteString(matchStyle.Render(s))
		} else if i < dirLen {
			b.WriteString(dirStyle.Render(s))
		} else {
			b.WriteString(baseStyle.Render(s))
		}
	}
	return b.String()
}

func renderSubstringHighlight(text, query string, normalStyle, matchStyle lipgloss.Style) string {
	if text == "" {
		return ""
	}
	if query == "" {
		return normalStyle.Render(text)
	}
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)
	idx := strings.Index(lowerText, lowerQuery)
	if idx < 0 || idx+len(query) > len(text) {
		return normalStyle.Render(text)
	}
	var b strings.Builder
	if idx > 0 {
		b.WriteString(normalStyle.Render(text[:idx]))
	}
	b.WriteString(matchStyle.Render(text[idx : idx+len(query)]))
	if idx+len(query) < len(text) {
		b.WriteString(normalStyle.Render(text[idx+len(query):]))
	}
	return b.String()
}
