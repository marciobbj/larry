package ui

import (
	"strings"
	"sync"

	"larry/internal/plugin"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ansiStyle holds pre-rendered ANSI SGR open/close sequences extracted from a
// lipgloss.Style once per theme/color profile so hot rendering loops never
// invoke lipgloss.Style.Render (or termenv hex parsing) per character.
type ansiStyle struct {
	prefix string
	suffix string
}

func compileANSIStyle(st lipgloss.Style) ansiStyle {
	const sentinel = "\x00"
	rendered := st.Render(sentinel)
	idx := strings.IndexByte(rendered, 0)
	if idx < 0 {
		return ansiStyle{}
	}
	return ansiStyle{
		prefix: rendered[:idx],
		suffix: rendered[idx+1:],
	}
}

func (a ansiStyle) wrap(s string) string {
	if a.prefix == "" && a.suffix == "" {
		return s
	}
	return a.prefix + s + a.suffix
}

type compiledUIANSI struct {
	themeVer int
	profile  termenv.Profile
	paintBg  bool

	canvasBg       ansiStyle
	defaultText    ansiStyle
	cursor         ansiStyle
	selected       ansiStyle
	occurrence     ansiStyle
	search         ansiStyle
	diagError      ansiStyle
	diagWarn       ansiStyle
	lineNum        ansiStyle
	borderStr      string
	cursorSpace    string
	eolClear       string
	gitAddedStr    string
	gitModifiedStr string
	gitDeletedStr  string
	gitEmptyStr    string
}

var (
	uiANSIMu          sync.RWMutex
	cachedUIANSI      compiledUIANSI
	highlightANSICache sync.Map // highlightStyleKey -> ansiStyle
	gutterSignCache    sync.Map // gutterSignCacheKey -> string
	hlCacheThemeVer   int
	hlCacheProfile    termenv.Profile
)

type gutterSignCacheKey struct {
	glyph string
	color string
}

func getCompiledUIANSI() compiledUIANSI {
	prof := lipgloss.ColorProfile()
	tv := themeVersion

	uiANSIMu.RLock()
	if cachedUIANSI.themeVer == tv && cachedUIANSI.profile == prof && cachedUIANSI.themeVer > 0 {
		res := cachedUIANSI
		uiANSIMu.RUnlock()
		return res
	}
	uiANSIMu.RUnlock()

	uiANSIMu.Lock()
	defer uiANSIMu.Unlock()
	if cachedUIANSI.themeVer == tv && cachedUIANSI.profile == prof && cachedUIANSI.themeVer > 0 {
		return cachedUIANSI
	}

	// Invalidate highlight & gutter caches when theme or color profile changes.
	if hlCacheThemeVer != tv || hlCacheProfile != prof {
		highlightANSICache.Range(func(k, _ any) bool {
			highlightANSICache.Delete(k)
			return true
		})
		gutterSignCache.Range(func(k, _ any) bool {
			gutterSignCache.Delete(k)
			return true
		})
		hlCacheThemeVer = tv
		hlCacheProfile = prof
	}

	lineNumANSI := compileANSIStyle(lineNumStyle)
	cursorANSI := compileANSIStyle(styleCursor)
	canvasBgANSI := compileANSIStyle(canvasBlankStyle)
	defaultTextANSI := compileANSIStyle(withCanvasBg(lipgloss.NewStyle().Foreground(activePalette.Foreground)))
	eolClear := "\x1b[K"
	if paintCanvasBg && canvasBgANSI.prefix != "" {
		eolClear = canvasBgANSI.prefix + "\x1b[K" + canvasBgANSI.suffix
	}

	cachedUIANSI = compiledUIANSI{
		themeVer:       tv,
		profile:        prof,
		paintBg:        paintCanvasBg,
		canvasBg:       canvasBgANSI,
		defaultText:    defaultTextANSI,
		cursor:         cursorANSI,
		selected:       compileANSIStyle(styleSelected),
		occurrence:     compileANSIStyle(styleOccurrence),
		search:         compileANSIStyle(styleSearch),
		diagError:      compileANSIStyle(diagErrorUnderlineStyle),
		diagWarn:       compileANSIStyle(diagWarnUnderlineStyle),
		lineNum:        lineNumANSI,
		borderStr:      borderStyle.Render("│"),
		cursorSpace:    cursorANSI.wrap(" "),
		eolClear:       eolClear,
		gitAddedStr:    gitAddedStyle.Render("▎"),
		gitModifiedStr: gitModifiedStyle.Render("▎"),
		gitDeletedStr:  gitDeletedStyle.Render("▁"),
		gitEmptyStr:    lineNumANSI.wrap(" "),
	}
	return cachedUIANSI
}

func cachedHighlightANSI(h plugin.TextHighlight) ansiStyle {
	key := highlightStyleKey{
		fg:        h.FG,
		bg:        h.BG,
		bold:      h.Bold,
		italic:    h.Italic,
		underline: h.Underline,
	}
	if v, ok := highlightANSICache.Load(key); ok {
		return v.(ansiStyle)
	}
	st := lipgloss.NewStyle()
	if h.FG != "" {
		st = st.Foreground(paletteColorByName(h.FG))
	}
	if h.BG != "" {
		st = st.Background(paletteColorByName(h.BG))
	} else if paintCanvasBg {
		if bg := canvasBgColor(); bg != "" {
			st = st.Background(bg)
		}
	}
	if h.Bold {
		st = st.Bold(true)
	}
	if h.Italic {
		st = st.Italic(true)
	}
	if h.Underline {
		st = st.Underline(true)
	}
	compiled := compileANSIStyle(st)
	highlightANSICache.Store(key, compiled)
	return compiled
}

func cachedGutterSignStr(glyph, color string) string {
	key := gutterSignCacheKey{glyph: glyph, color: color}
	if v, ok := gutterSignCache.Load(key); ok {
		return v.(string)
	}
	rendered := withCanvasBg(lipgloss.NewStyle().
		Foreground(paletteColorByName(color)).
		Bold(true)).
		Render(glyph)
	gutterSignCache.Store(key, rendered)
	return rendered
}

// lineRunWriter coalesces adjacent characters sharing the same ansiStyle into
// a single ANSI escape sequence, cutting both CPU time and terminal output bytes.
type lineRunWriter struct {
	s      *strings.Builder
	active ansiStyle
	open   bool
}

func (w *lineRunWriter) setStyle(st ansiStyle) {
	if w.open && w.active == st {
		return
	}
	if w.open {
		w.s.WriteString(w.active.suffix)
		w.open = false
	}
	if st.prefix != "" || st.suffix != "" {
		w.s.WriteString(st.prefix)
		w.active = st
		w.open = true
	}
}

func (w *lineRunWriter) writeSpaces(count int, st ansiStyle, isCursorStart bool, cursorANSI, defaultBg ansiStyle) {
	if count <= 0 {
		return
	}
	if isCursorStart {
		w.setStyle(cursorANSI)
		w.s.WriteByte(' ')
		if count > 1 {
			w.setStyle(defaultBg)
			for i := 1; i < count; i++ {
				w.s.WriteByte(' ')
			}
		}
		return
	}
	w.setStyle(st)
	for i := 0; i < count; i++ {
		w.s.WriteByte(' ')
	}
}

func (w *lineRunWriter) flush() {
	if w.open {
		w.s.WriteString(w.active.suffix)
		w.open = false
	}
}

// joinSidebarAndMain stitches the already-padded sidebar, 1-column divider, and
// main view line-by-line without rescanning ANSI widths via lipgloss.JoinHorizontal.
func joinSidebarAndMain(treeView, divider, mainView string, treeW, mainW, height int, uiANSI compiledUIANSI) string {
	treeLines := strings.Split(treeView, "\n")
	divLines := strings.Split(divider, "\n")
	mainLines := strings.Split(mainView, "\n")

	rows := height
	if len(treeLines) > rows {
		rows = len(treeLines)
	}
	if len(mainLines) > rows {
		rows = len(mainLines)
	}

	emptyTree := uiANSI.canvasBg.wrap(strings.Repeat(" ", treeW))
	emptyMain := ""
	if uiANSI.paintBg && mainW > 0 {
		emptyMain = uiANSI.canvasBg.wrap(strings.Repeat(" ", mainW)) + uiANSI.eolClear
	}
	var b strings.Builder
	b.Grow(len(treeView) + len(divider) + len(mainView) + rows*8)

	for i := 0; i < rows; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < len(treeLines) {
			b.WriteString(treeLines[i])
		} else {
			b.WriteString(emptyTree)
		}
		if i < len(divLines) {
			b.WriteString(divLines[i])
		} else {
			b.WriteString(uiANSI.canvasBg.wrap(" "))
		}
		if i < len(mainLines) && (mainLines[i] != "" || !uiANSI.paintBg) {
			b.WriteString(mainLines[i])
		} else if emptyMain != "" {
			b.WriteString(emptyMain)
		}
	}
	return b.String()
}

// placeCentered centers content in a width×height viewport, filling the
// surrounding backdrop with activePalette.Background whenever a solid canvas
// theme (such as any curated light/dark colorscheme) is active.
func placeCentered(width, height int, content string) string {
	if paintCanvasBg {
		if bg := canvasBgColor(); bg != "" {
			return lipgloss.Place(
				width, height,
				lipgloss.Center, lipgloss.Center,
				content,
				lipgloss.WithWhitespaceBackground(bg),
			)
		}
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
