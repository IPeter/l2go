package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// cursedWeaponsXML is the file Mobius CursedWeaponsManager.load reads.
const cursedWeaponsXML = "dist/game/data/CursedWeapons.xml"

// The cursedweapons domain converts CursedWeapons.xml (Plan 14 M11m),
// loaded by data.Load.
func init() { registerDomain("cursedweapons", genCursedWeapons) }

// genCursedWeapons reads the cursed weapons and adds
// cursedweapons/cursedweapons.json, one weapon per line, in file order.
func genCursedWeapons(o options) error {
	list, err := readCursedWeapons(o)
	if err != nil {
		return fmt.Errorf("cursed weapons: %w", err)
	}
	if err := data.CheckCursedWeapons(list); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.CursedWeaponsFile, list); err != nil {
		return err
	}
	names := make([]string, len(list))
	for i, w := range list {
		names[i] = w.Name
	}
	fmt.Fprintf(o.log, "cursedweapons: %d weapons (%s)\n", len(list), strings.Join(names, ", "))
	return nil
}

// readCursedWeapons reads the <item> elements of the <list> root as Mobius
// does: id, skillId and name attributes, and the values of the dropRate,
// duration, durationLost, disapearChance and stageKills children. A value
// Mobius would leave at 0 is an error here, and so is an unknown element.
func readCursedWeapons(o options) ([]data.CursedWeapon, error) {
	root, err := readXMLTree(o.src, cursedWeaponsXML)
	if err != nil {
		return nil, err
	}
	if root.name != "list" {
		return nil, root.errorf("want <list>")
	}
	if err := root.checkChildren("item"); err != nil {
		return nil, err
	}
	var list []data.CursedWeapon
	for _, n := range root.children {
		w, err := readCursedWeapon(n)
		if err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	if len(list) == 0 {
		return nil, errors.New(cursedWeaponsXML + ": no weapons")
	}
	return list, nil
}

// readCursedWeapon reads one <item>.
func readCursedWeapon(n *xmlNode) (data.CursedWeapon, error) {
	var w data.CursedWeapon
	if err := n.checkAttrs("id", "skillId", "name"); err != nil {
		return w, err
	}
	var err error
	if w.ItemID, err = n.intAttr("id"); err != nil {
		return w, err
	}
	if w.SkillID, err = n.intAttr("skillId"); err != nil {
		return w, err
	}
	name, ok := n.attr("name")
	if !ok || name == "" {
		return w, n.errorf("no name")
	}
	w.Name = name
	fields := map[string]*int{
		"dropRate":       &w.DropRate,
		"duration":       &w.Duration,
		"durationLost":   &w.DurationLost,
		"disapearChance": &w.DisappearChance,
		"stageKills":     &w.StageKills,
	}
	seen := make(map[string]bool, len(fields))
	for _, c := range n.children {
		dst, ok := fields[c.name]
		if !ok {
			return w, c.errorf("unknown element")
		}
		if seen[c.name] {
			return w, c.errorf("given twice")
		}
		seen[c.name] = true
		if err := c.checkAttrs("val"); err != nil {
			return w, err
		}
		if *dst, err = c.intAttr("val"); err != nil {
			return w, err
		}
	}
	for _, name := range []string{"dropRate", "duration", "durationLost", "disapearChance", "stageKills"} {
		if !seen[name] {
			return w, n.errorf("no <%s>", name)
		}
	}
	return w, nil
}
