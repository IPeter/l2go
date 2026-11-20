package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// writeRoutes writes a routes file into a new data folder and returns the
// folder.
func writeRoutes(t *testing.T, routes ...data.WalkRoute) string {
	t.Helper()
	if _, err := data.NewRoutes(routes); err != nil {
		t.Fatalf("NewRoutes: %v", err)
	}
	b, err := json.Marshal(routes)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(data.RoutesFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// walkerState is what the walker tests read of an NPC.
type walkerState struct {
	pos     world.Position
	dest    world.Position
	moving  bool
	running bool
	cursor  int
}

func walkerStateOf(e *testEnv, n *Npc) walkerState {
	var st walkerState
	e.onLoop(func() {
		st = walkerState{pos: n.Position(), moving: n.Move != nil, running: n.Running(), cursor: -99}
		if n.Move != nil {
			st.dest = n.Move.Destination
		}
		if n.walker != nil && n.walker.info != nil {
			st.cursor = n.walker.info.cursor
		}
	})
	return st
}

func samePoint(a world.Position, x, y int) bool { return a.X == int32(x) && a.Y == int32(y) }

// Lector walks his route: from his spawn point, the first node, to the
// second at a run, on to the third at a walk, says its line there, waits
// 100 ms + 2 s, and walks back (repeat style back).
func TestWalkerRoute(t *testing.T) {
	start := npcTestStart(t)
	a := data.RouteNode{X: start.X + 1000, Y: start.Y, Z: start.Z}
	b := data.RouteNode{X: a.X + 300, Y: a.Y, Z: a.Z, Run: true}
	c := data.RouteNode{X: b.X, Y: b.Y + 300, Z: a.Z, Delay: 2, Chat: "Where has he gone?"}
	dir := writeRoutes(t, data.WalkRoute{
		Name: "lector", Repeat: true, RepeatStyle: data.RepeatBack,
		Targets: []data.RouteTarget{{NpcID: npcTestLector, X: a.X, Y: a.Y, Z: a.Z}},
		Nodes:   []data.RouteNode{a, b, c},
	})
	e := newNpcEnv(t, func(gs *config.GameServer) { gs.DataDir = dir })
	lector := npcSpawnedOne(t, e, npcTestLector)

	st := walkerStateOf(e, lector)
	if !st.moving || !samePoint(st.dest, b.X, b.Y) || !st.running || st.cursor != 1 {
		t.Fatalf("at the start: %+v; want Lector running to the second node", st)
	}
	e.createCharacter("alice", "Alice", classHumanFighter, 0)
	sess := newSession("alice")
	e.enter(sess, 0)
	sess.take(t)
	// 300 at 120 a second, less the collision radius.
	e.advance(2500 * time.Millisecond)
	if st = walkerStateOf(e, lector); st.moving || !samePoint(st.pos, b.X, b.Y) || st.cursor != 2 {
		t.Fatalf("after 2.5 s: %+v; want Lector waiting at the second node", st)
	}
	e.advance(100 * time.Millisecond)
	if st = walkerStateOf(e, lector); !st.moving || !samePoint(st.dest, c.X, c.Y) || st.running {
		t.Fatalf("100 ms later: %+v; want Lector walking to the third node", st)
	}
	sess.take(t)
	for i := 0; i < 70 && walkerStateOf(e, lector).moving; i++ {
		e.advance(100 * time.Millisecond)
	}
	if st = walkerStateOf(e, lector); st.moving || !samePoint(st.pos, c.X, c.Y) {
		t.Fatalf("Lector did not reach the third node: %+v", st)
	}
	// Past the last node the route goes back: the cursor is on the second.
	if st.cursor != 1 {
		t.Errorf("cursor at the end %d, want 1 (going back)", st.cursor)
	}
	says := only[testclient.NpcSay](sess.take(t))
	if len(says) != 1 || says[0].Text != c.Chat || says[0].ObjectID != int32(lector.ID()) {
		t.Errorf("at the third node Alice heard %+v; want %q", says, c.Chat)
	}
	e.advance(2 * time.Second)
	if st = walkerStateOf(e, lector); st.moving {
		t.Fatalf("Lector left before 2.1 s: %+v", st)
	}
	e.advance(100 * time.Millisecond)
	if st = walkerStateOf(e, lector); !st.moving || !samePoint(st.dest, b.X, b.Y) || !st.running {
		t.Fatalf("after 2.1 s: %+v; want Lector running back to the second node", st)
	}
}

// The repeat styles step the cursor as Mobius WalkInfo does.
func TestWalkerNextNode(t *testing.T) {
	e := newNpcEnv(t, nil)
	lector := npcSpawnedOne(t, e, npcTestLector)
	nodes := []data.RouteNode{{X: 1}, {X: 2}, {X: 3}}
	for _, tt := range []struct {
		style   string
		repeat  bool
		visits  []int // cursors after each node, from cursor 0
		running bool  // the walk still runs at the end
	}{
		{data.RepeatBack, true, []int{1, 2, 1, 0, 1, 2, 1}, true},
		{data.RepeatCycle, true, []int{1, 2, 0, 1, 2, 0}, true},
		{data.RepeatCycle, false, []int{1, 2, 3}, false},
	} {
		var got []int
		var active bool
		e.onLoop(func() {
			route := &data.WalkRoute{Name: "r", Repeat: tt.repeat, RepeatStyle: tt.style, Nodes: nodes}
			info := &walkInfo{route: route, step: 1}
			lector.walker = &npcWalker{info: info}
			for range tt.visits {
				e.srv.walkerNext(lector, info)
				got = append(got, info.cursor)
			}
			active = lector.walker.info != nil
			lector.walker = nil
		})
		if len(got) != len(tt.visits) {
			t.Fatalf("%s: %v", tt.style, got)
		}
		for i := range got {
			if got[i] != tt.visits[i] {
				t.Errorf("%s repeat %v: cursors %v, want %v", tt.style, tt.repeat, got, tt.visits)
				break
			}
		}
		if active != tt.running {
			t.Errorf("%s repeat %v: walk still running %v, want %v", tt.style, tt.repeat, active, tt.running)
		}
	}

	// A random route never picks the node it is at.
	e.onLoop(func() {
		info := &walkInfo{route: &data.WalkRoute{Name: "r", Repeat: true, RepeatStyle: data.RepeatRandom, Nodes: nodes}, step: 1}
		lector.walker = &npcWalker{info: info}
		for range 50 {
			before := info.cursor
			e.srv.walkerNext(lector, info)
			if info.cursor == before || info.cursor < 0 || info.cursor >= len(nodes) {
				t.Errorf("random: %d after %d", info.cursor, before)
			}
		}
		lector.walker = nil
	})
}

// A walker that fights when its route would start tries again every 10 s
// (Mobius's start retry), and a monster that walks a route never wanders
// home.
func TestWalkerStartRetry(t *testing.T) {
	start := npcTestStart(t)
	e := newNpcEnv(t, nil)
	friendly := npcSpawnedOne(t, e, npcTestFriendly)
	at := world.Position{X: int32(start.X), Y: int32(start.Y) - 300, Z: int32(start.Z)}
	far := data.RouteNode{X: start.X + 500, Y: start.Y - 300, Z: start.Z}
	routes, err := data.NewRoutes([]data.WalkRoute{{
		Name: "friendly", Repeat: true, RepeatStyle: data.RepeatCycle,
		Targets: []data.RouteTarget{{NpcID: npcTestFriendly, X: int(at.X), Y: int(at.Y), Z: int(at.Z)}},
		Nodes:   []data.RouteNode{far, {X: start.X, Y: start.Y - 300, Z: start.Z}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	e.onLoop(func() {
		e.srv.aiOf(friendly).intention = NpcAttack
		e.srv.UseWalkerRoutes(routes)
	})
	if st := walkerStateOf(e, friendly); st.moving || st.cursor != -99 {
		t.Fatalf("a fighting walker started: %+v", st)
	}
	e.advance(walkerRetry)
	if st := walkerStateOf(e, friendly); st.moving || st.cursor != -99 {
		t.Fatalf("still fighting after 10 s, it started: %+v", st)
	}
	e.onLoop(func() { friendly.ai.intention = NpcActive })
	e.advance(walkerRetry)
	st := walkerStateOf(e, friendly)
	if !st.moving || !samePoint(st.dest, far.X, far.Y) {
		t.Fatalf("20 s on, done fighting: %+v; want it walking its route", st)
	}
	var randomWalk, walker bool
	e.onLoop(func() { randomWalk, walker = e.srv.aiClassOf(friendly).randomWalk, friendly.isWalker() })
	if randomWalk || !walker {
		t.Errorf("random walk %v, walker %v; want false, true", randomWalk, walker)
	}
}

// A walker that leaves the world stops walking; it starts over when it
// spawns again.
func TestWalkerGoneAndBack(t *testing.T) {
	start := npcTestStart(t)
	a := data.RouteNode{X: start.X + 1000, Y: start.Y, Z: start.Z}
	dir := writeRoutes(t, data.WalkRoute{
		Name: "lector", Repeat: true, RepeatStyle: data.RepeatCycle,
		Targets: []data.RouteTarget{{NpcID: npcTestLector, X: a.X, Y: a.Y, Z: a.Z}},
		Nodes:   []data.RouteNode{a, {X: a.X + 2000, Y: a.Y, Z: a.Z}},
	})
	e := newNpcEnv(t, func(gs *config.GameServer) { gs.DataDir = dir })
	lector := npcSpawnedOne(t, e, npcTestLector)
	e.advance(time.Second)
	e.onLoop(func() { e.srv.DespawnNpc(lector) })
	var info *walkInfo
	e.onLoop(func() { info = lector.walker.info })
	if info != nil {
		t.Fatal("the walk goes on after the NPC left the world")
	}
	e.onLoop(func() { e.srv.respawnSlotNow(lector.spawn) })
	if st := walkerStateOf(e, lector); !st.moving || st.cursor != 1 || !samePoint(st.pos, a.X, a.Y) {
		t.Errorf("after the respawn: %+v; want a new walk from the spawn point", st)
	}
}
