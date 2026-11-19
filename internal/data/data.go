// Package data loads the static game data L2Go needs at runtime: the
// starting class templates, their initial equipment and the items it
// refers to.
//
// The data lives in JSON files under the data/ folder. They are generated
// from the L2J Mobius Interlude datapack by cmd/datagen, which uses the
// types of this package, so the JSON schema is defined here:
//
//   - players/templates.json: []Template, sorted by class ID.
//   - players/initial_equipment.json: []Equipment, sorted by class ID.
//   - items/items.json: []Item, sorted by item ID.
//
// Load and New check the data (see New) and fail with a message naming the
// file and the record at fault.
package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// Paths of the data files, relative to the data folder.
const (
	TemplatesFile = "players/templates.json"
	EquipmentFile = "players/initial_equipment.json"
	ItemsFile     = "items/items.json"
)

// Item types, as Mobius names them.
const (
	ItemTypeWeapon = "Weapon"
	ItemTypeArmor  = "Armor"
	ItemTypeEtc    = "EtcItem"
)

// ErrInvalid is wrapped by the errors of New and Load when the data breaks
// a rule.
var ErrInvalid = errors.New("invalid game data")

// BaseStats are the base attributes of a class.
type BaseStats struct {
	STR int `json:"str"`
	CON int `json:"con"`
	DEX int `json:"dex"`
	INT int `json:"int"`
	WIT int `json:"wit"`
	MEN int `json:"men"`
}

// Vitals are maximum HP, MP and CP.
type Vitals struct {
	HP float64 `json:"hp"`
	MP float64 `json:"mp"`
	CP float64 `json:"cp"`
}

// Location is a point in the world.
type Location struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}

// Template describes a starting class: what character creation offers and
// what a new character starts with.
type Template struct {
	ClassID   int    `json:"classId"`
	ClassName string `json:"className"`
	// Race is the race ID the client uses (Mobius Race ordinal): 0 Human,
	// 1 Elf, 2 Dark Elf, 3 Orc, 4 Dwarf.
	Race     int       `json:"race"`
	RaceName string    `json:"raceName"`
	Stats    BaseStats `json:"baseStats"`
	// Level1 holds the maximum HP, MP and CP at level 1.
	Level1 Vitals `json:"level1"`
	// CreationPoints are the possible start locations; a new character
	// gets one of them at random.
	CreationPoints []Location `json:"creationPoints"`
}

// EquipmentItem is one item a new character receives.
type EquipmentItem struct {
	ItemID   int   `json:"itemId"`
	Count    int64 `json:"count"`
	Equipped bool  `json:"equipped"`
}

// Equipment is the initial equipment of a starting class.
type Equipment struct {
	ClassID int             `json:"classId"`
	Items   []EquipmentItem `json:"items"`
}

// Item holds the item data the server needs so far.
type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// Type is ItemTypeWeapon, ItemTypeArmor or ItemTypeEtc.
	Type string `json:"type"`
	// BodyPart is the Mobius equip slot name (for example "rhand",
	// "lrhand", "chest", "legs"), or empty for items that cannot be
	// equipped.
	BodyPart string `json:"bodyPart,omitempty"`
}

// Data is the loaded game data. The exported slices keep the order of the
// files. It must not be modified after New or Load returns, and is then
// safe for concurrent use.
type Data struct {
	Templates        []Template
	InitialEquipment []Equipment
	Items            []Item

	templates map[int]int // class ID -> index in Templates
	equipment map[int]int // class ID -> index in InitialEquipment
	items     map[int]int // item ID -> index in Items
}

// Load reads and checks the data files in dir.
func Load(dir string) (*Data, error) {
	d, err := LoadFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return d, nil
}

// LoadFS reads and checks the data files in fsys.
func LoadFS(fsys fs.FS) (*Data, error) {
	var (
		templates []Template
		equipment []Equipment
		items     []Item
	)
	if err := readJSON(fsys, TemplatesFile, &templates); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, EquipmentFile, &equipment); err != nil {
		return nil, err
	}
	if err := readJSON(fsys, ItemsFile, &items); err != nil {
		return nil, err
	}
	return New(templates, equipment, items)
}

// readJSON decodes the JSON file name into v. Unknown fields and trailing
// data are errors.
func readJSON(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("data: read %s: %w", name, err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("data: parse %s: %w", name, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("data: parse %s: unexpected data after the JSON value", name)
	}
	return nil
}

// New checks the data and builds the lookup maps. The slices are used as
// they are, not copied. It checks that:
//
//   - there is at least one template, and IDs are unique in each list;
//   - each template has a name, a race, positive base stats, positive
//     level 1 HP, MP and CP, and at least one creation point;
//   - each equipment list belongs to a template class and has items;
//   - each equipment item exists, has a positive count, and can be worn
//     (has a body part) when it is marked equipped;
//   - each item has a positive ID, a name and a known type.
//
// All problems are reported together in one error wrapping ErrInvalid.
func New(templates []Template, equipment []Equipment, items []Item) (*Data, error) {
	d := &Data{
		Templates:        templates,
		InitialEquipment: equipment,
		Items:            items,
		templates:        make(map[int]int, len(templates)),
		equipment:        make(map[int]int, len(equipment)),
		items:            make(map[int]int, len(items)),
	}
	var p problems

	for i, it := range items {
		where := fmt.Sprintf("%s: item %d (%q)", ItemsFile, it.ID, it.Name)
		if _, dup := d.items[it.ID]; dup {
			p.addf("%s: duplicate item ID", where)
			continue
		}
		d.items[it.ID] = i
		if it.ID <= 0 {
			p.addf("%s: item ID must be positive", where)
		}
		if it.Name == "" {
			p.addf("%s: no name", where)
		}
		switch it.Type {
		case ItemTypeWeapon, ItemTypeArmor, ItemTypeEtc:
		default:
			p.addf("%s: unknown type %q (want %s, %s or %s)", where, it.Type, ItemTypeWeapon, ItemTypeArmor, ItemTypeEtc)
		}
	}

	if len(templates) == 0 {
		p.addf("%s: no starting class templates", TemplatesFile)
	}
	for i, t := range templates {
		where := fmt.Sprintf("%s: class %d (%q)", TemplatesFile, t.ClassID, t.ClassName)
		if _, dup := d.templates[t.ClassID]; dup {
			p.addf("%s: duplicate class ID", where)
			continue
		}
		d.templates[t.ClassID] = i
		if t.ClassID < 0 {
			p.addf("%s: class ID must not be negative", where)
		}
		if t.ClassName == "" {
			p.addf("%s: no class name", where)
		}
		if t.Race < 0 || t.RaceName == "" {
			p.addf("%s: missing race (got %d %q)", where, t.Race, t.RaceName)
		}
		s := t.Stats
		if s.STR <= 0 || s.CON <= 0 || s.DEX <= 0 || s.INT <= 0 || s.WIT <= 0 || s.MEN <= 0 {
			p.addf("%s: base stats must be positive (got %+v)", where, s)
		}
		if t.Level1.HP <= 0 || t.Level1.MP <= 0 || t.Level1.CP <= 0 {
			p.addf("%s: level 1 HP, MP and CP must be positive (got %+v)", where, t.Level1)
		}
		if len(t.CreationPoints) == 0 {
			p.addf("%s: no creation points", where)
		}
	}

	for i, e := range equipment {
		where := fmt.Sprintf("%s: class %d", EquipmentFile, e.ClassID)
		if _, dup := d.equipment[e.ClassID]; dup {
			p.addf("%s: duplicate class ID", where)
			continue
		}
		d.equipment[e.ClassID] = i
		if _, ok := d.templates[e.ClassID]; !ok {
			p.addf("%s: no template for this class in %s", where, TemplatesFile)
		}
		if len(e.Items) == 0 {
			p.addf("%s: no items", where)
		}
		for _, ei := range e.Items {
			idx, ok := d.items[ei.ItemID]
			if !ok {
				p.addf("%s: item %d does not exist in %s", where, ei.ItemID, ItemsFile)
				continue
			}
			if ei.Count <= 0 {
				p.addf("%s: item %d: count must be positive (got %d)", where, ei.ItemID, ei.Count)
			}
			if ei.Equipped && items[idx].BodyPart == "" {
				p.addf("%s: item %d (%q) is marked equipped but has no body part", where, ei.ItemID, items[idx].Name)
			}
		}
	}

	if err := p.err(); err != nil {
		return nil, err
	}
	return d, nil
}

// problems collects validation messages.
type problems []string

func (p *problems) addf(format string, args ...any) {
	*p = append(*p, fmt.Sprintf(format, args...))
}

func (p problems) err() error {
	if len(p) == 0 {
		return nil
	}
	return fmt.Errorf("data: %w: %s", ErrInvalid, strings.Join(p, "; "))
}

// Template returns the template of a starting class.
func (d *Data) Template(classID int) (Template, bool) {
	i, ok := d.templates[classID]
	if !ok {
		return Template{}, false
	}
	return d.Templates[i], true
}

// InitialItems returns the items a new character of the class receives, or
// nil if the class has none. The slice must not be modified.
func (d *Data) InitialItems(classID int) []EquipmentItem {
	i, ok := d.equipment[classID]
	if !ok {
		return nil
	}
	return d.InitialEquipment[i].Items
}

// Item returns the item with the given ID.
func (d *Data) Item(id int) (Item, bool) {
	i, ok := d.items[id]
	if !ok {
		return Item{}, false
	}
	return d.Items[i], true
}
