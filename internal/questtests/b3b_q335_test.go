package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ335 = "Q00335_TheSongOfTheHunter"

// b3bQ335Link is one request link of Tor's list.
func b3bQ335Link(id, text string) string {
	return `<a action="bypass Script Q00335_TheSongOfTheHunter 30745-request-` + id + `">` + text + "</a>"
}

// b3bQ335Spawned returns the template IDs the quest spawned.
func b3bQ335Spawned(w *scripttest.World) []int {
	var ids []int
	for _, n := range w.Spawned {
		ids = append(ids, n.ID)
	}
	return ids
}

// Song of the Hunter: Grey's first circle test (a spawned overlord's head
// and Windsus Aleph's skin), a request from Tor's list, Cybellin's blood
// crystals with her dagger, and quitting with twenty pins.
func TestB3bQ335Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ335)
	w := h.World()
	grey := &scripttest.Npc{ObjectID: 0x60000001, ID: 30744}
	tor := &scripttest.Npc{ObjectID: 0x60000002, ID: 30745}
	cybellin := &scripttest.Npc{ObjectID: 0x60000003, ID: 30746}
	basilisk := &scripttest.Npc{ObjectID: 0x60000004, ID: 20550}
	breka := &scripttest.Npc{ObjectID: 0x60000005, ID: 20271}
	haka := &scripttest.Npc{ObjectID: 0x60000006, ID: 27140}
	windsus := &scripttest.Npc{ObjectID: 0x60000007, ID: 20553}
	aleph := &scripttest.Npc{ObjectID: 0x60000008, ID: 27143}
	lizardman := &scripttest.Npc{ObjectID: 0x60000009, ID: 20578}

	// Tor and Cybellin know nobody without the quest.
	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 40}
	reply, ok := h.Talk(b3bQ335, stranger, tor)
	b3bHTML(t, "Tor stranger", reply, ok, scripting.DefaultNoQuestMsg)
	b3bNoQuest(t, "Tor stranger", stranger, b3bQ335)

	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 34}
	reply, ok = h.Talk(b3bQ335, young, grey)
	b3bPage(t, "level 34", reply, ok, "30744_01.htm")
	b3bQuest(t, "level 34", young, b3bQ335, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 35}
	reply, ok = h.Talk(b3bQ335, hero, grey)
	b3bPage(t, "offer", reply, ok, "30744_02.htm")
	reply, ok = h.Event(b3bQ335, "30744_03.htm", hero, grey)
	b3bPage(t, "accept", reply, ok, "30744_03.htm")
	b3bQuest(t, "accept", hero, b3bQ335, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{3695: 1})
	reply, ok = h.Talk(b3bQ335, hero, grey)
	b3bPage(t, "first circle", reply, ok, "30744_05.htm")

	h.SetRand(rnd.NewScripted(t,
		rnd.Get(100, 74), // basilisk scale (75%)
		rnd.Get(100, 9),  // Breka Orc Warrior calls Haka (10%)
		rnd.Get(100, 9),  // Windsus calls Windsus Aleph (10%)
		rnd.Get(100, 99), // Aleph's skin (100%)
	))
	h.Kill(hero, basilisk)
	h.Kill(hero, breka)
	h.Kill(hero, haka)
	h.Kill(hero, windsus)
	h.Kill(hero, aleph)
	if got, want := b3bQ335Spawned(w), []int{27140, 27143}; !slices.Equal(got, want) {
		t.Errorf("spawned %v, want %v", got, want)
	}
	b3bItems(t, "first circle kills", hero, map[int]int64{3709: 1, 3711: 1, 3714: 1})

	// Scales, heads and the skin: three of the six conditions.
	hero.Give(3709, 39)
	hero.Give(3712, 1)
	hero.Give(3713, 1)
	reply, ok = h.Talk(b3bQ335, hero, grey)
	b3bPage(t, "license", reply, ok, "30744_06.htm")
	b3bQuest(t, "license", hero, b3bQ335, scripting.StateStarted, 2)
	b3bItems(t, "license", hero, map[int]int64{3695: 0, 3709: 0, 3711: 0, 3712: 0, 3713: 0, 3714: 0, 3692: 1})
	reply, ok = h.Talk(b3bQ335, hero, grey)
	b3bPage(t, "below 45", reply, ok, "30744_07.htm")

	// Tor's list: five draws of rnd(12) without pins, sorted.
	reply, ok = h.Talk(b3bQ335, hero, tor)
	b3bPage(t, "Tor below 45", reply, ok, "30745_01b.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(12, 3), rnd.Get(12, 0), rnd.Get(12, 7), rnd.Get(12, 0), rnd.Get(12, 11)))
	reply, ok = h.Event(b3bQ335, "30745-list1", hero, tor)
	kadesh := b3bQ335Link("3727", "C: 40 Totems of Kadesh")
	want := "<html><head><body>Guild Member Tor:<br>" + kadesh + "<br>" + kadesh + "<br>" +
		b3bQ335Link("3730", "C: 30 Pieces Monster Eye Meat") + "<br>" +
		b3bQ335Link("3734", "C: 30 Windsus Tusks") + "<br>" +
		b3bQ335Link("3738", "C: 40 Ivory Charms of Tamlin") + "<br></body></html>"
	b3bHTML(t, "list", reply, ok, want)
	// The list is kept: no draws the second time.
	reply, ok = h.Event(b3bQ335, "30745-list1", hero, tor)
	b3bHTML(t, "list again", reply, ok, want)
	reply, ok = h.Event(b3bQ335, "30745-request-3734", hero, tor)
	b3bPage(t, "request", reply, ok, "30745-3734.htm")
	b3bItems(t, "request", hero, map[int]int64{3734: 1})
	reply, ok = h.Talk(b3bQ335, hero, tor)
	b3bPage(t, "request open", reply, ok, "30745_05.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49), rnd.Get(100, 50)))
	h.Kill(hero, windsus)
	h.Kill(hero, windsus)
	b3bItems(t, "tusks", hero, map[int]int64{3776: 1})
	hero.Give(3776, 29)
	reply, ok = h.Talk(b3bQ335, hero, tor)
	b3bPage(t, "request done", reply, ok, "30745_06a.htm")
	b3bItems(t, "request done", hero, map[int]int64{3734: 0, 3776: 0, 3694: 1, 57: 5200})
	if _, ok := hero.Quest(b3bQ335).Get("list"); ok {
		t.Error("the list stays after a request")
	}

	// Cybellin's blood crystals, raised only with her dagger in hand.
	reply, ok = h.Talk(b3bQ335, hero, cybellin)
	b3bPage(t, "Cybellin", reply, ok, "30746_02.htm")
	h.Event(b3bQ335, "30746_03.htm", hero, cybellin)
	b3bItems(t, "Cybellin's request", hero, map[int]int64{3471: 1, 3697: 1, 3698: 1})
	reply, ok = h.Talk(b3bQ335, hero, cybellin)
	b3bPage(t, "crystal 1", reply, ok, "30746_04.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49)))
	h.Kill(hero, lizardman) // bare hands: no draw
	b3bItems(t, "bare hands", hero, map[int]int64{3698: 1})
	const rhand = 7 // Paperdoll.RHAND
	hero.Equip(rhand, 3471)
	h.Kill(hero, lizardman)
	b3bItems(t, "crystal 2", hero, map[int]int64{3698: 0, 3699: 1})
	reply, ok = h.Talk(b3bQ335, hero, cybellin)
	b3bPage(t, "crystal 2", reply, ok, "30746_05.htm")
	reply, ok = h.Event(b3bQ335, "30746_06.htm", hero, cybellin)
	b3bPage(t, "crystals sold", reply, ok, "30746_06.htm")
	b3bItems(t, "crystals sold", hero, map[int]int64{3699: 0, 57: 8600})

	hero.Give(3694, 19)
	reply, ok = h.Event(b3bQ335, "30744_16.htm", hero, grey)
	b3bPage(t, "quit", reply, ok, "30744_17.htm")
	b3bNoQuest(t, "quit", hero, b3bQ335)
	b3bItems(t, "quit", hero, map[int]int64{57: 28600, 3694: 0, 3692: 0, 3471: 0, 3697: 0})
}
