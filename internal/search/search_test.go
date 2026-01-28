package search

import (
	"context"
	"testing"
)

func TestBoyerMooreSearch(t *testing.T) {
	// Test basic search
	searcher := NewBoyerMooreSearch("test")
	lines := []string{
		"This is a test line",
		"Another line",
		"test case here",
	}

	results := searcher.SearchInLines(context.Background(), lines)
	expected := []SearchMatch{
		{Line: 0, Col: 10, Length: 4},
		{Line: 2, Col: 0, Length: 4},
	}

	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Expected result %d to be %+v, got %+v", i, expected[i], result)
		}
	}
}

func TestBoyerMooreSearchNoResults(t *testing.T) {
	searcher := NewBoyerMooreSearch("notfound")
	lines := []string{"This is a test"}

	results := searcher.SearchInLines(context.Background(), lines)
	if len(results) != 0 {
		t.Errorf("Expected no results, got %d", len(results))
	}
}

func TestBoyerMooreSearchEmptyPattern(t *testing.T) {
	searcher := NewBoyerMooreSearch("")
	lines := []string{"test"}

	results := searcher.SearchInLines(context.Background(), lines)
	if len(results) != 0 {
		t.Errorf("Expected no results for empty pattern, got %d", len(results))
	}
}

func TestBoyerMooreSearchOverlapping(t *testing.T) {
	searcher := NewBoyerMooreSearch("aa")
	lines := []string{"aaaa"}

	results := searcher.SearchInLines(context.Background(), lines)
	expected := []SearchMatch{
		{Line: 0, Col: 0, Length: 2},
		{Line: 0, Col: 2, Length: 2},
	}

	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("Expected result %d to be %+v, got %+v", i, expected[i], result)
		}
	}
}
