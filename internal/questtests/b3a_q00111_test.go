package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Elrokian Hunter's Proof, level 75. As in Mobius, the monsters drop
// nothing (the drops wait for a memo state the quest never sets), so the
// test gives the drops by hand where the kills would.
func TestB3aQ111ElrokianHuntersProof(t *testing.T) {
	q := b3aStart(t, "Q00111_ElrokianHuntersProof")
	marquez := &scripttest.Npc{ObjectID: 0x60000001, ID: 32113, Name: "Marquez"}
	mushika := &scripttest.Npc{ObjectID: 0x60000002, ID: 32114, Name: "Mushika"}
	asamah := &scripttest.Npc{ObjectID: 0x60000003, ID: 32115, Name: "Asamah"}
	kirikashin := &scripttest.Npc{ObjectID: 0x60000004, ID: 32116, Name: "Kirikashin"}
	raptor := &scripttest.Npc{ObjectID: 0x60000005, ID: 22196, Name: "Velociraptor"}
	ornithomimus := &scripttest.Npc{ObjectID: 0x60000006, ID: 22200, Name: "Ornithomimus"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	q.talk(young, marquez, "32113-000.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	q.talk(hero, marquez, "32113-001.htm")
	q.event(hero, marquez, "32113-002.htm", "32113-002.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, marquez, "32113-002.htm")
	q.talk(hero, kirikashin, "32116-008.htm")
	q.talk(hero, mushika, "32114-001.htm")
	q.want("Mushika", hero, scripting.StateStarted, 2)
	q.talk(hero, asamah, "32115-001.htm")
	q.event(hero, asamah, "32115-002.htm", "32115-002.htm")
	q.want("Asamah", hero, scripting.StateStarted, 3)
	q.talk(hero, marquez, "32113-003.htm")
	q.event(hero, marquez, "32113-009.htm", "32113-009.htm")
	q.want("diaries", hero, scripting.StateStarted, 4)

	// No drop and no draw at cond 4.
	q.kill(hero, raptor)
	b3aWantItems(t, "raptor", hero, map[int]int64{8768: 0})
	hero.Give(8768, 50)
	hero.Quest(q.name).Set("cond", "5")

	q.talk(hero, marquez, "32113-010.htm")
	q.event(hero, marquez, "32113-018.htm", "32113-018.htm")
	q.want("letter", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "letter", hero, map[int]int64{8768: 0, 8769: 1})
	q.talk(hero, kirikashin, "32116-001.htm")
	b3aWantItems(t, "letter given", hero, map[int]int64{8769: 0})
	q.event(hero, kirikashin, "32116-003.htm", "32116-003.htm")
	q.want("song", hero, scripting.StateStarted, 7)
	q.talk(hero, kirikashin, "32116-004.htm")
	q.event(hero, kirikashin, "32116-005.htm", "32116-005.htm")
	q.want("to Asamah", hero, scripting.StateStarted, 8)
	q.talk(hero, asamah, "32115-003.htm")
	q.event(hero, asamah, "32115-004.htm", "32115-004.htm")
	q.want("Asamah 9", hero, scripting.StateStarted, 9)
	q.talk(hero, asamah, "32115-004.htm")
	q.event(hero, asamah, "32115-006.htm", "32115-006.htm")
	q.want("hunt", hero, scripting.StateStarted, 10)
	q.talk(hero, asamah, "32115-006.htm")

	q.kill(hero, ornithomimus)
	b3aWantItems(t, "ornithomimus", hero, map[int]int64{8770: 0})
	hero.Give(8770, 10)
	hero.Give(8771, 10)
	hero.Give(8772, 10)
	hero.Quest(q.name).Set("cond", "11")

	q.talk(hero, asamah, "32115-007.htm")
	q.want("trap", hero, scripting.StateStarted, 12)
	b3aWantItems(t, "trap", hero, map[int]int64{8770: 0, 8771: 0, 8772: 0, 8773: 1})
	q.talk(hero, kirikashin, "32116-006.htm")
	q.event(hero, kirikashin, "32116-007.htm", "32116-007.htm")
	q.want("done", hero, scripting.StateCompleted, 12)
	b3aWantItems(t, "done", hero, map[int]int64{8773: 0, 8763: 1, 8764: 100, b3aAdena: 1022636})
	q.talk(hero, marquez, b3aCompleted)
}
