package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// EditorTab stores the independent buffer, cursor, viewport, selection, and
// undo/redo history of a single open file tab.
type EditorTab struct {
	FileName  string
	AbsPath   string
	Lines     []string
	CursorRow int
	CursorCol int
	YOffset   int
	XOffset   int
	StartRow  int
	StartCol  int
	Selecting bool
	UndoStack []EditOp
	RedoStack []EditOp
	Modified  bool
	ViewMode  ViewMode
}

func normalizeAbsPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func (m *Model) ensureTabsInitialized() {
	if len(m.tabs) == 0 {
		lines := m.Lines
		if len(lines) == 0 {
			lines = []string{""}
		}
		m.tabs = []EditorTab{{
			FileName:  m.FileName,
			AbsPath:   normalizeAbsPath(m.FileName),
			Lines:     lines,
			CursorRow: m.CursorRow,
			CursorCol: m.CursorCol,
			YOffset:   m.yOffset,
			XOffset:   m.xOffset,
			StartRow:  m.startRow,
			StartCol:  m.startCol,
			Selecting: m.selecting,
			UndoStack: m.UndoStack,
			RedoStack: m.RedoStack,
			Modified:  m.Modified,
			ViewMode:  m.viewMode,
		}}
		m.activeTab = 0
	}
	if m.activeTab < 0 {
		m.activeTab = 0
	}
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
}

func (m *Model) saveActiveTabState() {
	m.ensureTabsInitialized()
	m.tabs[m.activeTab] = EditorTab{
		FileName:  m.FileName,
		AbsPath:   normalizeAbsPath(m.FileName),
		Lines:     m.Lines,
		CursorRow: m.CursorRow,
		CursorCol: m.CursorCol,
		YOffset:   m.yOffset,
		XOffset:   m.xOffset,
		StartRow:  m.startRow,
		StartCol:  m.startCol,
		Selecting: m.selecting,
		UndoStack: m.UndoStack,
		RedoStack: m.RedoStack,
		Modified:  m.Modified,
		ViewMode:  m.viewMode,
	}
}

func (m Model) loadTabState(idx int) Model {
	m.ensureTabsInitialized()
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.tabs) {
		idx = len(m.tabs) - 1
	}
	m.activeTab = idx
	tab := m.tabs[idx]

	m.FileName = tab.FileName
	m.Lines = tab.Lines
	if len(m.Lines) == 0 {
		m.Lines = []string{""}
	}
	m.CursorRow = tab.CursorRow
	m.CursorCol = tab.CursorCol
	m.yOffset = tab.YOffset
	m.xOffset = tab.XOffset
	m.startRow = tab.StartRow
	m.startCol = tab.StartCol
	m.selecting = tab.Selecting
	m.UndoStack = tab.UndoStack
	m.RedoStack = tab.RedoStack
	m.Modified = tab.Modified
	m.viewMode = tab.ViewMode
	m.markdownRenderer = nil
	m.markdownCache = ""
	m.markdownCacheValid = false
	m.searchResults = nil
	m.replaceResults = nil
	m.currentResultIndex = -1
	m.currReplaceIndex = -1
	m.invalidateVisualCache()

	if m.FileName != "" {
		m.fileTree.SetActiveFile(m.FileName)
	} else {
		m.fileTree.SetActiveFile("")
	}
	m = m.syncMarkdownForLayout()
	return m.updateViewport()
}

func (m Model) findTabByAbsPath(absPath string) int {
	if absPath == "" {
		return -1
	}
	clean := filepath.Clean(absPath)
	for i, tab := range m.tabs {
		if tab.AbsPath != "" && filepath.Clean(tab.AbsPath) == clean {
			return i
		}
	}
	return -1
}

func (m Model) isEmptyInitialTab() bool {
	if len(m.tabs) != 1 || m.activeTab != 0 {
		return false
	}
	return m.FileName == "" && !m.Modified && len(m.Lines) == 1 && m.Lines[0] == ""
}

func (m Model) switchToTab(idx int) Model {
	m.ensureTabsInitialized()
	if len(m.tabs) == 0 {
		return m
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(m.tabs) {
		idx = len(m.tabs) - 1
	}
	if idx == m.activeTab {
		m.showWelcome = false
		return m
	}
	m.saveActiveTabState()
	m.showWelcome = false
	return m.loadTabState(idx)
}

func (m Model) nextTab() Model {
	m.ensureTabsInitialized()
	if len(m.tabs) <= 1 {
		return m
	}
	return m.switchToTab((m.activeTab + 1) % len(m.tabs))
}

func (m Model) prevTab() Model {
	m.ensureTabsInitialized()
	if len(m.tabs) <= 1 {
		return m
	}
	return m.switchToTab((m.activeTab - 1 + len(m.tabs)) % len(m.tabs))
}

func (m Model) newTab() Model {
	m.saveActiveTabState()
	m.tabs = append(m.tabs, EditorTab{
		Lines:    []string{""},
		ViewMode: ViewModeEditor,
	})
	m.showWelcome = false
	m = m.loadTabState(len(m.tabs) - 1)
	m.statusMsg = "New tab"
	return m
}

func (m Model) closeActiveTab() Model {
	m.ensureTabsInitialized()
	if len(m.tabs) <= 1 {
		// Reset to a clean untitled buffer and return to welcome screen.
		m.tabs = []EditorTab{{
			Lines:    []string{""},
			ViewMode: ViewModeEditor,
		}}
		m = m.loadTabState(0)
		m.showWelcome = true
		m.statusMsg = "Closed tab"
		return m
	}

	idx := m.activeTab
	m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
	if idx >= len(m.tabs) {
		idx = len(m.tabs) - 1
	}
	m = m.loadTabState(idx)
	m.statusMsg = "Closed tab"
	return m
}

// OpenFiles opens additional files as tabs (used for multi-file CLI startup).
func (m *Model) OpenFiles(paths []string) {
	for _, p := range paths {
		if updated, err := m.openFile(p); err == nil {
			*m = updated
		}
	}
}

func (m Model) reloadModifiedTabs(modifiedPaths []string) Model {
	if len(modifiedPaths) == 0 {
		return m
	}
	m.saveActiveTabState()

	modSet := make(map[string]bool, len(modifiedPaths))
	for _, p := range modifiedPaths {
		modSet[filepath.Clean(p)] = true
	}

	activeReloaded := false
	for i := range m.tabs {
		tab := &m.tabs[i]
		if tab.AbsPath == "" || !modSet[filepath.Clean(tab.AbsPath)] {
			continue
		}
		raw, err := os.ReadFile(tab.AbsPath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(raw), "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}
		tab.Lines = lines
		if tab.CursorRow >= len(lines) {
			tab.CursorRow = len(lines) - 1
		}
		if tab.CursorRow < 0 {
			tab.CursorRow = 0
		}
		lineLen := len([]rune(lines[tab.CursorRow]))
		if tab.CursorCol > lineLen {
			tab.CursorCol = lineLen
		}
		tab.UndoStack = nil
		tab.RedoStack = nil
		tab.Selecting = false
		tab.Modified = false
		if i == m.activeTab {
			activeReloaded = true
		}
	}

	if activeReloaded {
		m = m.loadTabState(m.activeTab)
	}
	return m
}

// tabDisplayLabels computes concise tab labels, automatically adding the parent
// directory name whenever two or more open tabs share the same filename.
func (m Model) tabDisplayLabels() []string {
	labels := make([]string, len(m.tabs))
	baseCounts := make(map[string]int, len(m.tabs))

	for i, tab := range m.tabs {
		name := tab.FileName
		if i == m.activeTab {
			name = m.FileName
		}
		if name == "" {
			labels[i] = "[No Name]"
			continue
		}
		base := filepath.Base(name)
		labels[i] = base
		baseCounts[base]++
	}

	for i, tab := range m.tabs {
		name := tab.FileName
		if i == m.activeTab {
			name = m.FileName
		}
		if name == "" {
			continue
		}
		base := filepath.Base(name)
		if baseCounts[base] > 1 {
			parent := filepath.Base(filepath.Dir(filepath.Clean(name)))
			if parent != "" && parent != "." && parent != string(filepath.Separator) {
				labels[i] = parent + "/" + base
			}
		}
	}

	return labels
}
