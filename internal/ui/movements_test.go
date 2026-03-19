package ui

import "testing"

func TestFindNextWordBoundaryStaysOnLineEnd(t *testing.T) {
	row, col := FindNextWordBoundary([]string{"alpha", "beta"}, 0, 0)
	if row != 0 || col != 4 {
		t.Fatalf("FindNextWordBoundary() = (%d,%d), want (0,4)", row, col)
	}
}

func TestFindPrevWordBoundaryStaysOnLineStart(t *testing.T) {
	row, col := FindPrevWordBoundary([]string{"alpha", "beta"}, 1, 0)
	if row != 1 || col != 0 {
		t.Fatalf("FindPrevWordBoundary() = (%d,%d), want (1,0)", row, col)
	}
}
