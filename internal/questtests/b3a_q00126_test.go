package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ126Step is one talk (event "") or event of the Q126 walkthrough and
// the cond it leaves.
type b3aQ126Step struct {
	npc   *scripttest.Npc
	event string
	page  string
	cond  int
}

// The Name of Evil - 2: after Q125, the Kaimu statues, the warrior's grave
// and its three songs, the Shilen statue, Asamanah and Mushika.
func TestB3aQ126TheNameOfEvil2(t *testing.T) {
	const name = "Q00126_TheNameOfEvil2"
	q := b3aStart(t, name)
	w := q.h.World()
	mushika := &scripttest.Npc{ObjectID: 0x60000001, ID: 32114, Name: "Mushika"}
	asamanah := &scripttest.Npc{ObjectID: 0x60000002, ID: 32115, Name: "Asamanah"}
	ulu := &scripttest.Npc{ObjectID: 0x60000003, ID: 32119, Name: "Ulu Kaimu"}
	balu := &scripttest.Npc{ObjectID: 0x60000004, ID: 32120, Name: "Balu Kaimu"}
	chuta := &scripttest.Npc{ObjectID: 0x60000005, ID: 32121, Name: "Chuta Kaimu"}
	grave := &scripttest.Npc{ObjectID: 0x60000006, ID: 32122, Name: "Warrior's Grave"}
	statue := &scripttest.Npc{ObjectID: 0x60000007, ID: 32109, Name: "Shilen Stone Statue"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 76}
	q.talk(young, asamanah, "32115-02.htm")
	stranger := &scripttest.Player{ObjectID: 0x10000002, Name: "Stranger", Level: 77}
	q.talk(stranger, asamanah, "32115-04.htm")
	q.want("no Q125", stranger, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 77}
	w.NewQuestState(hero.Ref(), "Q00125_TheNameOfEvil1").SetState(scripting.StateCompleted)
	q.talk(hero, asamanah, "32115-01.htm")
	steps := []b3aQ126Step{
		{asamanah, "32115-05.htm", "32115-05.htm", 1},
		{ulu, "", "32119-01a.htm", 1},
		{asamanah, "", "32115-11.htm", 2},
		{asamanah, "", "32115-12.htm", 2},
		{ulu, "", "32119-02.htm", 2},
		{ulu, "32119-02.htm", "32119-02.htm", 3},
		{ulu, "", "32119-08.htm", 3},
		{ulu, "32119-09.htm", "32119-09.htm", 4},
		{ulu, "32119-11.htm", "32119-11.htm", 5},
		{ulu, "", "32119-12.htm", 5},
		{balu, "", "32120-01.htm", 5},
		{balu, "32120-07.htm", "32120-07.htm", 6},
		{balu, "32120-09.htm", "32120-09.htm", 7},
		{balu, "32120-11.htm", "32120-11.htm", 8},
		{balu, "", "32120-12.htm", 8},
		{chuta, "", "32121-01.htm", 8},
		{chuta, "32121-07.htm", "32121-07.htm", 9},
		{chuta, "32121-10.htm", "32121-10.htm", 10},
		{chuta, "32121-15.htm", "32121-15.htm", 11},
		{grave, "", "32122-01.htm", 11},
		{grave, "32122-03.htm", "32122-03.htm", 12},
		{grave, "32122-15.htm", "32122-15.htm", 13},
		{grave, "32122-18.htm", "32122-18.htm", 14},
		// The first song: a note missing leaves it unsung.
		{grave, "DOOne", "32122-26.htm", 14},
		{grave, "MIOne", "32122-30.htm", 14},
		{grave, "FAOne", "32122-34.htm", 14},
		{grave, "FA_2One", "32122-24.htm", 14},
		{grave, "SOLOne", "32122-38.htm", 14},
		{grave, "FA_2One", "32122-42.htm", 15},
		{grave, "FATwo", "32122-47.htm", 15},
		{grave, "SOLTwo", "32122-51.htm", 15},
		{grave, "TITwo", "32122-55.htm", 15},
		{grave, "SOL_2Two", "32122-59.htm", 15},
		{grave, "FA_2Two", "32122-63.htm", 16},
		{grave, "", "32122-66.htm", 16},
		{grave, "SOLTri", "32122-68.htm", 16},
		{grave, "FATri", "32122-72.htm", 16},
		{grave, "MITri", "32122-76.htm", 16},
		{grave, "MI_2Tri", "32122-84.htm", 17}, // FA_2 stays set from the second song (Mobius)
		{grave, "32122-87.htm", "32122-87.htm", 17},
		{grave, "32122-90.htm", "32122-90.htm", 18},
		{grave, "", "32122-91.htm", 18},
		{statue, "", "32109-02.htm", 18},
		{statue, "32109-02.htm", "32109-02.htm", 19},
		{statue, "", "32109-05.htm", 19},
		{statue, "32109-19.htm", "32109-19.htm", 20},
		{asamanah, "", "32115-13.htm", 20},
		{asamanah, "32115-21.htm", "32115-21.htm", 21},
		{asamanah, "32115-28.htm", "32115-28.htm", 22},
		{mushika, "", "32114-01.htm", 22},
		{mushika, "32114-08.htm", "32114-08.htm", 23},
		{mushika, "", "32114-04.htm", 23},
	}
	for _, s := range steps {
		if s.event == "" {
			q.talk(hero, s.npc, s.page)
		} else {
			q.event(hero, s.npc, s.event, s.page)
		}
		q.want(s.npc.Name+" "+s.page, hero, scripting.StateStarted, s.cond)
		switch s.page {
		case "32122-87.htm":
			b3aWantItems(t, "bone powder", hero, map[int]int64{8783: 1})
		case "32109-19.htm":
			b3aWantItems(t, "bone powder given", hero, map[int]int64{8783: 0})
		}
	}
	q.event(hero, mushika, "32114-09.htm", "32114-09.htm")
	q.want("done", hero, scripting.StateCompleted, 23)
	b3aWantItems(t, "done", hero, map[int]int64{729: 1})
	q.talk(hero, asamanah, b3aCompleted)
}
