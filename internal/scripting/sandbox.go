package scripting

import (
	"regexp"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// removedGlobals are base library functions scripts must not have. They
// read files or compile code from strings (dofile, loadfile, load,
// loadstring), need the package library (module), force a full garbage
// collection (collectgarbage), or write to stdout (_printregs).
var removedGlobals = []string{"dofile", "loadfile", "load", "loadstring", "module", "collectgarbage", "_printregs"}

// newState returns a Lua state with only the sandboxed libraries: base
// (without removedGlobals), table, string and math. There is no os, io,
// debug or package library. print logs, and require reads from the
// scripts folder only.
func (v *vm) newState() *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	for _, lib := range []struct {
		name string
		open lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		L.Push(L.NewFunction(lib.open))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}
	for _, name := range removedGlobals {
		L.SetGlobal(name, lua.LNil)
	}
	L.SetField(L.GetGlobal(lua.StringLibName), "rep", L.NewFunction(strRep))
	L.SetGlobal("require", L.NewFunction(v.luaRequire))
	L.SetGlobal("print", L.NewFunction(v.luaPrint))
	for _, m := range v.modules() {
		L.SetGlobal(m.name, L.SetFuncs(L.NewTable(), m.funcs))
	}
	initRejectionType(L)
	v.initTypes(L)
	return L
}

// module is a global table of Go functions, part of the script API.
type module struct {
	name  string
	funcs map[string]lua.LGFunction
}

// modules returns the API modules. scripts/types/l2go.lua documents every
// function in them (and the methods of the userdata types), and a test
// keeps the two in sync.
func (v *vm) modules() []module {
	return []module{
		{name: "events", funcs: v.eventsFuncs()},
		{name: "html", funcs: v.htmlFuncs()},
		{name: "log", funcs: v.logFuncs()},
		{name: "script", funcs: v.scriptFuncs()},
	}
}

// moduleName is a dotted require name such as "lib.names".
var moduleName = regexp.MustCompile(`^[A-Za-z0-9_]+(\.[A-Za-z0-9_]+)*$`)

// modulePath maps a require name to a script path: "lib.names" is
// "lib/names.lua". Names with slashes, "..", or a leading dot (relative or
// absolute paths) are refused, as is the types folder.
func modulePath(name string) (string, bool) {
	if !moduleName.MatchString(name) {
		return "", false
	}
	path := strings.ReplaceAll(name, ".", "/") + ".lua"
	if kindOf(path) == kindTypes {
		return "", false
	}
	return path, true
}

// loadingModule marks a module whose file is running, to catch circular
// requires.
var loadingModule = &lua.LUserData{}

// luaRequire is require(name): it runs a script from the scripts folder
// once and returns its result (true if it returns nothing).
func (v *vm) luaRequire(L *lua.LState) int {
	name := L.CheckString(1)
	path, ok := modulePath(name)
	if !ok {
		L.RaiseError(`require %q: invalid module name, use a dotted name inside the scripts folder like "lib.names"`, name)
		return 0 // not reached: RaiseError does not return
	}
	L.Push(v.loadModule(L, path, name))
	return 1
}

// loadModule runs the script at path, or returns its cached result. It
// raises a Lua error on failure, so it must run inside a Lua call.
func (v *vm) loadModule(L *lua.LState, path, name string) lua.LValue {
	if val, ok := v.loaded[path]; ok {
		if val == loadingModule {
			L.RaiseError("require %q: circular require", name)
		}
		return val
	}
	proto, ok := v.protos[path]
	switch {
	case !ok && v.broken[path]:
		L.RaiseError("require %q: %s does not compile", name, v.display(path))
		return lua.LNil
	case !ok:
		L.RaiseError("require %q: %s not found in the scripts folder", name, path)
		return lua.LNil
	}
	v.loaded[path] = loadingModule
	v.running = append(v.running, path)
	defer func() {
		v.running = v.running[:len(v.running)-1]
		// The file failed (we are panicking); let a later require retry.
		if v.loaded[path] == loadingModule {
			delete(v.loaded, path)
		}
	}()
	L.Push(L.NewFunctionFromProto(proto))
	L.Call(0, 1)
	val := L.Get(-1)
	L.Pop(1)
	if val == lua.LNil {
		val = lua.LTrue
	}
	v.loaded[path] = val
	return val
}

// maxRepSize bounds the result of string.rep.
const maxRepSize = 1 << 20

// strRep is string.rep(s, n) with a size limit. A single huge string would
// run out of memory (and crash the server) before the event timeout could
// stop the script.
func strRep(L *lua.LState) int {
	s := L.CheckString(1)
	n := L.CheckInt(2)
	if n <= 0 || s == "" {
		L.Push(lua.LString(""))
		return 1
	}
	if len(s) > maxRepSize/n {
		L.RaiseError("string.rep: result longer than %d bytes", maxRepSize)
		return 0
	}
	L.Push(lua.LString(strings.Repeat(s, n)))
	return 1
}

// luaPrint is print(...): it logs the arguments, tab-separated, at info
// level.
func (v *vm) luaPrint(L *lua.LState) int {
	parts := make([]string, L.GetTop())
	for i := range parts {
		parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
	}
	v.logger.Info(strings.Join(parts, "\t"), "script", where(L))
	return 0
}

// where returns "file:line" of the Lua code calling the current Go
// function.
func where(L *lua.LState) string {
	return strings.TrimSuffix(L.Where(1), ":")
}
