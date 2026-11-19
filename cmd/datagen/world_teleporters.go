package main

import (
	"cmp"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// teleporterXMLDir holds the teleporter lists (read recursively, as Mobius
// does).
const teleporterXMLDir = "dist/game/data/teleporters"

// readTeleporters reads the teleport lists of each NPC, sorted by NPC ID,
// with the lists in file order. An <npcs> element gives the same lists to
// more NPCs. Lists of NPCs that are not in npcs are left out.
func readTeleporters(o options, npcs map[int]bool) ([]data.Teleporter, error) {
	files, err := mxFilesUnder(o.src, teleporterXMLDir)
	if err != nil {
		return nil, err
	}
	var (
		teleporters []data.Teleporter
		byNpc       = make(map[int]int) // NPC ID -> index in teleporters
		dropped     = make(map[int]bool)
		errs        []error
	)
	for _, name := range files {
		rel := strings.TrimPrefix(name, teleporterXMLDir+"/")
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		for _, e := range list.children() {
			if e.name != "npc" {
				e.unknown()
				continue
			}
			ids := []int{e.reqInt("id")}
			var lists []data.TeleportList
			for _, c := range e.children() {
				switch c.name {
				case "teleport":
					l := parseTeleportList(c, rel)
					if slices.ContainsFunc(lists, func(x data.TeleportList) bool { return x.Name == l.Name }) {
						c.errorf("duplicate teleport list %q (Mobius keeps the first)", l.Name)
						continue
					}
					lists = append(lists, l)
				case "npcs":
					for _, k := range c.children() {
						if k.name != "npc" {
							k.unknown()
							continue
						}
						ids = append(ids, k.reqInt("id"))
					}
				default:
					c.unknown()
				}
			}
			for _, id := range ids {
				if !npcs[id] {
					dropped[id] = true
					continue
				}
				i, ok := byNpc[id]
				if !ok {
					i = len(teleporters)
					byNpc[id] = i
					teleporters = append(teleporters, data.Teleporter{NpcID: id})
				}
				t := &teleporters[i]
				for _, l := range lists {
					if j := slices.IndexFunc(t.Lists, func(x data.TeleportList) bool { return x.Name == l.Name }); j >= 0 {
						e.errorf("NPC %d: teleport list %q is also defined in %s (Mobius keeps the one it loads last)", id, l.Name, t.Lists[j].File)
						continue
					}
					t.Lists = append(t.Lists, l)
				}
			}
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	if len(dropped) > 0 {
		o.warnf("teleporters: left out the lists of NPCs that are not in the NPC data: %s",
			joinInts(slices.Sorted(maps.Keys(dropped)), 20))
	}
	slices.SortFunc(teleporters, func(a, b data.Teleporter) int { return cmp.Compare(a.NpcID, b.NpcID) })
	return teleporters, nil
}

// parseTeleportList reads one <teleport> list, as Mobius's TeleporterData:
// the name defaults to the type, a location's fee item to adena.
func parseTeleportList(e *mxElem, file string) data.TeleportList {
	typ := e.reqStr("type")
	l := data.TeleportList{Name: e.str("name", typ), Type: typ, File: file}
	for _, c := range e.children() {
		if c.name != "location" {
			c.unknown()
			continue
		}
		loc := data.TeleportLocation{
			Name:      c.str("name", ""),
			X:         c.reqInt("x"),
			Y:         c.reqInt("y"),
			Z:         c.reqInt("z"),
			FeeItemID: c.intAttr("feeId", data.DefaultTeleportFeeItem),
			FeeCount:  c.intAttr("feeCount", 0),
		}
		if v, ok := c.attr("castleId"); ok && v != "" {
			for _, s := range strings.Split(v, ";") {
				id, err := mxInt(s)
				if err != nil {
					c.errorf("castleId: %v", err)
				}
				loc.CastleIDs = append(loc.CastleIDs, id)
			}
		}
		l.Locations = append(l.Locations, loc)
	}
	return l
}
