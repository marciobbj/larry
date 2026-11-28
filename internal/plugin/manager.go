package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"larry/internal/config"

	lua "github.com/yuin/gopher-lua"
)

type eventHandler struct {
	id     int
	plugin string
	fn     *lua.LFunction
}

type registeredCommand struct {
	info CommandInfo
	fn   *lua.LFunction
}

type registeredKeymap struct {
	key    string
	desc   string
	plugin string
	cmdID  string
	fn     *lua.LFunction
}

// Manager owns the embedded Lua VM, discovered plugins, hooks, commands,
// keymaps, diagnostics, gutter signs, status badges, timers, and LSP clients.
type Manager struct {
	mu sync.RWMutex

	cfg        config.Config
	pluginsDir string

	L             *lua.LState
	activeAPI     EditorAPI
	currentPlugin string

	plugins  []PluginInfo
	builtIns []PluginInfo
	hooks    map[EventName][]eventHandler
	commands map[string]*registeredCommand
	keymaps  map[string]*registeredKeymap

	// absPath -> namespace -> []Diagnostic
	diagnostics map[string]map[string][]Diagnostic
	// absPath -> namespace -> row (0-based) -> GutterSign
	gutterSigns map[string]map[string]map[int]GutterSign
	// absPath -> namespace -> row (0-based) -> []TextHighlight
	highlights map[string]map[string]map[int][]TextHighlight
	// badge ID -> StatusBadge
	badges map[string]StatusBadge

	lspClients map[string]*LSPClient
	timers     map[int64]*time.Timer
	logs       []string

	nextHandlerID int
	nextTimerID   atomic.Int64

	asyncCh chan AsyncEvent
	closed  atomic.Bool
}

// NewManager creates an initialized plugin manager ready to load plugins from dir.
func NewManager(cfg config.Config, pluginsDir string) *Manager {
	m := &Manager{
		cfg:         cfg,
		pluginsDir:  pluginsDir,
		hooks:       make(map[EventName][]eventHandler),
		commands:    make(map[string]*registeredCommand),
		keymaps:     make(map[string]*registeredKeymap),
		diagnostics: make(map[string]map[string][]Diagnostic),
		gutterSigns: make(map[string]map[string]map[int]GutterSign),
		highlights:  make(map[string]map[string]map[int][]TextHighlight),
		badges:      make(map[string]StatusBadge),
		lspClients:  make(map[string]*LSPClient),
		timers:      make(map[int64]*time.Timer),
		asyncCh:     make(chan AsyncEvent, 256),
	}
	m.initVM()
	return m
}

func (m *Manager) initVM() {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: false,
	})
	m.L = L
	m.registerBindings(L)
}

// AsyncEvents returns the channel on which background work delivers UI-thread callbacks.
func (m *Manager) AsyncEvents() <-chan AsyncEvent {
	return m.asyncCh
}

// DispatchAsync enqueues an async callback for execution on the Bubble Tea Update loop.
func (m *Manager) DispatchAsync(evt AsyncEvent) {
	if m.closed.Load() {
		return
	}
	select {
	case m.asyncCh <- evt:
	default:
		// Drop if queue is saturated to avoid blocking background goroutines.
	}
}

// HandleAsyncEvent runs an AsyncEvent's callback with the active EditorAPI bound.
func (m *Manager) HandleAsyncEvent(api EditorAPI, evt AsyncEvent) {
	if evt.Fn == nil || m.closed.Load() {
		return
	}
	m.withAPI(api, func() {
		evt.Fn(api)
	})
}

// PluginsDir returns the fixed plugins directory watched by this manager.
func (m *Manager) PluginsDir() string {
	if m == nil {
		return ""
	}
	return m.pluginsDir
}

// StartupNotice returns a concise user-facing notification summarizing the
// plugin initialization outcome at startup or reload.
func (m *Manager) StartupNotice() string {
	if m == nil || !m.cfg.Plugins.IsEnabled() {
		return "Plugins disabled"
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	loaded := 0
	failed := 0
	for _, p := range m.plugins {
		if p.BuiltIn {
			continue
		}
		if p.Loaded {
			loaded++
		} else if p.Err != "" {
			failed++
		}
	}

	if loaded == 0 && failed == 0 {
		return "No plugins installed"
	}
	if failed > 0 {
		if loaded == 1 {
			return fmt.Sprintf("1 plugin initialized (%d failed)", failed)
		}
		return fmt.Sprintf("%d plugins initialized (%d failed)", loaded, failed)
	}
	if loaded == 1 {
		return "1 plugin initialized"
	}
	return fmt.Sprintf("%d plugins initialized", loaded)
}

// LoadAll discovers and executes all enabled plugins in the fixed plugins directory.
// Required directory structure:
//   - <pluginsDir>/<plugin-name>/init.lua
//
// Hidden directories (starting with "." or "_"), loose files, and folders
// without an init.lua entrypoint are ignored.
func (m *Manager) LoadAll() []PluginInfo {
	if !m.cfg.Plugins.IsEnabled() || m.pluginsDir == "" {
		return nil
	}

	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			m.addLog(fmt.Sprintf("[plugins] plugins directory not found (%s); starting without plugins", m.pluginsDir))
		} else {
			m.addLog(fmt.Sprintf("[plugins] failed to read %s: %v", m.pluginsDir, err))
		}
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}

		pluginDir := filepath.Join(m.pluginsDir, name)
		entryFile := filepath.Join(pluginDir, "init.lua")
		if info, statErr := os.Stat(entryFile); statErr != nil || info.IsDir() {
			continue
		}

		if m.cfg.Plugins.IsDisabled(name) {
			m.addLog(fmt.Sprintf("[plugins] skipped disabled plugin %q", name))
			continue
		}

		m.LoadFile(name, entryFile, pluginDir)
	}

	return m.Plugins()
}

// LoadFile loads a single Lua plugin file into the VM.
func (m *Manager) LoadFile(name, filePath, pluginDir string) PluginInfo {
	info := PluginInfo{
		Name: name,
		Path: filePath,
	}

	m.mu.Lock()
	m.currentPlugin = name
	idx := len(m.plugins)
	m.plugins = append(m.plugins, info)
	m.mu.Unlock()

	// Allow plugin-local require("foo") via package.path.
	if pluginDir != "" {
		m.addPackagePath(pluginDir)
	}

	err := m.L.DoFile(filePath)

	m.mu.Lock()
	m.currentPlugin = ""
	if err != nil {
		m.plugins[idx].Err = err.Error()
		m.plugins[idx].Loaded = false
		m.addLogLocked(fmt.Sprintf("[plugin:%s] load error: %v", name, err))
	} else {
		m.plugins[idx].Loaded = true
		m.addLogLocked(fmt.Sprintf("[plugin:%s] loaded (%s)", name, filePath))
	}
	res := m.plugins[idx]
	m.mu.Unlock()
	return res
}

// LoadSource loads an inline Lua snippet as a named plugin (useful for tests).
func (m *Manager) LoadSource(name, source string) PluginInfo {
	info := PluginInfo{
		Name: name,
		Path: "<inline>",
	}

	m.mu.Lock()
	m.currentPlugin = name
	idx := len(m.plugins)
	m.plugins = append(m.plugins, info)
	m.mu.Unlock()

	err := m.L.DoString(source)

	m.mu.Lock()
	m.currentPlugin = ""
	if err != nil {
		m.plugins[idx].Err = err.Error()
		m.plugins[idx].Loaded = false
		m.addLogLocked(fmt.Sprintf("[plugin:%s] error: %v", name, err))
	} else {
		m.plugins[idx].Loaded = true
	}
	res := m.plugins[idx]
	m.mu.Unlock()
	return res
}

func (m *Manager) addPackagePath(dir string) {
	pkg := m.L.GetGlobal("package")
	tbl, ok := pkg.(*lua.LTable)
	if !ok {
		return
	}
	cur := tbl.RawGetString("path").String()
	extra := filepath.Join(dir, "?.lua") + ";" + filepath.Join(dir, "?", "init.lua")
	if !strings.Contains(cur, extra) {
		tbl.RawSetString("path", lua.LString(extra+";"+cur))
	}
}

// RegisterBuiltIn registers a Go-native built-in plugin so it appears in the
// loaded plugins panel and respects plugins.disabled configuration.
func (m *Manager) RegisterBuiltIn(info PluginInfo) {
	if m == nil {
		return
	}
	info.BuiltIn = true
	info.Path = "<built-in>"
	info.Loaded = !m.cfg.Plugins.IsDisabled(info.Name)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.builtIns {
		if existing.Name == info.Name {
			m.builtIns[i] = info
			return
		}
	}
	m.builtIns = append(m.builtIns, info)
}

// Reload stops active LSP clients/timers, resets the Lua VM, and reloads all plugins.
func (m *Manager) Reload(api EditorAPI) []PluginInfo {
	m.stopBackgroundWork()

	m.mu.Lock()
	if m.L != nil {
		m.L.Close()
	}
	m.plugins = nil
	m.hooks = make(map[EventName][]eventHandler)
	m.commands = make(map[string]*registeredCommand)
	m.keymaps = make(map[string]*registeredKeymap)
	m.diagnostics = make(map[string]map[string][]Diagnostic)
	m.gutterSigns = make(map[string]map[string]map[int]GutterSign)
	m.highlights = make(map[string]map[string]map[int][]TextHighlight)
	m.badges = make(map[string]StatusBadge)
	m.lspClients = make(map[string]*LSPClient)
	m.timers = make(map[int64]*time.Timer)
	m.initVM()
	m.mu.Unlock()

	loaded := m.LoadAll()
	if api != nil {
		m.Emit(api, EventInit, nil)
		if api.FilePath() != "" {
			m.Emit(api, EventFileOpen, nil)
		}
	}
	return loaded
}

// Close shuts down all LSP servers, timers, and the Lua VM cleanly.
func (m *Manager) Close() {
	if !m.closed.CompareAndSwap(false, true) {
		return
	}
	m.stopBackgroundWork()
	m.mu.Lock()
	if m.L != nil {
		m.L.Close()
		m.L = nil
	}
	m.mu.Unlock()
}

func (m *Manager) stopBackgroundWork() {
	m.mu.Lock()
	clients := make([]*LSPClient, 0, len(m.lspClients))
	for _, c := range m.lspClients {
		clients = append(clients, c)
	}
	m.lspClients = make(map[string]*LSPClient)

	for id, t := range m.timers {
		t.Stop()
		delete(m.timers, id)
	}
	m.mu.Unlock()

	for _, c := range clients {
		c.Stop()
	}
}

// withAPI temporarily binds api as the active EditorAPI for the duration of fn.
func (m *Manager) withAPI(api EditorAPI, fn func()) {
	prev := m.activeAPI
	m.activeAPI = api
	defer func() {
		m.activeAPI = prev
	}()
	fn()
}

// Emit invokes all handlers registered for the given event.
func (m *Manager) Emit(api EditorAPI, event EventName, extra map[string]any) {
	if m == nil || m.closed.Load() || m.L == nil {
		return
	}
	m.mu.RLock()
	handlers := append([]eventHandler(nil), m.hooks[event]...)
	m.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	m.withAPI(api, func() {
		ctxTbl := m.buildEventContext(api, event, extra)
		for _, h := range handlers {
			if err := m.callLuaFn(h.fn, ctxTbl); err != nil {
				m.addLog(fmt.Sprintf("[plugin:%s] %s error: %v", h.plugin, event, err))
				if api != nil {
					api.SetStatus(fmt.Sprintf("Plugin %s error: %v", h.plugin, err))
				}
			}
		}
	})
}

func (m *Manager) buildEventContext(api EditorAPI, event EventName, extra map[string]any) *lua.LTable {
	tbl := m.L.NewTable()
	tbl.RawSetString("event", lua.LString(string(event)))
	if api != nil {
		tbl.RawSetString("file", lua.LString(api.FileName()))
		tbl.RawSetString("path", lua.LString(api.FilePath()))
		tbl.RawSetString("filetype", lua.LString(api.FileType()))
		tbl.RawSetString("root", lua.LString(api.RootDir()))
		row, col := api.Cursor()
		tbl.RawSetString("line", lua.LNumber(row+1))
		tbl.RawSetString("col", lua.LNumber(col+1))
	}
	for k, v := range extra {
		tbl.RawSetString(k, goToLua(m.L, v))
	}
	return tbl
}

// HandleKey dispatches a key string against plugin-registered keymaps.
// Returns true if a plugin keymap handled the key.
func (m *Manager) HandleKey(api EditorAPI, keyStr string) bool {
	if m == nil || m.closed.Load() || m.L == nil {
		return false
	}
	norm := m.normalizeKey(keyStr)

	m.mu.RLock()
	km, ok := m.keymaps[norm]
	m.mu.RUnlock()
	if !ok {
		return false
	}

	if km.cmdID != "" {
		_ = m.RunCommand(api, km.cmdID)
		return true
	}

	if km.fn != nil {
		m.withAPI(api, func() {
			if err := m.callLuaFn(km.fn); err != nil {
				m.addLog(fmt.Sprintf("[plugin:%s] keymap %q error: %v", km.plugin, km.key, err))
				if api != nil {
					api.SetStatus(fmt.Sprintf("Plugin %s error: %v", km.plugin, err))
				}
			}
		})
		return true
	}
	return false
}

// RunCommand executes a registered plugin command by ID.
func (m *Manager) RunCommand(api EditorAPI, id string) error {
	if m == nil || m.closed.Load() || m.L == nil {
		return fmt.Errorf("plugin manager unavailable")
	}
	m.mu.RLock()
	cmd, ok := m.commands[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown plugin command: %s", id)
	}

	var runErr error
	m.withAPI(api, func() {
		if err := m.callLuaFn(cmd.fn); err != nil {
			runErr = err
			m.addLog(fmt.Sprintf("[plugin:%s] command %q error: %v", cmd.info.Plugin, id, err))
			if api != nil {
				api.SetStatus(fmt.Sprintf("Command %s error: %v", id, err))
			}
		}
	})
	return runErr
}

func (m *Manager) callLuaFn(fn *lua.LFunction, args ...lua.LValue) error {
	if m.L == nil || fn == nil {
		return nil
	}
	return m.L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, args...)
}

func (m *Manager) normalizeKey(raw string) string {
	k := strings.ToLower(strings.TrimSpace(raw))
	leader := strings.ToLower(strings.TrimSpace(m.cfg.LeaderKey))
	if leader == "" {
		leader = "ctrl"
	}
	k = strings.ReplaceAll(k, "<leader>", leader)
	k = strings.ReplaceAll(k, "leader+", leader+"+")
	return k
}

// Commands returns all plugin-registered commands sorted by ID.
func (m *Manager) Commands() []CommandInfo {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]CommandInfo, 0, len(m.commands))
	for _, c := range m.commands {
		out = append(out, c.info)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// Keymaps returns all plugin-registered keymaps sorted by key.
func (m *Manager) Keymaps() []KeymapInfo {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]KeymapInfo, 0, len(m.keymaps))
	for _, km := range m.keymaps {
		out = append(out, KeymapInfo{
			Key:    km.key,
			Desc:   km.desc,
			Plugin: km.plugin,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return out
}

// Plugins returns metadata for all discovered plugins.
func (m *Manager) Plugins() []PluginInfo {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]PluginInfo, 0, len(m.builtIns)+len(m.plugins))
	out = append(out, m.builtIns...)
	out = append(out, m.plugins...)
	return out
}

// Logs returns the plugin log buffer.
func (m *Manager) Logs() []string {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.logs))
	copy(out, m.logs)
	return out
}

func (m *Manager) addLog(line string) {
	m.mu.Lock()
	m.addLogLocked(line)
	m.mu.Unlock()
}

func (m *Manager) addLogLocked(line string) {
	ts := time.Now().Format("15:04:05")
	m.logs = append(m.logs, ts+" "+line)
	if len(m.logs) > 200 {
		m.logs = m.logs[len(m.logs)-200:]
	}
}

// ── Diagnostics, Gutter Signs & Status Badges ──

func normalizePathKey(p string) string {
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(p)
}

func (m *Manager) SetDiagnostics(path, namespace string, diags []Diagnostic) {
	key := normalizePathKey(path)
	if key == "" {
		return
	}
	if namespace == "" {
		namespace = "default"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.diagnostics[key]; !ok {
		m.diagnostics[key] = make(map[string][]Diagnostic)
	}
	copied := make([]Diagnostic, len(diags))
	copy(copied, diags)
	m.diagnostics[key][namespace] = copied
}

func (m *Manager) ClearDiagnostics(path, namespace string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if path == "" {
		if namespace == "" {
			m.diagnostics = make(map[string]map[string][]Diagnostic)
			return
		}
		for _, byNS := range m.diagnostics {
			delete(byNS, namespace)
		}
		return
	}
	key := normalizePathKey(path)
	if namespace == "" {
		delete(m.diagnostics, key)
		return
	}
	if byNS, ok := m.diagnostics[key]; ok {
		delete(byNS, namespace)
	}
}

func (m *Manager) Diagnostics(path string) []Diagnostic {
	if m == nil || path == "" {
		return nil
	}
	key := normalizePathKey(path)
	m.mu.RLock()
	defer m.mu.RUnlock()
	byNS := m.diagnostics[key]
	if len(byNS) == 0 {
		return nil
	}
	var all []Diagnostic
	for _, list := range byNS {
		all = append(all, list...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Line != all[j].Line {
			return all[i].Line < all[j].Line
		}
		if all[i].Severity != all[j].Severity {
			return all[i].Severity < all[j].Severity
		}
		return all[i].Col < all[j].Col
	})
	return all
}

func (m *Manager) DiagnosticsByLine(path string) map[int][]Diagnostic {
	diags := m.Diagnostics(path)
	if len(diags) == 0 {
		return nil
	}
	byLine := make(map[int][]Diagnostic)
	for _, d := range diags {
		byLine[d.Line] = append(byLine[d.Line], d)
	}
	return byLine
}

func (m *Manager) DiagnosticCounts(path string) (errs, warns int) {
	for _, d := range m.Diagnostics(path) {
		switch d.Severity {
		case SeverityError:
			errs++
		case SeverityWarning:
			warns++
		}
	}
	return errs, warns
}

func (m *Manager) SetGutterSigns(path, namespace string, signs []GutterSign) {
	key := normalizePathKey(path)
	if key == "" {
		return
	}
	if namespace == "" {
		namespace = "default"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.gutterSigns[key]; !ok {
		m.gutterSigns[key] = make(map[string]map[int]GutterSign)
	}
	byLine := make(map[int]GutterSign, len(signs))
	for _, s := range signs {
		byLine[s.Line] = s
	}
	m.gutterSigns[key][namespace] = byLine
}

func (m *Manager) ClearGutterSigns(path, namespace string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if path == "" {
		if namespace == "" {
			m.gutterSigns = make(map[string]map[string]map[int]GutterSign)
			return
		}
		for _, byNS := range m.gutterSigns {
			delete(byNS, namespace)
		}
		return
	}
	key := normalizePathKey(path)
	if namespace == "" {
		delete(m.gutterSigns, key)
		return
	}
	if byNS, ok := m.gutterSigns[key]; ok {
		delete(byNS, namespace)
	}
}

// SetHighlights replaces inline text highlights for a file and namespace.
func (m *Manager) SetHighlights(path, namespace string, spans []TextHighlight) {
	key := normalizePathKey(path)
	if key == "" {
		return
	}
	if namespace == "" {
		namespace = "default"
	}
	byLine := make(map[int][]TextHighlight)
	for _, h := range spans {
		if h.Line < 0 || h.EndCol <= h.Col {
			continue
		}
		byLine[h.Line] = append(byLine[h.Line], h)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.highlights[key]; !ok {
		m.highlights[key] = make(map[string]map[int][]TextHighlight)
	}
	m.highlights[key][namespace] = byLine
}

// ClearHighlights clears inline text highlights for a file and/or namespace.
func (m *Manager) ClearHighlights(path, namespace string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if path == "" {
		if namespace == "" {
			m.highlights = make(map[string]map[string]map[int][]TextHighlight)
			return
		}
		for _, byNS := range m.highlights {
			delete(byNS, namespace)
		}
		return
	}
	key := normalizePathKey(path)
	if namespace == "" {
		delete(m.highlights, key)
		return
	}
	if byNS, ok := m.highlights[key]; ok {
		delete(byNS, namespace)
	}
}

// HighlightsByLine returns merged inline highlights for a file grouped by 0-based row,
// sorted by ascending priority so higher-priority spans win when overlapping.
func (m *Manager) HighlightsByLine(path string) map[int][]TextHighlight {
	if m == nil || path == "" {
		return nil
	}
	key := normalizePathKey(path)
	m.mu.RLock()
	defer m.mu.RUnlock()
	byNS := m.highlights[key]
	if len(byNS) == 0 {
		return nil
	}
	merged := make(map[int][]TextHighlight)
	for _, perLine := range byNS {
		for row, list := range perLine {
			merged[row] = append(merged[row], list...)
		}
	}
	if len(merged) == 0 {
		return nil
	}
	for row := range merged {
		list := merged[row]
		if len(list) > 1 {
			sort.Slice(list, func(i, j int) bool {
				if list[i].Priority != list[j].Priority {
					return list[i].Priority < list[j].Priority
				}
				return list[i].Col < list[j].Col
			})
		}
	}
	return merged
}

// EffectiveGutterSigns merges explicit gutter signs and diagnostic markers for a file.
// Higher priority wins (diagnostics default to priority 40..10 for Error..Hint).
func (m *Manager) EffectiveGutterSigns(path string) map[int]GutterSign {
	if m == nil || path == "" {
		return nil
	}
	key := normalizePathKey(path)
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[int]GutterSign)

	// 1. Add diagnostic indicators
	for _, list := range m.diagnostics[key] {
		for _, d := range list {
			sign := diagnosticGutterSign(d)
			if existing, ok := result[d.Line]; !ok || sign.Priority > existing.Priority {
				result[d.Line] = sign
			}
		}
	}

	// 2. Overlay explicit plugin gutter signs
	for _, byLine := range m.gutterSigns[key] {
		for line, sign := range byLine {
			if existing, ok := result[line]; !ok || sign.Priority >= existing.Priority {
				result[line] = sign
			}
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func diagnosticGutterSign(d Diagnostic) GutterSign {
	switch d.Severity {
	case SeverityError:
		return GutterSign{Line: d.Line, Text: "●", Color: "red", Priority: 40}
	case SeverityWarning:
		return GutterSign{Line: d.Line, Text: "▲", Color: "yellow", Priority: 30}
	case SeverityInfo:
		return GutterSign{Line: d.Line, Text: "ℹ", Color: "cyan", Priority: 20}
	default:
		return GutterSign{Line: d.Line, Text: "·", Color: "muted", Priority: 10}
	}
}

func (m *Manager) SetBadge(b StatusBadge) {
	if b.ID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if b.Text == "" {
		delete(m.badges, b.ID)
		return
	}
	m.badges[b.ID] = b
}

func (m *Manager) ClearBadge(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.badges, id)
}

func (m *Manager) Badges() []StatusBadge {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.badges) == 0 {
		return nil
	}
	out := make([]StatusBadge, 0, len(m.badges))
	for _, b := range m.badges {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// ── External command execution ──

func runCommand(command string, args []string, opts ExecOptions) ExecResult {
	timeout := 5 * time.Second
	if opts.TimeoutMs > 0 {
		timeout = time.Duration(opts.TimeoutMs) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	if opts.Dir != "" {
		if info, err := os.Stat(opts.Dir); err == nil && info.IsDir() {
			cmd.Dir = opts.Dir
		}
	}
	if opts.Stdin != "" {
		cmd.Stdin = strings.NewReader(opts.Stdin)
	}
	if len(opts.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res := ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err != nil {
		res.Err = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
		} else {
			res.ExitCode = -1
		}
	}
	return res
}
