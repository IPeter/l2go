package main

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// routeXML is the walker route list Mobius's WalkingManager reads.
const routeXML = "dist/game/data/Routes.xml"

// The routes domain converts the walker routes, loaded by data.LoadRoutes.
func init() { registerDomain("routes", genRoutes) }

// genRoutes reads the routes, checks them with data.NewRoutes and adds
// routes/routes.json, one route per line.
func genRoutes(o options) error {
	routes, err := readRoutes(o)
	if err != nil {
		return fmt.Errorf("routes: %w", err)
	}
	if _, err := data.NewRoutes(routes); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.RoutesFile, routes); err != nil {
		return err
	}
	nodes, targets := 0, 0
	for _, r := range routes {
		nodes += len(r.Nodes)
		targets += len(r.Targets)
	}
	fmt.Fprintf(o.log, "routes: %d walker routes with %d nodes and %d walkers\n", len(routes), nodes, targets)
	return nil
}

// readRoutes reads the routes, sorted by name, as Mobius's WalkingManager
// does. Targets whose NPC is not in the NPC data are left out with a
// warning, as Mobius leaves them out. A route name used twice fails (Mobius
// keeps the last); so does a repeatStyle Mobius does not know (it would
// not repeat) and the npcString and npcStringId attributes, which the
// Interlude WalkingManager does not read.
func readRoutes(o options) ([]data.WalkRoute, error) {
	doc, err := parseMX(o.src, routeXML)
	if err != nil {
		return nil, err
	}
	if doc.root.name != "routes" {
		return nil, fmt.Errorf("%s: root element is <%s>, want <routes>", routeXML, doc.root.name)
	}
	quiet := o
	quiet.log = io.Discard
	npcs, err := readNpcs(quiet)
	if err != nil {
		return nil, fmt.Errorf("NPCs: %w", err)
	}
	known := make(map[int]bool, len(npcs))
	for _, n := range npcs {
		known[n.ID] = true
	}
	var (
		routes  []data.WalkRoute
		seen    = make(map[string]bool)
		missing []string
	)
	for _, e := range doc.root.children() {
		if e.name != "route" {
			e.unknown()
			continue
		}
		r := data.WalkRoute{
			Name:        e.reqStr("name"),
			Repeat:      routeBool(e, "repeat"),
			RepeatStyle: strings.ToLower(e.reqStr("repeatStyle")),
			Targets:     []data.RouteTarget{},
		}
		switch r.RepeatStyle {
		case data.RepeatBack, data.RepeatCycle, data.RepeatConveyor, data.RepeatRandom:
		default:
			e.errorf("unknown repeatStyle %q (Mobius would not repeat the route)", r.RepeatStyle)
		}
		for _, c := range e.children() {
			switch c.name {
			case "point":
				n := data.RouteNode{
					X:     c.reqInt("X"),
					Y:     c.reqInt("Y"),
					Z:     c.reqInt("Z"),
					Delay: c.reqInt("delay"),
					Run:   routeBool(c, "run"),
					Chat:  c.str("string", ""),
				}
				r.Nodes = append(r.Nodes, n)
			case "target":
				t := data.RouteTarget{
					NpcID: routeInt(c, "id"),
					X:     routeInt(c, "spawnX"),
					Y:     routeInt(c, "spawnY"),
					Z:     routeInt(c, "spawnZ"),
				}
				if !known[t.NpcID] {
					missing = append(missing, fmt.Sprintf("%d (%s)", t.NpcID, r.Name))
					continue
				}
				r.Targets = append(r.Targets, t)
			default:
				c.unknown()
			}
		}
		if seen[r.Name] {
			e.errorf("route %q is listed twice (Mobius keeps the last)", r.Name)
			continue
		}
		seen[r.Name] = true
		routes = append(routes, r)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	if len(missing) > 0 {
		o.warnf("routes: left out walkers that are not in the NPC data (Mobius skips them): %s", strings.Join(missing, ", "))
	}
	slices.SortFunc(routes, func(a, b data.WalkRoute) int { return cmp.Compare(a.Name, b.Name) })
	return routes, nil
}

// routeBool reads a required boolean attribute (Mobius's parseBoolean
// returns null for a missing one, which fails when unboxed).
func routeBool(e *mxElem, name string) bool {
	if _, ok := e.attr(name); !ok {
		e.errorf("missing attribute %s", name)
		return false
	}
	return e.boolAttr(name, false)
}

// routeInt reads a required attribute of a route <target>, which Mobius
// parses with Integer.parseInt (a bad one drops the target with a
// warning there; datagen fails).
func routeInt(e *mxElem, name string) int {
	v, ok := e.attr(name)
	if !ok {
		e.errorf("missing attribute %s", name)
		return 0
	}
	n, err := parseJavaInt(v)
	if err != nil {
		e.errorf("attribute %s: %v", name, err)
	}
	return n
}
