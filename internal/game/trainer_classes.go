package game

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// The classes each skill trainer teaches: Mobius dist/game/data/SkillLearn.xml
// (SkillLearnData, which NpcData copies into the templates as their teach
// info). The data generator does not export the file yet, so the table is
// written out here, in the file's order; TestTrainerClassesMatchMobius
// compares it with the XML when the Mobius clone is there.

// trainerClassList builds a class list from class IDs.
func trainerClassList(ids ...int) []data.PlayerClass {
	out := make([]data.PlayerClass, len(ids))
	for i, id := range ids {
		out[i] = data.PlayerClass(id) //nolint:gosec // class IDs are small
	}
	return out
}

// trainerJoin joins class lists into a new one.
func trainerJoin(lists ...[]data.PlayerClass) []data.PlayerClass {
	return slices.Concat(lists...)
}

// The class groups SkillLearn.xml repeats.
var (
	tcHumanFighters  = trainerClassList(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	tcHumanWizards   = trainerClassList(10, 11, 12, 13, 14)
	tcHumanClerics   = trainerClassList(10, 15, 16, 17)
	tcElfFighters    = trainerClassList(18, 19, 20, 21, 22, 23, 24)
	tcElfWizards     = trainerClassList(25, 26, 27, 28)
	tcElfOracles     = trainerClassList(25, 29, 30)
	tcDarkFighters   = trainerClassList(31, 32, 33, 34, 35, 36, 37)
	tcDarkMystics    = trainerClassList(38, 39, 40, 41, 42, 43)
	tcOrcFighters    = trainerClassList(44, 45, 46, 47, 48)
	tcOrcMystics     = trainerClassList(49, 50, 51, 52)
	tcDwarfScavenger = trainerClassList(53, 54, 55)
	tcDwarfArtisan   = trainerClassList(53, 56, 57)
	tcDwarvesAll     = trainerClassList(53, 54, 55, 56, 57, 117, 118)

	tcHumanFighters3 = trainerJoin(tcHumanFighters, trainerClassList(88, 89, 90, 91, 92, 93))
	tcHumanMystics3  = trainerClassList(10, 11, 12, 13, 14, 15, 16, 17, 94, 95, 96, 97, 98)
	tcElfFighters3   = trainerJoin(tcElfFighters, trainerClassList(99, 100, 101, 102))
	tcElfMystics3    = trainerClassList(25, 26, 27, 28, 29, 30, 103, 104, 105)
	tcDarkFighters3  = trainerJoin(tcDarkFighters, trainerClassList(106, 107, 108, 109))
	tcDarkMystics3   = trainerJoin(tcDarkMystics, trainerClassList(110, 111, 112))
	tcOrcFighters3   = trainerJoin(tcOrcFighters, trainerClassList(113, 114))
	tcOrcMystics3    = trainerJoin(tcOrcMystics, trainerClassList(115, 116))
)

// trainerEntry is one <npc> of SkillLearn.xml.
type trainerEntry struct {
	npc     int
	name    string
	classes []data.PlayerClass
}

// trainerTable is SkillLearn.xml in file order.
var trainerTable = []trainerEntry{
	{30010, "Auron", tcHumanFighters},
	{30014, "Celma", tcHumanFighters},
	{30022, "Zigaunt", tcHumanClerics},
	{30027, "Gwinter", tcHumanFighters},
	{30028, "Pintage", tcHumanFighters},
	{30029, "Minia", tcHumanFighters},
	{30030, "Vivyan", tcHumanClerics},
	{30032, "Yohanes", tcHumanClerics},
	{30033, "Baulro", tcHumanWizards},
	{30034, "Iris", tcHumanWizards},
	{30035, "Harrys", tcHumanWizards},
	{30036, "Petron", tcHumanClerics},
	{30057, "Aldo", tcDwarfScavenger},
	{30058, "Holvas", tcDwarfScavenger},
	{30064, "Terry", tcHumanFighters},
	{30065, "Arnelle", tcHumanFighters},
	{30067, "Glyvka", tcHumanClerics},
	{30068, "Shegfield", tcElfOracles},
	{30069, "Rollant", tcHumanWizards},
	{30103, "Valkon", tcDwarfScavenger},
	{30104, "Parman", tcDwarfScavenger},
	{30105, "Genwitter", tcHumanFighters},
	{30106, "Dufner", tcHumanFighters},
	{30107, "Goldian", tcElfFighters},
	{30108, "Macken", tcHumanFighters},
	{30110, "Iker", tcHumanWizards},
	{30111, "Dieter", tcHumanWizards},
	{30112, "Maurius", tcElfWizards},
	{30113, "Juris", tcElfWizards},
	{30114, "Roa", tcHumanWizards},
	{30116, "Dustin", tcHumanClerics},
	{30117, "Primos", tcHumanClerics},
	{30118, "Pupina", tcHumanClerics},
	{30119, "Isabellin", tcElfOracles},
	{30143, "Trudy", tcDarkFighters},
	{30144, "Harne", tcDarkMystics},
	{30145, "Vlasty", tcDarkMystics},
	{30155, "Ellenia", tcElfFighters},
	{30156, "Cobendell", tcElfFighters},
	{30157, "Greenis", tcElfWizards},
	{30158, "Esrandell", tcElfOracles},
	{30169, "Marty", trainerClassList(54, 55, 117)},
	{30170, "Radic", trainerClassList(54, 55, 117)},
	{30171, "Gaius", tcHumanWizards},
	{30182, "Cliff", tcDwarfScavenger},
	{30183, "Hagger", tcDwarfScavenger},
	{30184, "Rigol", tcHumanFighters},
	{30185, "Taniac Blackbird", tcHumanFighters},
	{30186, "Bhan", tcElfFighters},
	{30188, "Vadin", tcHumanClerics},
	{30189, "Rovia", tcElfWizards},
	{30190, "Phanovia", tcHumanWizards},
	{30192, "Darya", tcDarkFighters},
	{30194, "Gideon", tcDarkMystics},
	{30210, "Norman", tcDwarfScavenger},
	{30232, "Sorint", tcDwarfScavenger},
	{30255, "Haprock", tcDwarfScavenger},
	{30293, "Manuel", tcElfOracles},
	{30298, "Pinter", tcDwarfArtisan},
	{30300, "Pushkin", tcDwarfArtisan},
	{30316, "Raut", tcDwarfScavenger},
	{30317, "Kluto", tcDwarfArtisan},
	{30320, "Richlin", tcDwarfScavenger},
	{30322, "Ballin", tcDwarfScavenger},
	{30325, "Audiberti", tcHumanFighters},
	{30326, "Leona", tcHumanFighters},
	{30327, "Sorius", tcElfFighters},
	{30328, "Reisa", tcElfFighters},
	{30329, "Virgil", tcDarkFighters},
	{30330, "Sidra", tcDarkMystics},
	{30344, "Rohmer", tcHumanWizards},
	{30345, "Ramoniell", tcElfWizards},
	{30360, "Harant", tcDarkFighters},
	{30369, "Baenedes", tcElfFighters},
	{30374, "Rhodiell", tcElfFighters},
	{30375, "Adonius", tcElfOracles},
	{30376, "Nell", tcElfWizards},
	{30377, "Talbot", tcDarkMystics},
	{30378, "Estella", tcDarkFighters},
	{30458, "Poitan", tcDwarfArtisan},
	{30459, "Wandius", tcElfFighters},
	{30460, "Cardien", tcElfFighters},
	{30461, "Mirien", tcElfWizards},
	{30463, "Ixia", tcDarkFighters},
	{30464, "Clayton", tcDarkMystics},
	{30471, "Rupio", tcDwarfArtisan},
	{30472, "Rosheria", tcElfFighters},
	{30473, "Bandellos", tcElfOracles},
	{30475, "Stapin", tcDarkFighters},
	{30476, "Kaira", tcDarkMystics},
	{30501, "Kasman", tcOrcFighters},
	{30502, "Umos", tcOrcMystics},
	{30506, "Buka", tcOrcFighters},
	{30507, "Racoy", tcOrcMystics},
	{30509, "Dowki", tcOrcFighters},
	{30510, "Somak", tcOrcMystics},
	{30514, "Vokian", tcOrcFighters},
	{30515, "Manakia", tcOrcMystics},
	{30516, "Reep", trainerClassList(53, 54, 55, 56, 57)},
	{30520, "Reed", tcDwarfScavenger},
	{30521, "Murdoc", tcDwarfScavenger},
	{30522, "Airy", tcDwarfScavenger},
	{30526, "Brunon", tcDwarfArtisan},
	{30527, "Silvera", tcDwarfArtisan},
	{30569, "Brukurse", tcOrcFighters},
	{30570, "Karukia", tcOrcFighters},
	{30571, "Tanapi", tcOrcMystics},
	{30572, "Livina", tcOrcMystics},
	{30678, "Helton", tcDwarfArtisan},
	{30679, "Roameria", tcElfFighters},
	{30680, "Egnos", tcElfOracles},
	{30682, "Pekiron", tcOrcMystics},
	{30683, "Daunt", tcOrcFighters},
	{30686, "Silva", tcDwarfScavenger},
	{30688, "Duning", tcDwarfArtisan},
	{30690, "Luther", tcHumanFighters},
	{30691, "Aren Athebaldt", tcHumanFighters},
	{30692, "Stedmiel", tcElfFighters},
	{30693, "Queenien", tcElfFighters},
	{30695, "Moses", tcHumanWizards},
	{30696, "Page", tcHumanWizards},
	{30697, "Videlrien", tcElfWizards},
	{30698, "Evelyn", tcElfOracles},
	{30700, "Prestan", tcDarkFighters},
	{30701, "Errickin", tcDarkMystics},
	{30705, "Chakiris", tcOrcFighters},
	{30706, "Lazenby", tcOrcMystics},
	{30715, "Marina", tcHumanWizards},
	{30717, "Gauen", tcElfWizards},
	{30718, "Joan", tcElfWizards},
	{30720, "Kaiena", tcDarkMystics},
	{30721, "Ladd", tcDarkMystics},
	{30833, "Kaspar", tcDarkMystics3},
	{30835, "Icarus", trainerClassList(11, 12, 13, 14, 15, 16, 17, 94, 95, 96, 97, 98)},
	{30843, "Romp", tcDwarfScavenger},
	{30844, "Walderal", tcDwarfScavenger},
	{30846, "Wilbert", tcDwarfArtisan},
	{30849, "Sedrick", tcHumanFighters},
	{30850, "Aiken", tcHumanFighters},
	{30851, "Kendra", tcHumanFighters},
	{30852, "Sinden", tcElfFighters},
	{30853, "Raien", tcElfFighters},
	{30855, "Desmond", tcHumanWizards},
	{30856, "Winonin", tcElfWizards},
	{30858, "Ross", tcHumanClerics},
	{30859, "Vivyan", tcHumanClerics},
	{30860, "Flownia", tcElfOracles},
	{30861, "Leonard", tcElfOracles},
	{30863, "Ghest", tcDarkFighters},
	{30864, "Hanellin", tcDarkMystics},
	{30866, "Marestella", tcOrcFighters},
	{30867, "Reva", tcOrcMystics},
	{30895, "Kluck", tcDwarfScavenger},
	{30896, "Mia", tcDwarfScavenger},
	{30898, "Morning", tcDwarfArtisan},
	{30900, "Marcus", tcHumanFighters},
	{30901, "Arti", tcHumanFighters},
	{30902, "Karuna", tcHumanFighters},
	{30903, "Traus", tcElfFighters},
	{30904, "Naiel", tcElfFighters},
	{30906, "Ranton", tcHumanClerics},
	{30907, "Minevia", tcHumanWizards},
	{30908, "Tanios", tcElfOracles},
	{30909, "Anabel", tcElfWizards},
	{30911, "Brikus", tcDarkFighters},
	{30912, "Xenovia", tcDarkMystics},
	{30914, "Sorbo", tcOrcFighters},
	{30915, "Takina", tcOrcMystics},
	{31270, "Stefano", tcDwarvesAll},
	{31268, "Hakon", tcDwarvesAll},
	{31267, "Lietta", tcDwarvesAll},
	{31583, "Feynn", tcDwarvesAll},
	{31290, "Skahi", tcOrcMystics3},
	{31289, "Lakan", tcOrcFighters3},
	{31287, "Kamilen", tcDarkMystics3},
	{31286, "Drakon", tcDarkFighters3},
	{31283, "Alminas", tcElfMystics3},
	{31282, "Justin", tcHumanMystics3},
	{31281, "Cerenas", trainerJoin(tcElfMystics3, tcHumanMystics3)},
	{31280, "Bastian", trainerJoin(tcElfMystics3, tcHumanMystics3)},
	{31278, "Bronwyn", tcElfFighters3},
	{31277, "Felix", tcHumanFighters3},
	{31271, "Hilda", tcDwarvesAll},
	{31315, "Daisy", tcDwarvesAll},
	{31313, "Lunin", tcDwarvesAll},
	{31312, "Durin", tcDwarvesAll},
	{31311, "Hugin", tcDwarvesAll},
	{31582, "Mordred", tcDarkFighters3},
	{31581, "Anastia", tcDarkMystics3},
	{31580, "Galadrid", tcDarkFighters3},
	{31337, "Mekara", tcOrcMystics3},
	{31335, "Kayan", tcDarkMystics3},
	{31333, "Rumiel", tcElfMystics3},
	{31332, "Amelia", tcHumanMystics3},
	{31330, "Wagner", tcElfMystics3},
	{31329, "Eliyah", tcHumanMystics3},
	{31327, "Tazki", tcOrcFighters3},
	{31325, "Themis", tcDarkFighters3},
	{31323, "Beryl", tcElfFighters3},
	{31322, "Erian", tcHumanFighters3},
	{31316, "Vincenz", tcDwarvesAll},
	{31956, "Rydel", trainerClassList(53, 54, 55, 117)},
	{31957, "Cherbal", trainerClassList(53, 54, 55, 117)},
	{31959, "Axel", trainerClassList(53, 54, 55, 117)},
	{31960, "Buryun", trainerClassList(53, 56, 57, 118)},
	{31966, "Karamon", trainerJoin(tcHumanFighters, trainerClassList(89, 90, 91, 92, 93))}, // no Duelist
	{31967, "Euline", tcElfFighters3},
	{31969, "Evelyn", tcHumanMystics3},
	{31970, "Sinis", trainerClassList(25, 26, 27, 28, 29, 30)},
	{31971, "Otillo", trainerClassList(25, 26, 27, 28, 29, 103, 104, 105)}, // no Elven Elder
	{31972, "Atraxia", tcElfMystics3},
	{31975, "Aerina", tcDarkFighters3},
	{31976, "Talia", tcDarkMystics3},
	{31978, "Tazeer", tcOrcFighters3},
	{31979, "Moira", tcOrcMystics3},
	{31990, "Siger", trainerClassList(53, 56, 118)},
	{32098, "Celes", trainerClassList(10, 11, 12, 13, 14, 15, 16, 17, 94, 95, 96, 97)}, // no Hierophant
}

// trainerClasses are the classes each NPC teaches, by NPC ID.
var trainerClasses = func() map[int][]data.PlayerClass {
	m := make(map[int][]data.PlayerClass, len(trainerTable))
	for _, t := range trainerTable {
		m[t.npc] = t.classes
	}
	return m
}()

// trainerCanTeach is Mobius NpcTemplate.canTeach: whether NPC npcID teaches
// class; a third class is taught where its parent class is.
func trainerCanTeach(npcID int, class data.PlayerClass) bool {
	if class.Level() == 3 {
		parent, ok := class.Parent()
		if !ok {
			return false
		}
		class = parent
	}
	return slices.Contains(trainerClasses[npcID], class)
}
