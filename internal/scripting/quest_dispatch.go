package scripting

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// QuestInfo describes a script made by quest.new (or script.new), for the
// game's quest window, quest list and GM commands.
type QuestInfo struct {
	// ID is the quest ID; -1 for a plain script.
	ID          int
	Name        string
	Description string
	// Custom quests get no quest mark and cannot be started with
	// start_quest (Mobius isCustomQuest).
	Custom bool
	// OnEnterWorld: the script's on_enter_world runs when a player enters
	// the world.
	OnEnterWorld bool
	// Requires lists the features the quest needs (q:requires); until they
	// exist the quest is not offered.
	Requires []string
	// StartNpcs offer the quest; TalkNpcs reach its talk handler (the start
	// NPCs included).
	StartNpcs []int
	TalkNpcs  []int
	// QuestItems are taken when the quest ends or is aborted.
	QuestItems []int
	// Dir is the folder of the script's file; RelDir the same relative to
	// the scripts folder, slash-separated (for data/html/scripts/<RelDir>).
	Dir    string
	RelDir string
}

func (s *npcScript) info() QuestInfo {
	return QuestInfo{
		ID:           s.id,
		Name:         s.name,
		Description:  s.description,
		Custom:       s.custom,
		OnEnterWorld: s.onEnterWorld,
		Requires:     slices.Clone(s.requires),
		StartNpcs:    slices.Clone(s.startNpcs.order),
		TalkNpcs:     slices.Clone(s.hooks[hookTalk].ids.order),
		QuestItems:   slices.Clone(s.questItems),
		Dir:          s.dir,
		RelDir:       s.rel,
	}
}

// Quests returns the quests (ID ≥ 1), by ID.
func (e *Engine) Quests() []QuestInfo {
	if e.closed {
		return nil
	}
	var out []QuestInfo
	for _, s := range e.vm.order {
		if s.isQuest() {
			out = append(out, s.info())
		}
	}
	slices.SortFunc(out, func(a, b QuestInfo) int { return a.ID - b.ID })
	return out
}

// Quest returns the script or quest with this name.
func (e *Engine) Quest(name string) (QuestInfo, bool) {
	if e.closed {
		return QuestInfo{}, false
	}
	s, ok := e.vm.scripts[name]
	if !ok {
		return QuestInfo{}, false
	}
	return s.info(), true
}

// QuestByID returns the quest with this ID.
func (e *Engine) QuestByID(id int) (QuestInfo, bool) {
	if e.closed {
		return QuestInfo{}, false
	}
	s, ok := e.vm.questIDs[id]
	if !ok {
		return QuestInfo{}, false
	}
	return s.info(), true
}

// StartQuests returns the names of the quests the NPC template offers, in
// load order.
func (e *Engine) StartQuests(npcID int) []string {
	if e.closed {
		return nil
	}
	var names []string
	for _, s := range e.vm.byID[hookTalk][npcID] {
		if s.startNpcs.has[npcID] {
			names = append(names, s.name)
		}
	}
	return names
}

// HasKill reports whether a script handles kills of the NPC template, so
// the game can skip scheduling the kill event otherwise.
func (e *Engine) HasKill(npcID int) bool {
	return e.listens(hookKill, npcID)
}

// HasAttack reports whether a script handles attacks on the NPC template.
func (e *Engine) HasAttack(npcID int) bool {
	return e.listens(hookAttack, npcID)
}

func (e *Engine) listens(k hookKind, id int) bool {
	if e.closed {
		return false
	}
	for _, s := range e.vm.byID[k][id] {
		if _, ok := s.handlerFor(k, id); ok {
			return true
		}
	}
	return false
}

// call runs one handler of s that returns nothing the game uses, in its
// own run, and returns its first result.
func (v *vm) handlerCall(ctx context.Context, timeout time.Duration, s *npcScript, h handler, what string, args func() []lua.LValue) (lua.LValue, error) {
	ctx, end := v.begin(ctx, timeout)
	defer end()
	prev := v.event
	v.event = nil
	defer func() { v.event = prev }()
	ret, _, err := v.callWith(h.fn, args)
	if err != nil {
		herr := handlerError(ctx, err, h, s.name+" "+what+" handler", timeout)
		v.logger.Warn("script handler failed", "script", s.name, "handler", h.where(), "err", herr)
		return lua.LNil, herr
	}
	return ret, nil
}

// notify runs the handler of kind k of every script in scripts that has
// one for id, in order. Failures are logged and joined.
func (e *Engine) notify(ctx context.Context, k hookKind, id int, scripts []*npcScript, args func(v *vm) []lua.LValue) error {
	if err := e.stopped(ctx); err != nil {
		return err
	}
	if len(scripts) == 0 {
		return nil
	}
	defer e.settle()
	v := e.vm
	var errs []error
	for _, s := range slices.Clone(scripts) {
		reg, ok := s.handlerFor(k, id)
		if !ok {
			continue
		}
		if _, err := v.handlerCall(ctx, e.opts.Timeout, s, reg.handler, "on_"+hookSpecs[k].name, func() []lua.LValue { return args(v) }); err != nil {
			errs = append(errs, err)
		}
		if ctx.Err() != nil {
			break
		}
	}
	return errors.Join(errs...)
}

// Start runs the on_start handlers, in load order: spawns, timers and
// global variables that Mobius scripts set up in their constructors. The
// game calls it on the loop once the world is ready; Swap calls it after a
// reload.
func (e *Engine) Start(ctx context.Context) error {
	if e.closed {
		return ErrStopped
	}
	return e.notify(ctx, hookStart, 0, e.vm.order, func(*vm) []lua.LValue { return nil })
}

// Kill runs on_kill(npc, killer, is_summon) of the scripts registered for
// the NPC's template. The game calls it 2.5 s after the death (Mobius
// onKillDelay), with the acting player, unless the killer left the world.
func (e *Engine) Kill(ctx context.Context, npc NpcRef, killer PlayerRef, isSummon bool) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notify(ctx, hookKill, npc.ID(), e.vm.byID[hookKill][npc.ID()], func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, killer), lua.LBool(isSummon)}
	})
}

// Attack runs on_attack(npc, attacker, damage, is_summon, skill); skill is
// nil for a plain hit.
func (e *Engine) Attack(ctx context.Context, npc NpcRef, attacker PlayerRef, damage int, isSummon bool, skill *Skill) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notify(ctx, hookAttack, npc.ID(), e.vm.byID[hookAttack][npc.ID()], func(v *vm) []lua.LValue {
		sk := lua.LValue(lua.LNil)
		if skill != nil {
			sk = newValue(v.L, *skill, skillType)
		}
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, attacker), lua.LNumber(damage), lua.LBool(isSummon), sk}
	})
}

// Spawn runs on_spawn(npc), also for respawns.
func (e *Engine) Spawn(ctx context.Context, npc NpcRef) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notify(ctx, hookSpawn, npc.ID(), e.vm.byID[hookSpawn][npc.ID()], func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc)}
	})
}

// SkillSee runs on_skill_see(npc, caster, skill, targets, is_summon);
// targets are PlayerRef and NpcRef values.
func (e *Engine) SkillSee(ctx context.Context, npc NpcRef, caster PlayerRef, skill Skill, targets []any, isSummon bool) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notify(ctx, hookSkillSee, npc.ID(), e.vm.byID[hookSkillSee][npc.ID()], func(v *vm) []lua.LValue {
		t := v.L.CreateTable(len(targets), 0)
		for _, obj := range targets {
			if val := v.objectValue(v.L, obj); val != lua.LNil {
				t.Append(val)
			}
		}
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, caster), newValue(v.L, skill, skillType), t, lua.LBool(isSummon)}
	})
}

// SpellFinished runs on_spell_finished(npc, player, skill).
func (e *Engine) SpellFinished(ctx context.Context, npc NpcRef, player PlayerRef, skill Skill) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notify(ctx, hookSpellFinished, npc.ID(), e.vm.byID[hookSpellFinished][npc.ID()], func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, player), newValue(v.L, skill, skillType)}
	})
}

// AggroRangeEnter runs on_aggro_range_enter(npc, player, is_summon).
func (e *Engine) AggroRangeEnter(ctx context.Context, npc NpcRef, player PlayerRef, isSummon bool) error {
	if npc == nil || e.closed {
		return e.stopped(ctx)
	}
	return e.notify(ctx, hookAggroRangeEnter, npc.ID(), e.vm.byID[hookAggroRangeEnter][npc.ID()], func(v *vm) []lua.LValue {
		return []lua.LValue{v.newNpc(v.L, npc), v.newPlayer(v.L, player), lua.LBool(isSummon)}
	})
}

// EnterZone runs on_enter_zone(creature, zone); creature is a PlayerRef or
// an NpcRef.
func (e *Engine) EnterZone(ctx context.Context, creature any, zone Zone) error {
	return e.zone(ctx, hookEnterZone, creature, zone)
}

// ExitZone runs on_exit_zone(creature, zone).
func (e *Engine) ExitZone(ctx context.Context, creature any, zone Zone) error {
	return e.zone(ctx, hookExitZone, creature, zone)
}

func (e *Engine) zone(ctx context.Context, k hookKind, creature any, zone Zone) error {
	if e.closed {
		return ErrStopped
	}
	return e.notify(ctx, k, zone.ID, e.vm.byID[k][zone.ID], func(v *vm) []lua.LValue {
		return []lua.LValue{v.objectValue(v.L, creature), newValue(v.L, zone, zoneType)}
	})
}

// EnterWorld runs on_enter_world(player) of the scripts that asked for it
// with q:set_on_enter_world(true).
func (e *Engine) EnterWorld(ctx context.Context, player PlayerRef) error {
	if e.closed {
		return ErrStopped
	}
	var scripts []*npcScript
	for _, s := range e.vm.order {
		if s.onEnterWorld {
			scripts = append(scripts, s)
		}
	}
	return e.notify(ctx, hookEnterWorld, 0, scripts, func(v *vm) []lua.LValue {
		return []lua.LValue{v.newPlayer(v.L, player)}
	})
}

// Death runs on_death(killer, victim, st) of script, for a player whose
// quest state asked to be told of its death. killer is a PlayerRef, an
// NpcRef or nil. Without a death handler it runs on_event("", npc,
// player) as Mobius does, npc being the killer when it is an NPC; that
// reply is dropped, as Mobius drops it.
func (e *Engine) Death(ctx context.Context, script string, killer any, victim PlayerRef) error {
	if err := e.stopped(ctx); err != nil {
		return err
	}
	s, ok := e.vm.scripts[script]
	if !ok || victim == nil {
		return nil
	}
	v := e.vm
	if reg, ok := s.handlerFor(hookDeath, 0); ok {
		defer e.settle()
		_, err := v.handlerCall(ctx, e.opts.Timeout, s, reg.handler, "on_death", func() []lua.LValue {
			st := lua.LNil
			if v.quest.world != nil {
				st = v.newQuestState(v.L, victim, v.quest.world.QuestState(victim, s.name))
			}
			return []lua.LValue{v.objectValue(v.L, killer), v.newPlayer(v.L, victim), st}
		})
		return err
	}
	npc, _ := killer.(NpcRef)
	reply, _ := e.Event(ctx, script, "", victim, npc)
	return reply.Err
}

// ItemTalk runs on_item_talk(item, player) of the scripts registered for
// the item's template and returns the replies, in load order.
func (e *Engine) ItemTalk(ctx context.Context, item Item, player PlayerRef) []Reply {
	if err := e.stopped(ctx); err != nil {
		return []Reply{{Err: err}}
	}
	scripts := slices.Clone(e.vm.byID[hookItemTalk][item.ID])
	if len(scripts) == 0 {
		return nil
	}
	defer e.settle()
	v := e.vm
	var replies []Reply
	for _, s := range scripts {
		reg, ok := s.handlerFor(hookItemTalk, item.ID)
		if !ok {
			continue
		}
		reply, ok := v.npcCall(ctx, e.opts.Timeout, s, reg.handler, "on_item_talk", func() []lua.LValue {
			return []lua.LValue{newValue(v.L, item, itemType), v.newPlayer(v.L, player)}
		})
		if ok || reply.Err != nil {
			replies = append(replies, reply)
		}
	}
	return replies
}

// String names a QuestInfo in logs.
func (q QuestInfo) String() string {
	return fmt.Sprintf("%s (%d)", q.Name, q.ID)
}
