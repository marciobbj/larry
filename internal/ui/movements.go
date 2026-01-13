package ui

import (
	"unicode"
)

const JumpLineCount = 5

func FindNextWordBoundary(lines []string, row, col int) (int, int) {
	if len(lines) == 0 {
		return 0, 0
	}

	// Ensure valid row
	if row >= len(lines) {
		row = len(lines) - 1
	}
	if row < 0 {
		row = 0
	}

	lineRunes := []rune(lines[row])
	lineLen := len(lineRunes)

	// If at end of line, move to start of next line
	if col >= lineLen {
		if row < len(lines)-1 {
			return row + 1, 0
		}
		return row, col
	}

	// Skip current word (non-space characters)
	for col < lineLen && !unicode.IsSpace(lineRunes[col]) {
		col++
	}

	// Skip spaces
	for col < lineLen && unicode.IsSpace(lineRunes[col]) {
		col++
	}

	// If we reached end of line, try next line
	if col >= lineLen && row < len(lines)-1 {
		row++
		col = 0
		// Skip leading spaces on new line
		newLineRunes := []rune(lines[row])
		for col < len(newLineRunes) && unicode.IsSpace(newLineRunes[col]) {
			col++
		}
	}

	return row, col
}

func FindPrevWordBoundary(lines []string, row, col int) (int, int) {
	if len(lines) == 0 {
		return 0, 0
	}

	// Ensure valid row
	if row >= len(lines) {
		row = len(lines) - 1
	}
	if row < 0 {
		row = 0
	}

	lineRunes := []rune(lines[row])

	// If at start of line, move to end of previous line
	if col <= 0 {
		if row > 0 {
			row--
			lineRunes = []rune(lines[row])
			col = len(lineRunes)
		} else {
			return 0, 0
		}
	}

	// Move back one to start scanning
	if col > 0 {
		col--
	}

	// Skip spaces backwards
	for col > 0 && unicode.IsSpace(lineRunes[col]) {
		col--
	}

	// Skip current word backwards (non-space characters)
	for col > 0 && !unicode.IsSpace(lineRunes[col-1]) {
		col--
	}

	return row, col
}

func MoveToLineStart(row, col int) (int, int) {
	return row, 0
}

func MoveToLineEnd(lines []string, row int) (int, int) {
	if row < 0 || row >= len(lines) {
		return row, 0
	}
	return row, len([]rune(lines[row]))
}

func MoveToFileStart() (int, int) {
	return 0, 0
}

func MoveToFileEnd(lines []string) (int, int) {
	if len(lines) == 0 {
		return 0, 0
	}
	lastRow := len(lines) - 1
	return lastRow, len([]rune(lines[lastRow]))
}

func JumpLinesUp(lines []string, row, col int) (int, int) {
	newRow := row - JumpLineCount
	if newRow < 0 {
		newRow = 0
	}

	// Adjust column if new line is shorter
	if newRow < len(lines) {
		lineLen := len([]rune(lines[newRow]))
		if col > lineLen {
			col = lineLen
		}
	}

	return newRow, col
}

func JumpLinesDown(lines []string, row, col int) (int, int) {
	newRow := row + JumpLineCount
	maxRow := len(lines) - 1
	if maxRow < 0 {
		maxRow = 0
	}
	if newRow > maxRow {
		newRow = maxRow
	}

	// Adjust column if new line is shorter
	if newRow < len(lines) {
		lineLen := len([]rune(lines[newRow]))
		if col > lineLen {
			col = lineLen
		}
	}

	return newRow, col
}
