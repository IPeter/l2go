// Package enchant holds the rules of enchanting and crystallization (Plan
// 14, M11a): the enchant scrolls and their rate groups (Mobius
// EnchantItemData, EnchantItemGroupsData, EnchantScroll, EnchantItemGroup,
// EnchantScrollGroup, EnchantRateItem), the success roll, the crystals of a
// failed enchant and of crystallization (ItemTemplate.getCrystalCount), the
// over-enchant limits and the Crystallize (248) skill levels. It is pure:
// the game adapts it to the packets, the inventory and the random source.
package enchant

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// File is the path of the enchant data, relative to the data folder. It is
// written by cmd/datagen from Mobius EnchantItemData.xml and
// EnchantItemGroups.xml.
const File = "enchant/enchant.json"

// DefaultMaxEnchant is the maxEnchant of a scroll that names none (Mobius
// AbstractEnchantItem).
const DefaultMaxEnchant = 65535

// Data is the enchant data as datagen writes it.
type Data struct {
	// Scrolls are the <enchant> elements of EnchantItemData.xml, sorted by
	// ID.
	Scrolls []Scroll `json:"scrolls"`
	// RateGroups are the <enchantRateGroup> elements of
	// EnchantItemGroups.xml, in file order (the derived maximum levels
	// depend on their names).
	RateGroups []RateGroup `json:"rateGroups"`
	// ScrollGroups are the <enchantScrollGroup> elements, sorted by ID.
	ScrollGroups []ScrollGroup `json:"scrollGroups"`
}

// Scroll is one enchant scroll (Mobius EnchantScroll).
type Scroll struct {
	ID int `json:"id"`
	// Grade is the crystal grade of the items it enchants (targetGrade,
	// NONE when missing).
	Grade data.CrystalType `json:"grade"`
	// MaxEnchant is the level from which it cannot be used (maxEnchant,
	// DefaultMaxEnchant when missing; 0 means no limit).
	MaxEnchant int `json:"maxEnchant"`
	// BonusRate is added to the chance (bonusRate, a crystal scroll's 100).
	BonusRate float64 `json:"bonusRate,omitempty"`
	// ScrollGroup is the scroll group of its rates (scrollGroupId, 0 when
	// missing).
	ScrollGroup int `json:"scrollGroup"`
	// Items, when there are any, are the only items it enchants, each with
	// its scroll group (an <item>'s altScrollGroupId, else ScrollGroup).
	Items []ScrollItem `json:"items,omitempty"`
}

// ScrollItem is an item a scroll is bound to.
type ScrollItem struct {
	ID          int `json:"id"`
	ScrollGroup int `json:"scrollGroup"`
}

// RateGroup is a named chance table (Mobius EnchantItemGroup).
type RateGroup struct {
	Name    string        `json:"name"`
	Chances []RangeChance `json:"chances"`
}

// RangeChance is the chance in percent of enchant levels Min to Max (Mobius
// RangeChanceHolder).
type RangeChance struct {
	Min    int     `json:"min"`
	Max    int     `json:"max"`
	Chance float64 `json:"chance"`
}

// ScrollGroup binds items to rate groups (Mobius EnchantScrollGroup): the
// first rate whose filters the item passes gives its rate group.
type ScrollGroup struct {
	ID    int        `json:"id"`
	Rates []RateItem `json:"rates"`
}

// RateItem is one <enchantRate> (Mobius EnchantRateItem). An empty filter
// passes every item.
type RateItem struct {
	// Group is the rate group's name.
	Group string `json:"group"`
	// Slots is the body part mask an item must share a bit with (0: any).
	Slots int `json:"slots,omitempty"`
	// MagicWeapon, when set, is what the item's magic weapon flag must be.
	MagicWeapon *bool `json:"magicWeapon,omitempty"`
	// ItemIDs, when there are any, are the only items it takes.
	ItemIDs []int `json:"itemIds,omitempty"`
}

// Load reads and checks the enchant data in the data folder dir.
func Load(dir string) (Data, error) {
	d, err := LoadFS(os.DirFS(dir))
	if err != nil {
		return Data{}, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return d, nil
}

// LoadFS reads and checks the enchant data in fsys.
func LoadFS(fsys fs.FS) (Data, error) {
	b, err := fs.ReadFile(fsys, File)
	if err != nil {
		return Data{}, fmt.Errorf("enchant: %w", err)
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return Data{}, fmt.Errorf("enchant: %s: %w", File, err)
	}
	if err := d.Check(); err != nil {
		return Data{}, err
	}
	return d, nil
}

// Check reports what is wrong with the data: scroll and scroll group IDs
// must be positive or zero, unique and sorted, rate group names unique and
// not empty, every reference must resolve, and ranges must not be
// reversed.
func (d *Data) Check() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(File+": "+format, args...)) }
	groups := make(map[string]bool, len(d.RateGroups))
	for _, g := range d.RateGroups {
		switch {
		case g.Name == "":
			add("a rate group has no name")
		case groups[g.Name]:
			add("rate group %s is listed twice", g.Name)
		}
		groups[g.Name] = true
		for _, c := range g.Chances {
			if c.Min < 0 || c.Max < c.Min || c.Chance < 0 {
				add("rate group %s: bad range %+v", g.Name, c)
			}
		}
	}
	scrollGroups := make(map[int]bool, len(d.ScrollGroups))
	for i, sg := range d.ScrollGroups {
		if i > 0 && d.ScrollGroups[i-1].ID >= sg.ID {
			add("scroll group %d: scroll groups are not sorted and unique", sg.ID)
		}
		scrollGroups[sg.ID] = true
		for _, r := range sg.Rates {
			if !groups[r.Group] {
				add("scroll group %d: unknown rate group %q", sg.ID, r.Group)
			}
		}
	}
	for i, s := range d.Scrolls {
		if s.ID <= 0 {
			add("scroll %d: the ID must be positive", s.ID)
		}
		if i > 0 && d.Scrolls[i-1].ID >= s.ID {
			add("scroll %d: scrolls are not sorted and unique", s.ID)
		}
		if !s.Grade.Valid() || s.MaxEnchant < 0 || s.BonusRate < 0 {
			add("scroll %d: bad grade, maxEnchant or bonusRate", s.ID)
		}
		if !scrollGroups[s.ScrollGroup] {
			add("scroll %d: unknown scroll group %d", s.ID, s.ScrollGroup)
		}
		for _, it := range s.Items {
			if !scrollGroups[it.ScrollGroup] {
				add("scroll %d: item %d: unknown scroll group %d", s.ID, it.ID, it.ScrollGroup)
			}
		}
	}
	return errors.Join(errs...)
}

// maxLevels returns the derived maximum levels of Mobius
// EnchantItemGroupsData: the highest level with a chance above 0, by the
// rate group's name (WEAPON; ACCESSORIES, RING, EARRING or NECK; else
// armor), the accessory one falling back to armor, each plus one.
func (d *Data) maxLevels() (weapon, armor, accessory int) {
	for _, g := range d.RateGroups {
		for _, c := range g.Chances {
			if c.Chance <= 0 {
				continue
			}
			switch {
			case strings.Contains(g.Name, "WEAPON"):
				weapon = max(weapon, c.Max)
			case strings.Contains(g.Name, "ACCESSORIES") || strings.Contains(g.Name, "RING") ||
				strings.Contains(g.Name, "EARRING") || strings.Contains(g.Name, "NECK"):
				accessory = max(accessory, c.Max)
			default:
				armor = max(armor, c.Max)
			}
		}
	}
	if accessory == 0 {
		accessory = armor
	}
	return weapon + 1, armor + 1, accessory + 1
}

// rateGroup returns the rate group named name, nil if there is none.
func (d *Data) rateGroup(name string) *RateGroup {
	i := slices.IndexFunc(d.RateGroups, func(g RateGroup) bool { return g.Name == name })
	if i < 0 {
		return nil
	}
	return &d.RateGroups[i]
}

// scrollGroup returns the scroll group with the ID, nil if there is none.
func (d *Data) scrollGroup(id int) *ScrollGroup {
	i, ok := slices.BinarySearchFunc(d.ScrollGroups, id, func(g ScrollGroup, id int) int { return g.ID - id })
	if !ok {
		return nil
	}
	return &d.ScrollGroups[i]
}
