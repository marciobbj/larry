// internal/buffer/buffer.go
// Package buffer handles the text buffer operations for the text editor.
// It provides methods to manipulate the internal data structure ([][]rune) representing lines of text.
package buffer

// Buffer represents the text buffer as a slice of rune slices.
// Each inner slice is a line of text.
type Buffer struct {
	Lines [][]rune
}

// NewBuffer creates a new empty buffer with one empty line.
func NewBuffer() *Buffer {
	return &Buffer{
		Lines: [][]rune{{}}, // Start with one empty line
	}
}

// InsertRune inserts a rune at the specified position (line, col).
// If the line or column is out of bounds, it adjusts accordingly.
func (b *Buffer) InsertRune(line, col int, r rune) {
	if line < 0 || line >= len(b.Lines) {
		return
	}
	if col < 0 {
		col = 0
	}
	if col > len(b.Lines[line]) {
		col = len(b.Lines[line])
	}

	// Insert the rune at the specified column in the line
	before := b.Lines[line][:col]
	after := b.Lines[line][col:]
	b.Lines[line] = append(before, append([]rune{r}, after...)...)
}

// DeleteRune deletes a rune at the specified position (line, col).
// If out of bounds, does nothing.
func (b *Buffer) DeleteRune(line, col int) {
	if line < 0 || line >= len(b.Lines) || col < 0 || col >= len(b.Lines[line]) {
		return
	}

	// Remove the rune at the specified column
	before := b.Lines[line][:col]
	after := b.Lines[line][col+1:]
	b.Lines[line] = append(before, after...)
}

// GetLine returns the string representation of a line at the given index.
// Returns empty string if line is out of bounds.
func (b *Buffer) GetLine(line int) string {
	if line < 0 || line >= len(b.Lines) {
		return ""
	}
	return string(b.Lines[line])
}

// GetAllLines returns all lines as a slice of strings.
func (b *Buffer) GetAllLines() []string {
	lines := make([]string, len(b.Lines))
	for i, line := range b.Lines {
		lines[i] = string(line)
	}
	return lines
}
