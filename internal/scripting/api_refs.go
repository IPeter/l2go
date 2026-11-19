package scripting

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// PlayerRef is a player as scripts see it (the Player class in
// types/l2go.lua). The game layer implements it; the engine calls it only
// on the goroutine that called the engine (the world loop), and only while
// the handler that got it runs.
type PlayerRef interface {
	ObjectID() int32
	Name() string
	Level() int
	// ClassID is the current class (Mobius ClassId).
	ClassID() int
	// Race is 0 human, 1 elf, 2 dark elf, 3 orc, 4 dwarf.
	Race() int
}

// NpcRef is an NPC as scripts see it (the Npc class in types/l2go.lua).
// The game layer implements it, with the same rules as PlayerRef.
type NpcRef interface {
	ObjectID() int32
	// ID is the NPC template ID, for example 30598 for a Newbie Guide.
	ID() int
	Name() string
}

// Output carries what scripts do to the game world, besides what their
// handlers return. The game layer implements it and passes it in Options.
// Its methods run on the goroutine that called the engine (the world
// loop), while a handler runs.
type Output interface {
	// NpcSay makes npc say text to the players around it, in normal NPC
	// chat (Mobius npc.broadcastSay with NPC_GENERAL; the NpcSay packet).
	NpcSay(npc NpcRef, text string)
}

// Metatable names of the userdata values scripts get.
const (
	playerType = "l2go.player"
	npcType    = "l2go.npc"
	scriptType = "l2go.script"
)

// playerArg and npcArg mark refs in Event.Data, so the event table gets
// Player and Npc values instead of strings.
type (
	playerArg struct{ ref PlayerRef }
	npcArg    struct{ ref NpcRef }
)

// playerBox and npcBox are the Go values inside Player and Npc userdata.
// The ref is cleared when the run that handed it out ends, so a script
// that keeps a player in a global cannot use it later.
type (
	playerBox struct{ ref PlayerRef }
	npcBox    struct{ ref NpcRef }
)

// newPlayer returns a Player value for p, or nil.
func (v *vm) newPlayer(L *lua.LState, p PlayerRef) lua.LValue {
	if p == nil {
		return lua.LNil
	}
	return v.newRef(L, &playerBox{ref: p}, playerType)
}

// newNpc returns an Npc value for n, or nil.
func (v *vm) newNpc(L *lua.LState, n NpcRef) lua.LValue {
	if n == nil {
		return lua.LNil
	}
	return v.newRef(L, &npcBox{ref: n}, npcType)
}

func (v *vm) newRef(L *lua.LState, box any, typ string) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = box
	L.SetMetatable(ud, L.GetTypeMetatable(typ))
	v.refs = append(v.refs, ud)
	return ud
}

// expireRefs clears every ref handed out since the last call. It runs when
// the outermost run ends.
func (v *vm) expireRefs() {
	for i, ud := range v.refs {
		switch b := ud.Value.(type) {
		case *playerBox:
			b.ref = nil
		case *npcBox:
			b.ref = nil
		}
		v.refs[i] = nil
	}
	v.refs = v.refs[:0]
}

// luaType is a userdata class of the script API. scripts/types/l2go.lua
// documents its methods, and a test keeps the two in sync.
type luaType struct {
	name     string // class name in types/l2go.lua
	meta     string // metatable name
	methods  map[string]lua.LGFunction
	tostring func(ud *lua.LUserData) string
}

// types returns the userdata classes.
func (v *vm) types() []luaType {
	return []luaType{
		{name: "Script", meta: scriptType, methods: v.scriptMethods(), tostring: scriptString},
		{name: "Player", meta: playerType, methods: playerMethods(), tostring: playerString},
		{name: "Npc", meta: npcType, methods: v.npcMethods(), tostring: npcString},
	}
}

// initTypes registers the metatables of the userdata classes. Methods are
// looked up through __index; the metatables cannot be read or changed
// from Lua.
func (v *vm) initTypes(L *lua.LState) {
	for _, t := range v.types() {
		mt := L.NewTypeMetatable(t.meta)
		L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), t.methods))
		L.SetField(mt, "__tostring", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LString(t.tostring(L.CheckUserData(1))))
			return 1
		}))
		L.SetField(mt, "__metatable", lua.LFalse)
	}
}

// playerMethods are the methods of Player values.
func playerMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"object_id": playerGetter("object_id", func(p PlayerRef) lua.LValue { return lua.LNumber(p.ObjectID()) }),
		"name":      playerGetter("name", func(p PlayerRef) lua.LValue { return lua.LString(p.Name()) }),
		"level":     playerGetter("level", func(p PlayerRef) lua.LValue { return lua.LNumber(p.Level()) }),
		"class_id":  playerGetter("class_id", func(p PlayerRef) lua.LValue { return lua.LNumber(p.ClassID()) }),
		"race":      playerGetter("race", func(p PlayerRef) lua.LValue { return lua.LNumber(p.Race()) }),
	}
}

func playerGetter(method string, get func(PlayerRef) lua.LValue) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(get(checkPlayer(L, method)))
		return 1
	}
}

// npcMethods are the methods of Npc values.
func (v *vm) npcMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"object_id": npcGetter("object_id", func(n NpcRef) lua.LValue { return lua.LNumber(n.ObjectID()) }),
		"id":        npcGetter("id", func(n NpcRef) lua.LValue { return lua.LNumber(n.ID()) }),
		"name":      npcGetter("name", func(n NpcRef) lua.LValue { return lua.LString(n.Name()) }),
		"say":       v.npcSay,
	}
}

func npcGetter(method string, get func(NpcRef) lua.LValue) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(get(checkNpc(L, method)))
		return 1
	}
}

// npcSay is npc:say(text).
func (v *vm) npcSay(L *lua.LState) int {
	n := checkNpc(L, "say")
	text := L.CheckString(2)
	if text == "" {
		return 0
	}
	if v.output == nil {
		v.logger.Debug("npc:say has no output", "npc", n.ID(), "text", text, "script", where(L))
		return 0
	}
	v.output.NpcSay(n, text)
	return 0
}

// checkPlayer returns the player that method was called on. RaiseError
// does not return; the returns after it are for readers.
func checkPlayer(L *lua.LState, method string) PlayerRef {
	ud, _ := L.Get(1).(*lua.LUserData)
	var b *playerBox
	if ud != nil {
		b, _ = ud.Value.(*playerBox)
	}
	if b == nil {
		L.RaiseError("%s must be called on a player: player:%s()", method, method)
		return nil
	}
	if b.ref == nil {
		L.RaiseError("player:%s(): this player was passed to an earlier handler; keep player:object_id() instead of the player", method)
		return nil
	}
	return b.ref
}

// checkNpc returns the NPC that method was called on.
func checkNpc(L *lua.LState, method string) NpcRef {
	ud, _ := L.Get(1).(*lua.LUserData)
	var b *npcBox
	if ud != nil {
		b, _ = ud.Value.(*npcBox)
	}
	if b == nil {
		L.RaiseError("%s must be called on an NPC: npc:%s()", method, method)
		return nil
	}
	if b.ref == nil {
		L.RaiseError("npc:%s(): this NPC was passed to an earlier handler; keep npc:object_id() instead of the NPC", method)
		return nil
	}
	return b.ref
}

func playerString(ud *lua.LUserData) string {
	if b, ok := ud.Value.(*playerBox); ok && b.ref != nil {
		return fmt.Sprintf("player %s (%d)", b.ref.Name(), b.ref.ObjectID())
	}
	return "player (expired)"
}

func npcString(ud *lua.LUserData) string {
	if b, ok := ud.Value.(*npcBox); ok && b.ref != nil {
		return fmt.Sprintf("npc %s (%d, object %d)", b.ref.Name(), b.ref.ID(), b.ref.ObjectID())
	}
	return "npc (expired)"
}
