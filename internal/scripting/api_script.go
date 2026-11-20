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

// npcScript is a script object made by script.new or quest.new: a name,
// as in the "Script <name> <event>" links of its pages, the IDs it is
// registered for and its handlers. It follows Mobius's Script and Quest
// classes; script.new(name) is quest.new(-1, name, "").
type npcScript struct {
	name string
	at   string // file:line of script.new or quest.new
	dir  string // folder of the file that made the script; page names are relative to it
	rel  string // that folder relative to the scripts folder, slash-separated

	id           int // quest ID; -1 for a plain script
	description  string
	custom       bool
	onEnterWorld bool
	requires     []string
	questItems   []int
	startNpcs    idList

	hooks [numHooks]hook
}

// isQuest reports whether the script is a quest (listed, abortable).
func (s *npcScript) isQuest() bool { return s.id > 0 }

// idList is a set of IDs that keeps the order they were added in.
type idList struct {
	order []int
	has   map[int]bool
}

// add adds id and reports whether it is new.
func (l *idList) add(id int) bool {
	if l.has == nil {
		l.has = make(map[int]bool)
	}
	if l.has[id] {
		return false
	}
	l.has[id] = true
	l.order = append(l.order, id)
	return true
}

// hook is one kind of handler of a script: the IDs registered for it, the
// handlers bound to some of them (the M4 form s:on_talk(ids, fn)) and the
// handler of every other registered ID (q:on_talk(fn)).
type hook struct {
	ids   idList
	bound map[int]registration
	def   *registration
}

// handlerFor returns the handler of kind k for id. Kinds without IDs use
// id 0.
func (s *npcScript) handlerFor(k hookKind, id int) (registration, bool) {
	h := &s.hooks[k]
	if !hookSpecs[k].ids {
		if h.def != nil {
			return *h.def, true
		}
		return registration{}, false
	}
	if r, ok := h.bound[id]; ok {
		return r, true
	}
	if h.def != nil && h.ids.has[id] {
		return *h.def, true
	}
	return registration{}, false
}

// hookKind is a kind of handler.
type hookKind int

const (
	hookTalk hookKind = iota
	hookFirstTalk
	hookEvent
	hookKill
	hookAttack
	hookSpawn
	hookSkillSee
	hookSpellFinished
	hookAggroRangeEnter
	hookEnterZone
	hookExitZone
	hookItemTalk
	hookDeath
	hookEnterWorld
	hookCheckPartyMember
	hookCheckPartyMemberState
	hookActionForEachPlayer
	hookStart
	// The AI hooks (Plan 13, A7).
	hookFactionCall
	hookCreatureSee
	hookEventReceived
	hookMoveFinished
	hookNodeArrived
	hookRouteFinished
	hookTeleport
	hookNpcHate
	hookDayNightChange
	numHooks
)

// hookSpec describes a kind of handler.
type hookSpec struct {
	name string // on_<name>
	ids  bool   // registered for IDs
	what string // what the IDs are: "NPC", "zone" or "item"
	add  string // the q:add_* function for the IDs, "" for none
	// desc names the handler in errors, e.g. "talk".
	desc string
}

var hookSpecs = [numHooks]hookSpec{
	hookTalk:                  {name: "talk", ids: true, what: "NPC", add: "add_talk_id", desc: "talk"},
	hookFirstTalk:             {name: "first_talk", ids: true, what: "NPC", add: "add_first_talk_id", desc: "first-talk"},
	hookEvent:                 {name: "event", desc: "event"},
	hookKill:                  {name: "kill", ids: true, what: "NPC", add: "add_kill_id", desc: "kill"},
	hookAttack:                {name: "attack", ids: true, what: "NPC", add: "add_attack_id", desc: "attack"},
	hookSpawn:                 {name: "spawn", ids: true, what: "NPC", add: "add_spawn_id", desc: "spawn"},
	hookSkillSee:              {name: "skill_see", ids: true, what: "NPC", add: "add_skill_see_id", desc: "skill-see"},
	hookSpellFinished:         {name: "spell_finished", ids: true, what: "NPC", add: "add_spell_finished_id", desc: "spell-finished"},
	hookAggroRangeEnter:       {name: "aggro_range_enter", ids: true, what: "NPC", add: "add_aggro_range_enter_id", desc: "aggro-range"},
	hookEnterZone:             {name: "enter_zone", ids: true, what: "zone", add: "add_enter_zone_id", desc: "enter-zone"},
	hookExitZone:              {name: "exit_zone", ids: true, what: "zone", add: "add_exit_zone_id", desc: "exit-zone"},
	hookItemTalk:              {name: "item_talk", ids: true, what: "item", add: "add_item_talk_id", desc: "item-talk"},
	hookDeath:                 {name: "death", desc: "death"},
	hookEnterWorld:            {name: "enter_world", desc: "enter-world"},
	hookCheckPartyMember:      {name: "check_party_member", desc: "party member check"},
	hookCheckPartyMemberState: {name: "check_party_member_state", desc: "party member state check"},
	hookActionForEachPlayer:   {name: "action_for_each_player", desc: "for-each-player"},
	hookStart:                 {name: "start", desc: "start"},
	hookFactionCall:           {name: "faction_call", ids: true, what: "NPC", add: "add_faction_call_id", desc: "faction-call"},
	hookCreatureSee:           {name: "creature_see", ids: true, what: "NPC", add: "add_creature_see_id", desc: "creature-see"},
	hookEventReceived:         {name: "event_received", ids: true, what: "NPC", add: "add_event_received_id", desc: "event-received"},
	hookMoveFinished:          {name: "move_finished", ids: true, what: "NPC", add: "add_move_finished_id", desc: "move-finished"},
	hookNodeArrived:           {name: "node_arrived", ids: true, what: "NPC", add: "add_node_arrived_id", desc: "node-arrived"},
	hookRouteFinished:         {name: "route_finished", ids: true, what: "NPC", add: "add_route_finished_id", desc: "route-finished"},
	hookTeleport:              {name: "teleport", ids: true, what: "NPC", add: "add_teleport_id", desc: "teleport"},
	hookNpcHate:               {name: "npc_hate", ids: true, what: "NPC", add: "add_npc_hate_id", desc: "NPC-hate"},
	hookDayNightChange:        {name: "day_night_change", desc: "day-night"},
}

// registration is a handler with the place it was registered.
type registration struct {
	handler
	at string // file:line of the call that registered it
}

// firstTalkOwner is the script that answers the first talk of an NPC.
type firstTalkOwner struct {
	script *npcScript
	at     string // where it claimed the NPC
}

// scriptNamePattern is what script names may contain. They appear in
// bypass links, which are split on spaces.
var scriptNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// scriptFuncs is the script module: script.new(name).
func (v *vm) scriptFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{"new": v.scriptNew}
}

// questFuncs is the quest module: quest.new(id, name, description).
func (v *vm) questFuncs() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{"new": v.questNew}
}

// scriptNew is script.new(name): quest.new(-1, name, "").
func (v *vm) scriptNew(L *lua.LState) int {
	name := L.CheckString(1)
	L.Push(v.newScript(L, "script.new", -1, name, ""))
	return 1
}

// questNew is quest.new(id, name, description).
func (v *vm) questNew(L *lua.LState) int {
	f := float64(L.CheckNumber(1))
	name := L.CheckString(2)
	desc := L.OptString(3, "")
	if f != math.Trunc(f) || (f < 1 && f != -1) || f > math.MaxInt32 {
		L.ArgError(1, "the quest ID must be a positive integer, or -1 for a plain script")
		return 0
	}
	L.Push(v.newScript(L, "quest.new", int(f), name, desc))
	return 1
}

// newScript creates a script while the scripts load. Names are unique
// across all scripts, quest IDs across all quests.
func (v *vm) newScript(L *lua.LState, fn string, id int, name, desc string) lua.LValue {
	if !v.loading {
		L.RaiseError("%s: scripts can only be created while the scripts load", fn)
		return nil
	}
	if !scriptNamePattern.MatchString(name) {
		L.ArgError(1, fmt.Sprintf("invalid script name %q: use 1 to 64 letters, digits and underscores", name))
		return nil
	}
	if old, ok := v.scripts[name]; ok {
		L.RaiseError("%s: a script named %q already exists (%s)", fn, name, old.at)
		return nil
	}
	if id > 0 {
		if old, ok := v.questIDs[id]; ok {
			L.RaiseError("%s: quest ID %d is already used by %s (%s)", fn, id, old.name, old.at)
			return nil
		}
	}
	rel := path.Dir(v.currentFile())
	s := &npcScript{
		name:        name,
		at:          where(L),
		dir:         filepath.Join(v.dir, filepath.FromSlash(rel)),
		rel:         rel,
		id:          id,
		description: desc,
	}
	v.scripts[name] = s
	v.order = append(v.order, s)
	if id > 0 {
		v.questIDs[id] = s
	}
	ud := L.NewUserData()
	ud.Value = s
	L.SetMetatable(ud, L.GetTypeMetatable(scriptType))
	return ud
}

// scriptMethods are the methods of Script values: registration, handlers
// and the quest helpers.
func (v *vm) scriptMethods() map[string]lua.LGFunction {
	m := map[string]lua.LGFunction{
		"name": func(L *lua.LState) int {
			L.Push(lua.LString(checkScript(L, "name").name))
			return 1
		},
		"id": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkScript(L, "id").id))
			return 1
		},
		"description": func(L *lua.LState) int {
			L.Push(lua.LString(checkScript(L, "description").description))
			return 1
		},
		"is_custom": func(L *lua.LState) int {
			L.Push(lua.LBool(checkScript(L, "is_custom").custom))
			return 1
		},
		"add_start_npc":        v.addStartNpc,
		"register_quest_items": v.registerQuestItems,
		"get_registered_item_ids": func(L *lua.LState) int {
			s := checkScript(L, "get_registered_item_ids")
			t := L.CreateTable(len(s.questItems), 0)
			for i, id := range s.questItems {
				t.RawSetInt(i+1, lua.LNumber(id))
			}
			L.Push(t)
			return 1
		},
		"set_on_enter_world": v.setFlag("set_on_enter_world", func(s *npcScript, b bool) { s.onEnterWorld = b }),
		"set_custom":         v.setFlag("set_custom", func(s *npcScript, b bool) { s.custom = b }),
		"requires":           v.requires,
	}
	for k, spec := range hookSpecs {
		m["on_"+spec.name] = v.onHook(hookKind(k))
		if spec.add != "" {
			m[spec.add] = v.addIDs(hookKind(k))
		}
	}
	for name, fn := range v.questHelpers() {
		m[name] = fn
	}
	return m
}

// checkLoading raises an error unless the scripts are loading.
func (v *vm) checkLoading(L *lua.LState, s *npcScript, method string) {
	if !v.loading {
		L.RaiseError("%s:%s: handlers can only be registered while the scripts load", s.name, method)
	}
}

// onHook returns q:on_<kind>. With IDs (the M4 form, kinds with IDs only)
// it registers them and binds fn to them; with fn alone it handles every
// other registered ID. Each form binds one handler per ID or kind.
func (v *vm) onHook(k hookKind) lua.LGFunction {
	spec := hookSpecs[k]
	method := "on_" + spec.name
	return func(L *lua.LState) int {
		s := checkScript(L, method)
		v.checkLoading(L, s, method)
		h := &s.hooks[k]
		if fn, ok := L.Get(2).(*lua.LFunction); ok {
			if h.def != nil {
				L.RaiseError("%s:%s: the script already has a%s %s handler (%s)", s.name, method, an(spec.desc), spec.desc, h.def.at)
				return 0
			}
			reg := newRegistration(L, fn)
			h.def = &reg
			return 0
		}
		if !spec.ids {
			L.CheckFunction(2)
			return 0
		}
		ids := checkNpcIDs(L, 2)
		reg := newRegistration(L, L.CheckFunction(3))
		if k == hookFirstTalk {
			v.claimFirstTalk(L, s, ids)
		}
		for _, id := range ids {
			if old, ok := h.bound[id]; ok {
				L.RaiseError("%s:%s: %s %d already has a%s %s handler in this script (%s)", s.name, method, spec.what, id, an(spec.desc), spec.desc, old.at)
				return 0
			}
		}
		if h.bound == nil {
			h.bound = make(map[int]registration)
		}
		for _, id := range ids {
			h.bound[id] = reg
			v.register(s, k, id)
		}
		return 0
	}
}

// an returns the "n" of "an" before a word starting with a vowel.
func an(word string) string {
	if word != "" && (word[0] == 'a' || word[0] == 'e' || word[0] == 'i' || word[0] == 'o' || word[0] == 'u') {
		return "n"
	}
	return ""
}

// register adds id to the IDs of kind k of s and to the engine's index.
func (v *vm) register(s *npcScript, k hookKind, id int) {
	if s.hooks[k].ids.add(id) {
		v.byID[k][id] = append(v.byID[k][id], s)
	}
}

// claimFirstTalk makes s the owner of the first talk of ids. Another
// script owning one of them is an error, as in Mobius; s itself may claim
// again.
func (v *vm) claimFirstTalk(L *lua.LState, s *npcScript, ids []int) {
	for _, id := range ids {
		if owner, ok := v.firstTalk[id]; ok && owner.script != s {
			L.RaiseError("%s: NPC %d already has a first-talk handler, from script %s (%s); only one script can own an NPC's first talk",
				s.name, id, owner.script.name, owner.at)
			return
		}
	}
	at := where(L)
	for _, id := range ids {
		if _, ok := v.firstTalk[id]; !ok {
			v.firstTalk[id] = firstTalkOwner{script: s, at: at}
		}
	}
}

// addIDs returns q:add_<kind>_id(...): it registers IDs for the handler
// of kind k, ignoring duplicates.
func (v *vm) addIDs(k hookKind) lua.LGFunction {
	spec := hookSpecs[k]
	return func(L *lua.LState) int {
		s := checkScript(L, spec.add)
		v.checkLoading(L, s, spec.add)
		ids := checkIDList(L, 2, spec.what, false)
		if k == hookFirstTalk {
			v.claimFirstTalk(L, s, ids)
		}
		for _, id := range ids {
			v.register(s, k, id)
		}
		return 0
	}
}

// addStartNpc is q:add_start_npc(...): the NPCs that offer the quest. Their
// talks reach the quest's talk handler too.
func (v *vm) addStartNpc(L *lua.LState) int {
	s := checkScript(L, "add_start_npc")
	v.checkLoading(L, s, "add_start_npc")
	for _, id := range checkIDList(L, 2, "NPC", false) {
		s.startNpcs.add(id)
		v.register(s, hookTalk, id)
	}
	return 0
}

// registerQuestItems is q:register_quest_items(...): the items exit_quest
// and abort take away. 0 is ignored.
func (v *vm) registerQuestItems(L *lua.LState) int {
	s := checkScript(L, "register_quest_items")
	v.checkLoading(L, s, "register_quest_items")
	for _, id := range checkIDList(L, 2, "item", true) {
		if id != 0 && !slices.Contains(s.questItems, id) {
			s.questItems = append(s.questItems, id)
		}
	}
	return 0
}

// setFlag returns a q:set_*(bool) method.
func (v *vm) setFlag(method string, set func(*npcScript, bool)) lua.LGFunction {
	return func(L *lua.LState) int {
		s := checkScript(L, method)
		v.checkLoading(L, s, method)
		set(s, L.OptBool(2, true))
		return 0
	}
}

// Features is what q:requires accepts: features later milestones bring.
var Features = []string{
	"clans", "clan_halls", "subclass", "noblesse", "pets", "fishing",
	"seven_signs", "sepulchers", "grand_bosses", "doors",
}

// requires is q:requires(feature, ...).
func (v *vm) requires(L *lua.LState) int {
	s := checkScript(L, "requires")
	v.checkLoading(L, s, "requires")
	for i := 2; i <= L.GetTop(); i++ {
		f := L.CheckString(i)
		if !slices.Contains(Features, f) {
			L.ArgError(i, fmt.Sprintf("unknown feature %q", f))
			return 0
		}
		if !slices.Contains(s.requires, f) {
			s.requires = append(s.requires, f)
		}
	}
	return 0
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

// checkNpcIDs reads argument n of the M4 form: one NPC template ID or a
// list of them, each once.
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
		id, ok := toID(val)
		if !ok || id < 1 {
			L.ArgError(n, "invalid NPC ID "+val.String())
			return nil
		}
		if slices.Contains(ids, id) {
			L.ArgError(n, fmt.Sprintf("NPC ID %d is listed twice", id))
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

// checkIDList reads the arguments from n on: numbers or lists of numbers,
// as Mobius's varargs take them. Duplicates are left in. zero allows 0.
func checkIDList(L *lua.LState, n int, what string, zero bool) []int {
	var ids []int
	add := func(i int, val lua.LValue) {
		id, ok := toID(val)
		if !ok || id < 0 || (id == 0 && !zero) {
			L.ArgError(i, fmt.Sprintf("invalid %s ID %s", what, val.String()))
		}
		ids = append(ids, id)
	}
	for i := n; i <= L.GetTop(); i++ {
		switch x := L.Get(i).(type) {
		case lua.LNumber:
			add(i, x)
		case *lua.LTable:
			for j := 1; j <= x.Len(); j++ {
				add(i, x.RawGetInt(j))
			}
		default:
			L.ArgError(i, fmt.Sprintf("%s ID or list of %s IDs expected, got %s", what, what, x.Type()))
			return nil
		}
	}
	return ids
}

// toID returns val as an integer ID up to MaxInt32.
func toID(val lua.LValue) (int, bool) {
	num, ok := val.(lua.LNumber)
	f := float64(num)
	if !ok || f != math.Trunc(f) || f < math.MinInt32 || f > math.MaxInt32 {
		return 0, false
	}
	return int(f), true
}

func scriptString(ud *lua.LUserData) string {
	if s, ok := ud.Value.(*npcScript); ok {
		if s.isQuest() {
			return fmt.Sprintf("quest %s (%d)", s.name, s.id)
		}
		return "script " + s.name
	}
	return "script"
}
