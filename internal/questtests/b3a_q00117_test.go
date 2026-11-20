package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The Ocean of Distant Stars, level 39: Abey, the ancient ghost, Obi, the
// box, a Grey Star from the bandits (20%), back to the ghosts.
func TestB3aQ117TheOceanOfDistantStars(t *testing.T) {
	q := b3aStart(t, "Q00117_TheOceanOfDistantStars")
	abey := &scripttest.Npc{ObjectID: 0x60000001, ID: 32053, Name: "Abey"}
	ghost := &scripttest.Npc{ObjectID: 0x60000002, ID: 32054, Name: "Ghost"}
	ancient := &scripttest.Npc{ObjectID: 0x60000003, ID: 32055, Name: "Ancient Ghost"}
	obi := &scripttest.Npc{ObjectID: 0x60000004, ID: 32052, Name: "Obi"}
	box := &scripttest.Npc{ObjectID: 0x60000005, ID: 32076, Name: "Box"}
	bandit := &scripttest.Npc{ObjectID: 0x60000006, ID: 22023, Name: "Bandit Warrior"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 38}
	q.talk(young, abey, "32053-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 39}
	q.talk(hero, abey, "32053-01.htm")
	q.event(hero, abey, "32053-02.htm", "32053-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, ancient, "32055-01.htm")
	q.event(hero, ancient, "32055-02.htm", "32055-02.htm")
	q.want("ancient ghost", hero, scripting.StateStarted, 2)
	q.talk(hero, obi, "32052-01.htm")
	q.event(hero, obi, "32052-02.htm", "32052-02.htm")
	q.want("Obi", hero, scripting.StateStarted, 3)
	q.talk(hero, abey, "32053-03.htm")
	q.event(hero, abey, "32053-04.htm", "32053-04.htm")
	q.want("Abey", hero, scripting.StateStarted, 4)
	q.talk(hero, box, "32076-01.htm")
	q.event(hero, box, "32076-02.htm", "32076-02.htm")
	q.want("hammer", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "hammer", hero, map[int]int64{8488: 1})
	q.talk(hero, abey, "32053-05.htm")
	q.event(hero, abey, "32053-06.htm", "32053-06.htm")
	q.want("Abey 6", hero, scripting.StateStarted, 6)

	// Before cond 7 the bandits draw nothing.
	q.kill(hero, bandit)
	q.talk(hero, obi, "32052-03.htm")
	q.event(hero, obi, "32052-04.htm", "32052-04.htm")
	q.want("stars", hero, scripting.StateStarted, 7)

	q.kill(hero, bandit, rnd.Float64(0.5))
	b3aWantItems(t, "missed", hero, map[int]int64{8495: 0})
	q.kill(hero, bandit, rnd.Float64(0.1))
	q.want("star", hero, scripting.StateStarted, 8)
	b3aWantItems(t, "star", hero, map[int]int64{8495: 1})

	q.talk(hero, obi, "32052-05.htm")
	q.event(hero, obi, "32052-06.htm", "32052-06.htm")
	q.want("star given", hero, scripting.StateStarted, 9)
	b3aWantItems(t, "star given", hero, map[int]int64{8495: 0})
	q.talk(hero, ancient, "32055-03.htm")
	q.event(hero, ancient, "32055-04.htm", "32055-04.htm")
	q.want("hammer given", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "hammer given", hero, map[int]int64{8488: 0})
	q.talk(hero, ghost, "32054-01.htm")
	q.event(hero, ghost, "32054-03.htm", "32054-03.htm")
	q.want("done", hero, scripting.StateCompleted, 10)
	b3aWantExp(t, "done", hero, 63591, 0)
	q.talk(hero, abey, b3aCompleted)
}
