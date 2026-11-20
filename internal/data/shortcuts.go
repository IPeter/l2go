package data

// Initial shortcut data: the shortcut bar of a new character, from Mobius
// stats/players/initialShortcuts.xml, loaded with LoadInitialShortcuts.

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// InitialShortcutsFile is the path of the initial shortcut file, relative to
// the data folder.
const InitialShortcutsFile = "shortcuts/initial_shortcuts.json"

// AllClasses is the ClassID of an initial shortcut every class gets (a
// <shortcuts> without classId).
const AllClasses = -1

// The shortcut types of an InitialShortcut (Mobius ShortcutType names).
const (
	ShortcutTypeItem   = "ITEM"
	ShortcutTypeSkill  = "SKILL"
	ShortcutTypeAction = "ACTION"
	ShortcutTypeMacro  = "MACRO"
	ShortcutTypeRecipe = "RECIPE"
)

// shortcutTypes are the valid InitialShortcut types.
var shortcutTypes = []string{ShortcutTypeItem, ShortcutTypeSkill, ShortcutTypeAction, ShortcutTypeMacro, ShortcutTypeRecipe}

// InitialShortcut is one <slot> of initialShortcuts.xml.
type InitialShortcut struct {
	// ClassID is the class that gets the shortcut, AllClasses for every
	// class.
	ClassID int `json:"classId"`
	// Page is 0-10 and Slot 0-11, as the XML's pageId and slotId.
	Page int `json:"page"`
	Slot int `json:"slot"`
	// Type is ShortcutTypeItem, ShortcutTypeSkill, ...
	Type string `json:"type"`
	// ID is the item template ID of an item shortcut (the game puts the
	// object ID of the character's item there), else the skill, action,
	// macro or recipe ID.
	ID int `json:"id"`
	// Level is the skill level of a skill shortcut, 0 when the XML has none.
	Level int `json:"level,omitempty"`
}

// InitialShortcuts is the checked initial shortcut data. It must not be
// modified after NewInitialShortcuts or LoadInitialShortcuts returns, and
// is then safe for concurrent use.
type InitialShortcuts struct {
	// Shortcuts are in the order of the XML: the shortcuts of every class
	// first, then those of each class.
	Shortcuts []InitialShortcut
}

// LoadInitialShortcuts reads and checks the initial shortcut file in dir.
func LoadInitialShortcuts(dir string) (*InitialShortcuts, error) {
	s, err := LoadInitialShortcutsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return s, nil
}

// LoadInitialShortcutsFS reads and checks the initial shortcut file in fsys.
func LoadInitialShortcutsFS(fsys fs.FS) (*InitialShortcuts, error) {
	var list []InitialShortcut
	if err := readJSON(fsys, InitialShortcutsFile, &list); err != nil {
		return nil, err
	}
	return NewInitialShortcuts(list)
}

// NewInitialShortcuts checks the shortcuts: pages 0-10, slots 0-11, a known
// type, a class ID of AllClasses or above, the shortcuts of every class
// before the others, and no slot of a class taken twice. The slice is used
// as it is.
func NewInitialShortcuts(list []InitialShortcut) (*InitialShortcuts, error) {
	var p problems
	type key struct{ class, page, slot int }
	seen := make(map[key]bool, len(list))
	for i, sc := range list {
		where := fmt.Sprintf("%s: shortcut %d (class %d, page %d, slot %d)", InitialShortcutsFile, i, sc.ClassID, sc.Page, sc.Slot)
		switch {
		case sc.Page < 0 || sc.Page > 10:
			p.addf("%s: page out of 0-10", where)
		case sc.Slot < 0 || sc.Slot > 11:
			p.addf("%s: slot out of 0-11", where)
		case !slices.Contains(shortcutTypes, sc.Type):
			p.addf("%s: unknown type %q", where, sc.Type)
		case sc.ClassID < AllClasses:
			p.addf("%s: invalid class", where)
		case i > 0 && sc.ClassID == AllClasses && list[i-1].ClassID != AllClasses:
			p.addf("%s: the shortcuts of every class must come first", where)
		}
		k := key{sc.ClassID, sc.Page, sc.Slot}
		if seen[k] {
			p.addf("%s: the slot is listed twice", where)
		}
		seen[k] = true
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return &InitialShortcuts{Shortcuts: list}, nil
}

// For returns the initial shortcuts of a class in Mobius's order
// (InitialShortcutData.registerAllShortcuts): those of every class, then the
// class's own.
func (s *InitialShortcuts) For(classID int) []InitialShortcut {
	var out []InitialShortcut
	for _, sc := range s.Shortcuts {
		if sc.ClassID == AllClasses {
			out = append(out, sc)
		}
	}
	for _, sc := range s.Shortcuts {
		if sc.ClassID == classID && classID != AllClasses {
			out = append(out, sc)
		}
	}
	return out
}
