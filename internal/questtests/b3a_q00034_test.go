package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// In Search of Cloth: Radia, Varan, Ralford, ten Spinnerets from Trisalim
// spiders, then suede and thread for the Mysterious Cloth.
func TestB3aQ034InSearchOfCloth(t *testing.T) {
	q := b3aStart(t, "Q00034_InSearchOfCloth")
	radia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30088, Name: "Radia"}
	ralford := &scripttest.Npc{ObjectID: 0x60000002, ID: 30165, Name: "Ralford"}
	varan := &scripttest.Npc{ObjectID: 0x60000003, ID: 30294, Name: "Varan"}
	spider := &scripttest.Npc{ObjectID: 0x60000004, ID: 20560, Name: "Trisalim Spider"}
	tarantula := &scripttest.Npc{ObjectID: 0x60000005, ID: 20561, Name: "Trisalim Tarantula"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	q.talk(young, radia, "30088-0b.htm")
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	q.talk(hero, radia, "30088-0a.htm")
	b3aQ033FormalWear(q, hero, "6")
	q.talk(hero, radia, "30088-0.htm")

	q.event(hero, radia, "30088-1.htm", "30088-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, radia, "30088-1a.htm")
	q.talk(hero, varan, "30294-0.htm")
	q.event(hero, varan, "30294-1.htm", "30294-1.htm")
	q.want("Varan", hero, scripting.StateStarted, 2)
	q.talk(hero, varan, "30294-1a.htm")
	q.talk(hero, radia, "30088-2.htm")
	q.event(hero, radia, "30088-3.htm", "30088-3.htm")
	q.want("Ralford", hero, scripting.StateStarted, 3)
	q.talk(hero, radia, "30088-3a.htm")
	// Spiders give nothing before Ralford's word, and draw nothing.
	q.kill(hero, spider)
	q.talk(hero, ralford, "30165-0.htm")
	q.event(hero, ralford, "30165-1.htm", "30165-1.htm")
	q.want("spinnerets", hero, scripting.StateStarted, 4)
	q.talk(hero, ralford, "30165-1a.htm")

	q.kill(hero, spider, rnd.Bool(false))
	b3aWantItems(t, "miss", hero, map[int]int64{7528: 0})
	for i := range 10 {
		if i%2 == 0 {
			q.kill(hero, spider, rnd.Bool(true))
		} else {
			q.kill(hero, tarantula, rnd.Bool(true))
		}
	}
	q.want("ten spinnerets", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "ten spinnerets", hero, map[int]int64{7528: 10})
	q.kill(hero, spider)

	q.talk(hero, ralford, "30165-2.htm")
	q.event(hero, ralford, "30165-3.htm", "30165-3.htm")
	q.want("silk", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "silk", hero, map[int]int64{7528: 0, 7161: 1})
	q.talk(hero, ralford, "30165-3a.htm")

	q.talk(hero, radia, "30088-4a.htm")
	q.event(hero, radia, "30088-5.htm", "30088-4a.htm")
	hero.Give(1866, 3000)
	hero.Give(1868, 5000)
	q.talk(hero, radia, "30088-4.htm")
	q.event(hero, radia, "30088-5.htm", "30088-5.htm")
	q.want("done", hero, scripting.StateCompleted, 6)
	b3aWantItems(t, "done", hero, map[int]int64{7161: 0, 1866: 0, 1868: 0, 7076: 1})
	q.talk(hero, radia, b3aCompleted)
}
