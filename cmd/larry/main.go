// cmd/larry/main.go
// Main entry point for the Larry Text Editor.
// This program starts a Bubble Tea TUI application for a minimalist text editor.
package main

import (
	"bufio"
	"flag"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"golang.design/x/clipboard"
	"larry/internal/config"
	"larry/internal/ui"
	"os"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	help := flag.Bool("help", false, "Show help information")
	flag.Parse()

	// Show help if requested
	if *help {
		showHelp()
		return
	}

	// Initialize the system clipboard
	err := clipboard.Init()
	if err != nil {
		fmt.Printf("Warning: Could not initialize clipboard: %v\n", err)
		// Continue anyway - clipboard features may not work
	}

	// Handle CLI arguments
	filename := ""
	var lines []string
	args := flag.Args()
	if len(args) > 0 {
		filename = args[0]
		file, err := os.Open(filename)
		if err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			// Increase buffer size to handle potential long lines (1MB)
			const maxCapacity = 1024 * 1024
			buf := make([]byte, maxCapacity)
			scanner.Buffer(buf, maxCapacity)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
		}
		// If read fails (e.g. new file), we start with empty lines and the filename
	}

	if len(lines) == 0 {
		lines = []string{""}
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
	m := ui.InitialModel(filename, lines, cfg)

	// Create and run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen()) // Use alternate screen for clean TUI

	// Run the program and handle any errors
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running the text editor: %v\n", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`Larry - The Text Editor

A minimalist, high-performance TUI text editor written in Go.

USAGE:
  larry [OPTIONS] [FILE]

ARGUMENTS:
  FILE    Optional file to open on startup

OPTIONS:
  -config string    Path to configuration file (default: uses built-in defaults)
  -help             Show this help information

EXAMPLES:
  # Start with empty file
  larry

  # Open a specific file
  larry myfile.txt

  # Open with custom configuration
  larry -config ~/.config/larry/config.json myfile.txt

  # Show help
  larry --help

CONFIGURATION:
  Larry supports configuration via a JSON file specified with -config flag.

  Configuration options:
    theme       - Syntax highlighting theme (e.g., "dracula", "monokai", "nord")
    tab_width   - Number of spaces for tab character (default: 4)
    line_numbers - Show/hide line numbers (default: true)
    leader_key  - Base key for shortcuts (default: "ctrl", use "cmd" for macOS)

  Example config.json:
    {
      "theme": "dracula",
      "tab_width": 4,
      "line_numbers": true,
      "leader_key": "ctrl"
    }

For more information, see the README.md file.`)
}
