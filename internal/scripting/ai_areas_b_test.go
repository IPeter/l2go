package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The area scripts of Plan 13 group B, second half (Mobius ai/areas from
// KetraOrcOutpust on), on the fake world.

// areaActions returns the actions of one kind.
func areaActions(w *scripttest.World, kind string) []scripttest.Action {
	var out []scripttest.Action
	for _, a := range w.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// wantReply checks a page reply.
func wantReply(t *testing.T, what string, got scripting.Reply, ok bool, want string) {
	t.Helper()
	if !ok || got.Content != want {
		t.Errorf("%s: reply %q (%v); want %q", what, got.Content, ok, want)
	}
}

func TestKetraOrcSupportScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/KetraOrcOutpust/KetraOrcSupport/KetraOrcSupport.lua")
	stranger := &scripttest.Player{ObjectID: 1, Name: "Stranger"}
	ally := func(id int32, mark int) *scripttest.Player {
		p := &scripttest.Player{ObjectID: id, Name: "Ally"}
		p.Give(mark, 1)
		return p
	}
	npc := func(id int) *scripttest.Npc {
		return &scripttest.Npc{ObjectID: int32(id) * 10, ID: id, MaxHP: 100, MaxMP: 50}
	}

	for _, c := range []struct {
		p    *scripttest.Player
		npc  int
		want string
	}{
		{stranger, 31370, "31370-no.html"},
		{ally(2, 7211), 31370, "31370-friend.html"},
		{stranger, 31372, "31372-03.html"},
		{ally(3, 7212), 31372, "31372-01.html"},
		{ally(4, 7213), 31372, "31372-04.html"},
		{ally(5, 7211), 31374, "31374-01.html"},
		{ally(6, 7214), 31374, "31374-02.html"},
		{ally(7, 7215), 31375, "31375-03.html"},
		{stranger, 31375, "31375-no.html"},
		{ally(8, 7213), 31376, "31376-01.html"},
		{ally(9, 7214), 31376, "31376-02.html"},
		{stranger, 31376, "31376-no.html"},
	} {
		reply, ok := h.FirstTalk(c.p, npc(c.npc))
		wantReply(t, "first talk", reply, ok, c.want)
	}

	// Asefa's Might costs 3 horns.
	asefa := npc(31372)
	buyer := ally(10, 7213)
	buyer.Give(7186, 2)
	reply, ok := h.Event("KetraOrcSupport", "3", buyer, asefa)
	wantReply(t, "buff without horns", reply, ok, "31372-02.html")
	buyer.Give(7186, 2)
	if _, ok := h.Event("KetraOrcSupport", "3", buyer, asefa); ok {
		t.Errorf("the buff answered with a page")
	}
	want := []scripttest.Action{
		{Kind: "set_target", Object: asefa.ObjectID, Target: buyer.ObjectID},
		{Kind: "cast", Object: asefa.ObjectID, Value: 4345, Level: 1},
		{Kind: "hp_mp", Object: asefa.ObjectID, Value: 100},
	}
	if got := h.World().Actions; !slices.Equal(got, want) || buyer.Items[7186] != 1 || asefa.MP != 50 {
		t.Errorf("buff: actions %+v, horns %d, MP %v; want %+v, 1, 50", got, buyer.Items[7186], asefa.MP, want)
	}

	// Kurfa's teleport list follows the alliance level.
	h.World().ClearRecords()
	h.World().AddTeleportList(31376, "list4", "list5")
	kurfa := npc(31376)
	friend := ally(11, 7214)
	h.Event("KetraOrcSupport", "TELEPORT", friend, kurfa)
	h.Event("KetraOrcSupport", "TELEPORT", stranger, kurfa)
	if got := areaActions(h.World(), "teleport_list"); len(got) != 1 || got[0].Text != "list4" || got[0].Target != friend.ObjectID {
		t.Errorf("teleport lists %+v; want list4 for the level 4 ally only", got)
	}
}

func TestVarkaSilenosSupportScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/VarkaSilenosBarracks/VarkaSilenosSupport/VarkaSilenosSupport.lua")
	stranger := &scripttest.Player{ObjectID: 1}
	friend := &scripttest.Player{ObjectID: 2}
	friend.Give(7224, 1) // level 4
	teranu := &scripttest.Npc{ObjectID: 10, ID: 31383}
	udan := &scripttest.Npc{ObjectID: 11, ID: 31379, MaxHP: 80, MaxMP: 40}

	reply, ok := h.FirstTalk(friend, teranu)
	wantReply(t, "Teranu", reply, ok, "31383-02.html")
	reply, ok = h.FirstTalk(stranger, teranu)
	wantReply(t, "Teranu to a stranger", reply, ok, "31383-no.html")
	reply, ok = h.FirstTalk(friend, udan)
	wantReply(t, "Udan", reply, ok, "31379-04.html")
	reply, ok = h.FirstTalk(friend, &scripttest.Npc{ObjectID: 12, ID: 31377})
	wantReply(t, "Ashas", reply, ok, "31377-friend.html")

	h.World().AddTeleportList(31383, "list-4")
	h.Event("VarkaSilenosSupport", "TELEPORT", friend, teranu)
	if got := areaActions(h.World(), "teleport_list"); len(got) != 1 || got[0].Text != "list-4" {
		t.Errorf("teleport lists %+v; want list-4", got)
	}

	friend.Give(7187, 6)
	h.Event("VarkaSilenosSupport", "8", friend, udan)
	if got := areaActions(h.World(), "cast"); len(got) != 1 || got[0].Value != 4357 || friend.Items[7187] != 0 {
		t.Errorf("Haste: casts %+v, seeds left %d; want 4357, 0", got, friend.Items[7187])
	}
	if _, ok := h.Event("VarkaSilenosSupport", "9", friend, udan); ok {
		t.Errorf("an unknown buff answered")
	}
}

func TestMonasteryOfSilenceScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/MonasteryOfSilence/MonasteryOfSilence.lua")
	armed := &scripttest.Player{ObjectID: 1, Name: "Hero", Weapon: 2369, X: 300}
	unarmed := &scripttest.Player{ObjectID: 2, Name: "Monk", X: 300}

	// Only armed players are hated.
	if !h.NpcHate(&scripttest.Npc{ObjectID: 10, ID: 22125}, armed) || h.NpcHate(&scripttest.Npc{ObjectID: 11, ID: 22125}, unarmed) {
		t.Errorf("hate: armed and unarmed players are not told apart")
	}

	// A caster guard warns and casts Silence, then attacks when it lands.
	caster := &scripttest.Npc{ObjectID: 20, ID: 22124}
	h.AggroRangeEnter(caster, unarmed, false)
	h.AggroRangeEnter(caster, armed, false)
	want := []scripttest.Action{
		{Kind: "set_target", Object: 20, Target: 1},
		{Kind: "say", Object: 20, Text: "You cannot carry a weapon without authorization!"},
		{Kind: "cast", Object: 20, Value: 4589, Level: 8},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("caster =\n%+v\nwant\n%+v", got, want)
	}
	h.World().ClearRecords()
	h.AggroRangeEnter(caster, armed, false) // it has a target now
	h.SpellFinished(caster, armed, scripting.Skill{ID: 4589, Level: 8})
	want = []scripttest.Action{
		{Kind: "run", Object: 20},
		{Kind: "hate", Object: 20, Target: 1, Value: 999},
		{Kind: "attack", Object: 20, Target: 1},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("after Silence =\n%+v\nwant\n%+v", got, want)
	}

	// The others answer a hostile skill aimed at them.
	h.World().ClearRecords()
	monk := &scripttest.Npc{ObjectID: 30, ID: 22134}
	h.World().SetSkillEffectTypes(1069, "AGGRESSION")
	h.SkillSee(armed, monk, scripting.Skill{ID: 1069, Level: 1}, armed)
	h.SkillSee(armed, monk, scripting.Skill{ID: 1011, Level: 1}, monk)
	if len(h.World().Actions) != 0 {
		t.Fatalf("a skill at someone else or a friendly one: %+v", h.World().Actions)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 1)))
	h.SkillSee(armed, monk, scripting.Skill{ID: 1069, Level: 1}, monk)
	want = []scripttest.Action{
		{Kind: "say", Object: 30, Text: "Hero! How dare you defy the will of Einhasad!"},
		{Kind: "hate", Object: 30, Target: 1, Value: 999},
		{Kind: "attack", Object: 30, Target: 1},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("hostile skill =\n%+v\nwant\n%+v", got, want)
	}

	// A guard that spawns next to an armed player (here through his pet)
	// charges him.
	h.World().ClearRecords()
	armed.X = 3000 // out of aggro range
	owner := &scripttest.Player{ObjectID: 3, Name: "Owner", Weapon: 2369, X: 2000}
	owner.Summon = &scripttest.Npc{ObjectID: 4, ID: 12077, X: 200, Types: []string{"Pet"}}
	h.World().Add(owner, owner.Summon, unarmed)
	guard := &scripttest.Npc{ObjectID: 40, ID: 22129, AggroRange: 500}
	h.Spawn(guard)
	want = []scripttest.Action{
		{Kind: "set_target", Object: 40, Target: 3},
		{Kind: "say", Object: 40, Text: "You cannot carry a weapon without authorization!"},
		{Kind: "run", Object: 40},
		{Kind: "hate", Object: 40, Target: 3, Value: 999},
		{Kind: "attack", Object: 40, Target: 3},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Fatalf("spawn =\n%+v\nwant\n%+v", got, want)
	}
}

func TestPaganTempleScripts(t *testing.T) {
	h := scripttest.Load(t, scriptsDir,
		"ai/areas/PaganTemple/AltarGatekeeper.lua", "ai/areas/PaganTemple/PaganKeys.lua",
		"ai/areas/PaganTemple/PaganTeleporters/PaganTeleporters.lua")
	w := h.World()

	// Four gatekeepers spawn: one shout, then none for 30 s.
	for i := range 4 {
		h.Spawn(&scripttest.Npc{ObjectID: int32(100 + i), ID: 32051})
	}
	h.Advance(30 * time.Second)
	h.Spawn(&scripttest.Npc{ObjectID: 104, ID: 32051})
	if got := areaActions(w, "say"); len(got) != 2 || got[0].Object != 100 || got[1].Object != 104 ||
		got[0].Text != "The door to the 3rd floor of the altar is now open." {
		t.Errorf("shouts %+v; want the first gatekeeper's and one 30 s later", got)
	}

	// Keys: one kill in ten drops one on the ground.
	w.ClearRecords()
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 9), rnd.Get(100, 10)))
	h.Kill(p, &scripttest.Npc{ObjectID: 200, ID: 22142})
	h.Kill(p, &scripttest.Npc{ObjectID: 201, ID: 22168})
	if got := areaActions(w, "drop_item"); len(got) != 1 || got[0].Value != 8274 || got[0].Target != 1 {
		t.Errorf("drops %+v; want one Chapel Key for the killer", got)
	}

	// Gates: the entrance opens for 10 s with a mark.
	w.ClearRecords()
	w.AddDoor(19160001, false)
	w.AddDoor(19160010, false)
	w.AddDoor(19160011, false)
	keeper := &scripttest.Npc{ObjectID: 300, ID: 32034}
	reply, ok := h.Talk("PaganTeleporters", p, keeper)
	wantReply(t, "no mark", reply, ok, "noItem.htm")
	p.Give(8065, 1)
	reply, ok = h.Talk("PaganTeleporters", p, keeper)
	wantReply(t, "faded mark", reply, ok, "FadedMark.htm")
	if !w.DoorOpen(19160001) {
		t.Fatalf("the entrance did not open")
	}
	h.Advance(10 * time.Second)
	if w.DoorOpen(19160001) {
		t.Errorf("the entrance is still open after 10 s")
	}
	chapel := &scripttest.Npc{ObjectID: 301, ID: 32036}
	reply, ok = h.Talk("PaganTeleporters", p, chapel)
	wantReply(t, "chapel without a pagan's mark", reply, ok, "noMark.htm")
	p.Give(8067, 1)
	reply, ok = h.Talk("PaganTeleporters", p, chapel)
	wantReply(t, "chapel", reply, ok, "openDoor.htm")
	if !w.DoorOpen(19160010) || !w.DoorOpen(19160011) {
		t.Errorf("the chapel doors did not open")
	}

	// A mirror takes the player through.
	h.FirstTalk(p, &scripttest.Npc{ObjectID: 302, ID: 32040})
	if p.X != 36640 || p.Y != -51218 || p.Z != 718 {
		t.Errorf("player at %d,%d,%d after the mirror; want 36640,-51218,718", p.X, p.Y, p.Z)
	}
}

func TestPaganKeysAutoLoot(t *testing.T) {
	h := scripttest.LoadWithTables(t, scriptsDir, map[string]map[string]any{"Config": {"AUTO_LOOT": true}},
		"ai/areas/PaganTemple/PaganKeys.lua")
	p := &scripttest.Player{ObjectID: 1}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Kill(p, &scripttest.Npc{ObjectID: 2, ID: 22140})
	if p.Items[8273] != 1 || len(areaActions(h.World(), "drop_item")) != 0 {
		t.Errorf("auto loot: %d Anteroom Keys, drops %+v; want the key in the inventory", p.Items[8273], h.World().Actions)
	}
}

// The Andreas Van Halter raid: balcony kills, the door swap, the ritual
// cinematic, the anti-glitch teleports and the end of the raid.
func TestAndreasVanHalterScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/PaganTemple/AndreasVanHalter.lua")
	w := h.World()
	for _, d := range []int{19160014, 19160015} {
		w.AddDoor(d, false)
	}
	for _, d := range []int{19160016, 19160017} {
		w.AddDoor(d, true)
	}
	boss := &scripttest.Npc{ObjectID: 1000, ID: 29062, X: -16393, Y: -53433, Z: -10439}
	h.Spawn(boss)
	if !w.DoorOpen(19160014) || !w.DoorOpen(19160015) || w.DoorOpen(19160016) || w.DoorOpen(19160017) {
		t.Fatalf("doors after the spawn: the balcony should be open, the altar closed")
	}

	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	for i := range 15 {
		h.Kill(p, &scripttest.Npc{ObjectID: int32(2000 + i), ID: 22176})
	}
	if len(w.Spawned) != 0 {
		t.Fatalf("the ritual started after 15 kills")
	}
	h.Kill(p, &scripttest.Npc{ObjectID: 2100, ID: 22188})
	if w.DoorOpen(19160014) || !w.DoorOpen(19160016) || !w.DoorOpen(19160017) {
		t.Errorf("doors after the 16th kill: the balcony should be closed, the altar open")
	}
	if len(w.Spawned) != 5 || w.Spawned[4].ID != 32038 {
		t.Fatalf("spawned %d NPCs; want 4 gatekeepers and the offering", len(w.Spawned))
	}
	victim := w.Spawned[4]
	h.Kill(p, &scripttest.Npc{ObjectID: 2101, ID: 22176}) // no second ritual
	if len(w.Spawned) != 5 {
		t.Fatalf("a kill during the ritual spawned more")
	}

	// The cinematic, 3 s after the 16th kill (the last kill took 2.5 s).
	w.ClearRecords()
	h.Advance(500 * time.Millisecond)
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 13014 {
		t.Fatalf("camera: spawned %+v", w.Spawned)
	}
	cam := w.Spawned[0]
	if !boss.Immobilized || boss.Heading != 16384 {
		t.Errorf("boss immobilized %v, heading %d; want true, 16384", boss.Immobilized, boss.Heading)
	}
	if got := areaActions(w, "cast"); len(got) != 1 || got[0].Value != 1168 || got[0].Level != 7 || boss.Target != victim {
		t.Errorf("ritual cast %+v at %v; want Curse Poison 7 at the offering", got, boss.Target)
	}
	cine := w.Cinematics()
	if len(cine) != 2 || cine[0].Kind != "sound" || cine[0].Text != "BS04_A" || cine[1].Kind != "camera" ||
		cine[1].Target != cam.ObjectID || cine[1].Camera != (scripting.Camera{Force: 1500, Angle1: 88, Angle2: 89, Range: 5000, RelPitch: 1}) {
		t.Errorf("cinematics %+v", cine)
	}
	h.Advance(300 * time.Millisecond)
	if got := areaActions(w, "social"); len(got) != 1 || got[0].Object != victim.ObjectID || got[0].Value != 1 {
		t.Errorf("the offering's social action: %+v", got)
	}
	h.Advance(300 * time.Millisecond)
	if !victim.Deleted {
		t.Errorf("the offering is still there")
	}
	h.Advance(9400*time.Millisecond + 5*time.Second + 6*time.Second)
	if boss.Immobilized || !cam.Deleted || len(w.Cinematics()) != 6 {
		t.Errorf("after the cinematic: immobilized %v, camera deleted %v, %d cinematics; want false, true, 6",
			boss.Immobilized, cam.Deleted, len(w.Cinematics()))
	}

	// Dragged below the floor or to an altar door, the boss comes back.
	w.ClearRecords()
	boss.Z = -10600
	h.Attack(p, boss, 10)
	boss.X, boss.Y, boss.Z = -15400, -54800, -10439
	h.Attack(p, boss, 10)
	got := areaActions(w, "npc_teleport")
	if len(got) != 2 || got[0].Loc != (scripting.Location{X: -16393, Y: -53433, Z: -10439}) ||
		got[1].Loc != (scripting.Location{X: -15690, Y: -54030, Z: -10439}) {
		t.Errorf("teleports %+v; want the altar, then the balcony door 1", got)
	}

	// Killed: the gatekeepers go, the doors are reset, the reset check
	// stops.
	boss.Dead = true
	h.Kill(p, boss)
	for _, gk := range w.Spawned {
		t.Errorf("spawned after the kill: %+v", gk)
	}
	if !w.DoorOpen(19160014) || w.DoorOpen(19160016) {
		t.Errorf("doors after the kill: the balcony should be open, the altar closed")
	}
	if n := w.Pending(); n != 0 {
		t.Errorf("%d timers left after the kill", n)
	}
}

// Five idle hours reset the altar; a reload keeps the state and the reset
// check.
func TestAndreasVanHalterReset(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/PaganTemple/AndreasVanHalter.lua")
	w := h.World()
	w.AddDoor(19160014, true)
	w.AddDoor(19160016, false)
	boss := &scripttest.Npc{ObjectID: 1000, ID: 29062}
	h.Spawn(boss)
	p := &scripttest.Player{ObjectID: 1}
	for i := range 16 {
		h.Kill(p, &scripttest.Npc{ObjectID: int32(2000 + i), ID: 22189})
	}
	h.Advance(30 * time.Second) // the cinematic ends
	if !w.DoorOpen(19160016) {
		t.Fatalf("the altar did not open")
	}
	if err := h.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	gatekeepers := w.Spawned[:4]
	boss.AttackingNow = true
	h.Advance(5 * time.Hour)
	if !w.DoorOpen(19160016) {
		t.Fatalf("reset while the boss swings")
	}
	boss.AttackingNow = false
	h.Advance(time.Minute)
	if w.DoorOpen(19160016) || !w.DoorOpen(19160014) || !gatekeepers[0].Deleted {
		t.Errorf("after 5 idle hours: altar open %v, balcony open %v, gatekeeper deleted %v; want false, true, true",
			w.DoorOpen(19160016), w.DoorOpen(19160014), gatekeepers[0].Deleted)
	}
}

func TestPlainsOfDionScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/PlainsOfDion/PlainsOfDion.lua")
	p := &scripttest.Player{ObjectID: 1, Name: "Hero"}
	monster := []string{"Monster"}
	lizard := &scripttest.Npc{ObjectID: 10, ID: 21104, ClanHelpRange: 500, Types: monster}
	helper := &scripttest.Npc{ObjectID: 11, ID: 21105, X: 100, Types: monster}
	busy := &scripttest.Npc{ObjectID: 12, ID: 21107, X: 100, Types: monster, AttackingNow: true}
	dead := &scripttest.Npc{ObjectID: 13, ID: 21107, X: 100, Types: monster, Dead: true}
	other := &scripttest.Npc{ObjectID: 14, ID: 21108, X: 100, Types: monster}
	hidden := &scripttest.Npc{ObjectID: 15, ID: 21105, X: 100, Types: monster}
	far := &scripttest.Npc{ObjectID: 16, ID: 21105, X: 900, Types: monster}
	h.World().Add(helper, busy, dead, other, hidden, far)
	h.World().SetSight(func(_, to any) bool { return to != hidden })

	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 0), rnd.Get(3, 2)))
	h.Attack(p, lizard, 10)
	h.Attack(p, lizard, 10) // the duel is over already
	want := []scripttest.Action{
		{Kind: "say", Object: 10, Text: "Hero! How dare you interrupt our fight! Hey guys, help!"},
		{Kind: "hate", Object: 11, Target: 1, Value: 999},
		{Kind: "run", Object: 11},
		{Kind: "attack", Object: 11, Target: 1},
		{Kind: "say", Object: 11, Text: "What are you looking at?"},
	}
	if got := h.World().Actions; !slices.Equal(got, want) {
		t.Errorf("duel =\n%+v\nwant\n%+v", got, want)
	}
}

func TestPrimevalIsleSprigantsAndTyrannosaurus(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/PrimevalIsle/PrimevalIsle.lua")
	w := h.World()

	// A sprigant poisons every 15 s, also after a reload.
	sprigant := &scripttest.Npc{ObjectID: 10, ID: 18346}
	h.Spawn(sprigant)
	h.Advance(15 * time.Second)
	if err := h.Reload(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	h.Advance(15 * time.Second)
	if got := areaActions(w, "cast"); len(got) != 2 || got[0].Value != 5086 || got[1].Value != 5086 {
		t.Errorf("sprigant casts %+v; want Deadly Poison twice", got)
	}

	// A Tyrannosaurus says "?" and stuns the player 6 s later.
	w.ClearRecords()
	p := &scripttest.Player{ObjectID: 1, X: 300}
	trex := &scripttest.Npc{ObjectID: 20, ID: 22215, HP: 100, MaxHP: 100}
	h.AggroRangeEnter(trex, p, false)
	h.AggroRangeEnter(trex, p, false)
	h.Advance(6 * time.Second)
	want := []scripttest.Action{
		{Kind: "say", Object: 20, Text: "?"},
		{Kind: "clear_aggro", Object: 20},
		{Kind: "set_target", Object: 20, Target: 1},
		{Kind: "cast", Object: 20, Value: 5120, Level: 1},
		{Kind: "hate", Object: 20, Target: 1, Value: 999},
		{Kind: "run", Object: 20},
		{Kind: "attack", Object: 20, Target: 1},
	}
	if got := w.Actions; !slices.Equal(got, want) || trex.ScriptValue != 0 {
		t.Fatalf("T-rex aggro =\n%+v\nwant\n%+v (script value %d)", got, want, trex.ScriptValue)
	}

	// Hit below 30 % after its first Berserk, it casts the second, and from
	// afar its stun.
	w.ClearRecords()
	trex.HP, trex.ScriptValue = 25, 1
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 10)))
	h.Attack(p, trex, 10)
	want = []scripttest.Action{
		{Kind: "cast", Object: 20, Value: 5087, Level: 2},
		{Kind: "set_target", Object: 20, Target: 1},
		{Kind: "cast", Object: 20, Value: 5120, Level: 1},
	}
	if got := w.Actions; !slices.Equal(got, want) {
		t.Fatalf("T-rex hit =\n%+v\nwant\n%+v", got, want)
	}

	// It hunts the plant eaters it sees.
	w.ClearRecords()
	ornit := &scripttest.Npc{ObjectID: 30, ID: 22201}
	h.CreatureSee(trex, ornit)
	h.CreatureSee(trex, &scripttest.Npc{ObjectID: 31, ID: 22196})
	if got := areaActions(w, "attack"); len(got) != 1 || got[0].Target != 30 {
		t.Errorf("T-rex attacks %+v; want the Ornithomimus only", got)
	}

	// A ghost Tyrannosaurus leaves after its despawn time out of combat.
	ghost := &scripttest.Npc{ObjectID: 40, ID: 22216, Params: map[string]string{"CollectGhost": "1", "CollectGhostDespawnTime": "2"}}
	h.Spawn(ghost)
	ghost.InCombat = true
	h.Advance(2 * time.Minute)
	if ghost.Deleted {
		t.Fatalf("the ghost left during a fight")
	}
	ghost.InCombat = false
	h.Advance(30 * time.Minute)
	if !ghost.Deleted {
		t.Errorf("the ghost is still there")
	}
}

func TestPrimevalIsleDinosaurs(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/PrimevalIsle/PrimevalIsle.lua")
	w := h.World()
	p := &scripttest.Player{ObjectID: 1}

	// A Deinonychus that sees a player may run 3,000 away from him.
	deino := &scripttest.Npc{ObjectID: 10, ID: 22205, X: 1000, Z: -3000}
	h.Spawn(deino)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 29)))
	h.CreatureSee(deino, p)
	if want := (scripting.Location{X: 4000, Z: -3000}); deino.Dest == nil || *deino.Dest != want || !deino.Running || deino.ScriptValue != 1 {
		t.Errorf("Deinonychus: dest %v, running %v, script value %d; want %v, true, 1", deino.Dest, deino.Running, deino.ScriptValue, want)
	}

	// An aggressive raptor uses its template skill on a player it sees.
	w.ClearRecords()
	raptor := &scripttest.Npc{ObjectID: 20, ID: 22196, HP: 100, MaxHP: 100}
	raptor.Params = map[string]string{"ag_type": "1", "ProbPhysicalSpecial1": "50", "ProbPhysicalSpecial2": "20"}
	raptor.ParamSkills = map[string]scripting.Skill{
		"PhysicalSpecial1": {ID: 4709, Level: 1},
		"PhysicalSpecial2": {ID: 4710, Level: 1},
		"SelfRangeBuff1":   {ID: 4711, Level: 1},
	}
	h.Spawn(raptor)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99), rnd.Get(100, 50)))
	h.CreatureSee(raptor, p)
	want := []scripttest.Action{
		{Kind: "set_target", Object: 20, Target: 1},
		{Kind: "cast", Object: 20, Value: 4709, Level: 1},
	}
	if got := w.Actions; !slices.Equal(got, want) {
		t.Fatalf("raptor sees =\n%+v\nwant\n%+v", got, want)
	}

	// Below 30 % it forgets its enemies, buffs itself once and hits the one
	// it hated most with its skills (twice as likely below 50 %).
	w.ClearRecords()
	raptor.HP = 20
	raptor.Hate = []scripttest.Hate{{Target: p, Hate: 100}}
	raptor.DisabledSkills = []int{4710}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99), rnd.Get(100, 0))) // PhysicalSpecial2 is disabled
	h.Attack(p, raptor, 10)
	want = []scripttest.Action{
		{Kind: "clear_aggro", Object: 20},
		{Kind: "cast", Object: 20, Value: 4711, Level: 1},
		{Kind: "run", Object: 20},
		{Kind: "attack", Object: 20, Target: 1},
		{Kind: "set_target", Object: 20, Target: 1},
		{Kind: "cast", Object: 20, Value: 4709, Level: 1},
	}
	if got := w.Actions; !slices.Equal(got, want) {
		t.Fatalf("raptor below 30 %% =\n%+v\nwant\n%+v", got, want)
	}
	w.ClearRecords()
	h.Attack(p, raptor, 10) // the buff is used, nothing hated: nothing
	if len(w.Actions) != 0 {
		t.Errorf("second hit: %+v", w.Actions)
	}

	// A hit egg calls the monsters around, each on a coin toss.
	w.ClearRecords()
	egg := &scripttest.Npc{ObjectID: 30, ID: 18344}
	near := &scripttest.Npc{ObjectID: 31, ID: 22196, X: 100, Types: []string{"Monster", "Attackable"}}
	lazy := &scripttest.Npc{ObjectID: 32, ID: 22196, X: 200, Types: []string{"Monster", "Attackable"}}
	w.Add(near, lazy)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 80), rnd.Bool(true), rnd.Bool(false), rnd.Get(100, 0)))
	h.Attack(p, egg, 1)
	h.Attack(p, egg, 1)
	if got := areaActions(w, "attack"); len(got) != 1 || got[0].Object != 31 || got[0].Target != 1 {
		t.Errorf("egg helpers %+v; want the first monster only, once", got)
	}
}

func TestElrokiTeleportersScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/PrimevalIsle/ElrokiTeleporters/ElrokiTeleporters.lua")
	p := &scripttest.Player{ObjectID: 1}
	orahochin := &scripttest.Npc{ObjectID: 10, ID: 32111}
	reply, ok := h.FirstTalk(p, orahochin)
	wantReply(t, "first talk", reply, ok, "32111.html")
	p.InCombat = true
	reply, ok = h.Event("ElrokiTeleporters", "TELEPORT", p, orahochin)
	wantReply(t, "in combat", reply, ok, "32111-no.html")
	p.InCombat = false
	h.Event("ElrokiTeleporters", "TELEPORT", p, &scripttest.Npc{ObjectID: 11, ID: 32112})
	if p.X != 7327 || p.Y != -5501 || p.Z != -3232 {
		t.Errorf("player at %d,%d,%d; want Gariachin's 7327,-5501,-3232", p.X, p.Y, p.Z)
	}
}

func TestToIVortexScript(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/areas/TowerOfInsolence/ToIVortex/ToIVortex.lua")
	p := &scripttest.Player{ObjectID: 1}
	vortex := &scripttest.Npc{ObjectID: 10, ID: 30953}
	reply, ok := h.Event("ToIVortex", "4", p, vortex)
	wantReply(t, "no stone", reply, ok, "no-stones.htm")
	p.Give(4402, 2)
	h.Event("ToIVortex", "4", p, vortex)
	if p.Items[4402] != 1 || p.X != 114636 || p.Z != -640 {
		t.Errorf("blue stones %d, player at %d,%d; want 1 left, floor 4", p.Items[4402], p.X, p.Z)
	}

	keplon := &scripttest.Npc{ObjectID: 11, ID: 30949}
	p.Give(57, 99999)
	reply, ok = h.Event("ToIVortex", "RED", p, keplon)
	wantReply(t, "not enough adena", reply, ok, "30949no-adena.htm")
	p.Give(57, 100001)
	h.Event("ToIVortex", "RED", p, keplon)
	if p.Items[57] != 100000 || p.Items[4403] != 1 {
		t.Errorf("adena %d, red stones %d; want 100000, 1", p.Items[57], p.Items[4403])
	}
}
