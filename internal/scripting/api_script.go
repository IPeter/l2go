package scripting

import (
	"fmt"
	"math"
	"path"
	"path/filepath"
	"regexp"
	"slices"

	lua "github.com/yuin/gopher-lua"
)

// npcScript is a script object made by script.new: a name, as in the
// "Script <name> <event>" links of its pages, and its NPC handlers. It
// follows Mobius's Script/Quest classes.
type npcScript struct {
	name  string
	at    string // file:line of script.new
	dir   string // folder of the file that made the script; page names are relative to it
	talk  map[int]registration
	event *registration
}

// registration is a handler with the place it was registered.
type registration struct {
	handler
	at string // file:line of the call that registered it
}

// firstTalkOwner is the script that answers the first talk of an NPC.
type firstTalkOwner struct {
	script *npcScript
	reg    registration
}

// scriptNamePattern is what script names may contain. They appear in
// bypass links, which are split on spaces.
var scriptNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// scriptFuncs is the script module: script.new(name).
func (v *vm) scriptFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{"new": v.scriptNew}
}

// scriptNew is script.new(name). It creates a script while the scripts
// load; names are unique across all scripts.
func (v *vm) scriptNew(L *lua.LState) int {
	name := L.CheckString(1)
	if !v.loading {
		L.RaiseError("script.new: scripts can only be created while the scripts load")
		return 0
	}
	if !scriptNamePattern.MatchString(name) {
		L.ArgError(1, fmt.Sprintf("invalid script name %q: use 1 to 64 letters, digits and underscores", name))
		return 0
	}
	if old, ok := v.scripts[name]; ok {
		L.RaiseError("script.new: a script named %q already exists (%s)", name, old.at)
		return 0
	}
	s := &npcScript{
		name: name,
		at:   where(L),
		dir:  filepath.Join(v.dir, filepath.FromSlash(path.Dir(v.currentFile()))),
		talk: make(map[int]registration),
	}
	v.scripts[name] = s
	ud := L.NewUserData()
	ud.Value = s
	L.SetMetatable(ud, L.GetTypeMetatable(scriptType))
	L.Push(ud)
	return 1
}

// scriptMethods are the methods of Script values.
func (v *vm) scriptMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"name": func(L *lua.LState) int {
			L.Push(lua.LString(checkScript(L, "name").name))
			return 1
		},
		"on_first_talk": v.onFirstTalk,
		"on_talk":       v.onTalk,
		"on_event":      v.onEvent,
	}
}

// onFirstTalk is s:on_first_talk(npc_ids, fn): fn answers the first click
// on those NPCs instead of their default dialog. Only one script may own
// the first talk of an NPC, as in Mobius.
func (v *vm) onFirstTalk(L *lua.LState) int {
	s := checkScript(L, "on_first_talk")
	v.checkLoading(L, s, "on_first_talk")
	ids := checkNpcIDs(L, 2)
	reg := newRegistration(L, L.CheckFunction(3))
	for _, id := range ids {
		if owner, ok := v.firstTalk[id]; ok {
			L.RaiseError("%s:on_first_talk: NPC %d already has a first-talk handler, from script %s (%s); only one script can own an NPC's first talk",
				s.name, id, owner.script.name, owner.reg.at)
			return 0
		}
	}
	for _, id := range ids {
		v.firstTalk[id] = firstTalkOwner{script: s, reg: reg}
	}
	return 0
}

// onTalk is s:on_talk(npc_ids, fn): fn answers the "Script <name>" link
// (Mobius's quest link) at those NPCs. Several scripts may talk at the
// same NPC; the game layer lets the player choose.
func (v *vm) onTalk(L *lua.LState) int {
	s := checkScript(L, "on_talk")
	v.checkLoading(L, s, "on_talk")
	ids := checkNpcIDs(L, 2)
	reg := newRegistration(L, L.CheckFunction(3))
	for _, id := range ids {
		if old, ok := s.talk[id]; ok {
			L.RaiseError("%s:on_talk: NPC %d already has a talk handler in this script (%s)", s.name, id, old.at)
			return 0
		}
	}
	for _, id := range ids {
		s.talk[id] = reg
		v.talkers[id] = append(v.talkers[id], s)
	}
	return 0
}

// onEvent is s:on_event(fn): fn handles the "Script <name> <event>" links.
func (v *vm) onEvent(L *lua.LState) int {
	s := checkScript(L, "on_event")
	v.checkLoading(L, s, "on_event")
	reg := newRegistration(L, L.CheckFunction(2))
	if s.event != nil {
		L.RaiseError("%s:on_event: the script already has an event handler (%s)", s.name, s.event.at)
		return 0
	}
	s.event = &reg
	return 0
}

func (v *vm) checkLoading(L *lua.LState, s *npcScript, method string) {
	if !v.loading {
		L.RaiseError("%s:%s: handlers can only be registered while the scripts load", s.name, method)
	}
}

func newRegistration(L *lua.LState, fn *lua.LFunction) registration {
	return registration{handler: newHandler(fn), at: where(L)}
}

// checkScript returns the script that method was called on.
func checkScript(L *lua.LState, method string) *npcScript {
	ud, _ := L.Get(1).(*lua.LUserData)
	var s *npcScript
	if ud != nil {
		s, _ = ud.Value.(*npcScript)
	}
	if s == nil {
		L.RaiseError("%s must be called on a script: s:%s(...)", method, method)
		return nil
	}
	return s
}

// checkNpcIDs reads argument n: one NPC template ID or a list of them.
func checkNpcIDs(L *lua.LState, n int) []int {
	var vals []lua.LValue
	switch x := L.Get(n).(type) {
	case lua.LNumber:
		vals = []lua.LValue{x}
	case *lua.LTable:
		for i := 1; i <= x.Len(); i++ {
			vals = append(vals, x.RawGetInt(i))
		}
	default:
		L.ArgError(n, "NPC ID or list of NPC IDs expected, got "+x.Type().String())
		return nil
	}
	if len(vals) == 0 {
		L.ArgError(n, "the list of NPC IDs is empty")
		return nil
	}
	ids := make([]int, 0, len(vals))
	for _, val := range vals {
		num, ok := val.(lua.LNumber)
		f := float64(num)
		if !ok || f != math.Trunc(f) || f < 1 || f > math.MaxInt32 {
			L.ArgError(n, "invalid NPC ID "+val.String())
			return nil
		}
		id := int(f)
		if slices.Contains(ids, id) {
			L.ArgError(n, fmt.Sprintf("NPC ID %d is listed twice", id))
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

func scriptString(ud *lua.LUserData) string {
	if s, ok := ud.Value.(*npcScript); ok {
		return "script " + s.name
	}
	return "script"
}
