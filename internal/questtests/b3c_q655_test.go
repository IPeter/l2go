package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ655 = "Q00655_AGrandPlanForTamingWildBeasts"

// A Grand Plan for Taming Wild Beasts needs clan halls: it loads and is
// marked so. The start reads the clan hall siege, which does not exist
// yet; a started quest (e.g. imported) hands in ten Crystals of Purity for
// the Trainer License.
func TestB3cQ655Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ655)
	b3cRequires(t, h, b3cQ655, "clan_halls")
	messenger := &scripttest.Npc{ObjectID: 0x60000001, ID: 35627, Name: "Messenger"}

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	st := h.World().NewQuestState(hero.Ref(), b3cQ655)
	st.SetState(scripting.StateStarted)
	st.Set("cond", "1")

	reply, ok := h.Talk(b3cQ655, hero, messenger)
	b3cPage(t, "few stones", reply, ok, "35627-02.htm")
	hero.Give(8084, 10)
	reply, ok = h.Talk(b3cQ655, hero, messenger)
	b3cPage(t, "license", reply, ok, "35627-03.htm")
	b3cItems(t, "license", hero, map[int]int64{8084: 0, 8293: 1})
	b3cNoQuest(t, "license", hero, b3cQ655)
}
