package questtests

import (
	"context"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	fixQ348 = "Q00348_AnArrogantSearch"

	fixQ348TitanPowerstone   = 4287
	fixQ348FirstLetter       = 4288
	fixQ348SecondLetter      = 4289
	fixQ348ThirdLetter       = 4290
	fixQ348FirstKey          = 4291
	fixQ348SecondKey         = 4292
	fixQ348ThirdKey          = 4293
	fixQ348BookOfSaint       = 4397
	fixQ348BloodOfSaint      = 4398
	fixQ348BoughOfSaint      = 4399
	fixQ348WhiteFabricTribe  = 4294
	fixQ348WhiteFabricAngels = 5232
	fixQ348BloodedFabric     = 4295
	fixQ348Antidote          = 1831
	fixQ348HealingPotion     = 1061
)

// fixQ348Spawn runs the spawn handlers for npc.
func fixQ348Spawn(t *testing.T, q *b3aQuest, npc *scripttest.Npc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := q.h.Engine().Spawn(ctx, npc.Ref()); err != nil {
		t.Fatalf("spawn handler failed: %v", err)
	}
}

// fixQ348Hit hits npc at hp of 1000 HP.
func fixQ348Hit(q *b3aQuest, p *scripttest.Player, npc *scripttest.Npc, hp float64) {
	q.t.Helper()
	npc.HP, npc.MaxHP = hp, 1000
	q.h.Attack(p, npc, 10)
}

// fixQ348Var checks a quest variable.
func fixQ348Var(t *testing.T, p *scripttest.Player, name, want string) {
	t.Helper()
	if got, _ := p.Quest(fixQ348).Get(name); got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// fixQ348Ark talks to an ark and wants page and one guard of guardID
// spawned on the ark.
func fixQ348Ark(t *testing.T, q *b3aQuest, p *scripttest.Player, ark *scripttest.Npc, page string, guardID int) *scripttest.Npc {
	t.Helper()
	w := q.h.World()
	q.talk(p, ark, page)
	q.talk(p, ark, page) // the guard is out already
	got := fixSpawnedOf(w, guardID)
	if len(got) != 1 || got[0].X != ark.X || got[0].Y != ark.Y || got[0].Z != ark.Z {
		t.Fatalf("%s: guards %+v", page, got)
	}
	return got[0]
}

// An Arrogant Search: the Titan Powerstone, Hanellin's three letters, the
// three arks and their guards (spawn speech, attack speech, the fleeing
// Angel Killer), the tribe's White Fabric bloodied, then the fabrics for
// Gustav, Hardin and Iason Heine and the angels' fabrics.
func TestFixQ348Walkthrough(t *testing.T) {
	q := b3aStart(t, fixQ348)
	b3cRequires(t, q.h, q.name)
	w := q.h.World()
	hanellin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30864, Name: "Hanellin"}
	claudia := &scripttest.Npc{ObjectID: 0x60000002, ID: 31001, Name: "Claudia Athebaldt"}
	martien := &scripttest.Npc{ObjectID: 0x60000003, ID: 30645, Name: "Martien"}
	harne := &scripttest.Npc{ObjectID: 0x60000004, ID: 30144, Name: "Harne"}
	corpse := &scripttest.Npc{ObjectID: 0x60000005, ID: 30980, Name: "Ark Guardian's Corpse", X: -418, Y: 44174, Z: -3568}
	ark1 := &scripttest.Npc{ObjectID: 0x60000006, ID: 30977, Name: "Holy Ark of Secrecy", X: -400, Y: 44200, Z: -3568}
	ark2 := &scripttest.Npc{ObjectID: 0x60000007, ID: 30978, Name: "Holy Ark of Secrecy", X: 181472, Y: 7158, Z: -2725}
	ark3 := &scripttest.Npc{ObjectID: 0x60000008, ID: 30979, Name: "Holy Ark of Secrecy", X: 50693, Y: 158674, Z: 376}
	gustav := &scripttest.Npc{ObjectID: 0x60000009, ID: 30760, Name: "Gustav Athebaldt"}
	hardin := &scripttest.Npc{ObjectID: 0x6000000a, ID: 30832, Name: "Hardin"}
	iason := &scripttest.Npc{ObjectID: 0x6000000b, ID: 30969, Name: "Iason Heine"}
	giant := &scripttest.Npc{ObjectID: 0x6000000c, ID: 20657, Name: "Lesser Giant Mage"}
	overlord := &scripttest.Npc{ObjectID: 0x6000000d, ID: 20829, Name: "Platinum Tribe Overlord"}
	shaman := &scripttest.Npc{ObjectID: 0x6000000e, ID: 20828, Name: "Platinum Tribe Shaman"}
	sealAngel := &scripttest.Npc{ObjectID: 0x6000000f, ID: 20860, Name: "Seal Angel"}

	marked := &scripttest.Player{ObjectID: 0x10000001, Name: "Marked", Level: 70, Items: map[int]int64{fixQ348BloodedFabric: 1}}
	q.talk(marked, hanellin, "30864-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 59}
	q.talk(young, hanellin, "30864-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 60}
	q.talk(hero, hanellin, "30864-02.htm")
	q.event(hero, hanellin, "30864-05.htm", "30864-05.htm")
	q.want("accept", hero, scripting.StateStarted, 2)
	fixQ348Var(t, hero, "points", "0")
	q.talk(hero, hanellin, "30864-06.htm")

	q.kill(hero, giant, rnd.Get(10, 1))
	b3aWantItems(t, "no powerstone", hero, map[int]int64{fixQ348TitanPowerstone: 0})
	q.kill(hero, giant, rnd.Get(10, 0))
	b3aWantItems(t, "powerstone", hero, map[int]int64{fixQ348TitanPowerstone: 1})
	q.kill(hero, giant, rnd.Get(10, 0)) // one is enough
	b3aWantItems(t, "one powerstone", hero, map[int]int64{fixQ348TitanPowerstone: 1})
	q.talk(hero, hanellin, "30864-07.htm")
	q.event(hero, hanellin, "30864-09.htm", "30864-09.htm")
	q.want("powerstone", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "powerstone given", hero, map[int]int64{fixQ348TitanPowerstone: 0})
	q.talk(hero, hanellin, "30864-09.htm")
	q.event(hero, hanellin, "30864-17.htm", "30864-17.htm")
	q.want("letters", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "letters", hero, map[int]int64{fixQ348FirstLetter: 1, fixQ348SecondLetter: 1, fixQ348ThirdLetter: 1})
	q.talk(hero, hanellin, "30864-24.htm")

	// The three letters, each with a radar mark.
	w.ClearRecords()
	q.talk(hero, harne, "30144-01.htm")
	q.want("harne", hero, scripting.StateStarted, 17)
	q.talk(hero, claudia, "31001-01.htm")
	q.want("claudia", hero, scripting.StateStarted, 9)
	q.talk(hero, martien, "30645-01.htm")
	q.want("martien", hero, scripting.StateStarted, 13)
	b3aWantItems(t, "letters given", hero, map[int]int64{fixQ348FirstLetter: 0, fixQ348SecondLetter: 0, fixQ348ThirdLetter: 0})
	if got := fixActions(w, "radar"); len(got) != 3 || got[0].Loc.X != -418 || got[1].Loc.X != 181472 || got[2].Loc.X != 50693 {
		t.Errorf("radar %+v", got)
	}
	q.talk(hero, harne, "30144-03.htm")

	// The corpse calls the Angel Killer, who flees at half health and
	// leaves the key in the corpse.
	killer := fixQ348Ark(t, q, hero, corpse, "30980-01.htm", 27184)
	q.want("angel killer", hero, scripting.StateStarted, 18)
	w.ClearRecords()
	fixQ348Spawn(t, q, killer)
	if !fixSays(w, killer.ObjectID, "I have the key, do you wish to steal it?") {
		t.Errorf("angel killer spawn speech: %+v", w.Actions)
	}
	fixQ348Hit(q, hero, killer, 800)
	fixQ348Hit(q, hero, killer, 700)
	if got := fixActions(w, "say"); len(got) != 2 || got[1].Text != "Haha.. Really amusing! As for the key, search the corpse!" {
		t.Errorf("angel killer attack speech %+v", got)
	}
	fixQ348Hit(q, hero, killer, 499)
	fixWantAction(t, "flee", w, "abort_attack", killer.ObjectID, 0, 0)
	fixWantAction(t, "flee", w, "delete", killer.ObjectID, 0, 0)
	if !killer.Deleted || !fixSays(w, killer.ObjectID, "Can't get rid of you... Did you get the key from the corpse?") {
		t.Errorf("flee: deleted %v, actions %+v", killer.Deleted, w.Actions)
	}
	q.want("fled", hero, scripting.StateStarted, 19)
	fixQ348Var(t, hero, "angelkiller", "1")
	q.talk(hero, corpse, "30980-02.htm")
	b3aWantItems(t, "first key", hero, map[int]int64{fixQ348FirstKey: 1})
	q.talk(hero, corpse, "30980-03.htm")
	q.talk(hero, harne, "30144-04.htm")
	q.talk(hero, ark1, "30977-02.htm")
	q.want("blood", hero, scripting.StateStarted, 20)
	b3aWantItems(t, "blood", hero, map[int]int64{fixQ348FirstKey: 0, fixQ348BloodOfSaint: 1})
	q.talk(hero, ark1, "30977-03.htm")
	q.talk(hero, hanellin, "30864-28.htm")

	// Elberoth guards the second ark.
	q.talk(hero, claudia, "31001-03.htm")
	elberoth := fixQ348Ark(t, q, hero, ark2, "30978-01.htm", 27182)
	w.ClearRecords()
	fixQ348Spawn(t, q, elberoth)
	fixQ348Hit(q, hero, elberoth, 900)
	fixQ348Hit(q, hero, elberoth, 800)
	q.kill(hero, elberoth)
	if got := fixActions(w, "say"); len(got) != 3 ||
		got[0].Text != "This does not belong to you. Take your hands out!" ||
		got[1].Text != "...I feel very sorry, but I have taken your life." ||
		got[2].Text != "Oh, dull-witted.. God, they..." {
		t.Errorf("elberoth speech %+v", got)
	}
	q.want("second key", hero, scripting.StateStarted, 11)
	q.talk(hero, claudia, "31001-04.htm")
	q.talk(hero, ark2, "30978-02.htm")
	q.want("book", hero, scripting.StateStarted, 12)
	b3aWantItems(t, "book", hero, map[int]int64{fixQ348SecondKey: 0, fixQ348BookOfSaint: 1})

	// Shadow Fang guards the third; with all three relics the cond is 21.
	shadowFang := fixQ348Ark(t, q, hero, ark3, "30979-01.htm", 27183)
	w.ClearRecords()
	fixQ348Spawn(t, q, shadowFang)
	fixQ348Hit(q, hero, shadowFang, 900)
	q.kill(hero, shadowFang)
	if got := fixActions(w, "say"); len(got) != 3 ||
		got[0].Text != "I don't believe it! Grrr!" ||
		got[1].Text != "I will cover this mountain with your blood!" ||
		got[2].Text != "You do not know.. Seven seals are.. coughs" {
		t.Errorf("shadow fang speech %+v", got)
	}
	q.want("third key", hero, scripting.StateStarted, 15)
	q.talk(hero, martien, "30645-04.htm")
	q.talk(hero, ark3, "30979-02.htm")
	q.want("bough", hero, scripting.StateStarted, 21)
	q.talk(hero, martien, "30645-05.htm")

	q.talk(hero, hanellin, "30864-29.htm")
	q.want("relics", hero, scripting.StateStarted, 22)
	b3aWantItems(t, "relics", hero, map[int]int64{fixQ348BookOfSaint: 0, fixQ348BloodOfSaint: 0, fixQ348BoughOfSaint: 0})
	q.talk(hero, hanellin, "30864-30.htm")
	hero.Give(fixQ348Antidote, 5)
	hero.Give(fixQ348HealingPotion, 1)
	q.talk(hero, hanellin, "30864-31.htm")
	b3aWantItems(t, "white fabric", hero, map[int]int64{fixQ348Antidote: 0, fixQ348HealingPotion: 0, fixQ348WhiteFabricTribe: 1})
	q.talk(hero, hanellin, "30864-31.htm")

	// The long way: 100000 points of Platinum Tribe blood.
	q.event(hero, hanellin, "30864-37.htm", "30864-37.htm")
	q.want("tribe", hero, scripting.StateStarted, 25)
	fixQ348Hit(q, hero, shaman, 900)
	fixQ348Var(t, hero, "points", "60")
	q.kill(hero, overlord)
	fixQ348Var(t, hero, "points", "760")
	q.talk(hero, hanellin, "30864-39.htm")
	hero.Quest(fixQ348).Set("points", "99300")
	q.kill(hero, overlord) // exactly 100000 is not enough
	fixQ348Var(t, hero, "points", "100000")
	w.ClearRecords()
	q.kill(hero, overlord)
	fixQ348Var(t, hero, "points", "0")
	b3aWantItems(t, "blooded", hero, map[int]int64{fixQ348WhiteFabricTribe: 0, fixQ348BloodedFabric: 1})
	b3cSounds(t, "blooded", w, hero, b3aSoundItemGet)
	q.want("blooded", hero, scripting.StateStarted, 25)
	q.talk(hero, hanellin, "30864-49.htm")

	// Ten fabrics for Gustav (3), Hardin (1) and Iason Heine (6).
	q.event(hero, hanellin, "30864-58.htm", "30864-58.htm")
	q.want("fabrics", hero, scripting.StateStarted, 27)
	q.talk(hero, gustav, "30760-03.htm") // too few: used on Baium
	fixQ348Var(t, hero, "usedonbaium", "1")
	hero.Quest(fixQ348).Unset("usedonbaium")
	hero.Give(fixQ348BloodedFabric, 9)
	q.talk(hero, hanellin, "30864-59.htm")
	q.talk(hero, gustav, "30760-01.htm")
	q.talk(hero, gustav, "30760-02.htm")
	q.talk(hero, hardin, "30832-01.htm")
	q.talk(hero, hardin, "30832-02.htm")
	q.talk(hero, iason, "30969-01.htm")
	q.talk(hero, iason, "30969-02.htm")
	b3aWantItems(t, "fabrics given", hero, map[int]int64{fixQ348BloodedFabric: 0})
	q.talk(hero, hanellin, "30864-60.htm")
	q.want("fabrics given", hero, scripting.StateStarted, 28)
	b3aWantItems(t, "fabrics paid", hero, map[int]int64{b3aAdena: 49000})
	q.talk(hero, hanellin, "30864-55.htm")

	// The angels' fabrics, paid by the bloodied one.
	q.event(hero, hanellin, "30864-56.htm", "30864-56.htm")
	q.want("angels", hero, scripting.StateStarted, 29)
	fixQ348Var(t, hero, "gustav", "0")
	b3aWantItems(t, "angels", hero, map[int]int64{fixQ348WhiteFabricAngels: 10})
	q.kill(hero, sealAngel, rnd.Get(4, 1))
	q.kill(hero, sealAngel, rnd.Get(4, 0))
	b3aWantItems(t, "angel blood", hero, map[int]int64{fixQ348WhiteFabricAngels: 9, fixQ348BloodedFabric: 1})
	q.talk(hero, hanellin, "30864-52.htm")
	hero.Give(fixQ348WhiteFabricAngels, -9) // lost
	w.ClearRecords()
	q.talk(hero, hanellin, "30864-54.htm")
	b3cNoQuest(t, "done", hero, fixQ348)
	b3aWantItems(t, "reward", hero, map[int]int64{b3aAdena: 54000, fixQ348BloodedFabric: 0})
	b3cSounds(t, "reward", w, hero, b3aSoundFinish)
}

// The short way: 80000 points with the cond 24 reward ends the quest with
// the bloodied fabric, which then bars a new start.
func TestFixQ348ShortWay(t *testing.T) {
	q := b3aStart(t, fixQ348)
	hanellin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30864, Name: "Hanellin"}
	shaman := &scripttest.Npc{ObjectID: 0x60000002, ID: 20828, Name: "Platinum Tribe Shaman"}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	q.talk(hero, hanellin, "30864-02.htm")
	q.event(hero, hanellin, "30864-05.htm", "30864-05.htm")
	hero.Quest(fixQ348).Set("cond", "22")
	hero.Give(fixQ348WhiteFabricTribe, 1)

	q.h.SetRand(rnd.NewScripted(t, rnd.Between(1, 2, 2)))
	q.event(hero, hanellin, "30864-36.htm", "30864-36.htm")
	q.want("short way", hero, scripting.StateStarted, 24)
	b3aWantItems(t, "short way", hero, map[int]int64{b3aAdena: 24000})
	q.talk(hero, hanellin, "30864-38.htm")

	hero.Quest(fixQ348).Set("points", "79950")
	fixQ348Hit(q, hero, shaman, 900)
	b3cNoQuest(t, "blooded", hero, fixQ348)
	b3aWantItems(t, "blooded", hero, map[int]int64{fixQ348BloodedFabric: 1, fixQ348WhiteFabricTribe: 0})
	q.talk(hero, hanellin, "30864-00.htm")

	// Giving up from Hanellin's offer ends the quest too.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 60}
	q.talk(other, hanellin, "30864-02.htm")
	q.event(other, hanellin, "30864-05.htm", "30864-05.htm")
	q.event(other, hanellin, "30864-57.htm", "30864-57.htm")
	b3cNoQuest(t, "given up", other, fixQ348)
}
