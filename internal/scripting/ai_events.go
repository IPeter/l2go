package scripting

import (
	"context"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// The AI events (Plan 13, A7), named after Mobius's Quest callbacks. All
// but NpcHate are asynchronous in Mobius (notifyEventAsync): raised while
// a handler runs (npc:broadcast_event, or a game call a handler caused),
// they wait until the outermost run ends and then run in order, each with
// its own time limit. The game calls them on the loop, and may post them
// after the current command itself, as it does for attacks.

// Listens reports whether a script handles the event for the NPC template
// (or zone) id, so the game can skip the work of raising it (Mobius
// hasListener). event is the handler's name, with or without "on_", e.g.
// "move_finished". Events without IDs (day_night_change) ignore id.
func (e *Engine) Listens(event string, id int) bool {
	k, ok := hookByName(event)
	if !ok || e.closed {
		return false
	}
	if !hookSpecs[k].ids {
		for _, s := range e.vm.order {
			if _, ok := s.handlerFor(k, 0); ok {
				return true
			}
		}
		return false
	}
	return e.listens(k, id)
}

// hookByName returns the hook kind of an on_<name> handler.
func hookByName(name string) (hookKind, bool) {
	name = strings.TrimPrefix(name, "on_")
	for k, spec := range hookSpecs {
		if spec.name == name {
			return hookKind(k), true
		}
	}
	return 0, false
}

// FactionCall runs on_faction_call(npc, caller, attacker, is_summon): npc
// was called to help caller against attacker (Mobius onFactionCall).
func (e *Engine) FactionCall(ctx context.Context, npc, caller NpcRef, attacker PlayerRef, isSummon bool) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notifyAsync(ctx, hookFactionCall, npc.ID(), func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.newNpc(v.L, caller), v.newPlayer(v.L, attacker), lua.LBool(isSummon)}
	})
}

// CreatureSee runs on_creature_see(npc, creature): npc saw a player or an
// NPC come into range (Mobius onCreatureSee).
func (e *Engine) CreatureSee(ctx context.Context, npc NpcRef, creature any) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notifyAsync(ctx, hookCreatureSee, npc.ID(), func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.objectValue(v.L, creature)}
	})
}

// EventReceived runs on_event_received(event, sender, receiver,
// reference) of the scripts registered for the receiver (Mobius
// onEventReceived, from broadcastEvent and sendScriptEvent). reference is
// a PlayerRef, an NpcRef or nil.
func (e *Engine) EventReceived(ctx context.Context, event string, sender, receiver NpcRef, reference any) error {
	if receiver == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notifyAsync(ctx, hookEventReceived, receiver.ID(), func(v *vm) []lua.LValue {
		return []lua.LValue{lua.LString(event), v.newNpc(v.L, sender), v.newNpc(v.L, receiver), v.objectValue(v.L, reference)}
	})
}

// MoveFinished runs on_move_finished(npc): the NPC arrived where it was
// sent (Mobius onMoveFinished, from CreatureAI.onEvtArrived).
func (e *Engine) MoveFinished(ctx context.Context, npc NpcRef) error {
	return e.npcEvent(ctx, hookMoveFinished, npc)
}

// NodeArrived runs on_node_arrived(npc): a walker reached a route node.
func (e *Engine) NodeArrived(ctx context.Context, npc NpcRef) error {
	return e.npcEvent(ctx, hookNodeArrived, npc)
}

// RouteFinished runs on_route_finished(npc): a walker reached the last
// node of its route.
func (e *Engine) RouteFinished(ctx context.Context, npc NpcRef) error {
	return e.npcEvent(ctx, hookRouteFinished, npc)
}

// Teleported runs on_teleport(npc): the NPC was teleported (Mobius
// onTeleport).
func (e *Engine) Teleported(ctx context.Context, npc NpcRef) error {
	return e.npcEvent(ctx, hookTeleport, npc)
}

func (e *Engine) npcEvent(ctx context.Context, k hookKind, npc NpcRef) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notifyAsync(ctx, k, npc.ID(), func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc)}
	})
}

// DayNightChange runs on_day_night_change(is_night) of every script that
// has one (Mobius OnDayNightChange, a global event).
func (e *Engine) DayNightChange(ctx context.Context, isNight bool) error {
	if e.closed {
		return ErrStopped
	}
	return e.notifyAsync(ctx, hookDayNightChange, 0, func(*vm) []lua.LValue {
		return []lua.LValue{lua.LBool(isNight)}
	})
}

// NpcHate runs on_npc_hate(npc, player, is_summon) at once, also inside
// another run (Mobius notifies it synchronously from AttackableAI): it
// decides whether the monster may hate the player. The first handler that
// returns false vetoes, and the rest do not run (Mobius TerminateReturn).
// A handler that fails or returns anything else allows.
func (e *Engine) NpcHate(ctx context.Context, npc NpcRef, player PlayerRef, isSummon bool) bool {
	if npc == nil || e.stopped(ctx) != nil {
		return true
	}
	scripts := e.vm.byID[hookNpcHate][npc.ID()]
	if len(scripts) == 0 {
		return true
	}
	defer e.settle()
	v := e.vm
	for _, s := range append([]*npcScript(nil), scripts...) {
		reg, ok := s.handlerFor(hookNpcHate, npc.ID())
		if !ok {
			continue
		}
		ret, err := v.handlerCall(ctx, e.opts.Timeout, s, reg.handler, "on_npc_hate", func() []lua.LValue {
			return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, player), lua.LBool(isSummon)}
		})
		if err == nil && ret == lua.LFalse {
			return false
		}
		if ctx.Err() != nil {
			break
		}
	}
	return true
}
