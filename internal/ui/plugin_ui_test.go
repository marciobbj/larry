package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"larry/internal/config"
	"larry/internal/plugin"

	tea "github.com/charmbracelet/bubbletea"
)

// isolatePluginsDir points the OS config directory to a fresh temp folder so
// config.PluginsDir() resolves to a clean, isolated path on Linux, macOS, and Windows.
func isolatePluginsDir(t *testing.T) string {
	t.Helper()
	cfgRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	t.Setenv("HOME", cfgRoot)
	t.Setenv("APPDATA", cfgRoot)
	return config.PluginsDir()
}

func TestPluginStartupNotifications(t *testing.T) {
	// 1. Missing plugins folder -> starts cleanly and notifies user
	pluginsDir := isolatePluginsDir(t)
	m := InitialModel("main.go", []string{"package main"}, config.DefaultConfig())
	defer m.ClosePlugins()
	if m.statusMsg != "No plugins installed" {
		t.Fatalf("expected 'No plugins installed', got %q", m.statusMsg)
	}

	// 2. Two valid plugins in fixed <plugins-dir>/<name>/init.lua structure
	for _, name := range []string{"alpha", "beta"} {
		dir := filepath.Join(pluginsDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "init.lua"), []byte(`larry.log("loaded")`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m2 := InitialModel("main.go", []string{"package main"}, config.DefaultConfig())
	defer m2.ClosePlugins()
	if m2.statusMsg != "2 plugins initialized" {
		t.Fatalf("expected '2 plugins initialized', got %q", m2.statusMsg)
	}
}

func TestPluginAutoFormatOnSaveAndUndo(t *testing.T) {
	pluginsDir := isolatePluginsDir(t)
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginsDir, "trimmer"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Plugin that trims trailing spaces on before_save and sets a badge
	luaCode := `
		larry.plugin({
			name = "trimmer",
			version = "0.1.0",
			description = "Trims trailing whitespace before saving"
		})
		larry.on("before_save", function(ctx)
			local lines = larry.buf.lines()
			local changed = false
			for i, l in ipairs(lines) do
				local trimmed = l:gsub("%s+$", "")
				if trimmed ~= l then
					lines[i] = trimmed
					changed = true
				end
			end
			if changed then
				larry.buf.set_lines(lines)
			end
		end)
		larry.on("after_save", function(ctx)
			larry.ui.set_badge("trimmer", "FMT", "green", 50)
		end)
	`
	if err := os.WriteFile(filepath.Join(pluginsDir, "trimmer", "init.lua"), []byte(luaCode), 0o644); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(tmpDir, "sample.txt")
	cfg := config.DefaultConfig()

	m := InitialModel(targetFile, []string{"hello   ", "world\t "}, cfg)
	defer m.ClosePlugins()
	if m.statusMsg != "1 plugin initialized" {
		t.Fatalf("expected '1 plugin initialized', got %q", m.statusMsg)
	}
	m.Width = 100
	m.Height = 30

	// Trigger Ctrl+S (save)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(Model)

	if m.Lines[0] != "hello" || m.Lines[1] != "world" {
		t.Fatalf("expected trimmed lines in buffer, got %#v", m.Lines)
	}

	diskBytes, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(diskBytes) != "hello\nworld" {
		t.Fatalf("expected trimmed content on disk, got %q", string(diskBytes))
	}

	// Status bar should include the FMT badge set in after_save
	statusBar := m.renderStatusBar()
	if !strings.Contains(statusBar, "FMT") {
		t.Fatalf("expected status bar to render plugin badge FMT, got %q", statusBar)
	}

	// Ctrl+Z should undo the entire plugin format in a single step
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})
	m = updated.(Model)
	if m.Lines[0] != "hello   " || m.Lines[1] != "world\t " {
		t.Fatalf("expected undo to restore pre-format lines, got %#v", m.Lines)
	}
}

func TestPluginDiagnosticsRenderAndCommandPalette(t *testing.T) {
	pluginsDir := isolatePluginsDir(t)
	demoDir := filepath.Join(pluginsDir, "demo")
	if err := os.MkdirAll(demoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	luaCode := `
		larry.cmd.register("lint.run", {
			title = "Run Demo Linter",
			category = "Linter",
			key = "alt+l",
		}, function()
			larry.diagnostics.set(larry.buf.filepath(), "demo-lint", {
				{
					line = 2,
					col = 1,
					end_col = 5,
					severity = "error",
					source = "demo-lint",
					message = "undefined symbol foo",
				},
			})
			larry.ui.set_status("Lint complete: 1 error")
		end)

		larry.cmd.register("lsp.hover", {
			title = "Show Hover Docs",
			category = "LSP",
			key = "alt+k",
		}, function()
			larry.ui.show_hover("func main()", { "Entry point of the Go program." })
		end)
	`
	if err := os.WriteFile(filepath.Join(demoDir, "init.lua"), []byte(luaCode), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()

	m := InitialModel("main.go", []string{"package main", "foo()", "func main() {}"}, cfg)
	defer m.ClosePlugins()
	m.Width = 100
	m.Height = 30

	// Trigger plugin keymap alt+l
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}, Alt: true})
	m = updated.(Model)
	if m.statusMsg != "Lint complete: 1 error" {
		t.Fatalf("expected plugin keymap alt+l to run, status=%q", m.statusMsg)
	}

	// Move cursor to line 2 (index 1) so status bar shows diagnostic message
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "●") {
		t.Fatalf("expected diagnostic gutter/status symbol in view")
	}
	if !strings.Contains(view, "undefined symbol foo") {
		t.Fatalf("expected active-line diagnostic message in status bar")
	}

	// Trigger Hover popup with alt+k
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}, Alt: true})
	m = updated.(Model)
	if !m.pluginHover.Visible {
		t.Fatalf("expected hover popup to be visible")
	}
	hoverView := m.View()
	if !strings.Contains(hoverView, "Entry point of the Go program.") {
		t.Fatalf("expected hover body in View()")
	}
	// Esc closes hover
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.pluginHover.Visible {
		t.Fatalf("expected Esc to close hover popup")
	}

	// Open Command Palette via F1, filter for "Demo Linter", and verify entry exists
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyF1})
	m = updated.(Model)
	if !m.pickerActive || !m.picker.IsPalette {
		t.Fatalf("expected Command Palette to open on F1")
	}
	paletteView := m.View()
	if !strings.Contains(paletteView, "Run Demo Linter") {
		t.Fatalf("expected plugin command in Command Palette view")
	}
}

func TestPluginAsyncEventFlowInModel(t *testing.T) {
	cfg := config.DefaultConfig()
	disabled := false
	cfg.Plugins.Enabled = &disabled

	m := InitialModel("test.go", []string{"package main"}, cfg)
	mgr := plugin.NewManager(config.DefaultConfig(), "")
	defer mgr.Close()

	info := mgr.LoadSource("async-demo", `
		larry.cmd.register("async.ping", function()
			larry.exec.async("sh", { "-c", "printf pong" }, function(res)
				larry.ui.set_status("got:" .. res.stdout)
			end)
		end)
	`)
	if !info.Loaded {
		t.Fatalf("load failed: %s", info.Err)
	}
	m.SetPluginManager(mgr)

	if err := mgr.RunCommand(m.editorAPI(), "async.ping"); err != nil {
		t.Fatal(err)
	}

	// Wait for Bubble Tea's async listener command to yield pluginAsyncEventMsg
	cmd := waitPluginAsyncCmd(mgr)
	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	select {
	case msg := <-done:
		updated, _ := m.Update(msg)
		m = updated.(Model)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for pluginAsyncEventMsg")
	}

	if m.statusMsg != "got:pong" {
		t.Fatalf("expected status 'got:pong', got %q", m.statusMsg)
	}
}

func TestGoFormatterAndAsyncLinterPluginEndToEnd(t *testing.T) {
	pluginsRoot := isolatePluginsDir(t)
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(pluginsRoot, "go-dev")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	luaPlugin := `
		larry.on("before_save", function(ctx)
			if ctx.filetype ~= "go" or not larry.exec.has("gofmt") then return end
			local src = larry.buf.text()
			local formatted, _, err = larry.exec.sync("gofmt", {}, { stdin = src, timeout_ms = 3000 })
			if err == nil and formatted ~= "" and formatted ~= src then
				if not src:match("\n$") and formatted:match("\n$") then
					formatted = formatted:sub(1, -2)
				end
				larry.buf.set_text(formatted)
			end
		end)

		larry.on("after_save", function(ctx)
			if ctx.filetype ~= "go" or not larry.exec.has("go") then return end
			local path = ctx.path
			local base = larry.fs.basename(path)
			larry.exec.async("go", { "vet", "." }, { dir = larry.fs.dirname(path), timeout_ms = 10000 }, function(res)
				local diags = {}
				for line in ((res.stderr or "") .. "\n" .. (res.stdout or "")):gmatch("[^\r\n]+") do
					local file, lnum, col, msg = line:match("^([^:]+):(%d+):(%d+):%s*(.+)$")
					if file and lnum and msg and larry.fs.basename(file) == base then
						table.insert(diags, {
							line = tonumber(lnum) or 1,
							col = tonumber(col) or 1,
							severity = "error",
							source = "go vet",
							message = msg,
						})
					end
				end
				larry.diagnostics.set(path, "go-vet", diags)
			end)
		end)
	`
	if err := os.WriteFile(filepath.Join(pluginsDir, "init.lua"), []byte(luaPlugin), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a tiny Go module with unformatted code AND a real `go vet` error
	projDir := filepath.Join(tmpDir, "proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unformatted := []string{
		"package main",
		`import "fmt"`,
		"func main(){",
		`fmt.Printf("%d", "not-an-int")`,
		"}",
	}
	goFile := filepath.Join(projDir, "main.go")
	if err := os.WriteFile(goFile, []byte(strings.Join(unformatted, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()

	m := InitialModel(goFile, unformatted, cfg)
	defer m.ClosePlugins()
	m.Width = 140
	m.Height = 30

	// 1. Save with Ctrl+S -> before_save runs gofmt, after_save triggers async go vet
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = updated.(Model)

	// Verify gofmt formatted the buffer (indented with tab & space before `{`)
	joined := strings.Join(m.Lines, "\n")
	if !strings.Contains(joined, "func main() {\n\tfmt.Printf") {
		t.Fatalf("expected go-dev plugin to format buffer with gofmt, got:\n%s", joined)
	}

	// 2. Wait for async `go vet` diagnostic event to arrive
	deadline := time.After(10 * time.Second)
	for len(m.Plugins().Diagnostics(goFile)) == 0 {
		select {
		case evt := <-m.Plugins().AsyncEvents():
			updated, _ = m.Update(pluginAsyncEventMsg{evt: evt})
			m = updated.(Model)
		case <-deadline:
			t.Fatalf("timed out waiting for go vet diagnostics; logs=%v", m.Plugins().Logs())
		}
	}

	diags := m.Plugins().Diagnostics(goFile)
	if len(diags) == 0 {
		t.Fatalf("expected at least 1 go vet diagnostic")
	}
	if diags[0].Source != "go vet" || !strings.Contains(diags[0].Message, "Printf") {
		t.Fatalf("unexpected go vet diagnostic: %+v", diags[0])
	}
}
