package scripting

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// htmlFuncs is the html module: html.replace(page, vars).
func (v *vm) htmlFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{"replace": luaHTMLReplace}
}

// luaHTMLReplace is html.replace(page, vars): it replaces %name% in page
// with vars.name.
func luaHTMLReplace(L *lua.LState) int {
	page := L.CheckString(1)
	vars, err := varsFromLua(L.CheckTable(2))
	if err != nil {
		L.ArgError(2, err.Error())
		return 0
	}
	L.Push(lua.LString(ReplaceVars(page, vars)))
	return 1
}

// ReplaceVars replaces every %name% in text with vars[name], in one pass:
// a value is never searched for placeholders again. Names may also be
// given with their percent signs ("%name%"). Placeholders without a value
// stay as they are. html.replace in Lua is this function, and the game
// layer uses it for Reply.Vars.
func ReplaceVars(text string, vars map[string]string) string {
	if len(vars) == 0 || !strings.Contains(text, "%") {
		return text
	}
	pairs := make([]string, 0, 2*len(vars))
	for _, name := range slices.Sorted(maps.Keys(vars)) {
		if key := strings.Trim(name, "%"); key != "" {
			pairs = append(pairs, "%"+key+"%", vars[name])
		}
	}
	if len(pairs) == 0 {
		return text
	}
	return strings.NewReplacer(pairs...).Replace(text)
}

// varsFromLua reads a placeholder table: string keys, and string, number
// or boolean values.
func varsFromLua(t *lua.LTable) (map[string]string, error) {
	vars := make(map[string]string)
	var err error
	t.ForEach(func(k, val lua.LValue) {
		if err != nil {
			return
		}
		key, ok := k.(lua.LString)
		if !ok {
			err = fmt.Errorf("placeholder names must be strings, got a %s", k.Type())
			return
		}
		switch x := val.(type) {
		case lua.LString, lua.LNumber, lua.LBool:
			vars[string(key)] = x.String()
		default:
			err = fmt.Errorf("placeholder %q has a %s value; use a string or a number", string(key), val.Type())
		}
	})
	return vars, err
}
