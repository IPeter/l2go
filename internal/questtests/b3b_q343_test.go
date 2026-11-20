package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ343 = "Q00343_UnderTheShadowOfTheIvoryTower"

// b3bQ343Event runs an event of Q343: every event draws rnd(3) and rnd(2)
// first, so the draws are scripted with it.
func b3bQ343Event(t *testing.T, h *scripttest.Harness, event string, p *scripttest.Player, n *scripttest.Npc, r1, r2 int, more ...rnd.Draw) (scripting.Reply, bool) {
	t.Helper()
	h.SetRand(rnd.NewScripted(t, append([]rnd.Draw{rnd.Get(3, r1), rnd.Get(2, r2)}, more...)...))
	return h.Event(b3bQ343, event, p, n)
}

// Under the Shadow of the Ivory Tower: orbs from the monsters, Cema buys
// them, Marsha's and Trumpin's games, Icarus's ectoplasm exchange.
func TestB3bQ343Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ343)
	npc := func(oid int32, id int) *scripttest.Npc { return &scripttest.Npc{ObjectID: oid, ID: id} }
	cema, icarus, marsha, trumpin := npc(0x60000001, 30834), npc(0x60000002, 30835), npc(0x60000003, 30934), npc(0x60000004, 30935)
	monster := npc(0x60000010, 20563)

	// Only level 40 wizards, sorcerers and the like.
	for _, p := range []*scripttest.Player{
		{ObjectID: 0x10000001, Name: "Fighter", Level: 40, ClassID: 0},
		{ObjectID: 0x10000002, Name: "Young", Level: 39, ClassID: 11},
	} {
		reply, ok := h.Talk(b3bQ343, p, cema)
		b3bPage(t, p.Name, reply, ok, "30834-07.htm")
	}

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 40, ClassID: 11}
	reply, ok := h.Talk(b3bQ343, hero, cema)
	b3bPage(t, "offer", reply, ok, "30834-01.htm")
	b3bQ343Event(t, h, "30834-03.htm", hero, cema, 0, 0)
	b3bQuest(t, "accept", hero, b3bQ343, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ343, hero, cema)
	b3bPage(t, "no orbs", reply, ok, "30834-05.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49), rnd.Get(100, 50)))
	h.Kill(hero, monster)
	h.Kill(hero, monster)
	b3bItems(t, "kills", hero, map[int]int64{4364: 1})
	reply, ok = h.Talk(b3bQ343, hero, cema)
	b3bPage(t, "orbs", reply, ok, "30834-06.htm")
	b3bQ343Event(t, h, "30834-08.htm", hero, cema, 0, 0)
	b3bItems(t, "sell", hero, map[int]int64{4364: 0, 57: 125})

	// Marsha: ten orbs on the table, random1 = 2 wins twenty.
	reply, ok = h.Talk(b3bQ343, hero, marsha)
	b3bPage(t, "Marsha", reply, ok, "30934-01.htm")
	reply, ok = b3bQ343Event(t, h, "30934-03.htm", hero, marsha, 0, 0)
	b3bPage(t, "no orbs to play", reply, ok, "noorbs.htm")
	hero.Give(4364, 10)
	b3bQ343Event(t, h, "30934-03.htm", hero, marsha, 0, 0)
	b3bItems(t, "stake", hero, map[int]int64{4364: 0})
	reply, ok = b3bQ343Event(t, h, "30934-04.htm", hero, marsha, 2, 0)
	b3bPage(t, "Marsha win", reply, ok, "30934-04.htm")
	b3bItems(t, "Marsha win", hero, map[int]int64{4364: 20})

	// Trumpin: one won toss (random2 = 0), then take the prize.
	reply, ok = b3bQ343Event(t, h, "30935-05.htm", hero, trumpin, 0, 0)
	b3bPage(t, "toss", reply, ok, "30935-04.htm")
	b3bQ343Event(t, h, "30935-06.htm", hero, trumpin, 0, 0)
	b3bItems(t, "Trumpin prize", hero, map[int]int64{4364: 30})

	// Icarus trades an ectoplasm: rnd(1000) <= 119 gives item 955.
	reply, ok = b3bQ343Event(t, h, "30835-02.htm", hero, icarus, 0, 0)
	b3bPage(t, "no ectoplasm", reply, ok, "30835-03.htm")
	hero.Give(4365, 1)
	b3bQ343Event(t, h, "30835-02.htm", hero, icarus, 0, 0, rnd.Get(1000, 119))
	b3bItems(t, "ectoplasm", hero, map[int]int64{4365: 0, 955: 1})

	// Picking a card without a stake: the orbs and the quest are gone.
	reply, ok = b3bQ343Event(t, h, "30934-05.htm", hero, marsha, 0, 0)
	if !ok || reply.Kind != scripting.ReplyText || reply.Content != "Player is cheating" {
		t.Fatalf("cheating: reply = %+v, %v", reply, ok)
	}
	b3bItems(t, "cheating", hero, map[int]int64{4364: 0})
	b3bNoQuest(t, "cheating", hero, b3bQ343)
}
