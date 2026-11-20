package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// fixQ022Done gives p a completed state of quest.
func fixQ022Done(w *scripttest.World, p *scripttest.Player, quest string) {
	w.Add(p)
	w.NewQuestState(p.Ref(), quest).SetState(scripting.StateCompleted)
}

// Tragedy in von Hellmann Forest: Tifaren, Innocentin's cross, the lost
// skull, the priest's ghost, the adventurer's ghost, the Soul of the Well
// (it attacks the player; after 90 s a hit gives the second jewel), the
// report box and Innocentin's reward.
func TestFixQ022TragedyInVonHellmannForest(t *testing.T) {
	q := b3aStart(t, "Q00022_TragedyInVonHellmannForest")
	w := q.h.World()
	tifaren := &scripttest.Npc{ObjectID: 0x60000001, ID: 31334, Name: "Tifaren"}
	innocentin := &scripttest.Npc{ObjectID: 0x60000002, ID: 31328, Name: "Innocentin"}
	well := &scripttest.Npc{ObjectID: 0x60000003, ID: 31527, Name: "Well"}
	adventurer := &scripttest.Npc{ObjectID: 0x60000004, ID: 31529, Name: "Ghost of an Adventurer"}
	mob := &scripttest.Npc{ObjectID: 0x60000005, ID: 21553}

	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 70}
	q.talk(stranger, tifaren, "31334-01.htm")
	q.event(stranger, tifaren, "31334-02.htm", "31334-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 63}
	fixQ022Done(w, hero, "Q00021_HiddenTruth")
	q.talk(hero, tifaren, "31334-01.htm")
	q.event(hero, tifaren, "31334-02.htm", "31334-02.htm")
	q.event(hero, tifaren, "31334-04.htm", "31334-04.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, tifaren, "31334-05.htm")
	q.event(hero, tifaren, "31334-07.htm", "31334-07.htm")
	q.want("no cross", hero, scripting.StateStarted, 2)

	// Innocentin gives a new cross.
	q.talk(hero, innocentin, "31328-01.htm")
	q.want("cross", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "cross", hero, map[int]int64{7141: 1})
	q.talk(hero, innocentin, "31328-01b.htm")
	q.talk(hero, tifaren, "31334-05.htm")
	q.event(hero, tifaren, "31334-08.htm", "31334-08.htm")
	q.want("skull hunt", hero, scripting.StateStarted, 4)
	q.talk(hero, tifaren, "31334-09.htm")

	q.kill(hero, mob, rnd.Get(100, 10))
	q.want("no skull", hero, scripting.StateStarted, 4)
	q.kill(hero, mob, rnd.Get(100, 9))
	q.want("skull", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "skull", hero, map[int]int64{7142: 1})

	// The priest's ghost, called for the hero only.
	q.talk(hero, tifaren, "31334-10.htm")
	w.ClearRecords()
	q.event(hero, tifaren, "31334-13.htm", "31334-13.htm")
	q.want("ghost", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "ghost", hero, map[int]int64{7142: 0})
	priests := fixSpawnedOf(w, 31528)
	if len(priests) != 1 {
		t.Fatalf("spawned %+v, want the priest's ghost", w.Spawned)
	}
	priest := priests[0]
	if !fixSays(w, priest.ObjectID, "Did you call me, Hero?") {
		t.Errorf("ghost actions %+v", w.Actions)
	}
	q.talk(hero, tifaren, "31334-15.htm")
	q.talk(stranger, priest, "31528-03.htm")
	q.talk(hero, priest, "31528-01.htm")
	q.event(hero, priest, "31528-04.htm", "31528-04.htm")
	q.event(hero, priest, "31528-08.htm", "31528-08.htm")
	q.want("ghost told", hero, scripting.StateStarted, 8)
	w.ClearRecords()
	q.h.Advance(3 * time.Second)
	if !priest.Deleted || len(fixActions(w, "say")) != 0 {
		t.Errorf("ghost leaves silently: deleted %v, actions %+v", priest.Deleted, w.Actions)
	}
	q.talk(hero, tifaren, "31334-18.htm")

	// Innocentin takes the cross and writes a letter.
	q.talk(hero, innocentin, "31328-02.htm")
	q.event(hero, innocentin, "31328-03.htm", "31328-03.htm")
	b3aWantItems(t, "cross taken", hero, map[int]int64{7141: 0})
	q.talk(hero, innocentin, "31328-04.htm")
	q.event(hero, innocentin, "31328-09.htm", "31328-09.htm")
	q.want("letter", hero, scripting.StateStarted, 9)
	b3aWantItems(t, "letter", hero, map[int]int64{7143: 1})
	q.talk(hero, innocentin, "31328-09a.htm")

	// The adventurer's ghost.
	q.talk(hero, adventurer, "31529-01.htm")
	q.event(hero, adventurer, "31529-03.htm", "31529-03.htm")
	q.talk(hero, adventurer, "31529-03a.htm")
	q.event(hero, adventurer, "31529-08.htm", "31529-08.htm")
	q.talk(hero, adventurer, "31529-10.htm")
	q.event(hero, adventurer, "31529-11.htm", "31529-11.htm")
	q.want("jewel", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "jewel", hero, map[int]int64{7144: 1})
	q.talk(hero, adventurer, "31529-12.htm")

	// The Soul of the Well attacks; a hit after 90 s gives the second jewel.
	q.talk(hero, well, "31527-01.htm")
	w.ClearRecords()
	q.event(hero, well, "31527-02.htm", "31527-02.htm")
	souls := fixSpawnedOf(w, 27217)
	if len(souls) != 1 {
		t.Fatalf("spawned %+v, want the soul", w.Spawned)
	}
	soul := souls[0]
	fixWantAction(t, "soul attacks", w, "attack", soul.ObjectID, hero.ObjectID, 0)
	q.event(hero, well, "31527-02.htm", "31527-03.htm")
	q.h.Attack(hero, soul, 10)
	q.talk(hero, adventurer, "31529-14.htm")
	q.h.Attack(hero, soul, 10)
	q.want("too early", hero, scripting.StateStarted, 10)
	q.h.Advance(90 * time.Second)
	q.h.Attack(hero, soul, 10)
	q.want("second jewel", hero, scripting.StateStarted, 11)
	b3aWantItems(t, "second jewel", hero, map[int]int64{7144: 0, 7145: 1})
	q.kill(hero, soul)

	q.talk(hero, adventurer, "31529-15.htm")
	q.want("to the well", hero, scripting.StateStarted, 12)
	q.talk(hero, well, "31527-04.htm")
	q.want("sealed box", hero, scripting.StateStarted, 13)
	b3aWantItems(t, "sealed box", hero, map[int]int64{7146: 1})
	q.talk(hero, well, "31527-05.htm")
	q.talk(hero, adventurer, "31529-16.htm")
	q.want("report box", hero, scripting.StateStarted, 14)
	b3aWantItems(t, "report box", hero, map[int]int64{7146: 0, 7145: 0, 7147: 1})
	q.talk(hero, adventurer, "31529-17.htm")

	q.talk(hero, innocentin, "31328-10.htm")
	q.event(hero, innocentin, "31328-11.htm", "31328-11.htm")
	q.want("report given", hero, scripting.StateStarted, 15)
	b3aWantItems(t, "report given", hero, map[int]int64{7147: 0})
	q.talk(hero, innocentin, "31328-12.htm")
	q.event(hero, innocentin, "31328-19.htm", "31328-19.htm")
	q.want("story", hero, scripting.StateStarted, 16)
	q.talk(hero, innocentin, "31328-20.htm")
	q.want("done", hero, scripting.StateCompleted, 16)
	b3aWantExp(t, "done", hero, 345966, 31578)
	q.talk(hero, tifaren, b3aCompleted)
}

// The priest's ghost leaves after two minutes, confused, and the soul of
// the well after two.
func TestFixQ022Despawns(t *testing.T) {
	q := b3aStart(t, "Q00022_TragedyInVonHellmannForest")
	w := q.h.World()
	tifaren := &scripttest.Npc{ObjectID: 0x60000001, ID: 31334}
	well := &scripttest.Npc{ObjectID: 0x60000003, ID: 31527}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 70}
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 70}
	for _, p := range []*scripttest.Player{hero, other} {
		fixQ022Done(w, p, "Q00021_HiddenTruth")
		q.talk(p, tifaren, "31334-01.htm")
		q.event(p, tifaren, "31334-04.htm", "31334-04.htm")
		p.Give(7141, 1)
		p.Quest("Q00022_TragedyInVonHellmannForest").Set("cond", "5")
	}
	q.event(hero, tifaren, "31334-13.htm", "31334-13.htm")
	// One ghost at a time: the other player waits.
	q.event(other, tifaren, "31334-13.htm", "31334-14.htm")
	q.talk(other, tifaren, "31334-16.htm")
	q.want("other waits", other, scripting.StateStarted, 6)
	priest := fixSpawnedOf(w, 31528)[0]
	w.ClearRecords()
	q.h.Advance(120 * time.Second)
	if !priest.Deleted || !fixSays(w, priest.ObjectID, "I'm confused! Maybe it's time to go back.") {
		t.Errorf("ghost: deleted %v, actions %+v", priest.Deleted, w.Actions)
	}
	q.talk(other, tifaren, "31334-17.htm")

	// The soul of the well leaves after two minutes; the well calls it again.
	hero.Quest("Q00022_TragedyInVonHellmannForest").Set("cond", "10")
	q.event(hero, well, "31527-02.htm", "31527-02.htm")
	soul := fixSpawnedOf(w, 27217)[0]
	q.h.Advance(120 * time.Second)
	if !soul.Deleted {
		t.Error("the soul stays after two minutes")
	}
	q.event(hero, well, "31527-02.htm", "31527-02.htm")
	if n := len(fixSpawnedOf(w, 27217)); n != 2 {
		t.Errorf("%d souls, want 2", n)
	}
}
