package scripttest

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

// The fake world of the AI API (Plan 13, Phase 10.6): NPC control with
// hate lists, minions, chests, world scans, zones, doors, the game clock,
// the grand boss store and cinematics. Tests set the state through the
// fields of Npc, Player and the World methods here, and read what scripts
// did from the same fields and World.Actions.

var (
	_ scripting.NpcControl     = (*World)(nil)
	_ scripting.NpcMinions     = (*World)(nil)
	_ scripting.ChestActions   = (*World)(nil)
	_ scripting.SkillTargets   = (*World)(nil)
	_ scripting.WorldScan      = (*World)(nil)
	_ scripting.WorldGeo       = (*World)(nil)
	_ scripting.ZoneControl    = (*World)(nil)
	_ scripting.DoorControl    = (*World)(nil)
	_ scripting.GameClock      = (*World)(nil)
	_ scripting.GrandBossStore = (*World)(nil)
	_ scripting.Cinematics     = (*World)(nil)

	_ scripting.NpcSpawnInfo      = npcRef{}
	_ scripting.NpcTemplateParams = npcRef{}
	_ scripting.PlayerHP          = playerRef{}
	_ scripting.PlayerSummon      = playerRef{}
)

// VisibleRange is the radius of world scans that give none.
const VisibleRange = 4000

// aiWorld is the World's AI state.
type aiWorld struct {
	zones       map[int]*Zone
	doors       map[int]bool
	bosses      map[int]*Boss
	targetTypes map[int]string
	geo         func(from, to scripting.Location) scripting.Location
	spawnEvents bool
	// gameStart is the real time of game 00:00 on day 0.
	gameStart time.Time
}

func (w *World) aiState() *aiWorld {
	a := &w.ai
	if a.zones == nil {
		a.zones = make(map[int]*Zone)
		a.doors = make(map[int]bool)
		a.bosses = make(map[int]*Boss)
		a.targetTypes = make(map[int]string)
		a.gameStart = Start
	}
	return a
}

// fakeOf returns the *Player or *Npc behind a ref, or nil.
func fakeOf(obj any) any {
	switch x := obj.(type) {
	case playerRef:
		return x.p
	case npcRef:
		return x.n
	}
	return nil
}

// refOf returns the ref of a *Player or *Npc, or nil.
func refOf(obj any) any {
	switch x := obj.(type) {
	case *Player:
		if x != nil {
			return x.Ref()
		}
	case *Npc:
		if x != nil {
			return x.Ref()
		}
	}
	return nil
}

func (n *Npc) addHate(target any, damage, hate int64) {
	if target == nil {
		return
	}
	for i := range n.Hate {
		if n.Hate[i].Target == target {
			n.Hate[i].Hate += hate
			n.Hate[i].Damage += damage
			n.sortHate()
			return
		}
	}
	n.Hate = append(n.Hate, Hate{Target: target, Hate: hate, Damage: damage})
	n.sortHate()
}

func (n *Npc) sortHate() {
	slices.SortStableFunc(n.Hate, func(a, b Hate) int {
		switch {
		case a.Hate > b.Hate:
			return -1
		case a.Hate < b.Hate:
			return 1
		}
		return 0
	})
}

// --- NpcControl ---

func (w *World) npcDo(npc scripting.NpcRef, kind string, value int, do func(n *Npc)) {
	if n := npcOf(npc); n != nil && do != nil {
		do(n)
	}
	w.act(Action{Kind: kind, Object: npc.ObjectID(), Value: value})
}

func flag(on bool) int {
	if on {
		return 1
	}
	return 0
}

func npcFlag(npc scripting.NpcRef, get func(n *Npc) bool) bool {
	if n := npcOf(npc); n != nil {
		return get(n)
	}
	return false
}

// NpcSetWalking clears Running and sets Walking.
func (w *World) NpcSetWalking(npc scripting.NpcRef) {
	w.npcDo(npc, "walk", 0, func(n *Npc) { n.Running, n.Walking = false, true })
}

// NpcIsRunning reads Running.
func (w *World) NpcIsRunning(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.Running })
}

// NpcDisableCoreAI sets CoreAIDisabled (action "core_ai", Value 1 for
// disabled).
func (w *World) NpcDisableCoreAI(npc scripting.NpcRef, disabled bool) {
	w.npcDo(npc, "core_ai", flag(disabled), func(n *Npc) { n.CoreAIDisabled = disabled })
}

// NpcIsCoreAIDisabled reads CoreAIDisabled.
func (w *World) NpcIsCoreAIDisabled(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.CoreAIDisabled })
}

// NpcSetImmobilized sets Immobilized.
func (w *World) NpcSetImmobilized(npc scripting.NpcRef, on bool) {
	w.npcDo(npc, "immobilized", flag(on), func(n *Npc) { n.Immobilized = on })
}

// NpcSetInvul sets Invul.
func (w *World) NpcSetInvul(npc scripting.NpcRef, on bool) {
	w.npcDo(npc, "invul", flag(on), func(n *Npc) { n.Invul = on })
}

// NpcIsInvul reads Invul.
func (w *World) NpcIsInvul(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.Invul })
}

// NpcSetMortal sets Immortal to the opposite.
func (w *World) NpcSetMortal(npc scripting.NpcRef, on bool) {
	w.npcDo(npc, "mortal", flag(on), func(n *Npc) { n.Immortal = !on })
}

// NpcIsMortal reads !Immortal.
func (w *World) NpcIsMortal(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return !n.Immortal })
}

// NpcSetRandomWalking sets NoRandomWalk to the opposite.
func (w *World) NpcSetRandomWalking(npc scripting.NpcRef, on bool) {
	w.npcDo(npc, "random_walk", flag(on), func(n *Npc) { n.NoRandomWalk = !on })
}

// NpcReturnHome clears the hate and sends the NPC to its spawn point (it
// arrives with Harness.Arrive).
func (w *World) NpcReturnHome(npc scripting.NpcRef) {
	w.npcDo(npc, "return_home", 0, func(n *Npc) {
		n.Hate = nil
		if n.Spawn != nil {
			dest := *n.Spawn
			n.Dest, n.Moving, n.Intention = &dest, true, "MOVE_TO"
		}
	})
}

// NpcIntention reads Intention, "ACTIVE" when empty.
func (w *World) NpcIntention(npc scripting.NpcRef) string {
	if n := npcOf(npc); n != nil && n.Intention != "" {
		return n.Intention
	}
	return "ACTIVE"
}

// NpcSetIntention sets Intention; MOVE_TO with a position sets Dest.
func (w *World) NpcSetIntention(npc scripting.NpcRef, intention string, arg any) {
	a := Action{Kind: "intention", Object: npc.ObjectID(), Text: intention, Target: objectID(arg)}
	if loc, ok := arg.(*scripting.Location); ok {
		a.Loc = *loc
	}
	if n := npcOf(npc); n != nil {
		n.Intention = intention
		if loc, ok := arg.(*scripting.Location); ok && intention == "MOVE_TO" {
			dest := *loc
			n.Dest, n.Moving = &dest, true
		}
	}
	w.act(a)
}

// NpcFollow records the follow; the intention becomes FOLLOW.
func (w *World) NpcFollow(npc scripting.NpcRef, target any) {
	if n := npcOf(npc); n != nil {
		n.Intention = "FOLLOW"
	}
	w.act(Action{Kind: "follow", Object: npc.ObjectID(), Target: objectID(target)})
}

// NpcStopFollow records it.
func (w *World) NpcStopFollow(npc scripting.NpcRef) { w.npcDo(npc, "stop_follow", 0, nil) }

// NpcStopMoving clears Moving and Dest.
func (w *World) NpcStopMoving(npc scripting.NpcRef) {
	w.npcDo(npc, "stop_moving", 0, func(n *Npc) { n.Moving, n.Dest = false, nil })
}

// NpcIsMoving reads Moving.
func (w *World) NpcIsMoving(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.Moving })
}

// NpcIsInCombat reads InCombat.
func (w *World) NpcIsInCombat(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.InCombat })
}

// NpcIsCastingNow reads CastingNow.
func (w *World) NpcIsCastingNow(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.CastingNow })
}

// NpcHateList returns Hate.
func (w *World) NpcHateList(npc scripting.NpcRef) []scripting.HateEntry {
	n := npcOf(npc)
	if n == nil {
		return nil
	}
	out := make([]scripting.HateEntry, 0, len(n.Hate))
	for _, h := range n.Hate {
		out = append(out, scripting.HateEntry{Target: refOf(h.Target), Hate: h.Hate, Damage: h.Damage})
	}
	return out
}

// NpcMostHated returns the first of Hate.
func (w *World) NpcMostHated(npc scripting.NpcRef) any {
	if n := npcOf(npc); n != nil && len(n.Hate) > 0 {
		return refOf(n.Hate[0].Target)
	}
	return nil
}

// NpcClearAggroList empties Hate.
func (w *World) NpcClearAggroList(npc scripting.NpcRef) {
	w.npcDo(npc, "clear_aggro", 0, func(n *Npc) { n.Hate = nil })
}

// NpcStopHating removes target from Hate.
func (w *World) NpcStopHating(npc scripting.NpcRef, target any) {
	t := fakeOf(target)
	if n := npcOf(npc); n != nil {
		n.Hate = slices.DeleteFunc(n.Hate, func(h Hate) bool { return h.Target == t })
	}
	w.act(Action{Kind: "stop_hating", Object: npc.ObjectID(), Target: objectID(target)})
}

// NpcReduceHate lowers target's hate, removing it at 0.
func (w *World) NpcReduceHate(npc scripting.NpcRef, target any, amount int64) {
	t := fakeOf(target)
	if n := npcOf(npc); n != nil {
		for i := range n.Hate {
			if n.Hate[i].Target == t {
				n.Hate[i].Hate -= amount
			}
		}
		n.Hate = slices.DeleteFunc(n.Hate, func(h Hate) bool { return h.Hate <= 0 })
		n.sortHate()
	}
	w.act(Action{Kind: "reduce_hate", Object: npc.ObjectID(), Target: objectID(target), Damage: amount})
}

// NpcSetHeading sets Heading.
func (w *World) NpcSetHeading(npc scripting.NpcRef, heading int) {
	w.npcDo(npc, "heading", heading, func(n *Npc) { n.Heading = heading })
}

// --- NpcSpawnInfo, NpcTemplateParams, PlayerHP, PlayerSummon ---

func (r npcRef) SpawnLocation() (scripting.Location, bool) {
	if r.n.Spawn == nil {
		return scripting.Location{}, false
	}
	return *r.n.Spawn, true
}

func (r npcRef) AIValue(name string) (int, bool) {
	v, ok := r.n.AIData[name]
	return v, ok
}

func (r npcRef) TemplateParam(name string) (string, bool) {
	v, ok := r.n.Params[name]
	return v, ok
}

func (r playerRef) CurrentHP() float64  { return r.p.HP }
func (r playerRef) MaxHPValue() float64 { return r.p.MaxHP }

func (r playerRef) SummonObject() any {
	if r.p.Summon == nil {
		return nil
	}
	return r.p.Summon.Ref()
}

// --- NpcMinions ---

// NpcLeader returns Leader.
func (w *World) NpcLeader(npc scripting.NpcRef) scripting.NpcRef {
	if n := npcOf(npc); n != nil && n.Leader != nil {
		return n.Leader.Ref()
	}
	return nil
}

// NpcMinionsOf returns the living Minions.
func (w *World) NpcMinionsOf(npc scripting.NpcRef) []scripting.NpcRef {
	n := npcOf(npc)
	if n == nil {
		return nil
	}
	var out []scripting.NpcRef
	for _, m := range n.Minions {
		if !m.Dead && !m.Deleted {
			out = append(out, m.Ref())
		}
	}
	return out
}

// NpcAddMinion spawns a minion next to the leader.
func (w *World) NpcAddMinion(leader scripting.NpcRef, npcID int) scripting.NpcRef {
	l := npcOf(leader)
	if l == nil {
		return nil
	}
	m := npcOf(w.AddSpawn(npcID, scripting.Location{X: l.X, Y: l.Y, Z: l.Z, Heading: l.Heading}, false, 0, false))
	m.Leader = l
	l.Minions = append(l.Minions, m)
	return m.Ref()
}

// --- ChestActions, SkillTargets ---

func (w *World) ChestIsBox(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.Box })
}

func (w *World) ChestIsInteracted(npc scripting.NpcRef) bool {
	return npcFlag(npc, func(n *Npc) bool { return n.Interacted })
}

func (w *World) ChestSetInteracted(npc scripting.NpcRef) {
	w.npcDo(npc, "interacted", 0, func(n *Npc) { n.Interacted = true })
}

// ChestTreasureBomb sets Bombed.
func (w *World) ChestTreasureBomb(npc scripting.NpcRef) {
	w.npcDo(npc, "treasure_bomb", 0, func(n *Npc) { n.Bombed = true })
}

// SetSkillTargetType sets a skill's target type; others are "ONE".
func (w *World) SetSkillTargetType(skillID int, targetType string) {
	w.aiState().targetTypes[skillID] = targetType
}

// SkillTargetType returns the type set with SetSkillTargetType, else "ONE".
func (w *World) SkillTargetType(s scripting.Skill) string {
	if t, ok := w.aiState().targetTypes[s.ID]; ok {
		return t
	}
	return "ONE"
}

// --- WorldScan and WorldGeo ---

func dist2D(a, b scripting.Location) float64 {
	return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))
}

func locOf(obj any) (scripting.Location, bool) {
	if l, ok := obj.(scripting.Located); ok {
		return l.Location(), true
	}
	return scripting.Location{}, false
}

// NpcsAround returns the NPCs within radius (VisibleRange for 0) of obj,
// by object ID.
func (w *World) NpcsAround(obj any, radius int) []scripting.NpcRef {
	center, ok := locOf(obj)
	if !ok {
		return nil
	}
	if radius <= 0 {
		radius = VisibleRange
	}
	self := objectID(obj)
	var out []scripting.NpcRef
	for _, id := range sortedIDs(w.npcs) {
		n := w.npcs[id]
		if id == self || n.Deleted {
			continue
		}
		if dist2D(center, scripting.Location{X: n.X, Y: n.Y}) <= float64(radius) {
			out = append(out, n.Ref())
		}
	}
	return out
}

// PlayersAround returns the players within radius of obj, by object ID.
func (w *World) PlayersAround(obj any, radius int) []scripting.PlayerRef {
	center, ok := locOf(obj)
	if !ok {
		return nil
	}
	if radius <= 0 {
		radius = VisibleRange
	}
	self := objectID(obj)
	var out []scripting.PlayerRef
	for _, id := range sortedIDs(w.players) {
		p := w.players[id]
		if id == self {
			continue
		}
		if dist2D(center, scripting.Location{X: p.X, Y: p.Y}) <= float64(radius) {
			out = append(out, p.Ref())
		}
	}
	return out
}

// NpcsByID returns the NPCs of the template, by object ID.
func (w *World) NpcsByID(npcID int) []scripting.NpcRef {
	var out []scripting.NpcRef
	for _, id := range sortedIDs(w.npcs) {
		if n := w.npcs[id]; n.ID == npcID && !n.Deleted {
			out = append(out, n.Ref())
		}
	}
	return out
}

func sortedIDs[V any](m map[int32]V) []int32 {
	ids := make([]int32, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// SetGeo makes world.valid_location answer with fn; without it the
// destination is valid.
func (w *World) SetGeo(fn func(from, to scripting.Location) scripting.Location) {
	w.aiState().geo = fn
}

// ValidLocation is the SetGeo function, or to.
func (w *World) ValidLocation(from, to scripting.Location) scripting.Location {
	if g := w.aiState().geo; g != nil {
		return g(from, to)
	}
	return to
}

// --- Zones and doors ---

// Zone is a fake zone. Players lists who is inside; Allowed the players
// allowed in, with how long.
type Zone struct {
	ID      int
	Players []*Player
	Npcs    []*Npc
	Allowed map[int32]time.Duration
	// Oust is where zones.oust_all sends players.
	Oust scripting.Location
}

// AddZone makes a zone; put players and NPCs inside through its fields.
func (w *World) AddZone(id int) *Zone {
	z := &Zone{ID: id, Allowed: make(map[int32]time.Duration)}
	w.aiState().zones[id] = z
	return z
}

func (w *World) zone(id int) *Zone {
	if z, ok := w.aiState().zones[id]; ok {
		return z
	}
	return w.AddZone(id)
}

// ZonePlayers returns Players.
func (w *World) ZonePlayers(id int) []scripting.PlayerRef {
	var out []scripting.PlayerRef
	for _, p := range w.zone(id).Players {
		out = append(out, p.Ref())
	}
	return out
}

// ZoneIsInside looks in Players and Npcs.
func (w *World) ZoneIsInside(id int, obj any) bool {
	z := w.zone(id)
	switch x := fakeOf(obj).(type) {
	case *Player:
		return slices.Contains(z.Players, x)
	case *Npc:
		return slices.Contains(z.Npcs, x)
	}
	return false
}

// ZoneMovePlayers teleports the players inside.
func (w *World) ZoneMovePlayers(id int, loc scripting.Location) {
	for _, p := range w.zone(id).Players {
		p.X, p.Y, p.Z = loc.X, loc.Y, loc.Z
	}
	w.act(Action{Kind: "zone_move", Value: id, Loc: loc})
}

// ZoneOustAll sends the players to Oust and empties the zone.
func (w *World) ZoneOustAll(id int) {
	z := w.zone(id)
	for _, p := range z.Players {
		p.X, p.Y, p.Z = z.Oust.X, z.Oust.Y, z.Oust.Z
	}
	z.Players = nil
	clear(z.Allowed)
	w.act(Action{Kind: "zone_oust", Value: id})
}

// ZoneAllowEntry records the allowance.
func (w *World) ZoneAllowEntry(id int, p scripting.PlayerRef, d time.Duration) {
	w.zone(id).Allowed[p.ObjectID()] = d
	w.act(Action{Kind: "zone_allow", Value: id, Target: p.ObjectID()})
}

// ZoneRemovePlayer removes the allowance.
func (w *World) ZoneRemovePlayer(id int, p scripting.PlayerRef) {
	delete(w.zone(id).Allowed, p.ObjectID())
	w.act(Action{Kind: "zone_remove", Value: id, Target: p.ObjectID()})
}

// AddDoor makes a door, open or closed.
func (w *World) AddDoor(id int, open bool) { w.aiState().doors[id] = open }

// DoorOpen reports a door's state.
func (w *World) DoorOpen(id int) bool { return w.aiState().doors[id] }

// OpenDoor opens a door made with AddDoor.
func (w *World) OpenDoor(id int) bool { return w.setDoor(id, true) }

// CloseDoor closes a door made with AddDoor.
func (w *World) CloseDoor(id int) bool { return w.setDoor(id, false) }

func (w *World) setDoor(id int, open bool) bool {
	d := w.aiState().doors
	if _, ok := d[id]; !ok {
		return false
	}
	d[id] = open
	kind := "door_close"
	if open {
		kind = "door_open"
	}
	w.act(Action{Kind: kind, Value: id})
	return true
}

// DoorIsOpen reports a door made with AddDoor.
func (w *World) DoorIsOpen(id int) (bool, bool) {
	open, ok := w.aiState().doors[id]
	return open, ok
}

// --- GameClock ---

// Game time runs 6 game days per real day, as in Mobius: one game minute
// is 10 real seconds.
const gameMinute = 10 * time.Second

// SetGameTime moves the game clock so that the game time now is minutes
// since midnight (the real clock does not move).
func (w *World) SetGameTime(minutes int) {
	a := w.aiState()
	elapsed := time.Duration(minutes) * gameMinute
	a.gameStart = w.now.Add(-elapsed)
}

// GameTicks counts 100 ms ticks since the game clock started.
func (w *World) GameTicks() int64 {
	return int64(w.now.Sub(w.aiState().gameStart) / (100 * time.Millisecond))
}

// GameTime returns the game minutes since midnight.
func (w *World) GameTime() int {
	m := int(w.now.Sub(w.aiState().gameStart) / gameMinute)
	return ((m % 1440) + 1440) % 1440
}

// IsNight is game hour < 6.
func (w *World) IsNight() bool { return w.GameTime() < 6*60 }

// --- GrandBossStore ---

// Boss is a grand boss's saved state in the fake store.
type Boss struct {
	Status int
	Info   scripting.BossInfo
	Live   *Npc
	// Saves counts status and info writes.
	Saves int
}

// AddGrandBoss puts a boss into the store.
func (w *World) AddGrandBoss(id, status int) *Boss {
	b := &Boss{Status: status}
	w.aiState().bosses[id] = b
	return b
}

// GrandBoss returns a boss of the store, or nil.
func (w *World) GrandBoss(id int) *Boss { return w.aiState().bosses[id] }

// BossStatus reads the store.
func (w *World) BossStatus(id int) (int, bool) {
	if b := w.GrandBoss(id); b != nil {
		return b.Status, true
	}
	return 0, false
}

// SetBossStatus writes the store (a boss is added when missing).
func (w *World) SetBossStatus(id, status int) {
	b := w.GrandBoss(id)
	if b == nil {
		b = w.AddGrandBoss(id, status)
	}
	b.Status = status
	b.Saves++
}

// BossInfo reads the store.
func (w *World) BossInfo(id int) (scripting.BossInfo, bool) {
	if b := w.GrandBoss(id); b != nil {
		return b.Info, true
	}
	return scripting.BossInfo{}, false
}

// SetBossInfo writes the store.
func (w *World) SetBossInfo(id int, info scripting.BossInfo) {
	b := w.GrandBoss(id)
	if b == nil {
		b = w.AddGrandBoss(id, 0)
	}
	b.Info = info
	b.Saves++
}

// LiveBoss returns the registered boss if it is still in the world.
func (w *World) LiveBoss(id int) scripting.NpcRef {
	if b := w.GrandBoss(id); b != nil && b.Live != nil && !b.Live.Deleted {
		if _, ok := w.npcs[b.Live.ObjectID]; ok {
			return b.Live.Ref()
		}
	}
	return nil
}

// AddBoss registers the live boss of its template.
func (w *World) AddBoss(npc scripting.NpcRef) {
	n := npcOf(npc)
	if n == nil {
		return
	}
	b := w.GrandBoss(n.ID)
	if b == nil {
		b = w.AddGrandBoss(n.ID, 0)
	}
	b.Live = n
}

// --- Cinematics ---

// Cinematic is one cinematics call: Kind is camera, earthquake, sound or
// screen; To the audience.
type Cinematic struct {
	Kind   string
	To     scripting.Audience
	Target int32
	Camera scripting.Camera
	Loc    scripting.Location
	Text   string
	Value  int // intensity, position
	Time   int // duration, ms
}

// Cinematics returns the cinematics shown so far.
func (w *World) Cinematics() []Cinematic { return slices.Clone(w.cinematics) }

func (w *World) cine(c Cinematic) { w.cinematics = append(w.cinematics, c) }

// SpecialCamera records a camera.
func (w *World) SpecialCamera(to scripting.Audience, target any, cam scripting.Camera) {
	w.cine(Cinematic{Kind: "camera", To: to, Target: objectID(target), Camera: cam})
}

// Earthquake records an earthquake.
func (w *World) Earthquake(to scripting.Audience, loc scripting.Location, intensity, duration int) {
	w.cine(Cinematic{Kind: "earthquake", To: to, Loc: loc, Value: intensity, Time: duration})
}

// PlaySoundFrom records a sound.
func (w *World) PlaySoundFrom(to scripting.Audience, sound string, source any) {
	w.cine(Cinematic{Kind: "sound", To: to, Text: sound, Target: objectID(source)})
}

// ScreenMessage records a screen message.
func (w *World) ScreenMessage(to scripting.Audience, text string, position, ms int) {
	w.cine(Cinematic{Kind: "screen", To: to, Text: text, Value: position, Time: ms})
}

// --- Harness: the AI events ---

// SpawnEvents makes q:add_spawn and npc:add_minion run the spawn handlers
// of the new NPC, as the game does (after the handler that spawned it).
func (h *Harness) SpawnEvents(on bool) { h.world.aiState().spawnEvents = on }

func (h *Harness) call(what string, fn func(ctx context.Context) error, objs ...any) {
	h.t.Helper()
	h.world.Add(objs...)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	if err := fn(ctx); err != nil {
		h.t.Fatalf("%s handler failed: %v", what, err)
	}
}

// Spawn runs the spawn handlers of npc, which enters the world.
func (h *Harness) Spawn(npc *Npc) {
	h.t.Helper()
	h.call("spawn", func(ctx context.Context) error { return h.engine.Spawn(ctx, npc.Ref()) }, npc)
}

// AttackWith runs the attack handlers with a skill (nil for a hit) and the
// summon flag.
func (h *Harness) AttackWith(player *Player, npc *Npc, damage int, isSummon bool, skill *scripting.Skill) {
	h.t.Helper()
	h.call("attack", func(ctx context.Context) error {
		return h.engine.Attack(ctx, npc.Ref(), player.Ref(), damage, isSummon, skill)
	}, player, npc)
}

// SkillSee runs the skill-see handlers of npc: player cast skill at
// targets (*Player or *Npc).
func (h *Harness) SkillSee(player *Player, npc *Npc, skill scripting.Skill, targets ...any) {
	h.t.Helper()
	refs := make([]any, 0, len(targets))
	for _, t := range targets {
		refs = append(refs, refOf(t))
	}
	h.call("skill-see", func(ctx context.Context) error {
		return h.engine.SkillSee(ctx, npc.Ref(), player.Ref(), skill, refs, false)
	}, player, npc)
}

// MoveFinished runs the move-finished handlers of npc where it stands.
func (h *Harness) MoveFinished(npc *Npc) {
	h.t.Helper()
	h.call("move-finished", func(ctx context.Context) error { return h.engine.MoveFinished(ctx, npc.Ref()) }, npc)
}

// Arrive puts npc at the place its last move sent it (npc:move_to,
// npc:return_home), ends the move and runs its move-finished handlers. It
// fails the test if the NPC was not moving.
func (h *Harness) Arrive(npc *Npc) {
	h.t.Helper()
	if npc.Dest == nil {
		h.t.Fatalf("npc %d is not moving", npc.ObjectID)
	}
	npc.X, npc.Y, npc.Z = npc.Dest.X, npc.Dest.Y, npc.Dest.Z
	npc.Dest, npc.Moving = nil, false
	if npc.Intention == "MOVE_TO" {
		npc.Intention = "ACTIVE"
	}
	h.MoveFinished(npc)
}

// NodeArrived and RouteFinished run the walker handlers of npc.
func (h *Harness) NodeArrived(npc *Npc) {
	h.t.Helper()
	h.call("node-arrived", func(ctx context.Context) error { return h.engine.NodeArrived(ctx, npc.Ref()) }, npc)
}

func (h *Harness) RouteFinished(npc *Npc) {
	h.t.Helper()
	h.call("route-finished", func(ctx context.Context) error { return h.engine.RouteFinished(ctx, npc.Ref()) }, npc)
}

// EventReceived runs the event-received handlers of receiver.
func (h *Harness) EventReceived(event string, sender, receiver *Npc, reference any) {
	h.t.Helper()
	h.call("event-received", func(ctx context.Context) error {
		return h.engine.EventReceived(ctx, event, sender.Ref(), receiver.Ref(), refOf(reference))
	}, sender, receiver)
}

// CreatureSee runs the creature-see handlers: npc saw creature (*Player
// or *Npc).
func (h *Harness) CreatureSee(npc *Npc, creature any) {
	h.t.Helper()
	h.call("creature-see", func(ctx context.Context) error {
		return h.engine.CreatureSee(ctx, npc.Ref(), refOf(creature))
	}, npc, creature)
}

// FactionCall runs the faction-call handlers: caller called npc against
// attacker.
func (h *Harness) FactionCall(npc, caller *Npc, attacker *Player) {
	h.t.Helper()
	h.call("faction-call", func(ctx context.Context) error {
		return h.engine.FactionCall(ctx, npc.Ref(), caller.Ref(), attacker.Ref(), false)
	}, npc, caller, attacker)
}

// NpcHate asks the npc-hate handlers whether npc may hate player.
func (h *Harness) NpcHate(npc *Npc, player *Player) bool {
	h.t.Helper()
	h.world.Add(npc, player)
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	return h.engine.NpcHate(ctx, npc.Ref(), player.Ref(), false)
}

// DayNightChange runs the day-night handlers.
func (h *Harness) DayNightChange(isNight bool) {
	h.t.Helper()
	h.call("day-night", func(ctx context.Context) error { return h.engine.DayNightChange(ctx, isNight) })
}

// Despawn takes npc out of the world (it decayed), as the game does.
func (h *Harness) Despawn(npc *Npc) {
	npc.Deleted = true
	delete(h.world.npcs, npc.ObjectID)
	h.engine.NpcGone(npc.ObjectID)
}
