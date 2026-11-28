package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"larry/internal/plugin"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
)

type statusBarCacheKey struct {
	width         int
	fileName      string
	rootPath      string
	modified      bool
	gitBranch     string
	wordWrap      bool
	cursorRow     int
	cursorCol     int
	lineCount     int
	cursorCount   int
	statusMsg     string
	treeVisible   bool
	treeFocused   bool
	viewMode      ViewMode
	leaderKey     string
	searchQuery   string
	searchCount   int
	searchIndex   int
	themeVer      int
	profile       termenv.Profile
	pluginUIVer   int64
	activeDiagMsg string
	activeDiagSev plugin.DiagnosticSeverity
}

type statusBarCache struct {
	valid    bool
	key      statusBarCacheKey
	rendered string
}

type statusSegment struct {
	text  string
	width int
}

func makeStatusSegment(rendered string) statusSegment {
	return statusSegment{
		text:  rendered,
		width: lipgloss.Width(rendered),
	}
}

// renderStatusBar builds a strictly single-line, IDE/Lualine-style status bar
// with mode & file context on the left and responsive telemetry on the right.
func (m *Model) renderStatusBar() string {
	width := m.Width
	if width < 20 {
		width = 20
	}

	var pluginUIVer int64
	var activeDiagMsg string
	var activeDiagSev plugin.DiagnosticSeverity
	if m.plugins != nil {
		pluginUIVer = m.plugins.UIVersion()
		if m.statusMsg == "" && m.FileName != "" {
			if lineDiags := m.plugins.DiagnosticsByLine(m.activeAbsPath())[m.CursorRow]; len(lineDiags) > 0 {
				activeDiagMsg = lineDiags[0].Message
				activeDiagSev = lineDiags[0].Severity
			}
		}
	}

	key := statusBarCacheKey{
		width:         width,
		fileName:      m.FileName,
		rootPath:      m.fileTree.RootPath,
		modified:      m.Modified,
		gitBranch:     m.fileTree.GitBranch,
		wordWrap:      m.Config.WordWrap,
		cursorRow:     m.CursorRow,
		cursorCol:     m.CursorCol,
		lineCount:     len(m.Lines),
		cursorCount:   len(m.extraCursors),
		statusMsg:     m.statusMsg,
		treeVisible:   m.fileTree.Visible,
		treeFocused:   m.fileTree.Focused,
		viewMode:      m.viewMode,
		leaderKey:     m.Config.LeaderKey,
		searchQuery:   m.searchQuery,
		searchCount:   len(m.searchResults),
		searchIndex:   m.currentResultIndex,
		themeVer:      themeVersion,
		profile:       lipgloss.ColorProfile(),
		pluginUIVer:   pluginUIVer,
		activeDiagMsg: activeDiagMsg,
		activeDiagSev: activeDiagSev,
	}

	if m.statusCache != nil && m.statusCache.valid && m.statusCache.key == key {
		return m.statusCache.rendered
	}

	rendered := m.renderStatusBarUncached(width)
	if m.statusCache != nil {
		m.statusCache.key = key
		m.statusCache.rendered = rendered
		m.statusCache.valid = true
	}
	return rendered
}

func (m *Model) renderStatusBarUncached(width int) string {
	p := activePalette
	barBg := p.Surface
	pillBg := p.Background

	bgFill := lipgloss.NewStyle().Background(barBg)
	sepDot := lipgloss.NewStyle().Foreground(p.Border).Background(barBg).Render(" · ")

	// ── 1. Mode pill (far left) ──
	modeLabel, modeBg, modeFg := m.statusModeBadge(p)
	modeSeg := makeStatusSegment(
		lipgloss.NewStyle().
			Background(modeBg).
			Foreground(modeFg).
			Bold(true).
			Render(" " + modeLabel + " "),
	)

	// ── 2. Git branch pill (subtle elevated contrast) ──
	var branchSeg statusSegment
	branch := strings.TrimPrefix(strings.TrimSpace(m.fileTree.GitBranch), "heads/")
	if branch != "" {
		branchText := truncateDisplay(branch, 22)
		branchSeg = makeStatusSegment(
			lipgloss.NewStyle().
				Background(pillBg).
				Foreground(p.Cyan).
				Bold(true).
				Render(" ⎇ " + branchText + " "),
		)
	}

	// ── 3. Inline status badges (diagnostics, wrap, plugin badges) ──
	var badges []statusSegment
	if m.plugins != nil && m.FileName != "" {
		errs, warns := m.plugins.DiagnosticCounts(m.activeAbsPath())
		if errs > 0 {
			badges = append(badges, makeStatusSegment(
				lipgloss.NewStyle().
					Background(barBg).
					Foreground(p.Red).
					Bold(true).
					Render(fmt.Sprintf(" ● %d", errs)),
			))
		}
		if warns > 0 {
			badges = append(badges, makeStatusSegment(
				lipgloss.NewStyle().
					Background(barBg).
					Foreground(p.Yellow).
					Bold(true).
					Render(fmt.Sprintf(" ▲ %d", warns)),
			))
		}
	}
	if m.Config.WordWrap {
		badges = append(badges, makeStatusSegment(
			lipgloss.NewStyle().
				Background(barBg).
				Foreground(p.AccentAlt).
				Bold(true).
				Render(" WRAP"),
		))
	}
	if m.plugins != nil {
		for _, badge := range m.plugins.Badges() {
			if strings.TrimSpace(badge.Text) == "" {
				continue
			}
			badges = append(badges, makeStatusSegment(
				lipgloss.NewStyle().
					Background(barBg).
					Foreground(paletteColorByName(badge.Color)).
					Bold(true).
					Render(" "+badge.Text),
			))
		}
	}

	// ── 4. Determine right-hand content (transient status / line diagnostic / telemetry) ──
	var rightStr string
	var rightW int

	// Reserve minimum room for left mode + branch + short file + active badges before sizing right side.
	totalBadgesW := 0
	for _, b := range badges {
		totalBadgesW += b.width
	}
	minLeft := modeSeg.width + 12 + totalBadgesW
	if branchSeg.width > 0 && width >= 50 {
		minLeft += branchSeg.width
	}
	if maxMinLeft := width * 60 / 100; minLeft > maxMinLeft {
		minLeft = maxMinLeft
	}
	maxRightW := width - minLeft - 2
	if maxRightW < 0 {
		maxRightW = 0
	}

	if m.statusMsg != "" && maxRightW >= 8 {
		msg := truncateDisplay(strings.ReplaceAll(m.statusMsg, "\n", " "), maxRightW-2)
		seg := makeStatusSegment(
			lipgloss.NewStyle().
				Background(pillBg).
				Foreground(p.Foreground).
				Render(" " + msg + " "),
		)
		rightStr, rightW = seg.text, seg.width
	} else if diagSeg, ok := m.renderCursorDiagnosticSegment(p, pillBg, maxRightW); ok {
		rightStr, rightW = diagSeg.text, diagSeg.width
	} else {
		rightStr, rightW = m.renderRightTelemetry(p, barBg, pillBg, sepDot, maxRightW)
	}

	// ── 5. Compose left side within remaining width (never wraps) ──
	leftBudget := width - rightW
	if rightW > 0 {
		leftBudget -= 2 // breathing gap between left and right
	}
	if leftBudget < 1 {
		leftBudget = width
		rightStr = ""
		rightW = 0
	}

	var leftBuilder strings.Builder
	leftW := 0

	// Mode pill always first (clipped if terminal is impossibly narrow)
	if modeSeg.width <= leftBudget {
		leftBuilder.WriteString(modeSeg.text)
		leftW += modeSeg.width
	} else {
		clipped := truncateDisplay(" "+modeLabel+" ", leftBudget)
		seg := makeStatusSegment(
			lipgloss.NewStyle().
				Background(modeBg).
				Foreground(modeFg).
				Bold(true).
				Render(clipped),
		)
		leftBuilder.WriteString(seg.text)
		leftW += seg.width
	}

	// Branch pill if it leaves enough room for at least a short filename
	if branchSeg.width > 0 && leftW+branchSeg.width+10 <= leftBudget {
		leftBuilder.WriteString(branchSeg.text)
		leftW += branchSeg.width
	}

	// Reserve space for badges that fit
	var keptBadges []statusSegment
	badgesW := 0
	for _, b := range badges {
		if leftW+10+badgesW+b.width <= leftBudget {
			keptBadges = append(keptBadges, b)
			badgesW += b.width
		}
	}

	// Smart-truncated file path + modified dot
	fileAvail := leftBudget - leftW - badgesW
	if fileAvail >= 4 {
		fileSeg := m.renderStatusFileSegment(p, barBg, fileAvail)
		leftBuilder.WriteString(fileSeg.text)
		leftW += fileSeg.width
	}

	for _, b := range keptBadges {
		leftBuilder.WriteString(b.text)
		leftW += b.width
	}

	// ── 6. Fill exact middle gap and join into a single line ──
	gap := width - leftW - rightW
	if gap < 0 {
		gap = 0
	}

	return leftBuilder.String() + bgFill.Render(strings.Repeat(" ", gap)) + rightStr
}

func statusBadgeFg(bg lipgloss.Color, fallback lipgloss.Color) lipgloss.Color {
	if s := string(bg); isHexColor(s) {
		return contrastingTextColor(s, "#0c0b0c", "#ffffff")
	}
	return fallback
}

func (m Model) statusModeBadge(p UIPalette) (label string, bg, fg lipgloss.Color) {
	darkFg := lipgloss.Color("0")
	switch {
	case m.fileTree.Visible && m.fileTree.Focused:
		return "TREE", p.Blue, statusBadgeFg(p.Blue, p.AccentFg)
	case m.hasMultiCursor():
		fallback := darkFg
		if !p.IsDark {
			fallback = p.AccentFg
		}
		return fmt.Sprintf("%d CURSORS", len(m.allCursors())), p.Green, statusBadgeFg(p.Green, fallback)
	case len(m.searchResults) > 0:
		return "FIND", p.Yellow, statusBadgeFg(p.Yellow, darkFg)
	case m.viewMode == ViewModeSplit && isMarkdownFile(m.FileName):
		return "PREVIEW", p.Magenta, statusBadgeFg(p.Magenta, p.AccentFg)
	default:
		return "EDIT", p.Accent, p.AccentFg
	}
}

func (m Model) renderStatusFileSegment(p UIPalette, barBg lipgloss.Color, maxW int) statusSegment {
	modStr := ""
	modW := 0
	if m.Modified {
		modStr = lipgloss.NewStyle().
			Background(barBg).
			Foreground(p.Yellow).
			Bold(true).
			Render(" ●")
		modW = 2
	}

	// Leading space + trailing space around filename
	padLeft := lipgloss.NewStyle().Background(barBg).Render(" ")
	nameBudget := maxW - 1 - modW
	if nameBudget < 1 {
		return statusSegment{}
	}

	if m.FileName == "" {
		txt := truncateDisplay("[No Name]", nameBudget)
		rendered := padLeft +
			lipgloss.NewStyle().Background(barBg).Foreground(p.MutedFg).Italic(true).Render(txt) +
			modStr
		return makeStatusSegment(rendered)
	}

	displayPath := m.statusDisplayPath()
	dirPart, basePart := formatCompactPath(displayPath, nameBudget)

	dirStyle := lipgloss.NewStyle().Background(barBg).Foreground(p.SubtleFg)
	baseStyle := lipgloss.NewStyle().Background(barBg).Foreground(p.Foreground).Bold(true)

	rendered := padLeft
	if dirPart != "" {
		rendered += dirStyle.Render(dirPart)
	}
	rendered += baseStyle.Render(basePart) + modStr
	return makeStatusSegment(rendered)
}

func (m Model) statusDisplayPath() string {
	raw := filepath.Clean(m.FileName)
	if m.fileTree.RootPath != "" && filepath.IsAbs(raw) {
		if rel, err := filepath.Rel(m.fileTree.RootPath, raw); err == nil && !strings.HasPrefix(rel, "..") {
			raw = rel
		}
	}
	return filepath.ToSlash(raw)
}

// formatCompactPath splits a path into a dimmed directory prefix and bold
// filename, collapsing leading directories into "…/parent/" before ever
// truncating the filename itself.
func formatCompactPath(p string, maxW int) (dirPart, basePart string) {
	if maxW <= 0 {
		return "", ""
	}
	slashIdx := strings.LastIndex(p, "/")
	if slashIdx < 0 {
		return "", truncateDisplay(p, maxW)
	}

	base := p[slashIdx+1:]
	dir := p[:slashIdx]

	if runewidth.StringWidth(p) <= maxW {
		return dir + "/", base
	}

	// Try "…/<parent>/<base>"
	parts := strings.Split(dir, "/")
	parent := parts[len(parts)-1]
	if parent != "" && parent != "." {
		shortDir := "…/" + parent + "/"
		if runewidth.StringWidth(shortDir)+runewidth.StringWidth(base) <= maxW {
			return shortDir, base
		}
	}

	// Try "…/<base>"
	if runewidth.StringWidth("…/")+runewidth.StringWidth(base) <= maxW {
		return "…/", base
	}

	// Fallback: just filename (truncated if still wider than budget)
	return "", truncateDisplay(base, maxW)
}

func (m Model) renderCursorDiagnosticSegment(p UIPalette, pillBg lipgloss.Color, maxW int) (statusSegment, bool) {
	if m.plugins == nil || m.FileName == "" || maxW < 14 {
		return statusSegment{}, false
	}
	lineDiags := m.plugins.DiagnosticsByLine(m.activeAbsPath())[m.CursorRow]
	if len(lineDiags) == 0 {
		return statusSegment{}, false
	}

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
	label := prefix + strings.ReplaceAll(d.Message, "\n", " ")
	if d.Source != "" {
		label += " [" + d.Source + "]"
	}
	txt := truncateDisplay(label, maxW-2)
	seg := makeStatusSegment(
		lipgloss.NewStyle().
			Background(pillBg).
			Foreground(diagColor).
			Render(" " + txt + " "),
	)
	return seg, true
}

func (m Model) renderRightTelemetry(p UIPalette, barBg, pillBg lipgloss.Color, sepDot string, maxW int) (string, int) {
	if maxW < 6 {
		return "", 0
	}

	leader := displayLeaderKey(m.Config.LeaderKey)
	keySt := lipgloss.NewStyle().Foreground(p.AccentAlt).Background(barBg).Bold(true)
	descSt := lipgloss.NewStyle().Foreground(p.SubtleFg).Background(barBg)
	mutedSt := lipgloss.NewStyle().Foreground(p.SubtleFg).Background(barBg)
	posSt := lipgloss.NewStyle().Foreground(p.Foreground).Background(barBg).Bold(true)
	bgFill := lipgloss.NewStyle().Background(barBg)

	helpPill := makeStatusSegment(
		lipgloss.NewStyle().Background(pillBg).Foreground(p.AccentAlt).Bold(true).Render(" "+leader+"+h") +
			lipgloss.NewStyle().Background(pillBg).Foreground(p.SubtleFg).Render(" Help "),
	)

	// ── File Tree focused: concise navigation hints ──
	if m.fileTree.Visible && m.fileTree.Focused {
		type hint struct{ k, d string }
		hints := []hint{
			{"Enter", "Open"},
			{"←/→", "Fold"},
			{".", "Hidden"},
			{"Esc", "Editor"},
		}
		var parts []statusSegment
		for _, h := range hints {
			parts = append(parts, makeStatusSegment(keySt.Render(h.k)+descSt.Render(" "+h.d)))
		}
		return packRightSegments(parts, helpPill, sepDot, bgFill, maxW)
	}

	// ── Editor / Preview / Search telemetry ──
	var items []statusSegment

	// Search match counter when active
	if len(m.searchResults) > 0 {
		matchStr := fmt.Sprintf("%d/%d", m.currentResultIndex+1, len(m.searchResults))
		if m.searchQuery != "" {
			q := truncateDisplay(m.searchQuery, 14)
			matchStr = fmt.Sprintf("“%s” %s", q, matchStr)
		}
		items = append(items, makeStatusSegment(
			lipgloss.NewStyle().Background(barBg).Foreground(p.Yellow).Bold(true).Render(matchStr),
		))
	}

	// Language / filetype
	if lang := detectStatusLanguage(m.FileName); lang != "" {
		items = append(items, makeStatusSegment(mutedSt.Render(lang)))
	}

	// Cursor position (1-based)
	ln := m.CursorRow + 1
	if ln < 1 {
		ln = 1
	}
	col := m.CursorCol + 1
	if col < 1 {
		col = 1
	}
	posLong := makeStatusSegment(posSt.Render(fmt.Sprintf("Ln %d, Col %d", ln, col)))
	posShort := makeStatusSegment(posSt.Render(fmt.Sprintf("%d:%d", ln, col)))

	// Scroll progress
	pct := makeStatusSegment(mutedSt.Render(scrollProgressLabel(m.CursorRow, len(m.Lines))))

	// Try full layout: [search?, lang?, Ln X, Col Y, pct] + [Help pill]
	full := append(append([]statusSegment{}, items...), posLong, pct)
	if out, w, ok := joinRightWithHelp(full, helpPill, sepDot, bgFill, maxW); ok {
		return out, w
	}

	// Drop language first
	var noLang []statusSegment
	if len(m.searchResults) > 0 && len(items) > 0 {
		noLang = append(noLang, items[0])
	}
	noLang = append(noLang, posLong, pct)
	if out, w, ok := joinRightWithHelp(noLang, helpPill, sepDot, bgFill, maxW); ok {
		return out, w
	}

	// Compact position + Help pill
	if out, w, ok := joinRightWithHelp([]statusSegment{posShort}, helpPill, sepDot, bgFill, maxW); ok {
		return out, w
	}

	// Compact position only (very narrow terminal)
	compactOnly := makeStatusSegment(bgFill.Render(" ") + posShort.text + bgFill.Render(" "))
	if compactOnly.width <= maxW {
		return compactOnly.text, compactOnly.width
	}

	return "", 0
}

func packRightSegments(parts []statusSegment, helpPill statusSegment, sepDot string, bgFill lipgloss.Style, maxW int) (string, int) {
	// Include as many leading hints as fit alongside the Help pill
	for n := len(parts); n >= 0; n-- {
		if out, w, ok := joinRightWithHelp(parts[:n], helpPill, sepDot, bgFill, maxW); ok {
			return out, w
		}
	}
	return "", 0
}

func joinRightWithHelp(items []statusSegment, helpPill statusSegment, sepDot string, bgFill lipgloss.Style, maxW int) (string, int, bool) {
	sepW := lipgloss.Width(sepDot)
	var b strings.Builder
	w := 0

	if len(items) > 0 {
		for i, it := range items {
			if i > 0 {
				b.WriteString(sepDot)
				w += sepW
			}
			b.WriteString(it.text)
			w += it.width
		}
		if helpPill.width > 0 {
			b.WriteString(bgFill.Render("  "))
			w += 2
		} else {
			b.WriteString(bgFill.Render(" "))
			w++
		}
	}
	if helpPill.width > 0 {
		b.WriteString(helpPill.text)
		w += helpPill.width
	}

	if w > maxW {
		return "", 0, false
	}
	return b.String(), w, true
}

func scrollProgressLabel(row, totalLines int) string {
	if totalLines <= 1 {
		return "All"
	}
	if row <= 0 {
		return "Top"
	}
	if row >= totalLines-1 {
		return "Bot"
	}
	pct := (row * 100) / (totalLines - 1)
	if pct < 1 {
		pct = 1
	}
	if pct > 99 {
		pct = 99
	}
	return fmt.Sprintf("%d%%", pct)
}

func detectStatusLanguage(filename string) string {
	if filename == "" {
		return ""
	}
	base := strings.ToLower(filepath.Base(filename))
	switch base {
	case "dockerfile":
		return "Dockerfile"
	case "makefile", "gnumakefile":
		return "Makefile"
	case "cmakelists.txt":
		return "CMake"
	}

	switch strings.ToLower(filepath.Ext(base)) {
	case ".go":
		return "Go"
	case ".dart":
		return "Dart"
	case ".ts":
		return "TypeScript"
	case ".tsx":
		return "TSX"
	case ".js", ".mjs", ".cjs":
		return "JavaScript"
	case ".jsx":
		return "JSX"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".md", ".markdown", ".mdx":
		return "Markdown"
	case ".json", ".jsonc":
		return "JSON"
	case ".yaml", ".yml":
		return "YAML"
	case ".toml":
		return "TOML"
	case ".lua":
		return "Lua"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".scss":
		return "SCSS"
	case ".sql":
		return "SQL"
	case ".xml", ".svg":
		return "XML"
	case ".c", ".h":
		return "C"
	case ".cc", ".cpp", ".cxx", ".hpp":
		return "C++"
	case ".java":
		return "Java"
	case ".kt", ".kts":
		return "Kotlin"
	case ".swift":
		return "Swift"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".zig":
		return "Zig"
	case ".txt":
		return "Plain Text"
	}

	if lexer := lexers.Match(filename); lexer != nil && lexer != lexers.Fallback {
		if cfg := lexer.Config(); cfg != nil && cfg.Name != "" {
			if cfg.Name == "larry-markdown" {
				return "Markdown"
			}
			return cfg.Name
		}
	}
	return ""
}

func (m Model) statusBarHeight() int {
	return 1
}

func (m Model) editorViewportHeight() int {
	viewportHeight := m.workspaceHeight() - m.tabBarHeight()
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	return viewportHeight
}
