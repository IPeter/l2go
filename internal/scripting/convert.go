package scripting

import (
	"fmt"
	"math"

	lua "github.com/yuin/gopher-lua"
)

// maxTableDepth bounds how deep fromLua follows nested tables.
const maxTableDepth = 4

// toLua converts a Go value to a Lua value: nil, bool, string, numbers,
// []any and map[string]any (as tables), and refs marked with playerArg or
// npcArg (as Player and Npc values). Anything else becomes its fmt string.
func (v *vm) toLua(L *lua.LState, val any) lua.LValue {
	switch x := val.(type) {
	case nil:
		return lua.LNil
	case lua.LValue:
		return x
	case playerArg:
		return v.newPlayer(L, x.ref)
	case npcArg:
		return v.newNpc(L, x.ref)
	case bool:
		return lua.LBool(x)
	case string:
		return lua.LString(x)
	case int:
		return lua.LNumber(x)
	case int8:
		return lua.LNumber(x)
	case int16:
		return lua.LNumber(x)
	case int32:
		return lua.LNumber(x)
	case int64:
		return lua.LNumber(x)
	case uint:
		return lua.LNumber(x)
	case uint8:
		return lua.LNumber(x)
	case uint16:
		return lua.LNumber(x)
	case uint32:
		return lua.LNumber(x)
	case uint64:
		return lua.LNumber(x)
	case float32:
		return lua.LNumber(x)
	case float64:
		return lua.LNumber(x)
	case []any:
		t := L.CreateTable(len(x), 0)
		for i, e := range x {
			t.RawSetInt(i+1, v.toLua(L, e))
		}
		return t
	case map[string]any:
		t := L.CreateTable(0, len(x))
		for k, e := range x {
			t.RawSetString(k, v.toLua(L, e))
		}
		return t
	default:
		return lua.LString(fmt.Sprint(x))
	}
}

// fromLua converts a Lua value to Go, for log attributes. Whole numbers
// become int64, tables become []any (1..n keys) or map[string]any.
func fromLua(v lua.LValue) any { return fromLuaDepth(v, 0) }

func fromLuaDepth(v lua.LValue, depth int) any {
	switch x := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(x)
	case lua.LString:
		return string(x)
	case lua.LNumber:
		f := float64(x)
		if f == math.Trunc(f) && math.Abs(f) <= 1<<53 {
			return int64(f)
		}
		return f
	case *lua.LTable:
		if depth >= maxTableDepth {
			return x.String()
		}
		return tableFromLua(x, depth+1)
	default:
		return v.String()
	}
}

func tableFromLua(t *lua.LTable, depth int) any {
	n := t.Len()
	m := make(map[string]any)
	count := 0
	t.ForEach(func(k, v lua.LValue) {
		count++
		m[k.String()] = fromLuaDepth(v, depth)
	})
	if n > 0 && count == n {
		list := make([]any, n)
		for i := range n {
			list[i] = fromLuaDepth(t.RawGetInt(i+1), depth)
		}
		return list
	}
	return m
}
