package ui

import "testing"

func TestFindNextWordBoundaryStaysOnLineEnd(t *testing.T) {
	row, col := FindNextWordBoundary([]string{"alpha", "beta"}, 0, 0)
	if row != 0 || col != 5 {
		t.Fatalf("FindNextWordBoundary() = (%d,%d), want (0,5)", row, col)
	}
	row, col = FindNextWordBoundary([]string{"alpha", "beta"}, row, col)
	if row != 0 || col != 5 {
		t.Fatalf("repeated FindNextWordBoundary() at EOL = (%d,%d), want (0,5)", row, col)
	}
}

func TestFindPrevWordBoundaryStaysOnLineStart(t *testing.T) {
	row, col := FindPrevWordBoundary([]string{"alpha", "beta"}, 1, 0)
	if row != 1 || col != 0 {
		t.Fatalf("FindPrevWordBoundary() = (%d,%d), want (1,0)", row, col)
	}
}

func TestFindNextWordBoundaryStopsAtEachWordInChainedCalls(t *testing.T) {
	lines := []string{"PaintingBinding.instance.imageCache.clear();"}
	wantCols := []int{
		len("PaintingBinding"),
		len("PaintingBinding.instance"),
		len("PaintingBinding.instance.imageCache"),
		len("PaintingBinding.instance.imageCache.clear"),
		len("PaintingBinding.instance.imageCache.clear();"),
	}

	row, col := 0, 0
	for i, want := range wantCols {
		row, col = FindNextWordBoundary(lines, row, col)
		if row != 0 || col != want {
			t.Fatalf("step %d: FindNextWordBoundary() = (%d,%d), want (0,%d)", i+1, row, col, want)
		}
	}
}

func TestFindPrevWordBoundaryStopsAtStartOfEachWordInChainedCalls(t *testing.T) {
	lines := []string{"PaintingBinding.instance.imageCache.clear();"}
	wantCols := []int{
		len("PaintingBinding.instance.imageCache."),
		len("PaintingBinding.instance."),
		len("PaintingBinding."),
		0,
	}

	row, col := 0, len(lines[0])
	for i, want := range wantCols {
		row, col = FindPrevWordBoundary(lines, row, col)
		if row != 0 || col != want {
			t.Fatalf("step %d: FindPrevWordBoundary() = (%d,%d), want (0,%d)", i+1, row, col, want)
		}
	}
}

func TestFindWordBoundaryHandlesSpacesPunctuationAndUnicode(t *testing.T) {
	lines := []string{"  return ação(foo_bar, 123);"}

	// Forward: stops at end of return, ação, foo_bar, 123, then EOL
	var gotForward []int
	row, col := 0, 0
	for i := 0; i < 5; i++ {
		row, col = FindNextWordBoundary(lines, row, col)
		gotForward = append(gotForward, col)
	}
	wantForward := []int{8, 13, 21, 26, 28}
	for i := range wantForward {
		if gotForward[i] != wantForward[i] {
			t.Fatalf("forward step %d: got %d, want %d (all=%v)", i+1, gotForward[i], wantForward[i], gotForward)
		}
	}

	// Backward from EOL: stops at start of 123, foo_bar, ação, return, then 0
	var gotBackward []int
	for i := 0; i < 5; i++ {
		row, col = FindPrevWordBoundary(lines, row, col)
		gotBackward = append(gotBackward, col)
	}
	wantBackward := []int{23, 14, 9, 2, 0}
	for i := range wantBackward {
		if gotBackward[i] != wantBackward[i] {
			t.Fatalf("backward step %d: got %d, want %d (all=%v)", i+1, gotBackward[i], wantBackward[i], gotBackward)
		}
	}
}
