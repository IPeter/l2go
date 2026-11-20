package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q221 = "Q00221_TestimonyOfProsperity"

// Testimony of Prosperity: Lockirin's five contributions (the old account
// book), Lilith's elven wafer, Piotur's blessed seed, Emily's recipe, then
// Nikola and the Box of Titan (the Maphr tablet fragment), the Mark of
// Prosperity from Parman.
func TestB2Q221Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q221)
	var (
		parman   = b2Npc214(30104)
		lockirin = b2Npc214(30531)
		spiron   = b2Npc214(30532)
		balanki  = b2Npc214(30533)
		keef     = b2Npc214(30534)
		filaur   = b2Npc214(30535)
		arin     = b2Npc214(30536)
		shari    = b2Npc214(30517)
		mion     = b2Npc214(30519)
		maryse   = b2Npc214(30553)
		bolter   = b2Npc214(30554)
		torocco  = b2Npc214(30555)
		toma     = b2Npc214(30556)
		wilford  = b2Npc214(30005)
		lilith   = b2Npc214(30368)
		piotur   = b2Npc214(30597)
		bright   = b2Npc214(30466)
		emily    = b2Npc214(30620)
		nikola   = b2Npc214(30621)
		box      = b2Npc214(30622)
	)

	b2Refused214(t, h, b2Q221, parman, &scripttest.Player{ObjectID: 0x10000001, Name: "Human", ClassID: 4, Level: 40}, "30104-01.htm")
	b2Refused214(t, h, b2Q221, parman, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 54, Race: 4, Level: 36}, "30104-02.htm")
	b2Refused214(t, h, b2Q221, parman, &scripttest.Player{ObjectID: 0x10000003, Name: "Novice", ClassID: 53, Race: 4, Level: 40}, "30104-01a.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q221, p: &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 54, Race: 4, Level: 37}}
	w.talk(parman, "30104-03.htm")
	h.World().ClearRecords()
	w.event(parman, "30104-04.htm", "30104-04e.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{3239: 1, 7562: 50})
	if got := w.p.Vars["secondClassChange37"]; got != "true" {
		t.Errorf("secondClassChange37 = %q, want true", got)
	}
	w.talk(parman, "30104-05.htm")

	// Lockirin's collection.
	w.talk(lockirin, "30531-01.htm")
	w.event(lockirin, "30531-02.htm", "30531-02.htm")
	w.event(lockirin, "30531-03.htm", "30531-03.htm")
	w.items("license", map[int]int64{3246: 1, 3247: 1, 3248: 1, 3249: 1, 3250: 1, 3251: 1})
	w.event(lockirin, "30531-02.htm", "30531-04.htm")
	w.talk(lockirin, "30531-04.htm")
	// Spiron: Shari's contribution.
	w.talk(spiron, "30532-01.htm")
	w.talk(spiron, "30532-02.htm")
	w.talk(shari, "30517-01.htm")
	w.talk(shari, "30517-02.htm")
	w.talk(spiron, "30532-03.htm")
	w.talk(spiron, "30532-04.htm")
	w.items("receipt 1", map[int]int64{3247: 0, 3252: 0, 3258: 1})
	// Balanki: Mion's and Maryse's contributions.
	w.talk(balanki, "30533-01.htm")
	w.talk(mion, "30519-01.htm")
	w.talk(maryse, "30553-01.htm")
	w.talk(maryse, "30553-02.htm")
	w.p.Give(1867, 100)
	w.talk(maryse, "30553-03.htm")
	w.talk(maryse, "30553-04.htm")
	w.items("Maryse", map[int]int64{1867: 0, 3255: 0, 3254: 1, 3253: 1})
	w.talk(balanki, "30533-03.htm")
	w.talk(balanki, "30533-04.htm")
	w.items("receipt 2", map[int]int64{3248: 0, 3254: 0, 3253: 0, 3259: 1})
	// Keef: Torocco's procuration and 5000 adena.
	w.talk(keef, "30534-01.htm")
	w.talk(keef, "30534-02.htm")
	w.talk(torocco, "30555-01.htm")
	w.event(torocco, "30555-02.htm", "30555-02.htm")
	w.talk(torocco, "30555-03.htm")
	w.talk(keef, "30534-03.htm")
	w.event(keef, "30534-03a.htm", "30534-03a.htm")
	w.items("no adena", map[int]int64{3263: 1, 3260: 0})
	w.p.Give(57, 5000)
	w.event(keef, "30534-03a.htm", "30534-03b.htm")
	w.talk(keef, "30534-04.htm")
	w.items("receipt 3", map[int]int64{3249: 0, 57: 0, 3263: 0, 3260: 1})
	// Filaur: Bolter's receipt.
	w.talk(filaur, "30535-01.htm")
	w.talk(bolter, "30554-01.htm")
	w.talk(bolter, "30554-02.htm")
	w.talk(filaur, "30535-03.htm")
	w.talk(filaur, "30535-04.htm")
	w.items("receipt 4", map[int]int64{3250: 0, 3257: 0, 3261: 1})
	// Arin: Toma's contribution.
	w.talk(arin, "30536-01.htm")
	w.talk(toma, "30556-01.htm")
	w.talk(toma, "30556-02.htm")
	w.talk(arin, "30536-03.htm")
	w.talk(arin, "30536-04.htm")
	w.items("receipt 5", map[int]int64{3251: 0, 3256: 0, 3262: 1})
	h.World().ClearRecords()
	w.talk(lockirin, "30531-05.htm")
	w.cond("account book", 1)
	b2Sounds(t, "account book", h, w.p, b2ItemGet)
	w.items("account book", map[int]int64{3246: 0, 3258: 0, 3259: 0, 3260: 0, 3261: 0, 3262: 0, 3241: 1})
	w.talk(lockirin, "30531-06.htm")

	// Wilford's brooch for Lilith's wafer.
	w.talk(wilford, "30005-01.htm")
	w.event(wilford, "30005-04.htm", "30005-04.htm")
	w.talk(wilford, "30005-05.htm")
	w.talk(lilith, "30368-01.htm")
	w.event(lilith, "30368-03.htm", "30368-03.htm")
	w.items("wafer", map[int]int64{3428: 0, 3244: 1})
	w.talk(lilith, "30368-04.htm")
	w.talk(wilford, "30005-06.htm")

	// Piotur's seed.
	w.talk(piotur, "30597-01.htm")
	w.event(piotur, "30597-02.htm", "30597-02.htm")
	w.items("seed", map[int]int64{3242: 1})
	w.talk(piotur, "30597-03.htm")
	w.cond("three of four", 1)

	// Bright's bouquet for Emily's recipe.
	w.talk(bright, "30466-01.htm")
	w.event(bright, "30466-03.htm", "30466-03.htm")
	w.items("Bright list", map[int]int64{3264: 1})
	w.talk(bright, "30466-04.htm")
	w.rand(rnd.Get(10, 3), rnd.Get(10, 2), rnd.Get(10, 5), rnd.Get(10, 7))
	w.kill(b2Npc214(20223), 2)
	w.kill(b2Npc214(20154), 1)
	w.kill(b2Npc214(20155), 1)
	w.kill(b2Npc214(20156), 18)
	w.kill(b2Npc214(20228), 11)
	w.items("petals and moss", map[int]int64{3265: 20, 3266: 10})
	w.talk(bright, "30466-05.htm")
	w.items("bouquet", map[int]int64{3264: 0, 3265: 0, 3266: 0, 3267: 1})
	w.talk(bright, "30466-06.htm")
	w.talk(emily, "30620-01.htm")
	w.event(emily, "30620-03.htm", "30620-03.htm")
	w.cond("recipe", 2)
	w.items("recipe", map[int]int64{3267: 0, 3243: 1})
	w.talk(emily, "30620-04.htm")
	w.talk(bright, "30466-07.htm")

	// Parman wants level 38.
	w.talk(parman, "30104-06.htm")
	w.event(parman, "30104-07.htm", "30104-07.htm")
	w.cond("level 37", 3)
	w.items("level 37", map[int]int64{3242: 0, 3243: 0, 3244: 0, 3241: 0, 3239: 0, 3268: 1})
	w.talk(parman, "30104-09.htm")
	w.p.Level = 38
	w.talk(parman, "30104-10.htm")
	w.cond("ring 2", 4)
	w.items("ring 2", map[int]int64{3268: 0, 3269: 1, 3240: 1})
	w.talk(parman, "30104-11.htm")

	// Nikola and the Box of Titan.
	w.talk(nikola, "30621-01.htm")
	w.items("Nikola", map[int]int64{3269: 0})
	w.event(nikola, "30621-04.htm", "30621-04.htm")
	w.cond("clay", 5)
	w.items("clay", map[int]int64{3270: 1})
	w.talk(nikola, "30621-05.htm")
	w.talk(box, "30622-01.htm")
	w.event(box, "30622-02.htm", "30622-02.htm")
	w.cond("keyhole", 6)
	w.items("keyhole", map[int]int64{3270: 0, 3271: 1})
	w.talk(box, "30622-05.htm")
	w.talk(nikola, "30621-06.htm")
	w.cond("Nikola list", 7)
	w.items("Nikola list", map[int]int64{3271: 0, 3272: 1, 3023: 1})
	w.talk(nikola, "30621-07.htm")
	draws := []rnd.Draw{rnd.Get(10, 2)}
	for range 40 {
		draws = append(draws, rnd.Get(10, 1))
	}
	w.rand(draws...)
	w.kill(b2Npc214(20157), 21)
	w.kill(b2Npc214(20231), 10)
	w.kill(b2Npc214(20233), 9)
	w.cond("one thorn short", 7)
	w.kill(b2Npc214(20233), 1)
	w.cond("Nikola's list done", 8)
	w.items("Nikola's list done", map[int]int64{3273: 20, 3274: 10, 3275: 10})
	w.talk(nikola, "30621-07.htm")
	w.p.Give(3030, 1) // the key of Titan, crafted from the recipe
	w.talk(nikola, "30621-08.htm")
	w.talk(box, "30622-03.htm")
	w.event(box, "30622-04.htm", "30622-04.htm")
	w.cond("tablet", 9)
	w.items("tablet", map[int]int64{3030: 0, 3272: 0, 3023: 0, 3273: 0, 3274: 0, 3275: 0, 3245: 1})
	w.talk(nikola, "30621-09.htm")

	h.World().ClearRecords()
	w.talk(parman, "30104-13.htm")
	b2Quest(t, "mark", w.p, b2Q221, scripting.StateCompleted, 9)
	w.items("mark", map[int]int64{3245: 0, 3240: 0, 3238: 1})
	b2ExpSp(t, "mark", w.p, 12969, 1000)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q221, w.p, parman)
	b2Completed(t, "completed", reply, ok)
}
