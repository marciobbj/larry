package plugin

import (
	"encoding/json"
	"math"

	lua "github.com/yuin/gopher-lua"
)

// goToLua converts an arbitrary Go JSON-compatible value into a Lua LValue.
func goToLua(L *lua.LState, v any) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	switch val := v.(type) {
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(val)
	case int64:
		return lua.LNumber(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	case []any:
		tbl := L.NewTable()
		for i, item := range val {
			tbl.RawSetInt(i+1, goToLua(L, item))
		}
		return tbl
	case []string:
		tbl := L.NewTable()
		for i, item := range val {
			tbl.RawSetInt(i+1, lua.LString(item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, item := range val {
			tbl.RawSetString(k, goToLua(L, item))
		}
		return tbl
	case map[string]string:
		tbl := L.NewTable()
		for k, item := range val {
			tbl.RawSetString(k, lua.LString(item))
		}
		return tbl
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return lua.LNumber(f)
		}
		return lua.LString(val.String())
	default:
		// Fallback via JSON roundtrip for structured Go types.
		raw, err := json.Marshal(val)
		if err != nil {
			return lua.LNil
		}
		var generic any
		if err := json.Unmarshal(raw, &generic); err != nil {
			return lua.LNil
		}
		return goToLua(L, generic)
	}
}

// luaToGo converts a Lua LValue into a Go JSON-compatible value.
func luaToGo(lv lua.LValue) any {
	if lv == nil || lv == lua.LNil {
		return nil
	}
	switch v := lv.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		f := float64(v)
		if !math.IsNaN(f) && !math.IsInf(f, 0) && f == math.Trunc(f) && f >= math.MinInt64/2 && f <= math.MaxInt64/2 {
			return int64(f)
		}
		return f
	case lua.LString:
		return string(v)
	case *lua.LTable:
		return luaTableToGo(v)
	default:
		return nil
	}
}

func luaTableToGo(tbl *lua.LTable) any {
	// Detect whether this table is a pure 1..N array.
	maxIdx := 0
	count := 0
	isArray := true
	tbl.ForEach(func(k, _ lua.LValue) {
		count++
		if kn, ok := k.(lua.LNumber); ok {
			idx := int(kn)
			if float64(idx) == float64(kn) && idx >= 1 {
				if idx > maxIdx {
					maxIdx = idx
				}
				return
			}
		}
		isArray = false
	})

	if count == 0 {
		// Empty tables in LSP params/options are almost always JSON objects `{}`.
		return map[string]any{}
	}

	if isArray && maxIdx == count {
		arr := make([]any, count)
		for i := 1; i <= count; i++ {
			arr[i-1] = luaToGo(tbl.RawGetInt(i))
		}
		return arr
	}

	obj := make(map[string]any, count)
	tbl.ForEach(func(k, v lua.LValue) {
		obj[k.String()] = luaToGo(v)
	})
	return obj
}

func registerJSONModule(L *lua.LState, larryTbl *lua.LTable) {
	jsonTbl := L.NewTable()

	L.SetField(jsonTbl, "encode", L.NewFunction(func(L *lua.LState) int {
		val := luaToGo(L.CheckAny(1))
		raw, err := json.Marshal(val)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(raw)))
		L.Push(lua.LNil)
		return 2
	}))

	L.SetField(jsonTbl, "decode", L.NewFunction(func(L *lua.LState) int {
		input := L.CheckString(1)
		var out any
		if err := json.Unmarshal([]byte(input), &out); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(goToLua(L, out))
		L.Push(lua.LNil)
		return 2
	}))

	L.SetField(larryTbl, "json", jsonTbl)
}
