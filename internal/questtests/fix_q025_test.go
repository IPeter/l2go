package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// fixQ025 is Hiding Behind the Truth.
const fixQ025 = "Q00025_HidingBehindTheTruth"

// Hiding Behind the Truth: Benedict, the wizard's totem, Agripel's key, the
// bookshelf and Triol (who runs at and attacks the player), Lidia, the
// coffin and the dress, then Lidia's ending.
func TestFixQ025HidingBehindTheTruth(t *testing.T) {
	q := b3aStart(t, fixQ025)
	w := q.h.World()
	agripel := &scripttest.Npc{ObjectID: 0x60000001, ID: 31348, Name: "Agripel"}
	benedict := &scripttest.Npc{ObjectID: 0x60000002, ID: 31349, Name: "Benedict"}
	wizard := &scripttest.Npc{ObjectID: 0x60000003, ID: 31522, Name: "Mysterious Wizard"}
	tombstone := &scripttest.Npc{ObjectID: 0x60000004, ID: 31531, Name: "Tombstone"}
	lidia := &scripttest.Npc{ObjectID: 0x60000005, ID: 31532, Name: "Lidia"}
	shelf1 := &scripttest.Npc{ObjectID: 0x60000006, ID: 31533}
	shelf2 := &scripttest.Npc{ObjectID: 0x60000007, ID: 31534}
	shelf3 := &scripttest.Npc{ObjectID: 0x60000008, ID: 31535}

	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 70}
	q.talk(stranger, benedict, b3aNoQuest)
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 65}
	w.NewQuestState(young.Ref(), "Q00024_InhabitantsOfTheForestOfTheDead").SetState(scripting.StateCompleted)
	q.talk(young, benedict, "31349-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 66}
	w.NewQuestState(hero.Ref(), "Q00024_InhabitantsOfTheForestOfTheDead").SetState(scripting.StateCompleted)
	hero.Give(7063, 1)
	q.talk(hero, benedict, "31349-01.htm")
	q.event(hero, benedict, "31349-02.htm", "31349-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, benedict, "31349-02.htm")
	q.event(hero, benedict, "31349-03.htm", "31349-03.htm")
	q.want("to the wizard", hero, scripting.StateStarted, 2)
	q.talk(hero, benedict, "31349-04.htm")

	q.talk(hero, wizard, "31522-01.htm")
	q.want("totem", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "totem", hero, map[int]int64{7156: 1})
	q.talk(hero, wizard, "31522-02.htm")
	q.event(hero, benedict, "31349-03.htm", "31349-05.htm")
	q.event(hero, benedict, "31349-10.htm", "31349-10.htm")
	q.want("to Agripel", hero, scripting.StateStarted, 4)
	q.talk(hero, benedict, "31349-10.htm")

	q.talk(hero, agripel, "31348-01.htm")
	q.event(hero, agripel, "31348-02.htm", "31348-02.htm")
	q.event(hero, agripel, "31348-07.htm", "31348-07.htm")
	q.want("key", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "key", hero, map[int]int64{7156: 0, 7157: 1})
	q.talk(hero, agripel, "31348-08.htm")
	q.talk(hero, wizard, "31522-03.htm")
	q.event(hero, wizard, "31522-04.htm", "31522-04.htm")
	q.want("bookshelves", hero, scripting.StateStarted, 6)
	q.talk(hero, wizard, "31522-04.htm")

	// Triol guards the box: it runs at the player, hates and attacks.
	q.talk(hero, shelf1, "31533-01.htm")
	q.talk(hero, shelf2, "31534-01.htm")
	q.talk(hero, shelf3, "31535-01.htm")
	w.ClearRecords()
	q.event(hero, shelf3, "31535-03.htm", "31535-03.htm")
	q.want("Triol", hero, scripting.StateStarted, 7)
	triols := fixSpawnedOf(w, 27218)
	if len(triols) != 1 {
		t.Fatalf("spawned %+v, want Triol", w.Spawned)
	}
	triol := triols[0]
	if triol.Heading != -1 || !triol.Running {
		t.Errorf("Triol %+v, want a random heading and running", triol)
	}
	if !fixSays(w, triol.ObjectID, "That box was sealed by my master. Don't touch it!") {
		t.Errorf("Triol actions %+v", w.Actions)
	}
	fixWantAction(t, "Triol hate", w, "hate", triol.ObjectID, hero.ObjectID, 999)
	fixWantAction(t, "Triol attack", w, "attack", triol.ObjectID, hero.ObjectID, 0)
	q.event(hero, shelf3, "31535-03.htm", "31535-03.htm")
	if len(w.Spawned) != 1 {
		t.Errorf("Triol spawned twice")
	}
	w.ClearRecords()
	q.kill(hero, triol)
	q.want("Triol killed", hero, scripting.StateStarted, 8)
	b3aWantItems(t, "Triol killed", hero, map[int]int64{7158: 1})
	if !fixSays(w, triol.ObjectID, "You've ended my immortal life! You've protected by the feudal lord, aren't you?") {
		t.Errorf("Triol actions %+v", w.Actions)
	}
	q.event(hero, shelf3, "31535-03.htm", "31535-04.htm")
	q.event(hero, shelf3, "31535-05.htm", "31535-05.htm")
	q.want("contract", hero, scripting.StateStarted, 9)
	b3aWantItems(t, "contract", hero, map[int]int64{7157: 0, 7066: 1})
	q.talk(hero, shelf3, "31535-06.htm")

	q.talk(hero, wizard, "31522-05.htm")
	q.want("to Lidia", hero, scripting.StateStarted, 10)
	q.talk(hero, wizard, "31522-05.htm")
	q.talk(hero, lidia, "31532-01.htm")
	q.event(hero, lidia, "31532-02.htm", "31532-02.htm")
	b3aWantItems(t, "contract given", hero, map[int]int64{7066: 0})
	q.event(hero, lidia, "31532-06.htm", "31532-06.htm")
	q.want("tombstone", hero, scripting.StateStarted, 11)
	q.talk(hero, lidia, "31532-06.htm")

	// The coffin with the dress.
	q.talk(hero, tombstone, "31531-01.htm")
	q.event(hero, tombstone, "31531-02.htm", "31531-02.htm")
	q.want("coffin", hero, scripting.StateStarted, 12)
	coffins := fixSpawnedOf(w, 31536)
	if len(coffins) != 1 {
		t.Fatalf("spawned %+v, want the coffin", w.Spawned)
	}
	q.talk(hero, coffins[0], "31536-01.htm")
	q.want("dress", hero, scripting.StateStarted, 13)
	b3aWantItems(t, "dress", hero, map[int]int64{7155: 1})
	if !coffins[0].Deleted {
		t.Error("the coffin stays")
	}
	q.talk(hero, tombstone, "31531-03.htm")
	q.talk(hero, lidia, "31532-07.htm")
	q.want("dress given", hero, scripting.StateStarted, 14)
	b3aWantItems(t, "dress given", hero, map[int]int64{7155: 0})
	q.talk(hero, lidia, "31532-08.htm")
	q.event(hero, lidia, "31532-18.htm", "31532-18.htm")
	q.want("to the wizard again", hero, scripting.StateStarted, 15)
	q.talk(hero, lidia, "31532-18.htm")

	q.talk(hero, wizard, "31522-06.htm")
	q.event(hero, wizard, "31522-12.htm", "31522-12.htm")
	q.want("to Agripel again", hero, scripting.StateStarted, 16)
	q.talk(hero, wizard, "31522-13.htm")
	q.talk(hero, agripel, "31348-09.htm")
	q.event(hero, agripel, "31348-10.htm", "31348-10.htm")
	b3aWantItems(t, "doll given", hero, map[int]int64{7158: 0})
	q.event(hero, agripel, "31348-15.htm", "31348-15.htm")
	q.want("Lidia's ending", hero, scripting.StateStarted, 17)
	q.talk(hero, agripel, "31348-17.htm")
	q.talk(hero, wizard, "31522-16.htm")

	q.talk(hero, lidia, "31532-19.htm")
	q.event(hero, lidia, "31532-20.htm", "31532-20.htm")
	b3aWantItems(t, "Lidia's reward", hero, map[int]int64{905: 2, 874: 1, 7063: 0})
	b3aWantExp(t, "Lidia's reward", hero, 572277, 53750)
	if st := hero.Quest(fixQ025); st != nil {
		t.Errorf("exit_quest(true) keeps the state: %+v", st.Vars())
	}
}

// The wizard's ending: Agripel sends the player back to the wizard.
func TestFixQ025WizardEnding(t *testing.T) {
	q := b3aStart(t, fixQ025)
	w := q.h.World()
	agripel := &scripttest.Npc{ObjectID: 0x60000001, ID: 31348}
	benedict := &scripttest.Npc{ObjectID: 0x60000002, ID: 31349}
	wizard := &scripttest.Npc{ObjectID: 0x60000003, ID: 31522}
	lidia := &scripttest.Npc{ObjectID: 0x60000005, ID: 31532}
	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 66}
	w.NewQuestState(hero.Ref(), "Q00024_InhabitantsOfTheForestOfTheDead").SetState(scripting.StateCompleted)
	q.talk(hero, benedict, "31349-01.htm")
	q.event(hero, benedict, "31349-02.htm", "31349-02.htm")
	hero.Quest(fixQ025).Set("cond", "16")
	q.event(hero, agripel, "31348-16.htm", "31348-16.htm")
	q.want("wizard's ending", hero, scripting.StateStarted, 18)
	q.talk(hero, agripel, "31348-18.htm")
	q.talk(hero, lidia, "31532-21.htm")
	q.talk(hero, wizard, "31522-14.htm")
	q.event(hero, wizard, "31522-15.htm", "31522-15.htm")
	b3aWantItems(t, "wizard's reward", hero, map[int]int64{936: 1, 874: 1})
	b3aWantExp(t, "wizard's reward", hero, 572277, 53750)
	if st := hero.Quest(fixQ025); st != nil {
		t.Errorf("exit_quest(true) keeps the state: %+v", st.Vars())
	}
}
