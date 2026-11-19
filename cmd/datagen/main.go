// Command datagen converts the parts of the L2J Mobius Interlude datapack
// that L2Go needs into the JSON files of the data/ folder. The JSON schema
// is defined by internal/data.
//
// Usage:
//
//	go run ./cmd/datagen -mobius L2J_Mobius/L2J_Mobius_CT_0_Interlude -out data
//
// It reads:
//
//   - dist/game/data/stats/players/templates/StartingClass/*.xml: base
//     stats, creation points and level 1 HP/MP/CP of the starting classes;
//   - dist/game/data/stats/players/classList.xml: class names;
//   - dist/game/data/stats/players/initialEquipment.xml;
//   - dist/game/data/stats/items/*.xml (not custom/): the items used by the
//     initial equipment;
//   - java/.../enums/player/PlayerClass.java and .../creature/Race.java:
//     the race of each class and the race IDs, which Mobius keeps in code,
//     not in its XML data.
//
// The output is sorted by ID and does not depend on file order or time, so
// running it again on the same input gives the same files.
package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Paths inside the Mobius Interlude project folder.
const (
	statsDir         = "dist/game/data/stats"
	classListFile    = statsDir + "/players/classList.xml"
	startingClassDir = statsDir + "/players/templates/StartingClass"
	equipmentFile    = statsDir + "/players/initialEquipment.xml"
	itemsDir         = statsDir + "/items"
	playerClassFile  = "java/org/l2jmobius/gameserver/entity/actor/enums/player/PlayerClass.java"
	raceFile         = "java/org/l2jmobius/gameserver/entity/actor/enums/creature/Race.java"
)

func main() {
	mobius := flag.String("mobius", "L2J_Mobius/L2J_Mobius_CT_0_Interlude", "L2J Mobius Interlude project folder")
	out := flag.String("out", "data", "output data folder")
	flag.Parse()
	if err := run(*mobius, *out, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "datagen:", err)
		os.Exit(1)
	}
}

// run generates the data files from mobiusDir into outDir and reports each
// file it writes to log.
func run(mobiusDir, outDir string, log io.Writer) error {
	files, err := generate(os.DirFS(mobiusDir))
	if err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		p := filepath.Join(outDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { //nolint:gosec // data folder is meant to be readable
			return err
		}
		if err := os.WriteFile(p, files[name], 0o644); err != nil { //nolint:gosec // generated data files are public
			return err
		}
		fmt.Fprintf(log, "wrote %s\n", p)
	}
	return nil
}

// generate reads the Mobius project in fsys and returns the output files,
// keyed by their path in the data folder. The data is checked with
// data.New before it is encoded.
func generate(fsys fs.FS) (map[string][]byte, error) {
	classes, err := readClassList(fsys)
	if err != nil {
		return nil, err
	}
	races, err := readRaces(fsys)
	if err != nil {
		return nil, err
	}
	templates, err := readTemplates(fsys, classes, races)
	if err != nil {
		return nil, err
	}
	equipment, err := readEquipment(fsys)
	if err != nil {
		return nil, err
	}
	items, err := readItems(fsys, equipment)
	if err != nil {
		return nil, err
	}
	if _, err := data.New(templates, equipment, items); err != nil {
		return nil, err
	}

	files := make(map[string][]byte, 3)
	for name, v := range map[string]any{
		data.TemplatesFile: templates,
		data.EquipmentFile: equipment,
		data.ItemsFile:     items,
	} {
		b, err := encodeJSON(v)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", name, err)
		}
		files[name] = b
	}
	return files, nil
}

// encodeJSON returns v as indented JSON with a final newline. HTML
// characters are not escaped, so names stay readable.
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readXML(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := xml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	return nil
}

// classInfo is one class from classList.xml.
type classInfo struct {
	name      string
	hasParent bool
}

func readClassList(fsys fs.FS) (map[int]classInfo, error) {
	var doc struct {
		Classes []struct {
			ID     *int   `xml:"classId,attr"`
			Name   string `xml:"name,attr"`
			Parent *int   `xml:"parentClassId,attr"`
		} `xml:"class"`
	}
	if err := readXML(fsys, classListFile, &doc); err != nil {
		return nil, err
	}
	classes := make(map[int]classInfo, len(doc.Classes))
	for _, c := range doc.Classes {
		if c.ID == nil {
			return nil, fmt.Errorf("%s: class %q has no classId", classListFile, c.Name)
		}
		if _, dup := classes[*c.ID]; dup {
			return nil, fmt.Errorf("%s: duplicate class %d", classListFile, *c.ID)
		}
		classes[*c.ID] = classInfo{name: c.Name, hasParent: c.Parent != nil}
	}
	return classes, nil
}

// race is a race from the Mobius Race enum.
type race struct {
	id   int // enum ordinal, the ID the client uses
	name string
}

var (
	// playerClassRe matches an enum constant of PlayerClass, such as
	// FIGHTER(0, false, Race.HUMAN, null) or
	// WARLOCK(14, true, true, Race.HUMAN, WIZARD).
	playerClassRe = regexp.MustCompile(`(?m)^\s*[A-Z][A-Z0-9_]*\(\s*(\d+)\s*,[^)]*?\bRace\.([A-Z][A-Z0-9_]*)`)
	commentRe     = regexp.MustCompile(`(?s)/\*.*?\*/|//[^\n]*`)
	identRe       = regexp.MustCompile(`^[A-Z][A-Z0-9_]*`)
)

// readRaces returns the race of each class ID, read from PlayerClass.java,
// with the race IDs taken from the order of the Race enum in Race.java.
func readRaces(fsys fs.FS) (map[int]race, error) {
	ordinals, err := readRaceEnum(fsys)
	if err != nil {
		return nil, err
	}
	src, err := fs.ReadFile(fsys, playerClassFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", playerClassFile, err)
	}
	races := make(map[int]race)
	for _, m := range playerClassRe.FindAllStringSubmatch(string(src), -1) {
		var id int
		if _, err := fmt.Sscan(m[1], &id); err != nil {
			return nil, fmt.Errorf("%s: class ID %q: %w", playerClassFile, m[1], err)
		}
		ord, ok := ordinals[m[2]]
		if !ok {
			return nil, fmt.Errorf("%s: class %d uses Race.%s, which %s does not define", playerClassFile, id, m[2], raceFile)
		}
		races[id] = race{id: ord, name: raceName(m[2])}
	}
	if len(races) == 0 {
		return nil, fmt.Errorf("%s: no classes found", playerClassFile)
	}
	return races, nil
}

// readRaceEnum returns the ordinal of each constant of the Race enum.
func readRaceEnum(fsys fs.FS) (map[string]int, error) {
	src, err := fs.ReadFile(fsys, raceFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", raceFile, err)
	}
	text := commentRe.ReplaceAllString(string(src), "")
	_, body, ok := strings.Cut(text, "enum Race")
	if ok {
		_, body, ok = strings.Cut(body, "{")
	}
	if !ok {
		return nil, fmt.Errorf("%s: enum Race not found", raceFile)
	}
	if end := strings.IndexAny(body, ";}"); end >= 0 {
		body = body[:end]
	}
	ordinals := make(map[string]int)
	for i, part := range strings.Split(body, ",") {
		name := identRe.FindString(strings.TrimSpace(part))
		if name == "" {
			if strings.TrimSpace(part) == "" {
				continue // trailing comma
			}
			return nil, fmt.Errorf("%s: cannot read enum constant %q", raceFile, strings.TrimSpace(part))
		}
		ordinals[name] = i
	}
	if len(ordinals) == 0 {
		return nil, fmt.Errorf("%s: enum Race has no constants", raceFile)
	}
	return ordinals, nil
}

// raceName turns an enum constant such as DARK_ELF into "Dark Elf".
func raceName(constant string) string {
	words := strings.Split(strings.ToLower(constant), "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

type xmlTemplate struct {
	ClassID *int `xml:"classId"`
	Static  struct {
		INT    int `xml:"baseINT"`
		STR    int `xml:"baseSTR"`
		CON    int `xml:"baseCON"`
		MEN    int `xml:"baseMEN"`
		DEX    int `xml:"baseDEX"`
		WIT    int `xml:"baseWIT"`
		Points []struct {
			X int `xml:"x,attr"`
			Y int `xml:"y,attr"`
			Z int `xml:"z,attr"`
		} `xml:"creationPoints>node"`
	} `xml:"staticData"`
	Levels []struct {
		Level int     `xml:"val,attr"`
		HP    float64 `xml:"hp"`
		MP    float64 `xml:"mp"`
		CP    float64 `xml:"cp"`
	} `xml:"lvlUpgainData>level"`
}

func readTemplates(fsys fs.FS, classes map[int]classInfo, races map[int]race) ([]data.Template, error) {
	names, err := fs.Glob(fsys, startingClassDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no templates found", startingClassDir)
	}
	var templates []data.Template
	for _, name := range names {
		var x xmlTemplate
		if err := readXML(fsys, name, &x); err != nil {
			return nil, err
		}
		if x.ClassID == nil {
			return nil, fmt.Errorf("%s: no classId", name)
		}
		id := *x.ClassID
		class, ok := classes[id]
		if !ok {
			return nil, fmt.Errorf("%s: class %d is not in %s", name, id, classListFile)
		}
		if class.hasParent {
			return nil, fmt.Errorf("%s: class %d (%s) has a parent class, so it is not a starting class", name, id, class.name)
		}
		r, ok := races[id]
		if !ok {
			return nil, fmt.Errorf("%s: class %d has no race in %s", name, id, playerClassFile)
		}
		t := data.Template{
			ClassID:   id,
			ClassName: class.name,
			Race:      r.id,
			RaceName:  r.name,
			Stats: data.BaseStats{
				STR: x.Static.STR, CON: x.Static.CON, DEX: x.Static.DEX,
				INT: x.Static.INT, WIT: x.Static.WIT, MEN: x.Static.MEN,
			},
			CreationPoints: make([]data.Location, 0, len(x.Static.Points)),
		}
		for _, p := range x.Static.Points {
			t.CreationPoints = append(t.CreationPoints, data.Location{X: p.X, Y: p.Y, Z: p.Z})
		}
		found := false
		for _, l := range x.Levels {
			if l.Level == 1 {
				t.Level1 = data.Vitals{HP: l.HP, MP: l.MP, CP: l.CP}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%s: class %d has no level 1 data", name, id)
		}
		templates = append(templates, t)
	}
	slices.SortFunc(templates, func(a, b data.Template) int { return cmp.Compare(a.ClassID, b.ClassID) })
	return templates, nil
}

func readEquipment(fsys fs.FS) ([]data.Equipment, error) {
	var doc struct {
		Lists []struct {
			ClassID *int `xml:"classId,attr"`
			Items   []struct {
				ID       *int   `xml:"id,attr"`
				Count    *int64 `xml:"count,attr"`
				Equipped bool   `xml:"equipped,attr"`
			} `xml:"item"`
		} `xml:"equipment"`
	}
	if err := readXML(fsys, equipmentFile, &doc); err != nil {
		return nil, err
	}
	var equipment []data.Equipment
	for _, l := range doc.Lists {
		if l.ClassID == nil {
			return nil, fmt.Errorf("%s: equipment without classId", equipmentFile)
		}
		e := data.Equipment{ClassID: *l.ClassID, Items: make([]data.EquipmentItem, 0, len(l.Items))}
		for _, it := range l.Items {
			if it.ID == nil || it.Count == nil {
				return nil, fmt.Errorf("%s: class %d: item without id or count", equipmentFile, e.ClassID)
			}
			e.Items = append(e.Items, data.EquipmentItem{ItemID: *it.ID, Count: *it.Count, Equipped: it.Equipped})
		}
		slices.SortStableFunc(e.Items, func(a, b data.EquipmentItem) int { return cmp.Compare(a.ItemID, b.ItemID) })
		equipment = append(equipment, e)
	}
	slices.SortStableFunc(equipment, func(a, b data.Equipment) int { return cmp.Compare(a.ClassID, b.ClassID) })
	return equipment, nil
}

type xmlItems struct {
	Items []struct {
		ID   int    `xml:"id,attr"`
		Type string `xml:"type,attr"`
		Name string `xml:"name,attr"`
		Sets []struct {
			Name string `xml:"name,attr"`
			Val  string `xml:"val,attr"`
		} `xml:"set"`
	} `xml:"item"`
}

// readItems returns the items used by equipment, read from the item files
// directly in the items folder (custom items are left out).
func readItems(fsys fs.FS, equipment []data.Equipment) ([]data.Item, error) {
	wanted := make(map[int]int) // item ID -> a class that uses it
	for _, e := range equipment {
		for _, it := range e.Items {
			if _, ok := wanted[it.ItemID]; !ok {
				wanted[it.ItemID] = e.ClassID
			}
		}
	}
	names, err := fs.Glob(fsys, itemsDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no item files found", itemsDir)
	}
	found := make(map[int]string) // item ID -> file
	var items []data.Item
	for _, name := range names {
		var x xmlItems
		if err := readXML(fsys, name, &x); err != nil {
			return nil, err
		}
		for _, it := range x.Items {
			if _, ok := wanted[it.ID]; !ok {
				continue
			}
			if prev, dup := found[it.ID]; dup {
				return nil, fmt.Errorf("item %d is defined in both %s and %s", it.ID, prev, name)
			}
			found[it.ID] = name
			item := data.Item{ID: it.ID, Name: it.Name, Type: it.Type}
			for _, s := range it.Sets {
				if s.Name == "bodypart" {
					item.BodyPart = s.Val
				}
			}
			items = append(items, item)
		}
	}
	var missing []error
	for _, id := range slices.Sorted(maps.Keys(wanted)) {
		if _, ok := found[id]; !ok {
			missing = append(missing, fmt.Errorf("item %d (initial equipment of class %d) not found in %s", id, wanted[id], path.Join(itemsDir, "*.xml")))
		}
	}
	if err := errors.Join(missing...); err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(a, b data.Item) int { return cmp.Compare(a.ID, b.ID) })
	return items, nil
}
