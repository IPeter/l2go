package main

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// The world object lists Mobius's DoorData, StaticObjectData and FenceData
// read.
const (
	doorXML         = "dist/game/data/Doors.xml"
	staticObjectXML = "dist/game/data/StaticObjects.xml"
	fenceXML        = "dist/game/data/FenceData.xml"
)

// The objects domain converts the doors, static objects and fences, loaded
// by data.LoadObjects.
func init() { registerDomain("objects", genObjects) }

// genObjects reads the world objects, checks them with data.NewObjects and
// adds objects/doors.json, objects/staticobjects.json and
// objects/fences.json, one object per line.
func genObjects(o options) error {
	var (
		d   data.ObjectsData
		err error
	)
	if d.Doors, err = readDoors(o); err != nil {
		return fmt.Errorf("doors: %w", err)
	}
	if d.StaticObjects, err = readStaticObjects(o); err != nil {
		return fmt.Errorf("static objects: %w", err)
	}
	if d.Fences, err = readFences(o); err != nil {
		return fmt.Errorf("fences: %w", err)
	}
	objs, err := data.NewObjects(d)
	if err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.DoorsFile, d.Doors); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.StaticObjectsFile, d.StaticObjects); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.FencesFile, d.Fences); err != nil {
		return err
	}
	open, timed := 0, 0
	for i := range d.Doors {
		if d.Doors[i].DefaultOpen {
			open++
		}
		if d.Doors[i].OpensBy(data.DoorOpenByTime) {
			timed++
		}
	}
	fmt.Fprintf(o.log, "objects: %d doors (%d open at start, %d open by time, %d groups), %d static objects, %d fences\n",
		len(d.Doors), open, timed, len(objs.DoorGroups()), len(d.StaticObjects), len(d.Fences))
	return nil
}

// readDoors reads the doors, sorted by ID, as Mobius's DoorData and
// DoorTemplate do. Every attribute must be one Mobius reads; a door ID
// listed twice fails (Mobius would keep the last).
func readDoors(o options) ([]data.Door, error) {
	doc, err := parseMX(o.src, doorXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var (
		doors []data.Door
		seen  = make(map[int]bool)
	)
	for _, e := range root.children() {
		if e.name != "door" {
			e.unknown()
			continue
		}
		d := parseDoor(e)
		if seen[d.ID] {
			e.errorf("door %d is listed twice (Mobius keeps the last)", d.ID)
			continue
		}
		seen[d.ID] = true
		doors = append(doors, d)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	if len(doors) == 0 {
		return nil, fmt.Errorf("%s: no doors", doorXML)
	}
	slices.SortFunc(doors, func(a, b data.Door) int { return cmp.Compare(a.ID, b.ID) })
	return doors, nil
}

// parseDoor reads one <door>.
func parseDoor(e *mxElem) data.Door {
	d := data.Door{
		ID:     e.reqInt("id"),
		Name:   e.reqStr("name"),
		NodeZ:  e.reqInt("nodeZ"),
		Height: e.reqInt("height"),
		// DoorData sets baseHpMax to 1 before the attributes.
		HP:             e.floatAttr("baseHpMax", 1),
		PDef:           e.intAttr("basePDef", 0),
		MDef:           e.intAttr("baseMDef", 0),
		Level:          e.intAttr("level", 0),
		OpenMethod:     e.intAttr("open_method", 0),
		ShowHP:         e.boolAttr("hp_showable", true),
		Wall:           e.boolAttr("is_wall", false),
		Group:          e.str("group", ""),
		Targetable:     e.boolAttr("targetable", true),
		CheckCollision: e.boolAttr("check_collision", true),
		Attackable:     e.boolAttr("is_attackable", false),
		ClanHallID:     e.intAttr("clanhall_id", 0),
		Emitter:        e.intAttr("emitter_id", 0),
		Stealth:        e.boolAttr("stealth", false),
	}
	if xyz := doorInts(e, "pos", 3); xyz != nil {
		d.X, d.Y, d.Z = xyz[0], xyz[1], xyz[2]
	}
	for i := range d.Nodes {
		if xy := doorInts(e, fmt.Sprintf("node%d", i+1), 2); xy != nil {
			d.Nodes[i] = data.Point{X: xy[0], Y: xy[1]}
		}
	}
	switch s := e.str("default_status", "close"); s {
	case "open":
		d.DefaultOpen = true
	case "close":
	default:
		// Mobius takes anything but "open" as closed.
		e.errorf("attribute default_status: %q is not open or close", s)
	}
	if v, ok := e.attr("child_id_event"); ok {
		id, err := mxInt(v)
		switch {
		case err != nil:
			e.errorf("attribute child_id_event: %v", err)
		case id <= 0:
			e.errorf("attribute child_id_event: %d is not a door ID (Mobius default -1: none)", id)
		}
		d.ChildID = id
	}
	d.MasterOpen = doorAction(e, "master_open_event")
	d.MasterClose = doorAction(e, "master_close_event")
	if v, ok := e.attr("close_time"); ok {
		t, err := mxInt(v)
		switch {
		case err != nil:
			e.errorf("attribute close_time: %v", err)
		case t <= 0:
			// 0 would close at once, a negative time means none; the data
			// format keeps 0 for none.
			e.errorf("attribute close_time: %d is not positive", t)
		}
		d.CloseTime = t
	}
	if d.OpensBy(data.DoorOpenByTime) {
		// DoorTemplate requires open_time on these doors; random_time
		// counts only above 0.
		d.OpenTime = e.reqInt("open_time")
		d.RandomTime = max(0, e.intAttr("random_time", 0))
	} else {
		for _, name := range []string{"open_time", "random_time"} {
			if _, ok := e.attr(name); ok {
				e.errorf("attribute %s on a door that does not open by time (Mobius ignores it)", name)
			}
		}
	}
	return d
}

// doorInts reads a door attribute of n comma-separated integers, such as
// pos="x,y,z" (Mobius: Integer.parseInt of each part).
func doorInts(e *mxElem, name string, n int) []int {
	s := e.reqStr(name)
	parts := strings.Split(s, ",")
	if len(parts) != n {
		e.errorf("attribute %s: %q does not have %d values", name, s, n)
		return nil
	}
	vals := make([]int, n)
	for i, p := range parts {
		v, err := parseJavaInt(p)
		if err != nil {
			e.errorf("attribute %s: %v", name, err)
			return nil
		}
		vals[i] = v
	}
	return vals
}

// doorAction reads master_open_event or master_close_event: act_open,
// act_close or act_nothing (the default). Mobius takes any other value as
// act_nothing; datagen refuses it.
func doorAction(e *mxElem, name string) data.DoorAction {
	switch v := e.str(name, "act_nothing"); v {
	case "act_nothing":
		return ""
	case string(data.DoorActOpen), string(data.DoorActClose):
		return data.DoorAction(v)
	default:
		e.errorf("attribute %s: unknown action %q", name, v)
		return ""
	}
}

// readStaticObjects reads the static objects, sorted by ID, as Mobius's
// StaticObjectData does.
func readStaticObjects(o options) ([]data.StaticObject, error) {
	doc, err := parseMX(o.src, staticObjectXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var (
		objs []data.StaticObject
		seen = make(map[int]bool)
	)
	for _, e := range root.children() {
		if e.name != "object" {
			e.unknown()
			continue
		}
		s := data.StaticObject{
			ID:   e.reqInt("id"),
			Name: e.reqStr("name"),
			Type: e.intAttr("type", data.StaticTownMap),
			X:    e.reqInt("x"),
			Y:    e.reqInt("y"),
			Z:    e.reqInt("z"),
			MapX: e.intAttr("map_x", 0),
			MapY: e.intAttr("map_y", 0),
		}
		if t := e.str("texture", "none"); t != "none" {
			s.Texture = t
		}
		if seen[s.ID] {
			e.errorf("static object %d is listed twice", s.ID)
			continue
		}
		seen[s.ID] = true
		objs = append(objs, s)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	slices.SortFunc(objs, func(a, b data.StaticObject) int { return cmp.Compare(a.ID, b.ID) })
	return objs, nil
}

// readFences reads the fences in file order, as Mobius's FenceData does.
func readFences(o options) ([]data.Fence, error) {
	doc, err := parseMX(o.src, fenceXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	fences := []data.Fence{}
	for _, e := range root.children() {
		if e.name != "fence" {
			e.unknown()
			continue
		}
		f := data.Fence{
			Name:   e.reqStr("name"),
			X:      e.reqInt("x"),
			Y:      e.reqInt("y"),
			Z:      e.reqInt("z"),
			Width:  e.reqInt("width"),
			Length: e.reqInt("length"),
			Height: e.reqInt("height"),
			State:  e.str("state", data.FenceClosed),
		}
		switch f.State {
		case data.FenceHidden, data.FenceOpened, data.FenceClosed, data.FenceClosedHidden:
		default:
			e.errorf("attribute state: unknown fence state %q", f.State)
		}
		fences = append(fences, f)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	return fences, nil
}
