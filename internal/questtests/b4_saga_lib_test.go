package questtests

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Tests of scripts/lib/saga.lua, Mobius AbstractSagaQuest: the data
// checks, the registration of every saga, and the spawn list with its
// timers, kills, attacks and skills.

// Every saga loads and registers what AbstractSagaQuest.registerNPCs does.
func TestB4SagaRegistration(t *testing.T) {
	h := b4Load(t, b4Sagas...)
	e := h.Engine()
	for i, name := range b4Sagas {
		info, ok := e.QuestByID(70 + i)
		if !ok || info.Name != name {
			t.Errorf("QuestByID(%d) = %+v, %v; want %s", 70+i, info, ok, name)
		}
	}
	info, _ := e.QuestByID(70)
	// Not the Ice Crystal (items[1]) and the Divine Stone (items[3]).
	wantItems := []int{b4Item2, b4HalishaMark, b4Item5, b4Item6, b4Item7, b4Item8, b4Item9, b4Item10, b4Item11, b4Item12}
	if !slices.Equal(info.QuestItems, wantItems) {
		t.Errorf("Q070 quest items = %v, want %v", info.QuestItems, wantItems)
	}
	if info.Description != "Saga of the Phoenix Knight" {
		t.Errorf("Q070 description = %q", info.Description)
	}
	// Sedrick starts the Phoenix Knight and the Duelist sagas.
	if got := e.StartQuests(b4Sedrick); !slices.Equal(got, []string{b4Q070, "Q00073_SagaOfTheDuelist"}) {
		t.Errorf("StartQuests(Sedrick) = %v", got)
	}
	// The twelve NPCs of a saga are its talk NPCs; Tablets of Vision are
	// shared by all of them.
	for _, id := range []int{b4Sedrick, b4Npc2, b4Npc3, b4Npc5, b4Tablet1, b4Tablet6} {
		if got := e.TalkScripts(id); !slices.Contains(got, b4Q070) {
			t.Errorf("TalkScripts(%d) = %v, want Q070 in it", id, got)
		}
	}
	if got := e.TalkScripts(b4Tablet1); len(got) != len(b4Sagas) {
		t.Errorf("TalkScripts(Tablet of Vision) has %d sagas, want %d", len(got), len(b4Sagas))
	}
	if !e.HasFirstTalk(b4Npc5) {
		t.Error("npc[5] has no first-talk handler")
	}
	for _, id := range []int{b4Mob1, b4Archon, b4Mob3, b4Minion, 21651, b4HalishaNm, 18218, b4Guardian, 27216} {
		if !e.HasKill(id) {
			t.Errorf("HasKill(%d) = false", id)
		}
	}
	for _, id := range []int{b4Archon, b4Mob3} {
		if !e.HasAttack(id) {
			t.Errorf("HasAttack(%d) = false", id)
		}
	}
	if e.HasAttack(b4Mob1) {
		t.Error("mob[1] has an attack handler")
	}
}

// b4SagaSource is the Lua of a saga file like Q070's with fields replaced
// (field name -> Lua value).
func b4SagaSource(replace map[string]string) string {
	fields := []struct{ name, value string }{
		{"id", "70"},
		{"name", `"Q00070_SagaOfThePhoenixKnight"`},
		{"description", `"Saga of the Phoenix Knight"`},
		{"npc", "{30849, 31624, 31277, 30849, 31631, 31646, 31647, 31650, 31654, 31655, 31657, 31277}"},
		{"items", "{7080, 7534, 7081, 7485, 7268, 7299, 7330, 7361, 7392, 7423, 7093, 0}"},
		{"mob", "{27286, 27219, 27278}"},
		{"class_id", "{90}"},
		{"prev_class", "{5}"},
		{"x", "{191046, 46087, 46066}"},
		{"y", "{-40640, -36372, -36396}"},
		{"z", "{-3042, -1685, -1685}"},
		{"text", `{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18"}`},
	}
	var b strings.Builder
	b.WriteString("local saga = require(\"lib.saga\")\nsaga.define({\n")
	for _, f := range fields {
		v := f.value
		if r, ok := replace[f.name]; ok {
			v = r
		}
		if v != "" {
			b.WriteString("  " + f.name + " = " + v + ",\n")
		}
	}
	b.WriteString("})\n")
	return b.String()
}

// saga.define refuses data AbstractSagaQuest could not use, at the saga
// file's line.
func TestB4SagaDefineErrors(t *testing.T) {
	var libs = map[string]string{}
	for _, name := range []string{"lib/saga.lua", "lib/java.lua"} {
		src, err := os.ReadFile(filepath.Join(b4ScriptsDir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		libs[name] = string(src)
	}
	// The data itself loads.
	files := map[string]string{"quests/QX/QX.lua": b4SagaSource(nil)}
	for k, v := range libs {
		files[k] = v
	}
	scripttest.LoadSource(t, files)

	for _, tt := range []struct {
		replace map[string]string
		want    string
	}{
		{map[string]string{"id": "101"}, "id must be a saga quest ID"},
		{map[string]string{"id": ""}, "id must be a saga quest ID"},
		{map[string]string{"description": ""}, "name and description must be strings"},
		{map[string]string{"npc": "{30849, 31624}"}, "npc must have 12 entries, has 2"},
		{map[string]string{"npc": "{30849, 31624, 31277, 30849, 0, 31646, 31647, 31650, 31654, 31655, 31657, 31277}"}, "npc[5] must be a non-zero integer"},
		{map[string]string{"items": "{}"}, "items must have 12 entries"},
		{map[string]string{"mob": "27286"}, "mob must be a list"},
		{map[string]string{"class_id": "{}"}, "class_id must have 1 to 2 entries, has 0"},
		{map[string]string{"z": `{1, 2, "3"}`}, "z[3] must be an integer"},
		{map[string]string{"text": `{"a"}`}, "text must have 18 entries, has 1"},
	} {
		files := map[string]string{"quests/QX/QX.lua": b4SagaSource(tt.replace)}
		for k, v := range libs {
			files[k] = v
		}
		dir := scripttest.WriteScripts(t, files)
		e, err := scripting.New(t.Context(), scripting.Options{Dir: dir})
		if err == nil {
			e.Close()
			t.Errorf("%v: loaded, want error %q", tt.replace, tt.want)
			continue
		}
		if msg := err.Error(); !strings.Contains(msg, tt.want) || !strings.Contains(msg, "QX.lua:2") {
			t.Errorf("%v: error %q, want %q at QX.lua:2", tt.replace, msg, tt.want)
		}
	}
}

// The first spawn (step 7): Mobius's timers, the despawn, and a kill by
// someone else.
func TestB4SagaMob1(t *testing.T) {
	h := b4Load(t, b4Q070)
	w := h.World()
	tablet := &scripttest.Npc{ObjectID: 0x60000001, ID: b4Tablet3}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 76, ClassID: b4Paladin}
	b4Start(t, h, hero, 8)
	b4Var(t, "before", hero, b4Q070, "spawned", "")

	reply, ok := h.Event(b4Q070, "7-1", hero, tablet)
	b4Page(t, "spawn", reply, ok, "7-02.htm")
	mobs := b4Spawned(h, b4Mob1)
	if len(mobs) != 1 || mobs[0].X != 191046 || mobs[0].Y != -40640 || mobs[0].Z != -3042 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	b4Var(t, "spawned", hero, b4Q070, "spawned", "1")
	reply, ok = h.Event(b4Q070, "7-1", hero, tablet)
	b4Page(t, "already there", reply, ok, "7-03.htm")
	hero.Quest(b4Q070).Set("spawned", "2")
	reply, ok = h.Event(b4Q070, "7-1", hero, tablet)
	b4Page(t, "spawned 2", reply, ok, "7-04.htm")
	hero.Quest(b4Q070).Set("spawned", "1")

	w.ClearRecords()
	h.Advance(500 * time.Millisecond)
	b4WantSaid(t, "timer 1", h, strings.ReplaceAll(b4Text1, "PLAYERNAME", "Hero"))

	// After five minutes the monster leaves, and the tablet spawns it again.
	w.ClearRecords()
	h.Advance(300 * time.Second)
	b4WantSaid(t, "despawn", h, b4Text2)
	if !mobs[0].Deleted {
		t.Error("the monster is still there")
	}
	b4Var(t, "despawned", hero, b4Q070, "spawned", "0")
	b4NoTimerErrors(t, h)
	reply, ok = h.Event(b4Q070, "7-1", hero, tablet)
	b4Page(t, "again", reply, ok, "7-02.htm")
	mob := b4Last(t, h, b4Mob1)
	if mob == mobs[0] {
		t.Fatal("no new monster")
	}

	// Another Paladin on the same step kills it: no item for anyone, the
	// monster goes, and the owner may spawn it again.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 76, ClassID: b4Paladin}
	b4Start(t, h, other, 8)
	h.Advance(time.Second) // Mob_1 Timer 1
	w.ClearRecords()
	h.Kill(other, mob)
	b4WantSaid(t, "other kill", h)
	if !mob.Deleted {
		t.Error("the monster is still there after the kill")
	}
	b4Items(t, "other kill", other, map[int]int64{b4Item7: 0})
	b4Items(t, "other kill", hero, map[int]int64{b4Item7: 0})
	b4Var(t, "other kill", hero, b4Q070, "spawned", "0")
	b4Quest(t, "other kill", hero, b4Q070, scripting.StateStarted, 8)
	// The despawn timer went with it.
	w.ClearRecords()
	h.Advance(301 * time.Second)
	b4WantSaid(t, "no despawn", h)
	if w.Pending() != 0 {
		t.Errorf("%d timers left", w.Pending())
	}
}

// Marks of Halisha (step 15): random marks below 700, then Archon of
// Halisha comes for the player; its timer, attacks and skills of others.
func TestB4SagaArchon(t *testing.T) {
	h := b4Load(t, b4Q070)
	w := h.World()
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 76, ClassID: b4Paladin, X: 100, Y: 200, Z: 300}
	minion := &scripttest.Npc{ObjectID: 0x60000010, ID: b4Minion}
	b4Start(t, h, hero, 15)

	// Another class gets nothing.
	mage := &scripttest.Player{ObjectID: 0x10000003, Name: "Mage", Level: 76, ClassID: 12}
	h.SetRand(rnd.NewScripted(t))
	h.Kill(mage, minion)

	h.SetRand(rnd.NewScripted(t, rnd.Between(1, 4, 3)))
	h.Kill(hero, minion)
	b4Items(t, "one mark", hero, map[int]int64{b4HalishaMark: 3})

	hero.Give(b4HalishaMark, 697)
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, minion)
	b4Items(t, "700 marks", hero, map[int]int64{b4HalishaMark: 680})
	archons := b4Spawned(h, b4Archon)
	if len(archons) != 1 || archons[0].X != 100 || archons[0].Y != 200 || archons[0].Z != 300 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	b4WantSaid(t, "archon", h, b4Text14)
	b4Var(t, "archon", hero, b4Q070, "spawned", "1")
	// While it is there, minions give nothing.
	h.Kill(hero, minion)
	b4Items(t, "archon there", hero, map[int]int64{b4HalishaMark: 680})

	// It leaves after ten minutes.
	w.ClearRecords()
	h.Advance(600 * time.Second)
	b4WantSaid(t, "archon leaves", h, b4Text7)
	if !archons[0].Deleted {
		t.Error("the archon is still there")
	}
	b4Var(t, "archon leaves", hero, b4Q070, "spawned", "0")

	// Someone else hitting it makes it leave.
	hero.Give(b4HalishaMark, 20)
	h.Kill(hero, minion)
	archon := b4Last(t, h, b4Archon)
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 76, ClassID: b4Paladin}
	w.ClearRecords()
	h.Attack(other, archon, 10)
	b4WantSaid(t, "attacked by another", h, b4Text6)
	if !archon.Deleted {
		t.Error("the archon stays after another player's attack")
	}
	// The owner's own attacks do nothing.
	hero.Give(b4HalishaMark, 20)
	h.Kill(hero, minion)
	archon = b4Last(t, h, b4Archon)
	w.ClearRecords()
	h.Attack(hero, archon, 10)
	b4WantSaid(t, "attacked by the owner", h)

	// Someone else's skill on the owner makes it leave too.
	w.Add(other)
	w.ClearRecords()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Engine().SkillSee(ctx, archon.Ref(), other.Ref(), scripting.Skill{ID: 1011, Level: 1}, []any{hero.Ref()}, false); err != nil {
		t.Fatal(err)
	}
	b4WantSaid(t, "skill on the owner", h, b4Text6)
	if !archon.Deleted {
		t.Error("the archon stays after another player's skill")
	}

	// Killing it gives items[9] and takes the marks.
	hero.Give(b4HalishaMark, 20)
	h.Kill(hero, minion)
	archon = b4Last(t, h, b4Archon)
	w.ClearRecords()
	h.Kill(hero, archon)
	b4WantSaid(t, "archon dies", h, b4Text5)
	b4Items(t, "archon dies", hero, map[int]int64{b4HalishaMark: 0, b4Item9: 1})
	b4Quest(t, "archon dies", hero, b4Q070, scripting.StateStarted, 16)
	if !archon.Deleted {
		t.Error("the dead archon was not deleted")
	}
	b4NoTimerErrors(t, h)

	// The archons of Mobius's normal Halisha (raid minions) end step 15
	// too.
	other2 := &scripttest.Player{ObjectID: 0x10000004, Name: "Third", Level: 76, ClassID: b4Paladin}
	b4Start(t, h, other2, 15)
	other2.Give(b4HalishaMark, 5)
	h.Kill(other2, &scripttest.Npc{ObjectID: 0x60000020, ID: b4HalishaNm})
	b4Items(t, "Halisha", other2, map[int]int64{b4HalishaMark: 0, b4Item9: 1})
	b4Quest(t, "Halisha", other2, b4Q070, scripting.StateStarted, 16)
}

// Guardian Angels (step 6): the tenth kill gives the item.
func TestB4SagaGuardianAngels(t *testing.T) {
	h := b4Load(t, b4Q070)
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 76, ClassID: b4Paladin}
	angel := &scripttest.Npc{ObjectID: 0x60000010, ID: b4Guardian}
	b4Start(t, h, hero, 6)
	for range 9 {
		h.Kill(hero, angel)
	}
	b4Var(t, "nine", hero, b4Q070, "kills", "9")
	b4Quest(t, "nine", hero, b4Q070, scripting.StateStarted, 6)
	h.Kill(hero, angel)
	b4Items(t, "ten", hero, map[int]int64{b4Item6: 1})
	b4Quest(t, "ten", hero, b4Q070, scripting.StateStarted, 7)
}
