package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Resurrection of an Old Manager, level 49 after Pavel the Giant: Yumi and
// Wendy, Wendy's golem (the third choice), the detector, the stones, the
// starstone bought from Wendy.
func TestB3aQ114ResurrectionOfAnOldManager(t *testing.T) {
	q := b3aStart(t, "Q00114_ResurrectionOfAnOldManager")
	w := q.h.World()
	yumi := &scripttest.Npc{ObjectID: 0x60000001, ID: 32041, Name: "Yumi"}
	stone := &scripttest.Npc{ObjectID: 0x60000002, ID: 32046, Name: "Stones"}
	wendy := &scripttest.Npc{ObjectID: 0x60000003, ID: 32047, Name: "Wendy"}
	box := &scripttest.Npc{ObjectID: 0x60000004, ID: 32050, Name: "Box"}

	// Pavel the Giant must be done, and the level 49.
	nopavel := &scripttest.Player{ObjectID: 0x10000001, Name: "NoPavel", Level: 60}
	q.talk(nopavel, yumi, "32041-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 48}
	w.NewQuestState(young.Ref(), "Q00121_PavelTheGiant").SetState(scripting.StateCompleted)
	q.talk(young, yumi, "32041-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 49}
	w.NewQuestState(hero.Ref(), "Q00121_PavelTheGiant").SetState(scripting.StateCompleted)
	q.talk(hero, yumi, "32041-01.htm")
	q.event(hero, yumi, "32041-02.htm", "32041-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, yumi, "32041-02.htm")
	q.event(hero, yumi, "32041-06.htm", "32041-06.htm")
	q.talk(hero, yumi, "32041-06.htm")
	q.event(hero, yumi, "32041-07.htm", "32041-07.htm")
	q.want("to Wendy", hero, scripting.StateStarted, 2)
	q.talk(hero, yumi, "32041-08.htm")

	q.talk(hero, wendy, "32047-01.htm")
	q.event(hero, wendy, "32047-02.htm", "32047-02.htm")
	q.event(hero, wendy, "32047-03.htm", "32047-03.htm")
	q.talk(hero, wendy, "32047-04.htm")
	q.event(hero, wendy, "32047-07.htm", "32047-07.htm")
	q.want("choice 3", hero, scripting.StateStarted, 5)
	q.talk(hero, wendy, "32047-09a.htm")

	q.talk(hero, yumi, "32041-09.htm")
	q.event(hero, yumi, "32041-10.htm", "32041-10b.htm")
	q.event(hero, yumi, "32041-11.htm", "32041-11.htm")
	q.talk(hero, yumi, "32041-11.htm")
	q.event(hero, yumi, "32041-18.htm", "32041-18.htm")
	q.talk(hero, yumi, "32041-18.htm")
	q.event(hero, yumi, "32041-20.htm", "32041-20.htm")
	q.want("back to Wendy", hero, scripting.StateStarted, 6)
	q.talk(hero, yumi, "32041-21.htm")

	q.talk(hero, wendy, "32047-14.htm")
	q.event(hero, wendy, "32047-13a.htm", "32047-13a.htm")
	q.want("golem", hero, scripting.StateStarted, 10)
	q.talk(hero, wendy, "32047-14a.htm")
	w.ClearRecords()
	q.event(hero, wendy, "32047-15a.htm", "32047-15a.htm")
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 27318 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	// The golem hates and attacks the player.
	golem := w.Spawned[0].ObjectID
	var hate, attack bool
	for _, a := range w.Actions {
		switch {
		case a.Kind == "hate" && a.Object == golem && a.Target == hero.ObjectID && a.Value == 999:
			hate = true
		case a.Kind == "attack" && a.Object == golem && a.Target == hero.ObjectID:
			attack = true
		}
	}
	if !hate || !attack {
		t.Errorf("golem actions %+v, want hate 999 and an attack on the player", w.Actions)
	}
	// One golem at a time, until it leaves after 15 minutes.
	q.event(hero, wendy, "32047-15a.htm", "32047-19a.htm")
	q.h.Advance(900 * time.Second)
	q.event(hero, wendy, "32047-15a.htm", "32047-15a.htm")
	if len(w.Spawned) != 2 {
		t.Fatalf("spawned %d golems, want 2", len(w.Spawned))
	}
	q.kill(hero, w.Spawned[1])
	q.want("golem beaten", hero, scripting.StateStarted, 11)
	q.talk(hero, wendy, "32047-16a.htm")
	q.event(hero, wendy, "32047-17a.htm", "32047-17a.htm")
	q.want("Wendy 12", hero, scripting.StateStarted, 12)
	q.talk(hero, wendy, "32047-18a.htm")
	q.talk(hero, box, b3aNoQuest)

	q.talk(hero, yumi, "32041-22.htm")
	q.event(hero, yumi, "32041-25.htm", "32041-25.htm")
	q.want("detector", hero, scripting.StateStarted, 17)
	b3aWantItems(t, "detector", hero, map[int]int64{8090: 1})
	q.talk(hero, yumi, "32041-26.htm")
	w.ClearRecords()
	q.talk(hero, stone, "")
	q.want("stones", hero, scripting.StateStarted, 18)
	b3aWantItems(t, "stones", hero, map[int]int64{8090: 0, 8091: 1})
	if len(w.Actions) != 1 || w.Actions[0].Kind != "screen" {
		t.Errorf("actions %+v, want the screen message", w.Actions)
	}
	q.talk(hero, stone, "32046-01.htm")
	q.event(hero, stone, "32046-02.htm", "32046-02.htm")
	q.want("stones 19", hero, scripting.StateStarted, 19)
	q.talk(hero, yumi, "32041-27.htm")
	q.event(hero, yumi, "32041-28.htm", "32041-28.htm")
	b3aWantItems(t, "detector given", hero, map[int]int64{8091: 0})
	q.talk(hero, yumi, "32041-28.htm")
	q.event(hero, yumi, "32041-31.htm", "32041-37.htm")
	q.event(hero, yumi, "32041-36.htm", "32041-36.htm")
	q.want("buy the starstone", hero, scripting.StateStarted, 20)
	q.talk(hero, yumi, "32041-36.htm")
	q.talk(hero, wendy, "32047-35.htm")
	hero.Give(b3aAdena, 2999)
	q.event(hero, wendy, "32047-38.htm", "32047-39.htm")
	hero.Give(b3aAdena, 1)
	q.event(hero, wendy, "32047-38.htm", "32047-38.htm")
	q.want("starstone", hero, scripting.StateStarted, 26)
	b3aWantItems(t, "starstone", hero, map[int]int64{b3aAdena: 0, 8289: 1})
	q.talk(hero, wendy, "32047-40.htm")
	q.talk(hero, yumi, "32041-34.htm")
	q.want("Yumi 27", hero, scripting.StateStarted, 27)
	q.talk(hero, yumi, "32041-35.htm")
	q.talk(hero, stone, "32046-03.htm")
	q.event(hero, stone, "32046-06.htm", "32046-06.htm")
	q.want("done", hero, scripting.StateCompleted, 27)
	b3aWantItems(t, "done", hero, map[int]int64{8289: 0})
	q.talk(hero, yumi, b3aCompleted)
}
