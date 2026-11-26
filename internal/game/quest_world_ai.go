package game

import (
	"cmp"
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/world"
)

// The AI scripts' NPC control (Plan 13, Phase 10.6, on ScriptOutput): the
// core AI, run and walk, invulnerability and mortality, intentions, follow,
// the hate list, leaders and minions, chests, skill target types, world
// scans, geodata, the game clock, and the NPC ref's spawn data and template
// parameters. Zones, grand bosses and cinematics are in
// quest_world_ai_zone.go, the events the game raises in
// quest_world_ai_events.go. Everything runs on the loop, while a handler
// runs or a timer fires.

var (
	_ scripting.NpcControl   = (*ScriptOutput)(nil)
	_ scripting.NpcMinions   = (*ScriptOutput)(nil)
	_ scripting.ChestActions = (*ScriptOutput)(nil)
	_ scripting.SkillTargets = (*ScriptOutput)(nil)
	_ scripting.WorldScan    = (*ScriptOutput)(nil)
	_ scripting.WorldGeo     = (*ScriptOutput)(nil)
	_ scripting.GameClock    = (*ScriptOutput)(nil)

	_ scripting.NpcSpawnInfo      = npcRef{}
	_ scripting.NpcTemplateParams = npcRef{}
)

// Mobius numbers of the script NPC control.
const (
	// npcFollowPeriod is Mobius CreatureFollowTaskManager's normal follow
	// period; npcFollowMin/Max bound its random follow range and
	// npcFollowGiveUp is how far a target may get before the NPC stops.
	npcFollowPeriod            = time.Second
	npcFollowMin, npcFollowMax = 50, 100
	npcFollowGiveUp            = 3000
	// treasureBombSkill is the skill of Chest.treasureBomb, at a level of 1
	// to 10.
	treasureBombSkill                = 4143
	treasureBombMin, treasureBombMax = 1, 10
)

// chestBoxes are the chest templates that are real boxes (Mobius
// Chest.BOXES); every other chest is a mimic.
var chestBoxes = []int{
	18265, 18266, 18267, 18268, 18269, 18270, 18271, 18272, 18273, 18274,
	18275, 18276, 18277, 18278, 18279, 18280, 18281, 18282, 18283, 18284,
	18285, 18286, 18287, 18288, 18289, 18290, 18291, 18292, 18293, 18294,
	18295, 18296, 18297, 18298,
}

// npcScriptState is what the AI scripts set on an NPC (Mobius Creature's
// _isInvul, _isMortal, Npc._isRandomWalkingEnabled, Chest's state and the
// follow task). It lives as long as the NPC, across respawns, as on Mobius,
// where a respawn reuses the Npc; the random walking flag and a chest's
// state start again on each spawn (Npc.onSpawn, Chest.onSpawn).
type npcScriptState struct {
	invul    bool
	immortal bool
	// walkSet is set when a script (or Chest.onSpawn) chose the random
	// walking since the spawn; walk is the choice.
	walkSet, walk bool
	// box is Chest.isBox; interacted Chest.isInteracted.
	box, interacted bool
	// follow is the follow task, 0 for none; followTarget its target.
	follow       world.TimerID
	followTarget Creature
	// noReturn is Mobius !canReturnToSpawnPoint (clan_script.go).
	noReturn bool
}

// scriptStateSpawned is the scripts' part of Mobius Npc.onSpawn and
// Chest.onSpawn: the random walking comes from the template again, a chest
// is not interacted and does not walk.
func (s *Server) scriptStateSpawned(n *Npc) {
	st := &n.script
	st.walkSet, st.walk = false, false
	st.box, st.interacted = false, false
	if s.npcIs(n, "Chest") {
		st.box = slices.Contains(chestBoxes, n.Template.ID)
		st.walkSet = true // Chest.onSpawn: setRandomWalking(false)
	}
}

// npcRandomWalks is Mobius isRandomWalkingEnabled: a script's choice since
// the spawn, else the template's (and the class's) default.
func (s *Server) npcRandomWalks(n *Npc) bool {
	if n.script.walkSet {
		return n.script.walk
	}
	return s.aiClassOf(n).randomWalk
}

// isInvul is Mobius isInvul for an NPC: a script's setInvul or an
// Invincible effect.
func (n *Npc) isInvul() bool { return n.script.invul || (n.skill != nil && n.skill.IsInvul()) }

// anyNpc returns the server and NPC of ref, in the world or not.
func (o *ScriptOutput) anyNpc(ref scripting.NpcRef) (*Server, *Npc, bool) {
	s, n := o.srv.Load(), npcOfRef(ref)
	return s, n, s != nil && n != nil
}

// attackableNpc returns the server and NPC of ref when it is an attackable
// in the world.
func (o *ScriptOutput) attackableNpc(ref scripting.NpcRef) (*Server, *Npc, bool) {
	s, n, ok := o.npc(ref)
	return s, n, ok && n.class != nil && n.class.attackable
}

// scriptObject returns a creature as scripts see it: a *Player, an NpcRef or
// nil.
func scriptObject(c world.Object) any {
	switch x := c.(type) {
	case *Player:
		if x != nil {
			return x
		}
	case *Npc:
		if x != nil {
			return x.ScriptRef()
		}
	}
	return nil
}

// ----- NpcControl -----

// NpcSetWalking is Mobius setWalking.
func (o *ScriptOutput) NpcSetWalking(npc scripting.NpcRef) {
	if s, n, ok := o.anyNpc(npc); ok {
		s.SetNpcRunning(n, false)
	}
}

// NpcIsRunning is Mobius isRunning.
func (o *ScriptOutput) NpcIsRunning(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && n.Running()
}

// NpcDisableCoreAI is Mobius disableCoreAI.
func (o *ScriptOutput) NpcDisableCoreAI(npc scripting.NpcRef, disabled bool) {
	if s, n, ok := o.anyNpc(npc); ok {
		s.SetNpcCoreAIDisabled(n, disabled)
	}
}

// NpcIsCoreAIDisabled is Mobius isCoreAIDisabled.
func (o *ScriptOutput) NpcIsCoreAIDisabled(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && n.NpcCoreAIDisabled()
}

// NpcSetImmobilized is Mobius setImmobilized: the flag the immobilizing
// effects set too.
func (o *ScriptOutput) NpcSetImmobilized(npc scripting.NpcRef, immobilized bool) {
	if n := npcOfRef(npc); n != nil && n.skill != nil {
		n.skill.SetImmobilized(immobilized)
	}
}

// NpcSetInvul is Mobius setInvul.
func (o *ScriptOutput) NpcSetInvul(npc scripting.NpcRef, invul bool) {
	if n := npcOfRef(npc); n != nil {
		n.script.invul = invul
	}
}

// NpcIsInvul is Mobius isInvul.
func (o *ScriptOutput) NpcIsInvul(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && n.isInvul()
}

// NpcSetMortal is Mobius setMortal: an immortal NPC goes down to 0 HP but
// does not die.
func (o *ScriptOutput) NpcSetMortal(npc scripting.NpcRef, mortal bool) {
	if n := npcOfRef(npc); n != nil {
		n.script.immortal = !mortal
	}
}

// NpcIsMortal is Mobius isMortal.
func (o *ScriptOutput) NpcIsMortal(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && !n.script.immortal
}

// NpcSetRandomWalking is Mobius setRandomWalking, until the NPC spawns
// again.
func (o *ScriptOutput) NpcSetRandomWalking(npc scripting.NpcRef, enabled bool) {
	if n := npcOfRef(npc); n != nil {
		n.script.walkSet, n.script.walk = true, enabled
	}
}

// NpcReturnHome is Mobius Attackable.returnHome: the hate list empties and
// the NPC walks to its spawn point.
func (o *ScriptOutput) NpcReturnHome(npc scripting.NpcRef) {
	if s, n, ok := o.attackableNpc(npc); ok && !n.Dead {
		s.returnHome(n)
	}
}

// NpcIntention is Mobius getAI().getIntention(): IDLE for an NPC without
// an AI.
func (o *ScriptOutput) NpcIntention(npc scripting.NpcRef) string {
	n := npcOfRef(npc)
	if n == nil || !n.ai.attached {
		return NpcIdle.String()
	}
	if name := n.ai.intention.String(); name != "?" {
		return name
	}
	return NpcActive.String()
}

// NpcSetIntention is Mobius getAI().setIntention(intention, arg) for the
// intentions the game's AI has: IDLE, ACTIVE, ATTACK (a target), MOVE_TO (a
// position) and FOLLOW (a target). CAST needs a skill (npc:cast), and REST,
// PICK_UP and INTERACT do nothing for NPCs.
func (o *ScriptOutput) NpcSetIntention(npc scripting.NpcRef, intention string, arg any) {
	s, n, ok := o.npc(npc)
	if !ok || n.Dead {
		return
	}
	attackable := n.class != nil && n.class.attackable
	switch intention {
	case "IDLE":
		if attackable {
			s.npcSetIdle(n)
		} else {
			s.NpcStopMove(n)
		}
	case "ACTIVE":
		if attackable {
			s.npcSetActive(n)
		}
	case "ATTACK":
		o.NpcAttack(npc, arg)
	case "MOVE_TO":
		if loc, isLoc := arg.(*scripting.Location); isLoc && loc != nil {
			o.NpcMoveTo(npc, *loc)
		}
	case "FOLLOW":
		o.NpcFollow(npc, arg)
	}
}

// NpcFollow is Mobius getAI().startFollow(target): every second the NPC
// walks up to 50–100 of the target, and gives up (IDLE) when the target is
// 3,000 away or gone (CreatureFollowTaskManager's normal follow).
func (o *ScriptOutput) NpcFollow(npc scripting.NpcRef, target any) {
	s, n, ok := o.npc(npc)
	t := creatureOfRef(target)
	if !ok || t == nil || Creature(n) == t {
		return
	}
	s.stopNpcFollow(n)
	n.script.followTarget = t
	n.script.follow = s.loop.Every(npcFollowPeriod, func() { s.npcFollowTick(n) })
}

// npcFollowTick is one run of the follow task.
func (s *Server) npcFollowTick(n *Npc) {
	t := n.script.followTarget
	if !n.InWorld() || n.Dead {
		s.stopNpcFollow(n)
		return
	}
	if t == nil || !creatureInWorld(t) {
		s.stopNpcFollow(n)
		s.npcFollowLost(n)
		return
	}
	r := s.rnd.Between(npcFollowMin, npcFollowMax)
	if within3D(n.Position(), t.Position(), r) {
		return
	}
	if !within3D(n.Position(), t.Position(), npcFollowGiveUp) {
		s.stopNpcFollow(n)
		s.npcFollowLost(n)
		return
	}
	s.NpcMoveToPawn(n, t, int32(r)) //nolint:gosec // 50 to 100
}

// npcFollowLost is the follow task's setIntentionIdle.
func (s *Server) npcFollowLost(n *Npc) {
	if n.class != nil && n.class.attackable {
		s.npcSetIdle(n)
		return
	}
	s.NpcStopMove(n)
}

// stopNpcFollow ends n's follow task (Mobius stopFollow).
func (s *Server) stopNpcFollow(n *Npc) {
	if n.script.follow != 0 {
		s.loop.Cancel(n.script.follow)
	}
	n.script.follow, n.script.followTarget = 0, nil
}

// NpcStopFollow is Mobius getAI().stopFollow().
func (o *ScriptOutput) NpcStopFollow(npc scripting.NpcRef) {
	if s, n, ok := o.anyNpc(npc); ok {
		s.stopNpcFollow(n)
	}
}

// NpcStopMoving is Mobius getAI().clientStopMoving(null).
func (o *ScriptOutput) NpcStopMoving(npc scripting.NpcRef) {
	if s, n, ok := o.npc(npc); ok {
		s.NpcStopMove(n)
	}
}

// NpcIsMoving is Mobius isMoving.
func (o *ScriptOutput) NpcIsMoving(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && n.Move != nil
}

// NpcIsInCombat is Mobius isInCombat (the attack stance).
func (o *ScriptOutput) NpcIsInCombat(npc scripting.NpcRef) bool {
	s, n, ok := o.anyNpc(npc)
	return ok && s.InCombat(n)
}

// NpcIsCastingNow is Mobius isCastingNow.
func (o *ScriptOutput) NpcIsCastingNow(npc scripting.NpcRef) bool {
	s, n, ok := o.anyNpc(npc)
	return ok && s.CastingNow(n)
}

// NpcHateList is Mobius getAggroList, most hated first (equal hates in the
// order they joined).
func (o *ScriptOutput) NpcHateList(npc scripting.NpcRef) []scripting.HateEntry {
	n := npcOfRef(npc)
	if n == nil {
		return nil
	}
	aggro := n.Aggro()
	slices.SortStableFunc(aggro, func(a, b AggroInfo) int { return cmp.Compare(b.Hate, a.Hate) })
	out := make([]scripting.HateEntry, 0, len(aggro))
	for _, a := range aggro {
		if t := scriptObject(a.Attacker); t != nil {
			out = append(out, scripting.HateEntry{Target: t, Hate: a.Hate, Damage: a.Damage})
		}
	}
	return out
}

// NpcMostHated is Mobius getMostHated.
func (o *ScriptOutput) NpcMostHated(npc scripting.NpcRef) any {
	n := npcOfRef(npc)
	if n == nil {
		return nil
	}
	if c := n.MostHated(); c != nil {
		return scriptObject(c)
	}
	return nil
}

// NpcClearAggroList is Mobius clearAggroList.
func (o *ScriptOutput) NpcClearAggroList(npc scripting.NpcRef) {
	if n := npcOfRef(npc); n != nil {
		n.ClearAggroList()
	}
}

// NpcStopHating is Mobius stopHating: the hate drops to 0, the target stays
// on the list.
func (o *ScriptOutput) NpcStopHating(npc scripting.NpcRef, target any) {
	if n, t := npcOfRef(npc), creatureOfRef(target); n != nil && t != nil {
		n.StopHating(t)
	}
}

// npcCalmAggro is the global aggression an NPC gets when reduceHate calms
// it (Mobius setGlobalAggro(-25)).
const npcCalmAggro = -25

// NpcReduceHate is Mobius Attackable.reduceHate(target, -amount): the hate
// for target goes down by amount; when it is 0 or more and nobody is hated
// any more, the NPC calms down: no aggression for 25 thinks, an empty hate
// list, ACTIVE and walking.
func (o *ScriptOutput) NpcReduceHate(npc scripting.NpcRef, target any, amount int64) {
	s, n, ok := o.anyNpc(npc)
	t := creatureOfRef(target)
	if !ok || t == nil || n.class == nil || !n.class.attackable {
		return
	}
	a := n.aggroOf(t)
	if a == nil {
		return
	}
	a.addHate(-amount)
	if a.Hate >= 0 && n.MostHated() == nil {
		s.aiOf(n).globalAggro = npcCalmAggro
		n.ClearAggroList()
		s.npcSetActive(n)
		s.SetNpcRunning(n, false)
	}
}

// NpcSetHeading turns the NPC and shows it to the players around (Mobius
// setHeading and broadcastInfo).
func (o *ScriptOutput) NpcSetHeading(npc scripting.NpcRef, heading int) {
	if _, n, ok := o.npc(npc); ok {
		n.SetHeading(int32(heading)) //nolint:gosec // headings fit in int32
		n.BroadcastInfo()
	}
}

// ----- NpcMinions -----

// NpcLeader is Mobius getLeader.
func (o *ScriptOutput) NpcLeader(npc scripting.NpcRef) scripting.NpcRef {
	if n := npcOfRef(npc); n != nil {
		if l := n.Leader(); l != nil {
			return l.ScriptRef()
		}
	}
	return nil
}

// NpcMinionsOf is Mobius getMinionList().getSpawnedMinions(): the living
// minions, in spawn order.
func (o *ScriptOutput) NpcMinionsOf(npc scripting.NpcRef) []scripting.NpcRef {
	n := npcOfRef(npc)
	if n == nil {
		return nil
	}
	var out []scripting.NpcRef
	for _, m := range n.Minions() {
		if !m.Dead && m.InWorld() {
			out = append(out, m.ScriptRef())
		}
	}
	return out
}

// NpcAddMinion is Mobius addMinion(master, minionId).
func (o *ScriptOutput) NpcAddMinion(leader scripting.NpcRef, npcID int) scripting.NpcRef {
	s, n, ok := o.npc(leader)
	if !ok || n.Dead {
		return nil
	}
	if m := s.SpawnMinion(n, npcID); m != nil {
		return m.ScriptRef()
	}
	return nil
}

// ----- ChestActions and SkillTargets -----

// ChestIsBox is Mobius Chest.isBox.
func (o *ScriptOutput) ChestIsBox(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && n.script.box
}

// ChestIsInteracted is Mobius Chest.isInteracted.
func (o *ScriptOutput) ChestIsInteracted(npc scripting.NpcRef) bool {
	n := npcOfRef(npc)
	return n != nil && n.script.interacted
}

// ChestSetInteracted is Mobius Chest.setInteracted.
func (o *ScriptOutput) ChestSetInteracted(npc scripting.NpcRef) {
	if n := npcOfRef(npc); n != nil {
		n.script.interacted = true
	}
}

// ChestTreasureBomb is Mobius Chest.treasureBomb: the chest casts Treasure
// Bomb (4143) at the level round(level / 10), 1 to 10, on itself.
func (o *ScriptOutput) ChestTreasureBomb(npc scripting.NpcRef) {
	s, n, ok := o.npc(npc)
	if !ok || n.Dead || n.skill == nil {
		return
	}
	lvl := int(math.Floor(float64(float32(n.Level())/10) + 0.5)) // Java Math.round(float)
	lvl = max(treasureBombMin, min(treasureBombMax, lvl))
	sk, found := s.Skill(treasureBombSkill, lvl)
	if !found {
		return
	}
	n.skill.SetTarget(s.skillCreatureOf(n))
	s.DoCast(n, sk)
}

// SkillTargetType is skill.getTargetType().name(); "NONE" for a skill the
// data does not have.
func (o *ScriptOutput) SkillTargetType(skill scripting.Skill) string {
	if s := o.srv.Load(); s != nil {
		if sk, ok := s.Skill(skill.ID, skill.Level); ok {
			return sk.TargetType.String()
		}
	}
	return "NONE"
}

// ----- WorldScan, WorldGeo, GameClock -----

// scanCenter returns the world object of a PlayerRef or NpcRef in the
// world.
func scanCenter(obj any) world.Object {
	c := creatureOfRef(obj)
	if c == nil || !creatureInWorld(c) {
		return nil
	}
	return c
}

// NpcsAround is World.getVisibleObjectsInRange(obj, Npc.class, radius):
// the NPCs in obj's regions within radius (3D), all of them for 0.
func (o *ScriptOutput) NpcsAround(obj any, radius int) []scripting.NpcRef {
	s, c := o.srv.Load(), scanCenter(obj)
	if s == nil || c == nil {
		return nil
	}
	seq := s.world.Visible(c)
	if radius > 0 {
		seq = s.world.VisibleInRange(c, int32(min(radius, math.MaxInt32))) //nolint:gosec // clamped
	}
	var out []scripting.NpcRef
	for x := range seq {
		if n, ok := x.(*Npc); ok {
			out = append(out, n.ScriptRef())
		}
	}
	return out
}

// PlayersAround is NpcsAround for players.
func (o *ScriptOutput) PlayersAround(obj any, radius int) []scripting.PlayerRef {
	s, c := o.srv.Load(), scanCenter(obj)
	if s == nil || c == nil {
		return nil
	}
	seq := s.world.VisiblePlayers(c)
	if radius > 0 {
		seq = s.world.VisiblePlayersInRange(c, int32(min(radius, math.MaxInt32))) //nolint:gosec // clamped
	}
	var out []scripting.PlayerRef
	for v := range seq {
		if p, ok := v.(*Player); ok {
			out = append(out, p)
		}
	}
	return out
}

// NpcsByID returns the NPCs of the template in the world, dead ones too, by
// object ID.
func (o *ScriptOutput) NpcsByID(npcID int) []scripting.NpcRef {
	s := o.srv.Load()
	if s == nil {
		return nil
	}
	var found []*Npc
	for x := range s.world.Objects() {
		if n, ok := x.(*Npc); ok && n.Template.ID == npcID {
			found = append(found, n)
		}
	}
	slices.SortFunc(found, func(a, b *Npc) int { return cmp.Compare(a.ID(), b.ID()) })
	out := make([]scripting.NpcRef, len(found))
	for i, n := range found {
		out[i] = n.ScriptRef()
	}
	return out
}

// ValidLocation is GeoEngine.getValidLocation.
func (o *ScriptOutput) ValidLocation(from, to scripting.Location) scripting.Location {
	s := o.srv.Load()
	if s == nil {
		return to
	}
	//nolint:gosec // world coordinates fit in int32
	p := s.validLocation(world.Position{X: int32(from.X), Y: int32(from.Y), Z: int32(from.Z)},
		world.Position{X: int32(to.X), Y: int32(to.Y), Z: int32(to.Z)})
	return scripting.Location{X: int(p.X), Y: int(p.Y), Z: int(p.Z), Heading: to.Heading}
}

// GameTicks is GameTimeTaskManager.getGameTicks.
func (o *ScriptOutput) GameTicks() int64 {
	if s := o.srv.Load(); s != nil {
		return s.world.GameTime().Ticks()
	}
	return 0
}

// GameTime is GameTimeTaskManager.getGameTime: minutes since game
// midnight.
func (o *ScriptOutput) GameTime() int {
	if s := o.srv.Load(); s != nil {
		return int(s.world.GameTime().Minutes())
	}
	return 0
}

// IsNight is GameTimeTaskManager.isNight.
func (o *ScriptOutput) IsNight() bool {
	s := o.srv.Load()
	return s != nil && s.world.GameTime().IsNight()
}

// ----- the NPC ref's spawn data and template parameters -----

// SpawnLocation is getSpawn().getLocation(): where the NPC last spawned.
func (r npcRef) SpawnLocation() (scripting.Location, bool) {
	if r.n.srv == nil {
		return scripting.Location{}, false
	}
	pos, _, ok := r.n.srv.npcHome(r.n)
	if !ok {
		return scripting.Location{}, false
	}
	return scripting.Location{X: int(pos.X), Y: int(pos.Y), Z: int(pos.Z), Heading: int(pos.Heading)}, true
}

// AIValue is a value of the spawn block's AI data (Mobius getAIValue).
func (r npcRef) AIValue(name string) (int, bool) {
	if r.n.spawn == nil {
		return 0, false
	}
	v, ok := r.n.spawn.aiData()[name]
	return v, ok
}

// TemplateParam is a template parameter (Mobius getParameters()).
func (r npcRef) TemplateParam(name string) (string, bool) {
	v, ok := r.n.Template.Parameters.Params[name]
	return v, ok
}
