// internal/ui/model.go
// Package ui defines the main model for the Bubble Tea TUI application.
// It manages the editor's state, including textarea for editing.
package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyMap defines key bindings for editor shortcuts.
// This allows easy customization of key mappings.
type KeyMap struct {
	Quit key.Binding
}

// DefaultKeyMap provides the default key bindings.
var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(key.WithKeys("ctrl+c", "q")),
}

// Model represents the state of the text editor.
type Model struct {
	TextArea textarea.Model // Text area for editing with cursor support
	Width    int            // Terminal width
	Height   int            // Terminal height
	FileName string         // Current file name (for save/load)
	KeyMap   KeyMap         // Key bindings for shortcuts
	Quitting bool           // Flag to indicate if the app is quitting
}

// InitialModel creates and returns a new initial model.
func InitialModel() Model {
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.Placeholder = "Start typing..."
	ta.Focus()

	return Model{
		TextArea: ta,
		Width:    80,
		Height:   20,
		FileName: "untitled.txt",
		KeyMap:   DefaultKeyMap,
		Quitting: false,
	}
}

// Init initializes the model. No initial commands needed.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state.
// It intercepts quit commands and delegates other input to the textarea.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Intercept quit before passing to textarea
		if key.Matches(msg, m.KeyMap.Quit) {
			m.Quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Update textarea size on terminal resize
		m.Width = msg.Width
		m.Height = msg.Height
		m.TextArea.SetWidth(msg.Width)
		m.TextArea.SetHeight(msg.Height)
	}

	// Delegate other messages to textarea
	var taCmd tea.Cmd
	m.TextArea, taCmd = m.TextArea.Update(msg)
	return m, taCmd
}

// View renders the current state of the model as a string.
// It displays the textarea with cursor.
func (m Model) View() string {
	if m.Quitting {
		return "Goodbye!\n"
	}

	// Render the textarea (cursor is handled by textarea)
	return m.TextArea.View()
}
