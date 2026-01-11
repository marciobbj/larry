// cmd/larry/main.go
// Main entry point for the Larry Text Editor.
// This program starts a Bubble Tea TUI application for a minimalist text editor.
package main

import (
	"flag"
	"fmt"
	"os"
	"larry/internal/config"
	"larry/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Initialize the system clipboard
	err := clipboard.Init()
	if err != nil {
		fmt.Printf("Warning: Could not initialize clipboard: %v\n", err)
		// Continue anyway - clipboard features may not work
	}

	// Handle CLI arguments
	filename := ""
	content := ""
	args := flag.Args()
	if len(args) > 0 {
		filename = args[0]
		data, err := os.ReadFile(filename)
		if err == nil {
			content = string(data)
		}
		// If read fails (e.g. new file), we start with empty content and the filename
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		// Only print error if user explicitly provided a path that failed
		if *configPath != "" {
			fmt.Printf("Warning: Could not load config: %v. Using defaults.\n", err)
		}
		// If no path provided or fallback, LoadConfig returns valid cfg (default) usually,
	}

	// Initialize the model
	m := ui.InitialModel(filename, content, cfg)

	// Create and run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen()) // Use alternate screen for clean TUI

	// Run the program and handle any errors
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running the text editor: %v\n", err)
		os.Exit(1)
	}
}
