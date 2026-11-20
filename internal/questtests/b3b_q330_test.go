package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ330 = "Q00330_AdeptOfTaste"

// Adept of Taste: the five ingredients (the special ones), Jonas cooks the
// best dish, Mirien reviews it, Jonas pays.
func TestB3bQ330Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ330)
	w := h.World()
	npc := func(oid int32, id int) *scripttest.Npc { return &scripttest.Npc{ObjectID: oid, ID: id} }
	jonas, sonia, glyvka, rollant := npc(0x60000001, 30469), npc(0x60000002, 30062), npc(0x60000003, 30067), npc(0x60000004, 30069)
	jacob, pano, mirien := npc(0x60000005, 30073), npc(0x60000006, 30078), npc(0x60000007, 30461)

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 23}
	reply, ok := h.Talk(b3bQ330, young, jonas)
	b3bPage(t, "level 23", reply, ok, "30469-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 24}
	reply, ok = h.Talk(b3bQ330, hero, jonas)
	b3bPage(t, "offer", reply, ok, "30469-02.htm")
	h.Event(b3bQ330, "30469-03.htm", hero, jonas)
	b3bQuest(t, "accept", hero, b3bQ330, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{1420: 1})
	reply, ok = h.Talk(b3bQ330, hero, jonas)
	b3bPage(t, "no ingredients", reply, ok, "30469-04.htm")

	// Sonia: mandragora roots from 20223 (red below 70, white below 77).
	reply, ok = h.Talk(b3bQ330, hero, sonia)
	b3bPage(t, "Sonia book", reply, ok, "30062-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 69), rnd.Get(100, 76), rnd.Get(100, 77)))
	for range 3 {
		h.Kill(hero, npc(0x60000010, 20223))
	}
	b3bItems(t, "roots", hero, map[int]int64{1422: 1, 1423: 1})
	reply, ok = h.Talk(b3bQ330, hero, sonia)
	b3bPage(t, "Sonia few roots", reply, ok, "30062-02.htm")
	hero.Give(1422, 39)
	hero.Give(1423, 39)
	reply, ok = h.Talk(b3bQ330, hero, sonia)
	b3bPage(t, "Sonia white sap", reply, ok, "30062-06.htm")
	b3bItems(t, "white sap", hero, map[int]int64{1421: 0, 1422: 0, 1423: 0, 1425: 1})

	// Jacob: nectar below 92, royal jelly below 100 from 20204.
	reply, ok = h.Talk(b3bQ330, hero, jacob)
	b3bPage(t, "Jacob book", reply, ok, "30073-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 91), rnd.Get(100, 92)))
	h.Kill(hero, npc(0x60000011, 20204))
	h.Kill(hero, npc(0x60000011, 20204))
	b3bItems(t, "honey drops", hero, map[int]int64{1427: 1, 1428: 1})
	hero.Give(1427, 19)
	reply, ok = h.Talk(b3bQ330, hero, jacob)
	b3bPage(t, "Jacob jelly", reply, ok, "30073-03.htm")
	hero.Give(1428, 9)
	reply, ok = h.Talk(b3bQ330, hero, jacob)
	b3bPage(t, "Jacob golden honey", reply, ok, "30073-06.htm")
	b3bItems(t, "golden honey", hero, map[int]int64{1426: 0, 1427: 0, 1428: 0, 1430: 1})

	// Pano: hobgoblin amulets from 20147, no draw.
	reply, ok = h.Talk(b3bQ330, hero, pano)
	b3bPage(t, "Pano contract", reply, ok, "30078-01.htm")
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, npc(0x60000012, 20147))
	hero.Give(1432, 29)
	reply, ok = h.Talk(b3bQ330, hero, pano)
	b3bPage(t, "Pano potato", reply, ok, "30078-03.htm")
	b3bItems(t, "potato", hero, map[int]int64{1431: 0, 1432: 0, 1433: 1})

	// Glyvka: marsh moss from 20228 (green below 9 of 10).
	reply, ok = h.Talk(b3bQ330, hero, glyvka)
	b3bPage(t, "Glyvka book", reply, ok, "30067-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 8), rnd.Get(10, 9)))
	h.Kill(hero, npc(0x60000013, 20228))
	h.Kill(hero, npc(0x60000013, 20228))
	b3bItems(t, "moss", hero, map[int]int64{1435: 1, 1436: 1})
	hero.Give(1435, 19)
	hero.Give(1436, 19)
	reply, ok = h.Talk(b3bQ330, hero, glyvka)
	b3bPage(t, "Glyvka brown bundle", reply, ok, "30067-06.htm")
	b3bItems(t, "brown bundle", hero, map[int]int64{1434: 0, 1435: 0, 1436: 0, 1438: 1})

	// Rollant: monster eye bodies from 20265 (2 below 77 of 97, else 3).
	reply, ok = h.Talk(b3bQ330, hero, rollant)
	b3bPage(t, "Rollant book", reply, ok, "30069-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 96), rnd.Get(97, 77), rnd.Get(100, 97)))
	h.Kill(hero, npc(0x60000014, 20265))
	h.Kill(hero, npc(0x60000014, 20265))
	b3bItems(t, "eye bodies", hero, map[int]int64{1440: 3})
	hero.Give(1440, 27)
	reply, ok = h.Talk(b3bQ330, hero, rollant)
	b3bPage(t, "Rollant meat", reply, ok, "30069-03.htm")
	b3bItems(t, "meat", hero, map[int]int64{1439: 0, 1440: 0, 1441: 1})

	// Three special ingredients and a lucky draw: the fifth dish.
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true)))
	w.ClearRecords()
	reply, ok = h.Talk(b3bQ330, hero, jonas)
	b3bPage(t, "dish", reply, ok, "30469-05t5.htm")
	b3bItems(t, "dish", hero, map[int]int64{1420: 0, 1425: 0, 1430: 0, 1433: 0, 1438: 0, 1441: 0, 1446: 1})
	if got := w.SoundsOf(hero); !slices.Equal(got, []string{"ItemSound.quest_jackpot"}) {
		t.Errorf("dish sounds = %v", got)
	}
	reply, ok = h.Talk(b3bQ330, hero, jonas)
	b3bPage(t, "Jonas waits", reply, ok, "30469-06.htm")

	reply, ok = h.Talk(b3bQ330, hero, mirien)
	b3bPage(t, "review", reply, ok, "30461-02t5.htm")
	b3bItems(t, "review", hero, map[int]int64{1446: 0, 1451: 1})
	reply, ok = h.Talk(b3bQ330, hero, mirien)
	b3bPage(t, "Mirien again", reply, ok, "30461-04.htm")

	reply, ok = h.Talk(b3bQ330, hero, jonas)
	b3bPage(t, "reward", reply, ok, "30469-06t5.htm")
	b3bItems(t, "reward", hero, map[int]int64{1451: 0, 57: 7800, 1457: 1})
	b3bExpSp(t, "reward", hero, 12000, 0)
	b3bNoQuest(t, "reward", hero, b3bQ330)
}
