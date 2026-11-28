package ui

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	maxFullSyntaxLines = 5000
	maxFullSyntaxBytes = 256 * 1024
)

var (
	lexerCache       sync.Map // map[string]chroma.Lexer (filename -> lexer)
	theme            *chroma.Style
	themeVersion     int
	lipglossCache    sync.Map // map[chroma.TokenType]lipgloss.Style
	ansiTokenCache   sync.Map // map[chroma.TokenType]ansiStyle
	ansiCacheProfile termenv.Profile
	ansiCacheProfMu  sync.Mutex
	omarchyActive    bool
	omarchyColorPath string
	omarchySignature string

	// larryMarkdownLexer extends Chroma's Markdown highlighting with inline HTML
	// tags/comments, column-0 bold/italic, horizontal rules, and fenced code blocks.
	larryMarkdownLexer = chroma.Coalesce(chroma.MustNewLexer(
		&chroma.Config{
			Name:      "larry-markdown",
			Aliases:   []string{"md", "markdown"},
			Filenames: []string{"*.md", "*.markdown", "*.mdown", "*.mkd", "*.mdx"},
			DotAll:    false,
		},
		func() chroma.Rules {
			return chroma.Rules{
				"root": {
					// Fenced code blocks with language delegation
					{Pattern: `^(` + "```" + `|~~~)([a-zA-Z0-9_+-]+)(\n)([\s\S]*?)(^(?:` + "```" + `|~~~)\s*$)`, Type: chroma.UsingByGroup(2, 4, chroma.LiteralStringBacktick, chroma.NameAttribute, chroma.Text, chroma.Text, chroma.LiteralStringBacktick)},
					{Pattern: `^(` + "```" + `|~~~)(\n)([\s\S]*?)(^(?:` + "```" + `|~~~)\s*$)`, Type: chroma.ByGroups(chroma.LiteralStringBacktick, chroma.Text, chroma.LiteralStringBacktick, chroma.LiteralStringBacktick)},
					{Pattern: `^\s*(?:` + "```" + `|~~~).*$`, Type: chroma.LiteralStringBacktick},

					// Headings
					{Pattern: `^(#)( .*)(\n|$)`, Type: chroma.ByGroups(chroma.GenericHeading, chroma.GenericHeading, chroma.Text)},
					{Pattern: `^(#{2,6})( .*)(\n|$)`, Type: chroma.ByGroups(chroma.GenericSubheading, chroma.GenericSubheading, chroma.Text)},

					// Horizontal rules
					{Pattern: `^\s*(?:---+|\*\*\*+|___+)\s*(\n|$)`, Type: chroma.Comment},

					// Task lists & bullet/numbered lists
					{Pattern: `^(\s*)([*+-]\s+)(\[[ xX]\])(\s+)(.*)(\n|$)`, Type: chroma.ByGroups(chroma.Text, chroma.Keyword, chroma.NameAttribute, chroma.Text, chroma.UsingSelf("inline"), chroma.Text)},
					{Pattern: `^(\s*)([*+-])(\s+)(.*)(\n|$)`, Type: chroma.ByGroups(chroma.Text, chroma.Keyword, chroma.Text, chroma.UsingSelf("inline"), chroma.Text)},
					{Pattern: `^(\s*)([0-9]+\.)(\s+)(.*)(\n|$)`, Type: chroma.ByGroups(chroma.Text, chroma.Keyword, chroma.Text, chroma.UsingSelf("inline"), chroma.Text)},

					// Blockquotes
					{Pattern: `^(\s*>\s?)(.*)(\n|$)`, Type: chroma.ByGroups(chroma.Keyword, chroma.GenericEmph, chroma.Text)},

					chroma.Include("inline"),
				},
				"inline": {
					{Pattern: `\\.`, Type: chroma.Text},
					{Pattern: `<!--`, Type: chroma.Comment, Mutator: chroma.Push("html_comment")},
					{Pattern: `<\s*/\s*[\w:.-]+\s*>`, Type: chroma.NameTag},
					{Pattern: `<\s*[\w:.-]+`, Type: chroma.NameTag, Mutator: chroma.Push("html_tag")},
					{Pattern: "`[^`\n]+`", Type: chroma.LiteralStringBacktick},
					{Pattern: `(!?\[)([^\]]+)(\])(\()([^)]+)(\))`, Type: chroma.ByGroups(chroma.Punctuation, chroma.NameTag, chroma.Punctuation, chroma.Punctuation, chroma.NameAttribute, chroma.Punctuation)},
					{Pattern: `(\*\*|__)([^*\n_]+)(\1)`, Type: chroma.ByGroups(chroma.GenericStrong, chroma.GenericStrong, chroma.GenericStrong)},
					{Pattern: `(~~)([^~\n]+)(~~)`, Type: chroma.ByGroups(chroma.GenericDeleted, chroma.GenericDeleted, chroma.GenericDeleted)},
					{Pattern: `(\*)([^*\n]+)(\*)`, Type: chroma.ByGroups(chroma.GenericEmph, chroma.GenericEmph, chroma.GenericEmph)},
					{Pattern: `&[a-zA-Z0-9#]+;`, Type: chroma.NameEntity},
					{Pattern: `.|\n`, Type: chroma.Text},
				},
				"html_comment": {
					{Pattern: `-->`, Type: chroma.Comment, Mutator: chroma.Pop(1)},
					{Pattern: `[^-]+|-`, Type: chroma.Comment},
				},
				"html_tag": {
					{Pattern: `\s+`, Type: chroma.Text},
					{Pattern: `[\w.:-]+\s*=`, Type: chroma.NameAttribute, Mutator: chroma.Push("html_attr")},
					{Pattern: `/?\s*>`, Type: chroma.NameTag, Mutator: chroma.Pop(1)},
					{Pattern: `[\w.:-]+`, Type: chroma.NameAttribute},
					{Pattern: `.`, Type: chroma.Text, Mutator: chroma.Pop(1)},
				},
				"html_attr": {
					{Pattern: `\s+`, Type: chroma.Text},
					{Pattern: `"[^"]*"`, Type: chroma.LiteralString, Mutator: chroma.Pop(1)},
					{Pattern: `'[^']*'`, Type: chroma.LiteralString, Mutator: chroma.Pop(1)},
					{Pattern: `[^\s>]+`, Type: chroma.LiteralString, Mutator: chroma.Pop(1)},
				},
			}
		},
	).SetRegistry(lexers.GlobalLexerRegistry))

	xmlExtensions = map[string]bool{
		".xml":             true,
		".xsl":             true,
		".xslt":            true,
		".xsd":             true,
		".wsdl":            true,
		".wsf":             true,
		".rss":             true,
		".svg":             true,
		".csproj":          true,
		".vcxproj":         true,
		".fsproj":          true,
		".vbproj":          true,
		".iml":             true,
		".plist":           true,
		".xaml":            true,
		".axaml":           true,
		".xhtml":           true,
		".pom":             true,
		".nuspec":          true,
		".props":           true,
		".targets":         true,
		".manifest":        true,
		".storyboard":      true,
		".xib":             true,
		".glade":           true,
		".ui":              true,
		".gpx":             true,
		".kml":             true,
		".resx":            true,
		".xcscheme":        true,
		".xcworkspacedata": true,
	}

	markdownExtensions = map[string]bool{
		".md":       true,
		".markdown": true,
		".mdown":    true,
		".mkd":      true,
		".mdx":      true,
	}
)

type fileSyntaxCache struct {
	filename      string
	contentVer    int
	themeVer      int
	profile       termenv.Profile
	lineCount     int
	fullFileValid bool
	lines         [][]lipgloss.Style
	ansiLines     [][]ansiStyle
	ansiLineCache map[int][]ansiStyle
}

func init() {
	if lipgloss.HasDarkBackground() {
		SetTheme("dracula")
	} else {
		SetTheme("github")
	}
}

func isOmarchyThemeName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "omarchy" || n == "auto"
}

func clearSyntaxCache() {
	themeVersion++
	lipglossCache.Range(func(key, value interface{}) bool {
		lipglossCache.Delete(key)
		return true
	})
	ansiTokenCache.Range(func(key, value interface{}) bool {
		ansiTokenCache.Delete(key)
		return true
	})
}

func ensureANSITokenCacheProfile() termenv.Profile {
	prof := lipgloss.ColorProfile()
	ansiCacheProfMu.Lock()
	if ansiCacheProfile != prof {
		ansiTokenCache.Range(func(key, _ any) bool {
			ansiTokenCache.Delete(key)
			return true
		})
		ansiCacheProfile = prof
	}
	ansiCacheProfMu.Unlock()
	return prof
}

func getANSIStyleForToken(tokenType chroma.TokenType) ansiStyle {
	if s, ok := ansiTokenCache.Load(tokenType); ok {
		return s.(ansiStyle)
	}
	compiled := compileANSIStyle(getStyleForToken(tokenType))
	ansiTokenCache.Store(tokenType, compiled)
	return compiled
}

func tokenizeLinesANSI(lines []string, filename string) [][]ansiStyle {
	result := make([][]ansiStyle, len(lines))
	for i, line := range lines {
		result[i] = make([]ansiStyle, len([]rune(line)))
	}
	if len(lines) == 0 {
		return result
	}

	sample := lines[0]
	if strings.TrimSpace(sample) == "" {
		for i := 1; i < len(lines) && i < 10; i++ {
			if strings.TrimSpace(lines[i]) != "" {
				sample = lines[i]
				break
			}
		}
	}

	lexer := resolveLexer(filename, sample)
	fullText := strings.Join(lines, "\n") + "\n"
	iterator, err := lexer.Tokenise(nil, fullText)
	if err != nil {
		return result
	}

	row := 0
	col := 0
	for _, token := range iterator.Tokens() {
		style := getANSIStyleForToken(token.Type)
		for _, r := range token.Value {
			if r == '\n' {
				row++
				col = 0
				continue
			}
			if row < len(result) && col < len(result[row]) {
				result[row][col] = style
				col++
			}
		}
	}

	return result
}

func getLineANSIStyles(line string, filename string) []ansiStyle {
	runes := []rune(line)
	result := make([]ansiStyle, len(runes))
	if len(runes) == 0 {
		return result
	}

	lexer := resolveLexer(filename, line)
	iterator, err := lexer.Tokenise(nil, line+"\n")
	if err != nil {
		return result
	}

	cursor := 0
	for _, token := range iterator.Tokens() {
		style := getANSIStyleForToken(token.Type)
		for _, r := range token.Value {
			if r == '\n' {
				continue
			}
			if cursor < len(result) {
				result[cursor] = style
				cursor++
			}
		}
	}
	return result
}

func (m *Model) getANSIStylesForLine(lineNum int) []ansiStyle {
	if lineNum < 0 || lineNum >= len(m.Lines) {
		return nil
	}
	prof := ensureANSITokenCacheProfile()
	if m.syntaxCache != nil {
		sc := m.syntaxCache
		if sc.filename != m.FileName ||
			sc.contentVer != m.visualLineVersion ||
			sc.themeVer != themeVersion ||
			sc.profile != prof ||
			sc.lineCount != len(m.Lines) {
			sc.filename = m.FileName
			sc.contentVer = m.visualLineVersion
			sc.themeVer = themeVersion
			sc.profile = prof
			sc.lineCount = len(m.Lines)
			sc.fullFileValid = false
			sc.lines = nil
			sc.ansiLines = nil
			sc.ansiLineCache = nil

			if len(m.Lines) <= maxFullSyntaxLines {
				totalBytes := 0
				for _, l := range m.Lines {
					totalBytes += len(l) + 1
					if totalBytes > maxFullSyntaxBytes {
						break
					}
				}
				if totalBytes <= maxFullSyntaxBytes {
					sc.ansiLines = tokenizeLinesANSI(m.Lines, m.FileName)
					sc.fullFileValid = true
				}
			}
		}
		if sc.fullFileValid && lineNum < len(sc.ansiLines) {
			return sc.ansiLines[lineNum]
		}
		// Per-line cache for huge files (>5000 lines or >256KB) so scrolling/navigation stays O(1).
		if sc.ansiLineCache == nil {
			sc.ansiLineCache = make(map[int][]ansiStyle, 64)
		}
		if cached, ok := sc.ansiLineCache[lineNum]; ok {
			return cached
		}
		if len(sc.ansiLineCache) > 512 {
			sc.ansiLineCache = make(map[int][]ansiStyle, 64)
		}
		computed := getLineANSIStyles(m.Lines[lineNum], m.FileName)
		sc.ansiLineCache[lineNum] = computed
		return computed
	}

	return getLineANSIStyles(m.Lines[lineNum], m.FileName)
}

func SetTheme(themeName string) {
	if isOmarchyThemeName(themeName) {
		if colors, path, err := LoadCurrentOmarchyColors(); err == nil {
			if customStyle, styleErr := BuildOmarchyChromaStyle(colors); styleErr == nil {
				theme = customStyle
				activePalette = paletteFromOmarchy(colors)
				omarchyActive = true
				paintCanvasBg = false
				omarchyColorPath = path
				omarchySignature = OmarchyThemeSignature(path)
				clearSyntaxCache()
				initStyles()
				return
			}
		}
		// Fallback when Omarchy is not installed
		if lipgloss.HasDarkBackground() {
			themeName = "dracula"
		} else {
			themeName = "github"
		}
	}

	omarchyActive = false
	omarchyColorPath = ""
	omarchySignature = ""

	if cs, ok := LookupColorScheme(themeName); ok {
		if customStyle, err := BuildChromaStyleFromColors(cs.ID, cs.Colors); err == nil {
			theme = customStyle
			activePalette = paletteFromOmarchy(cs.Colors)
			paintCanvasBg = true
			clearSyntaxCache()
			initStyles()
			return
		}
	}

	activePalette = defaultPalette(lipgloss.HasDarkBackground())
	paintCanvasBg = !activePalette.IsDark

	t := styles.Get(themeName)
	if t == nil {
		t = styles.Fallback
	}
	theme = t
	clearSyntaxCache()
	initStyles()
}

func resolveLexer(filename string, sampleContent string) chroma.Lexer {
	ext := strings.ToLower(filepath.Ext(filename))

	if markdownExtensions[ext] {
		return larryMarkdownLexer
	}
	if xmlExtensions[ext] {
		if l := lexers.Get("xml"); l != nil {
			return chroma.Coalesce(l)
		}
	}
	if ext == ".arb" {
		if l := lexers.Get("json"); l != nil {
			return chroma.Coalesce(l)
		}
	}

	if filename != "" {
		if cached, ok := lexerCache.Load(filename); ok {
			return cached.(chroma.Lexer)
		}
	}

	var lexer chroma.Lexer
	if filename != "" {
		lexer = lexers.Match(filename)
	}

	if lexer == nil && sampleContent != "" {
		trimmed := strings.TrimSpace(sampleContent)
		if strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<svg") {
			lexer = lexers.Get("xml")
		} else if strings.HasPrefix(strings.ToLower(trimmed), "<!doctype html") || strings.HasPrefix(strings.ToLower(trimmed), "<html") {
			lexer = lexers.Get("html")
		} else {
			lexer = lexers.Analyse(sampleContent)
		}
	}

	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	if filename != "" && lexer != lexers.Fallback {
		lexerCache.Store(filename, lexer)
	}
	return lexer
}

// GetLineStyles tokenizes a single line (appending a synthetic newline so
// line-oriented lexers such as Markdown, YAML, and Shell match reliably).
func GetLineStyles(line string, filename string) []lipgloss.Style {
	runes := []rune(line)
	result := make([]lipgloss.Style, len(runes))
	if len(runes) == 0 {
		return result
	}

	lexer := resolveLexer(filename, line)
	iterator, err := lexer.Tokenise(nil, line+"\n")
	if err != nil {
		return result
	}

	defaultStyle := lipgloss.NewStyle()
	cursor := 0
	for _, token := range iterator.Tokens() {
		style := getStyleForToken(token.Type)
		for _, r := range token.Value {
			if r == '\n' {
				continue
			}
			if cursor < len(result) {
				result[cursor] = style
				cursor++
			}
		}
	}

	for i := cursor; i < len(result); i++ {
		result[i] = defaultStyle
	}
	return result
}

// TokenizeLines tokenizes an entire document at once so multi-line constructs
// (XML multi-line tags/comments, Markdown fenced code blocks, block comments)
// highlight accurately across lines.
func TokenizeLines(lines []string, filename string) [][]lipgloss.Style {
	result := make([][]lipgloss.Style, len(lines))
	for i, line := range lines {
		result[i] = make([]lipgloss.Style, len([]rune(line)))
	}
	if len(lines) == 0 {
		return result
	}

	sample := lines[0]
	if strings.TrimSpace(sample) == "" {
		for i := 1; i < len(lines) && i < 10; i++ {
			if strings.TrimSpace(lines[i]) != "" {
				sample = lines[i]
				break
			}
		}
	}

	lexer := resolveLexer(filename, sample)
	fullText := strings.Join(lines, "\n") + "\n"
	iterator, err := lexer.Tokenise(nil, fullText)
	if err != nil {
		return result
	}

	row := 0
	col := 0
	for _, token := range iterator.Tokens() {
		style := getStyleForToken(token.Type)
		for _, r := range token.Value {
			if r == '\n' {
				row++
				col = 0
				continue
			}
			if row < len(result) && col < len(result[row]) {
				result[row][col] = style
				col++
			}
		}
	}

	return result
}

func (m Model) getStylesForLine(lineNum int) []lipgloss.Style {
	if lineNum < 0 || lineNum >= len(m.Lines) {
		return nil
	}
	if m.syntaxCache != nil {
		sc := m.syntaxCache
		if sc.filename != m.FileName ||
			sc.contentVer != m.visualLineVersion ||
			sc.themeVer != themeVersion ||
			sc.lineCount != len(m.Lines) {
			sc.filename = m.FileName
			sc.contentVer = m.visualLineVersion
			sc.themeVer = themeVersion
			sc.lineCount = len(m.Lines)
			sc.fullFileValid = false
			sc.lines = nil

			if len(m.Lines) <= maxFullSyntaxLines {
				totalBytes := 0
				for _, l := range m.Lines {
					totalBytes += len(l) + 1
					if totalBytes > maxFullSyntaxBytes {
						break
					}
				}
				if totalBytes <= maxFullSyntaxBytes {
					sc.lines = TokenizeLines(m.Lines, m.FileName)
					sc.fullFileValid = true
				}
			}
		}
		if sc.fullFileValid && lineNum < len(sc.lines) {
			return sc.lines[lineNum]
		}
	}

	return GetLineStyles(m.Lines[lineNum], m.FileName)
}

func getStyleForToken(tokenType chroma.TokenType) lipgloss.Style {
	if s, ok := lipglossCache.Load(tokenType); ok {
		return s.(lipgloss.Style)
	}

	entry := theme.Get(tokenType)
	style := withCanvasBg(lipgloss.NewStyle())

	hasColor := entry.Colour.IsSet()
	if hasColor {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	} else if paintCanvasBg && activePalette.Foreground != "" {
		style = style.Foreground(activePalette.Foreground)
	}
	if entry.Bold == chroma.Yes {
		style = style.Bold(true)
	}
	if entry.Italic == chroma.Yes {
		style = style.Italic(true)
	}
	if entry.Underline == chroma.Yes {
		style = style.Underline(true)
	}

	// Ensure Markdown and XML tokens always have visible semantic styling even
	// in Chroma themes that omit entries for them (e.g. GenericHeading in github).
	p := activePalette
	switch tokenType {
	case chroma.GenericHeading:
		if !hasColor {
			style = style.Foreground(p.Accent).Bold(true)
		}
	case chroma.GenericSubheading:
		if !hasColor {
			style = style.Foreground(p.Blue).Bold(true)
		}
	case chroma.GenericStrong:
		style = style.Bold(true)
		if !hasColor {
			style = style.Foreground(p.Foreground)
		}
	case chroma.GenericEmph:
		style = style.Italic(true)
	case chroma.GenericDeleted:
		if !hasColor {
			style = style.Foreground(p.Red)
		}
	case chroma.LiteralStringBacktick:
		if !hasColor {
			style = style.Foreground(p.Cyan)
		}
	case chroma.NameTag:
		if !hasColor {
			style = style.Foreground(p.Red)
		}
	case chroma.NameAttribute:
		if !hasColor {
			style = style.Foreground(p.Blue)
		}
	case chroma.NameEntity:
		if !hasColor {
			style = style.Foreground(p.Yellow)
		}
	case chroma.CommentPreproc:
		if !hasColor {
			style = style.Foreground(p.Cyan)
		}
	}

	lipglossCache.Store(tokenType, style)
	return style
}

// GetSelectionStyle returns a lipgloss style for text selection based on the current theme
func GetSelectionStyle() lipgloss.Style {
	entry := theme.Get(chroma.GenericStrong)

	style := lipgloss.NewStyle()

	if entry.Background.IsSet() {
		style = style.Background(lipgloss.Color(entry.Background.String()))
	} else {
		style = style.Background(activePalette.Selection)
	}

	if entry.Colour.IsSet() {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	} else {
		style = style.Foreground(activePalette.SelectionFg)
	}

	return style
}
