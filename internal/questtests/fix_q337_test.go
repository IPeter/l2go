package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	fixQ337Feather        = 3852
	fixQ337Mark           = 3864
	fixQ337Remains        = 3857
	fixQ337Totem          = 3858
	fixQ337KranrotSkin    = 3855
	fixQ337HamrutLeg      = 3856
	fixQ337DrakeTalons    = 3854
	fixQ337StalkerHorn    = 3853
	fixQ337FirstFragment  = 3859
	fixQ337MaraFang       = 3862
	fixQ337SecondFragment = 3860
	fixQ337MusfelFang     = 3863
	fixQ337Herald         = 3890
	fixQ337ThirdFragment  = 3861
	fixQ337PortalStone    = 3865
)

// fixQ337Attack hits a jewel at hp of 1000 HP for damage, with the given
// random draws (none: no draw allowed).
func fixQ337Attack(q *b3aQuest, p *scripttest.Player, jewel *scripttest.Npc, hp float64, damage int, draws ...rnd.Draw) {
	q.t.Helper()
	jewel.HP, jewel.MaxHP = hp, 1000
	q.h.SetRand(rnd.NewScripted(q.t, draws...))
	q.h.Attack(p, jewel, damage)
}

// fixQ337WantGuards checks n guards of a template spawned around a jewel
// (offset dx, dy), each running at p with 500 hate.
func fixQ337WantGuards(t *testing.T, w *scripttest.World, npcID, n int, jewel *scripttest.Npc, dx, dy int, p *scripttest.Player) {
	t.Helper()
	got := fixSpawnedOf(w, npcID)
	if len(got) != n {
		t.Fatalf("guards %d: %d spawned, want %d", npcID, len(got), n)
	}
	for _, g := range got {
		if g.X != jewel.X+dx || g.Y != jewel.Y+dy || g.Z != jewel.Z || !g.Running {
			t.Errorf("guard %+v, want at %d, %d running", g, jewel.X+dx, jewel.Y+dy)
		}
		fixWantAction(t, "guard hate", w, "hate", g.ObjectID, p.ObjectID, 500)
		fixWantAction(t, "guard attack", w, "attack", g.ObjectID, p.ObjectID, 0)
	}
}

// Audience with the Land Dragon: four Marks of Watchman from the hunters
// around Giran, two from the abyssal jewels (attack drops and guard spawns),
// then the third jewel's fragment for Theodric's Portal Stone.
func TestFixQ337Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00337_AudienceWithTheLandDragon")
	b3cRequires(t, q.h, q.name)
	w := q.h.World()
	gabrielle := &scripttest.Npc{ObjectID: 0x60000001, ID: 30753, Name: "Gabrielle"}
	orven := &scripttest.Npc{ObjectID: 0x60000002, ID: 30857, Name: "Orven"}
	kendra := &scripttest.Npc{ObjectID: 0x60000003, ID: 30851, Name: "Kendra"}
	chakiris := &scripttest.Npc{ObjectID: 0x60000004, ID: 30705, Name: "Chakiris"}
	kaiena := &scripttest.Npc{ObjectID: 0x60000005, ID: 30720, Name: "Kaiena"}
	moke := &scripttest.Npc{ObjectID: 0x60000006, ID: 30498, Name: "Moke"}
	helton := &scripttest.Npc{ObjectID: 0x60000007, ID: 30678, Name: "Helton"}
	gilmore := &scripttest.Npc{ObjectID: 0x60000008, ID: 30754, Name: "Gilmore"}
	theodric := &scripttest.Npc{ObjectID: 0x60000009, ID: 30755, Name: "Theodric"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 49}
	q.talk(young, gabrielle, "30753-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 50}
	q.talk(hero, gabrielle, "30753-01.htm")
	q.event(hero, gabrielle, "30753-05.htm", "30753-05.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{fixQ337Feather: 1})
	q.talk(hero, gabrielle, "30753-06.htm")
	q.event(hero, gabrielle, "30753-09.htm", "") // not yet

	// Orven: the Blood Queen calls eight Sacrifices; one of them has the
	// remains.
	q.talk(hero, orven, "30857-01.htm")
	queen := &scripttest.Npc{ObjectID: 0x60000010, ID: 18001, Name: "Blood Queen", X: 10000, Y: 20000, Z: -3000, Heading: 100}
	q.kill(hero, queen, b3aDraws(16, rnd.Between(-100, 100, 40))...)
	sacrifices := fixSpawnedOf(w, 27171)
	if len(sacrifices) != 8 || sacrifices[0].X != 10040 || sacrifices[0].Y != 20040 || sacrifices[0].Z != -3000 {
		t.Fatalf("sacrifices %+v", sacrifices)
	}
	q.kill(hero, sacrifices[0])
	b3aWantItems(t, "remains", hero, map[int]int64{fixQ337Remains: 1})
	q.kill(hero, sacrifices[1]) // one is enough
	b3aWantItems(t, "remains", hero, map[int]int64{fixQ337Remains: 1})
	q.kill(hero, queen) // the remains are in the bag: no more Sacrifices
	q.talk(hero, orven, "30857-02.htm")
	b3aWantItems(t, "orven", hero, map[int]int64{fixQ337Remains: 0, fixQ337Mark: 1})
	q.talk(hero, orven, "30857-03.htm")

	// Kendra: a Harit shaman sometimes calls three Zealots.
	q.talk(hero, kendra, "30851-01.htm")
	shaman := &scripttest.Npc{ObjectID: 0x60000011, ID: 20644, Name: "Harit Lizardman Shaman", X: -5000, Y: 6000, Z: -100}
	q.kill(hero, shaman, rnd.Get(5, 1))
	if got := fixSpawnedOf(w, 27172); len(got) != 0 {
		t.Fatalf("zealots on a miss: %+v", got)
	}
	q.kill(hero, shaman, append([]rnd.Draw{rnd.Get(5, 0)}, b3aDraws(6, rnd.Between(-50, 50, -50))...)...)
	zealots := fixSpawnedOf(w, 27172)
	if len(zealots) != 3 || zealots[2].X != -5050 || zealots[2].Y != 5950 {
		t.Fatalf("zealots %+v", zealots)
	}
	q.kill(hero, zealots[0])
	q.talk(hero, kendra, "30851-02.htm")
	b3aWantItems(t, "kendra", hero, map[int]int64{fixQ337Totem: 0, fixQ337Mark: 2})

	// Chakiris and Kaiena: two drops each.
	q.talk(hero, chakiris, "30705-01.htm")
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20650, Name: "Kranrot"})
	q.talk(hero, chakiris, "30705-01.htm")
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20649, Name: "Hamrut"})
	q.talk(hero, chakiris, "30705-02.htm")
	b3aWantItems(t, "chakiris", hero, map[int]int64{fixQ337KranrotSkin: 0, fixQ337HamrutLeg: 0, fixQ337Mark: 3})
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20650, Name: "Kranrot"})
	b3aWantItems(t, "chakiris done", hero, map[int]int64{fixQ337KranrotSkin: 0})
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000014, ID: 20680, Name: "Marsh Drake"})
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000015, ID: 20679, Name: "Marsh Stalker"})
	q.talk(hero, kaiena, "30720-02.htm")
	b3aWantItems(t, "kaiena", hero, map[int]int64{fixQ337DrakeTalons: 0, fixQ337StalkerHorn: 0, fixQ337Mark: 4})
	q.talk(hero, kaiena, "30720-04.htm")

	q.talk(hero, gabrielle, "30753-08.htm")
	w.ClearRecords()
	q.event(hero, gabrielle, "30753-09.htm", "30753-09.htm")
	q.want("four marks", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "four marks", hero, map[int]int64{fixQ337Mark: 0})
	b3cSounds(t, "four marks", w, hero, b3aSoundMiddle)
	q.talk(hero, gabrielle, "30753-10.htm")
	q.talk(hero, gilmore, "30754-01.htm")
	q.talk(hero, theodric, "30755-01.htm")

	// Moke's jewel: full health arms the guards, half health calls twenty
	// Maras once, low health may drop the fragment.
	q.talk(hero, moke, "30498-01.htm")
	q.talk(hero, moke, "30498-02.htm")
	jewel1 := &scripttest.Npc{ObjectID: 0x60000020, ID: 27165, Name: "Abyssal Jewel", X: 30000, Y: 40000, Z: -2000, Heading: 7}
	fixQ337Attack(q, hero, jewel1, 950, 10)
	w.ClearRecords()
	fixQ337Attack(q, hero, jewel1, 500, 10, append([]rnd.Draw{rnd.Get(100, 32)}, b3aDraws(40, rnd.Between(-150, 150, 150))...)...)
	fixQ337WantGuards(t, w, 27168, 20, jewel1, 150, 150, hero)
	fixQ337Attack(q, hero, jewel1, 500, 10, rnd.Get(100, 0)) // the guards came already
	if got := fixSpawnedOf(w, 27168); len(got) != 20 {
		t.Fatalf("maras %d after the second hit", len(got))
	}
	fixQ337Attack(q, hero, jewel1, 200, 10, rnd.Get(100, 33))
	b3aWantItems(t, "missed fragment", hero, map[int]int64{fixQ337FirstFragment: 0})
	fixQ337Attack(q, hero, jewel1, 200, 10, rnd.Get(100, 32))
	b3aWantItems(t, "first fragment", hero, map[int]int64{fixQ337FirstFragment: 1})
	q.kill(hero, fixSpawnedOf(w, 27168)[0])
	q.talk(hero, moke, "30498-03.htm")
	b3aWantItems(t, "moke", hero, map[int]int64{fixQ337FirstFragment: 0, fixQ337MaraFang: 0, fixQ337Mark: 1})
	q.talk(hero, moke, "30498-04.htm")

	// Helton's jewel.
	q.talk(hero, helton, "30678-01.htm")
	jewel2 := &scripttest.Npc{ObjectID: 0x60000021, ID: 27166, Name: "Abyssal Jewel", X: -30000, Y: 40000, Z: -2000}
	fixQ337Attack(q, hero, jewel2, 100, 0, rnd.Get(100, 0))
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000022, ID: 27169, Name: "Jewel Guardian Musfel"})
	q.talk(hero, helton, "30678-03.htm")
	b3aWantItems(t, "helton", hero, map[int]int64{fixQ337SecondFragment: 0, fixQ337MusfelFang: 0, fixQ337Mark: 2})
	q.talk(hero, helton, "30678-05.htm")
	q.talk(hero, moke, "30498-05.htm")

	q.talk(hero, gabrielle, "30753-11.htm")
	q.want("herald", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "herald", hero, map[int]int64{fixQ337Feather: 0, fixQ337Mark: 1, fixQ337Herald: 1})
	q.talk(hero, gabrielle, "30753-12.htm")
	q.talk(hero, theodric, "30755-02.htm")
	// The second jewel does nothing at cond 3.
	fixQ337Attack(q, hero, jewel2, 100, 0)

	q.talk(hero, gilmore, "30754-02.htm")
	q.want("gilmore", hero, scripting.StateStarted, 4)
	q.talk(hero, gilmore, "30754-04.htm")
	q.talk(hero, theodric, "30755-03.htm")
	q.talk(hero, gabrielle, "30753-13.htm")

	// A cave monster sometimes brings the third jewel; three Pytons guard it.
	maiden := &scripttest.Npc{ObjectID: 0x60000023, ID: 20134, Name: "Cave Maiden", X: 50000, Y: 60000, Z: -4000}
	q.kill(hero, maiden, rnd.Get(5, 3))
	q.kill(hero, maiden, rnd.Get(5, 0), rnd.Between(-50, 50, 20), rnd.Between(-50, 50, -20))
	jewels := fixSpawnedOf(w, 27167)
	if len(jewels) != 1 || jewels[0].X != 50020 || jewels[0].Y != 59980 {
		t.Fatalf("third jewel %+v", jewels)
	}
	jewel3 := jewels[0]
	fixQ337Attack(q, hero, jewel3, 950, 0)
	fixQ337Attack(q, hero, jewel3, 600, 0, append([]rnd.Draw{rnd.Get(100, 10)}, b3aDraws(6, rnd.Between(-150, 150, -150))...)...)
	fixQ337WantGuards(t, w, 27170, 3, jewel3, -150, -150, hero)
	fixQ337Attack(q, hero, jewel3, 300, 0, rnd.Get(100, 1))
	b3aWantItems(t, "third fragment", hero, map[int]int64{fixQ337ThirdFragment: 1})
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000024, ID: 20246, Name: "Cave Keeper"}, rnd.Get(5, 0))
	if got := fixSpawnedOf(w, 27167); len(got) != 1 {
		t.Fatalf("a jewel after the fragment: %+v", got)
	}

	q.talk(hero, gilmore, "30754-05.htm")
	q.talk(hero, theodric, "30755-04.htm")
	w.ClearRecords()
	q.event(hero, theodric, "30755-05.htm", "30755-05.htm")
	b3cNoQuest(t, "done", hero, q.name)
	b3aWantItems(t, "reward", hero, map[int]int64{fixQ337PortalStone: 1, fixQ337ThirdFragment: 0, fixQ337Herald: 0, fixQ337Mark: 0})
	b3cSounds(t, "reward", w, hero, b3aSoundFinish)
}
