package questtests

import (
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	b3cQ610 = "Q00610_MagicalPowerOfWaterPart2"
	b3cQ616 = "Q00616_MagicalPowerOfFirePart2"
)

// b3cSummonCase describes the two raid summon quests: a totem item summons
// the raid boss at the alliance totem, its death gives the heart to everyone
// in range and starts the boss's respawn delay.
type b3cSummonCase struct {
	quest               string
	master, altar, raid int
	totem, heart        int
	respawnVar          string
}

func TestB3cQ610Walkthrough(t *testing.T) {
	b3cWalkSummon(t, b3cSummonCase{quest: b3cQ610, master: 31372, altar: 31560, raid: 25316, totem: 7238, heart: 7239, respawnVar: "Q00610_respawn"})
}

func TestB3cQ616Walkthrough(t *testing.T) {
	b3cWalkSummon(t, b3cSummonCase{quest: b3cQ616, master: 31379, altar: 31558, raid: 25306, totem: 7243, heart: 7244, respawnVar: "Q00616_respawn"})
}

func b3cWalkSummon(t *testing.T, c b3cSummonCase) {
	h := b3cLoad(t, c.quest)
	w := h.World()
	h.SetRand(rnd.NewScripted(t))
	master := &scripttest.Npc{ObjectID: 0x60000001, ID: c.master}
	mPage := func(suffix string) string { return strconv.Itoa(c.master) + suffix }
	aPage := func(suffix string) string { return strconv.Itoa(c.altar) + suffix }
	lastSpawn := func(what string, id int) *scripttest.Npc {
		t.Helper()
		if len(w.Spawned) == 0 || w.Spawned[len(w.Spawned)-1].ID != id {
			t.Fatalf("%s: spawned %+v, want %d last", what, w.Spawned, id)
		}
		return w.Spawned[len(w.Spawned)-1]
	}

	// The constructor's totem.
	h.Start()
	altar := lastSpawn("start", c.altar)

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	young.Give(c.totem, 1)
	reply, ok := h.Talk(c.quest, young, master)
	b3cPage(t, "level 74", reply, ok, mPage("-00b.html"))
	empty := &scripttest.Player{ObjectID: 0x10000003, Name: "Empty", Level: 75}
	reply, ok = h.Talk(c.quest, empty, master)
	b3cPage(t, "no totem", reply, ok, mPage("-00a.html"))

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	hero.Give(c.totem, 1)
	reply, ok = h.Talk(c.quest, hero, master)
	b3cPage(t, "offer", reply, ok, mPage("-01.htm"))
	h.Event(c.quest, mPage("-02.html"), hero, master)
	b3cQuest(t, "accept", hero, c.quest, scripting.StateStarted, 1)
	reply, ok = h.Talk(c.quest, hero, master)
	b3cPage(t, "cond 1", reply, ok, mPage("-03.html"))
	reply, ok = h.Talk(c.quest, hero, altar)
	b3cPage(t, "altar cond 1", reply, ok, aPage("-01.html"))

	// The totem summons the boss in place of the altar.
	reply, ok = h.Event(c.quest, "spawn_totem", hero, altar)
	b3cPage(t, "summon", reply, ok, aPage("-02.html"))
	b3cQuest(t, "summon", hero, c.quest, scripting.StateStarted, 2)
	b3cItems(t, "summon", hero, map[int]int64{c.totem: 0})
	if !altar.Deleted {
		t.Error("the altar stays after the summon")
	}
	boss := lastSpawn("summon", c.raid)

	// Not killed in 20 minutes: the boss leaves, the altar comes back.
	h.Advance(20 * time.Minute)
	if !boss.Deleted {
		t.Error("the boss stays after 20 minutes")
	}
	altar = lastSpawn("despawn", c.altar)

	// At cond 2 the altar summons again without a totem.
	reply, ok = h.Talk(c.quest, hero, altar)
	b3cPage(t, "summon again", reply, ok, aPage("-02.html"))
	boss = lastSpawn("summon again", c.raid)
	reply, ok = h.Talk(c.quest, hero, master)
	b3cPage(t, "no heart", reply, ok, mPage("-05.html"))

	// The kill: the heart, the respawn delay drawn in [12 h, 36 h].
	hero.X, hero.Y, hero.Z = boss.X, boss.Y, boss.Z
	const delay = 50000000
	h.SetRand(rnd.NewScripted(t, rnd.Between(43200000, 129600000, delay)))
	h.Kill(hero, boss)
	b3cQuest(t, "kill", hero, c.quest, scripting.StateStarted, 3)
	b3cItems(t, "kill", hero, map[int]int64{c.heart: 1})
	if v, ok := w.Global(c.respawnVar); !ok || v == "" {
		t.Errorf("respawn variable %s not set", c.respawnVar)
	}
	reply, ok = h.Talk(c.quest, hero, altar)
	b3cPage(t, "altar cond 3", reply, ok, aPage("-05.html"))

	// While the boss is down nobody can summon it.
	other := &scripttest.Player{ObjectID: 0x10000004, Name: "Other", Level: 76}
	other.Give(c.totem, 1)
	h.Talk(c.quest, other, master)
	h.Event(c.quest, mPage("-02.html"), other, master)
	reply, ok = h.Event(c.quest, "spawn_totem", other, altar)
	b3cPage(t, "respawn wait", reply, ok, aPage("-03.html"))
	b3cQuest(t, "respawn wait", other, c.quest, scripting.StateStarted, 1)
	reply, ok = h.Event(c.quest, "give_heart", other, master)
	b3cPage(t, "no heart to give", reply, ok, mPage("-07.html"))

	reply, ok = h.Talk(c.quest, hero, master)
	b3cPage(t, "heart", reply, ok, mPage("-04.html"))
	reply, ok = h.Event(c.quest, "give_heart", hero, master)
	b3cPage(t, "reward", reply, ok, mPage("-06.html"))
	b3cNoQuest(t, "reward", hero, c.quest)
	b3cItems(t, "reward", hero, map[int]int64{c.heart: 0})
	if hero.Exp != 10000 {
		t.Errorf("exp = %d, want 10000", hero.Exp)
	}

	// The altar returns with the boss's respawn.
	spawned := len(w.Spawned)
	h.Advance(delay * time.Millisecond)
	if len(w.Spawned) != spawned+1 {
		t.Fatalf("no altar after the respawn delay: %+v", w.Spawned)
	}
	lastSpawn("respawn", c.altar)
}
