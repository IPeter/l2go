package scripttest

import (
	"maps"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
)

// Start is the fake clock's start: a Monday morning, server time UTC.
var Start = time.Date(2026, 11, 16, 5, 0, 0, 0, time.UTC)

// World is a fake game world for quest scripts: it implements
// scripting.QuestWorld and scripting.QuestActions over the fake players and
// NPCs, with a fake clock, and records what scripts did. The harness makes
// one per test; get it with Harness.World.
type World struct {
	now    time.Time
	timers []*fakeTimer
	seq    uint64

	players map[int32]*Player
	npcs    map[int32]*Npc
	globals map[string]string
	pages   map[string]string
	nextID  int32

	// What scripts did, in order.
	Sounds   []Sound
	Replies  []ShownReply
	Actions  []Action
	Messages []Message
	Spawned  []*Npc
	// Given lists the items given (give_items, reward_items), one entry
	// per call.
	Given []Give
	// QuestLists counts the QuestList packets per player object ID;
	// QuestMarks lists the quests ExShowQuestMark was sent for.
	QuestLists map[int32]int
	QuestMarks []string

	// Custom lists the quests set_custom(true) made custom, as the game
	// tells its model.
	Custom map[string]bool

	// ItemNames are the item names items.get returns; an item without one
	// is "Item <id>". Every item ID exists.
	ItemNames map[int]string
	// StackableItems lists the item IDs items.get calls stackable.
	StackableItems []int
	// MissingSkills lists the skill IDs skills.get finds no level of; every
	// other skill has every level.
	MissingSkills []int

	// The AI world (ai.go).
	ai         aiWorld
	cinematics []Cinematic
	// areasB is the state of the second half of the area scripts (areas_b.go).
	areasB areasBWorld
	// longEvents are the seasonal events (longevents.go).
	longEvents longEventsWorld
	// heroes are the hero table (hero.go).
	heroes heroWorld
	// Manor is the castle manor (manor.go).
	Manor ManorState
	// SevenSigns is the Seven Signs state (sevensigns.go).
	SevenSigns SevenSignsState
	// Halls and HallClans are the clan halls and the clans the hall scripts
	// read (clanhall.go).
	Halls     map[int]*ClanHall
	HallClans map[int32]scripting.ClanInfo
	// effectPoints and castRanges are the skill data of boss.go.
	effectPoints map[int]int
	castRanges   map[int]int
	// onGone tells the engine an NPC left the world (the game's NpcGone);
	// onSpawn runs the spawn handlers of a new NPC.
	onGone  func(objectID int32)
	onSpawn func(npc scripting.NpcRef)
}

var (
	_ scripting.QuestWorld   = (*World)(nil)
	_ scripting.QuestActions = (*World)(nil)
	_ scripting.NpcActions   = (*World)(nil)
	_ scripting.QuestSkills  = (*World)(nil)
	_ scripting.QuestItems   = (*World)(nil)
)

// NewWorld returns an empty world at Start.
func NewWorld() *World {
	return &World{
		now:        Start,
		players:    make(map[int32]*Player),
		npcs:       make(map[int32]*Npc),
		globals:    make(map[string]string),
		pages:      make(map[string]string),
		QuestLists: make(map[int32]int),
		Custom:     make(map[string]bool),
		ItemNames:  make(map[int]string),
		nextID:     0x70000000,
	}
}

// Sound is one PlaySound.
type Sound struct {
	Player int32
	Name   string
}

// ShownReply is a reply shown without a caller (timers, show_html_file).
type ShownReply struct {
	Player int32
	Npc    int32
	Reply  scripting.Reply
}

// Action is one thing QuestActions or NpcActions did, e.g. {Kind:
// "social", Object: id, Value: 3}.
//
// Kinds: social, teleport, radar, tutorial, chat_window, delete, say,
// screen; and of NpcActions (Object the NPC, Target the other object):
// attack, hate (Value the hate, Damage the damage), remove_attacker,
// abort_attack, move (Loc), run, die (Target the killer), respawn (Value 1
// on, 0 off), cast (Value the skill ID, Level its level; Target 0 for the
// current target), magic_skill_use (Object the caster, Value the skill ID,
// Level, HitTime), apply_effects (Object the effector, Value the skill ID),
// set_target (Target 0 for none), npc_teleport (Loc), drop_item (Target the
// player, Value the item ID, Count).
type Action struct {
	Kind   string
	Object int32
	Target int32
	Value  int
	Level  int
	Damage int64
	Count  int64
	Text   string
	Loc    scripting.Location
	// HitTime is the hit time of magic_skill_use.
	HitTime time.Duration
}

// Give is one item give: count items of a template to a player.
type Give struct {
	Player int32
	Item   int
	Count  int64
}

// Message is a text or system message sent to a player.
type Message struct {
	Player int32
	ID     int // the SystemMessageId; 0 for text
	Text   string
	Params []string
}

// Add puts players and NPCs into the world, so world.find_object finds
// them. The harness adds those it passes to the engine.
func (w *World) Add(objs ...any) {
	for _, o := range objs {
		switch x := o.(type) {
		case *Player:
			if x != nil {
				w.players[x.ObjectID] = x
			}
		case *Npc:
			if x != nil {
				w.npcs[x.ObjectID] = x
			}
		}
	}
}

// SetPage puts a page into the fake HTML folder (path relative to it, e.g.
// "noquest.htm").
func (w *World) SetPage(path, html string) { w.pages[path] = html }

// SoundsOf returns the sounds a player heard, in order.
func (w *World) SoundsOf(p *Player) []string {
	var out []string
	for _, s := range w.Sounds {
		if s.Player == p.ObjectID {
			out = append(out, s.Name)
		}
	}
	return out
}

// ClearRecords forgets what scripts did so far.
func (w *World) ClearRecords() {
	w.Sounds, w.Replies, w.Actions, w.Messages, w.Spawned, w.QuestMarks, w.Given = nil, nil, nil, nil, nil, nil, nil
	w.cinematics = nil
	clear(w.QuestLists)
}

// Global returns a global variable.
func (w *World) Global(name string) (string, bool) {
	v, ok := w.globals[name]
	return v, ok
}

// --- Clock and timers ---

type fakeTimer struct {
	at      time.Time
	every   time.Duration
	seq     uint64
	fn      func()
	stopped bool
}

// Now is the fake clock.
func (w *World) Now() time.Time { return w.now }

// After runs fn when the clock passes d from now.
func (w *World) After(d time.Duration, fn func()) func() { return w.add(d, 0, fn) }

// Every runs fn every d.
func (w *World) Every(d time.Duration, fn func()) func() { return w.add(d, d, fn) }

func (w *World) add(d, every time.Duration, fn func()) func() {
	w.seq++
	t := &fakeTimer{at: w.now.Add(d), every: every, seq: w.seq, fn: fn}
	w.timers = append(w.timers, t)
	return func() { t.stopped = true }
}

// Advance moves the clock by d and runs the timers that come due, in due
// order then start order, each at its own time.
func (w *World) Advance(d time.Duration) {
	end := w.now.Add(d)
	for {
		w.timers = slices.DeleteFunc(w.timers, func(t *fakeTimer) bool { return t.stopped })
		sort.SliceStable(w.timers, func(i, j int) bool {
			a, b := w.timers[i], w.timers[j]
			if !a.at.Equal(b.at) {
				return a.at.Before(b.at)
			}
			return a.seq < b.seq
		})
		if len(w.timers) == 0 || w.timers[0].at.After(end) {
			break
		}
		t := w.timers[0]
		w.now = t.at
		if t.every > 0 {
			t.at = t.at.Add(t.every)
		} else {
			t.stopped = true
		}
		t.fn()
	}
	w.now = end
}

// Pending returns the number of timers not yet run.
func (w *World) Pending() int {
	n := 0
	for _, t := range w.timers {
		if !t.stopped {
			n++
		}
	}
	return n
}

// --- Quest states ---

func playerOf(p scripting.PlayerRef) *Player {
	if r, ok := p.(playerRef); ok {
		return r.p
	}
	return nil
}

func npcOf(n scripting.NpcRef) *Npc {
	if r, ok := n.(npcRef); ok {
		return r.n
	}
	return nil
}

// QuestState returns the player's state of the quest.
func (w *World) QuestState(p scripting.PlayerRef, quest string) scripting.QuestState {
	pl := playerOf(p)
	if pl == nil || pl.quests[quest] == nil {
		return nil
	}
	return pl.quests[quest]
}

// NewQuestState creates a CREATED state.
func (w *World) NewQuestState(p scripting.PlayerRef, quest string) scripting.QuestState {
	pl := playerOf(p)
	if pl == nil {
		return nil
	}
	if pl.quests == nil {
		pl.quests = make(map[string]*QuestState)
	}
	st := &QuestState{quest: quest, owner: pl, world: w, custom: w.Custom[quest]}
	pl.quests[quest] = st
	return st
}

func (w *World) questList(p *Player) { w.QuestLists[p.ObjectID]++ }

func (w *World) questMark(_ *Player, quest string) { w.QuestMarks = append(w.QuestMarks, quest) }

// --- Items ---

// ItemCount counts the player's items.
func (w *World) ItemCount(p scripting.PlayerRef, itemID int) int64 {
	if pl := playerOf(p); pl != nil {
		return pl.Items[itemID]
	}
	return 0
}

// EnchantLevel is always 0 for fake items.
func (w *World) EnchantLevel(scripting.PlayerRef, int) int { return 0 }

// EquippedItemID returns the player's Equipped slot.
func (w *World) EquippedItemID(p scripting.PlayerRef, slot int) int {
	if pl := playerOf(p); pl != nil {
		return pl.Equipped[slot]
	}
	return 0
}

// GiveItems adds items and records the call in Given.
func (w *World) GiveItems(p scripting.PlayerRef, itemID int, count int64, _ int) {
	w.Given = append(w.Given, Give{Player: p.ObjectID(), Item: itemID, Count: count})
	if pl := playerOf(p); pl != nil {
		pl.addItem(itemID, count)
	}
}

// RewardItems adds items (rates 1).
func (w *World) RewardItems(p scripting.PlayerRef, itemID int, count int64) {
	w.GiveItems(p, itemID, count, 0)
}

// AddItem adds items.
func (w *World) AddItem(p scripting.PlayerRef, itemID int, count int64, _ scripting.NpcRef) bool {
	w.GiveItems(p, itemID, count, 0)
	return true
}

// CanAddItem is always true.
func (w *World) CanAddItem(scripting.PlayerRef, int) bool { return true }

// TakeItems takes items; equipped ones come off.
func (w *World) TakeItems(p scripting.PlayerRef, itemID int, count int64) {
	pl := playerOf(p)
	if pl == nil {
		return
	}
	have := pl.Items[itemID]
	if count < 0 || count >= have {
		delete(pl.Items, itemID)
		for slot, id := range pl.Equipped {
			if id == itemID {
				delete(pl.Equipped, slot)
			}
		}
		return
	}
	pl.Items[itemID] = have - count
}

// AddExpAndSp adds to the player's Exp and Sp.
func (w *World) AddExpAndSp(p scripting.PlayerRef, exp, sp int64) {
	if pl := playerOf(p); pl != nil {
		pl.Exp += exp
		pl.Sp += sp
	}
}

// --- Packets ---

// PlaySound records a sound.
func (w *World) PlaySound(p scripting.PlayerRef, sound string) {
	w.Sounds = append(w.Sounds, Sound{Player: p.ObjectID(), Name: sound})
}

// ShowReply records a reply.
func (w *World) ShowReply(p scripting.PlayerRef, npc scripting.NpcRef, r scripting.Reply) {
	sr := ShownReply{Player: p.ObjectID(), Reply: r}
	if npc != nil {
		sr.Npc = npc.ObjectID()
	}
	w.Replies = append(w.Replies, sr)
}

// HTML returns a page set with SetPage.
func (w *World) HTML(_ scripting.PlayerRef, path string) (string, bool) {
	html, ok := w.pages[path]
	return html, ok
}

// FindObject finds a player or NPC added to the world.
func (w *World) FindObject(id int32) any {
	if p, ok := w.players[id]; ok {
		return p.Ref()
	}
	if n, ok := w.npcs[id]; ok {
		return n.Ref()
	}
	return nil
}

// GlobalVar returns a global variable.
func (w *World) GlobalVar(name string) (string, bool) { return w.Global(name) }

// SetGlobalVar sets a global variable.
func (w *World) SetGlobalVar(name, value string) { w.globals[name] = value }

// Globals returns a copy of the global variables.
func (w *World) Globals() map[string]string { return maps.Clone(w.globals) }

// --- QuestActions ---

func objectID(obj any) int32 {
	if o, ok := obj.(interface{ ObjectID() int32 }); ok {
		return o.ObjectID()
	}
	return 0
}

func (w *World) act(a Action) { w.Actions = append(w.Actions, a) }

// SocialAction records a social action.
func (w *World) SocialAction(obj any, action int) {
	w.act(Action{Kind: "social", Object: objectID(obj), Value: action})
}

// Teleport moves the player.
func (w *World) Teleport(p scripting.PlayerRef, loc scripting.Location) {
	if pl := playerOf(p); pl != nil {
		pl.X, pl.Y, pl.Z = loc.X, loc.Y, loc.Z
	}
	w.act(Action{Kind: "teleport", Object: p.ObjectID(), Loc: loc})
}

// SendMessage records a text message.
func (w *World) SendMessage(p scripting.PlayerRef, text string) {
	w.Messages = append(w.Messages, Message{Player: p.ObjectID(), Text: text})
}

// SendSystemMessage records a system message.
func (w *World) SendSystemMessage(p scripting.PlayerRef, id int, params []string) {
	w.Messages = append(w.Messages, Message{Player: p.ObjectID(), ID: id, Params: params})
}

// ShowOnScreenMsg records a screen message.
func (w *World) ShowOnScreenMsg(p scripting.PlayerRef, text string, ms int) {
	w.act(Action{Kind: "screen", Object: p.ObjectID(), Value: ms, Text: text})
}

// Radar records a radar change.
func (w *World) Radar(p scripting.PlayerRef, op int, x, y, z int) {
	w.act(Action{Kind: "radar", Object: p.ObjectID(), Value: op, Loc: scripting.Location{X: x, Y: y, Z: z}})
}

// Tutorial records a tutorial packet.
func (w *World) Tutorial(p scripting.PlayerRef, op scripting.TutorialOp, arg string) {
	w.act(Action{Kind: "tutorial", Object: p.ObjectID(), Value: int(op), Text: arg})
}

// NpcBroadcastSay records NPC speech.
func (w *World) NpcBroadcastSay(npc scripting.NpcRef, chatType int, text string) {
	w.act(Action{Kind: "say", Object: npc.ObjectID(), Value: chatType, Text: text})
}

// ShowChatWindow records the NPC's own dialog.
func (w *World) ShowChatWindow(npc scripting.NpcRef, p scripting.PlayerRef) {
	w.act(Action{Kind: "chat_window", Object: npc.ObjectID(), Value: int(p.ObjectID())})
}

// DeleteNpc removes an NPC.
func (w *World) DeleteNpc(npc scripting.NpcRef) {
	if n := npcOf(npc); n != nil {
		n.Deleted = true
		delete(w.npcs, n.ObjectID)
	}
	w.act(Action{Kind: "delete", Object: npc.ObjectID()})
	if w.onGone != nil {
		w.onGone(npc.ObjectID())
	}
}

// AddSpawn makes a new fake NPC. A heading of -1 (random) stays -1.
func (w *World) AddSpawn(npcID int, loc scripting.Location, _ bool, _ time.Duration, _ bool) scripting.NpcRef {
	w.nextID++
	n := &Npc{ObjectID: w.nextID, ID: npcID, X: loc.X, Y: loc.Y, Z: loc.Z, Heading: loc.Heading}
	w.npcs[n.ObjectID] = n
	w.Spawned = append(w.Spawned, n)
	if w.ai.spawnEvents && w.onSpawn != nil {
		w.onSpawn(n.Ref())
	}
	return n.Ref()
}

// --- NpcActions ---

// NpcAttack records an attack order; the intention becomes ATTACK.
func (w *World) NpcAttack(npc scripting.NpcRef, target any) {
	if n := npcOf(npc); n != nil {
		n.Intention = "ATTACK"
	}
	w.act(Action{Kind: "attack", Object: npc.ObjectID(), Target: objectID(target)})
}

// NpcAddDamageHate records hate and adds it to the NPC's Hate list.
func (w *World) NpcAddDamageHate(npc scripting.NpcRef, target any, damage, hate int64) {
	if n := npcOf(npc); n != nil {
		n.addHate(fakeOf(target), damage, hate)
	}
	w.act(Action{Kind: "hate", Object: npc.ObjectID(), Target: objectID(target), Value: int(hate), Damage: damage})
}

// NpcRemoveAttacker records the removal.
func (w *World) NpcRemoveAttacker(npc scripting.NpcRef, target any) {
	w.act(Action{Kind: "remove_attacker", Object: npc.ObjectID(), Target: objectID(target)})
}

// NpcAbortAttack records the abort.
func (w *World) NpcAbortAttack(npc scripting.NpcRef) {
	w.act(Action{Kind: "abort_attack", Object: npc.ObjectID()})
}

// NpcMoveTo records the walk; the NPC stays where it is until
// Harness.Arrive.
func (w *World) NpcMoveTo(npc scripting.NpcRef, loc scripting.Location) {
	if n := npcOf(npc); n != nil {
		dest := loc
		n.Dest, n.Moving, n.Intention = &dest, true, "MOVE_TO"
	}
	w.act(Action{Kind: "move", Object: npc.ObjectID(), Loc: loc})
}

// NpcSetRunning sets Running.
func (w *World) NpcSetRunning(npc scripting.NpcRef) {
	if n := npcOf(npc); n != nil {
		n.Running = true
	}
	w.act(Action{Kind: "run", Object: npc.ObjectID()})
}

// NpcDie sets Dead. It runs no kill handlers; Harness.Kill does.
func (w *World) NpcDie(npc scripting.NpcRef, killer any) {
	if n := npcOf(npc); n != nil {
		n.Dead = true
		n.HP = 0
	}
	w.act(Action{Kind: "die", Object: npc.ObjectID(), Target: objectID(killer)})
}

// NpcSetRespawn sets Respawn.
func (w *World) NpcSetRespawn(npc scripting.NpcRef, on bool) {
	v := 0
	if n := npcOf(npc); n != nil {
		n.Respawn = on
	}
	if on {
		v = 1
	}
	w.act(Action{Kind: "respawn", Object: npc.ObjectID(), Value: v})
}

// NpcCast records the cast.
func (w *World) NpcCast(npc scripting.NpcRef, skill scripting.Skill, target any) {
	w.act(Action{Kind: "cast", Object: npc.ObjectID(), Target: objectID(target), Value: skill.ID, Level: skill.Level})
}

// MagicSkillUse records the animation.
func (w *World) MagicSkillUse(caster, target any, skillID, level int, hitTime, _ time.Duration) {
	w.act(Action{Kind: "magic_skill_use", Object: objectID(caster), Target: objectID(target), Value: skillID, Level: level, HitTime: hitTime})
}

// NpcSetTarget records the target and sets the NPC's Target.
func (w *World) NpcSetTarget(npc scripting.NpcRef, target any) {
	if n := npcOf(npc); n != nil {
		n.Target = fakeOf(target)
	}
	w.act(Action{Kind: "set_target", Object: npc.ObjectID(), Target: objectID(target)})
}

// NpcTeleport moves the NPC.
func (w *World) NpcTeleport(npc scripting.NpcRef, loc scripting.Location) {
	if n := npcOf(npc); n != nil {
		n.X, n.Y, n.Z = loc.X, loc.Y, loc.Z
	}
	w.act(Action{Kind: "npc_teleport", Object: npc.ObjectID(), Loc: loc})
}

// NpcDropItem records the drop; nothing lands anywhere.
func (w *World) NpcDropItem(npc scripting.NpcRef, p scripting.PlayerRef, itemID int, count int64) {
	w.act(Action{Kind: "drop_item", Object: npc.ObjectID(), Target: p.ObjectID(), Value: itemID, Count: count})
}

// NpcExists reports whether an NPC of the template was added to the world
// (Add, the harness calls, q:add_spawn) and not deleted.
func (w *World) NpcExists(npcID int) bool {
	for _, n := range w.npcs {
		if n.ID == npcID && !n.Deleted {
			return true
		}
	}
	return false
}

// --- QuestSkills and QuestItems ---

// SkillExists is true but for MissingSkills.
func (w *World) SkillExists(id, _ int) bool { return !slices.Contains(w.MissingSkills, id) }

// effectsOf returns the Effects list of a fake player or NPC.
func effectsOf(obj any) *[]int {
	switch x := obj.(type) {
	case playerRef:
		return &x.p.Effects
	case npcRef:
		return &x.n.Effects
	}
	return nil
}

// IsAffectedBySkill looks in the object's Effects.
func (w *World) IsAffectedBySkill(obj any, skillID int) bool {
	e := effectsOf(obj)
	return e != nil && slices.Contains(*e, skillID)
}

// ApplyEffects adds the skill to the effected's Effects and records it.
func (w *World) ApplyEffects(skill scripting.Skill, effector, effected any) {
	if e := effectsOf(effected); e != nil && !slices.Contains(*e, skill.ID) {
		*e = append(*e, skill.ID)
	}
	w.act(Action{Kind: "apply_effects", Object: objectID(effector), Target: objectID(effected), Value: skill.ID, Level: skill.Level})
}

// ItemTemplate returns the item with its ItemNames name.
func (w *World) ItemTemplate(id int) (scripting.ItemTemplate, bool) {
	name, ok := w.ItemNames[id]
	if !ok {
		name = "Item " + strconv.Itoa(id)
	}
	return scripting.ItemTemplate{ID: id, Name: name, Stackable: slices.Contains(w.StackableItems, id)}, true
}
