package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit               key.Binding
	SelectAll          key.Binding
	MoveSelectionDown  key.Binding
	MoveSelectionUp    key.Binding
	MoveSelectionLeft  key.Binding
	MoveSelectionRight key.Binding
	Copy               key.Binding
	Paste              key.Binding
	Cut                key.Binding
	Save               key.Binding
	Open               key.Binding
	CursorUp           key.Binding
	CursorDown         key.Binding
	CursorLeft         key.Binding
	CursorRight        key.Binding
	Delete             key.Binding
	Undo               key.Binding
	Redo               key.Binding
	GoToLine           key.Binding
	ToggleHelp         key.Binding
	Search             key.Binding
	Replace            key.Binding
	GlobalFinder       key.Binding
	// Agile Navigation
	JumpWordLeft          key.Binding
	JumpWordRight         key.Binding
	JumpLinesUp           key.Binding
	JumpLinesDown         key.Binding
	SelectWordLeft        key.Binding
	SelectWordRight       key.Binding
	SelectLinesUp         key.Binding
	SelectLinesDown       key.Binding
	LineStart             key.Binding
	LineEnd               key.Binding
	FileStart             key.Binding
	FileEnd               key.Binding
	SelectToLineStart     key.Binding
	SelectToLineEnd       key.Binding
	ToggleMarkdownPreview key.Binding
}

func NewKeyMap(leader string) KeyMap {
	return KeyMap{
		Quit:               key.NewBinding(key.WithKeys(leader + "+q")),
		SelectAll:          key.NewBinding(key.WithKeys(leader + "+a")),
		MoveSelectionDown:  key.NewBinding(key.WithKeys("shift+down")),
		MoveSelectionLeft:  key.NewBinding(key.WithKeys("shift+left")),
		MoveSelectionRight: key.NewBinding(key.WithKeys("shift+right")),
		MoveSelectionUp:    key.NewBinding(key.WithKeys("shift+up")),
		Copy:               key.NewBinding(key.WithKeys(leader + "+c")),
		Paste:              key.NewBinding(key.WithKeys(leader + "+v")),
		Cut:                key.NewBinding(key.WithKeys(leader + "+x")),
		Save:               key.NewBinding(key.WithKeys(leader + "+s")),
		Open:               key.NewBinding(key.WithKeys(leader + "+o")),
		CursorUp:           key.NewBinding(key.WithKeys("up")),
		CursorDown:         key.NewBinding(key.WithKeys("down")),
		CursorLeft:         key.NewBinding(key.WithKeys("left")),
		CursorRight:        key.NewBinding(key.WithKeys("right")),
		Delete:             key.NewBinding(key.WithKeys("backspace", "delete")),
		Undo:               key.NewBinding(key.WithKeys(leader + "+z")),
		Redo:               key.NewBinding(key.WithKeys(leader + "+r")),
		GoToLine:           key.NewBinding(key.WithKeys(leader + "+g")),
		ToggleHelp:         key.NewBinding(key.WithKeys(leader + "+h")),
		Search:             key.NewBinding(key.WithKeys(leader + "+f")),
		Replace:            key.NewBinding(key.WithKeys(leader + "+t")),
		GlobalFinder:       key.NewBinding(key.WithKeys(leader + "+p")),
		// Agile Navigation
		JumpWordLeft:          key.NewBinding(key.WithKeys(leader + "+left")),
		JumpWordRight:         key.NewBinding(key.WithKeys(leader + "+right")),
		JumpLinesUp:           key.NewBinding(key.WithKeys(leader + "+up")),
		JumpLinesDown:         key.NewBinding(key.WithKeys(leader + "+down")),
		SelectWordLeft:        key.NewBinding(key.WithKeys(leader + "+shift+left")),
		SelectWordRight:       key.NewBinding(key.WithKeys(leader + "+shift+right")),
		SelectLinesUp:         key.NewBinding(key.WithKeys(leader + "+shift+up")),
		SelectLinesDown:       key.NewBinding(key.WithKeys(leader + "+shift+down")),
		LineStart:             key.NewBinding(key.WithKeys("home")),
		LineEnd:               key.NewBinding(key.WithKeys("end")),
		FileStart:             key.NewBinding(key.WithKeys(leader + "+home")),
		FileEnd:               key.NewBinding(key.WithKeys(leader + "+end")),
		SelectToLineStart:     key.NewBinding(key.WithKeys("shift+home")),
		SelectToLineEnd:       key.NewBinding(key.WithKeys("shift+end")),
		ToggleMarkdownPreview: key.NewBinding(key.WithKeys(leader+"+m", leader+"+shift+m", "alt+m")),
	}
}

var DefaultKeyMap = NewKeyMap("ctrl")
