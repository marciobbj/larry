package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewSplit() string {
	totalWidth := m.mainContentWidth()
	totalHeight := m.editorViewportHeight()

	editorWidth := totalWidth / 2
	previewWidth := totalWidth - editorWidth - 1
	if previewWidth < 1 {
		previewWidth = 1
	}

	editorView := m.viewEditor(editorViewConfig{
		width:          editorWidth,
		height:         totalHeight,
		showSearchUI:   m.searching,
		hideLeftBorder: m.sidebarWidth() > 0,
	})

	previewView := m.viewMarkdownPreview(previewWidth, totalHeight)

	var dividerBuilder strings.Builder
	dividerBuilder.Grow(totalHeight * 4)
	for i := 0; i < totalHeight; i++ {
		dividerBuilder.WriteString(borderStyle.Render("│"))
		if i < totalHeight-1 {
			dividerBuilder.WriteByte('\n')
		}
	}

	editorStyle := lipgloss.NewStyle().
		Width(editorWidth).
		Height(totalHeight)

	previewStyle := lipgloss.NewStyle().
		Width(previewWidth).
		Height(totalHeight)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		editorStyle.Render(editorView),
		dividerBuilder.String(),
		previewStyle.Render(previewView),
	)
}
