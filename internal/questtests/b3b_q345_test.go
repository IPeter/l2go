package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ345 = "Q00345_MethodToRaiseTheDead"

// Method to Raise the Dead: the five bones, Xenovia's powder, the medium
// jar (the scared girl), Orpheus buys bone pieces, Dorothy pays.
func TestB3bQ345Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ345)
	npc := func(oid int32, id int) *scripttest.Npc { return &scripttest.Npc{ObjectID: oid, ID: id} }
	dorothy, xenovia, orpheus, jar := npc(0x60000001, 30970), npc(0x60000002, 30912), npc(0x60000003, 30971), npc(0x60000004, 30973)
	zombie := npc(0x60000010, 20789)

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 34}
	reply, ok := h.Talk(b3bQ345, young, dorothy)
	b3bPage(t, "level 34", reply, ok, "30970-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 35}
	reply, ok = h.Talk(b3bQ345, hero, dorothy)
	b3bPage(t, "offer", reply, ok, "30970-01.htm")
	h.Event(b3bQ345, "30970-03.htm", hero, dorothy)
	b3bQuest(t, "accept", hero, b3bQ345, scripting.StateStarted, 1)

	// rnd(4) == 0 picks a bone (rnd(4274, 4278)); else a useless piece. A
	// bone already held also becomes a piece.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(4, 0), rnd.Between(4274, 4278, 4274),
		rnd.Get(4, 1),
		rnd.Get(4, 0), rnd.Between(4274, 4278, 4274),
	))
	h.Kill(hero, zombie)
	h.Kill(hero, zombie)
	h.Kill(hero, zombie)
	b3bItems(t, "kills", hero, map[int]int64{4274: 1, 4280: 2})
	reply, ok = h.Talk(b3bQ345, hero, dorothy)
	b3bPage(t, "one bone", reply, ok, "30970-04.htm")
	for id := 4275; id <= 4278; id++ {
		hero.Give(id, 1)
	}
	reply, ok = h.Talk(b3bQ345, hero, dorothy)
	b3bPage(t, "all bones", reply, ok, "30970-05.htm")
	h.Event(b3bQ345, "30970-06.htm", hero, dorothy)
	b3bQuest(t, "bones", hero, b3bQ345, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3bQ345, hero, xenovia)
	b3bPage(t, "Xenovia", reply, ok, "30912-01.htm")
	reply, ok = h.Event(b3bQ345, "30912-04.htm", hero, xenovia)
	b3bPage(t, "no adena", reply, ok, "30912-04.htm")
	hero.Give(57, 1000)
	reply, ok = h.Event(b3bQ345, "30912-04.htm", hero, xenovia)
	b3bPage(t, "powder", reply, ok, "30912-03.htm")
	b3bQuest(t, "powder", hero, b3bQ345, scripting.StateStarted, 3)
	b3bItems(t, "powder", hero, map[int]int64{57: 0, 4281: 1})

	// Orpheus buys one piece for 104; one piece stays for Dorothy.
	reply, ok = h.Event(b3bQ345, "30971-02a.htm", hero, orpheus)
	b3bPage(t, "Orpheus", reply, ok, "30971-02.htm")
	hero.Items[4280] = 1
	h.Event(b3bQ345, "30971-03.htm", hero, orpheus)
	b3bItems(t, "Orpheus", hero, map[int]int64{4280: 0, 57: 104})
	hero.Give(4280, 1)

	reply, ok = h.Talk(b3bQ345, hero, jar)
	b3bPage(t, "jar", reply, ok, "30973-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(3, 2)))
	reply, ok = h.Event(b3bQ345, "30973-04.htm", hero, jar)
	b3bPage(t, "summon", reply, ok, "30973-02c.htm")
	b3bQuest(t, "summon", hero, b3bQ345, scripting.StateStarted, 7)
	b3bItems(t, "summon", hero, map[int]int64{4281: 0, 4274: 0, 4278: 0})

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 9)))
	reply, ok = h.Talk(b3bQ345, hero, dorothy)
	b3bPage(t, "reward", reply, ok, "30970-10.htm")
	b3bItems(t, "reward", hero, map[int]int64{4280: 0, 57: 104 + 3040 + 70, 3456: 1})
	b3bNoQuest(t, "reward", hero, b3bQ345)
}
