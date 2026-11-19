// Package data loads the static game data L2Go needs at runtime.
//
// The data lives in JSON files under the data/ folder. They are generated
// from the L2J Mobius Interlude datapack by cmd/datagen, which uses the
// types of this package, so the JSON schema is defined here. A file bigger
// than 10 MB is stored gzipped (name.json.gz); the loaders read either
// form.
//
// Load reads the player and item data, in two domains:
//
//   - players: players/classes.json ([]ClassTemplate, all player classes,
//     sorted by class ID), players/initial_equipment.json ([]Equipment,
//     sorted by class ID) and the player tables: players/stat_bonus.json
//     (StatBonusTable), players/experience.json (ExperienceTable),
//     players/experience_loss.json and players/karma_loss.json
//     (LevelTable), players/hit_condition_bonus.json (HitConditionBonus).
//   - items: items/items.json ([]Item, all items, sorted by ID, one item
//     per line).
//
// LoadWorld reads the world data (NPCs, spawns, teleporters, zones, map
// regions; see world.go).
//
// The enums (Stat, BodyPart, WeaponType, PlayerClass, ...) mirror the Mobius
// Java enums and are generated from them (see enum.go).
//
// The loaders check the data and fail with a message naming the file and
// the record at fault. All problems found are reported together, in an
// error wrapping ErrInvalid.
package data

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
)

// Paths of the player and item data files, relative to the data folder.
const (
	ClassesFile           = "players/classes.json"
	EquipmentFile         = "players/initial_equipment.json"
	StatBonusFile         = "players/stat_bonus.json"
	ExperienceFile        = "players/experience.json"
	ExperienceLossFile    = "players/experience_loss.json"
	KarmaLossFile         = "players/karma_loss.json"
	HitConditionBonusFile = "players/hit_condition_bonus.json"
	ItemsFile             = "items/items.json"
)

// The kinds of item (Item.Type), named like the Mobius item template
// classes.
const (
	ItemTypeWeapon = "Weapon"
	ItemTypeArmor  = "Armor"
	ItemTypeEtc    = "EtcItem"
)

// ErrInvalid is wrapped by the errors of the loaders when the data breaks
// a rule.
var ErrInvalid = errors.New("invalid game data")

// Data is the loaded player and item data. The exported fields hold the
// files' content in file order. Data must not be modified after a loader
// returns it, and is then safe for concurrent use; this includes the
// values the lookups return pointers to.
type Data struct {
	// Templates are the starting classes, the ones character creation
	// offers, sorted by class ID. They are built from Classes.
	Templates []Template
	// Classes are all player class templates, sorted by class ID.
	Classes []ClassTemplate
	// InitialEquipment lists the items of each starting class.
	InitialEquipment []Equipment
	// StatBonuses are the bonuses of the base attribute values.
	StatBonuses StatBonusTable
	// Experience is the experience needed per level.
	Experience ExperienceTable
	// ExperienceLoss is the percent of experience lost on death, per
	// level.
	ExperienceLoss LevelTable
	// KarmaLoss is the karma loss modifier per level.
	KarmaLoss LevelTable
	// HitConditionBonus is the accuracy bonus of the attack position.
	HitConditionBonus HitConditionBonus
	// Items are all item templates, sorted by item ID.
	Items []Item

	loaded    map[string]bool
	classes   map[int]int // class ID -> index in Classes
	templates map[int]int // class ID -> index in Templates
	equipment map[int]int // class ID -> index in InitialEquipment
	items     []int32     // item ID -> index+1 in Items, 0 if there is none
}

// A domain is one part of the data folder: the files it reads and the
// fields of Data it fills. Domains register from init in the file that
// defines their types.
type domain struct {
	name string
	// load reads the domain's files from fsys into d and checks them on
	// their own.
	load func(fsys fs.FS, d *Data) error
	// link checks references into other domains. It runs once every
	// requested domain is loaded; Data.Loaded tells which are. May be nil.
	link func(d *Data) error
}

// domains are the registered domains.
var domains []domain

func registerDomain(dom domain) { domains = append(domains, dom) }

// DomainNames returns the names of the domains Load reads, sorted.
func DomainNames() []string {
	names := make([]string, 0, len(domains))
	for _, dom := range domains {
		names = append(names, dom.name)
	}
	slices.Sort(names)
	return slices.Compact(names)
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
	return LoadDomains(fsys, DomainNames()...)
}

// LoadDomains reads and checks the files of the named domains only, such
// as "players" or "items". References into domains that are not loaded are
// not checked. cmd/datagen uses it to check the files it generates.
func LoadDomains(fsys fs.FS, names ...string) (*Data, error) {
	var picked []domain
	for _, name := range names {
		var found []domain
		for _, dom := range domains {
			if dom.name == name {
				found = append(found, dom)
			}
		}
		switch {
		case len(found) == 0:
			return nil, fmt.Errorf("data: unknown domain %q (have %s)", name, strings.Join(DomainNames(), ", "))
		case len(found) > 1:
			return nil, fmt.Errorf("data: domain %q is registered %d times", name, len(found))
		}
		if !slices.ContainsFunc(picked, func(dom domain) bool { return dom.name == name }) {
			picked = append(picked, found[0])
		}
	}

	d := &Data{loaded: make(map[string]bool, len(picked))}
	var errs []error
	for _, dom := range picked {
		if err := dom.load(fsys, d); err != nil {
			errs = append(errs, err)
			continue
		}
		d.loaded[dom.name] = true
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	for _, dom := range picked {
		if dom.link == nil {
			continue
		}
		if err := dom.link(d); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return d, nil
}

// Loaded reports whether the domain name was loaded into d.
func (d *Data) Loaded(name string) bool { return d.loaded[name] }

// readJSON decodes the data file name, or its gzipped form name+".gz",
// into v. Unknown fields and trailing data are errors, and so is having
// both forms (one of them is stale).
func readJSON(fsys fs.FS, name string, v any) error {
	b, err := readDataFile(fsys, name)
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

// readDataFile returns the content of name, or of name+".gz" uncompressed
// when only that exists.
func readDataFile(fsys fs.FS, name string) ([]byte, error) {
	plain, plainErr := fs.ReadFile(fsys, name)
	zipped, zipErr := fs.ReadFile(fsys, name+".gz")
	switch {
	case plainErr == nil && zipErr == nil:
		return nil, fmt.Errorf("both %s and %s.gz exist; remove the stale one", name, name)
	case plainErr == nil:
		return plain, nil
	case !errors.Is(plainErr, fs.ErrNotExist):
		return nil, plainErr
	case zipErr == nil:
		zr, err := gzip.NewReader(bytes.NewReader(zipped))
		if err != nil {
			return nil, fmt.Errorf("%s.gz: %w", name, err)
		}
		b, err := io.ReadAll(zr)
		if err != nil {
			return nil, fmt.Errorf("%s.gz: %w", name, err)
		}
		return b, nil
	case errors.Is(zipErr, fs.ErrNotExist):
		return nil, plainErr
	default:
		return nil, zipErr
	}
}

// maxProblems caps the messages of one error; the rest are counted.
const maxProblems = 50

// problems collects validation messages.
type problems struct {
	msgs []string
	more int
}

func (p *problems) addf(format string, args ...any) {
	if len(p.msgs) == maxProblems {
		p.more++
		return
	}
	p.msgs = append(p.msgs, fmt.Sprintf(format, args...))
}

func (p *problems) err() error {
	if len(p.msgs) == 0 {
		return nil
	}
	msg := strings.Join(p.msgs, "; ")
	if p.more > 0 {
		msg += fmt.Sprintf("; and %d more", p.more)
	}
	return fmt.Errorf("data: %w: %s", ErrInvalid, msg)
}
