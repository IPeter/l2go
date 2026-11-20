package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Kamael: A Window to the Future: Dominic's tour of the five races'
// marks, Rodemai's story and back to Dominic.
func TestB3aQ127KamaelAWindowToTheFuture(t *testing.T) {
	const name = "Q00127_KamaelAWindowToTheFuture"
	q := b3aStart(t, name)
	dominic := &scripttest.Npc{ObjectID: 0x60000001, ID: 31350, Name: "Dominic"}
	klaus := &scripttest.Npc{ObjectID: 0x60000002, ID: 30187, Name: "Klaus"}
	alder := &scripttest.Npc{ObjectID: 0x60000003, ID: 32092, Name: "Alder"}
	aklan := &scripttest.Npc{ObjectID: 0x60000004, ID: 31288, Name: "Aklan"}
	oltlin := &scripttest.Npc{ObjectID: 0x60000005, ID: 30862, Name: "Oltlin"}
	juris := &scripttest.Npc{ObjectID: 0x60000006, ID: 30113, Name: "Juris"}
	rodemai := &scripttest.Npc{ObjectID: 0x60000007, ID: 30756, Name: "Rodemai"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 1}
	// Links before any state just show their page and start nothing.
	q.event(hero, dominic, "31350-04.htm", "31350-04.htm")
	if hero.Quest(name) != nil {
		t.Fatal("a link made a state")
	}
	q.talk(hero, dominic, "31350-01.htm")
	q.event(hero, dominic, "31350-04.htm", "31350-04.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{8939: 1})
	// Out of order: nobody else has anything to say yet.
	q.talk(hero, alder, b3aNoQuest)
	q.talk(hero, dominic, b3aNoQuest)

	q.talk(hero, klaus, "30187-01.htm")
	q.event(hero, klaus, "30187-06.htm", "30187-06.htm")
	q.want("Klaus", hero, scripting.StateStarted, 2)
	q.talk(hero, klaus, "30187-06.htm")
	q.event(hero, klaus, "30187-08.htm", "30187-08.htm")
	q.want("human", hero, scripting.StateStarted, 3)
	q.talk(hero, alder, "32092-01.htm")
	q.event(hero, alder, "32092-05.htm", "32092-05.htm")
	q.want("dwarf", hero, scripting.StateStarted, 4)
	q.talk(hero, aklan, "31288-01.htm")
	q.event(hero, aklan, "31288-04.htm", "31288-04.htm")
	q.want("orc", hero, scripting.StateStarted, 5)
	q.talk(hero, oltlin, "30862-01.htm")
	q.event(hero, oltlin, "30862-04.htm", "30862-04.htm")
	q.want("dark elf", hero, scripting.StateStarted, 6)
	q.talk(hero, juris, "30113-01.htm")
	q.event(hero, juris, "30113-04.htm", "30113-04.htm")
	q.want("elf", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "marks", hero, map[int]int64{8940: 1, 8941: 1, 8944: 1, 8943: 1, 8942: 1})
	q.talk(hero, rodemai, "30756-01.htm")
	q.event(hero, rodemai, "kamaelstory", "")
	q.want("story", hero, scripting.StateStarted, 8)
	q.talk(hero, rodemai, "30756-04.htm")
	q.event(hero, rodemai, "30756-05.htm", "30756-05.htm")
	q.want("Rodemai", hero, scripting.StateStarted, 9)

	q.talk(hero, dominic, "31350-05.htm")
	q.event(hero, dominic, "31350-06.htm", "31350-06.htm")
	q.want("done", hero, scripting.StateCompleted, 9)
	b3aWantItems(t, "done", hero, map[int]int64{8939: 0, 8940: 0, 8941: 0, 8944: 0, 8943: 0, 8942: 0, b3aAdena: 159100})
	q.talk(hero, dominic, b3aCompleted)
}
