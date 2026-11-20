package scripttest

import (
	"maps"
	"slices"
	"sort"
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
	// QuestLists counts the QuestList packets per player object ID;
	// QuestMarks lists the quests ExShowQuestMark was sent for.
	QuestLists map[int32]int
	QuestMarks []string

	// Custom lists the quests set_custom(true) made custom, as the game
	// tells its model.
	Custom map[string]bool
}

var (
	_ scripting.QuestWorld   = (*World)(nil)
	_ scripting.QuestActions = (*World)(nil)
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

// Action is one thing QuestActions did, e.g. {Kind: "social", Object: id,
// Value: 3}.
type Action struct {
	Kind   string // social, teleport, radar, tutorial, chat_window, delete, say
	Object int32
	Value  int
	Text   string
	Loc    scripting.Location
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
	w.Sounds, w.Replies, w.Actions, w.Messages, w.Spawned, w.QuestMarks = nil, nil, nil, nil, nil, nil
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

// GiveItems adds items.
func (w *World) GiveItems(p scripting.PlayerRef, itemID int, count int64, _ int) {
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
}

// AddSpawn makes a new fake NPC.
func (w *World) AddSpawn(npcID int, loc scripting.Location, _ bool, _ time.Duration, _ bool) scripting.NpcRef {
	w.nextID++
	n := &Npc{ObjectID: w.nextID, ID: npcID, X: loc.X, Y: loc.Y, Z: loc.Z, Heading: loc.Heading}
	w.npcs[n.ObjectID] = n
	w.Spawned = append(w.Spawned, n)
	return n.Ref()
}
