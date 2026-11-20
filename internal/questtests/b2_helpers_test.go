package questtests

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of the batch 2 walkthroughs (class change quests and the village
// masters). Every name starts with b2: other batches share the package.

// b2ScriptsDir and b2DataDir are the scripts and data folders, relative to
// this package.
const (
	b2ScriptsDir = "../../scripts"
	b2DataDir    = "../../data"
)

// Sounds the batch 2 quests play.
const (
	b2Accept  = "ItemSound.quest_accept"
	b2Middle  = "ItemSound.quest_middle"
	b2Finish  = "ItemSound.quest_finish"
	b2ItemGet = "ItemSound.quest_itemget"
	b2Giveup  = "ItemSound.quest_giveup"
	b2Fanfare = "ItemSound.quest_fanfare_2"
)

// b2Load loads one quest script, quests/<name>/<name>.lua.
func b2Load(t *testing.T, name string) *scripttest.Harness {
	t.Helper()
	return scripttest.Load(t, b2ScriptsDir, "quests/"+name+"/"+name+".lua")
}

// b2Rand forces the next random draws; the test fails if the script draws
// anything else or leaves draws unused.
func b2Rand(t *testing.T, h *scripttest.Harness, draws ...rnd.Draw) {
	t.Helper()
	h.SetRand(rnd.NewScripted(t, draws...))
}

// b2Page checks that a talk or event answered with a page.
func b2Page(t *testing.T, what string, reply scripting.Reply, ok bool, page string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyPage || reply.Content != page {
		t.Fatalf("%s: reply = %+v, %v; want page %s", what, reply, ok, page)
	}
}

// b2HTML checks that a talk or event answered with inline HTML.
func b2HTML(t *testing.T, what string, reply scripting.Reply, ok bool, html string) {
	t.Helper()
	if !ok || reply.Kind != scripting.ReplyHTML || reply.Content != html {
		t.Fatalf("%s: reply = %+v, %v; want %q", what, reply, ok, html)
	}
}

// b2NoQuestMsg checks the default "no quest" answer.
func b2NoQuestMsg(t *testing.T, what string, reply scripting.Reply, ok bool) {
	t.Helper()
	b2HTML(t, what, reply, ok, scripting.DefaultNoQuestMsg)
}

// b2Completed checks the default "already completed" answer.
func b2Completed(t *testing.T, what string, reply scripting.Reply, ok bool) {
	t.Helper()
	b2HTML(t, what, reply, ok, scripting.DefaultAlreadyCompletedMsg)
}

// b2Quest checks a quest state's state and cond.
func b2Quest(t *testing.T, what string, p *scripttest.Player, quest string, state, cond int) {
	t.Helper()
	st := p.Quest(quest)
	if st == nil {
		t.Fatalf("%s: no %s state", what, quest)
	}
	if st.State() != state || st.Cond() != cond {
		t.Fatalf("%s: state %d cond %d, want %d and %d (vars %v)", what, st.State(), st.Cond(), state, cond, st.Vars())
	}
}

// b2NoQuest checks that the player has no state of the quest
// (exit_quest(true)).
func b2NoQuest(t *testing.T, what string, p *scripttest.Player, quest string) {
	t.Helper()
	if st := p.Quest(quest); st != nil {
		t.Fatalf("%s: %s state left: state %d cond %d vars %v", what, quest, st.State(), st.Cond(), st.Vars())
	}
}

// b2Items checks item counts (0 means none).
func b2Items(t *testing.T, what string, p *scripttest.Player, counts map[int]int64) {
	t.Helper()
	for id, n := range counts {
		if got := p.Items[id]; got != n {
			t.Errorf("%s: item %d count %d, want %d", what, id, got, n)
		}
	}
}

// b2ExpSp checks the exp and SP a player got.
func b2ExpSp(t *testing.T, what string, p *scripttest.Player, exp, sp int64) {
	t.Helper()
	if p.Exp != exp || p.Sp != sp {
		t.Errorf("%s: exp, sp = %d, %d; want %d, %d", what, p.Exp, p.Sp, exp, sp)
	}
}

// b2Sounds checks the sounds a player heard since the last ClearRecords.
func b2Sounds(t *testing.T, what string, h *scripttest.Harness, p *scripttest.Player, want ...string) {
	t.Helper()
	if got := h.World().SoundsOf(p); !slices.Equal(got, want) {
		t.Errorf("%s: sounds = %v, want %v", what, got, want)
	}
}

// ----- Village masters -----
//
// The village masters change classes (player:set_class_id), which the
// scripttest fake players cannot do, and ask for the class categories
// (player:is_in_category), which need the Category constants and the
// category data. b2Masters runs them in an engine of its own over the same
// scripttest.Player values, so a test can play a quest in a
// scripttest.Harness and then change the class with the same player.

// b2Masters is an engine with village master scripts loaded.
type b2Masters struct {
	t      *testing.T
	engine *scripting.Engine
	world  *b2MasterWorld
}

// b2LoadMasters loads village master scripts by folder name, e.g.
// "ElfHumanFighterChange1" for village_master/ElfHumanFighterChange1/.
func b2LoadMasters(t *testing.T, names ...string) *b2Masters {
	t.Helper()
	files := make([]string, len(names))
	for i, n := range names {
		files[i] = "village_master/" + n + "/" + n + ".lua"
	}
	cats, err := data.LoadCategories(b2DataDir)
	if err != nil {
		t.Fatal(err)
	}
	consts, err := data.LoadConstants(b2DataDir)
	if err != nil {
		t.Fatal(err)
	}
	c := scripttest.Constants()
	c.Tables = make(map[string]map[string]any)
	for _, tb := range consts.Tables {
		m := make(map[string]any, tb.Len())
		for k, v := range tb.Ints {
			m[k] = v
		}
		for k, v := range tb.Strings {
			m[k] = v
		}
		c.Tables[tb.Name] = m
	}
	w := &b2MasterWorld{cats: cats, now: time.Unix(0, 0)}
	e, err := scripting.New(t.Context(), scripting.Options{
		Dir:       b2ScriptsDir,
		Include:   func(path string) bool { return slices.Contains(files, path) },
		World:     w,
		Rand:      rnd.New(1),
		Constants: c,
	})
	if err != nil {
		t.Fatalf("loading the village masters: %v", err)
	}
	t.Cleanup(e.Close)
	return &b2Masters{t: t, engine: e, world: w}
}

// Talk is the "Script <script>" link at npc.
func (m *b2Masters) Talk(script string, p *scripttest.Player, npc *scripttest.Npc) (scripting.Reply, bool) {
	m.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reply, ok := m.engine.Talk(ctx, script, npc.ID, m.world.ref(p), npc.Ref())
	if reply.Err != nil {
		m.t.Fatalf("talk %s: %v", script, reply.Err)
	}
	return reply, ok
}

// Event is the "Script <script> <event>" link at npc.
func (m *b2Masters) Event(script, event string, p *scripttest.Player, npc *scripttest.Npc) (scripting.Reply, bool) {
	m.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	reply, ok := m.engine.Event(ctx, script, event, m.world.ref(p), npc.Ref())
	if reply.Err != nil {
		m.t.Fatalf("event %s %s: %v", script, event, reply.Err)
	}
	return reply, ok
}

// b2ClassChange is one player:set_class_id call.
type b2ClassChange struct {
	Player int32
	Class  int
}

// b2MasterWorld is the scripting.QuestWorld of b2Masters: items, exp and
// sounds of the scripttest players; no quest states (village masters keep
// none), no pages, no timers.
type b2MasterWorld struct {
	cats *data.Categories
	now  time.Time

	// Records.
	ClassChanges   []b2ClassChange
	BaseClasses    []b2ClassChange
	UserInfos      int
	Sounds         []string
	playersByRefID map[int32]*scripttest.Player
}

func (w *b2MasterWorld) ref(p *scripttest.Player) scripting.PlayerRef {
	if w.playersByRefID == nil {
		w.playersByRefID = make(map[int32]*scripttest.Player)
	}
	w.playersByRefID[p.ObjectID] = p
	return b2MasterPlayer{p: p, w: w}
}

func b2PlayerOf(ref scripting.PlayerRef) *scripttest.Player {
	if r, ok := ref.(b2MasterPlayer); ok {
		return r.p
	}
	return nil
}

func (w *b2MasterWorld) QuestState(scripting.PlayerRef, string) scripting.QuestState {
	return nil
}

func (w *b2MasterWorld) NewQuestState(scripting.PlayerRef, string) scripting.QuestState {
	return nil
}

func (w *b2MasterWorld) Now() time.Time {
	return w.now
}

func (w *b2MasterWorld) After(time.Duration, func()) func() {
	return func() {}
}

func (w *b2MasterWorld) Every(time.Duration, func()) func() {
	return func() {}
}

func (w *b2MasterWorld) ItemCount(ref scripting.PlayerRef, itemID int) int64 {
	if p := b2PlayerOf(ref); p != nil {
		return p.Items[itemID]
	}
	return 0
}

func (w *b2MasterWorld) EnchantLevel(scripting.PlayerRef, int) int { return 0 }

func (w *b2MasterWorld) EquippedItemID(ref scripting.PlayerRef, slot int) int {
	if p := b2PlayerOf(ref); p != nil {
		return p.Equipped[slot]
	}
	return 0
}

func (w *b2MasterWorld) GiveItems(ref scripting.PlayerRef, itemID int, count int64, _ int) {
	if p := b2PlayerOf(ref); p != nil {
		p.Give(itemID, count)
	}
}

func (w *b2MasterWorld) RewardItems(ref scripting.PlayerRef, itemID int, count int64) {
	w.GiveItems(ref, itemID, count, 0)
}

func (w *b2MasterWorld) AddItem(ref scripting.PlayerRef, itemID int, count int64, _ scripting.NpcRef) bool {
	w.GiveItems(ref, itemID, count, 0)
	return true
}

func (w *b2MasterWorld) CanAddItem(scripting.PlayerRef, int) bool { return true }

func (w *b2MasterWorld) TakeItems(ref scripting.PlayerRef, itemID int, count int64) {
	p := b2PlayerOf(ref)
	if p == nil {
		return
	}
	if have := p.Items[itemID]; count < 0 || count >= have {
		delete(p.Items, itemID)
		for slot, id := range p.Equipped {
			if id == itemID {
				delete(p.Equipped, slot)
			}
		}
		return
	}
	p.Items[itemID] -= count
}

func (w *b2MasterWorld) AddExpAndSp(ref scripting.PlayerRef, exp, sp int64) {
	if p := b2PlayerOf(ref); p != nil {
		p.Exp += exp
		p.Sp += sp
	}
}

func (w *b2MasterWorld) PlaySound(_ scripting.PlayerRef, sound string) {
	w.Sounds = append(w.Sounds, sound)
}

func (w *b2MasterWorld) ShowReply(scripting.PlayerRef, scripting.NpcRef, scripting.Reply) {}

func (w *b2MasterWorld) HTML(scripting.PlayerRef, string) (string, bool) {
	return "", false
}

func (w *b2MasterWorld) FindObject(id int32) any {
	if p, ok := w.playersByRefID[id]; ok {
		return w.ref(p)
	}
	return nil
}

func (w *b2MasterWorld) GlobalVar(string) (string, bool) { return "", false }
func (w *b2MasterWorld) SetGlobalVar(string, string)     {}

// b2MasterPlayer is a scripttest.Player as b2Masters' engine sees it, with
// the class change and the class categories of the real game.
type b2MasterPlayer struct {
	p *scripttest.Player
	w *b2MasterWorld
}

var (
	_ scripting.PlayerItems   = b2MasterPlayer{}
	_ scripting.PlayerDetails = b2MasterPlayer{}
	_ scripting.PlayerVars    = b2MasterPlayer{}
	_ scripting.PlayerClasses = b2MasterPlayer{}
	_ scripting.Located       = b2MasterPlayer{}
)

func (r b2MasterPlayer) ObjectID() int32 { return r.p.ObjectID }
func (r b2MasterPlayer) Name() string    { return r.p.Name }
func (r b2MasterPlayer) Level() int      { return r.p.Level }
func (r b2MasterPlayer) ClassID() int    { return r.p.ClassID }

// Race is the race of the class, as for real players.
func (r b2MasterPlayer) Race() int { return int(data.PlayerClass(r.p.ClassID).Race()) } //nolint:gosec // class IDs are small

func (r b2MasterPlayer) Sex() int         { return r.p.Sex }
func (r b2MasterPlayer) BaseClassID() int { return r.p.BaseClass }

// IsMageClass is the class's, as for real players.
func (r b2MasterPlayer) IsMageClass() bool { return data.PlayerClass(r.p.ClassID).IsMage() } //nolint:gosec // class IDs are small

func (r b2MasterPlayer) IsNewbie() bool { return r.p.Newbie }
func (r b2MasterPlayer) IsDead() bool   { return r.p.Dead }

// InCategory tests the class against the category data, as the game does.
func (r b2MasterPlayer) InCategory(c int) bool {
	return r.w.cats.InCategory(data.CategoryType(c), r.p.ClassID) //nolint:gosec // categories are small
}

func (r b2MasterPlayer) Location() scripting.Location {
	return scripting.Location{X: r.p.X, Y: r.p.Y, Z: r.p.Z, Heading: r.p.Heading}
}

func (r b2MasterPlayer) Var(name string) (string, bool) {
	v, ok := r.p.Vars[name]
	return v, ok
}

func (r b2MasterPlayer) SetVar(name, value string) {
	if r.p.Vars == nil {
		r.p.Vars = make(map[string]string)
	}
	r.p.Vars[name] = value
}

func (r b2MasterPlayer) Adena() int64               { return r.p.Items[57] }
func (r b2MasterPlayer) ItemCount(itemID int) int64 { return r.p.Items[itemID] }

func (r b2MasterPlayer) GiveItems(itemID int, count int64) bool {
	r.p.Give(itemID, count)
	return true
}

func (r b2MasterPlayer) TakeItems(itemID int, count int64) bool {
	have := r.p.Items[itemID]
	r.w.TakeItems(r, itemID, count)
	return count < 0 || count <= have
}

// SetClassID is the class change: the class and the record.
func (r b2MasterPlayer) SetClassID(id int) {
	r.p.ClassID = id
	r.w.ClassChanges = append(r.w.ClassChanges, b2ClassChange{Player: r.p.ObjectID, Class: id})
}

func (r b2MasterPlayer) SetBaseClassID(id int) {
	r.p.BaseClass = id
	r.w.BaseClasses = append(r.w.BaseClasses, b2ClassChange{Player: r.p.ObjectID, Class: id})
}

func (r b2MasterPlayer) BroadcastUserInfo() { r.w.UserInfos++ }

// b2ClassChanged checks that a master changed p's class (class and base
// class) to class once, with the user info broadcast, and clears the
// records.
func (m *b2Masters) b2ClassChanged(what string, p *scripttest.Player, class int) {
	m.t.Helper()
	want := []b2ClassChange{{Player: p.ObjectID, Class: class}}
	if !slices.Equal(m.world.ClassChanges, want) || !slices.Equal(m.world.BaseClasses, want) || m.world.UserInfos != 1 || p.ClassID != class {
		m.t.Errorf("%s: class changes %v, base classes %v, user infos %d, class %d; want class %d once",
			what, m.world.ClassChanges, m.world.BaseClasses, m.world.UserInfos, p.ClassID, class)
	}
	m.b2Clear()
}

// b2NoClassChange checks that no class changed.
func (m *b2Masters) b2NoClassChange(what string) {
	m.t.Helper()
	if len(m.world.ClassChanges) != 0 || len(m.world.BaseClasses) != 0 || m.world.UserInfos != 0 {
		m.t.Errorf("%s: class changes %v, base classes %v, user infos %d; want none",
			what, m.world.ClassChanges, m.world.BaseClasses, m.world.UserInfos)
	}
}

// b2Clear forgets the records.
func (m *b2Masters) b2Clear() {
	m.world.ClassChanges = nil
	m.world.BaseClasses = nil
	m.world.UserInfos = 0
	m.world.Sounds = nil
}
