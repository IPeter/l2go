package scripting_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The area scripts of Plan 13 group B, first half (Mobius ai/areas
// CrumaTower to ImperialTomb), on the fake world.

// b1Actions returns the recorded actions of one kind.
func b1Actions(h *scripttest.Harness, kind string) []scripttest.Action {
	var out []scripttest.Action
	for _, a := range h.World().Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// b1Spawned returns the NPCs of a template scripts spawned so far.
func b1Spawned(h *scripttest.Harness, npcID int) []*scripttest.Npc {
	var out []*scripttest.Npc
	for _, n := range h.World().Spawned {
		if n.ID == npcID {
			out = append(out, n)
		}
	}
	return out
}

func TestCrumaTowerScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/CrumaTower/CrumaTower.lua")
	p := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 55}
	mozella := &scripttest.Npc{ObjectID: 2, ID: 30483}
	if reply, ok := h.FirstTalk(p, mozella); !ok || reply.Content != "30483.html" {
		t.Errorf("first talk = %+v", reply)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Bool(false)))
	if reply, ok := h.Event("CrumaTower", "ENTER", p, mozella); ok {
		t.Errorf("ENTER showed %+v", reply)
	}
	if p.X != 17680 || p.Y != 113968 || p.Z != -11671 {
		t.Errorf("player at %d, %d, %d; want the second spot", p.X, p.Y, p.Z)
	}
	p.Level = 56
	if reply, _ := h.Event("CrumaTower", "ENTER", p, mozella); reply.Content != "30483-1.html" {
		t.Errorf("level 56: %+v", reply)
	}
}

func TestSummonPcScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/CrumaTower/SummonPc.lua")
	far := &scripttest.Player{ObjectID: 1, Name: "Hero", X: 400}
	porta := &scripttest.Npc{ObjectID: 2, ID: 20213, X: 0, Y: 0, Z: -100}

	// From afar, 50 %: Mobius casts twice, the second marks the summon.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49)))
	h.Attack(far, porta, 10)
	want := []scripttest.Action{
		{Kind: "set_target", Object: 2, Target: 1},
		{Kind: "cast", Object: 2, Value: 4161, Level: 1},
		{Kind: "set_target", Object: 2, Target: 1},
		{Kind: "cast", Object: 2, Value: 4161, Level: 1},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("far attack =\n%+v\nwant\n%+v", got, want)
	}
	// While a summon is in flight, nothing more.
	h.World().ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Attack(far, porta, 10)
	if len(h.World().Actions) != 0 {
		t.Fatalf("second attack did %+v", h.World().Actions)
	}
	// The cast lands: the player stands at the monster.
	h.SpellFinished(porta, far, scripting.Skill{ID: 4161, Level: 1})
	if far.X != 0 || far.Z != -100 {
		t.Errorf("player at %d, %d, %d; want at the monster", far.X, far.Y, far.Z)
	}

	// Close (100 to 300): the most hated attacker at 50 %, others at 10 %.
	near := &scripttest.Player{ObjectID: 3, Name: "Near", X: 200}
	other := &scripttest.Player{ObjectID: 4, Name: "Other", X: 250}
	porta.Hate = []scripttest.Hate{{Target: near, Hate: 100}, {Target: other, Hate: 10}}
	h.World().ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 30), rnd.Get(100, 30)))
	h.Attack(other, porta, 10)
	if len(b1Actions(h, "cast")) != 0 {
		t.Fatalf("a cast at the less hated attacker at 30 %%")
	}
	h.Attack(near, porta, 10)
	if casts := b1Actions(h, "cast"); len(casts) != 1 {
		t.Fatalf("casts = %+v, want one at the most hated", casts)
	}
	// A monster that cannot cast does not.
	h.SpellFinished(porta, near, scripting.Skill{ID: 4161, Level: 1})
	porta.CannotCast = true
	h.World().ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Attack(near, porta, 10)
	if len(h.World().Actions) != 0 {
		t.Errorf("a monster that cannot cast did %+v", h.World().Actions)
	}
}

func TestTomaScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/DwarvenVillage/Toma/Toma.lua")
	h.SetRand(rnd.NewScripted(t, rnd.Get(3, 2), rnd.Get(3, 0), rnd.Get(3, 1)))
	h.Start()
	toma := b1Spawned(h, 30556)
	if len(toma) != 1 || toma[0].X != 178834 || toma[0].Heading != 41400 {
		t.Fatalf("Toma = %+v, want one at the third place", toma)
	}
	p := &scripttest.Player{ObjectID: 1}
	if reply, _ := h.FirstTalk(p, toma[0]); reply.Content != "30556.htm" {
		t.Errorf("first talk = %+v", reply)
	}

	// 30 minutes later he moves.
	h.Advance(30*time.Minute - time.Millisecond)
	if len(b1Spawned(h, 30556)) != 1 {
		t.Fatalf("moved early")
	}
	h.Advance(time.Millisecond)
	toma = b1Spawned(h, 30556)
	if len(toma) != 2 || !toma[0].Deleted || toma[1].X != 151680 {
		t.Fatalf("after 30 minutes: %+v", toma)
	}

	// A reload replaces him instead of adding a second Toma.
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	toma = b1Spawned(h, 30556)
	if len(toma) != 3 || !toma[1].Deleted || toma[2].Deleted || toma[2].X != 154153 {
		t.Errorf("after a reload: %+v", toma)
	}
}

func TestEilhalderVonHellmannScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/ForestOfTheDead/EilhalderVonHellmann.lua")
	h.DayNightChange(true)
	h.DayNightChange(true)
	boss := b1Spawned(h, 25328)
	if len(boss) != 1 || boss[0].X != 59090 || boss[0].Y != -42188 {
		t.Fatalf("night: %+v, want one boss", boss)
	}

	// Day: in combat he stays and tries again every 30 s.
	boss[0].InCombat = true
	h.World().SetGameTime(6 * 60)
	h.DayNightChange(false)
	h.Advance(30 * time.Second)
	if boss[0].Deleted {
		t.Fatalf("left while fighting")
	}
	boss[0].InCombat = false
	h.Advance(30 * time.Second)
	if !boss[0].Deleted {
		t.Fatalf("still there 30 s after the fight")
	}

	// The next night he is back; a reload during the day restarts the leave
	// timer of a boss still there.
	h.DayNightChange(true)
	boss = b1Spawned(h, 25328)
	if len(boss) != 2 {
		t.Fatalf("second night: %+v", boss)
	}
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	h.Advance(30 * time.Second)
	if !boss[1].Deleted {
		t.Errorf("the reload by day left the boss")
	}
}

func TestMagmaDrakeScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/ForgeOfTheGods/MagmaDrake.lua")
	p := &scripttest.Player{ObjectID: 1}
	drake := &scripttest.Npc{ObjectID: 2, ID: 21657, X: 100, Y: 200, Z: -300, Heading: 1000}
	h.Kill(p, drake)
	young := b1Spawned(h, 21393)
	if len(young) != 5 {
		t.Fatalf("%d drakes, want 5", len(young))
	}
	for _, n := range young {
		if n.X != 100 || n.Y != 200 || n.Z != -280 || n.Heading != 1000 || !n.Running || n.Intention != "ATTACK" ||
			len(n.Hate) != 1 || n.Hate[0].Target != p || n.Hate[0].Hate != 500 {
			t.Errorf("drake %+v", n)
		}
	}

	// A summon's kill sends them at the summon.
	p.Summon = &scripttest.Npc{ObjectID: 3, ID: 12077}
	other := &scripttest.Npc{ObjectID: 4, ID: 21657}
	h.World().Add(p, other, p.Summon)
	if err := h.Engine().Kill(context.Background(), other.Ref(), p.Ref(), true); err != nil {
		t.Fatal(err)
	}
	young = b1Spawned(h, 21393)
	if len(young) != 10 || young[9].Hate[0].Target != p.Summon {
		t.Errorf("summon kill: %d drakes, last hates %+v", len(young), young[9].Hate)
	}
}

func TestRooneyScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/ForgeOfTheGods/Rooney.lua")
	h.SetRand(rnd.NewScripted(t, rnd.Get(39, 5)))
	h.Start()
	rooney := b1Spawned(h, 32049)
	if len(rooney) != 1 || rooney[0].X != 181857 || rooney[0].Y != -109491 {
		t.Fatalf("Rooney = %+v", rooney)
	}
	r := rooney[0]
	p := &scripttest.Player{ObjectID: 1}
	h.CreatureSee(r, &scripttest.Npc{ObjectID: 3, ID: 20001})
	h.CreatureSee(r, p)
	h.CreatureSee(r, p)
	says := b1Actions(h, "say")
	if len(says) != 1 || says[0].Text != "Welcome!" || r.ScriptValue != 1 {
		t.Fatalf("greeting: %+v, value %d", says, r.ScriptValue)
	}

	// A minute per line, then she moves.
	h.Advance(3 * time.Minute)
	if r.ScriptValue != 4 || len(b1Actions(h, "say")) != 4 {
		t.Fatalf("after 3 minutes: value %d, said %+v", r.ScriptValue, b1Actions(h, "say"))
	}
	// A reload keeps her minute timer going.
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if len(b1Spawned(h, 32049)) != 1 {
		t.Fatalf("the reload spawned a second Rooney")
	}
	h.Advance(time.Minute)
	if says := b1Actions(h, "say"); len(says) != 5 || says[4].Text != "Why don't I go that way this time" {
		t.Fatalf("minute 4: %+v", says)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(39, 0)))
	h.Advance(time.Minute)
	if r.X != 175937 || r.ScriptValue != 0 || len(b1Actions(h, "npc_teleport")) != 1 {
		t.Errorf("after 5 minutes: at %d, value %d", r.X, r.ScriptValue)
	}
}

func TestFrozenLabyrinthScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/FrozenLabyrinth/FrozenLabyrinth.lua")
	h.World().SetSkillMagic(1177)
	p := &scripttest.Player{ObjectID: 1}
	pronghorn := &scripttest.Npc{ObjectID: 2, ID: 22088, X: 1000, Y: 2000, Z: -50, Heading: 7}
	h.Attack(p, pronghorn, 10)
	h.AttackWith(p, pronghorn, 10, false, &scripting.Skill{ID: 1177, Level: 1})
	if len(h.World().Spawned) != 0 {
		t.Fatalf("a hit or a magic skill split it")
	}
	h.AttackWith(p, pronghorn, 10, false, &scripting.Skill{ID: 3, Level: 1})
	spirits := b1Spawned(h, 22087)
	want := [][2]int{{1000, 2000}, {1020, 2000}, {1040, 2000}, {1000, 2020}, {1000, 2040}, {1000, 2060}}
	if len(spirits) != len(want) {
		t.Fatalf("%d spirits, want 6", len(spirits))
	}
	for i, s := range spirits {
		if s.X != want[i][0] || s.Y != want[i][1] || s.Z != -50 || s.Heading != 7 ||
			s.Intention != "ATTACK" || !s.Running || s.Hate[0].Hate != 999 {
			t.Errorf("spirit %d = %+v", i, s)
		}
	}
	if !pronghorn.Deleted || pronghorn.ScriptValue != 1 {
		t.Errorf("pronghorn = %+v, want deleted", pronghorn)
	}
	buffalo := &scripttest.Npc{ObjectID: 3, ID: 22094}
	h.AttackWith(p, buffalo, 10, false, &scripting.Skill{ID: 3, Level: 1})
	if len(b1Spawned(h, 22093)) != 6 {
		t.Errorf("a Frost Buffalo did not split into Lost Buffalos")
	}
}

func TestFreyasStewardScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/FrozenLabyrinth/FreyasSteward/FreyasSteward.lua")
	p := &scripttest.Player{ObjectID: 1, Level: 81}
	steward := &scripttest.Npc{ObjectID: 2, ID: 32029}
	if reply, _ := h.FirstTalk(p, steward); reply.Content != "32029.html" {
		t.Errorf("first talk = %+v", reply)
	}
	if reply, _ := h.Event("FreyasSteward", "enter", p, steward); reply.Content != "32029-1.html" {
		t.Errorf("level 81: %+v", reply)
	}
	p.Level = 82
	h.Event("FreyasSteward", "ENTER", p, steward)
	if p.X != 103045 || p.Y != -124361 {
		t.Errorf("level 82 at %d, %d", p.X, p.Y)
	}
}

func TestRaffortyScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/FrozenLabyrinth/Rafforty/Rafforty.lua")
	p := &scripttest.Player{ObjectID: 1}
	rafforty := &scripttest.Npc{ObjectID: 2, ID: 32020}
	if reply, _ := h.FirstTalk(p, rafforty); reply.Content != "32020.html" {
		t.Errorf("first talk = %+v", reply)
	}
	for event, want := range map[string]string{"32020-01.html": "32020-02.html", "32020-04.html": "32020-05.html", "32020-07.html": "32020-08.html", "32020-03.html": "32020-03.html"} {
		if reply, _ := h.Event("Rafforty", event, p, rafforty); reply.Content != want {
			t.Errorf("%s without items = %+v, want %s", event, reply, want)
		}
	}
	p.Give(16025, 1)
	p.Give(16027, 1)
	if reply, _ := h.Event("Rafforty", "32020-01.html", p, rafforty); reply.Content != "32020-01.html" {
		t.Errorf("with the necklace: %+v", reply)
	}
	h.Event("Rafforty", "32020-07.html", p, rafforty)
	if p.Items[16025] != 0 || p.Items[16027] != 0 || p.Items[16026] != 1 {
		t.Errorf("items %v, want the blessed necklace only", p.Items)
	}
}

func TestHotSpringsScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/HotSprings/HotSprings.lua")
	p := &scripttest.Player{ObjectID: 1}
	atrox := &scripttest.Npc{ObjectID: 2, ID: 21321}

	// Malaria at 10 %, level 1 on a healthy player.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 9), rnd.Get(100, 10)))
	h.Attack(p, atrox, 10)
	want := []scripttest.Action{
		{Kind: "set_target", Object: 2, Target: 1},
		{Kind: "cast", Object: 2, Value: 4554, Level: 1},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("malaria =\n%+v\nwant\n%+v", got, want)
	}

	// Its own disease one level up; level 10 stays 10.
	p.Effects = []int{4551, 4554}
	p.EffectLevels = map[int]int{4551: 3, 4554: 10}
	h.World().ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0), rnd.Get(100, 0)))
	h.Attack(p, atrox, 10)
	casts := b1Actions(h, "cast")
	if len(casts) != 2 || casts[0].Value != 4554 || casts[0].Level != 10 || casts[1].Value != 4551 || casts[1].Level != 4 {
		t.Fatalf("casts = %+v, want malaria 10 and rheumatism 4", casts)
	}

	// Casting already: the draws happen, nothing is cast.
	atrox.CastingNow = true
	h.World().ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0), rnd.Get(100, 0)))
	h.Attack(p, atrox, 10)
	if len(h.World().Actions) != 0 {
		t.Fatalf("a casting monster did %+v", h.World().Actions)
	}

	// Skills: only one aimed at the monster.
	flava := &scripttest.Npc{ObjectID: 3, ID: 21316}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 50), rnd.Get(100, 5)))
	h.SkillSee(p, flava, scripting.Skill{ID: 1177, Level: 1}, p)
	h.SkillSee(p, flava, scripting.Skill{ID: 1177, Level: 1}, p, flava)
	if casts := b1Actions(h, "cast"); len(casts) != 1 || casts[0].Value != 4552 || casts[0].Object != 3 {
		t.Errorf("skill see: %+v, want cholera from the Flava", casts)
	}
}

// The Four Sepulchers with quest 620: the calls, the entry, the first
// halls, the boss, the monsters' scripts and the end.
func TestFourSepulchersScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/ImperialTomb/FourSepulchers/FourSepulchers.lua")
	w := h.World()
	const (
		zone1   = 200221
		manager = 31921
		q620    = "Q00620_FourGoblets"
	)
	mgr := &scripttest.Npc{ObjectID: 10, ID: manager, Name: "Conquerors' Sepulcher Manager", X: 181061, Y: -85595, Z: -7200}
	w.Add(mgr)
	hall := w.AddZone(zone1)
	effect := w.AddZone(300100)
	effect.Type, effect.Area = "EffectZone", &scripttest.Area{MinX: 187000, MinY: -86000, MaxX: 188000, MaxY: -85000}
	effect.Params = map[string]string{"skillIdLvl": "4146-3;"}
	for _, d := range []int{25150012, 25150013, 25150014, 25150015, 25150016} {
		w.AddDoor(d, true)
	}

	// The managers call at xx:55 (the clock starts at 05:00).
	h.Start()
	h.Advance(55*time.Minute - time.Millisecond)
	if len(b1Actions(h, "say")) != 0 {
		t.Fatalf("called before xx:55")
	}
	h.Advance(time.Millisecond)
	if says := b1Actions(h, "say"); len(says) != 2 || says[0].Object != 10 || says[0].Text != "You may now enter the Sepulcher." {
		t.Fatalf("calls = %+v", says)
	}

	// The party: four with the quest and an entrance pass.
	var party []*scripttest.Player
	for i := range 4 {
		p := &scripttest.Player{ObjectID: int32(i + 1), Name: []string{"Leader", "Two", "Three", "Four"}[i], X: 181061 + 100*i, Y: -85595, Z: -7200}
		st := w.NewQuestState(p.Ref(), q620).(*scripttest.QuestState)
		st.SetState(scripting.StateStarted)
		p.Give(7075, 1)
		party = append(party, p)
	}
	leader := party[0]
	party[3].X = 181061 + 900 // too far to come along

	enter := func(p *scripttest.Player) scripting.Reply {
		t.Helper()
		reply, _ := h.Event("FourSepulchers", "ENTER", p, mgr)
		return reply
	}
	if reply := enter(leader); reply.Content != "31921-SP.html" {
		t.Errorf("alone: %+v", reply)
	}
	scripttest.MakeParty(party[:3]...)
	if reply := enter(party[1]); reply.Content != "31921-NL.html" {
		t.Errorf("not the leader: %+v", reply)
	}
	if reply := enter(leader); reply.Content != "31921-SP.html" {
		t.Errorf("three members: %+v", reply)
	}
	scripttest.MakeParty(party...)
	hall.Players = []*scripttest.Player{{ObjectID: 99}}
	if reply := enter(leader); reply.Content != "31921-FULL.html" {
		t.Errorf("a party inside: %+v", reply)
	}
	hall.Players = nil

	// 05:56: the entry window. A member without a pass stops the entry (the
	// chest timer starts anyway, as in Mobius).
	h.Advance(time.Minute)
	party[2].Items[7075] = 0
	if reply := enter(leader); reply.Content != "31921-SE.html" || reply.Vars["member"] != "Three" {
		t.Errorf("without a pass: %+v", reply)
	}
	party[2].Give(7075, 1)

	monster := &scripttest.Npc{ObjectID: 50, ID: 18132, Types: []string{"Monster"}}
	keyChest := &scripttest.Npc{ObjectID: 51, ID: 31467}
	folk := &scripttest.Npc{ObjectID: 52, ID: 31925}
	w.Add(monster, keyChest, folk)
	hall.Npcs = []*scripttest.Npc{monster, keyChest, folk}
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t,
		rnd.Between(-80, 80, 10), rnd.Between(-80, 80, -10),
		rnd.Between(-80, 80, 0), rnd.Between(-80, 80, 0),
		rnd.Between(-80, 80, 0), rnd.Between(-80, 80, 0)))
	if reply := enter(leader); reply.Content != "31921-OK.html" {
		t.Fatalf("entry: %+v", reply)
	}
	if !monster.Deleted || !keyChest.Deleted || folk.Deleted {
		t.Errorf("deleted: monster %v, key chest %v, gatekeeper %v; want true, true, false", monster.Deleted, keyChest.Deleted, folk.Deleted)
	}
	if !effect.Disabled || len(b1Actions(h, "door_close")) != 5 {
		t.Errorf("effect zone disabled %v, doors closed %+v", effect.Disabled, b1Actions(h, "door_close"))
	}
	if leader.X != 181642 || leader.Y != -85597 || party[3].X == 181632 {
		t.Errorf("leader at %d, %d; the far member at %d", leader.X, leader.Y, party[3].X)
	}
	if leader.Items[7075] != 0 || leader.Items[7261] != 1 || party[3].Items[7075] != 1 {
		t.Errorf("leader items %v, far member %v", leader.Items, party[3].Items)
	}
	hall.Players = party[:3]

	// 06:00: the first chest; it calls the first wave.
	h.Advance(4 * time.Minute)
	chests := b1Spawned(h, 31468)
	if len(chests) != 1 || chests[0].X != 182074 {
		t.Fatalf("chests = %+v", chests)
	}
	if reply, ok := h.FirstTalk(leader, chests[0]); !ok || reply.Content != "" {
		t.Errorf("chest talk = %+v", reply)
	}
	wave1 := 0
	for _, n := range w.Spawned {
		if n.ID != 31468 {
			wave1++
		}
	}
	if !chests[0].Deleted || wave1 != 15 {
		t.Fatalf("wave 1: %d monsters, chest deleted %v", wave1, chests[0].Deleted)
	}

	// The key rewarder leaves a key chest; its key opens the next gate.
	rewarder := b1Spawned(h, 18120)[0]
	h.Kill(leader, rewarder)
	keys := b1Spawned(h, 31467)
	if len(keys) != 1 || keys[0].X != rewarder.X {
		t.Fatalf("key chests = %+v", keys)
	}
	h.FirstTalk(leader, keys[0])
	h.FirstTalk(leader, keys[0]) // once only
	if leader.Items[7260] != 1 {
		t.Fatalf("keys = %d", leader.Items[7260])
	}
	gate := &scripttest.Npc{ObjectID: 60, ID: 31925, Name: "Baron's Hall Gatekeeper"}
	w.ClearRecords()
	if reply, ok := h.Event("FourSepulchers", "OPEN_GATE", leader, gate); ok {
		t.Errorf("open gate showed %+v", reply)
	}
	if !w.DoorOpen(25150012) || leader.Items[7260] != 0 || gate.ScriptValue != 1 {
		t.Fatalf("door open %v, keys %d, gatekeeper value %d", w.DoorOpen(25150012), leader.Items[7260], gate.ScriptValue)
	}
	if chests = b1Spawned(h, 31468); len(chests) != 1 || chests[0].X != 183868 {
		t.Fatalf("second chest = %+v", chests)
	}
	h.Advance(15 * time.Second)
	if w.DoorOpen(25150012) {
		t.Errorf("the gate is open 15 s later")
	}
	gate2 := &scripttest.Npc{ObjectID: 61, ID: 31926, Name: "Gatekeeper"}
	if reply, _ := h.Event("FourSepulchers", "OPEN_GATE", leader, gate2); reply.Content != "Gatekeeper-no.html" || reply.Vars["npcname"] != "Gatekeeper" {
		t.Errorf("without a key: %+v", reply)
	}

	// Wave 2: checked every 5 s until all are dead, then a key chest (or
	// the wave again) where the last one stood.
	h.FirstTalk(leader, chests[0])
	wave2 := b1Spawned(h, 18141)
	if len(wave2) != 10 {
		t.Fatalf("wave 2: %d", len(wave2))
	}
	h.Advance(5 * time.Second)
	for _, n := range wave2 {
		n.Dead = true
	}
	h.SetRand(rnd.NewScripted(t, rnd.Bool(false)))
	h.Advance(5 * time.Second)
	keys = b1Spawned(h, 31467)
	if len(keys) != 1 || keys[0].X != wave2[9].X || keys[0].Y != wave2[9].Y {
		t.Fatalf("key chests after wave 2 = %+v", keys)
	}

	// The boss: the goblet for those on the quest without the brooch, the
	// next wave and the teleporter.
	party[1].Give(7262, 1)
	boss := &scripttest.Npc{ObjectID: 70, ID: 25346, X: 189000, Y: -85500, Z: -7216}
	w.ClearRecords()
	h.Kill(leader, boss)
	if leader.Items[7256] != 1 || party[1].Items[7256] != 0 || party[2].Items[7256] != 1 {
		t.Errorf("goblets: %d, %d, %d; want 1, 0, 1", leader.Items[7256], party[1].Items[7256], party[2].Items[7256])
	}
	if got := w.SoundsOf(leader); !slices.Equal(got, []string{"ItemSound.quest_middle"}) {
		t.Errorf("sounds = %q", got)
	}
	if len(b1Spawned(h, 31452)) != 1 || len(b1Spawned(h, 18150)) != 1 {
		t.Errorf("teleporters %d, victims %d after the boss", len(b1Spawned(h, 31452)), len(b1Spawned(h, 18150)))
	}

	// The 50 minutes are over: everyone out.
	h.Advance(50*time.Minute - 4*time.Minute - 25*time.Second - 2*scripttest.KillDelay - time.Millisecond)
	if len(b1Actions(h, "zone_oust")) != 0 {
		t.Fatalf("ousted early")
	}
	h.Advance(time.Millisecond)
	if ousts := b1Actions(h, "zone_oust"); len(ousts) != 1 || ousts[0].Value != zone1 {
		t.Errorf("ousts = %+v", ousts)
	}
}

// The monsters' own scripts: the fleeing victim, the petrified statue, the
// charm devices and the reward chest.
func TestFourSepulchersMonsters(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/ImperialTomb/FourSepulchers/FourSepulchers.lua")
	w := h.World()
	home := scripting.Location{X: 1000, Y: 2000, Z: -7216}
	victim := &scripttest.Npc{ObjectID: 1, ID: 18150, X: 1000, Y: 2000, Z: -7216, Spawn: &home}
	h.Spawn(victim)
	if !victim.CoreAIDisabled || !victim.Running || !victim.NoRandomWalk {
		t.Fatalf("victim = %+v", victim)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Between(-400, 400, 100), rnd.Between(-400, 400, -50), rnd.Get(3, 2)))
	h.Advance(time.Second)
	if victim.Dest == nil || victim.Dest.X != 1100 || victim.Dest.Y != 1950 {
		t.Errorf("victim runs to %v", victim.Dest)
	}
	if says := b1Actions(h, "say"); len(says) != 1 || says[0].Text != "Keep pushing!" {
		t.Errorf("victim says %+v", says)
	}
	// Too far from home: it only shouts.
	victim.X, victim.Dest = 1700, nil
	h.SetRand(rnd.NewScripted(t, rnd.Between(-400, 400, 0), rnd.Between(-400, 400, 0), rnd.Get(3, 0)))
	h.Advance(3 * time.Second)
	if victim.Dest != nil {
		t.Errorf("a victim 700 from home ran to %v", victim.Dest)
	}
	p := &scripttest.Player{ObjectID: 5}
	victim.Dead = true
	h.Kill(p, victim)
	if len(b1Spawned(h, 18158)) != 1 {
		t.Errorf("no chest rewarder for the victim")
	}

	statue := &scripttest.Npc{ObjectID: 2, ID: 18232, Effects: []int{4616}}
	w.ClearRecords()
	h.Spawn(statue)
	if casts := b1Actions(h, "cast"); len(casts) != 1 || casts[0].Value != 4616 || !statue.NoReturn {
		t.Fatalf("statue: casts %+v, no return %v", casts, statue.NoReturn)
	}
	h.Advance(5 * time.Minute)
	if len(statue.Effects) != 0 || len(b1Actions(h, "stop_effects")) != 1 {
		t.Errorf("statue effects %v after 5 minutes", statue.Effects)
	}

	// A charm device switches off the effect zone of its skill.
	weak := w.AddZone(300100)
	weak.Type, weak.Area, weak.Params = "EffectZone", &scripttest.Area{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100}, map[string]string{"skillIdLvl": "4146-3;"}
	poison := w.AddZone(300101)
	poison.Type, poison.Area, poison.Params = "EffectZone", &scripttest.Area{MinX: 0, MinY: 0, MaxX: 100, MaxY: 100}, map[string]string{"skillIdLvl": "4148-3;"}
	p.X, p.Y = 50, 50
	charm := &scripttest.Npc{ObjectID: 3, ID: 18196}
	w.ClearRecords()
	h.Kill(p, charm)
	if !weak.Disabled || poison.Disabled {
		t.Errorf("weakness zone disabled %v, poison %v; want true, false", weak.Disabled, poison.Disabled)
	}
	if says := b1Actions(h, "say"); len(says) != 1 || says[0].Text != "The P. Atk. reduction device has now been destroyed." {
		t.Errorf("charm says %+v", says)
	}

	chest := &scripttest.Npc{ObjectID: 4, ID: 18256}
	h.SetRand(rnd.NewScripted(t, rnd.Between(300, 1300, 777)))
	h.Kill(p, chest)
	if drops := b1Actions(h, "drop_item"); len(drops) != 1 || drops[0].Value != 57 || drops[0].Count != 777 {
		t.Errorf("drops = %+v", drops)
	}
}
