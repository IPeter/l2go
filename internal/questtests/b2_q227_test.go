package questtests

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q227 = "Q00227_TestOfTheReformer"

// b2Q227Cast runs the attack handlers for npc, hit by player with a skill.
func b2Q227Cast(t *testing.T, h *scripttest.Harness, p *scripttest.Player, npc *scripttest.Npc, skillID int) {
	t.Helper()
	h.World().Add(p, npc)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Engine().Attack(ctx, npc.Ref(), p.Ref(), 10, false, &scripting.Skill{ID: skillID, Level: 1}); err != nil {
		t.Fatalf("attack with skill %d: %v", skillID, err)
	}
}

// b2Q227Spawn checks that exactly one NPC of id was spawned since the last
// ClearRecords and returns it.
func b2Q227Spawn(t *testing.T, h *scripttest.Harness, what string, id int) *scripttest.Npc {
	t.Helper()
	return b2Q226Spawned(t, h, what, id)[0]
}

// Test of the Reformer: Pupina and the Nameless Revenants' ripped diary,
// Sla, Katari and the Ol Mahums, Kakan's Crimson Werewolf, Nyakuri's Krudel
// Lizardman, Ramus's bone fragments, the Mark of Reformer.
func TestB2Q227Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q227)
	pupina := &scripttest.Npc{ObjectID: 0x60000001, ID: 30118}
	sla := &scripttest.Npc{ObjectID: 0x60000002, ID: 30666}
	ramus := &scripttest.Npc{ObjectID: 0x60000003, ID: 30667}
	katari := &scripttest.Npc{ObjectID: 0x60000004, ID: 30668}
	kakan := &scripttest.Npc{ObjectID: 0x60000005, ID: 30669}
	nyakuri := &scripttest.Npc{ObjectID: 0x60000006, ID: 30670}

	oracle := &scripttest.Player{ObjectID: 0x10000001, Name: "Oracle", ClassID: 29, Race: 1, Level: 39}
	reply, ok := h.Talk(b2Q227, oracle, pupina)
	b2Page(t, "oracle", reply, ok, "30118-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 42, Race: 2, Level: 38}
	reply, ok = h.Talk(b2Q227, young, pupina)
	b2Page(t, "level 38", reply, ok, "30118-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 15, Level: 39}
	reply, ok = h.Talk(b2Q227, hero, pupina)
	b2Page(t, "offer", reply, ok, "30118-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q227, "30118-04.htm", hero, pupina)
	b2Page(t, "accept", reply, ok, "30118-04b.htm")
	b2Quest(t, "accept", hero, b2Q227, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{2822: 1, 7562: 60})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q227, hero, pupina)
	b2Page(t, "Pupina cond 1", reply, ok, "30118-04a.htm")

	// Only a kill right after Disrupt Undead gives a diary page.
	revenant := &scripttest.Npc{ObjectID: 0x60000010, ID: 27099}
	b2Q227Cast(t, h, hero, revenant, 1031)
	h.Attack(hero, revenant, 10)
	h.Kill(hero, revenant)
	b2Items(t, "plain hit", hero, map[int]int64{2831: 0})
	for range 6 {
		b2Q227Cast(t, h, hero, revenant, 1031)
		h.Kill(hero, revenant)
	}
	b2Items(t, "6 pages", hero, map[int]int64{2831: 6})
	h.World().ClearRecords()
	b2Q227Cast(t, h, hero, revenant, 1031)
	h.Kill(hero, revenant)
	b2Quest(t, "7 pages", hero, b2Q227, scripting.StateStarted, 2)
	b2Items(t, "7 pages", hero, map[int]int64{2831: 0})
	aruraune := b2Q227Spawn(t, h, "Aruraune", 27128)
	h.World().ClearRecords()
	h.Kill(hero, aruraune)
	b2Quest(t, "nail", hero, b2Q227, scripting.StateStarted, 3)
	b2Items(t, "nail", hero, map[int]int64{2832: 1})
	if want := []scripttest.Action{{Kind: "say", Object: aruraune.ObjectID, Text: "The concealed truth will always be revealed...!"}}; !slices.Equal(h.World().Actions, want) {
		t.Errorf("Aruraune: actions %+v, want %+v", h.World().Actions, want)
	}
	reply, ok = h.Talk(b2Q227, hero, pupina)
	b2Page(t, "Pupina nail", reply, ok, "30118-05.htm")
	h.Event(b2Q227, "30118-06.htm", hero, pupina)
	b2Quest(t, "introduction", hero, b2Q227, scripting.StateStarted, 4)
	b2Items(t, "introduction", hero, map[int]int64{2822: 0, 2832: 0, 2823: 1})

	// Sla, Katari and the Ol Mahums.
	reply, ok = h.Talk(b2Q227, hero, sla)
	b2Page(t, "Sla", reply, ok, "30666-01.htm")
	h.Event(b2Q227, "30666-04.htm", hero, sla)
	b2Quest(t, "Sla letter", hero, b2Q227, scripting.StateStarted, 5)
	b2Items(t, "Sla letter", hero, map[int]int64{2823: 0, 2824: 1})
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q227, hero, katari)
	b2Page(t, "Katari", reply, ok, "30668-01.htm")
	b2Quest(t, "inspector", hero, b2Q227, scripting.StateStarted, 6)
	b2Items(t, "inspector", hero, map[int]int64{2824: 0})
	mahums := b2Q226Spawned(t, h, "Ol Mahums", 30732, 27129)
	pilgrim, inspector := mahums[0], mahums[1]
	// The inspector hates and attacks the pilgrim.
	if want := []scripttest.Action{
		{Kind: "hate", Object: inspector.ObjectID, Target: pilgrim.ObjectID, Value: 99999},
		{Kind: "attack", Object: inspector.ObjectID, Target: pilgrim.ObjectID},
	}; !slices.Equal(h.World().Actions, want) {
		t.Errorf("Ol Mahums: actions %+v, want %+v", h.World().Actions, want)
	}
	h.Advance(5 * time.Second) // the despawn check: not yet
	if pilgrim.Deleted || inspector.Deleted {
		t.Fatal("the Ol Mahums left too early")
	}
	h.Kill(hero, inspector)
	b2Quest(t, "pilgrim", hero, b2Q227, scripting.StateStarted, 7)
	reply, ok = h.Talk(b2Q227, hero, pilgrim)
	b2Page(t, "pilgrim", reply, ok, "30732-01.htm")
	b2Quest(t, "money", hero, b2Q227, scripting.StateStarted, 8)
	b2Items(t, "money", hero, map[int]int64{2826: 1})
	h.Advance(5 * time.Second)
	if !pilgrim.Deleted {
		t.Error("the pilgrim is still there at cond 8")
	}
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q227, hero, katari)
	b2Page(t, "Katari betrayer", reply, ok, "30668-02.htm")
	betrayer := b2Q227Spawn(t, h, "betrayer", 27130)
	// The betrayer runs away.
	if want := []scripttest.Action{
		{Kind: "run", Object: betrayer.ObjectID},
		{Kind: "move", Object: betrayer.ObjectID, Loc: scripting.Location{X: -7732, Y: 36787, Z: -3709}},
	}; !slices.Equal(h.World().Actions, want) || !betrayer.Running {
		t.Errorf("betrayer: actions %+v, want %+v", h.World().Actions, want)
	}
	h.Kill(hero, betrayer)
	b2Quest(t, "betrayer", hero, b2Q227, scripting.StateStarted, 9)
	b2Items(t, "betrayer", hero, map[int]int64{2833: 1})
	if n := h.World().Pending(); n != 0 {
		t.Errorf("%d timers left after the betrayer died", n)
	}
	reply, ok = h.Talk(b2Q227, hero, katari)
	b2Page(t, "Katari letter", reply, ok, "30668-03.htm")
	b2Quest(t, "Katari letter", hero, b2Q227, scripting.StateStarted, 10)
	b2Items(t, "Katari letter", hero, map[int]int64{2833: 0, 2827: 1})
	reply, ok = h.Talk(b2Q227, hero, sla)
	b2Page(t, "Sla greetings", reply, ok, "30666-06.htm")
	b2Quest(t, "greetings", hero, b2Q227, scripting.StateStarted, 11)
	b2Items(t, "greetings", hero, map[int]int64{2826: 0, 2825: 3})

	// Kakan's Crimson Werewolf leaves when hit with a skill not allowed.
	reply, ok = h.Talk(b2Q227, hero, kakan)
	b2Page(t, "Kakan", reply, ok, "30669-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q227, "30669-03.htm", hero, kakan)
	b2Quest(t, "werewolf", hero, b2Q227, scripting.StateStarted, 12)
	werewolf := b2Q227Spawn(t, h, "werewolf", 27131)
	h.Attack(hero, werewolf, 10)
	h.Advance(time.Second)
	if !werewolf.Deleted {
		t.Fatal("the werewolf stayed after a plain hit")
	}
	if want := []scripttest.Action{{Kind: "abort_attack", Object: werewolf.ObjectID}, {Kind: "say", Object: werewolf.ObjectID, Text: "Cowardly guy!"}, {Kind: "delete", Object: werewolf.ObjectID}}; !slices.Equal(h.World().Actions, want) {
		t.Errorf("werewolf: actions %+v, want %+v", h.World().Actions, want)
	}
	h.World().ClearRecords()
	h.Event(b2Q227, "30669-03.htm", hero, kakan)
	werewolf = b2Q227Spawn(t, h, "second werewolf", 27131)
	b2Q227Cast(t, h, hero, werewolf, 1177)
	h.Advance(time.Second)
	if werewolf.Deleted {
		t.Fatal("the werewolf left after Wind Strike")
	}
	h.Kill(hero, werewolf)
	b2Quest(t, "werewolf dead", hero, b2Q227, scripting.StateStarted, 13)
	reply, ok = h.Talk(b2Q227, hero, kakan)
	b2Page(t, "Kakan letter", reply, ok, "30669-04.htm")
	b2Quest(t, "Kakan letter", hero, b2Q227, scripting.StateStarted, 14)
	b2Items(t, "Kakan letter", hero, map[int]int64{2825: 2, 3037: 1})

	// Nyakuri's Krudel Lizardman.
	reply, ok = h.Talk(b2Q227, hero, nyakuri)
	b2Page(t, "Nyakuri", reply, ok, "30670-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q227, "30670-03.htm", hero, nyakuri)
	b2Quest(t, "Krudel", hero, b2Q227, scripting.StateStarted, 15)
	h.Kill(hero, b2Q227Spawn(t, h, "Krudel", 27132))
	b2Quest(t, "Krudel dead", hero, b2Q227, scripting.StateStarted, 16)
	reply, ok = h.Talk(b2Q227, hero, nyakuri)
	b2Page(t, "Nyakuri letter", reply, ok, "30670-04.htm")
	b2Quest(t, "Nyakuri letter", hero, b2Q227, scripting.StateStarted, 17)
	b2Items(t, "Nyakuri letter", hero, map[int]int64{2825: 1, 2828: 1})

	// Ramus's bone fragments.
	reply, ok = h.Talk(b2Q227, hero, ramus)
	b2Page(t, "Ramus", reply, ok, "30667-01.htm")
	b2Quest(t, "undead list", hero, b2Q227, scripting.StateStarted, 18)
	b2Items(t, "undead list", hero, map[int]int64{2825: 0, 2829: 1})
	for i, id := range []int{20404, 20104, 20102, 20022} {
		h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000020 + int32(i), ID: id})
	}
	b2Quest(t, "four bones", hero, b2Q227, scripting.StateStarted, 18)
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000030, ID: 20100})
	b2Quest(t, "five bones", hero, b2Q227, scripting.StateStarted, 19)
	b2Items(t, "five bones", hero, map[int]int64{2834: 1, 2835: 1, 2836: 1, 2837: 1, 2838: 1})
	reply, ok = h.Talk(b2Q227, hero, ramus)
	b2Page(t, "Ramus letter", reply, ok, "30667-03.htm")
	b2Quest(t, "Ramus letter", hero, b2Q227, scripting.StateStarted, 20)
	b2Items(t, "Ramus letter", hero, map[int]int64{2834: 0, 2835: 0, 2836: 0, 2837: 0, 2838: 0, 2829: 0, 2830: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q227, hero, sla)
	b2Page(t, "mark", reply, ok, "30666-07.htm")
	b2Quest(t, "mark", hero, b2Q227, scripting.StateCompleted, 20)
	b2Items(t, "mark", hero, map[int]int64{2827: 0, 3037: 0, 2828: 0, 2830: 0, 2821: 1, 7562: 60})
	b2ExpSp(t, "mark", hero, 164032, 17500)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q227, hero, sla)
	b2Completed(t, "completed", reply, ok)
}
