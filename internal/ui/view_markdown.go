package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) initMarkdownRenderer(width int) error {
	styleCfg := styles.DarkStyleConfig
	if !lipgloss.HasDarkBackground() {
		styleCfg = styles.LightStyleConfig
	}

	styleCfg.H1.Prefix = ""
	styleCfg.H2.Prefix = ""
	styleCfg.H3.Prefix = ""
	styleCfg.H4.Prefix = ""
	styleCfg.H5.Prefix = ""
	styleCfg.H6.Prefix = ""

	styleCfg.H1.Suffix = ""
	styleCfg.H2.Suffix = ""
	styleCfg.H3.Suffix = ""
	styleCfg.H4.Suffix = ""
	styleCfg.H5.Suffix = ""
	styleCfg.H6.Suffix = ""

	underline := true
	styleCfg.H1.Underline = &underline
	styleCfg.H2.Underline = &underline

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styleCfg),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return err
	}

	m.markdownRenderer = renderer
	return nil
}

func (m Model) viewMarkdownPreview(width, height int) string {
	if m.markdownRenderer == nil {
		return "Markdown renderer not initialized"
	}

	content := strings.Join(m.Lines, "\n")
	rendered, err := m.markdownRenderer.Render(content)
	if err != nil {
		return "Error rendering markdown: " + err.Error()
	}

	rendered = strings.TrimPrefix(rendered, "\n")
	rendered = strings.TrimSuffix(rendered, "\n")
	lines := strings.Split(rendered, "\n")

	totalSourceLines := len(m.Lines)
	if totalSourceLines == 0 {
		totalSourceLines = 1
	}

	cursorPositionRatio := float64(m.CursorRow) / float64(totalSourceLines)

	totalRenderedLines := len(lines)
	targetLine := int(cursorPositionRatio * float64(totalRenderedLines))

	maxLines := height - 1
	if maxLines < 1 {
		maxLines = 1
	}

	startLine := targetLine - (maxLines / 2)
	if startLine < 0 {
		startLine = 0
	}

	endLine := startLine + maxLines
	if endLine > totalRenderedLines {
		endLine = totalRenderedLines
		startLine = endLine - maxLines
		if startLine < 0 {
			startLine = 0
		}
	}

	if startLine >= len(lines) {
		startLine = 0
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleLines := lines[startLine:endLine]

	previewStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxHeight(height)

	return previewStyle.Render(strings.Join(visibleLines, "\n"))
}
