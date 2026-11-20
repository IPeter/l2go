package scripting

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// defaultPartyRange is Mobius's AltPartyRange (Player.ini).
const defaultPartyRange = 1500

// locationOf returns the position of a ref, false if it has none.
func locationOf(obj any) (Location, bool) {
	if l, ok := obj.(Located); ok {
		return l.Location(), true
	}
	return Location{}, false
}

// Collider is a PlayerRef or NpcRef with a collision radius, which Mobius
// LocationUtil.checkIfInRange adds to the range. Without it the radius is 0.
type Collider interface {
	CollisionRadius() float64
}

func collisionRadius(obj any) float64 {
	if c, ok := obj.(Collider); ok {
		return c.CollisionRadius()
	}
	return 0
}

// distance returns the distance between two positions, in 2D or 3D.
func distance(a, b Location, withZ bool) float64 {
	dx, dy := float64(a.X-b.X), float64(a.Y-b.Y)
	d := dx*dx + dy*dy
	if withZ {
		dz := float64(a.Z - b.Z)
		d += dz * dz
	}
	return math.Sqrt(d)
}

// inRange is Mobius LocationUtil.checkIfInRange(range, a, b, includeZ):
// false for a missing object; objects without a position count as in
// range.
func inRange(r int, a, b any, withZ bool) bool {
	if a == nil || b == nil {
		return false
	}
	if r == -1 {
		return true
	}
	la, okA := locationOf(a)
	lb, okB := locationOf(b)
	if !okA || !okB {
		return true
	}
	return distance(la, lb, withZ) <= float64(r)+collisionRadius(a)+collisionRadius(b)
}

// playerAs returns the player method was called on as T, or raises an
// error naming what the player lacks.
func playerAs[T any](L *lua.LState, method, what string) T {
	p := checkPlayer(L, method)
	x, ok := p.(T)
	if !ok {
		L.RaiseError("player:%s(): this player has no %s", method, what)
	}
	return x
}

// npcAs returns the NPC method was called on as T, or raises an error.
func npcAs[T any](L *lua.LState, method, what string) T {
	n := checkNpc(L, method)
	x, ok := n.(T)
	if !ok {
		L.RaiseError("npc:%s(): this NPC has no %s", method, what)
	}
	return x
}

// locationMethods are x, y, z and heading for a ref type; self returns the
// ref.
func locationMethods(kind string, self func(L *lua.LState, method string) any) map[string]lua.LGFunction {
	coord := func(method string, get func(Location) int) lua.LGFunction {
		return func(L *lua.LState) int {
			loc, ok := locationOf(self(L, method))
			if !ok {
				L.RaiseError("%s:%s(): this %s has no position", kind, method, kind)
				return 0
			}
			L.Push(lua.LNumber(get(loc)))
			return 1
		}
	}
	radius := func(method string, withZ bool) lua.LGFunction {
		return func(L *lua.LState) int {
			obj := self(L, method)
			other := objectArg(L, 2)
			r := float64(L.CheckNumber(3))
			la, okA := locationOf(obj)
			lb, okB := locationOf(other)
			if other == nil || !okA || !okB {
				L.Push(lua.LFalse)
				return 1
			}
			L.Push(lua.LBool(distance(la, lb, withZ) <= r))
			return 1
		}
	}
	return map[string]lua.LGFunction{
		"x":                   coord("x", func(l Location) int { return l.X }),
		"y":                   coord("y", func(l Location) int { return l.Y }),
		"z":                   coord("z", func(l Location) int { return l.Z }),
		"heading":             coord("heading", func(l Location) int { return l.Heading }),
		"is_inside_radius_2d": radius("is_inside_radius_2d", false),
		"is_inside_radius_3d": radius("is_inside_radius_3d", true),
	}
}

// playerQuestMethods are the Player methods quests use (Mobius Player in
// M4's style).
func (v *vm) playerQuestMethods() map[string]lua.LGFunction {
	self := func(L *lua.LState, method string) any { return checkPlayer(L, method) }
	details := func(method string, get func(PlayerDetails) lua.LValue) lua.LGFunction {
		return func(L *lua.LState) int {
			L.Push(get(playerAs[PlayerDetails](L, method, "details")))
			return 1
		}
	}
	solo := func(method string, val lua.LValue) lua.LGFunction {
		return func(L *lua.LState) int {
			checkPlayer(L, method)
			L.Push(val)
			return 1
		}
	}
	m := map[string]lua.LGFunction{
		"get_quest_state": func(L *lua.LState) int {
			p := checkPlayer(L, "get_quest_state")
			name := L.CheckString(2)
			st := v.needWorld(L, "player:get_quest_state").QuestState(p, name)
			L.Push(v.newQuestState(L, p, st))
			return 1
		},
		"sex":           details("sex", func(d PlayerDetails) lua.LValue { return lua.LNumber(d.Sex()) }),
		"base_class":    details("base_class", func(d PlayerDetails) lua.LValue { return lua.LNumber(d.BaseClassID()) }),
		"is_mage_class": details("is_mage_class", func(d PlayerDetails) lua.LValue { return lua.LBool(d.IsMageClass()) }),
		"is_newbie":     details("is_newbie", func(d PlayerDetails) lua.LValue { return lua.LBool(d.IsNewbie()) }),
		"is_dead":       details("is_dead", func(d PlayerDetails) lua.LValue { return lua.LBool(d.IsDead()) }),
		"is_in_category": func(L *lua.LState) int {
			d := playerAs[PlayerDetails](L, "is_in_category", "details")
			L.Push(lua.LBool(d.InCategory(int(checkInt(L, 2, "player:is_in_category")))))
			return 1
		},
		"var": func(L *lua.LState) int {
			pv := playerAs[PlayerVars](L, "var", "variables")
			if s, ok := pv.Var(L.CheckString(2)); ok {
				L.Push(lua.LString(s))
			} else {
				L.Push(L.Get(3))
			}
			return 1
		},
		"var_int": func(L *lua.LState) int {
			pv := playerAs[PlayerVars](L, "var_int", "variables")
			def := optInt(L, 3, "player:var_int", 0)
			s, ok := pv.Var(L.CheckString(2))
			n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
			if !ok || err != nil {
				n = def
			}
			L.Push(lua.LNumber(n))
			return 1
		},
		"var_bool": func(L *lua.LState) int {
			pv := playerAs[PlayerVars](L, "var_bool", "variables")
			def := L.ToBool(3)
			s, ok := pv.Var(L.CheckString(2))
			if !ok {
				L.Push(lua.LBool(def))
			} else {
				L.Push(lua.LBool(strings.EqualFold(s, "true")))
			}
			return 1
		},
		"set_class_id": func(L *lua.LState) int {
			pc := playerAs[PlayerClasses](L, "set_class_id", "class change")
			pc.SetClassID(int(checkInt(L, 2, "player:set_class_id")))
			return 0
		},
		"set_base_class": func(L *lua.LState) int {
			pc := playerAs[PlayerClasses](L, "set_base_class", "class change")
			pc.SetBaseClassID(int(checkInt(L, 2, "player:set_base_class")))
			return 0
		},
		"broadcast_user_info": func(L *lua.LState) int {
			playerAs[PlayerClasses](L, "broadcast_user_info", "class change").BroadcastUserInfo()
			return 0
		},
		"set_var": func(L *lua.LState) int {
			pv := playerAs[PlayerVars](L, "set_var", "variables")
			pv.SetVar(L.CheckString(2), valueString(L, 3, "player:set_var"))
			return 0
		},
		// Parties, clans and subclasses come in later milestones; until then
		// every player is alone, without a clan.
		"is_in_party":         solo("is_in_party", lua.LFalse),
		"party":               solo("party", lua.LNil),
		"clan":                solo("clan", lua.LNil),
		"is_clan_leader":      solo("is_clan_leader", lua.LFalse),
		"is_sub_class_active": solo("is_sub_class_active", lua.LFalse),
		"party_members": func(L *lua.LState) int {
			p := checkPlayer(L, "party_members")
			t := L.CreateTable(1, 0)
			t.RawSetInt(1, v.newPlayer(L, p))
			L.Push(t)
			return 1
		},
		"social_action": func(L *lua.LState) int {
			p := checkPlayer(L, "social_action")
			id := int(checkInt(L, 2, "player:social_action"))
			v.needActions(L, "player:social_action").SocialAction(p, id)
			return 0
		},
		"teleport": func(L *lua.LState) int {
			p := checkPlayer(L, "teleport")
			var loc Location
			if L.Get(2).Type() == lua.LTNumber {
				loc = Location{X: int(checkInt(L, 2, "player:teleport")), Y: int(checkInt(L, 3, "player:teleport")), Z: int(checkInt(L, 4, "player:teleport"))}
			} else {
				loc = locationArg(L, 2, "player:teleport")
			}
			v.needActions(L, "player:teleport").Teleport(p, loc)
			return 0
		},
		"send_message": func(L *lua.LState) int {
			p := checkPlayer(L, "send_message")
			v.needActions(L, "player:send_message").SendMessage(p, L.CheckString(2))
			return 0
		},
		"send_system_message": func(L *lua.LState) int {
			p := checkPlayer(L, "send_system_message")
			id := int(checkInt(L, 2, "player:send_system_message"))
			var params []string
			for i := 3; i <= L.GetTop(); i++ {
				params = append(params, L.ToStringMeta(L.Get(i)).String())
			}
			v.needActions(L, "player:send_system_message").SendSystemMessage(p, id, params)
			return 0
		},
	}
	for name, fn := range locationMethods("player", self) {
		m[name] = fn
	}
	for name, fn := range v.playerAIMethods() {
		m[name] = fn
	}
	return m
}

// npcQuestMethods are the Npc methods quests use.
func (v *vm) npcQuestMethods() map[string]lua.LGFunction {
	self := func(L *lua.LState, method string) any { return checkNpc(L, method) }
	m := map[string]lua.LGFunction{
		"is_dead": func(L *lua.LState) int {
			L.Push(lua.LBool(npcAs[NpcDetails](L, "is_dead", "details").IsDead()))
			return 1
		},
		"get_script_value": func(L *lua.LState) int {
			L.Push(lua.LNumber(npcAs[NpcDetails](L, "get_script_value", "script value").ScriptValue()))
			return 1
		},
		"set_script_value": func(L *lua.LState) int {
			d := npcAs[NpcDetails](L, "set_script_value", "script value")
			d.SetScriptValue(int(checkInt(L, 2, "npc:set_script_value")))
			return 0
		},
		"is_script_value": func(L *lua.LState) int {
			d := npcAs[NpcDetails](L, "is_script_value", "script value")
			L.Push(lua.LBool(int64(d.ScriptValue()) == checkInt(L, 2, "npc:is_script_value")))
			return 1
		},
		"is_type": func(L *lua.LState) int {
			d := npcAs[NpcDetails](L, "is_type", "details")
			L.Push(lua.LBool(d.IsType(L.CheckString(2))))
			return 1
		},
		"delete": func(L *lua.LState) int {
			n := checkNpc(L, "delete")
			v.needActions(L, "npc:delete").DeleteNpc(n)
			return 0
		},
		"broadcast_say": func(L *lua.LState) int {
			n := checkNpc(L, "broadcast_say")
			chat := int(checkInt(L, 2, "npc:broadcast_say"))
			text := L.CheckString(3)
			v.needActions(L, "npc:broadcast_say").NpcBroadcastSay(n, chat, text)
			return 0
		},
		"show_chat_window": func(L *lua.LState) int {
			n := checkNpc(L, "show_chat_window")
			p := checkPlayerArg(L, 2, "npc:show_chat_window")
			v.needActions(L, "npc:show_chat_window").ShowChatWindow(n, p)
			return 0
		},
		"social_action": func(L *lua.LState) int {
			n := checkNpc(L, "social_action")
			id := int(checkInt(L, 2, "npc:social_action"))
			v.needActions(L, "npc:social_action").SocialAction(n, id)
			return 0
		},
	}
	for name, fn := range locationMethods("npc", self) {
		m[name] = fn
	}
	for name, fn := range v.npcAIMethods() {
		m[name] = fn
	}
	return m
}

// newItem, newZone and newSkill return the value types of the handlers.
// They hold copies, so they never expire.
func newValue(L *lua.LState, val any, meta string) lua.LValue {
	ud := L.NewUserData()
	ud.Value = val
	L.SetMetatable(ud, L.GetTypeMetatable(meta))
	return ud
}

func valueOf[T any](L *lua.LState, class, method string) T {
	ud, _ := L.Get(1).(*lua.LUserData)
	var x T
	ok := false
	if ud != nil {
		x, ok = ud.Value.(T)
	}
	if !ok {
		L.RaiseError("%s must be called on a %s: %s:%s()", method, strings.ToLower(class), strings.ToLower(class), method)
	}
	return x
}

func itemMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"id":        func(L *lua.LState) int { L.Push(lua.LNumber(valueOf[Item](L, "Item", "id").ID)); return 1 },
		"object_id": func(L *lua.LState) int { L.Push(lua.LNumber(valueOf[Item](L, "Item", "object_id").ObjectID)); return 1 },
		"count":     func(L *lua.LState) int { L.Push(lua.LNumber(valueOf[Item](L, "Item", "count").Count)); return 1 },
	}
}

func zoneMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"id":   func(L *lua.LState) int { L.Push(lua.LNumber(valueOf[Zone](L, "Zone", "id").ID)); return 1 },
		"type": func(L *lua.LState) int { L.Push(lua.LString(valueOf[Zone](L, "Zone", "type").Type)); return 1 },
	}
}

func skillMethods() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"id":    func(L *lua.LState) int { L.Push(lua.LNumber(valueOf[Skill](L, "Skill", "id").ID)); return 1 },
		"level": func(L *lua.LState) int { L.Push(lua.LNumber(valueOf[Skill](L, "Skill", "level").Level)); return 1 },
	}
}

func itemString(ud *lua.LUserData) string {
	i, _ := ud.Value.(Item)
	return fmt.Sprintf("item %d (object %d, count %d)", i.ID, i.ObjectID, i.Count)
}

func zoneString(ud *lua.LUserData) string {
	z, _ := ud.Value.(Zone)
	return fmt.Sprintf("zone %d (%s)", z.ID, z.Type)
}

func skillString(ud *lua.LUserData) string {
	s, _ := ud.Value.(Skill)
	return fmt.Sprintf("skill %d level %d", s.ID, s.Level)
}
