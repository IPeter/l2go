package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q232 = "Q00232_TestOfTheLord"

// Test of the Lord: Kakai's ordeal necklace and the five tokens (Varkees
// and Manakia, Tantus and Jakal, Hatos with Sumari and Somak, Takuna,
// Chianta), Ancestor Martankus and the Ragna orcs, the First Orc's
// Immortal Flame, the Mark of Lord.
func TestB2Q232Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q232)
	somak := &scripttest.Npc{ObjectID: 0x60000001, ID: 30510}
	manakia := &scripttest.Npc{ObjectID: 0x60000002, ID: 30515}
	jakal := &scripttest.Npc{ObjectID: 0x60000003, ID: 30558}
	sumari := &scripttest.Npc{ObjectID: 0x60000004, ID: 30564}
	kakai := &scripttest.Npc{ObjectID: 0x60000005, ID: 30565}
	varkees := &scripttest.Npc{ObjectID: 0x60000006, ID: 30566}
	tantus := &scripttest.Npc{ObjectID: 0x60000007, ID: 30567}
	hatos := &scripttest.Npc{ObjectID: 0x60000008, ID: 30568}
	takuna := &scripttest.Npc{ObjectID: 0x60000009, ID: 30641}
	chianta := &scripttest.Npc{ObjectID: 0x6000000a, ID: 30642}
	martankus := &scripttest.Npc{ObjectID: 0x6000000b, ID: 30649}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", ClassID: 11, Level: 39}
	reply, ok := h.Talk(b2Q232, human, kakai)
	b2Page(t, "human", reply, ok, "30565-01.htm")
	raider := &scripttest.Player{ObjectID: 0x10000002, Name: "Raider", ClassID: 45, Race: 3, Level: 39}
	reply, ok = h.Talk(b2Q232, raider, kakai)
	b2Page(t, "raider", reply, ok, "30565-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000003, Name: "Young", ClassID: 50, Race: 3, Level: 38}
	reply, ok = h.Talk(b2Q232, young, kakai)
	b2Page(t, "level 38", reply, ok, "30565-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 50, Race: 3, Level: 39}
	reply, ok = h.Talk(b2Q232, hero, kakai)
	b2Page(t, "offer", reply, ok, "30565-04.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q232, "30565-05.htm", hero, kakai)
	b2Page(t, "accept", reply, ok, "30565-05b.htm")
	b2Quest(t, "accept", hero, b2Q232, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{3391: 1, 7562: 92})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q232, hero, kakai)
	b2Page(t, "Kakai cond 1", reply, ok, "30565-06.htm")

	// Varkees and Manakia: the huge orc fang.
	reply, ok = h.Talk(b2Q232, hero, varkees)
	b2Page(t, "Varkees", reply, ok, "30566-01.htm")
	h.Event(b2Q232, "30566-02.htm", hero, varkees)
	reply, ok = h.Talk(b2Q232, hero, manakia)
	b2Page(t, "Manakia", reply, ok, "30515-01.htm")
	b2Items(t, "orders", hero, map[int]int64{3392: 1, 3397: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 20269}, 1, rnd.Get(100, 41))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 20269}, 10, rnd.Get(100, 40))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 20270}, 10, rnd.Get(100, 50))
	b2Items(t, "Breka fangs", hero, map[int]int64{3398: 20})
	reply, ok = h.Talk(b2Q232, hero, manakia)
	b2Page(t, "Manakia fangs", reply, ok, "30515-03.htm")
	b2Items(t, "amulet", hero, map[int]int64{3397: 0, 3398: 0, 3399: 1})
	reply, ok = h.Talk(b2Q232, hero, varkees)
	b2Page(t, "Varkees amulet", reply, ok, "30566-04.htm")
	b2Items(t, "huge fang", hero, map[int]int64{3392: 0, 3399: 0, 3400: 1})

	// Tantus and Jakal: the Axe of Ceremony.
	reply, ok = h.Talk(b2Q232, hero, tantus)
	b2Page(t, "Tantus", reply, ok, "30567-01.htm")
	h.Event(b2Q232, "30567-02.htm", hero, tantus)
	reply, ok = h.Talk(b2Q232, hero, jakal)
	b2Page(t, "Jakal poor", reply, ok, "30558-03.htm")
	hero.Give(57, 1500)
	reply, ok = h.Talk(b2Q232, hero, jakal)
	b2Page(t, "Jakal", reply, ok, "30558-01.htm")
	h.Event(b2Q232, "30558-02.htm", hero, jakal)
	b2Items(t, "axe blade", hero, map[int]int64{57: 500, 3405: 1})
	reply, ok = h.Talk(b2Q232, hero, tantus)
	b2Page(t, "Tantus no arrows", reply, ok, "30567-03.htm")
	hero.Give(1341, 1200)
	reply, ok = h.Talk(b2Q232, hero, tantus)
	b2Page(t, "Tantus arrows", reply, ok, "30567-04.htm")
	b2Items(t, "axe", hero, map[int]int64{1341: 200, 3405: 0, 3393: 0, 3406: 1})

	// Hatos, Sumari and Somak: the Sword into Skull.
	reply, ok = h.Talk(b2Q232, hero, hatos)
	b2Page(t, "Hatos", reply, ok, "30568-01.htm")
	h.Event(b2Q232, "30568-02.htm", hero, hatos)
	reply, ok = h.Talk(b2Q232, hero, sumari)
	b2Page(t, "Sumari", reply, ok, "30564-01.htm")
	reply, ok = h.Talk(b2Q232, hero, somak)
	b2Page(t, "Somak", reply, ok, "30510-01.htm")
	b2Items(t, "Urutu blade", hero, map[int]int64{3401: 0, 3402: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20583}, 1, rnd.Get(100, 71))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20583}, 5, rnd.Get(100, 70))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20586}, 4, rnd.Get(100, 80))
	reply, ok = h.Talk(b2Q232, hero, hatos)
	b2Page(t, "Hatos 9 skulls", reply, ok, "30568-03.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000014, ID: 20588}, 2)
	b2Items(t, "skulls", hero, map[int]int64{3403: 10})
	reply, ok = h.Talk(b2Q232, hero, hatos)
	b2Page(t, "Hatos skulls", reply, ok, "30568-04.htm")
	b2Items(t, "sword", hero, map[int]int64{3394: 0, 3402: 0, 3403: 0, 3404: 1})

	// Takuna: the spider brooch.
	reply, ok = h.Talk(b2Q232, hero, takuna)
	b2Page(t, "Takuna", reply, ok, "30641-01.htm")
	h.Event(b2Q232, "30641-02.htm", hero, takuna)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000015, ID: 20233}, 20)
	b2Items(t, "spider parts", hero, map[int]int64{3407: 10, 3408: 10})
	reply, ok = h.Talk(b2Q232, hero, takuna)
	b2Page(t, "Takuna parts", reply, ok, "30641-04.htm")
	b2Items(t, "brooch", hero, map[int]int64{3407: 0, 3408: 0, 3395: 0, 3409: 1})
	b2Quest(t, "four tokens", hero, b2Q232, scripting.StateStarted, 1)

	// Chianta: the monster eye woodcarving.
	reply, ok = h.Talk(b2Q232, hero, chianta)
	b2Page(t, "Chianta", reply, ok, "30642-01.htm")
	h.Event(b2Q232, "30642-02.htm", hero, chianta)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000016, ID: 20564}, 21)
	b2Items(t, "corneas", hero, map[int]int64{3410: 20})
	reply, ok = h.Talk(b2Q232, hero, chianta)
	b2Page(t, "Chianta corneas", reply, ok, "30642-04.htm")
	b2Quest(t, "five tokens", hero, b2Q232, scripting.StateStarted, 2)
	b2Items(t, "five tokens", hero, map[int]int64{3410: 0, 3396: 0, 3411: 1})

	// Kakai and Ancestor Martankus.
	reply, ok = h.Talk(b2Q232, hero, kakai)
	b2Page(t, "Kakai tokens", reply, ok, "30565-07.htm")
	h.Event(b2Q232, "30565-08.htm", hero, kakai)
	b2Quest(t, "bear fang", hero, b2Q232, scripting.StateStarted, 3)
	b2Items(t, "bear fang", hero, map[int]int64{3404: 0, 3406: 0, 3411: 0, 3409: 0, 3391: 0, 3400: 0, 3412: 1})
	reply, ok = h.Talk(b2Q232, hero, martankus)
	b2Page(t, "Martankus", reply, ok, "30649-01.htm")
	h.Event(b2Q232, "30649-04.htm", hero, martankus)
	b2Quest(t, "Ragna", hero, b2Q232, scripting.StateStarted, 4)
	b2Items(t, "Ragna", hero, map[int]int64{3412: 0, 3413: 1})
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000017, ID: 20778})
	b2Items(t, "notice", hero, map[int]int64{3415: 1})
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000018, ID: 20779})
	b2Quest(t, "head", hero, b2Q232, scripting.StateStarted, 5)
	b2Items(t, "head", hero, map[int]int64{3414: 1})
	reply, ok = h.Talk(b2Q232, hero, martankus)
	b2Page(t, "Martankus head", reply, ok, "30649-06.htm")
	b2Quest(t, "flame", hero, b2Q232, scripting.StateStarted, 6)
	b2Items(t, "flame", hero, map[int]int64{3413: 0, 3414: 0, 3415: 0, 3416: 1})

	// The First Orc.
	reply, ok = h.Talk(b2Q232, hero, martankus)
	b2Page(t, "Martankus flame", reply, ok, "30649-07.htm")
	h.World().ClearRecords()
	h.Event(b2Q232, "30649-07.htm", hero, martankus)
	h.Event(b2Q232, "30649-07.htm", hero, martankus)
	firstOrc := b2Q227Spawn(t, h, "First Orc", 30643)
	reply, ok = h.Talk(b2Q232, hero, firstOrc)
	b2Page(t, "First Orc", reply, ok, "30643-01.htm")
	h.Event(b2Q232, "30643-02.htm", hero, firstOrc)
	b2Quest(t, "First Orc", hero, b2Q232, scripting.StateStarted, 7)
	h.Advance(10 * time.Second)
	if !firstOrc.Deleted {
		t.Error("the First Orc stayed")
	}

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q232, hero, kakai)
	b2Page(t, "mark", reply, ok, "30565-11.htm")
	b2Quest(t, "mark", hero, b2Q232, scripting.StateCompleted, 7)
	b2Items(t, "mark", hero, map[int]int64{3416: 0, 3390: 1, 7562: 92})
	b2ExpSp(t, "mark", hero, 92955, 16250)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q232, hero, kakai)
	b2Completed(t, "completed", reply, ok)
}
