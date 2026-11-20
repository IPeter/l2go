package data

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func validRoutes() []WalkRoute {
	return []WalkRoute{
		{Name: "a", Repeat: true, RepeatStyle: RepeatCycle,
			Targets: []RouteTarget{{NpcID: 1, X: 1, Y: 2, Z: 3}},
			Nodes:   []RouteNode{{X: 1, Y: 2, Z: 3, Delay: 5, Run: true, Chat: "hi"}}},
		{Name: "b", Repeat: true, RepeatStyle: RepeatRandom,
			Targets: []RouteTarget{{NpcID: 1, X: 4, Y: 5, Z: 6}, {NpcID: 2, X: 1, Y: 2, Z: 3}},
			Nodes:   []RouteNode{{X: 4, Y: 5, Z: 6}}},
	}
}

func TestNewRoutes(t *testing.T) {
	r, err := NewRoutes(validRoutes())
	if err != nil {
		t.Fatal(err)
	}
	a, ok := r.Route("a")
	if !ok || !a.Repeats() {
		t.Fatalf("Route(a) = %+v, %v", a, ok)
	}
	if w := a.Nodes[0].Wait(); w != 5100*time.Millisecond {
		t.Errorf("Wait = %v, want 5.1s", w)
	}
	// Mobius WalkRoute never repeats a random route.
	if b, _ := r.Route("b"); b == nil || b.Repeats() {
		t.Errorf("Route(b) = %+v; want a route that does not repeat", b)
	}
	if _, ok := r.Route("c"); ok {
		t.Error("Route(c) found")
	}
	if got, ok := r.RouteAt(1, 4, 5, 6); !ok || got.Name != "b" {
		t.Errorf("RouteAt(1, 4, 5, 6) = %+v, %v", got, ok)
	}
	if _, ok := r.RouteAt(1, 4, 5, 7); ok {
		t.Error("RouteAt found a route at another point")
	}
	if at := r.RoutesOf(1); len(at) != 2 || at[0].Route.Name != "a" || at[1].Target.X != 4 {
		t.Errorf("RoutesOf(1) = %+v", at)
	}
	if at := r.RoutesOf(9); at != nil {
		t.Errorf("RoutesOf(9) = %+v", at)
	}
}

func TestNewRoutesErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(rs []WalkRoute)
		want   string
	}{
		{"no name", func(rs []WalkRoute) { rs[0].Name = "" }, `route "": no name`},
		{"order", func(rs []WalkRoute) { rs[1].Name = "a" }, `route "a": names are not sorted and unique`},
		{"style", func(rs []WalkRoute) { rs[0].RepeatStyle = "Cycle" }, `unknown repeat style "Cycle"`},
		{"no nodes", func(rs []WalkRoute) { rs[0].Nodes = nil }, `route "a": no nodes`},
		{"delay", func(rs []WalkRoute) { rs[1].Nodes[0].Delay = -1 }, "node 0: negative delay -1"},
		{"target", func(rs []WalkRoute) { rs[1].Targets[1].NpcID = 0 }, "target NPC ID 0 must be positive"},
		{"two routes", func(rs []WalkRoute) { rs[1].Targets[0] = rs[0].Targets[0] }, `NPC 1 at 1, 2, 3 also takes route "a"`},
	} {
		rs := validRoutes()
		tt.change(rs)
		_, err := NewRoutes(rs)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

func TestLoadRoutesFS(t *testing.T) {
	js := `[{"name":"r","repeat":false,"repeatStyle":"back","targets":[],"nodes":[{"x":1,"y":2,"z":3,"delay":0}]}]`
	if r, err := LoadRoutesFS(fstest.MapFS{RoutesFile: {Data: []byte(js)}}); err != nil || len(r.Routes) != 1 {
		t.Fatalf("LoadRoutesFS: %v", err)
	}
	if _, err := LoadRoutesFS(fstest.MapFS{RoutesFile: {Data: []byte(`[{"name":"r","speed":1}]`)}}); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("unknown field: %v", err)
	}
	if _, err := LoadRoutes(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadRoutes of an empty folder: %v", err)
	}
}

// Spot checks of the committed routes against Mobius Routes.xml.
func TestRepoRoutes(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoData, RoutesFile)); err != nil {
		t.Skipf("no route data in %s: %v", repoData, err)
	}
	r, err := LoadRoutes(repoData)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Routes) != 14 {
		t.Errorf("%d routes, want 14", len(r.Routes))
	}
	// <route name="scribe_leandro" repeat="true" repeatStyle="back">
	// <target id="31357" spawnX="-82428" spawnY="245203" spawnZ="-3712" />, 11 points, the 6th:
	// <point string="Where has he gone?" X="-83980" Y="242776" Z="-3728" delay="10" run="false" />
	l, ok := r.RouteAt(31357, -82428, 245203, -3712)
	if !ok || l.Name != "scribe_leandro" || l.RepeatStyle != RepeatBack || !l.Repeats() || len(l.Nodes) != 11 {
		t.Fatalf("RouteAt(31357) = %+v, %v", l, ok)
	}
	if n := l.Nodes[5]; n != (RouteNode{X: -83980, Y: 242776, Z: -3728, Delay: 10, Chat: "Where has he gone?"}) {
		t.Errorf("scribe_leandro node 5 = %+v", n)
	}
	// running_boy has two walkers, Adolph and Edwin.
	if b, ok := r.Route("running_boy"); !ok || len(b.Targets) != 2 || b.Targets[1].NpcID != 32072 || !b.Nodes[0].Run {
		t.Errorf("Route(running_boy) = %+v, %v", b, ok)
	}
	// rune_ghost1b walks once: repeat="false".
	if g, ok := r.Route("rune_ghost1b"); !ok || g.Repeats() || g.Nodes[4].Delay != 15 {
		t.Errorf("Route(rune_ghost1b) = %+v, %v", g, ok)
	}
	// The fake player (80000) is not in the NPC data, so its route has no walker.
	if f, ok := r.Route("FPC_Giran_Evi"); !ok || len(f.Targets) != 0 || len(f.Nodes) != 29 {
		t.Errorf("Route(FPC_Giran_Evi) = %+v, %v", f, ok)
	}
}
