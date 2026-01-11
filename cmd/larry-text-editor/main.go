// cmd/larry-text-editor/main.go
// Main entry point for the Larry Text Editor.
// This program starts a Bubble Tea TUI application for a minimalist text editor.
package main

import (
	"fmt"
	"os"

	"larry-text-editor/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

func main() {
	// Initialize the system clipboard
	err := clipboard.Init()
	if err != nil {
		fmt.Printf("Warning: Could not initialize clipboard: %v\n", err)
		// Continue anyway - clipboard features may not work
	}

	// Handle CLI arguments
	filename := ""
	content := ""
	if len(os.Args) > 1 {
		filename = os.Args[1]
		data, err := os.ReadFile(filename)
		if err == nil {
			content = string(data)
		}
		// If read fails (e.g. new file), we start with empty content and the filename
	}

	// Initialize the model
	m := ui.InitialModel(filename, content)

	// Create and run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen()) // Use alternate screen for clean TUI

	// Run the program and handle any errors
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running the text editor: %v\n", err)
		os.Exit(1)
	}
}
