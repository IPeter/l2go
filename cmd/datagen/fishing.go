package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The fishing sources (Plan 14, M11g): the three XML files Mobius FishData,
// FishingMonstersData and FishingRodsData read, and the Java code that holds
// the lure rules.
const (
	fishesXML          = "dist/game/data/stats/fishing/fishes.xml"
	fishingMonstersXML = "dist/game/data/stats/fishing/fishingMonsters.xml"
	fishingRodsXML     = "dist/game/data/stats/fishing/fishingRods.xml"
	fishingPlayerJava  = "java/org/l2jmobius/gameserver/entity/actor/Player.java"
	fishingItemJava    = "java/org/l2jmobius/gameserver/entity/item/instance/Item.java"
)

// The fishing domain converts the fish, the fishing monsters, the rods and
// the lure table, loaded by data.LoadFishing.
func init() { registerDomain("fishing", genFishing) }

// genFishing reads the fishing data, checks it with data.NewFishing and
// against the items, and adds the four files of data/fishing/.
func genFishing(o options) error {
	var (
		d   data.FishingData
		err error
	)
	if d.Fishes, err = readFishes(o.src); err != nil {
		return fmt.Errorf("fishes: %w", err)
	}
	if d.Monsters, err = readFishingMonsters(o.src); err != nil {
		return fmt.Errorf("fishing monsters: %w", err)
	}
	if d.Rods, err = readFishingRods(o.src); err != nil {
		return fmt.Errorf("fishing rods: %w", err)
	}
	if err := checkLureSources(o.src); err != nil {
		return fmt.Errorf("lures: %w", err)
	}
	d.Lures = fishingLures()
	if _, err := data.NewFishing(d); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.FishesFile, d.Fishes); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.FishingMonstersFile, d.Monsters); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.FishingRodsFile, d.Rods); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.FishingLuresFile, d.Lures); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "fishing: %d fishes, %d fishing monsters, %d rods, %d lures\n",
		len(d.Fishes), len(d.Monsters), len(d.Rods), len(d.Lures))
	return nil
}

// fishGroups and fishGrades are Mobius Fish.getGroupId and getGradeId. The
// Java switches fall back to wide and fish_normal; a name they do not list
// fails here instead, so a datapack change is noticed.
var (
	fishGroups = map[string]int{
		"wide": data.FishGroupWide, "swift": data.FishGroupSwift, "ugly": data.FishGroupUgly,
		"fish_box": data.FishGroupBox, "easy_wide": data.FishGroupEasyWide, "easy_swift": data.FishGroupEasySwift,
		"easy_ugly": data.FishGroupEasyUgly, "hard_wide": data.FishGroupHardWide, "hard_swift": data.FishGroupHardSwift,
		"hard_ugly": data.FishGroupHardUgly, "hs_fish": data.FishGroupHS,
	}
	fishGrades = map[string]int{"fish_easy": data.FishGradeEasy, "fish_normal": data.FishGradeNormal, "fish_hard": data.FishGradeHard}
)

// readFishes reads fishes.xml in file order.
func readFishes(fsys fs.FS) ([]data.Fish, error) {
	root, err := fishingList(fsys, fishesXML, "fish")
	if err != nil {
		return nil, err
	}
	var out []data.Fish
	for _, n := range root.children {
		if err := n.checkAttrs("fishId", "itemId", "itemName", "fishGroup", "fishLevel", "fishBiteRate", "fishGuts",
			"fishHp", "fishMaxLength", "fishLengthRate", "hpRegen", "startCombatTime", "combatDuration", "gutsCheckTime",
			"gutsCheckProbability", "cheatingProb", "fishGrade"); err != nil {
			return nil, err
		}
		var f data.Fish
		var errs []error
		i := func(name string, dst *int) {
			v, err := n.intAttr(name)
			errs = append(errs, err)
			*dst = v
		}
		x := func(name string, dst *float64) {
			v, err := n.floatAttr(name)
			errs = append(errs, err)
			*dst = v
		}
		i("fishId", &f.FishID)
		i("itemId", &f.ItemID)
		i("fishLevel", &f.Level)
		x("fishBiteRate", &f.BiteRate)
		x("fishGuts", &f.Guts)
		i("fishHp", &f.HP)
		i("fishMaxLength", &f.MaxLength)
		x("fishLengthRate", &f.LengthRate)
		x("hpRegen", &f.HPRegen)
		i("startCombatTime", &f.StartCombatTime)
		i("combatDuration", &f.CombatDuration)
		i("gutsCheckTime", &f.GutsCheckTime)
		x("gutsCheckProbability", &f.GutsCheckProbability)
		x("cheatingProb", &f.CheatingProb)
		f.ItemName, _ = n.attr("itemName")
		group, _ := n.attr("fishGroup")
		grade, _ := n.attr("fishGrade")
		var ok bool
		if f.Group, ok = fishGroups[group]; !ok {
			errs = append(errs, n.errorf("unknown fishGroup %q", group))
		}
		if f.Grade, ok = fishGrades[grade]; !ok {
			errs = append(errs, n.errorf("unknown fishGrade %q", grade))
		}
		for _, err := range errs {
			if err != nil {
				return nil, err
			}
		}
		out = append(out, f)
	}
	return out, nil
}

// readFishingMonsters reads fishingMonsters.xml in file order.
func readFishingMonsters(fsys fs.FS) ([]data.FishingMonster, error) {
	root, err := fishingList(fsys, fishingMonstersXML, "fishingMonster")
	if err != nil {
		return nil, err
	}
	var out []data.FishingMonster
	for _, n := range root.children {
		if err := n.checkAttrs("userMinLevel", "userMaxLevel", "fishingMonsterId", "probability"); err != nil {
			return nil, err
		}
		var m data.FishingMonster
		for _, a := range []struct {
			name string
			dst  *int
		}{{"userMinLevel", &m.MinLevel}, {"userMaxLevel", &m.MaxLevel}, {"fishingMonsterId", &m.NpcID}, {"probability", &m.Probability}} {
			v, err := n.intAttr(a.name)
			if err != nil {
				return nil, err
			}
			*a.dst = v
		}
		out = append(out, m)
	}
	return out, nil
}

// readFishingRods reads fishingRods.xml, sorted by rod ID.
func readFishingRods(fsys fs.FS) ([]data.FishingRod, error) {
	root, err := fishingList(fsys, fishingRodsXML, "fishingRod")
	if err != nil {
		return nil, err
	}
	var out []data.FishingRod
	for _, n := range root.children {
		if err := n.checkAttrs("fishingRodId", "fishingRodItemId", "fishingRodLevel", "fishingRodName", "fishingRodDamage"); err != nil {
			return nil, err
		}
		var r data.FishingRod
		for _, a := range []struct {
			name string
			dst  *int
		}{{"fishingRodId", &r.ID}, {"fishingRodItemId", &r.ItemID}, {"fishingRodLevel", &r.Level}} {
			v, err := n.intAttr(a.name)
			if err != nil {
				return nil, err
			}
			*a.dst = v
		}
		if r.Damage, err = n.floatAttr("fishingRodDamage"); err != nil {
			return nil, err
		}
		r.Name, _ = n.attr("fishingRodName")
		out = append(out, r)
	}
	slices.SortStableFunc(out, func(a, b data.FishingRod) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}

// fishingList reads a fishing XML file whose <list> holds only elem.
func fishingList(fsys fs.FS, file, elem string) (*xmlNode, error) {
	root, err := readXMLTree(fsys, file)
	if err != nil {
		return nil, err
	}
	if root.name != "list" {
		return nil, root.errorf("want <list>")
	}
	if err := root.checkChildren(elem); err != nil {
		return nil, err
	}
	if len(root.children) == 0 {
		return nil, root.errorf("no <%s>", elem)
	}
	for _, c := range root.children {
		if len(c.children) > 0 {
			return nil, c.errorf("want only attributes")
		}
	}
	return root, nil
}

// The lure table (decision X6). Mobius keeps it in Java: the grade in
// Player.getRandomFishGrade, the group switch in Player.getRandomFishGroup
// (check = Rnd.get(100), the first branch with check <= its bound wins), the
// check delay factor in Player.startLookingForFishTask (the second branch's
// range 8505-8513 comes before the third, so 8507, 8510 and 8513 get 100,
// not 66) and night lures in Item.isNightLure (8505-8513 and 8485).
// checkLureSources fails when that Java code changes.
func fishingLures() []data.Lure {
	g := func(pairs ...int) []data.LureGroup {
		var out []data.LureGroup
		for i := 0; i < len(pairs); i += 2 {
			out = append(out, data.LureGroup{UpTo: pairs[i], Group: pairs[i+1]})
		}
		return out
	}
	var (
		normalGreen  = g(54, data.FishGroupSwift, 74, data.FishGroupWide, 94, data.FishGroupUgly, 99, data.FishGroupBox)
		normalPurple = g(54, data.FishGroupWide, 74, data.FishGroupSwift, 94, data.FishGroupUgly, 99, data.FishGroupBox)
		normalYellow = g(55, data.FishGroupUgly, 74, data.FishGroupSwift, 94, data.FishGroupWide, 99, data.FishGroupBox)
		box          = g(99, data.FishGroupBox)
	)
	lure := func(id, grade, delay int, groups []data.LureGroup) data.Lure {
		return data.Lure{ItemID: id, Grade: grade, Night: (id >= 8505 && id <= 8513) || id == 8485, CheckDelay: delay, Groups: groups}
	}
	n, e, h := data.FishGradeNormal, data.FishGradeEasy, data.FishGradeHard
	lures := []data.Lure{
		lure(6519, n, 133, normalGreen),
		lure(6520, n, 100, normalGreen),
		lure(6521, n, 66, normalGreen),
		lure(6522, n, 133, normalPurple),
		lure(6523, n, 100, normalPurple),
		lure(6524, n, 66, normalPurple),
		lure(6525, n, 133, normalYellow),
		lure(6526, n, 100, normalYellow),
		lure(6527, n, 66, normalYellow),
		lure(7610, n, 100, box),
		lure(7611, n, 100, box),
		lure(7612, n, 100, box),
		lure(7613, n, 100, box),
		lure(7807, e, 100, g(54, data.FishGroupEasySwift, 77, data.FishGroupEasyWide, 99, data.FishGroupEasyUgly)),
		lure(7808, e, 100, g(54, data.FishGroupEasyWide, 77, data.FishGroupEasyUgly, 99, data.FishGroupEasySwift)),
		lure(7809, e, 100, g(54, data.FishGroupEasyUgly, 77, data.FishGroupEasySwift, 99, data.FishGroupEasyWide)),
		lure(8484, n, 100, g(33, data.FishGroupWide, 66, data.FishGroupSwift, 99, data.FishGroupUgly)),
		lure(8485, h, 100, g(33, data.FishGroupHardWide, 66, data.FishGroupHardSwift, 99, data.FishGroupHardUgly)),
		lure(8486, e, 100, g(33, data.FishGroupEasyWide, 66, data.FishGroupEasySwift, 99, data.FishGroupEasyUgly)),
		lure(8505, n, 133, normalGreen),
		lure(8506, h, 100, g(54, data.FishGroupHardSwift, 77, data.FishGroupHardWide, 99, data.FishGroupHardUgly)),
		lure(8507, n, 100, normalGreen),
		lure(8508, n, 133, normalPurple),
		lure(8509, h, 100, g(54, data.FishGroupHardWide, 77, data.FishGroupHardUgly, 99, data.FishGroupHardSwift)),
		lure(8510, n, 100, normalPurple),
		lure(8511, n, 133, normalYellow),
		lure(8512, h, 100, g(54, data.FishGroupHardUgly, 77, data.FishGroupHardSwift, 99, data.FishGroupHardWide)),
		lure(8513, n, 100, normalYellow),
	}
	return lures
}

// lureSources are the Java lines the lure table is copied from; each must
// still be there, whitespace aside.
var lureSources = map[string][]string{
	fishingPlayerJava: {
		"case 7807: case 7808: case 7809: case 8486: { return 0; }",
		"case 8485: case 8506: case 8509: case 8512: { return 2; } default: { return 1; }",
		"case 7610: case 7611: case 7612: case 7613: { type = 3; break; }",
		"case 6519: case 8505: case 6520: case 6521: case 8507: { if (check <= 54) { type = 1; } else if (check <= 74) { type = 0; } else if (check <= 94) { type = 2; } else { type = 3; }",
		"case 6522: case 8508: case 6523: case 6524: case 8510: { if (check <= 54) { type = 0; } else if (check <= 74) { type = 1; } else if (check <= 94) { type = 2; } else { type = 3; }",
		"case 6525: case 8511: case 6526: case 6527: case 8513: { if (check <= 55) { type = 2; } else if (check <= 74) { type = 1; } else if (check <= 94) { type = 0; } else { type = 3; }",
		"case 8484: { if (check <= 33) { type = 0; } else if (check <= 66) { type = 1; } else { type = 2; }",
		"case 7807: { if (check <= 54) { type = 5; } else if (check <= 77) { type = 4; } else { type = 6; }",
		"case 7808: { if (check <= 54) { type = 4; } else if (check <= 77) { type = 6; } else { type = 5; }",
		"case 7809: { if (check <= 54) { type = 6; } else if (check <= 77) { type = 5; } else { type = 4; }",
		"case 8486: { if (check <= 33) { type = 4; } else if (check <= 66) { type = 5; } else { type = 6; }",
		"case 8506: { if (check <= 54) { type = 8; } else if (check <= 77) { type = 7; } else { type = 9; }",
		"case 8509: { if (check <= 54) { type = 7; } else if (check <= 77) { type = 9; } else { type = 8; }",
		"case 8512: { if (check <= 54) { type = 9; } else if (check <= 77) { type = 8; } else { type = 7; }",
		"case 8485: { if (check <= 33) { type = 7; } else if (check <= 66) { type = 8; } else { type = 9; }",
		"if ((lureid == 6519) || (lureid == 6522) || (lureid == 6525) || (lureid == 8505) || (lureid == 8508) || (lureid == 8511)) { checkDelay = _fish.getGutsCheckTime() * 133; }",
		"else if ((lureid == 6520) || (lureid == 6523) || (lureid == 6526) || ((lureid >= 8505) && (lureid <= 8513)) || ((lureid >= 7610) && (lureid <= 7613)) || ((lureid >= 7807) && (lureid <= 7809)) || ((lureid >= 8484) && (lureid <= 8486))) { checkDelay = _fish.getGutsCheckTime() * 100; }",
		"else if ((lureid == 6521) || (lureid == 6524) || (lureid == 6527) || (lureid == 8507) || (lureid == 8510) || (lureid == 8513)) { checkDelay = _fish.getGutsCheckTime() * 66; }",
	},
	fishingItemJava: {
		"return (((_itemId >= 8505) && (_itemId <= 8513)) || (_itemId == 8485));",
	},
}

// checkLureSources checks that the Java code of the lure table is unchanged.
func checkLureSources(fsys fs.FS) error {
	for _, file := range []string{fishingPlayerJava, fishingItemJava} {
		b, err := fs.ReadFile(fsys, file)
		if err != nil {
			return err
		}
		src := strings.Join(strings.Fields(stripJavaComments(string(b))), " ")
		for _, want := range lureSources[file] {
			if !strings.Contains(src, want) {
				return fmt.Errorf("%s: the lure code changed; update the lure table (missing %q)", file, want)
			}
		}
	}
	return nil
}
