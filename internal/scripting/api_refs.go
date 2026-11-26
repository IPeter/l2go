package scripting

import (
	"fmt"
	"math"

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

// PlayerItems is the part of a PlayerRef that reaches the player's items:
// player:adena(), player:item_count(), player:give_items() and
// player:take_items(). The game's players implement it; the methods raise
// a Lua error for a PlayerRef without it.
type PlayerItems interface {
	// Adena is the adena the player carries.
	Adena() int64
	// ItemCount is how many items of the template the player carries, worn
	// ones included.
	ItemCount(itemID int) int64
	// GiveItems gives count items of the template, with the reward
	// messages, and reports whether they were given.
	GiveItems(itemID int, count int64) bool
	// TakeItems takes count items of the template (all for a negative
	// count) and reports whether the player had that many.
	TakeItems(itemID int, count int64) bool
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
	stateType  = "l2go.queststate"
	timerType  = "l2go.questtimer"
	itemType   = "l2go.item"
	zoneType   = "l2go.zone"
	skillType  = "l2go.skill"
)

// playerArg and npcArg mark refs in Event.Data, so the event table gets
// Player and Npc values instead of strings.
type (
	playerArg struct{ ref PlayerRef }
	npcArg    struct{ ref NpcRef }
)

// playerBox is the Go value inside Player userdata. The ref is cleared
// when the run that handed it out ends, so a script that keeps a player in
// a global cannot use it later.
type playerBox struct{ ref PlayerRef }

// npcBox is the Go value inside Npc userdata: a handle (Plan 02, L4). It
// outlives the run that handed it out, so AI scripts can keep their boss
// or minions in fields, but it never trusts the Go ref across runs: the
// first use in a later run looks the NPC up again by object ID
// (QuestWorld.FindObject). An NPC that left the world, or whose object ID
// now belongs to another template, makes the handle invalid for good.
type npcBox struct {
	v    *vm
	ref  NpcRef // checked in run seen
	id   int32  // object ID
	tmpl int    // template ID
	seen uint64 // the run in which ref was handed out or looked up
	gone bool
}

// resolve returns the NPC behind the handle, or nil when it is gone. An
// NPC that leaves the world during a run stays usable until the run ends,
// as a deleted Mobius Npc object does; npc:valid() is false at once.
func (b *npcBox) resolve() NpcRef {
	if b == nil {
		return nil
	}
	v := b.v
	if b.seen == v.runSeq && v.depth > 0 && b.ref != nil {
		return b.ref
	}
	if b.gone {
		return nil
	}
	w := v.quest.world
	if w == nil || v.depth == 0 {
		return nil
	}
	n, ok := w.FindObject(b.id).(NpcRef)
	if !ok || n == nil || n.ID() != b.tmpl {
		v.npcGone(b.id)
		return nil
	}
	b.ref, b.seen = n, v.runSeq
	return n
}

// npcGone makes the handle of an NPC that left the world invalid.
func (v *vm) npcGone(objectID int32) {
	ud, ok := v.npcs[objectID]
	if !ok {
		return
	}
	delete(v.npcs, objectID)
	if b, ok := ud.Value.(*npcBox); ok {
		b.gone = true
	}
}

// refKey identifies the object behind a ref, so that one object is one Lua
// value within a run (Java's identity checks, st == st2, keep working).
type refKey struct {
	kind  byte // 'p' player, 'n' NPC, 's' quest state
	id    int32
	quest string // quest states only
}

// newPlayer returns a Player value for p, or nil. Within a run, the same
// player is always the same value.
func (v *vm) newPlayer(L *lua.LState, p PlayerRef) lua.LValue {
	if p == nil {
		return lua.LNil
	}
	key := refKey{kind: 'p', id: p.ObjectID()}
	if ud, ok := v.refCache[key]; ok {
		return ud
	}
	ud := v.newRef(L, &playerBox{ref: p}, playerType)
	v.refCache[key] = ud
	return ud
}

// newNpc returns the Npc handle of n, or nil. The same NPC is always the
// same value while it stays in the world, across runs too.
func (v *vm) newNpc(L *lua.LState, n NpcRef) lua.LValue {
	if n == nil {
		return lua.LNil
	}
	id, tmpl := n.ObjectID(), n.ID()
	if ud, ok := v.npcs[id]; ok {
		if b, ok := ud.Value.(*npcBox); ok && b.tmpl == tmpl {
			b.ref, b.seen = n, v.runSeq
			return ud
		}
		v.npcGone(id)
	}
	ud := L.NewUserData()
	ud.Value = &npcBox{v: v, ref: n, id: id, tmpl: tmpl, seen: v.runSeq}
	L.SetMetatable(ud, L.GetTypeMetatable(npcType))
	v.npcs[id] = ud
	return ud
}

// npcHandle returns argument n's handle, nil when it is not an Npc.
func npcHandle(L *lua.LState, n int) *npcBox {
	ud, ok := L.Get(n).(*lua.LUserData)
	if !ok {
		return nil
	}
	b, _ := ud.Value.(*npcBox)
	return b
}

// valid reports whether the handle's NPC is in the world.
func (b *npcBox) valid() bool {
	return b.resolve() != nil && !b.gone
}

// goneNpcMsg explains an invalid Npc handle.
const goneNpcMsg = "this NPC is gone (it left the world, or it came from an earlier handler and cannot be looked up); check npc:valid() first"

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
		case *stateBox:
			b.st, b.player = nil, nil
		}
		v.refs[i] = nil
	}
	v.refs = v.refs[:0]
	clear(v.refCache)
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
		{name: "Player", meta: playerType, methods: v.playerMethods(), tostring: playerString},
		{name: "Npc", meta: npcType, methods: v.npcMethods(), tostring: npcString},
		{name: "QuestState", meta: stateType, methods: v.stateMethods(), tostring: stateString},
		{name: "QuestTimer", meta: timerType, methods: v.timerMethods(), tostring: timerString},
		{name: "Item", meta: itemType, methods: itemMethods(), tostring: itemString},
		{name: "Zone", meta: zoneType, methods: zoneMethods(), tostring: zoneString},
		{name: "Skill", meta: skillType, methods: v.skillMethods(), tostring: skillString},
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
func (v *vm) playerMethods() map[string]lua.LGFunction {
	m := map[string]lua.LGFunction{
		"object_id": playerGetter("object_id", func(p PlayerRef) lua.LValue { return lua.LNumber(p.ObjectID()) }),
		"name":      playerGetter("name", func(p PlayerRef) lua.LValue { return lua.LString(p.Name()) }),
		"level":     playerGetter("level", func(p PlayerRef) lua.LValue { return lua.LNumber(p.Level()) }),
		"class_id":  playerGetter("class_id", func(p PlayerRef) lua.LValue { return lua.LNumber(p.ClassID()) }),
		"race":      playerGetter("race", func(p PlayerRef) lua.LValue { return lua.LNumber(p.Race()) }),
		"adena": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkPlayerItems(L, "adena").Adena()))
			return 1
		},
		"item_count": func(L *lua.LState) int {
			items := checkPlayerItems(L, "item_count")
			L.Push(lua.LNumber(items.ItemCount(checkItemID(L, "item_count"))))
			return 1
		},
		"give_items": func(L *lua.LState) int {
			items := checkPlayerItems(L, "give_items")
			id := checkItemID(L, "give_items")
			count := checkCount(L, "give_items", 1)
			if count <= 0 {
				L.ArgError(3, "player:give_items(): the count must be at least 1")
				return 0
			}
			L.Push(lua.LBool(items.GiveItems(id, count)))
			return 1
		},
		"take_items": func(L *lua.LState) int {
			items := checkPlayerItems(L, "take_items")
			id := checkItemID(L, "take_items")
			count := checkCount(L, "take_items", -1)
			if count == 0 {
				L.ArgError(3, "player:take_items(): the count must not be 0 (-1 takes all)")
				return 0
			}
			L.Push(lua.LBool(items.TakeItems(id, count)))
			return 1
		},
	}
	for name, fn := range v.playerQuestMethods() {
		m[name] = fn
	}
	for name, fn := range v.areaPlayerMethods() { // ai_areas.go
		m[name] = fn
	}
	return m
}

// checkPlayerItems returns the items of the player method was called on.
func checkPlayerItems(L *lua.LState, method string) PlayerItems {
	p := checkPlayer(L, method)
	items, ok := p.(PlayerItems)
	if !ok {
		L.RaiseError("player:%s(): this player has no items", method)
		return nil
	}
	return items
}

// checkItemID returns argument 2, an item template ID.
func checkItemID(L *lua.LState, method string) int {
	v := float64(L.CheckNumber(2))
	if v != math.Trunc(v) || v < 1 || v > math.MaxInt32 {
		L.ArgError(2, fmt.Sprintf("player:%s(): the item ID must be a positive integer", method))
		return 0
	}
	return int(v)
}

// checkCount returns argument 3, an item count, or def when it is absent.
func checkCount(L *lua.LState, method string, def int64) int64 {
	if L.Get(3) == lua.LNil {
		return def
	}
	v := float64(L.CheckNumber(3))
	if v != math.Trunc(v) || v < math.MinInt32 || v > math.MaxInt32 {
		L.ArgError(3, fmt.Sprintf("player:%s(): the count must be an integer", method))
		return 0
	}
	return int64(v)
}

func playerGetter(method string, get func(PlayerRef) lua.LValue) lua.LGFunction {
	return func(L *lua.LState) int {
		L.Push(get(checkPlayer(L, method)))
		return 1
	}
}

// npcMethods are the methods of Npc values.
func (v *vm) npcMethods() map[string]lua.LGFunction {
	m := map[string]lua.LGFunction{
		"object_id": npcGetter("object_id", func(n NpcRef) lua.LValue { return lua.LNumber(n.ObjectID()) }),
		"id":        npcGetter("id", func(n NpcRef) lua.LValue { return lua.LNumber(n.ID()) }),
		"name":      npcGetter("name", func(n NpcRef) lua.LValue { return lua.LString(n.Name()) }),
		"say":       v.npcSay,
	}
	for name, fn := range v.npcQuestMethods() {
		m[name] = fn
	}
	for name, fn := range v.areaNpcMethods() { // ai_areas.go
		m[name] = fn
	}
	return m
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
	n := b.resolve()
	if n == nil {
		L.RaiseError("npc:%s(): %s", method, goneNpcMsg)
		return nil
	}
	return n
}

func playerString(ud *lua.LUserData) string {
	if b, ok := ud.Value.(*playerBox); ok && b.ref != nil {
		return fmt.Sprintf("player %s (%d)", b.ref.Name(), b.ref.ObjectID())
	}
	return "player (expired)"
}

func npcString(ud *lua.LUserData) string {
	b, ok := ud.Value.(*npcBox)
	if !ok {
		return "npc"
	}
	if n := b.resolve(); n != nil {
		return fmt.Sprintf("npc %s (%d, object %d)", n.Name(), n.ID(), n.ObjectID())
	}
	return fmt.Sprintf("npc %d (object %d, gone)", b.tmpl, b.id)
}
