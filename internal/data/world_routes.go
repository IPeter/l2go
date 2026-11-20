package data

// Walker routes: the routes of Mobius Routes.xml that walking NPCs follow
// (WalkingManager), converted by cmd/datagen and loaded with LoadRoutes.

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"time"
)

// RoutesFile is the path of the walker route file, relative to the data
// folder.
const RoutesFile = "routes/routes.json"

// The repeat styles of a route (WalkRoute.RepeatStyle), the repeatStyle
// values Mobius knows.
const (
	// RepeatBack walks from the first node to the last, then back.
	RepeatBack = "back"
	// RepeatCycle walks from the last node straight to the first.
	RepeatCycle = "cycle"
	// RepeatConveyor teleports from the last node to the first.
	RepeatConveyor = "conveyor"
	// RepeatRandom walks to random nodes.
	RepeatRandom = "random"
)

var repeatStyles = setOf(RepeatBack, RepeatCycle, RepeatConveyor, RepeatRandom)

// WalkRoute is one <route> of Routes.xml.
type WalkRoute struct {
	Name string `json:"name"`
	// Repeat is the repeat attribute; RepeatStyle a Repeat* value (lower
	// case, as Mobius reads it). See Repeats.
	Repeat      bool   `json:"repeat"`
	RepeatStyle string `json:"repeatStyle"`
	// Targets are the NPCs that walk the route when they spawn at the
	// target's point. Targets whose NPC is not in the NPC data are left out,
	// as Mobius leaves them out.
	Targets []RouteTarget `json:"targets"`
	// Nodes are the points of the route, in order; there is at least one.
	Nodes []RouteNode `json:"nodes"`
}

// Repeats reports whether an NPC walks the route again after its last
// node, as Mobius WalkRoute.repeatWalk: the repeat flag is set and the
// style is back, cycle or conveyor. (A random route never repeats in
// Mobius; no Interlude route is random.)
func (r *WalkRoute) Repeats() bool {
	return r.Repeat && (r.RepeatStyle == RepeatBack || r.RepeatStyle == RepeatCycle || r.RepeatStyle == RepeatConveyor)
}

// RouteTarget is one <target> of a route: an NPC that takes the route when
// it spawns at this point.
type RouteTarget struct {
	NpcID int `json:"npcId"`
	X     int `json:"x"`
	Y     int `json:"y"`
	Z     int `json:"z"`
}

// RouteNode is one <point> of a route.
type RouteNode struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
	// Delay is how long, in seconds, the NPC stays at the node (see Wait).
	Delay int `json:"delay"`
	// Run: the NPC runs to the node instead of walking.
	Run bool `json:"run,omitempty"`
	// Chat is what the NPC says (NPC_GENERAL) when it arrives.
	Chat string `json:"chat,omitempty"`
}

// Wait is how long the NPC waits at the node before it walks on: Mobius
// schedules the next step 100 + Delay × 1000 ms after the arrival.
func (n *RouteNode) Wait() time.Duration {
	return 100*time.Millisecond + time.Duration(n.Delay)*time.Second
}

// Routes is the checked walker route data. It must not be modified after
// NewRoutes or LoadRoutes returns, and is then safe for concurrent use.
type Routes struct {
	// Routes are sorted by name.
	Routes []WalkRoute
}

// LoadRoutes reads and checks the walker route file in dir.
func LoadRoutes(dir string) (*Routes, error) {
	r, err := LoadRoutesFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return r, nil
}

// LoadRoutesFS reads and checks the walker route file in fsys.
func LoadRoutesFS(fsys fs.FS) (*Routes, error) {
	var routes []WalkRoute
	if err := readJSON(fsys, RoutesFile, &routes); err != nil {
		return nil, err
	}
	return NewRoutes(routes)
}

// NewRoutes checks the routes: names present, unique and sorted, a known
// repeat style, at least one node, delays not negative, targets with an
// NPC ID, and no NPC and point that select two routes.
func NewRoutes(routes []WalkRoute) (*Routes, error) {
	var p problems
	targets := make(map[RouteTarget]string) // target -> route name
	for i, r := range routes {
		for _, t := range r.Targets {
			if other, dup := targets[t]; dup {
				p.addf("%s: route %q: NPC %d at %d, %d, %d also takes route %q", RoutesFile, r.Name, t.NpcID, t.X, t.Y, t.Z, other)
			}
			targets[t] = r.Name
		}
		where := fmt.Sprintf("%s: route %q", RoutesFile, r.Name)
		if r.Name == "" {
			p.addf("%s: no name", where)
		}
		if i > 0 && routes[i-1].Name >= r.Name {
			p.addf("%s: names are not sorted and unique", where)
		}
		if !repeatStyles[r.RepeatStyle] {
			p.addf("%s: unknown repeat style %q", where, r.RepeatStyle)
		}
		if len(r.Nodes) == 0 {
			p.addf("%s: no nodes", where)
		}
		for j, n := range r.Nodes {
			if n.Delay < 0 {
				p.addf("%s: node %d: negative delay %d", where, j, n.Delay)
			}
		}
		for _, t := range r.Targets {
			if t.NpcID <= 0 {
				p.addf("%s: target NPC ID %d must be positive", where, t.NpcID)
			}
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return &Routes{Routes: routes}, nil
}

// Route returns the route with the given name.
func (r *Routes) Route(name string) (*WalkRoute, bool) {
	i, ok := slices.BinarySearchFunc(r.Routes, name, func(w WalkRoute, name string) int { return cmp.Compare(w.Name, name) })
	if !ok {
		return nil, false
	}
	return &r.Routes[i], true
}

// RoutesOf returns the routes an NPC takes, by spawn point, in route name
// order; Mobius attaches the route whose target point is the NPC's spawn
// point.
func (r *Routes) RoutesOf(npcID int) []RouteAttachment {
	var out []RouteAttachment
	for i := range r.Routes {
		for _, t := range r.Routes[i].Targets {
			if t.NpcID == npcID {
				out = append(out, RouteAttachment{Route: &r.Routes[i], Target: t})
			}
		}
	}
	return out
}

// RouteAt returns the route an NPC spawned at x, y, z takes, as Mobius
// NpcRoutesHolder.getRouteName: the route with a target of that NPC at
// exactly that point.
func (r *Routes) RouteAt(npcID, x, y, z int) (*WalkRoute, bool) {
	for i := range r.Routes {
		for _, t := range r.Routes[i].Targets {
			if t.NpcID == npcID && t.X == x && t.Y == y && t.Z == z {
				return &r.Routes[i], true
			}
		}
	}
	return nil, false
}

// RouteAttachment is a route an NPC takes, with the spawn point that
// selects it.
type RouteAttachment struct {
	Route  *WalkRoute
	Target RouteTarget
}
