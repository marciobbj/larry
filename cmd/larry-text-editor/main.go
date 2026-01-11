// cmd/larry-text-editor/main.go
// Main entry point for the Larry Text Editor.
// This program starts a Bubble Tea TUI application for a minimalist text editor.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"larry-text-editor/internal/ui"
)

func main() {
	// Initialize the model
	m := ui.InitialModel()

	// Create and run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen()) // Use alternate screen for clean TUI

	// Run the program and handle any errors
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running the text editor: %v\n", err)
		os.Exit(1)
	}
}
