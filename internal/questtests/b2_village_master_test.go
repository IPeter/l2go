package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The class change village masters (scripts/village_master/), ported from
// Mobius village_master/*.

// Item IDs of the village master tests.
const (
	b2CouponD          = 8869
	b2CouponC          = 8870
	b2MedallionWarrior = 1145
	b2MarkChallenger   = 2627
	b2MarkTrust        = 2734
	b2MarkDuelist      = 2762
	b2MarkChampion     = 3276
)

// b2NoReply checks that a handler answered nothing (nil).
func b2NoReply(t *testing.T, what string, reply scripting.Reply, ok bool) {
	t.Helper()
	if ok {
		t.Errorf("%s: reply %+v, want none", what, reply)
	}
}

// A level 20 Human Fighter walks Path to a Warrior (Q401) and turns the
// Medallion of Warrior into the Warrior class at Grand Master Ramos (Plan 11
// demo items 7 and 8).
func TestB2HumanFighterBecomesWarrior(t *testing.T) {
	const q401 = "Q00401_PathOfTheWarrior"
	h := scripttest.Load(t, b2ScriptsDir, "quests/"+q401+"/"+q401+".lua")
	m := b2LoadMasters(t, "ElfHumanFighterChange1")
	auron := &scripttest.Npc{ObjectID: 0x60000001, ID: 30010, Name: "Auron"}
	simplon := &scripttest.Npc{ObjectID: 0x60000002, ID: 30253, Name: "Simplon"}
	skeleton := &scripttest.Npc{ObjectID: 0x60000003, ID: 20035, Name: "Tracker Skeleton"}
	spider := &scripttest.Npc{ObjectID: 0x60000004, ID: 20038, Name: "Venomous Spider"}
	ramos := &scripttest.Npc{ObjectID: 0x60000005, ID: 30373, Name: "Ramos"}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 20, ClassID: 0}

	// Before the quest Ramos has nothing to trade for.
	reply, ok := m.Talk("ElfHumanFighterChange1", hero, ramos)
	b2Page(t, "Ramos first", reply, ok, "30373-01.htm")
	reply, ok = m.Event("ElfHumanFighterChange1", "1", hero, ramos)
	b2Page(t, "no medallion", reply, ok, "30373-24.htm")
	m.b2NoClassChange("no medallion")

	// Path to a Warrior.
	reply, ok = h.Talk(q401, hero, auron)
	b2Page(t, "offer", reply, ok, "30010-01.htm")
	reply, ok = h.Event(q401, "30010-05.htm", hero, auron)
	b2Page(t, "fits", reply, ok, "30010-05.htm")
	h.Event(q401, "30010-06.htm", hero, auron)
	b2Quest(t, "accept", hero, q401, scripting.StateStarted, 1)
	h.Event(q401, "30253-02.htm", hero, simplon)
	b2Quest(t, "Simplon", hero, q401, scripting.StateStarted, 2)
	draws := make([]rnd.Draw, 10)
	for i := range draws {
		draws[i] = rnd.Get(10, 0)
	}
	b2Rand(t, h, draws...)
	for range 10 {
		h.Kill(hero, skeleton)
	}
	b2Quest(t, "swords", hero, q401, scripting.StateStarted, 3)
	reply, ok = h.Talk(q401, hero, simplon)
	b2Page(t, "Simplon swords", reply, ok, "30253-04.htm")
	h.Event(q401, "30010-11.htm", hero, auron)
	b2Quest(t, "sword 3", hero, q401, scripting.StateStarted, 5)
	hero.Equip(7, 1142) // Paperdoll.RHAND, the Rusted Bronze Sword
	for range 20 {
		h.Kill(hero, spider)
	}
	b2Quest(t, "legs", hero, q401, scripting.StateStarted, 6)
	reply, ok = h.Talk(q401, hero, auron)
	b2Page(t, "medallion", reply, ok, "30010-13.htm")
	b2NoQuest(t, "medallion", hero, q401)
	b2Items(t, "medallion", hero, map[int]int64{b2MedallionWarrior: 1})

	// The first class transfer at Ramos.
	reply, ok = m.Event("ElfHumanFighterChange1", "30373-02.htm", hero, ramos)
	b2Page(t, "warrior page", reply, ok, "30373-02.htm")
	reply, ok = m.Event("ElfHumanFighterChange1", "1", hero, ramos)
	b2Page(t, "warrior", reply, ok, "30373-23.htm")
	m.b2ClassChanged("warrior", hero, 1)
	if hero.BaseClass != 1 {
		t.Errorf("base class %d, want 1", hero.BaseClass)
	}
	b2Items(t, "warrior", hero, map[int]int64{b2MedallionWarrior: 0, b2CouponD: 15})

	// A Warrior is done with the first transfer; Auron sees a Warrior.
	reply, ok = m.Event("ElfHumanFighterChange1", "4", hero, ramos)
	b2Page(t, "second class", reply, ok, "30373-19.htm")
	m.b2NoClassChange("second class")
	reply, ok = h.Talk(q401, hero, auron)
	b2Page(t, "Auron again", reply, ok, "30010-01.htm")
	reply, ok = h.Event(q401, "30010-05.htm", hero, auron)
	b2Page(t, "Auron to a Warrior", reply, ok, "30010-03.htm")
}

// The refusals of the first class transfer of Human and Elven fighters.
func TestB2ElfHumanFighterChange1(t *testing.T) {
	const s = "ElfHumanFighterChange1"
	m := b2LoadMasters(t, s)
	pabris := &scripttest.Npc{ObjectID: 0x60000001, ID: 30066}

	elf := &scripttest.Player{ObjectID: 0x10000001, Name: "Elf", Level: 20, ClassID: 18}
	reply, ok := m.Talk(s, elf, pabris)
	b2Page(t, "elf", reply, ok, "30066-11.htm")
	mage := &scripttest.Player{ObjectID: 0x10000002, Name: "Mage", Level: 20, ClassID: 10}
	reply, ok = m.Talk(s, mage, pabris)
	b2Page(t, "mage", reply, ok, "30066-18.htm")
	orc := &scripttest.Player{ObjectID: 0x10000003, Name: "Orc", Level: 20, ClassID: 44}
	reply, ok = m.Talk(s, orc, pabris)
	b2Page(t, "orc", reply, ok, "30066-18.htm")

	// Below level 20, with and without the brooch.
	elf.Level = 19
	reply, ok = m.Event(s, "19", elf, pabris)
	b2Page(t, "low no proof", reply, ok, "30066-34.htm")
	elf.Give(1204, 1)
	reply, ok = m.Event(s, "19", elf, pabris)
	b2Page(t, "low with proof", reply, ok, "30066-33.htm")
	m.b2NoClassChange("low")

	// An Elven Fighter cannot become a Human class.
	elf.Level = 20
	reply, ok = m.Event(s, "4", elf, pabris)
	b2NoReply(t, "elf to knight", reply, ok)
	reply, ok = m.Event(s, "19", elf, pabris)
	b2Page(t, "elven knight", reply, ok, "30066-35.htm")
	m.b2ClassChanged("elven knight", elf, 19)
	b2Items(t, "elven knight", elf, map[int]int64{1204: 0, b2CouponD: 15})

	// Second and third classes.
	gladiator := &scripttest.Player{ObjectID: 0x10000004, Name: "Gladiator", Level: 45, ClassID: 2}
	reply, ok = m.Event(s, "1", gladiator, pabris)
	b2Page(t, "third class", reply, ok, "30066-20.htm")
	// Unknown events answer nothing; listed pages show.
	reply, ok = m.Event(s, "30066-99.htm", elf, pabris)
	b2NoReply(t, "unknown", reply, ok)
	reply, ok = m.Event(s, "30066-17.htm", elf, pabris)
	b2Page(t, "page", reply, ok, "30066-17.htm")
}

// The second class transfer: a level 40 Warrior with the three marks becomes
// a Gladiator or Warlord.
func TestB2ElfHumanFighterChange2(t *testing.T) {
	const s = "ElfHumanFighterChange2"
	m := b2LoadMasters(t, s)
	klaus := &scripttest.Npc{ObjectID: 0x60000001, ID: 30187}

	warrior := &scripttest.Player{ObjectID: 0x10000001, Name: "Warrior", Level: 39, ClassID: 1}
	reply, ok := m.Talk(s, warrior, klaus)
	b2Page(t, "warrior list", reply, ok, "30109-02.htm")
	fighter := &scripttest.Player{ObjectID: 0x10000002, Name: "Fighter", Level: 40, ClassID: 0}
	reply, ok = m.Talk(s, fighter, klaus)
	b2Page(t, "first class first", reply, ok, "30109-37.htm")
	mage := &scripttest.Player{ObjectID: 0x10000003, Name: "Mage", Level: 40, ClassID: 11}
	reply, ok = m.Talk(s, mage, klaus)
	b2Page(t, "mismatch", reply, ok, "30109-38.htm")
	duelist := &scripttest.Player{ObjectID: 0x10000004, Name: "Duelist", Level: 76, ClassID: 88}
	reply, ok = m.Talk(s, duelist, klaus)
	b2Page(t, "fourth class", reply, ok, "30109-01.htm")

	reply, ok = m.Event(s, "2", warrior, klaus)
	b2Page(t, "low no marks", reply, ok, "30109-41.htm")
	warrior.Give(b2MarkChallenger, 1)
	warrior.Give(b2MarkTrust, 1)
	warrior.Give(b2MarkDuelist, 1)
	reply, ok = m.Event(s, "2", warrior, klaus)
	b2Page(t, "low with marks", reply, ok, "30109-40.htm")
	warrior.Level = 40
	reply, ok = m.Event(s, "3", warrior, klaus)
	b2Page(t, "warlord without the champion mark", reply, ok, "30109-47.htm")
	m.b2NoClassChange("refusals")
	reply, ok = m.Event(s, "2", warrior, klaus)
	b2Page(t, "gladiator", reply, ok, "30109-42.htm")
	m.b2ClassChanged("gladiator", warrior, 2)
	b2Items(t, "gladiator", warrior, map[int]int64{b2MarkChallenger: 0, b2MarkTrust: 0, b2MarkDuelist: 0, b2CouponC: 15})
	reply, ok = m.Event(s, "3", warrior, klaus)
	b2Page(t, "already third", reply, ok, "30109-39.htm")

	lord := &scripttest.Player{ObjectID: 0x10000005, Name: "Lord", Level: 40, ClassID: 1}
	lord.Give(b2MarkChallenger, 1)
	lord.Give(b2MarkTrust, 1)
	lord.Give(b2MarkChampion, 1)
	reply, ok = m.Event(s, "3", lord, klaus)
	b2Page(t, "warlord", reply, ok, "30109-46.htm")
	m.b2ClassChanged("warlord", lord, 3)
}

// Every first class master: one class change each, the class lists and a
// refusal.
func TestB2FirstClassMasters(t *testing.T) {
	for _, tt := range []struct {
		script     string
		npc        int
		class      int // starting class
		event      string
		proof      int
		newClass   int
		list, done string
		mismatch   int // a class the master turns away
		mismatched string
	}{
		{"ElfHumanWizardChange1", 30037, 10, "11", 1292, 11, "30037-01.htm", "30037-20.htm", 44, "30037-15.htm"},
		{"ElfHumanWizardChange1", 30070, 25, "29", 1235, 29, "30070-08.htm", "30070-32.htm", 0, "30070-15.htm"},
		{"OrcChange1", 30500, 44, "47", 1615, 47, "30500-01.htm", "30500-18.htm", 0, "30500-23.htm"},
		{"OrcChange1", 30505, 49, "50", 1631, 50, "30505-06.htm", "30505-22.htm", 10, "30505-23.htm"},
		{"DwarfBlacksmithChange1", 30499, 53, "56", 1635, 56, "30499-01.htm", "30499-10.htm", 0, "30499-05.htm"},
		{"DwarfWarehouseChange1", 30498, 53, "54", 1642, 54, "30498-01.htm", "30498-10.htm", 56, "30498-05.htm"},
	} {
		t.Run(tt.script+"/"+tt.event, func(t *testing.T) {
			m := b2LoadMasters(t, tt.script)
			npc := &scripttest.Npc{ObjectID: 0x60000001, ID: tt.npc}
			p := &scripttest.Player{ObjectID: 0x10000001, Name: "P", Level: 20, ClassID: tt.class}
			reply, ok := m.Talk(tt.script, p, npc)
			b2Page(t, "list", reply, ok, tt.list)
			other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 20, ClassID: tt.mismatch}
			reply, ok = m.Talk(tt.script, other, npc)
			b2Page(t, "mismatch", reply, ok, tt.mismatched)
			reply, ok = m.Event(tt.script, tt.event, p, npc)
			if !ok || reply.Content == tt.done {
				t.Fatalf("no proof: %+v, %v", reply, ok)
			}
			m.b2NoClassChange("no proof")
			p.Give(tt.proof, 1)
			reply, ok = m.Event(tt.script, tt.event, p, npc)
			b2Page(t, "change", reply, ok, tt.done)
			m.b2ClassChanged("change", p, tt.newClass)
			b2Items(t, "change", p, map[int]int64{tt.proof: 0, b2CouponD: 15})
		})
	}
}

// Every second class master but ElfHumanFighterChange2 (above): one class
// change each with the three marks.
func TestB2SecondClassMasters(t *testing.T) {
	for _, tt := range []struct {
		script   string
		npc      int
		class    int
		event    string
		marks    []int
		newClass int
		list     string
		done     string
	}{
		{"ElfHumanWizardChange2", 30115, 11, "12", []int{2674, 2734, 2840}, 12, "30115-02.htm", "30115-24.htm"},
		{"ElfHumanWizardChange2", 30174, 26, "28", []int{2674, 3140, 3336}, 28, "30115-12.htm", "30115-40.htm"},
		{"ElfHumanClericChange2", 30120, 15, "17", []int{2721, 2734, 2821}, 17, "30120-02.htm", "30120-22.htm"},
		{"ElfHumanClericChange2", 30191, 29, "30", []int{2721, 3140, 2820}, 30, "30120-09.htm", "30120-26.htm"},
		{"OrcChange2", 30513, 45, "46", []int{2627, 3203, 3276}, 46, "30513-02.htm", "30513-22.htm"},
		{"OrcChange2", 30681, 50, "52", []int{2721, 3203, 2879}, 52, "30513-10.htm", "30513-34.htm"},
		{"DwarfBlacksmithChange2", 30512, 56, "57", []int{3119, 3238, 2867}, 57, "30512-02.htm", "30512-11.htm"},
		{"DwarfWarehouseChange2", 30511, 54, "55", []int{3119, 3238, 2809}, 55, "30511-02.htm", "30511-11.htm"},
	} {
		t.Run(tt.script+"/"+tt.event, func(t *testing.T) {
			m := b2LoadMasters(t, tt.script)
			npc := &scripttest.Npc{ObjectID: 0x60000001, ID: tt.npc}
			p := &scripttest.Player{ObjectID: 0x10000001, Name: "P", Level: 40, ClassID: tt.class}
			reply, ok := m.Talk(tt.script, p, npc)
			b2Page(t, "list", reply, ok, tt.list)
			for i, id := range tt.marks {
				reply, ok = m.Event(tt.script, tt.event, p, npc)
				if !ok || reply.Content == tt.done {
					t.Fatalf("with %d marks: %+v, %v", i, reply, ok)
				}
				p.Give(id, 1)
			}
			m.b2NoClassChange("missing marks")
			reply, ok = m.Event(tt.script, tt.event, p, npc)
			b2Page(t, "change", reply, ok, tt.done)
			m.b2ClassChanged("change", p, tt.newClass)
			b2Items(t, "change", p, map[int]int64{tt.marks[0]: 0, tt.marks[1]: 0, tt.marks[2]: 0, b2CouponC: 15})
		})
	}
}

// The Dark Elf masters number their classes by row (0-based) and play the
// fanfare.
func TestB2DarkElfMasters(t *testing.T) {
	m := b2LoadMasters(t, "DarkElfChange1", "DarkElfChange2")
	xenos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30290}
	angus := &scripttest.Npc{ObjectID: 0x60000002, ID: 30474}

	fighter := &scripttest.Player{ObjectID: 0x10000001, Name: "Dark", Level: 19, ClassID: 31}
	reply, ok := m.Talk("DarkElfChange1", fighter, xenos)
	b2Page(t, "fighter list", reply, ok, "30290-01.html")
	human := &scripttest.Player{ObjectID: 0x10000002, Name: "Human", Level: 20, ClassID: 0}
	reply, ok = m.Talk("DarkElfChange1", human, xenos)
	b2Page(t, "other race", reply, ok, "30290-33.html")
	reply, ok = m.Event("DarkElfChange1", "30290-02.html", fighter, xenos)
	b2Page(t, "page", reply, ok, "30290-02.html")

	// Row 1 is the Assassin (Iron Heart).
	reply, ok = m.Event("DarkElfChange1", "1", fighter, xenos)
	b2Page(t, "low no proof", reply, ok, "30290-19.html")
	fighter.Give(1252, 1)
	reply, ok = m.Event("DarkElfChange1", "1", fighter, xenos)
	b2Page(t, "low proof", reply, ok, "30290-20.html")
	fighter.Level = 20
	reply, ok = m.Event("DarkElfChange1", "1", fighter, xenos)
	b2Page(t, "assassin", reply, ok, "30290-22.html")
	if got := m.world.Sounds; len(got) != 1 || got[0] != b2Fanfare {
		t.Errorf("sounds %v, want the fanfare", got)
	}
	m.b2ClassChanged("assassin", fighter, 35)
	b2Items(t, "assassin", fighter, map[int]int64{1252: 0, b2CouponD: 15})
	reply, ok = m.Talk("DarkElfChange1", fighter, xenos)
	b2Page(t, "first class made", reply, ok, "30290-32.html")
	// A row Mobius does not have answers nothing.
	reply, ok = m.Event("DarkElfChange1", "9", fighter, xenos)
	b2NoReply(t, "row 9", reply, ok)

	// Row 4 of DarkElfChange2 is the Phantom Ranger.
	reply, ok = m.Talk("DarkElfChange2", fighter, angus)
	b2Page(t, "assassin list", reply, ok, "30474-12.html")
	fighter.Level = 40
	reply, ok = m.Event("DarkElfChange2", "4", fighter, angus)
	b2Page(t, "no marks", reply, ok, "30474-44.html")
	fighter.Give(2673, 1)
	fighter.Give(3172, 1)
	fighter.Give(3293, 1)
	reply, ok = m.Event("DarkElfChange2", "4", fighter, angus)
	b2Page(t, "phantom ranger", reply, ok, "30474-45.html")
	m.b2ClassChanged("phantom ranger", fighter, 37)
	b2Items(t, "phantom ranger", fighter, map[int]int64{2673: 0, 3172: 0, 3293: 0, b2CouponC: 15})
	reply, ok = m.Talk("DarkElfChange2", fighter, angus)
	b2Page(t, "second class made", reply, ok, "30474-54.html")
}

// The newbie village headmasters only talk about the first class change.
func TestB2FirstClassTransferTalk(t *testing.T) {
	const s = "FirstClassTransferTalk"
	m := b2LoadMasters(t, s)
	blitz := &scripttest.Npc{ObjectID: 0x60000001, ID: 30026, Types: []string{"VillageMasterFighter"}}
	biotin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30031, Types: []string{"VillageMasterPriest"}}
	kakai := &scripttest.Npc{ObjectID: 0x60000003, ID: 30565}

	for _, tt := range []struct {
		name  string
		class int
		npc   *scripttest.Npc
		page  string
	}{
		{"human fighter at Blitz", 0, blitz, "30026_fighter.html"},
		{"human mage at Blitz", 10, blitz, "30026_no.html"},
		{"human mage at Biotin", 10, biotin, "30031_mystic.html"},
		{"human fighter at Biotin", 0, biotin, "30031_no.html"},
		{"warrior at Blitz", 1, blitz, "30026_transfer_1.html"},
		{"gladiator at Blitz", 2, blitz, "30026_transfer_2.html"},
		{"orc mage at Kakai", 49, kakai, "30565_mystic.html"},
		{"orc fighter at Kakai", 44, kakai, "30565_fighter.html"},
		{"elf at Kakai", 18, kakai, "30565_no.html"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000001, Name: "P", Level: 10, ClassID: tt.class}
		reply, ok := m.Talk(s, p, tt.npc)
		b2Page(t, tt.name, reply, ok, tt.page)
	}
}

// The clan and alliance pages: nobody has a clan before M9.
func TestB2ClanAndAllianceMasters(t *testing.T) {
	m := b2LoadMasters(t, "ClanMaster", "AllianceMaster")
	ramos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30373}
	p := &scripttest.Player{ObjectID: 0x10000001, Name: "P", Level: 20}

	reply, ok := m.Talk("ClanMaster", p, ramos)
	b2Page(t, "clan", reply, ok, "9000-01.htm")
	reply, ok = m.Event("ClanMaster", "9000-02.htm", p, ramos)
	b2Page(t, "create", reply, ok, "9000-02.htm")
	reply, ok = m.Event("ClanMaster", "9000-12a.htm", p, ramos)
	b2Page(t, "leader page", reply, ok, "9000-07-no.htm")
	reply, ok = m.Event("ClanMaster", "9000-03.htm", p, ramos)
	b2Page(t, "leader page 3", reply, ok, "9000-03-no.htm")

	reply, ok = m.Talk("AllianceMaster", p, ramos)
	b2Page(t, "alliance", reply, ok, "9001-01.htm")
	reply, ok = m.Event("AllianceMaster", "9001-01.htm", p, ramos)
	b2Page(t, "alliance first", reply, ok, "9001-01.htm")
	reply, ok = m.Event("AllianceMaster", "9001-02.htm", p, ramos)
	b2Page(t, "no clan", reply, ok, "9001-04.htm")
}
