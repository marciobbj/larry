package plugin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"larry/internal/config"
)

// mockEditorAPI implements EditorAPI for isolated plugin engine tests.
type mockEditorAPI struct {
	fileName string
	filePath string
	fileType string
	rootDir  string
	lines    []string
	row      int
	col      int
	modified bool
	status   string

	selecting        bool
	selStartRow      int
	selStartCol      int
	openedPath       string
	openedRow        int
	openedCol        int
	saveCount        int
	pickerTitle      string
	pickerItems      []PickerItem
	pickerOnSelect   func(PickerItem)
	panelTitle       string
	panelLines       []string
	panelVisible     bool
	hoverTitle       string
	hoverLines       []string
	hoverVisible     bool
}

func newMockAPI() *mockEditorAPI {
	return &mockEditorAPI{
		fileName: "main.go",
		filePath: "/tmp/project/main.go",
		fileType: "go",
		rootDir:  "/tmp/project",
		lines:    []string{"package main", "", "func main() {}"},
	}
}

func (a *mockEditorAPI) FileName() string  { return a.fileName }
func (a *mockEditorAPI) FilePath() string  { return a.filePath }
func (a *mockEditorAPI) FileType() string  { return a.fileType }
func (a *mockEditorAPI) RootDir() string   { return a.rootDir }
func (a *mockEditorAPI) Lines() []string   { return append([]string(nil), a.lines...) }
func (a *mockEditorAPI) LineCount() int    { return len(a.lines) }
func (a *mockEditorAPI) IsModified() bool  { return a.modified }
func (a *mockEditorAPI) Cursor() (int, int) { return a.row, a.col }
func (a *mockEditorAPI) ThemeName() string { return "dracula" }
func (a *mockEditorAPI) Palette() map[string]string {
	return map[string]string{"red": "#ff5555", "accent": "#bd93f9"}
}

func (a *mockEditorAPI) GetLine(row int) string {
	if row < 0 || row >= len(a.lines) {
		return ""
	}
	return a.lines[row]
}

func (a *mockEditorAPI) SetLine(row int, text string) {
	if row < 0 || row >= len(a.lines) {
		return
	}
	a.lines[row] = text
	a.modified = true
}

func (a *mockEditorAPI) SetLines(lines []string) {
	a.lines = append([]string(nil), lines...)
	a.modified = true
}

func (a *mockEditorAPI) InsertText(row, col int, text string) {
	if row < 0 || row >= len(a.lines) {
		return
	}
	r := []rune(a.lines[row])
	if col < 0 {
		col = 0
	}
	if col > len(r) {
		col = len(r)
	}
	a.lines[row] = string(r[:col]) + text + string(r[col:])
	a.modified = true
}

func (a *mockEditorAPI) DeleteRange(sr, sc, er, ec int) {
	if sr == er && sr >= 0 && sr < len(a.lines) {
		r := []rune(a.lines[sr])
		if sc < 0 {
			sc = 0
		}
		if ec > len(r) {
			ec = len(r)
		}
		if sc <= ec {
			a.lines[sr] = string(r[:sc]) + string(r[ec:])
			a.modified = true
		}
	}
}

func (a *mockEditorAPI) SetCursor(row, col int) {
	a.row = row
	a.col = col
}

func (a *mockEditorAPI) Selection() (bool, int, int, int, int) {
	return a.selecting, a.selStartRow, a.selStartCol, a.row, a.col
}

func (a *mockEditorAPI) SelectedText() string {
	if !a.selecting {
		return ""
	}
	return "selected"
}

func (a *mockEditorAPI) OpenFile(path string, row, col int) error {
	a.openedPath = path
	a.openedRow = row
	a.openedCol = col
	return nil
}

func (a *mockEditorAPI) SaveFile() error {
	a.saveCount++
	a.modified = false
	return nil
}

func (a *mockEditorAPI) SetStatus(msg string) { a.status = msg }
func (a *mockEditorAPI) ShowPicker(title string, items []PickerItem, onSelect func(PickerItem)) {
	a.pickerTitle = title
	a.pickerItems = items
	a.pickerOnSelect = onSelect
}
func (a *mockEditorAPI) ShowPanel(title string, lines []string) {
	a.panelTitle = title
	a.panelLines = lines
	a.panelVisible = true
}
func (a *mockEditorAPI) ClosePanel() { a.panelVisible = false }
func (a *mockEditorAPI) ShowHover(title string, lines []string) {
	a.hoverTitle = title
	a.hoverLines = lines
	a.hoverVisible = true
}
func (a *mockEditorAPI) CloseHover() { a.hoverVisible = false }
func (a *mockEditorAPI) ShowCompletions(items []CompletionItem, startRow, startCol int, onSelect func(CompletionItem)) {
}
func (a *mockEditorAPI) CloseCompletions() {}

func TestDiscoveryAndConfig(t *testing.T) {
	dir := t.TempDir()
	// Folder plugin: alpha/init.lua
	alphaDir := filepath.Join(dir, "alpha")
	if err := os.MkdirAll(alphaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := os.WriteFile(filepath.Join(alphaDir, "init.lua"), []byte(`
		larry.plugin({
			name = "alpha",
			version = "1.2.0",
			description = "Test alpha plugin",
			author = "larry"
		})
		local cfg = larry.config("alpha")
		larry.on("init", function()
			larry.ui.set_status("alpha:" .. tostring(cfg.greeting))
		end)
	`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// Disabled folder plugin: beta/init.lua
	betaDir := filepath.Join(dir, "beta")
	if err := os.MkdirAll(betaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(betaDir, "init.lua"), []byte(`larry.ui.set_status("should not load")`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Loose file and folder without init.lua must be ignored under fixed structure
	if err := os.WriteFile(filepath.Join(dir, "loose.lua"), []byte(`larry.ui.set_status("loose ignored")`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "empty-folder"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Plugins.Disabled = []string{"beta"}
	cfg.Plugins.Settings = map[string]map[string]any{
		"alpha": {"greeting": "ola"},
	}

	mgr := NewManager(cfg, dir)
	defer mgr.Close()

	loaded := mgr.LoadAll()
	if len(loaded) != 1 {
		t.Fatalf("expected 1 loaded plugin, got %d: %+v", len(loaded), loaded)
	}
	if !loaded[0].Loaded || loaded[0].Version != "1.2.0" || loaded[0].Description != "Test alpha plugin" {
		t.Fatalf("unexpected plugin metadata: %+v", loaded[0])
	}
	if got := mgr.StartupNotice(); got != "1 plugin initialized" {
		t.Fatalf("expected startup notice '1 plugin initialized', got %q", got)
	}

	api := newMockAPI()
	mgr.Emit(api, EventInit, nil)
	if api.status != "alpha:ola" {
		t.Fatalf("expected status 'alpha:ola', got %q", api.status)
	}
}

func TestStartupNotices(t *testing.T) {
	// Missing plugins directory -> starts cleanly with notice
	missingDir := filepath.Join(t.TempDir(), "does-not-exist")
	mgrMissing := NewManager(config.DefaultConfig(), missingDir)
	defer mgrMissing.Close()
	mgrMissing.LoadAll()
	if got := mgrMissing.StartupNotice(); got != "No plugins installed" {
		t.Fatalf("missing dir notice = %q, want %q", got, "No plugins installed")
	}

	// Multiple plugins with 1 error
	dir := t.TempDir()
	for _, name := range []string{"p1", "p2"} {
		pDir := filepath.Join(dir, name)
		if err := os.MkdirAll(pDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pDir, "init.lua"), []byte(`larry.log("ok")`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mgrMulti := NewManager(config.DefaultConfig(), dir)
	defer mgrMulti.Close()
	mgrMulti.LoadAll()
	if got := mgrMulti.StartupNotice(); got != "2 plugins initialized" {
		t.Fatalf("multi notice = %q, want %q", got, "2 plugins initialized")
	}

	brokenDir := filepath.Join(dir, "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "init.lua"), []byte(`this is not valid lua !!!`), 0o644); err != nil {
		t.Fatal(err)
	}
	mgrBroken := NewManager(config.DefaultConfig(), dir)
	defer mgrBroken.Close()
	mgrBroken.LoadAll()
	if got := mgrBroken.StartupNotice(); got != "2 plugins initialized (1 failed)" {
		t.Fatalf("broken notice = %q, want %q", got, "2 plugins initialized (1 failed)")
	}
}

func TestBufferHooksCommandsAndKeymaps(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LeaderKey = "ctrl"
	mgr := NewManager(cfg, "")
	defer mgr.Close()

	info := mgr.LoadSource("demo", `
		larry.on("before_save", function(ctx)
			if ctx.filetype == "go" then
				larry.buf.set_line(1, "// formatted")
			end
		end)

		larry.cmd.register("demo.shout", {
			title = "Shout Line",
			category = "Demo",
			key = "alt+s",
		}, function()
			local row, _ = larry.cursor.get()
			local txt = larry.buf.line(row)
			larry.buf.set_line(row, string.upper(txt))
			larry.ui.set_status("shouted!")
		end)

		larry.keymap.set("<leader>+y", function()
			larry.cursor.set(3, 5)
		end, "Jump to 3:5")
	`)
	if !info.Loaded {
		t.Fatalf("plugin failed to load: %s", info.Err)
	}

	api := newMockAPI()
	mgr.Emit(api, EventBeforeSave, nil)
	if api.lines[0] != "// formatted" {
		t.Fatalf("expected before_save to edit line 1, got %q", api.lines[0])
	}

	// Trigger command via its registered keymap
	if !mgr.HandleKey(api, "alt+s") {
		t.Fatalf("expected alt+s to be handled by plugin")
	}
	if api.lines[0] != "// FORMATTED" || api.status != "shouted!" {
		t.Fatalf("unexpected result after alt+s: line=%q status=%q", api.lines[0], api.status)
	}

	// Trigger <leader>+y -> ctrl+y
	if !mgr.HandleKey(api, "ctrl+y") {
		t.Fatalf("expected ctrl+y to be handled")
	}
	if api.row != 2 || api.col != 4 {
		t.Fatalf("expected cursor (2,4), got (%d,%d)", api.row, api.col)
	}
}

func TestDiagnosticsGutterAndBadges(t *testing.T) {
	mgr := NewManager(config.DefaultConfig(), "")
	defer mgr.Close()

	info := mgr.LoadSource("linter", `
		larry.on("after_save", function(ctx)
			larry.diagnostics.set(ctx.path, "govet", {
				{ line = 2, col = 1, end_col = 8, severity = "error", source = "go vet", message = "unreachable code" },
				{ line = 3, col = 1, severity = "warning", source = "go vet", message = "unused var" },
			})
			larry.ui.set_badge("linter", "lint: 1E 1W", "yellow", 20)
		end)
	`)
	if !info.Loaded {
		t.Fatalf("load failed: %s", info.Err)
	}

	api := newMockAPI()
	mgr.Emit(api, EventAfterSave, nil)

	errs, warns := mgr.DiagnosticCounts(api.FilePath())
	if errs != 1 || warns != 1 {
		t.Fatalf("expected 1 error 1 warning, got %dE %dW", errs, warns)
	}

	signs := mgr.EffectiveGutterSigns(api.FilePath())
	if signs[1].Text != "●" || signs[1].Color != "red" {
		t.Fatalf("expected red error gutter sign on row 1, got %+v", signs[1])
	}
	if signs[2].Text != "▲" || signs[2].Color != "yellow" {
		t.Fatalf("expected yellow warning gutter sign on row 2, got %+v", signs[2])
	}

	badges := mgr.Badges()
	if len(badges) != 1 || badges[0].Text != "lint: 1E 1W" {
		t.Fatalf("unexpected badges: %+v", badges)
	}
}

func TestInlineHighlights(t *testing.T) {
	mgr := NewManager(config.DefaultConfig(), "")
	defer mgr.Close()

	info := mgr.LoadSource("colorizer", `
		larry.cmd.register("hl.apply", function()
			larry.highlights.set(larry.buf.filepath(), "colorizer", {
				{ line = 1, col = 1, end_col = 8, fg = "#ffffff", bg = "#ff5555", bold = true },
			})
		end)
		larry.cmd.register("hl.clear", function()
			larry.highlights.clear(larry.buf.filepath(), "colorizer")
		end)
	`)
	if !info.Loaded {
		t.Fatalf("load failed: %s", info.Err)
	}

	api := newMockAPI()
	if err := mgr.RunCommand(api, "hl.apply"); err != nil {
		t.Fatal(err)
	}
	hls := mgr.HighlightsByLine(api.FilePath())
	if len(hls[0]) != 1 {
		t.Fatalf("expected 1 highlight on row 0, got %+v", hls)
	}
	h := hls[0][0]
	if h.Col != 0 || h.EndCol != 7 || h.BG != "#ff5555" || h.FG != "#ffffff" || !h.Bold {
		t.Fatalf("unexpected highlight: %+v", h)
	}

	if err := mgr.RunCommand(api, "hl.clear"); err != nil {
		t.Fatal(err)
	}
	if len(mgr.HighlightsByLine(api.FilePath())) != 0 {
		t.Fatalf("expected highlights cleared")
	}
}

func TestExecSyncAsyncAndJSON(t *testing.T) {
	mgr := NewManager(config.DefaultConfig(), "")
	defer mgr.Close()

	info := mgr.LoadSource("exec-test", `
		larry.cmd.register("test.sync", function()
			local out, _, err = larry.exec.sync("sh", { "-c", "printf hello" })
			local enc = larry.json.encode({ msg = out, ok = (err == nil) })
			local dec = larry.json.decode(enc)
			larry.ui.set_status(dec.msg .. ":" .. tostring(dec.ok))
		end)

		larry.cmd.register("test.async", function()
			larry.exec.async("sh", { "-c", "printf async_ok" }, function(res)
				larry.ui.set_status(res.stdout .. ":" .. tostring(res.exit_code))
			end)
		end)
	`)
	if !info.Loaded {
		t.Fatalf("load failed: %s", info.Err)
	}

	api := newMockAPI()
	if err := mgr.RunCommand(api, "test.sync"); err != nil {
		t.Fatalf("sync command failed: %v", err)
	}
	if api.status != "hello:true" {
		t.Fatalf("unexpected sync status: %q", api.status)
	}

	if err := mgr.RunCommand(api, "test.async"); err != nil {
		t.Fatalf("async command failed: %v", err)
	}

	select {
	case evt := <-mgr.AsyncEvents():
		mgr.HandleAsyncEvent(api, evt)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for async exec event")
	}

	if api.status != "async_ok:0" {
		t.Fatalf("unexpected async status: %q", api.status)
	}
}

// TestHelperLSPServer acts as a minimal stdio JSON-RPC 2.0 LSP server when
// re-invoked by TestLSPClientRoundTrip via GO_WANT_LSP_HELPER=1.
func TestHelperLSPServer(t *testing.T) {
	if os.Getenv("GO_WANT_LSP_HELPER") != "1" {
		return
	}
	reader := bufio.NewReader(os.Stdin)
	writeMsg := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n%s", len(b), b)
	}

	for {
		n, err := readLSPHeaders(reader)
		if err != nil {
			os.Exit(0)
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(reader, body); err != nil {
			os.Exit(0)
		}
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}
		method, _ := req["method"].(string)
		switch method {
		case "initialize":
			writeMsg(map[string]any{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]any{
					"capabilities": map[string]any{"hoverProvider": true},
				},
			})
		case "textDocument/didOpen":
			params, _ := req["params"].(map[string]any)
			td, _ := params["textDocument"].(map[string]any)
			uri, _ := td["uri"].(string)
			writeMsg(map[string]any{
				"jsonrpc": "2.0",
				"method":  "textDocument/publishDiagnostics",
				"params": map[string]any{
					"uri": uri,
					"diagnostics": []map[string]any{
						{
							"range": map[string]any{
								"start": map[string]any{"line": 1, "character": 0},
								"end":   map[string]any{"line": 1, "character": 5},
							},
							"severity": 1,
							"source":   "mock-lsp",
							"message":  "mock syntax error",
						},
					},
				},
			})
		case "exit":
			os.Exit(0)
		}
	}
}

func TestLSPClientRoundTrip(t *testing.T) {
	mgr := NewManager(config.DefaultConfig(), "")
	defer mgr.Close()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	luaSrc := fmt.Sprintf(`
		local client
		larry.cmd.register("lsp.start_test", function()
			local c, err = larry.lsp.start({
				name = "mock-lsp",
				cmd = %q,
				args = { "-test.run=TestHelperLSPServer" },
				env = { GO_WANT_LSP_HELPER = "1" },
			})
			assert(err == nil, tostring(err))
			client = c

			client:on_notification("textDocument/publishDiagnostics", function(params)
				local path = larry.lsp.uri_to_path(params.uri)
				local items = {}
				for _, d in ipairs(params.diagnostics or {}) do
					table.insert(items, {
						line = d.range.start.line + 1,
						col = d.range.start.character + 1,
						end_col = d.range["end"].character + 1,
						severity = d.severity,
						source = d.source,
						message = d.message,
					})
				end
				larry.diagnostics.set(path, "mock-lsp", items)
				larry.ui.set_status("diags:" .. tostring(#items))
			end)

			client:request("initialize", { rootUri = larry.lsp.path_to_uri(larry.workspace.root()) }, function(res, rerr)
				assert(rerr == nil, tostring(rerr))
				if res and res.capabilities and res.capabilities.hoverProvider then
					client:notify("textDocument/didOpen", {
						textDocument = {
							uri = larry.lsp.path_to_uri(larry.buf.filepath()),
							languageId = "go",
							version = 1,
							text = larry.buf.text(),
						}
					})
				end
			end)
		end)
	`, exe)

	info := mgr.LoadSource("lsp-test", luaSrc)
	if !info.Loaded {
		t.Fatalf("load failed: %s", info.Err)
	}

	api := newMockAPI()
	if err := mgr.RunCommand(api, "lsp.start_test"); err != nil {
		t.Fatalf("start_test failed: %v", err)
	}

	deadline := time.After(5 * time.Second)
	for api.status != "diags:1" {
		select {
		case evt := <-mgr.AsyncEvents():
			mgr.HandleAsyncEvent(api, evt)
		case <-deadline:
			t.Fatalf("timed out waiting for LSP diagnostics, status=%q, logs=%v", api.status, mgr.Logs())
		}
	}

	diags := mgr.Diagnostics(api.FilePath())
	if len(diags) != 1 || diags[0].Message != "mock syntax error" || diags[0].Line != 1 {
		t.Fatalf("unexpected LSP diagnostics: %+v", diags)
	}
	if !strings.Contains(PathToURI("/tmp/project/main.go"), "file:///") {
		t.Fatalf("unexpected URI conversion")
	}
}
