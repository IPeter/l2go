package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Red-Eyed Invaders: Babenco, Bathis, a hundred each of the bone necklaces,
// then thirty each of the pouches and gems; fishing gear as the reward.
func TestB3aQ039RedEyedInvaders(t *testing.T) {
	q := b3aStart(t, "Q00039_RedEyedInvaders")
	babenco := &scripttest.Npc{ObjectID: 0x60000001, ID: 30334, Name: "Babenco"}
	bathis := &scripttest.Npc{ObjectID: 0x60000002, ID: 30332, Name: "Bathis"}
	lizardman := &scripttest.Npc{ObjectID: 0x60000003, ID: 20919, Name: "Maille Lizardman"}
	guard := &scripttest.Npc{ObjectID: 0x60000004, ID: 20921, Name: "Maille Lizardman Guard"}
	araneid := &scripttest.Npc{ObjectID: 0x60000005, ID: 20925, Name: "Araneid"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	q.talk(young, babenco, "30334-2.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	q.talk(hero, babenco, "30334-0.htm")
	q.event(hero, babenco, "30334-1.htm", "30334-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, babenco, "30334-3.htm")
	q.talk(hero, bathis, "30332-0.htm")
	q.kill(hero, lizardman) // nothing before Bathis
	q.event(hero, bathis, "30332-1.htm", "30332-1.htm")
	q.want("necklaces", hero, scripting.StateStarted, 2)
	q.talk(hero, bathis, "30332-2a.htm")

	// Lizardmen always drop; the guard drops the red necklace.
	q.kill(hero, lizardman, rnd.Float64(0.99))
	q.kill(hero, guard, rnd.Float64(0.99))
	q.kill(hero, araneid) // not in the first list: no draw
	b3aWantItems(t, "first drops", hero, map[int]int64{7178: 1, 7179: 1})
	hero.Give(7178, 98)
	hero.Give(7179, 99)
	q.kill(hero, lizardman, rnd.Float64(0))
	q.want("hundred each", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "hundred each", hero, map[int]int64{7178: 100, 7179: 100})
	q.talk(hero, bathis, "30332-2.htm")
	q.event(hero, bathis, "30332-3.htm", "30332-3.htm")
	q.want("pouches", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "pouches", hero, map[int]int64{7178: 0, 7179: 0})
	q.talk(hero, bathis, "30332-3a.htm")

	// The Araneid drops a gem half the time; plain lizardmen nothing now.
	q.kill(hero, araneid, rnd.Float64(0.51))
	q.kill(hero, lizardman)
	b3aWantItems(t, "misses", hero, map[int]int64{7181: 0})
	hero.Give(7181, 29)
	hero.Give(7180, 29)
	q.kill(hero, guard, rnd.Float64(0.29))
	q.want("29 gems", hero, scripting.StateStarted, 4)
	q.kill(hero, araneid, rnd.Float64(0.49))
	q.want("thirty each", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "thirty each", hero, map[int]int64{7180: 30, 7181: 30})
	q.talk(hero, bathis, "30332-4.htm")
	q.event(hero, bathis, "30332-5.htm", "30332-5.htm")
	q.want("done", hero, scripting.StateCompleted, 5)
	b3aWantItems(t, "done", hero, map[int]int64{7180: 0, 7181: 0, 6521: 60, 6529: 1, 6535: 500})
	q.talk(hero, babenco, b3aCompleted)
}
