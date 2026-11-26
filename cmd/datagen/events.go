package main

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// eventsDir holds Mobius's seasonal event scripts, one folder per
// LongTimeEvent class with its config.xml. The custom events
// (dp/scripts/custom/events/) stay out (Plan 14, X8).
const eventsDir = "dist/game/data/scripts/events"

// The events domain converts the config.xml of every seasonal event,
// loaded by data.LoadLongTimeEvents.
func init() { registerDomain("events", genEvents) }

// genEvents reads the event configs, checks them with data.NewLongTimeEvents
// and against the item and NPC data, and adds data.LongTimeEventsFile.
func genEvents(o options) error {
	list, err := readEvents(o)
	if err != nil {
		return fmt.Errorf("events: %w", err)
	}
	events, err := data.NewLongTimeEvents(list)
	if err != nil {
		return err
	}
	if err := checkEventRefs(o, events); err != nil {
		return err
	}
	if err := o.out.addJSON(data.LongTimeEventsFile, list); err != nil {
		return err
	}
	for _, e := range list {
		fmt.Fprintf(o.log, "events: %s (%s): %d drops, %d spawns\n", e.Script, e.Active, len(e.Drops), len(e.Spawns))
	}
	return nil
}

// readEvents reads eventsDir/<Script>/config.xml, sorted by script name.
func readEvents(o options) ([]data.LongTimeEvent, error) {
	files, err := fs.Glob(o.src, eventsDir+"/*/config.xml")
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%s: no event configs found", eventsDir)
	}
	slices.Sort(files)
	list := make([]data.LongTimeEvent, 0, len(files))
	for _, file := range files {
		root, err := readXMLTree(o.src, file)
		if err != nil {
			return nil, err
		}
		e, err := readEvent(root, path.Base(path.Dir(file)))
		if err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, nil
}

// readEvent reads one config.xml as Mobius LongTimeEvent.loadConfig does.
// Where Mobius skips a bad entry with a warning (a wrong number, min above
// max, a chance outside 0-100 or without "%"), datagen fails; so it does on
// elements and attributes Mobius does not read.
func readEvent(root *xmlNode, script string) (data.LongTimeEvent, error) {
	e := data.LongTimeEvent{Script: script}
	if !strings.EqualFold(root.name, "event") {
		return e, root.errorf("want <event>")
	}
	if err := root.checkAttrs("name", "active", "dropPeriod"); err != nil {
		return e, err
	}
	if err := root.checkChildren("droplist", "spawnlist", "messages", "destroyItemsOnEnd"); err != nil {
		return e, err
	}
	var ok bool
	if e.Name, ok = root.attr("name"); !ok {
		return e, root.errorf("no name attribute")
	}
	if e.Active, ok = root.attr("active"); !ok {
		return e, root.errorf("no active attribute")
	}
	e.DropPeriod, _ = root.attr("dropPeriod")
	for _, c := range root.children {
		var err error
		switch c.name {
		case "droplist":
			err = readEventDrops(c, &e)
		case "spawnlist":
			err = readEventSpawns(c, &e)
		case "messages":
			err = readEventMessages(c, &e)
		case "destroyItemsOnEnd":
			err = readEventDestroyItems(c, &e)
		}
		if err != nil {
			return e, err
		}
	}
	return e, nil
}

// readEventDrops reads <droplist><add item min max chance [minLevel]
// [maxLevel] [monsterIds]/>.
func readEventDrops(n *xmlNode, e *data.LongTimeEvent) error {
	if err := n.checkChildren("add"); err != nil {
		return err
	}
	for _, a := range n.children {
		if err := a.checkAttrs("item", "min", "max", "chance", "minLevel", "maxLevel", "monsterIds"); err != nil {
			return err
		}
		d := data.EventDrop{MinLevel: 1, MaxLevel: 2147483647} // Mobius: 1 and Integer.MAX_VALUE
		var err error
		if d.ItemID, err = a.intAttr("item"); err != nil {
			return err
		}
		if d.Min, err = a.intAttr("min"); err != nil {
			return err
		}
		if d.Max, err = a.intAttr("max"); err != nil {
			return err
		}
		chance, _ := a.attr("chance")
		number, percent := strings.CutSuffix(chance, "%")
		if !percent {
			return a.errorf("chance %q: want a percentage such as 4%%", chance)
		}
		if d.Chance, err = parseJavaFloat(number); err != nil {
			return a.errorf("chance: %v", err)
		}
		if _, has := a.attr("minLevel"); has {
			if d.MinLevel, err = a.intAttr("minLevel"); err != nil {
				return err
			}
		}
		if _, has := a.attr("maxLevel"); has {
			if d.MaxLevel, err = a.intAttr("maxLevel"); err != nil {
				return err
			}
		}
		if ids, has := a.attr("monsterIds"); has {
			for id := range strings.SplitSeq(ids, ",") {
				v, err := parseJavaInt(id)
				if err != nil {
					return a.errorf("monsterIds: %v", err)
				}
				if !slices.Contains(d.MonsterIDs, v) { // Mobius keeps a set
					d.MonsterIDs = append(d.MonsterIDs, v)
				}
			}
			slices.Sort(d.MonsterIDs)
		}
		if d.Min > d.Max {
			return a.errorf("item %d: min greater than max (Mobius skips it)", d.ItemID)
		}
		if d.Chance < 0 || d.Chance > 100 {
			return a.errorf("item %d: incorrect drop chance (Mobius skips it)", d.ItemID)
		}
		e.Drops = append(e.Drops, d)
	}
	return nil
}

// readEventSpawns reads <spawnlist><add npc x y z [heading]
// [respawnTime]/>.
func readEventSpawns(n *xmlNode, e *data.LongTimeEvent) error {
	if err := n.checkChildren("add"); err != nil {
		return err
	}
	for _, a := range n.children {
		if err := a.checkAttrs("npc", "x", "y", "z", "heading", "respawnTime"); err != nil {
			return err
		}
		var s data.EventSpawn
		var err error
		for _, f := range []struct {
			name string
			v    *int
			opt  bool
		}{
			{"npc", &s.NpcID, false}, {"x", &s.X, false}, {"y", &s.Y, false}, {"z", &s.Z, false},
			{"heading", &s.Heading, true}, {"respawnTime", &s.RespawnTime, true},
		} {
			if _, has := a.attr(f.name); !has && f.opt {
				continue
			}
			if *f.v, err = a.intAttr(f.name); err != nil {
				return err
			}
		}
		e.Spawns = append(e.Spawns, s)
	}
	return nil
}

// readEventMessages reads <messages><add type="onEnter|onEnd" text/>; a
// later message of a type replaces an earlier one, as in Mobius.
func readEventMessages(n *xmlNode, e *data.LongTimeEvent) error {
	if err := n.checkChildren("add"); err != nil {
		return err
	}
	for _, a := range n.children {
		if err := a.checkAttrs("type", "text"); err != nil {
			return err
		}
		typ, _ := a.attr("type")
		text, ok := a.attr("text")
		if !ok {
			return a.errorf("no text attribute")
		}
		switch {
		case strings.EqualFold(typ, "onEnd"):
			e.OnEnd = text
		case strings.EqualFold(typ, "onEnter"):
			e.OnEnter = text
		default:
			return a.errorf("message type %q: want onEnter or onEnd", typ)
		}
	}
	return nil
}

// readEventDestroyItems reads <destroyItemsOnEnd><item id/>.
func readEventDestroyItems(n *xmlNode, e *data.LongTimeEvent) error {
	if err := n.checkChildren("item"); err != nil {
		return err
	}
	for _, c := range n.children {
		if err := c.checkAttrs("id"); err != nil {
			return err
		}
		id, err := c.intAttr("id")
		if err != nil {
			return err
		}
		e.DestroyItemsOnEnd = append(e.DestroyItemsOnEnd, id)
	}
	return nil
}

// checkEventRefs checks the events' items and NPCs against the item and NPC
// data (Mobius skips unknown ones with a warning).
func checkEventRefs(o options, events *data.LongTimeEvents) error {
	quiet := o
	quiet.log = io.Discard
	items, err := readItems(quiet)
	if err != nil {
		return fmt.Errorf("items: %w", err)
	}
	npcs, err := readNpcs(quiet)
	if err != nil {
		return fmt.Errorf("NPCs: %w", err)
	}
	itemIDs := make(map[int]bool, len(items))
	for _, it := range items {
		itemIDs[it.ID] = true
	}
	npcIDs := make(map[int]bool, len(npcs))
	for _, n := range npcs {
		npcIDs[n.ID] = true
	}
	return events.CheckRefs(func(id int) bool { return itemIDs[id] }, func(id int) bool { return npcIDs[id] })
}
