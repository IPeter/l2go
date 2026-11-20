package main

import (
	"fmt"
	"slices"

	"github.com/IPeter/l2go/internal/data"
)

// initialShortcutsXML is the shortcut list Mobius's InitialShortcutData
// reads.
const initialShortcutsXML = "dist/game/data/stats/players/initialShortcuts.xml"

// The shortcuts domain converts initialShortcuts.xml, loaded by
// data.LoadInitialShortcuts.
func init() { registerDomain("shortcuts", genShortcuts) }

// genShortcuts reads the initial shortcuts and adds
// shortcuts/initial_shortcuts.json, one shortcut per line.
func genShortcuts(o options) error {
	list, err := readInitialShortcuts(o)
	if err != nil {
		return fmt.Errorf("shortcuts: %w", err)
	}
	if _, err := data.NewInitialShortcuts(list); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.InitialShortcutsFile, list); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "shortcuts: %d initial shortcuts\n", len(list))
	return nil
}

// readInitialShortcuts reads the shortcuts as Mobius's InitialShortcutData
// does: every <shortcuts> without classId adds to the shortcuts of every
// class, in XML order; those with a classId follow, one block per class.
// The macro presets (<macros>) wait for macros and are left out. A class
// listed twice fails, where Mobius would keep the last.
func readInitialShortcuts(o options) ([]data.InitialShortcut, error) {
	doc, err := parseMX(o.src, initialShortcutsXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var (
		global, classes []data.InitialShortcut
		seen            []int
	)
	for _, e := range root.children() {
		switch e.name {
		case "macros":
			e.skip()
			continue
		case "shortcuts":
		default:
			e.unknown()
			continue
		}
		class := e.intAttr("classId", data.AllClasses)
		if class != data.AllClasses {
			if slices.Contains(seen, class) {
				e.errorf("class %d is listed twice (Mobius keeps the last)", class)
				continue
			}
			seen = append(seen, class)
		}
		for _, page := range e.children() {
			if page.name != "page" {
				page.unknown()
				continue
			}
			pageID := page.reqInt("pageId")
			for _, slot := range page.children() {
				if slot.name != "slot" {
					slot.unknown()
					continue
				}
				sc := data.InitialShortcut{
					ClassID: class,
					Page:    pageID,
					Slot:    slot.reqInt("slotId"),
					Type:    slot.reqStr("shortcutType"),
					ID:      slot.reqInt("shortcutId"),
					Level:   slot.intAttr("shortcutLevel", 0),
				}
				if class == data.AllClasses {
					global = append(global, sc)
				} else {
					classes = append(classes, sc)
				}
			}
		}
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	return append(global, classes...), nil
}
