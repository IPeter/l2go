package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	fixQ234 = "Q00234_FatesWhisper"

	fixQ234ReiriaSoulOrb   = 4666
	fixQ234KermonScepter   = 4667
	fixQ234GolkondaScepter = 4668
	fixQ234HallateScepter  = 4669
	fixQ234ReorinHammer    = 4670
	fixQ234ReorinMold      = 4671
	fixQ234Varnish         = 4672
	fixQ234PipetteKnife    = 4665
	fixQ234RedPipetteKnife = 4673
	fixQ234CrystalB        = 1460
	fixQ234StarOfDestiny   = 5011
)

// fixQ234Pages puts the pages the quest fills in into the fake HTML folder;
// each shows its name and the weapon name.
func fixQ234Pages(w *scripttest.World) {
	for _, p := range []string{"31002-12.htm", "31002-13.htm", "31002-14.htm", "31002-15.htm", "31002-AGradeList.htm"} {
		w.SetPage("scripts/quests/"+fixQ234+"/"+p, "<html>"+p+"|%weaponname%</html>")
	}
}

// fixQ234Event sends an event that answers with filled-in HTML.
func fixQ234Event(t *testing.T, q *b3aQuest, p *scripttest.Player, npc *scripttest.Npc, event, html string) {
	t.Helper()
	reply, ok := q.h.Event(q.name, event, p, npc)
	b3cHTML(t, event, reply, ok, html)
}

// Fate's Whisper: the four raid boss chests, Cliff's varnish, Ferris's
// hammer, Baium's blood on the Pipette Knife, 984 B grade crystals, then a
// B grade weapon becomes an A grade one. Baium keeps the quest behind grand
// bosses.
func TestFixQ234Walkthrough(t *testing.T) {
	q := b3aStart(t, fixQ234)
	b3cRequires(t, q.h, q.name, "grand_bosses")
	w := q.h.World()
	fixQ234Pages(w)
	w.ItemNames[80] = "Tallum Blade"
	maestro := &scripttest.Npc{ObjectID: 0x60000001, ID: 31002, Name: "Maestro Reorin"}
	cliff := &scripttest.Npc{ObjectID: 0x60000002, ID: 30182, Name: "Cliff"}
	ferris := &scripttest.Npc{ObjectID: 0x60000003, ID: 30847, Name: "Ferris"}
	zenkin := &scripttest.Npc{ObjectID: 0x60000004, ID: 30178, Name: "Zenkin"}
	kaspar := &scripttest.Npc{ObjectID: 0x60000005, ID: 30833, Name: "Kaspar"}
	baium := &scripttest.Npc{ObjectID: 0x60000006, ID: 29020, Name: "Baium"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	q.talk(young, maestro, "31002-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	q.talk(hero, maestro, "31002-02.htm")
	q.event(hero, maestro, "31002-03.htm", "31002-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, maestro, "31002-04b.htm")

	// Each raid boss leaves its chest where it died.
	bosses := []struct {
		boss, chest int
	}{{25035, 31027}, {25054, 31028}, {25126, 31029}, {25220, 31030}}
	chests := map[int]*scripttest.Npc{}
	for i, b := range bosses {
		boss := &scripttest.Npc{ObjectID: 0x60000010 + int32(i), ID: b.boss, Name: "Raid boss", X: 1000 * (i + 1), Y: -2000, Z: -300, Heading: 500}
		q.kill(hero, boss)
		got := fixSpawnedOf(w, b.chest)
		if len(got) != 1 || got[0].X != boss.X || got[0].Y != boss.Y || got[0].Z != boss.Z || got[0].Heading != 500 {
			t.Fatalf("boss %d: chests %+v", b.boss, got)
		}
		chests[b.chest] = got[0]
	}

	q.talk(hero, chests[31027], "31027-01.htm")
	b3aWantItems(t, "soul orb", hero, map[int]int64{fixQ234ReiriaSoulOrb: 1})
	q.talk(hero, chests[31027], "31027-02.htm")
	q.talk(hero, maestro, "31002-05.htm")
	q.want("soul orb", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "soul orb given", hero, map[int]int64{fixQ234ReiriaSoulOrb: 0})

	q.talk(hero, maestro, "31002-05c.htm")
	q.talk(hero, chests[31028], "31028-01.htm")
	q.talk(hero, chests[31028], "31028-02.htm")
	q.talk(hero, chests[31029], "31029-01.htm")
	q.talk(hero, chests[31030], "31030-01.htm")
	b3aWantItems(t, "scepters", hero, map[int]int64{fixQ234KermonScepter: 1, fixQ234GolkondaScepter: 1, fixQ234HallateScepter: 1})
	q.talk(hero, maestro, "31002-06.htm")
	q.want("scepters", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "scepters given", hero, map[int]int64{fixQ234KermonScepter: 0, fixQ234GolkondaScepter: 0, fixQ234HallateScepter: 0})

	q.talk(hero, maestro, "31002-06b.htm")
	q.talk(hero, cliff, "30182-01.htm")
	q.event(hero, cliff, "30182-01c.htm", "30182-01c.htm")
	q.talk(hero, cliff, "30182-02.htm")
	q.talk(hero, maestro, "31002-07.htm")
	q.want("varnish", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "varnish", hero, map[int]int64{fixQ234Varnish: 0})

	q.talk(hero, maestro, "31002-07b.htm")
	q.talk(hero, ferris, "30847-01.htm")
	q.talk(hero, ferris, "30847-02.htm")
	q.talk(hero, maestro, "31002-08.htm")
	q.want("hammer", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "hammer", hero, map[int]int64{fixQ234ReorinHammer: 0})
	q.talk(hero, maestro, "31002-08b.htm")

	q.talk(hero, zenkin, "30178-01.htm")
	q.event(hero, zenkin, "30178-01a.htm", "30178-01a.htm")
	q.want("zenkin", hero, scripting.StateStarted, 6)
	q.talk(hero, zenkin, "30178-02.htm")
	q.talk(hero, kaspar, "30833-01.htm")
	q.event(hero, kaspar, "30833-01b.htm", "30833-01b.htm")
	q.want("kaspar", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "knife", hero, map[int]int64{fixQ234PipetteKnife: 1})
	q.talk(hero, kaspar, "30833-02.htm")

	// Baium's blood needs the knife in hand.
	w.ClearRecords()
	q.h.Attack(hero, baium, 100)
	b3aWantItems(t, "bare hands", hero, map[int]int64{fixQ234PipetteKnife: 1, fixQ234RedPipetteKnife: 0})
	hero.Weapon = 4
	q.h.Attack(hero, baium, 100)
	b3aWantItems(t, "other weapon", hero, map[int]int64{fixQ234PipetteKnife: 1, fixQ234RedPipetteKnife: 0})
	hero.Weapon = fixQ234PipetteKnife
	q.h.Attack(hero, baium, 100)
	b3aWantItems(t, "blood", hero, map[int]int64{fixQ234PipetteKnife: 0, fixQ234RedPipetteKnife: 1})
	b3cSounds(t, "blood", w, hero, b3aSoundItemGet)
	hero.Weapon = 0
	// Someone else's hit does nothing.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 80, Weapon: fixQ234PipetteKnife}
	q.h.Attack(other, baium, 100)
	if len(w.Given) != 1 {
		t.Errorf("gives after the hits: %+v", w.Given)
	}

	q.talk(hero, kaspar, "30833-03.htm")
	q.want("mold", hero, scripting.StateStarted, 8)
	b3aWantItems(t, "mold", hero, map[int]int64{fixQ234RedPipetteKnife: 0, fixQ234ReorinMold: 1})
	q.talk(hero, kaspar, "30833-04.htm")
	q.talk(hero, maestro, "31002-09.htm")
	q.want("mold given", hero, scripting.StateStarted, 9)
	b3aWantItems(t, "mold given", hero, map[int]int64{fixQ234ReorinMold: 0})

	hero.Give(fixQ234CrystalB, 983)
	q.talk(hero, maestro, "31002-09b.htm")
	hero.Give(fixQ234CrystalB, 2)
	q.talk(hero, maestro, "31002-BGradeList.htm")
	q.want("crystals", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "crystals", hero, map[int]int64{fixQ234CrystalB: 1})
	q.talk(hero, maestro, "31002-BGradeList.htm")

	// Pick a B grade weapon, change the mind, then confirm.
	fixQ234Event(t, q, hero, maestro, "selectBGrade_97", "<html>31002-13.htm|Lance</html>")
	fixQ234Event(t, q, hero, maestro, "selectBGrade_79", "<html>31002-13.htm|Sword of Damascus</html>")
	fixQ234Event(t, q, hero, maestro, "confirmWeapon", "<html>31002-14.htm|Sword of Damascus</html>")
	q.event(hero, maestro, "selectBGrade_97", "") // confirmed: no change
	if v, _ := hero.Quest(fixQ234).Get("weaponId"); v != "79" {
		t.Errorf("weaponId %q, want 79", v)
	}

	// Without the weapon Reorin waits for it.
	reply, ok := q.h.Talk(fixQ234, hero, maestro)
	b3cHTML(t, "no weapon", reply, ok, "<html>31002-15.htm|Sword of Damascus</html>")
	fixQ234Event(t, q, hero, maestro, "selectAGrade_80", "<html>31002-15.htm|Sword of Damascus</html>")

	hero.Give(79, 1)
	reply, ok = q.h.Talk(fixQ234, hero, maestro)
	b3cHTML(t, "weapon", reply, ok, "<html>31002-AGradeList.htm|Sword of Damascus</html>")
	w.ClearRecords()
	fixQ234Event(t, q, hero, maestro, "selectAGrade_80", "<html>31002-12.htm|Tallum Blade</html>")
	q.want("done", hero, scripting.StateCompleted, 10)
	b3aWantItems(t, "reward", hero, map[int]int64{79: 0, 80: 1, fixQ234StarOfDestiny: 1})
	fixWantAction(t, "reward", w, "social", hero.ObjectID, 0, 3)
	b3cSounds(t, "reward", w, hero, b3aSoundFinish)
	q.talk(hero, maestro, b3aCompleted)

	// An A grade pick before the B grade one is confirmed.
	third := &scripttest.Player{ObjectID: 0x10000004, Name: "Third", Level: 76}
	q.talk(third, maestro, "31002-02.htm")
	q.event(third, maestro, "31002-03.htm", "31002-03.htm")
	q.event(third, maestro, "selectAGrade_80", "31002-16.htm")
	b3aWantItems(t, "early pick", third, map[int]int64{80: 0, fixQ234StarOfDestiny: 0})
}
