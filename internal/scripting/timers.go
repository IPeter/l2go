package scripting

import (
	"context"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// timerKey identifies a quest timer, as Mobius QuestTimer.equals does: the
// script, the name, and the NPC and player object IDs (0 for nil, so nil
// matches only nil).
type timerKey struct {
	script string
	name   string
	npc    int32
	player int32
}

// questTimer is a running quest timer. It fires on_event(name, npc,
// player) of its script.
type questTimer struct {
	key    timerKey
	script *npcScript
	npc    NpcRef
	player PlayerRef
	repeat bool
	every  time.Duration
	seq    uint64 // start order
	stop   func()
}

func newTimerKey(s *npcScript, name string, npc NpcRef, p PlayerRef) timerKey {
	k := timerKey{script: s.name, name: name}
	if npc != nil {
		k.npc = npc.ObjectID()
	}
	if p != nil {
		k.player = p.ObjectID()
	}
	return k
}

// qStartQuestTimer is q:start_quest_timer(name, ms, npc, player[,
// repeating]). A timer with the same key already running is left alone.
func (v *vm) qStartQuestTimer(L *lua.LState) int {
	const fn = "q:start_quest_timer"
	s := checkScript(L, "start_quest_timer")
	if L.Get(2) == lua.LNil {
		return 0
	}
	name := L.CheckString(2)
	ms := checkInt(L, 3, fn)
	npc := optNpcArg(L, 4, fn)
	p := optPlayerArg(L, 5, fn)
	repeat := L.ToBool(6)
	w := v.needWorld(L, fn)
	key := newTimerKey(s, name, npc, p)
	if _, ok := v.timers[key]; ok {
		return 0
	}
	d := max(time.Duration(ms)*time.Millisecond, 0)
	v.timerSeq++
	t := &questTimer{key: key, script: s, npc: npc, player: p, repeat: repeat, seq: v.timerSeq}
	fire := func() { v.timerFired(t) }
	if repeat {
		t.every = d
		t.stop = w.Every(d, fire)
	} else {
		t.stop = w.After(d, fire)
	}
	v.timers[key] = t
	return 0
}

// qGetQuestTimer is q:get_quest_timer(name, npc, player): the timer, or
// nil.
func (v *vm) qGetQuestTimer(L *lua.LState) int {
	const fn = "q:get_quest_timer"
	s := checkScript(L, "get_quest_timer")
	if L.Get(2) == lua.LNil {
		L.Push(lua.LNil)
		return 1
	}
	key := newTimerKey(s, L.CheckString(2), optNpcArg(L, 3, fn), optPlayerArg(L, 4, fn))
	t, ok := v.timers[key]
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	ud := L.NewUserData()
	ud.Value = &timerBox{t: t}
	L.SetMetatable(ud, L.GetTypeMetatable(timerType))
	L.Push(ud)
	return 1
}

// qCancelQuestTimer is q:cancel_quest_timer(name, npc, player).
func (v *vm) qCancelQuestTimer(L *lua.LState) int {
	const fn = "q:cancel_quest_timer"
	s := checkScript(L, "cancel_quest_timer")
	if L.Get(2) == lua.LNil {
		return 0
	}
	key := newTimerKey(s, L.CheckString(2), optNpcArg(L, 3, fn), optPlayerArg(L, 4, fn))
	if t, ok := v.timers[key]; ok {
		v.cancelTimer(t)
	}
	return 0
}

// qCancelQuestTimers is q:cancel_quest_timers(name): every timer of the
// script with that name.
func (v *vm) qCancelQuestTimers(L *lua.LState) int {
	s := checkScript(L, "cancel_quest_timers")
	if L.Get(2) == lua.LNil {
		return 0
	}
	name := L.CheckString(2)
	v.cancelTimersWhere(func(k timerKey) bool { return k.script == s.name && k.name == name })
	return 0
}

// cancelTimer stops t and forgets it.
func (v *vm) cancelTimer(t *questTimer) {
	if v.timers[t.key] != t {
		return
	}
	delete(v.timers, t.key)
	if t.stop != nil {
		t.stop()
	}
}

// cancelTimersWhere cancels the timers whose key matches.
func (v *vm) cancelTimersWhere(match func(timerKey) bool) {
	for k, t := range v.timers {
		if match(k) {
			v.cancelTimer(t)
		}
	}
}

// stopTimers cancels every timer, on the loop (reload).
func (v *vm) stopTimers() {
	v.cancelTimersWhere(func(timerKey) bool { return true })
}

// dropTimers forgets every timer without touching the loop (Close).
func (v *vm) dropTimers() {
	clear(v.timers)
}

// timerFired runs a timer that the world fired. A one-shot timer is
// removed before its handler runs, so the handler may start it again.
func (v *vm) timerFired(t *questTimer) {
	e := v.engine
	if e == nil || e.closed || e.vm != v || v.timers[t.key] != t {
		return
	}
	if !t.repeat {
		delete(v.timers, t.key)
	}
	reg, ok := t.script.handlerFor(hookEvent, 0)
	if !ok {
		return
	}
	ctx := context.Background()
	defer e.settle()
	reply, ok := v.npcCall(ctx, e.opts.Timeout, t.script, reg.handler, "on_event", func() []lua.LValue {
		return []lua.LValue{lua.LString(t.key.name), v.newNpc(v.L, t.npc), v.newPlayer(v.L, t.player)}
	})
	if (ok || reply.Err != nil) && t.player != nil && v.quest.world != nil {
		v.quest.world.ShowReply(t.player, t.npc, reply)
	}
}

// PlayerLeft cancels the quest timers of a player who left the world. The
// game calls it on the loop.
func (e *Engine) PlayerLeft(objectID int32) {
	if e.closed {
		return
	}
	e.vm.cancelTimersWhere(func(k timerKey) bool { return k.player == objectID })
}

// NpcGone tells the engine that an NPC decayed or was deleted: its quest
// timers stop, its variables go (Mobius clears them on respawn) and the
// scripts' handles of it become invalid (npc:valid() is false). The game
// calls it on the loop.
func (e *Engine) NpcGone(objectID int32) {
	if e.closed {
		return
	}
	e.vm.cancelTimersWhere(func(k timerKey) bool { return k.npc == objectID })
	delete(e.npcVars, objectID)
	e.vm.npcGone(objectID)
}

// timerBox is the Go value inside a QuestTimer userdata.
type timerBox struct{ t *questTimer }

// timerMethods are the methods of QuestTimer values.
func (v *vm) timerMethods() map[string]lua.LGFunction {
	check := func(L *lua.LState, method string) *questTimer {
		ud, _ := L.Get(1).(*lua.LUserData)
		var b *timerBox
		if ud != nil {
			b, _ = ud.Value.(*timerBox)
		}
		if b == nil {
			L.RaiseError("%s must be called on a quest timer: timer:%s()", method, method)
			return nil
		}
		return b.t
	}
	return map[string]lua.LGFunction{
		"name": func(L *lua.LState) int {
			L.Push(lua.LString(check(L, "name").key.name))
			return 1
		},
		"is_repeating": func(L *lua.LState) int {
			L.Push(lua.LBool(check(L, "is_repeating").repeat))
			return 1
		},
		"is_active": func(L *lua.LState) int {
			t := check(L, "is_active")
			L.Push(lua.LBool(v.timers[t.key] == t))
			return 1
		},
		"cancel": func(L *lua.LState) int {
			v.cancelTimer(check(L, "cancel"))
			return 0
		},
	}
}

func timerString(ud *lua.LUserData) string {
	if b, ok := ud.Value.(*timerBox); ok {
		return fmt.Sprintf("quest timer %s of %s", b.t.key.name, b.t.key.script)
	}
	return "quest timer"
}
