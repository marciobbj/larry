// Package search provides efficient string searching algorithms for the text editor.
package search

import (
	"context"
)

type SearchMatch struct {
	Line   int
	Col    int
	Length int
}

type BoyerMooreSearch struct {
	pattern     string
	patternLen  int
	badCharSkip []int
}

func NewBoyerMooreSearch(pattern string) *BoyerMooreSearch {
	if pattern == "" {
		return &BoyerMooreSearch{}
	}

	bms := &BoyerMooreSearch{
		pattern:     pattern,
		patternLen:  len(pattern),
		badCharSkip: make([]int, 256), // ASCII characters
	}

	// Initialize bad character skip table with pattern length
	for i := range bms.badCharSkip {
		bms.badCharSkip[i] = bms.patternLen
	}

	// Fill bad character skip table with actual positions
	for i := 0; i < bms.patternLen-1; i++ {
		bms.badCharSkip[pattern[i]] = bms.patternLen - 1 - i
	}

	return bms
}

func (bms *BoyerMooreSearch) SearchInText(text string) []SearchMatch {
	if bms.patternLen == 0 {
		return nil
	}

	var matches []SearchMatch
	textLen := len(text)
	i := bms.patternLen - 1

	for i < textLen {
		k := bms.patternLen - 1
		j := i

		// Check for match from right to left
		for k >= 0 && text[j] == bms.pattern[k] {
			j--
			k--
		}

		if k < 0 {
			// Found a match at position j+1
			// Since we're working with flat text, we need to convert back to line/col
			// But for now, we'll return absolute positions and convert later
			matches = append(matches, SearchMatch{
				Line:   -1, // Will be set later when converting
				Col:    j + 1,
				Length: bms.patternLen,
			})
			i += bms.patternLen
		} else {
			// Skip based on bad character heuristic
			skip := bms.badCharSkip[text[i]]
			if skip < 1 {
				skip = 1
			}
			i += skip
		}
	}

	return matches
}

// SearchInLines searches for the pattern in multiple lines and returns matches with line/column positions
func (bms *BoyerMooreSearch) SearchInLines(ctx context.Context, lines []string) []SearchMatch {
	if bms.patternLen == 0 {
		return nil
	}

	var matches []SearchMatch
	// Pre-allocate assuming sparse matches to avoid frequent resizing
	matches = make([]SearchMatch, 0, 100)

	// Check context every N lines to avoid overhead
	const checkInterval = 1000

	for lineIdx, line := range lines {
		if lineIdx%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
		}

		lineMatches := bms.SearchInText(line)
		for _, match := range lineMatches {
			matches = append(matches, SearchMatch{
				Line:   lineIdx,
				Col:    match.Col,
				Length: match.Length,
			})
		}
	}

	return matches
}
