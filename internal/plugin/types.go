package plugin

// EventName identifies a lifecycle event that plugins can subscribe to.
type EventName string

const (
	EventInit         EventName = "init"
	EventFileOpen     EventName = "file_open"
	EventBufferChange EventName = "buffer_change"
	EventBeforeSave   EventName = "before_save"
	EventAfterSave    EventName = "after_save"
	EventTabSwitch    EventName = "tab_switch"
	EventCursorMove   EventName = "cursor_move"
	EventThemeChange  EventName = "theme_change"
	EventQuit         EventName = "quit"
)

// DiagnosticSeverity follows the LSP severity numbering (1=Error..4=Hint).
type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
	SeverityInfo    DiagnosticSeverity = 3
	SeverityHint    DiagnosticSeverity = 4
)

func (s DiagnosticSeverity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	case SeverityHint:
		return "hint"
	default:
		return "info"
	}
}

// Diagnostic represents a linter or LSP diagnostic attached to a buffer line/span.
// Line, Col, and EndCol are 0-based internally.
type Diagnostic struct {
	Line     int
	Col      int
	EndCol   int
	Severity DiagnosticSeverity
	Source   string
	Code     string
	Message  string
}

// GutterSign is a custom indicator rendered in the editor's line-number gutter.
// Line is 0-based internally.
type GutterSign struct {
	Line     int
	Text     string
	Color    string // semantic name: red, yellow, green, blue, cyan, magenta, accent, muted, or #hex
	Priority int
}

// TextHighlight styles an inline column range [Col, EndCol) on a 0-based Line.
// FG and BG accept semantic palette names or hex colors (e.g. "#ff5555").
type TextHighlight struct {
	Line      int
	Col       int
	EndCol    int
	FG        string
	BG        string
	Bold      bool
	Italic    bool
	Underline bool
	Priority  int
}

// StatusBadge is a compact segment rendered in the status bar.
type StatusBadge struct {
	ID       string
	Text     string
	Color    string
	Priority int
}

// CommandInfo describes a command registered by a plugin (or built-in).
type CommandInfo struct {
	ID          string
	Title       string
	Description string
	Category    string
	Key         string
	Plugin      string
}

// KeymapInfo describes a keybinding registered by a plugin.
type KeymapInfo struct {
	Key    string
	Desc   string
	Plugin string
}

// PickerItem represents a selectable item inside a plugin-opened fuzzy picker modal.
type PickerItem struct {
	ID     string
	Label  string
	Detail string
	Path   string
	Line   int // 0-based
	Col    int // 0-based
}

// PluginInfo reports discovery and load status for a single plugin.
type PluginInfo struct {
	Name        string
	Path        string
	Version     string
	Description string
	Author      string
	BuiltIn     bool
	Loaded      bool
	Err         string
}

// ExecOptions configures synchronous or asynchronous external command execution.
type ExecOptions struct {
	Dir       string
	Stdin     string
	Env       map[string]string
	TimeoutMs int
}

// ExecResult captures the outcome of running an external command.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      string
}

// AsyncEvent is delivered from background goroutines (exec_async, timers, LSP
// responses/notifications) back to Bubble Tea's main Update loop so Lua
// callbacks always execute on the UI thread.
type AsyncEvent struct {
	Kind string // "callback", "lsp_response", "lsp_notification", "lsp_exit"
	Fn   func(api EditorAPI)
}

// EditorAPI is the safe facade through which plugins inspect and mutate Larry.
// All row/col indices on EditorAPI are 0-based; Lua bindings translate 1-based <-> 0-based.
type EditorAPI interface {
	// Buffer access & mutation (mutations record undo and invalidate caches)
	FileName() string
	FilePath() string
	FileType() string
	RootDir() string
	Lines() []string
	LineCount() int
	GetLine(row int) string
	SetLine(row int, text string)
	SetLines(lines []string)
	InsertText(row, col int, text string)
	DeleteRange(startRow, startCol, endRow, endCol int)
	IsModified() bool

	// Cursor & selection
	Cursor() (row, col int)
	SetCursor(row, col int)
	Selection() (active bool, startRow, startCol, endRow, endCol int)
	SelectedText() string

	// File operations
	OpenFile(path string, row, col int) error
	SaveFile() error

	// UI primitives
	SetStatus(msg string)
	ShowPicker(title string, items []PickerItem, onSelect func(PickerItem))
	ShowPanel(title string, lines []string)
	ClosePanel()
	ShowHover(title string, lines []string)
	CloseHover()
	ThemeName() string
	Palette() map[string]string
}
