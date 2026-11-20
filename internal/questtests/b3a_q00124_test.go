package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Meeting the Elroki: Marquez, Mushika, Asamah, Karakawei, Mantarasa and
// the egg back to Asamah, level 75.
func TestB3aQ124MeetingTheElroki(t *testing.T) {
	q := b3aStart(t, "Q00124_MeetingTheElroki")
	marquez := &scripttest.Npc{ObjectID: 0x60000001, ID: 32113, Name: "Marquez"}
	mushika := &scripttest.Npc{ObjectID: 0x60000002, ID: 32114, Name: "Mushika"}
	asamah := &scripttest.Npc{ObjectID: 0x60000003, ID: 32115, Name: "Asamah"}
	karakawei := &scripttest.Npc{ObjectID: 0x60000004, ID: 32117, Name: "Karakawei"}
	mantarasa := &scripttest.Npc{ObjectID: 0x60000005, ID: 32118, Name: "Mantarasa"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	q.talk(young, marquez, "32113-01a.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	q.talk(hero, marquez, "32113-01.htm")
	q.event(hero, marquez, "32113-03.htm", "32113-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, marquez, "32113-03.htm")
	q.event(hero, marquez, "32113-04.htm", "32113-04.htm")
	q.want("Marquez", hero, scripting.StateStarted, 2)
	q.talk(hero, marquez, "32113-04a.htm")
	q.talk(hero, mushika, "32114-01.htm")
	q.event(hero, mushika, "32114-02.htm", "32114-02.htm")
	q.want("Mushika", hero, scripting.StateStarted, 3)
	q.talk(hero, mushika, "32114-03.htm")
	q.talk(hero, asamah, "32115-01.htm")
	q.event(hero, asamah, "32115-04.htm", "32115-04.htm")
	q.want("Asamah", hero, scripting.StateStarted, 4)
	q.talk(hero, karakawei, "32117-01.htm")
	q.event(hero, karakawei, "32117-02.htm", "32117-02.htm")
	q.talk(hero, karakawei, "32117-02.htm")
	q.event(hero, karakawei, "32117-03.htm", "32117-03.htm")
	q.want("Karakawei", hero, scripting.StateStarted, 5)
	q.talk(hero, karakawei, "32117-04.htm")
	q.talk(hero, mantarasa, "32118-01.htm")
	q.event(hero, mantarasa, "32118-02.htm", "32118-02.htm")
	q.want("Mantarasa", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "egg", hero, map[int]int64{8778: 1})
	q.talk(hero, mantarasa, "32118-03.htm")

	q.talk(hero, asamah, "32115-05.htm")
	q.want("done", hero, scripting.StateCompleted, 6)
	b3aWantItems(t, "done", hero, map[int]int64{8778: 0, b3aAdena: 71318})
	q.talk(hero, asamah, "32115-06.htm")
	q.talk(hero, marquez, b3aCompleted)
}
