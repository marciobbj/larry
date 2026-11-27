package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	defaultTreeWidth    = 30
	minTreeWidth        = 16
	maxTreeWidth        = 50
	minEditorWithTree   = 20
	minTermWidthForTree = minTreeWidth + 1 + minEditorWithTree
)

// TreeNode represents a single visible row in the lateral file tree.
type TreeNode struct {
	Name       string
	Path       string
	IsDir      bool
	IsSymlink  bool
	Depth      int
	Expanded   bool
	ParentPath string
}

// FileTreeModel manages the Neovim-style collapsible lateral file explorer.
type FileTreeModel struct {
	RootPath     string
	Nodes        []TreeNode
	Cursor       int
	ScrollOffset int
	Width        int
	Height       int
	Visible      bool
	Focused      bool
	ShowHidden   bool
	ActiveFile   string
	expanded     map[string]bool
}

// NewFileTreeModel creates a new lateral file tree rooted at rootPath.
// If rootPath is empty, the current working directory is used.
func NewFileTreeModel(rootPath string) FileTreeModel {
	if rootPath == "" {
		if wd, err := os.Getwd(); err == nil {
			rootPath = wd
		} else {
			rootPath = "."
		}
	}
	if abs, err := filepath.Abs(rootPath); err == nil {
		rootPath = abs
	}
	rootPath = filepath.Clean(rootPath)

	ft := FileTreeModel{
		RootPath:   rootPath,
		Width:      defaultTreeWidth,
		Height:     20,
		Visible:    false,
		Focused:    false,
		ShowHidden: false,
		expanded:   make(map[string]bool),
	}
	ft.Rebuild()
	return ft
}

// Rebuild rescans the visible portion of the tree from RootPath while
// preserving expanded directories and the selected path when possible.
func (ft *FileTreeModel) Rebuild() {
	var selectedPath string
	if ft.Cursor >= 0 && ft.Cursor < len(ft.Nodes) {
		selectedPath = ft.Nodes[ft.Cursor].Path
	}

	ft.Nodes = ft.Nodes[:0]
	ft.walkDir(ft.RootPath, 0, "")

	if len(ft.Nodes) == 0 {
		ft.Cursor = 0
		ft.ScrollOffset = 0
		return
	}

	if selectedPath != "" {
		if idx := ft.indexOfPath(selectedPath); idx >= 0 {
			ft.Cursor = idx
		}
	}
	ft.clampCursor()
	ft.ensureCursorVisible()
}

func (ft *FileTreeModel) walkDir(dirPath string, depth int, parentPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}

	type item struct {
		entry     os.DirEntry
		isDir     bool
		isSymlink bool
	}

	items := make([]item, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !ft.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		isSymlink := entry.Type()&os.ModeSymlink != 0
		isDir := entry.IsDir()
		if isSymlink {
			if info, err := os.Stat(filepath.Join(dirPath, name)); err == nil && info.IsDir() {
				isDir = true
			}
		}

		items = append(items, item{
			entry:     entry,
			isDir:     isDir,
			isSymlink: isSymlink,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir
		}
		return strings.ToLower(items[i].entry.Name()) < strings.ToLower(items[j].entry.Name())
	})

	for _, it := range items {
		fullPath := filepath.Join(dirPath, it.entry.Name())
		isExpanded := it.isDir && ft.expanded[fullPath]

		ft.Nodes = append(ft.Nodes, TreeNode{
			Name:       it.entry.Name(),
			Path:       fullPath,
			IsDir:      it.isDir,
			IsSymlink:  it.isSymlink,
			Depth:      depth,
			Expanded:   isExpanded,
			ParentPath: parentPath,
		})

		if isExpanded {
			ft.walkDir(fullPath, depth+1, fullPath)
		}
	}
}

func (ft *FileTreeModel) indexOfPath(target string) int {
	clean := filepath.Clean(target)
	for i, n := range ft.Nodes {
		if filepath.Clean(n.Path) == clean {
			return i
		}
	}
	return -1
}

// SelectedNode returns the node currently under the cursor.
func (ft FileTreeModel) SelectedNode() (TreeNode, bool) {
	if ft.Cursor < 0 || ft.Cursor >= len(ft.Nodes) {
		return TreeNode{}, false
	}
	return ft.Nodes[ft.Cursor], true
}

// SetActiveFile marks the currently edited file and expands its parent
// directories if it lives inside RootPath.
func (ft *FileTreeModel) SetActiveFile(path string) {
	if path == "" {
		ft.ActiveFile = ""
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	ft.ActiveFile = abs

	rel, err := filepath.Rel(ft.RootPath, abs)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return
	}

	// Expand ancestor directories so the active file is visible in the tree.
	curr := filepath.Dir(abs)
	changed := false
	for curr != ft.RootPath && strings.HasPrefix(curr, ft.RootPath) {
		if !ft.expanded[curr] {
			ft.expanded[curr] = true
			changed = true
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	// If the active file is inside a hidden directory or is itself hidden, don't force ShowHidden.
	if changed {
		ft.Rebuild()
	}
	if idx := ft.indexOfPath(abs); idx >= 0 {
		ft.Cursor = idx
		ft.ensureCursorVisible()
	}
}

// ToggleExpand expands or collapses the directory at the cursor.
func (ft *FileTreeModel) ToggleExpand() {
	node, ok := ft.SelectedNode()
	if !ok || !node.IsDir {
		return
	}
	if ft.expanded[node.Path] {
		delete(ft.expanded, node.Path)
	} else {
		ft.expanded[node.Path] = true
	}
	ft.Rebuild()
}

// ExpandOrEnterChild expands a collapsed directory, or moves to its first child
// if already expanded. Returns true if the node was a directory.
func (ft *FileTreeModel) ExpandOrEnterChild() bool {
	node, ok := ft.SelectedNode()
	if !ok || !node.IsDir {
		return false
	}
	if !ft.expanded[node.Path] {
		ft.expanded[node.Path] = true
		ft.Rebuild()
		return true
	}
	if ft.Cursor+1 < len(ft.Nodes) && ft.Nodes[ft.Cursor+1].ParentPath == node.Path {
		ft.Cursor++
		ft.ensureCursorVisible()
	}
	return true
}

// CollapseOrParent collapses an expanded directory, or jumps to the parent
// directory if the current node is a file or collapsed directory.
func (ft *FileTreeModel) CollapseOrParent() {
	node, ok := ft.SelectedNode()
	if !ok {
		return
	}
	if node.IsDir && ft.expanded[node.Path] {
		delete(ft.expanded, node.Path)
		ft.Rebuild()
		return
	}
	if node.ParentPath != "" {
		if idx := ft.indexOfPath(node.ParentPath); idx >= 0 {
			ft.Cursor = idx
			ft.ensureCursorVisible()
		}
	}
}

// GoUpDirectory moves the tree root up one level to the parent directory and
// selects the previous root directory.
func (ft *FileTreeModel) GoUpDirectory() {
	parent := filepath.Dir(ft.RootPath)
	if parent == ft.RootPath {
		return
	}
	oldRoot := ft.RootPath
	ft.RootPath = parent
	ft.Rebuild()
	if idx := ft.indexOfPath(oldRoot); idx >= 0 {
		ft.Cursor = idx
		ft.ensureCursorVisible()
	}
}

// ToggleHidden toggles visibility of hidden (dot) files and folders.
func (ft *FileTreeModel) ToggleHidden() {
	ft.ShowHidden = !ft.ShowHidden
	ft.Rebuild()
}

// MoveUp moves the tree cursor up by delta rows.
func (ft *FileTreeModel) MoveUp(delta int) {
	if len(ft.Nodes) == 0 {
		return
	}
	ft.Cursor -= delta
	ft.clampCursor()
	ft.ensureCursorVisible()
}

// MoveDown moves the tree cursor down by delta rows.
func (ft *FileTreeModel) MoveDown(delta int) {
	if len(ft.Nodes) == 0 {
		return
	}
	ft.Cursor += delta
	ft.clampCursor()
	ft.ensureCursorVisible()
}

// MoveTop moves the cursor to the first item.
func (ft *FileTreeModel) MoveTop() {
	ft.Cursor = 0
	ft.ensureCursorVisible()
}

// MoveBottom moves the cursor to the last item.
func (ft *FileTreeModel) MoveBottom() {
	if len(ft.Nodes) == 0 {
		ft.Cursor = 0
		return
	}
	ft.Cursor = len(ft.Nodes) - 1
	ft.ensureCursorVisible()
}

// Resize adjusts the preferred sidebar width by delta columns.
func (ft *FileTreeModel) Resize(delta int, totalTermWidth int) {
	w := ft.Width + delta
	if w < minTreeWidth {
		w = minTreeWidth
	}
	maxAllowed := maxTreeWidth
	if totalTermWidth > 0 {
		if capW := totalTermWidth - minEditorWithTree - 1; capW < maxAllowed {
			maxAllowed = capW
		}
	}
	if maxAllowed < minTreeWidth {
		maxAllowed = minTreeWidth
	}
	if w > maxAllowed {
		w = maxAllowed
	}
	ft.Width = w
}

// EffectiveWidth computes the actual rendered width of the sidebar for the
// given terminal width. Returns 0 when hidden or when the terminal is too narrow.
func (ft FileTreeModel) EffectiveWidth(totalTermWidth int) int {
	if !ft.Visible || totalTermWidth < minTermWidthForTree {
		return 0
	}
	w := ft.Width
	if w < minTreeWidth {
		w = minTreeWidth
	}
	maxByRatio := totalTermWidth * 45 / 100
	maxByEditor := totalTermWidth - minEditorWithTree - 1
	limit := maxByRatio
	if maxByEditor < limit {
		limit = maxByEditor
	}
	if limit > maxTreeWidth {
		limit = maxTreeWidth
	}
	if limit < minTreeWidth {
		limit = minTreeWidth
	}
	if w > limit {
		w = limit
	}
	return w
}

func (ft *FileTreeModel) listHeight() int {
	// 2 rows reserved for header + horizontal rule
	h := ft.Height - 2
	if h < 1 {
		return 1
	}
	return h
}

func (ft *FileTreeModel) clampCursor() {
	if len(ft.Nodes) == 0 {
		ft.Cursor = 0
		ft.ScrollOffset = 0
		return
	}
	if ft.Cursor < 0 {
		ft.Cursor = 0
	}
	if ft.Cursor >= len(ft.Nodes) {
		ft.Cursor = len(ft.Nodes) - 1
	}
}

func (ft *FileTreeModel) ensureCursorVisible() {
	listH := ft.listHeight()
	if ft.Cursor < ft.ScrollOffset {
		ft.ScrollOffset = ft.Cursor
	}
	if ft.Cursor >= ft.ScrollOffset+listH {
		ft.ScrollOffset = ft.Cursor - listH + 1
	}
	if ft.ScrollOffset < 0 {
		ft.ScrollOffset = 0
	}
	maxScroll := len(ft.Nodes) - listH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ft.ScrollOffset > maxScroll {
		ft.ScrollOffset = maxScroll
	}
}

// View renders the lateral file tree with exact width and height dimensions.
func (ft FileTreeModel) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	ft.Height = height
	ft.ensureCursorVisible()

	lines := make([]string, 0, height)

	// Header row
	rootName := filepath.Base(ft.RootPath)
	if rootName == "" || rootName == "." {
		rootName = ft.RootPath
	}
	headerText := " ▾ " + rootName + "/"
	if ft.ShowHidden {
		headerText += " [.*]"
	}
	headerText = padOrTruncate(headerText, width)
	if ft.Focused {
		lines = append(lines, treeHeaderFocusedStyle.Render(headerText))
	} else {
		lines = append(lines, treeHeaderStyle.Render(headerText))
	}

	// Horizontal rule below header
	if height >= 2 {
		lines = append(lines, treeRuleStyle.Render(strings.Repeat("─", width)))
	}

	listH := height - len(lines)
	if listH <= 0 {
		return strings.Join(lines, "\n")
	}

	if len(ft.Nodes) == 0 {
		lines = append(lines, treeEmptyStyle.Render(padOrTruncate(" (empty)", width)))
		for len(lines) < height {
			lines = append(lines, strings.Repeat(" ", width))
		}
		return strings.Join(lines, "\n")
	}

	end := ft.ScrollOffset + listH
	if end > len(ft.Nodes) {
		end = len(ft.Nodes)
	}

	for i := ft.ScrollOffset; i < end; i++ {
		node := ft.Nodes[i]
		lines = append(lines, ft.renderNodeLine(node, i == ft.Cursor, width))
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines[:height], "\n")
}

func (ft FileTreeModel) renderNodeLine(node TreeNode, isSelected bool, width int) string {
	// Leading padding + indent guides (2 chars per depth level)
	guideRaw := " " + strings.Repeat("│ ", node.Depth)

	// Icon / marker (2 columns)
	isActive := !node.IsDir && ft.ActiveFile != "" && filepath.Clean(node.Path) == filepath.Clean(ft.ActiveFile)

	var icon string
	if node.IsDir {
		if node.Expanded {
			icon = "▾ "
		} else {
			icon = "▸ "
		}
	} else if isActive {
		icon = "● "
	} else {
		icon = "  "
	}

	name := node.Name
	if node.IsDir {
		name += "/"
	} else if node.IsSymlink {
		name += "@"
	}

	// When cursor is on this row and tree is focused, render a cohesive full-row highlight.
	if isSelected && ft.Focused {
		full := padOrTruncate(guideRaw+icon+name, width)
		return treeCursorFocusedStyle.Render(full)
	}
	if isSelected && !ft.Focused {
		full := padOrTruncate(guideRaw+icon+name, width)
		return treeCursorUnfocusedStyle.Render(full)
	}

	// Unselected row with per-part syntax/tree styling.
	guideW := runewidth.StringWidth(guideRaw)
	iconW := runewidth.StringWidth(icon)
	availNameW := width - guideW - iconW

	if availNameW <= 0 {
		// Terminal/sidebar very narrow relative to depth: gracefully truncate raw text.
		return treeFileStyle.Render(padOrTruncate(guideRaw+icon+name, width))
	}

	truncatedName := truncateDisplay(name, availNameW)
	usedW := guideW + iconW + runewidth.StringWidth(truncatedName)
	trailingPad := ""
	if usedW < width {
		trailingPad = strings.Repeat(" ", width-usedW)
	}

	var nameStyle lipgloss.Style
	var iconStyle lipgloss.Style
	switch {
	case node.IsDir:
		nameStyle = treeDirStyle
		iconStyle = treeDirStyle
	case isActive:
		nameStyle = treeActiveFileStyle
		iconStyle = treeActiveFileStyle
	case node.IsSymlink:
		nameStyle = treeSymlinkStyle
		iconStyle = treeSymlinkStyle
	default:
		nameStyle = treeFileStyle
		iconStyle = treeFileStyle
	}

	var b strings.Builder
	if node.Depth > 0 {
		b.WriteString(" ")
		b.WriteString(treeGuideStyle.Render(strings.Repeat("│ ", node.Depth)))
	} else {
		b.WriteString(" ")
	}
	b.WriteString(iconStyle.Render(icon))
	b.WriteString(nameStyle.Render(truncatedName))
	b.WriteString(trailingPad)
	return b.String()
}

// DividerView renders the 1-column vertical separator between the tree and editor.
func (ft FileTreeModel) DividerView(height int) string {
	if height <= 0 {
		return ""
	}
	style := treeDividerStyle
	if ft.Focused {
		style = treeDividerFocusedStyle
	}
	bar := style.Render("│")
	var b strings.Builder
	b.Grow(height * (len(bar) + 1))
	for i := 0; i < height; i++ {
		b.WriteString(bar)
		if i < height-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func truncateDisplay(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth == 1 {
		return "…"
	}
	return runewidth.Truncate(s, maxWidth, "…")
}

func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	truncated := truncateDisplay(s, width)
	w := runewidth.StringWidth(truncated)
	if w < width {
		return truncated + strings.Repeat(" ", width-w)
	}
	return truncated
}
