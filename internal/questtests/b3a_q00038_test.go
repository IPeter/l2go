package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Dragon Fangs: Luis, a hundred Feather Ornaments, Iris, Rohmer, Iris, fifty
// Dragon Teeth, a random reward.
func TestB3aQ038DragonFangs(t *testing.T) {
	q := b3aStart(t, "Q00038_DragonFangs")
	luis := &scripttest.Npc{ObjectID: 0x60000001, ID: 30386, Name: "Luis"}
	iris := &scripttest.Npc{ObjectID: 0x60000002, ID: 30034, Name: "Iris"}
	rohmer := &scripttest.Npc{ObjectID: 0x60000003, ID: 30344, Name: "Rohmer"}
	harpy := &scripttest.Npc{ObjectID: 0x60000004, ID: 21100, Name: "Langk Lizardman Sentinel"}
	dragon := &scripttest.Npc{ObjectID: 0x60000005, ID: 21101, Name: "Langk Lizardman Shaman"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 18}
	q.talk(young, luis, "30386-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 19}
	q.talk(hero, luis, "30386-01.htm")
	q.event(hero, luis, "30386-02.htm", "30386-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, luis, "30386-02a.htm")
	q.kill(hero, dragon) // the wrong cond: no draw

	hero.Give(7173, 98)
	q.h.World().ClearRecords()
	q.kill(hero, harpy, rnd.Get(1000000, 999999))
	q.kill(hero, harpy, rnd.Get(1000000, 0))
	q.want("feathers", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "feathers", hero, map[int]int64{7173: 100})
	if got := q.h.World().SoundsOf(hero); !slices.Equal(got, []string{b3aSoundItemGet, b3aSoundItemGet, b3aSoundMiddle}) {
		t.Errorf("sounds = %v", got)
	}
	q.talk(hero, luis, "30386-03.htm")
	q.event(hero, luis, "30386-04.htm", "30386-04.htm")
	q.want("totem", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "totem", hero, map[int]int64{7173: 0, 7174: 1})
	q.talk(hero, luis, "30386-03a.htm")

	q.talk(hero, iris, "30034-01.htm")
	q.event(hero, iris, "30034-02a.htm", "30034-02.htm")
	q.want("Iris", hero, scripting.StateStarted, 4)
	q.event(hero, iris, "30034-02a.htm", "30034-02a.htm") // the tooth is gone
	q.talk(hero, iris, "30034-02b.htm")
	q.talk(hero, rohmer, "30344-01.htm")
	q.event(hero, rohmer, "30344-02a.htm", "30344-02.htm")
	q.want("Rohmer", hero, scripting.StateStarted, 5)
	q.talk(hero, rohmer, "30344-03.htm")
	q.talk(hero, iris, "30034-03.htm")
	q.event(hero, iris, "30034-04a.htm", "30034-04.htm")
	q.want("teeth", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "teeth", hero, map[int]int64{7175: 0, 7176: 0, 7177: 0})
	q.talk(hero, iris, "30034-05a.htm")
	q.event(hero, iris, "30034-06a.htm", "30034-06a.htm") // no teeth yet

	hero.Give(7175, 48)
	q.kill(hero, dragon, rnd.Get(1000000, 500000)) // the drop chance is 50%
	q.kill(hero, dragon, rnd.Get(1000000, 499999))
	b3aWantItems(t, "49 teeth", hero, map[int]int64{7175: 49})
	q.kill(hero, dragon, rnd.Get(1000000, 7))
	q.want("fifty teeth", hero, scripting.StateStarted, 7)
	q.talk(hero, iris, "30034-05.htm")

	q.h.SetRand(rnd.NewScripted(t, rnd.Get(4, 2)))
	q.event(hero, iris, "30034-06a.htm", "30034-06.htm")
	q.want("done", hero, scripting.StateCompleted, 7)
	b3aWantItems(t, "done", hero, map[int]int64{7175: 0, 1123: 1, b3aAdena: 3200})
	q.talk(hero, luis, b3aCompleted)
}
