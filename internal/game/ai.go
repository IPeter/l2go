package game

import (
	"math"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/npcai"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/world"
)

// The monster AI (Plan 09 decisions F5, F6 and F11): Mobius AttackableAI,
// CreatureAI and AbstractAI for attackables, without skills, minions and
// raids.
//
//   - ai.go: the AI state, the intentions, the events and the think
//     scheduler (Mobius AttackableThinkTaskManager).
//   - ai_think.go: the ACTIVE and ATTACK thinks, aggression and clan help.
//   - hate.go: the hate and attack-by lists.
//   - npc_move.go: NPC movement.
//
// An AI is attached (Mobius hasAI) once something asks for it: a player
// sees the monster, it is attacked or called to help. Attaching rolls the
// post-spawn aggression counter. IDLE detaches it again (setIntentionIdle),
// which only happens when no player sees the NPC and it is home.
//
// The attack feature (attack*.go) swings for the AI (DoAttack) and keeps
// its attack stance (StartAutoAttack, StopAutoAttack, InAttackStance); its
// OnNpcAttackEvent and OnAttacked hooks tell the AI about swings and hits.
//
// The API for the other features:
//
//   - Npc.ReduceHP tells the AI of an attackable about the attack and adds
//     the hate before the damage (Mobius Attackable.reduceCurrentHp and
//     addDamage); the attack-by list gets the attacker.
//   - Server.NotifyAttacked after a hit landed (Mobius onHitTimer's
//     notifyActionAttacked; the OnAttacked hook does it).
//   - Server.NotifyReadyToAct when an NPC may act again.
//   - Server.NotifyAggression for scripts and calls for help.
//   - Hate: Npc.Hating, MostHated, AddDamageHate, StopHating,
//     ClearAggroList, Aggro, AttackBy, AddAttackBy.
//   - Movement: Server.NpcMoveTo, NpcMoveToPawn, NpcStopMove, SetNpcRunning,
//     TeleportNpc; Npc.Intention, Npc.Running.
func init() { registerFeature("ai", setupAI) }

// NpcIntention is what an NPC's AI is doing (Mobius Intention), as far as
// the monster AI uses it.
type NpcIntention uint8

// The intentions.
const (
	// NpcIdle: the AI does not think (Mobius IDLE).
	NpcIdle NpcIntention = iota
	// NpcActive: the AI looks for enemies and wanders (ACTIVE).
	NpcActive
	// NpcAttack: the AI fights its most hated enemy (ATTACK).
	NpcAttack
	// NpcMoveTo: the NPC walks somewhere, home for example (MOVE_TO).
	NpcMoveTo
	// NpcCast: the NPC walks up to a target and casts a skill at it
	// (CAST, ai_cast.go).
	NpcCast
)

// String returns Mobius's name of the intention.
func (i NpcIntention) String() string {
	switch i {
	case NpcIdle:
		return "IDLE"
	case NpcActive:
		return "ACTIVE"
	case NpcAttack:
		return "ATTACK"
	case NpcMoveTo:
		return "MOVE_TO"
	case NpcCast:
		return "CAST"
	}
	return "?"
}

// Constants of Mobius AttackableAI and Attackable.
const (
	// aiThinkSlots spreads the thinkers over the ten 100 ms ticks of a
	// second (decision F5): each thinks once a second, in its slot.
	aiThinkSlots = 10
	// aiMaxAttackTimeout is how long, in game ticks, a monster fights
	// without swinging or being hit before it gives up (2 min).
	aiMaxAttackTimeout = 1200
	// aiNoTimeout is Mobius's Integer.MAX_VALUE attack timeout.
	aiNoTimeout = math.MaxInt32
	// aiRandomWalkRate: a monster wanders on 1 think in 30.
	aiRandomWalkRate = 30
	// aiForgetRate: a monster at full HP and MP forgets its attackers on 1
	// think in 500.
	aiForgetRate = 500
	// aiWanderAttempts is how many points a territory monster tries.
	aiWanderAttempts = 3
	// aiClanHelpMinAge: NPCs spawned less than 7 s ago do not help.
	aiClanHelpMinAge = 7 * time.Second
	// aiClanHelpMaxDZ: NPCs more than 600 above or below the attacker do not
	// help.
	aiClanHelpMaxDZ = 600
	// aiUnseenChaseRange: a monster walks towards a target it cannot see
	// within this 3D range.
	aiUnseenChaseRange = 6000
	// aiArcherRange is the bow range of NPC archers.
	aiArcherRange = 850
	// aiArcherNear and aiArcherStep: an archer steps 300 back from a target
	// within 60 (plus the collisions) on 15 % of its thinks.
	aiArcherNear, aiArcherStep, aiArcherChance = 60, 300, 15
	// aiCrowdChance: on 4 thinks in 100 (Rnd.get(100) <= 3) a monster
	// crowded by another one steps around its target.
	aiCrowdChance = 3
)

// npcAI is an NPC's AI state (Mobius AttackableAI and the parts of
// Attackable and Creature it uses). The zero value is a detached AI.
type npcAI struct {
	cls *aiClass

	// attached is Mobius hasAI.
	attached  bool
	intention NpcIntention
	// thinking blocks a think inside a think (Mobius _thinking).
	thinking bool
	// listed is set while the NPC is in its think slot.
	listed        bool
	slot, slotIdx int

	// globalAggro counts the thinks after a spawn up to 0; the monster is
	// not aggressive before (Mobius _globalAggro).
	globalAggro int
	// attackTimeout is the game tick an attack ends without action.
	attackTimeout int64
	// target is the AI's target (Mobius AbstractAI._target): the pawn of a
	// moveToPawn, the most hated enemy.
	target world.Object
	// attackTarget is the enemy of the ATTACK intention.
	attackTarget Creature
	// pawnOffset and pawnTimeout are Mobius _clientMovingToPawnOffset and
	// _moveToPawnTimeout.
	pawnOffset  int32
	pawnTimeout int64

	// hate and attackBy are the hate and attack-by lists (hate.go).
	hate     []AggroInfo
	attackBy []Creature
	// spawnTime is when the NPC last spawned (Mobius getSpawnTime).
	spawnTime time.Time
	// aggroRange is the spawn's aggroRange override, when aggroSet.
	aggroRange int
	aggroSet   bool
	// home is the spawn point of an NPC no spawn slot manages.
	home    world.Position
	hasHome bool

	// walk follows the walk in Npc.Move; moving is set while the NPC is in
	// the movement list (npc_move.go).
	walk   walkTrack
	moving bool
	// seen is set while the NPC waits to wake up (npcSeen).
	seen bool

	// x is the state of the full AI: skills, fear, CAST, sight (ai_npcai.go).
	x npcAIx
}

// aiClass is what the AI needs from a template, worked out once.
type aiClass struct {
	monster, guard, friendly, attackable bool
	// defender is a siege guard (Mobius Defender with SiegeGuardAI,
	// siege_ai.go).
	defender bool
	// aggressive is Monster.isAggressive: the template's flag, for monsters.
	aggressive bool
	// aggroRange is the template's, capped for the Monster type.
	aggroRange    int
	clanHelpRange int
	clans         []string
	ignoreIDs     []int
	archer        bool
	randomWalk    bool
	// aggressiveTime is the SetAggressiveTime parameter, or -1.
	aggressiveTime int

	hateGroup           data.CategoryType
	hateGroupOK         bool
	hateGroupRatio      int
	hateOccupation      int
	hateOccupationRatio int
	hateRace            data.Race
	hateRaceOK          bool
	hateRaceRatio       int

	// sk is the full AI's part: the skill profile and the rank (ai_npcai.go).
	sk aiSkillClass
}

// aiFeature is the AI's state on the Server (Server.ai). Loop-only.
type aiFeature struct {
	classes map[int]*aiClass
	// doAttack replaces the swing (DoAttack) in tests.
	doAttack func(n *Npc, target Creature)

	// slots holds the thinking NPCs by their slot; tick counts the ticks.
	slots    [aiThinkSlots][]*Npc
	nextSlot int
	tick     uint64
	batch    []*Npc

	// moving lists the walking NPCs.
	moving, moveBatch []*Npc

	// seen holds the monsters players started to see during a visibility
	// change; waking is set while they wake.
	seen, seenSpare []*Npc
	waking          bool

	// scan is a reusable list of the creatures a think looks at.
	scan []Creature

	// x is the state of the full AI (ai_npcai.go).
	x aiFeatureX
}

// setupAI registers the AI's ticks and hooks.
func setupAI(s *Server) error {
	s.ai.classes = make(map[int]*aiClass)
	s.loop.OnTick(s.aiTick)
	s.loop.OnTick(s.npcMoveTick)
	s.world.OnRegionActivation(s.aiRegionSwitched)
	s.OnForget(s.aiForget)
	// The NPC kill sequence (npc_death.go) calls npcDeathBeforeDie and
	// npcDeathAfterDie around the rewards and Die, in Mobius doDie's order.
	s.OnNpcAttackEvent(s.aiAttackEvent)
	s.OnAttacked(func(target, attacker Creature, _ int) { s.NotifyAttacked(target, attacker) })
	return nil
}

// Intention returns what the NPC's AI is doing. Loop-only.
func (n *Npc) Intention() NpcIntention { return n.ai.intention }

// npcRand gives the NPC code's math/rand-style calls the server's random
// source.
type npcRand struct{ rnd.Source }

// IntN is Rnd.get(n).
func (r npcRand) IntN(n int) int { return r.Get(n) }

// aiClassOf returns the AI class of n's template.
func (s *Server) aiClassOf(n *Npc) *aiClass {
	if c := n.ai.cls; c != nil {
		return c
	}
	t := n.Template
	c := s.ai.classes[t.ID]
	if c == nil {
		c = &aiClass{
			clanHelpRange:  t.AI.ClanHelpRange,
			clans:          t.AI.Clans,
			ignoreIDs:      t.AI.IgnoreNpcIDs,
			archer:         t.AI.Type == "ARCHER",
			randomWalk:     t.Status.RandomWalk && !s.walkerTargeted(t.ID), // Mobius Npc.onSpawn
			aggroRange:     t.AI.AggroRange,
			aggressiveTime: npcParamInt(t.Parameters.Params, "SetAggressiveTime", -1),
		}
		if nc := n.class; nc != nil {
			c.monster, c.friendly, c.attackable = nc.monster, nc.friendly, nc.attackable
		}
		if s.worldData != nil {
			c.guard = s.worldData.NpcTypeIs(t.Type, "Guard")
			c.defender = s.worldData.NpcTypeIs(t.Type, "Defender")
		}
		if c.defender {
			c.randomWalk = false // Mobius Defender: no random animation or walk
		}
		c.aggressive = c.monster && t.AI.IsAggressive
		if t.Type == "Monster" && c.aggroRange > s.settings.NPCs.MaxAggroRange {
			c.aggroRange = s.settings.NPCs.MaxAggroRange
		}
		if c.guard || (n.class != nil && n.class.raid) {
			c.randomWalk = false // Mobius Guard, RaidBoss and GrandBoss onSpawn
		}
		c.hateParams(t)
		c.sk = s.aiSkillClassOf(t)
		if s.ai.classes == nil {
			s.ai.classes = make(map[int]*aiClass)
		}
		s.ai.classes[t.ID] = c
	}
	n.ai.cls = c
	return c
}

// npcParamInt reads an int AI parameter, def when it is missing or no
// number (Mobius StatSet.getInt).
func npcParamInt(p map[string]string, name string, def int) int {
	v, ok := p[name]
	if !ok {
		return def
	}
	i := 0
	neg := false
	for j, r := range v {
		switch {
		case j == 0 && r == '-':
			neg = true
		case r >= '0' && r <= '9':
			i = i*10 + int(r-'0')
		default:
			return def
		}
	}
	if neg {
		return -i
	}
	return i
}

// aiOf returns n's AI, attaching a new one when it has none (Mobius
// getAI): a new AI is IDLE, has no attack timeout and rolls its post-spawn
// aggression counter.
func (s *Server) aiOf(n *Npc) *npcAI {
	ai := &n.ai
	if !ai.attached {
		s.aiClassOf(n)
		ai.attached = true
		ai.intention = NpcIdle
		ai.thinking = false
		ai.attackTimeout = aiNoTimeout
		ai.target, ai.attackTarget = nil, nil
		ai.pawnOffset, ai.pawnTimeout = 0, 0
		ai.x.brain = npcai.State{} // a new AttackableAI
		ai.x.castSkill, ai.x.castTarget = nil, nil
		s.rollGlobalAggro(n)
	}
	return ai
}

// rollGlobalAggro is Mobius rollGlobalAggro: 0 with SetAggressiveTime 0,
// one past that many thinks with a positive one, otherwise 4 to 8 thinks.
func (s *Server) rollGlobalAggro(n *Npc) {
	switch t := s.aiClassOf(n).aggressiveTime; {
	case t == 0:
		n.ai.globalAggro = 0
	case t > 0:
		n.ai.globalAggro = -(t + 1)
	default:
		n.ai.globalAggro = -(s.rnd.Get(5) + 4)
	}
}

// gameTicks returns the game tick of the loop's time.
func (s *Server) gameTicks() int64 { return gameTick(s.Now()) }

// npcAggroRange is Mobius Npc.getAggroRange: the spawn's AI value, or the
// template's.
func (s *Server) npcAggroRange(n *Npc) int {
	if n.ai.aggroSet {
		return n.ai.aggroRange
	}
	return s.aiClassOf(n).aggroRange
}

// ----- the think scheduler (decision F5) -----

// startAITask puts n into a think slot, round-robin (Mobius startAITask).
func (s *Server) startAITask(n *Npc) {
	ai := &n.ai
	if ai.listed {
		return
	}
	f := &s.ai
	slot := f.nextSlot
	f.nextSlot = (slot + 1) % aiThinkSlots
	ai.listed, ai.slot, ai.slotIdx = true, slot, len(f.slots[slot])
	f.slots[slot] = append(f.slots[slot], n)
}

// stopAITask takes n out of its think slot (Mobius stopAITask).
func (s *Server) stopAITask(n *Npc) {
	ai := &n.ai
	if !ai.listed {
		return
	}
	list := s.ai.slots[ai.slot]
	last := len(list) - 1
	moved := list[last]
	list[ai.slotIdx] = moved
	moved.ai.slotIdx = ai.slotIdx
	list[last] = nil
	s.ai.slots[ai.slot] = list[:last]
	ai.listed = false
}

// aiTick runs the thinks of one slot: each thinking NPC thinks once a
// second. NPCs that left the world leave their slot.
func (s *Server) aiTick(time.Time) {
	f := &s.ai
	slot := int(f.tick % aiThinkSlots)
	f.tick++
	if len(f.slots[slot]) == 0 {
		return
	}
	batch := append(f.batch[:0], f.slots[slot]...)
	for _, n := range batch {
		if !n.ai.listed || n.ai.slot != slot {
			continue
		}
		if !n.InWorld() {
			s.stopAITask(n)
			continue
		}
		s.npcThink(n)
	}
	clear(batch)
	f.batch = batch[:0]
}

// ThinkingNpcs returns how many NPCs are in the think slots. Loop-only.
func (s *Server) ThinkingNpcs() int {
	total := 0
	for _, l := range s.ai.slots {
		total += len(l)
	}
	return total
}

// npcThink is Mobius AttackableAI.notifyActionThink: one think, unless one
// runs, the NPC's neighbourhood is inactive, the NPC is dead or all its
// skills are disabled (stunned, asleep, paralyzed).
func (s *Server) npcThink(n *Npc) {
	ai := &n.ai
	if !ai.attached || ai.thinking || n.Dead || !s.world.NeighborsActive(n) {
		return
	}
	if n.skill != nil && allSkillsDisabled(&n.skill.skillBase) {
		return
	}
	ai.thinking = true
	defer func() { ai.thinking = false }()
	s.minionLeash(n) // minion.go: Mobius checkMinionLeash
	switch ai.intention {
	case NpcActive:
		s.npcThinkActive(n)
	case NpcAttack:
		s.npcThinkAttack(n)
	case NpcCast:
		s.npcThinkCast(n) // ai_cast.go
	}
}

// ----- intentions -----

// npcSetIdle is Mobius AttackableAI.setIntentionIdle: an NPC that a player
// sees, or that is away from home, stays ACTIVE; otherwise it stops moving
// and fighting, leaves its think slot and loses its AI.
func (s *Server) npcSetIdle(n *Npc) {
	if s.shouldPromoteIdleToActive(n) {
		s.npcSetActive(n)
		return
	}
	ai := s.aiOf(n)
	ai.intention = NpcIdle
	ai.attackTarget = nil
	s.npcClientStopMoving(n, nil)
	s.StopAutoAttack(n)
	s.stopAITask(n)
	ai.attached = false
}

// shouldPromoteIdleToActive is Mobius's check of setIntentionIdle.
func (s *Server) shouldPromoteIdleToActive(n *Npc) bool {
	if n.Dead {
		return false
	}
	for range s.world.VisiblePlayers(n) {
		return true
	}
	home, terr, ok := s.npcHome(n)
	if !ok {
		return false
	}
	pos := n.Position()
	if terr != nil {
		return !territoryContainsXY(terr, pos.X, pos.Y)
	}
	return !(pos.Distance3D(home) < float64(2*s.settings.NPCs.MaxDriftRange))
}

// npcSetActive is Mobius AttackableAI.setIntentionActive: the attack
// timeout ends; a new ACTIVE intention stops the walk and the attack stance
// and thinks at once while the neighbourhood is active; the NPC thinks
// every second.
func (s *Server) npcSetActive(n *Npc) {
	ai := s.aiOf(n)
	ai.attackTimeout = aiNoTimeout
	if ai.intention != NpcActive {
		ai.intention = NpcActive
		if s.world.NeighborsActive(n) {
			ai.attackTarget = nil
			s.npcClientStopMoving(n, nil)
			s.StopAutoAttack(n)
			s.npcThink(n)
		}
	}
	s.startAITask(n)
}

// npcSetAttack is Mobius AttackableAI.setIntentionAttack: the attack times
// out in 2 min, the NPC fights target and thinks at once.
func (s *Server) npcSetAttack(n *Npc, target Creature) {
	ai := s.aiOf(n)
	ai.attackTimeout = aiMaxAttackTimeout + s.gameTicks()
	if target != nil {
		if ai.intention != NpcAttack {
			ai.intention = NpcAttack
		}
		ai.attackTarget = target
		s.npcThink(n)
	}
	s.startAITask(n)
}

// npcSetMoveTo is Mobius setIntentionMoveTo: the NPC leaves its attack
// stance and walks to pos.
func (s *Server) npcSetMoveTo(n *Npc, pos world.Position) {
	ai := s.aiOf(n)
	ai.intention = NpcMoveTo
	s.StopAutoAttack(n)
	s.AbortAttack(n)
	s.NpcMoveTo(n, pos)
	s.startAITask(n)
}

// ----- events -----

// canHandleAction is Mobius CreatureAI.canHandleAction.
func canHandleAction(n *Npc) bool { return n.InWorld() && n.ai.attached }

// npcArrived is Mobius CreatureAI.notifyActionArrived: the walk is over (a
// walk up to an object ends with StopMove), MOVE_TO turns ACTIVE, and the
// NPC thinks.
func (s *Server) npcArrived(n *Npc) {
	if n.summon != nil {
		s.summonArrived(n.summon) // summon_ai.go
		return
	}
	if n.class == nil || !n.class.attackable {
		n.Move = nil
		s.walkerArrived(n)   // walker.go: Mobius WalkingManager.onArrived
		s.npcMoveFinished(n) // quest_world_ai_events.go: on_move_finished
		return
	}
	s.aiOf(n)
	if !canHandleAction(n) {
		return
	}
	n.Move = nil
	s.npcClientStoppedMoving(n)
	s.walkerArrived(n)   // walker.go: Mobius WalkingManager.onArrived
	s.npcMoveFinished(n) // quest_world_ai_events.go: on_move_finished
	if n.ai.intention == NpcMoveTo {
		s.npcSetActive(n)
	}
	s.npcThink(n)
}

// notifyArrivedRevalidate is Mobius notifyActionArrivedRevalidate: a long
// walk thinks once on the way.
func (s *Server) notifyArrivedRevalidate(n *Npc) {
	if canHandleAction(n) && n.Move != nil {
		s.npcThink(n)
	}
}

// NotifyAttacked tells the AI of target that attacker attacked it (Mobius
// notifyActionAttacked), as the attack code does after a hit landed.
// Npc.ReduceHP already did so before the damage. Loop-only.
func (s *Server) NotifyAttacked(target, attacker Creature) {
	if d, ok := target.(*Door); ok {
		s.doorAttacked(d, attacker) // door_combat.go: DoorAI
		return
	}
	if n, ok := target.(*Npc); ok && n.class != nil && n.class.attackable && n.InWorld() {
		s.notifyAttacked(n, attacker)
	}
}

// notifyAttacked is Mobius AttackableAI.notifyActionAttacked: the attack
// times out in 2 min, a fresh monster may fight at once, the attacker gets
// on the hate list, the NPC runs and attacks its most hated enemy, its clan
// comes to help if the attacker is on the attack-by list, and it takes up
// its attack stance.
func (s *Server) notifyAttacked(n *Npc, attacker Creature) {
	if attacker == nil {
		return
	}
	ai := s.aiOf(n)
	ai.attackTimeout = aiMaxAttackTimeout + s.gameTicks()
	if ai.globalAggro < 0 {
		ai.globalAggro = 0
	}
	if !n.InAggroList(attacker) {
		s.addDamageHate(n, attacker, 0, 1)
	}
	if !n.Running() {
		s.SetNpcRunning(n, true)
	}
	if !ai.x.coreAIDisabled && (ai.intention != NpcAttack || n.MostHated() != ai.attackTarget) {
		s.npcSetAttack(n, attacker)
	}
	s.minionsAssist(n, attacker) // minion.go: MinionList.onAssist
	if len(s.aiClassOf(n).clans) > 0 && n.inAttackBy(attacker) {
		s.callClanHelp(n, attacker)
	}
	if canHandleAction(n) {
		s.StartAutoAttack(n)
	}
}

// NotifyAggression makes n hate target by aggro and attack it unless it
// fights already (Mobius notifyActionAggression): calls for help and
// scripts use it. Loop-only.
func (s *Server) NotifyAggression(n *Npc, target Creature, aggro int) {
	if n.class == nil || !n.class.attackable {
		return
	}
	s.notifyAggression(n, target, aggro)
}

func (s *Server) notifyAggression(n *Npc, target Creature, aggro int) {
	if target == nil || n.Dead {
		return
	}
	ai := s.aiOf(n)
	s.addDamageHate(n, target, 0, int64(aggro))
	if ai.intention != NpcAttack {
		if !n.Running() {
			s.SetNpcRunning(n, true)
		}
		s.npcSetAttack(n, target)
	}
	s.minionsAssist(n, target) // minion.go: MinionList.onAssist
}

// NotifyReadyToAct tells the AI of c that it may act again (Mobius
// notifyActionReadyToAct), as the attack code does after a swing: a
// fighting NPC's attack timeout starts over and it thinks. Loop-only.
func (s *Server) NotifyReadyToAct(c Creature) {
	n, ok := c.(*Npc)
	if !ok || !n.ai.attached {
		return
	}
	if n.ai.intention == NpcAttack {
		n.ai.attackTimeout = aiMaxAttackTimeout + s.gameTicks()
	}
	if canHandleAction(n) {
		s.npcThink(n)
	}
}

// aiForget is the OnForget hook (Mobius notifyActionForgetObject): an NPC
// that loses sight of its target forgets it, and of its enemy turns ACTIVE.
func (s *Server) aiForget(viewer, object world.Object) {
	n, ok := viewer.(*Npc)
	if !ok || !canHandleAction(n) {
		return
	}
	ai := &n.ai
	if ai.target == object {
		ai.target = nil
	}
	if ai.attackTarget != nil && world.Object(ai.attackTarget) == object {
		ai.attackTarget = nil
		if c, ok := object.(Creature); !ok || !c.IsDead() {
			s.npcSetActive(n)
		}
	}
	if ai.x.castTarget != nil && world.Object(ai.x.castTarget) == object {
		ai.x.castTarget = nil
		if c, ok := object.(Creature); !ok || !c.IsDead() {
			s.npcSetActive(n)
		}
	}
}

// npcDeathBeforeDie is the AI's part of Mobius doDie before Die: the walk
// stops (StopMove) and the clan of an attackable killed by a player, not a
// GM, comes for the killer.
func (s *Server) npcDeathBeforeDie(n *Npc, killer Creature) {
	n.ai.target = nil
	s.stopNpc(n, nil)
	if p, ok := killer.(*Player); ok && !p.IsGM() {
		s.callClanOnKill(n, p)
	}
}

// npcDeathAfterDie is Mobius notifyActionDeath and the end of doDie: the AI
// stops thinking, turns IDLE without targets and walks (ChangeMoveType),
// and the attack-by list empties.
func (s *Server) npcDeathAfterDie(n *Npc) {
	if n.ai.attached {
		s.stopAITask(n)
		n.ai.intention = NpcIdle
		n.ai.target, n.ai.attackTarget = nil, nil
		s.SetNpcRunning(n, false)
	}
	n.clearAttackBy()
}

// npcSpawned is Mobius Attackable.onSpawn, run when an attackable enters
// the world (a spawn, a respawn or a teleport): the spawn time, an empty
// hate list, walking, and an attached AI rolls its counter again and turns
// ACTIVE, but thinks only in an active region. aiData is the spawn's AI
// data (nil keeps the last), home the spawn point of an NPC without a
// spawn slot (nil keeps the last).
func (s *Server) npcSpawned(n *Npc, aiData map[string]int, home *world.Position) {
	if n.class == nil || !n.class.attackable {
		return
	}
	c := s.aiClassOf(n)
	ai := &n.ai
	if home != nil {
		ai.aggroRange, ai.aggroSet = aiData["aggroRange"]
		// A minion has no spawn and no home (Mobius getSpawn() == null).
		ai.home, ai.hasHome = *home, n.spawn == nil && n.boss.leader == nil
	}
	ai.spawnTime = s.Now()
	n.ClearAggroList()
	s.SetNpcRunning(n, false)
	switch {
	case c.guard, c.defender:
		s.npcSetActive(n) // Mobius Guard.onSpawn; SiegeGuardAI starts active
	case ai.attached:
		s.rollGlobalAggro(n)
		s.npcSetActive(n)
		if !s.world.RegionActive(n) {
			s.stopAITask(n)
		}
	}
}

// removeNpc takes n out of the world as Mobius decayMe does: an attackable
// forgets its attack-by and hate lists and turns IDLE first.
func (s *Server) removeNpc(n *Npc) {
	if n.class != nil && n.class.attackable && n.ai.attached {
		n.clearAttackBy()
		n.ClearAggroList()
		s.npcSetIdle(n)
		s.stopAITask(n)
	}
	s.walkerGone(n) // walker.go: Mobius onDecay's WalkingManager.onDeath
	s.RemoveObject(n)
}

// npcSeen queues a monster a player starts to see; wakeSeenNpcs wakes the
// queue once the visibility change is sent. Guards wake too: Mobius wakes
// only monsters, so a guard whose region once turned off would stay IDLE
// until it respawns.
func (s *Server) npcSeen(n *Npc) {
	if n.ai.seen || n.class == nil || !n.class.attackable || n.Dead {
		return
	}
	if c := s.aiClassOf(n); !n.class.monster && !c.guard && !c.defender {
		return
	}
	n.ai.seen = true
	s.ai.seen = append(s.ai.seen, n)
}

// wakeSeenNpcs wakes the monsters players started to see (Mobius
// World.addVisibleObject and switchRegion): an IDLE monster, with a new AI
// if it had none, turns ACTIVE.
func (s *Server) wakeSeenNpcs() {
	f := &s.ai
	if f.waking || len(f.seen) == 0 {
		return
	}
	f.waking = true
	defer func() { f.waking = false }()
	for len(f.seen) > 0 {
		batch := f.seen
		f.seen = f.seenSpare[:0]
		for _, n := range batch {
			n.ai.seen = false
			if n.InWorld() && !n.Dead && s.aiOf(n).intention == NpcIdle {
				s.npcSetActive(n)
			}
		}
		clear(batch)
		f.seenSpare = batch[:0]
	}
}

// aiRegionSwitched is the region activation hook of the AI (Mobius
// WorldRegion.switchAI(false)): the attackables of a region that turns off
// stop, forget their enemies, go home when more than 300 from their spawn,
// and turn IDLE.
func (s *Server) aiRegionSwitched(_ world.RegionKey, active bool, objects []world.Object) {
	if active {
		return
	}
	for _, o := range objects {
		n, ok := o.(*Npc)
		if !ok || n.class == nil || !n.class.attackable || !n.InWorld() {
			continue
		}
		n.ai.target = nil
		s.stopNpc(n, nil)
		n.ClearAggroList()
		n.clearAttackBy()
		if home, _, ok := s.npcHome(n); ok && n.Position().Distance2D(home) > float64(s.settings.NPCs.MaxDriftRange) {
			s.TeleportNpc(n, home)
		}
		if n.ai.attached {
			s.npcSetIdle(n)
			s.stopAITask(n)
		}
		s.AbortAttack(n)
	}
}

// ----- the attack feature's part -----

// npcDoAttack swings n at target (Mobius doAttack), or does what the test
// hook says instead.
func (s *Server) npcDoAttack(n *Npc, target Creature) {
	if target == nil {
		return
	}
	if fn := s.ai.doAttack; fn != nil {
		fn(n, target)
		return
	}
	s.DoAttack(n, target)
}

// aiAttackEvent is the OnNpcAttackEvent hook: the end of a swing makes the
// AI think (Mobius notifyActionReadyToAct), a hit that could not land too
// (notifyActionCancel), and a refused swing turns the AI ACTIVE (doAttack's
// setIntentionActive).
func (s *Server) aiAttackEvent(n *Npc, ev AttackEvent) {
	if n.class == nil || !n.class.attackable {
		return
	}
	switch ev {
	case NpcAttackReady:
		s.NotifyReadyToAct(n)
	case NpcAttackCancelled:
		if canHandleAction(n) {
			s.npcThink(n)
		}
	case NpcAttackRefused:
		s.npcSetActive(n)
	}
}

// ----- helpers -----

// npcHome returns where n belongs: its spawn slot's point and territory,
// or the point an NPC without a slot was spawned at.
func (s *Server) npcHome(n *Npc) (world.Position, *data.SpawnTerritory, bool) {
	if sp := n.spawn; sp != nil {
		return sp.lastPoint, sp.territory, true
	}
	if n.ai.hasHome {
		return n.ai.home, nil, true
	}
	return world.Position{}, nil, false
}

// territoryContainsXY is Mobius NpcSpawnTerritory.isInsideZone(x, y): inside
// the shape at its low Z and outside every banned area.
func territoryContainsXY(t *data.SpawnTerritory, x, y int32) bool {
	z := t.Shape.LowZ()
	if !t.Shape.Contains(int(x), int(y), z) {
		return false
	}
	for _, b := range t.Banned {
		if b.Contains(int(x), int(y), z) {
			return false
		}
	}
	return true
}

// creatureInWorld reports whether c is in the world (Mobius isSpawned).
func creatureInWorld(c world.Object) bool {
	switch c := c.(type) {
	case *Player:
		return c.InWorld()
	case *Npc:
		return c.InWorld()
	}
	return false
}

// creatureInvisible reports whether o is invisible.
func creatureInvisible(o world.Object) bool {
	switch o := o.(type) {
	case *Player:
		return o.Invisible()
	case *Npc:
		return o.Invisible()
	}
	return false
}

// creatureCollision is a creature's collision radius as Mobius's templates
// keep it: an int.
func creatureCollision(c Creature) int32 {
	switch c := c.(type) {
	case *Player:
		return int32(walkCollision(c))
	case *Npc:
		return npcCollision(c)
	}
	return 0
}

// inSurroundingRegion reports whether b is in the 3×3×3 regions around a
// (Mobius isInSurroundingRegion).
func inSurroundingRegion(a, b world.Object) bool {
	ka, ok1 := objectRegion(a)
	kb, ok2 := objectRegion(b)
	return ok1 && ok2 && ka.Adjacent(kb)
}

func objectRegion(o world.Object) (world.RegionKey, bool) {
	switch o := o.(type) {
	case *Player:
		return o.Region()
	case *Npc:
		return o.Region()
	}
	return world.RegionKey{}, false
}

// within2D is Mobius isInsideRadius2D: closer than r (2D).
func within2D(a, b world.Position, r int) bool { return a.Distance2D(b) < float64(r) }

// within3D is Mobius isInsideRadius3D: closer than r (3D).
func within3D(a, b world.Position, r int) bool { return a.Distance3D(b) < float64(r) }
