package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ504 = "Q00504_CompetitionForTheBanditStronghold"

// Competition for the Bandit Stronghold waits for clan halls: it loads
// with requires("clan_halls"); taking it needs the siegable hall. A state
// taken earlier (as after an import) runs: 30 Tarlk amulets for the
// Trophy of Alliance.
func TestB3cQ504Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ504)
	w := h.World()
	b3cRequires(t, h, b3cQ504, "clan_halls")
	messenger := &scripttest.Npc{ObjectID: 0x60000001, ID: 35437, Name: "Messenger"}
	tarlk := &scripttest.Npc{ObjectID: 0x60000002, ID: 20572, Name: "Tarlk Bugbear"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	st, _ := w.NewQuestState(hero.Ref(), b3cQ504).(*scripttest.QuestState)
	st.SetState(scripting.StateStarted)
	st.Set("cond", "1")

	reply, ok := h.Talk(b3cQ504, hero, messenger)
	b3cPage(t, "no amulets", reply, ok, "35437-02.htm")

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, tarlk)
	b3cItems(t, "amulet", hero, map[int]int64{4332: 1})
	hero.Give(4332, 28)
	h.Kill(hero, tarlk)
	b3cQuest(t, "30 amulets", hero, b3cQ504, scripting.StateStarted, 2)
	b3cSounds(t, "amulets", w, hero, b3cSoundMiddle, b3cSoundMiddle)
	h.Kill(hero, tarlk) // cond 2: nothing more
	b3cItems(t, "cond 2", hero, map[int]int64{4332: 30})

	reply, ok = h.Talk(b3cQ504, hero, messenger)
	b3cPage(t, "trophy", reply, ok, "35437-03.htm")
	b3cItems(t, "trophy", hero, map[int]int64{4332: 0, 5009: 1})
	b3cNoQuest(t, "trophy", hero, b3cQ504)
}
