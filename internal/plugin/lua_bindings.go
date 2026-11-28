package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func (m *Manager) registerBindings(L *lua.LState) {
	larryTbl := L.NewTable()

	// larry.plugin({ name, version, description, author })
	L.SetField(larryTbl, "plugin", L.NewFunction(func(L *lua.LState) int {
		meta := L.CheckTable(1)
		m.mu.Lock()
		defer m.mu.Unlock()
		if len(m.plugins) > 0 && m.currentPlugin != "" {
			p := &m.plugins[len(m.plugins)-1]
			if v := meta.RawGetString("name"); v != lua.LNil {
				p.Name = v.String()
			}
			if v := meta.RawGetString("version"); v != lua.LNil {
				p.Version = v.String()
			}
			if v := meta.RawGetString("description"); v != lua.LNil {
				p.Description = v.String()
			}
			if v := meta.RawGetString("author"); v != lua.LNil {
				p.Author = v.String()
			}
		}
		return 0
	}))

	// larry.config(pluginName) -> table
	L.SetField(larryTbl, "config", L.NewFunction(func(L *lua.LState) int {
		name := L.OptString(1, m.currentPlugin)
		if m.cfg.Plugins.Settings != nil {
			if settings, ok := m.cfg.Plugins.Settings[name]; ok {
				L.Push(goToLua(L, settings))
				return 1
			}
		}
		L.Push(L.NewTable())
		return 1
	}))

	// larry.log(msg)
	L.SetField(larryTbl, "log", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		prefix := m.currentPlugin
		if prefix == "" {
			prefix = "lua"
		}
		m.addLog(fmt.Sprintf("[%s] %s", prefix, msg))
		return 0
	}))

	// larry.on(event, fn) -> handlerId
	L.SetField(larryTbl, "on", L.NewFunction(func(L *lua.LState) int {
		ev := EventName(strings.ToLower(strings.TrimSpace(L.CheckString(1))))
		fn := L.CheckFunction(2)

		m.mu.Lock()
		m.nextHandlerID++
		id := m.nextHandlerID
		m.hooks[ev] = append(m.hooks[ev], eventHandler{
			id:     id,
			plugin: m.currentPlugin,
			fn:     fn,
		})
		m.mu.Unlock()

		L.Push(lua.LNumber(id))
		return 1
	}))

	m.registerBufModule(L, larryTbl)
	m.registerCursorModule(L, larryTbl)
	m.registerWorkspaceModule(L, larryTbl)
	m.registerUIModule(L, larryTbl)
	m.registerDiagnosticsModule(L, larryTbl)
	m.registerGutterModule(L, larryTbl)
	m.registerHighlightsModule(L, larryTbl)
	m.registerCmdModule(L, larryTbl)
	m.registerKeymapModule(L, larryTbl)
	m.registerExecModule(L, larryTbl)
	m.registerLSPModule(L, larryTbl)
	m.registerTimerModule(L, larryTbl)
	m.registerFSModule(L, larryTbl)
	registerJSONModule(L, larryTbl)

	L.SetGlobal("larry", larryTbl)
}

func (m *Manager) registerBufModule(L *lua.LState, larryTbl *lua.LTable) {
	buf := L.NewTable()

	L.SetField(buf, "text", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(strings.Join(m.activeAPI.Lines(), "\n")))
		return 1
	}))

	L.SetField(buf, "set_text", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		text := strings.ReplaceAll(L.CheckString(1), "\r\n", "\n")
		lines := strings.Split(text, "\n")
		if len(lines) == 0 {
			lines = []string{""}
		}
		m.activeAPI.SetLines(lines)
		return 0
	}))

	L.SetField(buf, "lines", L.NewFunction(func(L *lua.LState) int {
		tbl := L.NewTable()
		if m.activeAPI != nil {
			for i, line := range m.activeAPI.Lines() {
				tbl.RawSetInt(i+1, lua.LString(line))
			}
		}
		L.Push(tbl)
		return 1
	}))

	L.SetField(buf, "set_lines", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		tbl := L.CheckTable(1)
		n := tbl.Len()
		lines := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			lines = append(lines, tbl.RawGetInt(i).String())
		}
		if len(lines) == 0 {
			lines = []string{""}
		}
		m.activeAPI.SetLines(lines)
		return 0
	}))

	L.SetField(buf, "line", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(""))
			return 1
		}
		row := L.CheckInt(1) - 1
		L.Push(lua.LString(m.activeAPI.GetLine(row)))
		return 1
	}))

	L.SetField(buf, "set_line", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		row := L.CheckInt(1) - 1
		text := L.CheckString(2)
		m.activeAPI.SetLine(row, text)
		return 0
	}))

	L.SetField(buf, "line_count", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LNumber(0))
			return 1
		}
		L.Push(lua.LNumber(m.activeAPI.LineCount()))
		return 1
	}))

	L.SetField(buf, "insert", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		row := L.CheckInt(1) - 1
		col := L.CheckInt(2) - 1
		text := L.CheckString(3)
		m.activeAPI.InsertText(row, col, text)
		return 0
	}))

	L.SetField(buf, "delete_range", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		sr := L.CheckInt(1) - 1
		sc := L.CheckInt(2) - 1
		er := L.CheckInt(3) - 1
		ec := L.CheckInt(4) - 1
		m.activeAPI.DeleteRange(sr, sc, er, ec)
		return 0
	}))

	L.SetField(buf, "filename", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(m.activeAPI.FileName()))
		return 1
	}))

	L.SetField(buf, "filepath", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(m.activeAPI.FilePath()))
		return 1
	}))

	L.SetField(buf, "filetype", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(m.activeAPI.FileType()))
		return 1
	}))

	L.SetField(buf, "modified", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LFalse)
			return 1
		}
		L.Push(lua.LBool(m.activeAPI.IsModified()))
		return 1
	}))

	L.SetField(buf, "save", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString("no active editor"))
			return 1
		}
		if err := m.activeAPI.SaveFile(); err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))

	L.SetField(larryTbl, "buf", buf)
}

func (m *Manager) registerCursorModule(L *lua.LState, larryTbl *lua.LTable) {
	cur := L.NewTable()

	L.SetField(cur, "get", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LNumber(1))
			L.Push(lua.LNumber(1))
			return 2
		}
		row, col := m.activeAPI.Cursor()
		L.Push(lua.LNumber(row + 1))
		L.Push(lua.LNumber(col + 1))
		return 2
	}))

	L.SetField(cur, "set", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		row := L.CheckInt(1) - 1
		col := L.OptInt(2, 1) - 1
		m.activeAPI.SetCursor(row, col)
		return 0
	}))

	L.SetField(cur, "selection", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LFalse)
			return 1
		}
		active, sr, sc, er, ec := m.activeAPI.Selection()
		L.Push(lua.LBool(active))
		L.Push(lua.LNumber(sr + 1))
		L.Push(lua.LNumber(sc + 1))
		L.Push(lua.LNumber(er + 1))
		L.Push(lua.LNumber(ec + 1))
		return 5
	}))

	L.SetField(cur, "selected_text", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(m.activeAPI.SelectedText()))
		return 1
	}))

	L.SetField(larryTbl, "cursor", cur)
}

func (m *Manager) registerWorkspaceModule(L *lua.LState, larryTbl *lua.LTable) {
	ws := L.NewTable()

	L.SetField(ws, "root", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			wd, _ := os.Getwd()
			L.Push(lua.LString(wd))
			return 1
		}
		L.Push(lua.LString(m.activeAPI.RootDir()))
		return 1
	}))

	L.SetField(ws, "open", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString("no active editor"))
			return 1
		}
		path := L.CheckString(1)
		row := L.OptInt(2, 1) - 1
		col := L.OptInt(3, 1) - 1
		if err := m.activeAPI.OpenFile(path, row, col); err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))

	L.SetField(larryTbl, "workspace", ws)
}

func (m *Manager) registerUIModule(L *lua.LState, larryTbl *lua.LTable) {
	uiTbl := L.NewTable()

	L.SetField(uiTbl, "set_status", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI != nil {
			m.activeAPI.SetStatus(L.CheckString(1))
		}
		return 0
	}))

	// larry.ui.set_badge(id, text, color?, priority?)
	L.SetField(uiTbl, "set_badge", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		text := L.CheckString(2)
		color := L.OptString(3, "cyan")
		priority := L.OptInt(4, 10)
		m.SetBadge(StatusBadge{
			ID:       id,
			Text:     text,
			Color:    color,
			Priority: priority,
		})
		return 0
	}))

	L.SetField(uiTbl, "clear_badge", L.NewFunction(func(L *lua.LState) int {
		m.ClearBadge(L.CheckString(1))
		return 0
	}))

	// larry.ui.show_picker(title, items, on_select?)
	L.SetField(uiTbl, "show_picker", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		title := L.CheckString(1)
		itemsTbl := L.CheckTable(2)
		var onSelectFn *lua.LFunction
		if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
			onSelectFn = L.CheckFunction(3)
		}

		items := make([]PickerItem, 0, itemsTbl.Len())
		for i := 1; i <= itemsTbl.Len(); i++ {
			v := itemsTbl.RawGetInt(i)
			switch elem := v.(type) {
			case lua.LString:
				items = append(items, PickerItem{
					ID:    fmt.Sprintf("%d", i),
					Label: string(elem),
				})
			case *lua.LTable:
				item := PickerItem{
					ID:     getTableString(elem, "id"),
					Label:  getTableString(elem, "label"),
					Detail: getTableString(elem, "detail"),
					Path:   getTableString(elem, "path"),
				}
				if line := getTableInt(elem, "line", 0); line > 0 {
					item.Line = line - 1
				}
				if col := getTableInt(elem, "col", 0); col > 0 {
					item.Col = col - 1
				}
				if item.Label == "" {
					item.Label = item.ID
				}
				items = append(items, item)
			}
		}

		var cb func(PickerItem)
		if onSelectFn != nil {
			cb = func(chosen PickerItem) {
				if m.L == nil {
					return
				}
				tbl := m.L.NewTable()
				tbl.RawSetString("id", lua.LString(chosen.ID))
				tbl.RawSetString("label", lua.LString(chosen.Label))
				tbl.RawSetString("detail", lua.LString(chosen.Detail))
				tbl.RawSetString("path", lua.LString(chosen.Path))
				tbl.RawSetString("line", lua.LNumber(chosen.Line+1))
				tbl.RawSetString("col", lua.LNumber(chosen.Col+1))
				if err := m.callLuaFn(onSelectFn, tbl); err != nil {
					m.addLog(fmt.Sprintf("[picker] callback error: %v", err))
				}
			}
		}

		m.activeAPI.ShowPicker(title, items, cb)
		return 0
	}))

	// larry.ui.show_panel(title, lines_or_string)
	L.SetField(uiTbl, "show_panel", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		title := L.CheckString(1)
		lines := luaValueToLines(L.CheckAny(2))
		m.activeAPI.ShowPanel(title, lines)
		return 0
	}))

	L.SetField(uiTbl, "close_panel", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI != nil {
			m.activeAPI.ClosePanel()
		}
		return 0
	}))

	// larry.ui.show_hover(title, lines_or_string)
	L.SetField(uiTbl, "show_hover", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		title := L.CheckString(1)
		lines := luaValueToLines(L.CheckAny(2))
		m.activeAPI.ShowHover(title, lines)
		return 0
	}))

	L.SetField(uiTbl, "close_hover", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI != nil {
			m.activeAPI.CloseHover()
		}
		return 0
	}))

	// larry.ui.show_completions(items, opts?, on_select?)
	// Also accepts 2-arg form: larry.ui.show_completions(items, on_select)
	L.SetField(uiTbl, "show_completions", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			return 0
		}
		itemsTbl := L.CheckTable(1)
		startRow := -1
		startCol := -1
		var onSelectFn *lua.LFunction

		if L.GetTop() >= 2 && L.Get(2) != lua.LNil {
			switch v := L.Get(2).(type) {
			case *lua.LFunction:
				onSelectFn = v
			case *lua.LTable:
				if line := getTableInt(v, "line", 0); line > 0 {
					startRow = line - 1
				}
				if col := getTableInt(v, "start_col", 0); col > 0 {
					startCol = col - 1
				}
			}
		}
		if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
			onSelectFn = L.CheckFunction(3)
		}

		items := make([]CompletionItem, 0, itemsTbl.Len())
		for i := 1; i <= itemsTbl.Len(); i++ {
			v := itemsTbl.RawGetInt(i)
			switch elem := v.(type) {
			case lua.LString:
				s := string(elem)
				if s != "" {
					items = append(items, CompletionItem{Label: s, InsertText: s})
				}
			case *lua.LTable:
				label := getTableString(elem, "label")
				insertText := getTableString(elem, "insert_text")
				if insertText == "" {
					insertText = getTableString(elem, "id")
				}
				if label == "" {
					label = insertText
				}
				if insertText == "" {
					insertText = label
				}
				if label == "" {
					continue
				}
				items = append(items, CompletionItem{
					Label:      label,
					InsertText: insertText,
					Kind:       getTableString(elem, "kind"),
					Detail:     getTableString(elem, "detail"),
					Doc:        getTableString(elem, "doc"),
				})
			}
		}

		var cb func(CompletionItem)
		if onSelectFn != nil {
			cb = func(chosen CompletionItem) {
				if m.L == nil {
					return
				}
				tbl := m.L.NewTable()
				tbl.RawSetString("label", lua.LString(chosen.Label))
				tbl.RawSetString("insert_text", lua.LString(chosen.InsertText))
				tbl.RawSetString("kind", lua.LString(chosen.Kind))
				tbl.RawSetString("detail", lua.LString(chosen.Detail))
				tbl.RawSetString("doc", lua.LString(chosen.Doc))
				if err := m.callLuaFn(onSelectFn, tbl); err != nil {
					m.addLog(fmt.Sprintf("[completions] callback error: %v", err))
				}
			}
		}

		m.activeAPI.ShowCompletions(items, startRow, startCol, cb)
		return 0
	}))

	L.SetField(uiTbl, "close_completions", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI != nil {
			m.activeAPI.CloseCompletions()
		}
		return 0
	}))

	L.SetField(uiTbl, "theme", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(lua.LString(m.cfg.Theme))
			return 1
		}
		L.Push(lua.LString(m.activeAPI.ThemeName()))
		return 1
	}))

	L.SetField(uiTbl, "palette", L.NewFunction(func(L *lua.LState) int {
		if m.activeAPI == nil {
			L.Push(L.NewTable())
			return 1
		}
		L.Push(goToLua(L, m.activeAPI.Palette()))
		return 1
	}))

	L.SetField(larryTbl, "ui", uiTbl)
}

func (m *Manager) registerDiagnosticsModule(L *lua.LState, larryTbl *lua.LTable) {
	diagTbl := L.NewTable()

	// larry.diagnostics.set(path, namespace, items)
	L.SetField(diagTbl, "set", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		if path == "" && m.activeAPI != nil {
			path = m.activeAPI.FilePath()
		}
		ns := L.CheckString(2)
		itemsTbl := L.CheckTable(3)

		diags := make([]Diagnostic, 0, itemsTbl.Len())
		for i := 1; i <= itemsTbl.Len(); i++ {
			tbl, ok := itemsTbl.RawGetInt(i).(*lua.LTable)
			if !ok {
				continue
			}
			line := getTableInt(tbl, "line", 1) - 1
			if line < 0 {
				line = 0
			}
			col := getTableInt(tbl, "col", 1) - 1
			if col < 0 {
				col = 0
			}
			endCol := getTableInt(tbl, "end_col", 0)
			if endCol > 0 {
				endCol--
			}
			sev := parseSeverity(tbl.RawGetString("severity"))
			diags = append(diags, Diagnostic{
				Line:     line,
				Col:      col,
				EndCol:   endCol,
				Severity: sev,
				Source:   getTableString(tbl, "source"),
				Code:     getTableString(tbl, "code"),
				Message:  getTableString(tbl, "message"),
			})
		}

		m.SetDiagnostics(path, ns, diags)
		return 0
	}))

	// larry.diagnostics.clear(path?, namespace?)
	L.SetField(diagTbl, "clear", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		ns := L.OptString(2, "")
		if path == "" && L.GetTop() < 1 && m.activeAPI != nil {
			path = m.activeAPI.FilePath()
		}
		m.ClearDiagnostics(path, ns)
		return 0
	}))

	// larry.diagnostics.get(path?) -> table
	L.SetField(diagTbl, "get", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		if path == "" && m.activeAPI != nil {
			path = m.activeAPI.FilePath()
		}
		diags := m.Diagnostics(path)
		tbl := L.NewTable()
		for i, d := range diags {
			item := L.NewTable()
			item.RawSetString("line", lua.LNumber(d.Line+1))
			item.RawSetString("col", lua.LNumber(d.Col+1))
			if d.EndCol > 0 {
				item.RawSetString("end_col", lua.LNumber(d.EndCol+1))
			}
			item.RawSetString("severity", lua.LNumber(int(d.Severity)))
			item.RawSetString("severity_name", lua.LString(d.Severity.String()))
			item.RawSetString("source", lua.LString(d.Source))
			item.RawSetString("code", lua.LString(d.Code))
			item.RawSetString("message", lua.LString(d.Message))
			tbl.RawSetInt(i+1, item)
		}
		L.Push(tbl)
		return 1
	}))

	L.SetField(larryTbl, "diagnostics", diagTbl)
}

func (m *Manager) registerGutterModule(L *lua.LState, larryTbl *lua.LTable) {
	gutterTbl := L.NewTable()

	// larry.gutter.set(path, namespace, signs)
	L.SetField(gutterTbl, "set", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		if path == "" && m.activeAPI != nil {
			path = m.activeAPI.FilePath()
		}
		ns := L.CheckString(2)
		signsTbl := L.CheckTable(3)

		signs := make([]GutterSign, 0, signsTbl.Len())
		for i := 1; i <= signsTbl.Len(); i++ {
			tbl, ok := signsTbl.RawGetInt(i).(*lua.LTable)
			if !ok {
				continue
			}
			line := getTableInt(tbl, "line", 1) - 1
			if line < 0 {
				line = 0
			}
			text := getTableString(tbl, "text")
			if text == "" {
				text = "●"
			}
			color := getTableString(tbl, "color")
			if color == "" {
				color = "accent"
			}
			prio := getTableInt(tbl, "priority", 50)
			signs = append(signs, GutterSign{
				Line:     line,
				Text:     text,
				Color:    color,
				Priority: prio,
			})
		}

		m.SetGutterSigns(path, ns, signs)
		return 0
	}))

	L.SetField(gutterTbl, "clear", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		ns := L.OptString(2, "")
		m.ClearGutterSigns(path, ns)
		return 0
	}))

	L.SetField(larryTbl, "gutter", gutterTbl)
}

func (m *Manager) registerHighlightsModule(L *lua.LState, larryTbl *lua.LTable) {
	hlTbl := L.NewTable()

	// larry.highlights.set(path, namespace, items)
	// Each item: { line = 1-based, col = 1-based inclusive, end_col = 1-based exclusive,
	//              fg = "#ffffff", bg = "#ff5555", bold = false, italic = false, underline = false, priority = 10 }
	L.SetField(hlTbl, "set", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		if path == "" && m.activeAPI != nil {
			path = m.activeAPI.FilePath()
		}
		ns := L.CheckString(2)
		itemsTbl := L.CheckTable(3)

		spans := make([]TextHighlight, 0, itemsTbl.Len())
		for i := 1; i <= itemsTbl.Len(); i++ {
			tbl, ok := itemsTbl.RawGetInt(i).(*lua.LTable)
			if !ok {
				continue
			}
			line := getTableInt(tbl, "line", 1) - 1
			col := getTableInt(tbl, "col", 1) - 1
			endCol := getTableInt(tbl, "end_col", 0) - 1
			if line < 0 || col < 0 || endCol <= col {
				continue
			}
			spans = append(spans, TextHighlight{
				Line:      line,
				Col:       col,
				EndCol:    endCol,
				FG:        getTableString(tbl, "fg"),
				BG:        getTableString(tbl, "bg"),
				Bold:      getTableBool(tbl, "bold"),
				Italic:    getTableBool(tbl, "italic"),
				Underline: getTableBool(tbl, "underline"),
				Priority:  getTableInt(tbl, "priority", 10),
			})
		}

		m.SetHighlights(path, ns, spans)
		return 0
	}))

	// larry.highlights.clear(path?, namespace?)
	L.SetField(hlTbl, "clear", L.NewFunction(func(L *lua.LState) int {
		path := L.OptString(1, "")
		ns := L.OptString(2, "")
		if path == "" && L.GetTop() < 1 && m.activeAPI != nil {
			path = m.activeAPI.FilePath()
		}
		m.ClearHighlights(path, ns)
		return 0
	}))

	L.SetField(larryTbl, "highlights", hlTbl)
}

func (m *Manager) registerCmdModule(L *lua.LState, larryTbl *lua.LTable) {
	cmdTbl := L.NewTable()

	// larry.cmd.register(id, opts_or_fn, fn?)
	L.SetField(cmdTbl, "register", L.NewFunction(func(L *lua.LState) int {
		id := strings.TrimSpace(L.CheckString(1))
		if id == "" {
			L.ArgError(1, "command id cannot be empty")
			return 0
		}

		info := CommandInfo{
			ID:       id,
			Title:    id,
			Category: "Plugin",
			Plugin:   m.currentPlugin,
		}
		var fn *lua.LFunction

		second := L.CheckAny(2)
		switch v := second.(type) {
		case *lua.LFunction:
			fn = v
		case *lua.LTable:
			if t := getTableString(v, "title"); t != "" {
				info.Title = t
			}
			if d := getTableString(v, "desc"); d != "" {
				info.Description = d
			} else if d := getTableString(v, "description"); d != "" {
				info.Description = d
			}
			if c := getTableString(v, "category"); c != "" {
				info.Category = c
			}
			if k := getTableString(v, "key"); k != "" {
				info.Key = m.normalizeKey(k)
			}
			fn = L.CheckFunction(3)
		default:
			L.ArgError(2, "expected options table or function")
			return 0
		}

		m.mu.Lock()
		m.commands[id] = &registeredCommand{info: info, fn: fn}
		if info.Key != "" {
			desc := info.Title
			if info.Description != "" {
				desc = info.Description
			}
			m.keymaps[info.Key] = &registeredKeymap{
				key:    info.Key,
				desc:   desc,
				plugin: info.Plugin,
				cmdID:  id,
			}
		}
		m.mu.Unlock()
		return 0
	}))

	L.SetField(cmdTbl, "run", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		if err := m.RunCommand(m.activeAPI, id); err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))

	L.SetField(larryTbl, "cmd", cmdTbl)
}

func (m *Manager) registerKeymapModule(L *lua.LState, larryTbl *lua.LTable) {
	kmTbl := L.NewTable()

	// larry.keymap.set(key, fn_or_cmd_id, desc?)
	L.SetField(kmTbl, "set", L.NewFunction(func(L *lua.LState) int {
		norm := m.normalizeKey(L.CheckString(1))
		if norm == "" {
			return 0
		}
		target := L.CheckAny(2)
		desc := L.OptString(3, "")

		km := &registeredKeymap{
			key:    norm,
			desc:   desc,
			plugin: m.currentPlugin,
		}
		switch v := target.(type) {
		case *lua.LFunction:
			km.fn = v
		case lua.LString:
			km.cmdID = string(v)
			if km.desc == "" {
				km.desc = km.cmdID
			}
		default:
			L.ArgError(2, "expected function or command id string")
			return 0
		}

		m.mu.Lock()
		m.keymaps[norm] = km
		m.mu.Unlock()
		return 0
	}))

	L.SetField(kmTbl, "del", L.NewFunction(func(L *lua.LState) int {
		norm := m.normalizeKey(L.CheckString(1))
		m.mu.Lock()
		delete(m.keymaps, norm)
		m.mu.Unlock()
		return 0
	}))

	L.SetField(larryTbl, "keymap", kmTbl)
}

func (m *Manager) registerExecModule(L *lua.LState, larryTbl *lua.LTable) {
	execTbl := L.NewTable()

	// larry.exec.has(binary) -> bool
	L.SetField(execTbl, "has", L.NewFunction(func(L *lua.LState) int {
		bin := L.CheckString(1)
		_, err := exec.LookPath(bin)
		L.Push(lua.LBool(err == nil))
		return 1
	}))

	// larry.exec.sync(cmd, args?, opts?) -> stdout, stderr, err
	L.SetField(execTbl, "sync", L.NewFunction(func(L *lua.LState) int {
		command := L.CheckString(1)
		args := luaTableToStrings(L.OptTable(2, nil))
		opts := parseExecOptions(L.OptTable(3, nil), m.activeAPI)
		res := runCommand(command, args, opts)
		L.Push(lua.LString(res.Stdout))
		L.Push(lua.LString(res.Stderr))
		if res.Err != "" {
			L.Push(lua.LString(res.Err))
		} else {
			L.Push(lua.LNil)
		}
		return 3
	}))

	// larry.exec.async(cmd, args?, opts?, callback)
	// Also accepts 3-arg form: larry.exec.async(cmd, args, callback)
	L.SetField(execTbl, "async", L.NewFunction(func(L *lua.LState) int {
		command := L.CheckString(1)
		args := luaTableToStrings(L.OptTable(2, nil))
		var opts ExecOptions
		var cb *lua.LFunction

		if L.GetTop() == 3 {
			cb = L.CheckFunction(3)
			opts = parseExecOptions(nil, m.activeAPI)
		} else {
			opts = parseExecOptions(L.OptTable(3, nil), m.activeAPI)
			cb = L.CheckFunction(4)
		}

		pluginName := m.currentPlugin
		go func() {
			res := runCommand(command, args, opts)
			m.DispatchAsync(AsyncEvent{
				Kind: "callback",
				Fn: func(api EditorAPI) {
					if m.L == nil {
						return
					}
					tbl := m.L.NewTable()
					tbl.RawSetString("stdout", lua.LString(res.Stdout))
					tbl.RawSetString("stderr", lua.LString(res.Stderr))
					tbl.RawSetString("exit_code", lua.LNumber(res.ExitCode))
					if res.Err != "" {
						tbl.RawSetString("err", lua.LString(res.Err))
					} else {
						tbl.RawSetString("err", lua.LNil)
					}
					if err := m.callLuaFn(cb, tbl); err != nil {
						m.addLog(fmt.Sprintf("[plugin:%s] exec.async callback error: %v", pluginName, err))
						if api != nil {
							api.SetStatus(fmt.Sprintf("Plugin error: %v", err))
						}
					}
				},
			})
		}()
		return 0
	}))

	L.SetField(larryTbl, "exec", execTbl)
}

func (m *Manager) registerLSPModule(L *lua.LState, larryTbl *lua.LTable) {
	lspTbl := L.NewTable()

	// larry.lsp.start({ name, cmd, args?, root?, env? }) -> client, err
	L.SetField(lspTbl, "start", L.NewFunction(func(L *lua.LState) int {
		opts := L.CheckTable(1)
		name := getTableString(opts, "name")
		command := getTableString(opts, "cmd")
		args := luaTableToStrings(getTableSubTable(opts, "args"))
		root := getTableString(opts, "root")
		if root == "" && m.activeAPI != nil {
			root = m.activeAPI.RootDir()
		}
		env := luaTableToStringMap(getTableSubTable(opts, "env"))

		client, err := StartLSPClient(name, command, args, root, env, m.DispatchAsync)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		m.mu.Lock()
		if old, exists := m.lspClients[client.Name]; exists {
			old.Stop()
		}
		m.lspClients[client.Name] = client
		m.mu.Unlock()

		L.Push(m.wrapLSPClient(L, client))
		L.Push(lua.LNil)
		return 2
	}))

	L.SetField(lspTbl, "clients", L.NewFunction(func(L *lua.LState) int {
		m.mu.RLock()
		defer m.mu.RUnlock()
		tbl := L.NewTable()
		i := 1
		for _, c := range m.lspClients {
			tbl.RawSetInt(i, m.wrapLSPClient(L, c))
			i++
		}
		L.Push(tbl)
		return 1
	}))

	L.SetField(lspTbl, "stop_all", L.NewFunction(func(L *lua.LState) int {
		m.mu.Lock()
		clients := make([]*LSPClient, 0, len(m.lspClients))
		for _, c := range m.lspClients {
			clients = append(clients, c)
		}
		m.lspClients = make(map[string]*LSPClient)
		m.mu.Unlock()

		for _, c := range clients {
			c.Stop()
		}
		return 0
	}))

	L.SetField(lspTbl, "path_to_uri", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(PathToURI(L.CheckString(1))))
		return 1
	}))

	L.SetField(lspTbl, "uri_to_path", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(URIToPath(L.CheckString(1))))
		return 1
	}))

	L.SetField(larryTbl, "lsp", lspTbl)
}

func (m *Manager) wrapLSPClient(L *lua.LState, client *LSPClient) *lua.LTable {
	tbl := L.NewTable()
	tbl.RawSetString("name", lua.LString(client.Name))
	tbl.RawSetString("root", lua.LString(client.RootDir))

	// client:is_running()
	L.SetField(tbl, "is_running", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(client.IsRunning()))
		return 1
	}))

	// client:request(method, params, callback) -> id, err
	L.SetField(tbl, "request", L.NewFunction(func(L *lua.LState) int {
		// Support both client:request(...) and client.request(...)
		offset := 1
		if _, ok := L.Get(1).(*lua.LTable); ok {
			offset = 2
		}
		method := L.CheckString(offset)
		var params any
		if L.GetTop() >= offset+1 && L.Get(offset+1) != lua.LNil {
			params = luaToGo(L.Get(offset + 1))
		}
		var cb *lua.LFunction
		if L.GetTop() >= offset+2 && L.Get(offset+2) != lua.LNil {
			cb = L.CheckFunction(offset + 2)
		}

		var goCB func(any, any)
		if cb != nil {
			goCB = func(result any, rpcErr any) {
				if m.L == nil {
					return
				}
				if err := m.callLuaFn(cb, goToLua(m.L, result), goToLua(m.L, rpcErr)); err != nil {
					m.addLog(fmt.Sprintf("[lsp:%s] response %q error: %v", client.Name, method, err))
				}
			}
		}

		id, err := client.Request(method, params, goCB)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LNumber(id))
		L.Push(lua.LNil)
		return 2
	}))

	// client:notify(method, params) -> err
	L.SetField(tbl, "notify", L.NewFunction(func(L *lua.LState) int {
		offset := 1
		if _, ok := L.Get(1).(*lua.LTable); ok {
			offset = 2
		}
		method := L.CheckString(offset)
		var params any
		if L.GetTop() >= offset+1 && L.Get(offset+1) != lua.LNil {
			params = luaToGo(L.Get(offset + 1))
		}
		if err := client.Notify(method, params); err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))

	// client:on_notification(method, fn)
	L.SetField(tbl, "on_notification", L.NewFunction(func(L *lua.LState) int {
		offset := 1
		if _, ok := L.Get(1).(*lua.LTable); ok {
			offset = 2
		}
		method := L.CheckString(offset)
		fn := L.CheckFunction(offset + 1)
		client.OnNotification(method, func(params any) {
			if m.L == nil {
				return
			}
			if err := m.callLuaFn(fn, goToLua(m.L, params)); err != nil {
				m.addLog(fmt.Sprintf("[lsp:%s] notification %q error: %v", client.Name, method, err))
			}
		})
		return 0
	}))

	// client:on_exit(fn)
	L.SetField(tbl, "on_exit", L.NewFunction(func(L *lua.LState) int {
		offset := 1
		if _, ok := L.Get(1).(*lua.LTable); ok {
			offset = 2
		}
		fn := L.CheckFunction(offset)
		client.OnExit(func(errMsg string) {
			if m.L == nil {
				return
			}
			var arg lua.LValue = lua.LNil
			if errMsg != "" {
				arg = lua.LString(errMsg)
			}
			_ = m.callLuaFn(fn, arg)
		})
		return 0
	}))

	// client:stop()
	L.SetField(tbl, "stop", L.NewFunction(func(L *lua.LState) int {
		client.Stop()
		m.mu.Lock()
		if cur, ok := m.lspClients[client.Name]; ok && cur == client {
			delete(m.lspClients, client.Name)
		}
		m.mu.Unlock()
		return 0
	}))

	return tbl
}

func (m *Manager) registerTimerModule(L *lua.LState, larryTbl *lua.LTable) {
	timerTbl := L.NewTable()

	L.SetField(timerTbl, "after", L.NewFunction(func(L *lua.LState) int {
		ms := L.CheckInt(1)
		fn := L.CheckFunction(2)
		if ms < 1 {
			ms = 1
		}
		id := m.nextTimerID.Add(1)
		t := time.AfterFunc(time.Duration(ms)*time.Millisecond, func() {
			m.mu.Lock()
			delete(m.timers, id)
			m.mu.Unlock()
			m.DispatchAsync(AsyncEvent{
				Kind: "callback",
				Fn: func(api EditorAPI) {
					if m.L == nil {
						return
					}
					if err := m.callLuaFn(fn); err != nil {
						m.addLog(fmt.Sprintf("[timer] error: %v", err))
					}
				},
			})
		})
		m.mu.Lock()
		m.timers[id] = t
		m.mu.Unlock()
		L.Push(lua.LNumber(id))
		return 1
	}))

	L.SetField(timerTbl, "cancel", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckInt64(1)
		m.mu.Lock()
		if t, ok := m.timers[id]; ok {
			t.Stop()
			delete(m.timers, id)
		}
		m.mu.Unlock()
		return 0
	}))

	L.SetField(larryTbl, "timer", timerTbl)
}

func (m *Manager) registerFSModule(L *lua.LState, larryTbl *lua.LTable) {
	fsTbl := L.NewTable()

	L.SetField(fsTbl, "exists", L.NewFunction(func(L *lua.LState) int {
		_, err := os.Stat(L.CheckString(1))
		L.Push(lua.LBool(err == nil))
		return 1
	}))

	L.SetField(fsTbl, "read", L.NewFunction(func(L *lua.LState) int {
		data, err := os.ReadFile(L.CheckString(1))
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(data)))
		L.Push(lua.LNil)
		return 2
	}))

	L.SetField(fsTbl, "dirname", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(filepath.Dir(L.CheckString(1))))
		return 1
	}))

	L.SetField(fsTbl, "basename", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(filepath.Base(L.CheckString(1))))
		return 1
	}))

	L.SetField(fsTbl, "join", L.NewFunction(func(L *lua.LState) int {
		n := L.GetTop()
		parts := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			parts = append(parts, L.CheckString(i))
		}
		L.Push(lua.LString(filepath.Join(parts...)))
		return 1
	}))

	L.SetField(larryTbl, "fs", fsTbl)
}

// ── Helpers ──

func getTableString(tbl *lua.LTable, key string) string {
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return ""
	}
	return v.String()
}

func getTableInt(tbl *lua.LTable, key string, fallback int) int {
	v := tbl.RawGetString(key)
	if n, ok := v.(lua.LNumber); ok {
		return int(n)
	}
	return fallback
}

func getTableBool(tbl *lua.LTable, key string) bool {
	v := tbl.RawGetString(key)
	if b, ok := v.(lua.LBool); ok {
		return bool(b)
	}
	return false
}

func getTableSubTable(tbl *lua.LTable, key string) *lua.LTable {
	v := tbl.RawGetString(key)
	if t, ok := v.(*lua.LTable); ok {
		return t
	}
	return nil
}

func luaTableToStrings(tbl *lua.LTable) []string {
	if tbl == nil {
		return nil
	}
	n := tbl.Len()
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, tbl.RawGetInt(i).String())
	}
	return out
}

func luaTableToStringMap(tbl *lua.LTable) map[string]string {
	if tbl == nil {
		return nil
	}
	out := make(map[string]string)
	tbl.ForEach(func(k, v lua.LValue) {
		out[k.String()] = v.String()
	})
	return out
}

func luaValueToLines(v lua.LValue) []string {
	switch val := v.(type) {
	case lua.LString:
		s := strings.ReplaceAll(string(val), "\r\n", "\n")
		return strings.Split(s, "\n")
	case *lua.LTable:
		return luaTableToStrings(val)
	default:
		if v == lua.LNil {
			return nil
		}
		return []string{v.String()}
	}
}

func parseSeverity(v lua.LValue) DiagnosticSeverity {
	switch val := v.(type) {
	case lua.LNumber:
		switch int(val) {
		case 1:
			return SeverityError
		case 2:
			return SeverityWarning
		case 3:
			return SeverityInfo
		case 4:
			return SeverityHint
		}
	case lua.LString:
		switch strings.ToLower(string(val)) {
		case "error", "err", "e":
			return SeverityError
		case "warning", "warn", "w":
			return SeverityWarning
		case "info", "i":
			return SeverityInfo
		case "hint", "h":
			return SeverityHint
		}
	}
	return SeverityWarning
}

func parseExecOptions(tbl *lua.LTable, api EditorAPI) ExecOptions {
	opts := ExecOptions{}
	if api != nil {
		opts.Dir = api.RootDir()
	}
	if tbl == nil {
		return opts
	}
	if d := getTableString(tbl, "dir"); d != "" {
		opts.Dir = d
	}
	if s := getTableString(tbl, "stdin"); s != "" {
		opts.Stdin = s
	}
	if t := getTableInt(tbl, "timeout_ms", 0); t > 0 {
		opts.TimeoutMs = t
	}
	if envTbl := getTableSubTable(tbl, "env"); envTbl != nil {
		opts.Env = luaTableToStringMap(envTbl)
	}
	return opts
}
