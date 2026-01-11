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
	theme = styles.Get("dracula")
	if theme == nil {
		theme = styles.Fallback
	}
}

// GetLineStyles returns a slice of lipgloss.Style for each character in the line
func GetLineStyles(line string, filename string) []lipgloss.Style {
	// 1. Get Lexer
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

	// 2. Tokenize line
	iterator, err := lexer.Tokenise(nil, line)
	if err != nil {
		return make([]lipgloss.Style, len([]rune(line)))
	}

	// 3. Build styles slice
	// Note: line runes count might differ from byte count. Chroma works on strings (bytes/runes mixed).
	// We need to map tokens to runes.
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
