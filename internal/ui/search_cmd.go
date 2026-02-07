package ui

import (
	"context"
	"larry/internal/search"

	tea "github.com/charmbracelet/bubbletea"
)

// SearchResultsMsg carries the results of a background search
type SearchResultsMsg struct {
	Query     string
	Results   []search.SearchMatch
	IsReplace bool // True if this is for the replace feature
}

// PerformSearchCmd creates a command to run the search in a goroutine
func PerformSearchCmd(lines []string, query string, isReplace bool) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := func() tea.Msg {
		if query == "" {
			return SearchResultsMsg{Query: query, Results: nil, IsReplace: isReplace}
		}

		// Passing 'lines' here passes the slice header at the moment of call.
		// Since strings are immutable in Go, searching the old lines is thread-safe
		// even if the main thread replaces them with new strings in a new slice.
		searcher := search.NewBoyerMooreSearch(query)
		results := searcher.SearchInLines(ctx, lines)

		if ctx.Err() != nil {
			return nil
		}

		return SearchResultsMsg{
			Query:     query,
			Results:   results,
			IsReplace: isReplace,
		}
	}

	return cmd, cancel
}
