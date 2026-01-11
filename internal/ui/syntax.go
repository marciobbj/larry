package ui

import (
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

var (
	lexerCache    sync.Map // map[string]chroma.Lexer (filename -> lexer)
	theme         *chroma.Style
	lipglossCache sync.Map // map[chroma.TokenType]lipgloss.Style
)

func init() {
	// Load a default theme (e.g., Dracula or Monokai)
	SetTheme("dracula")
}

func SetTheme(themeName string) {
	t := styles.Get(themeName)
	if t == nil {
		t = styles.Fallback
	}
	theme = t
	// Clear the cache as the theme has changed
	lipglossCache.Range(func(key, value interface{}) bool {
		lipglossCache.Delete(key)
		return true
	})
}

func GetLineStyles(line string, filename string) []lipgloss.Style {
	lexerInterface, ok := lexerCache.Load(filename)
	var lexer chroma.Lexer
	if !ok {
		// Try to match by filename
		lexer = lexers.Match(filename)
		if lexer == nil {
			// Fallback by content (too slow?) or just fallback
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)
		lexerCache.Store(filename, lexer)
	} else {
		lexer = lexerInterface.(chroma.Lexer)
	}

	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return make([]lipgloss.Style, len([]rune(line)))
	}

	runes := []rune(line)
	result := make([]lipgloss.Style, len(runes))

	// Default style
	defaultStyle := lipgloss.NewStyle()

	cursor := 0 // Rune index
	for _, token := range iterator.Tokens() {
		// Get style for this token type
		style := getStyleForToken(token.Type)

		// Token value length in RUNES
		tokenRunes := []rune(token.Value)
		length := len(tokenRunes)

		for i := 0; i < length; i++ {
			if cursor < len(result) {
				result[cursor] = style
				cursor++
			}
		}
	}

	// Fill remaining with default if any (shouldn't happen matching)
	for i := cursor; i < len(result); i++ {
		result[i] = defaultStyle
	}

	return result
}

func getStyleForToken(tokenType chroma.TokenType) lipgloss.Style {
	if s, ok := lipglossCache.Load(tokenType); ok {
		return s.(lipgloss.Style)
	}

	// Resolve style from Chroma Theme
	entry := theme.Get(tokenType)

	// Convert Chroma Style Entry to Lipgloss
	style := lipgloss.NewStyle()

	if entry.Colour.IsSet() {
		style = style.Foreground(lipgloss.Color(entry.Colour.String()))
	}
	if entry.Background.IsSet() {
		style = style.Background(lipgloss.Color(entry.Background.String()))
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

	lipglossCache.Store(tokenType, style)
	return style
}
