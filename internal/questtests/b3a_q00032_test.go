package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// An Obvious Lie: Maximilian, Gentler and Miki the Cat; twenty Medicinal
// Herbs from Alligators, Spirit Ore, then thread and suede for the ears.
func TestB3aQ032AnObviousLie(t *testing.T) {
	q := b3aStart(t, "Q00032_AnObviousLie")
	maximilian := &scripttest.Npc{ObjectID: 0x60000001, ID: 30120, Name: "Maximilian"}
	gentler := &scripttest.Npc{ObjectID: 0x60000002, ID: 30094, Name: "Gentler"}
	miki := &scripttest.Npc{ObjectID: 0x60000003, ID: 31706, Name: "Miki the Cat"}
	alligator := &scripttest.Npc{ObjectID: 0x60000004, ID: 20135, Name: "Alligator"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 44}
	q.talk(young, maximilian, "30120-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 45}
	q.talk(hero, maximilian, "30120-0.htm")
	q.event(hero, maximilian, "30120-1.htm", "30120-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, maximilian, "30120-2.htm")
	q.talk(hero, gentler, "30094-0.htm")
	q.event(hero, gentler, "30094-1.htm", "30094-1.htm")
	q.want("map", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "map", hero, map[int]int64{7165: 1})
	// Alligators give nothing before Miki's word, and draw nothing.
	q.kill(hero, alligator)
	q.talk(hero, gentler, "30094-2.htm")
	q.talk(hero, miki, "31706-0.htm")
	q.event(hero, miki, "31706-1.htm", "31706-1.htm")
	q.want("herbs", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "herbs", hero, map[int]int64{7165: 0})
	q.talk(hero, miki, "31706-2.htm")

	q.h.World().ClearRecords()
	for range 20 {
		q.kill(hero, alligator, rnd.Float64(0.5))
	}
	q.want("twenty herbs", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "twenty herbs", hero, map[int]int64{7166: 20})
	if got := q.h.World().SoundsOf(hero); len(got) != 21 || got[18] != b3aSoundItemGet || got[19] != b3aSoundMiddle || got[20] != b3aSoundMiddle {
		t.Errorf("sounds = %v", got)
	}
	q.kill(hero, alligator)

	q.talk(hero, gentler, "30094-3.htm")
	q.event(hero, gentler, "30094-4.htm", "30094-4.htm")
	q.want("herbs given", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "herbs given", hero, map[int]int64{7166: 0})
	// No Spirit Ore: Gentler waits.
	q.talk(hero, gentler, "30094-5.htm")
	q.event(hero, gentler, "30094-7.htm", "30094-5.htm")
	q.want("no ore", hero, scripting.StateStarted, 5)
	hero.Give(3031, 500)
	q.talk(hero, gentler, "30094-6.htm")
	q.event(hero, gentler, "30094-7.htm", "30094-7.htm")
	q.want("ore", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "ore", hero, map[int]int64{3031: 0})
	q.talk(hero, gentler, "30094-8.htm")
	q.talk(hero, miki, "31706-3.htm")
	q.event(hero, miki, "31706-4.htm", "31706-4.htm")
	q.want("miki", hero, scripting.StateStarted, 7)
	q.talk(hero, miki, "31706-5.htm")
	q.talk(hero, gentler, "30094-9.htm")
	q.event(hero, gentler, "30094-10.htm", "30094-10.htm")
	q.want("materials", hero, scripting.StateStarted, 8)

	q.talk(hero, gentler, "30094-11.htm")
	q.event(hero, gentler, "cat", "30094-11.htm")
	hero.Give(1868, 1000)
	hero.Give(1866, 500)
	q.talk(hero, gentler, "30094-12.htm")
	q.event(hero, gentler, "racoon", "30094-14.htm")
	q.want("done", hero, scripting.StateCompleted, 8)
	b3aWantItems(t, "done", hero, map[int]int64{1868: 0, 1866: 0, 7680: 1, 6843: 0})
	q.talk(hero, maximilian, b3aCompleted)
}
