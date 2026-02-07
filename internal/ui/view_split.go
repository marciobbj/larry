package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var splitDividerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("240"))

func (m Model) viewSplit() string {
	totalWidth := m.Width
	totalHeight := m.Height - 1

	editorWidth := totalWidth / 2
	previewWidth := totalWidth - editorWidth - 1

	editorView := m.viewEditor(editorViewConfig{
		width:        editorWidth,
		height:       totalHeight,
		showSearchUI: m.searching,
	})

	previewView := m.viewMarkdownPreview(previewWidth, totalHeight)

	var dividerBuilder strings.Builder
	dividerBuilder.Grow(totalHeight * 4)
	for i := 0; i < totalHeight; i++ {
		dividerBuilder.WriteString(splitDividerStyle.Render("│"))
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
