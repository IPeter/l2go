package game

import (
	"math"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/geo/geotest"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// walkInts encodes a client packet body of ints.
func walkInts(vs ...int32) []byte {
	w := packets.NewWriter(4 * len(vs))
	for _, v := range vs {
		w.WriteInt32(v)
	}
	return w.Bytes()
}

// walkRequest encodes the client MoveToLocation: the target, the origin the
// client shows and the mode (1 mouse, 0 arrow keys).
func walkRequest(target, origin world.Position, keyboard bool) []byte {
	mode := int32(clientpackets.MoveModeMouse)
	if keyboard {
		mode = clientpackets.MoveModeKeyboard
	}
	return walkInts(target.X, target.Y, target.Z, origin.X, origin.Y, origin.Z, mode)
}

// walkClick sends a mouse click on target, the client showing the player at
// from, and waits for the loop.
func walkClick(e *testEnv, sess *fakeSession, from, target world.Position) {
	e.srv.Handle(sess, clientpackets.OpMoveToLocation, walkRequest(target, from, false))
	e.sync()
}

// walkReport sends ValidatePosition at pos and waits for the loop.
func walkReport(e *testEnv, sess *fakeSession, pos world.Position) {
	e.srv.Handle(sess, clientpackets.OpValidatePosition, walkInts(pos.X, pos.Y, pos.Z, pos.Heading, 0))
	e.sync()
}

// walkSettle lets the packets the enter-world sequence sends later go out,
// and drops them.
func walkSettle(t *testing.T, e *testEnv, sessions ...*fakeSession) {
	t.Helper()
	e.advance(time.Second)
	for _, s := range sessions {
		s.take(t)
	}
}

// walkOffset returns pos moved by (dx, dy, dz), heading 0.
func walkOffset(pos world.Position, dx, dy, dz int32) world.Position {
	return world.Position{X: pos.X + dx, Y: pos.Y + dy, Z: pos.Z + dz}
}

// walkTC is a position as the test client has it.
func walkTC(pos world.Position) testclient.Position {
	return testclient.Position{X: pos.X, Y: pos.Y, Z: pos.Z}
}

// walkFor lets d pass in steps of ten seconds. The loop runs at most 100
// missed ticks back to back and drops older ones, and a walk then jumps
// ahead by the dropped ticks (its step counts the game ticks since the last
// update, as Mobius's does), skipping the region changes on the way.
func walkFor(e *testEnv, d time.Duration) {
	e.t.Helper()
	for d > 0 {
		step := min(d, 10*time.Second)
		e.advance(step)
		d -= step
	}
}

// walkState reads on the loop whether p walks, and where it is.
func walkState(e *testEnv, p *Player) (walking bool, pos world.Position) {
	e.onLoop(func() { walking, pos = p.Move != nil, p.Position() })
	return walking, pos
}

// A click starts a walk that p and the players around see; the server moves
// p along with Mobius's math (TestWalkStepMatchesMobius: 300 units take 22
// game ticks) and everyone sees the walk end at its destination.
func TestWalkClickAndArrival(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	dest := walkOffset(home, 300, 0, 0)

	walkClick(e, sa, home, dest)
	ps := sa.take(t)
	checkNames(t, "Alice after her click", ps, "MoveToLocation")
	start := ps[0].(testclient.MoveToLocation)
	if start.ObjectID != pa.ObjectID() || start.Origin() != walkTC(home) || start.Destination() != walkTC(dest) {
		t.Errorf("MoveToLocation %+v, want Alice from %v to %v", start, home, dest)
	}
	pbs := sb.take(t)
	checkNames(t, "Bob when Alice walks", pbs, "MoveToLocation")
	if pbs[0].(testclient.MoveToLocation) != start {
		t.Errorf("Bob got %+v, want %+v", pbs[0], start)
	}
	var walk MoveState
	e.onLoop(func() { walk = *pa.Move })
	if walk.Origin != home || walk.Destination.X != dest.X || walk.Destination.Y != dest.Y || walk.Destination.Z != dest.Z ||
		!walk.Running || !walk.Start.Equal(testStart.Add(time.Second)) || math.Abs(walk.Speed-walkTestSpeed) > 1e-9 {
		t.Errorf("MoveState %+v", walk)
	}

	e.advance(21 * world.GameTick)
	walking, pos := walkState(e, pa)
	if !walking || pos.X <= home.X || pos.X >= dest.X || pos.Y != home.Y || pos.Z != home.Z {
		t.Fatalf("after 21 ticks: walking %v at %v, want on the way", walking, pos)
	}
	if ps := sa.take(t); len(ps) != 0 {
		t.Errorf("Alice got %q while walking", testclient.Names(ps))
	}

	e.advance(world.GameTick)
	walking, pos = walkState(e, pa)
	if walking || pos.X != dest.X || pos.Y != dest.Y || pos.Z != dest.Z {
		t.Fatalf("after 22 ticks: walking %v at %v, want standing on %v", walking, pos, dest)
	}
	ps = sa.take(t)
	checkNames(t, "Alice on arrival", ps, "MoveToLocation")
	if m := ps[0].(testclient.MoveToLocation); m.Origin() != walkTC(dest) || m.Destination() != walkTC(dest) {
		t.Errorf("arrival %+v, want from and to %v", m, dest)
	}
	checkNames(t, "Bob on Alice's arrival", sb.take(t), "MoveToLocation")
}

// Walking two regions away takes the players out of each other's sight;
// walking back brings them in again, and Bob sees Alice walking.
func TestWalkOutOfSightAndBack(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, pb := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	// home is 342 units into its region (x -71680 to -69632); far is in the
	// region two to the east (from x -67584).
	far := walkOffset(home, 5000, 0, 0)

	walkClick(e, sa, home, far)
	walkFor(e, time.Minute)
	pas := sa.take(t)
	checkNames(t, "Alice walking away", pas, "MoveToLocation", "DeleteObject", "MoveToLocation")
	if d := pas[1].(testclient.DeleteObject); d.ObjectID != pb.ObjectID() {
		t.Errorf("Alice lost %d, want Bob", d.ObjectID)
	}
	pbs := sb.take(t)
	checkNames(t, "Bob as Alice walks away", pbs, "MoveToLocation", "DeleteObject")
	if d := pbs[1].(testclient.DeleteObject); d.ObjectID != pa.ObjectID() {
		t.Errorf("Bob lost %d, want Alice", d.ObjectID)
	}
	if walking, pos := walkState(e, pa); walking || pos.X != far.X {
		t.Fatalf("walking %v at %v, want standing on %v", walking, pos, far)
	}

	walkClick(e, sa, far, home)
	walkFor(e, time.Minute)
	checkNames(t, "Alice walking back", sa.take(t), "MoveToLocation", "CharInfo", "RelationChanged", "MoveToLocation")
	pbs = sb.take(t)
	checkNames(t, "Bob as Alice comes back", pbs, "CharInfo", "RelationChanged", "MoveToLocation", "MoveToLocation")
	// Alice came into sight walking: in the region next to Bob's, on her way
	// home.
	if m := pbs[2].(testclient.MoveToLocation); m.ObjectID != pa.ObjectID() || m.Destination() != walkTC(home) ||
		m.X >= -67584 || m.X <= home.X {
		t.Errorf("Bob saw Alice walk %+v, want from x below -67584 to %v", m, home)
	}
	if m := pbs[3].(testclient.MoveToLocation); m.Origin() != walkTC(home) || m.Destination() != walkTC(home) {
		t.Errorf("Alice's arrival %+v, want at %v", m, home)
	}
}

// Mobius refuses a click on a target more than 9,900 units away and a
// click while teleporting; a click on the client's own origin stops the
// player where the server has it.
func TestWalkRefusals(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)

	walkClick(e, sa, home, walkOffset(home, 9901, 0, 0))
	checkNames(t, "a click 9,901 units away", sa.take(t), "ActionFailed")
	checkNames(t, "Bob", sb.take(t))

	walkClick(e, sa, home, walkOffset(home, 9900, 0, 0))
	checkNames(t, "a click 9,900 units away", sa.take(t), "MoveToLocation")
	sb.take(t)

	e.advance(world.GameTick)
	_, here := walkState(e, pa)
	walkClick(e, sa, here, here)
	ps := sa.take(t)
	checkNames(t, "a click on the origin", ps, "StopMove", "ActionFailed")
	if s := ps[0].(testclient.StopMove); s.ObjectID != pa.ObjectID() || s.Position() != walkTC(here) {
		t.Errorf("StopMove %+v, want Alice at %v", s, here)
	}
	checkNames(t, "Bob sees the stop", sb.take(t), "StopMove")
	if walking, _ := walkState(e, pa); walking {
		t.Error("still walking after the stop")
	}

	e.onLoop(func() {
		if err := e.srv.Teleport(pa, gludio); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	sa.take(t)
	walkClick(e, sa, here, walkOffset(here, 100, 0, 0))
	checkNames(t, "a click while teleporting", sa.take(t), "ActionFailed")
}

// ValidatePosition follows Mobius's policy step by step (a Human Fighter
// runs 126.5 units a second).
func TestValidatePositionPolicy(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	at := func(dx, dz int32) world.Position { return walkOffset(home, dx, 0, dz) }

	for _, step := range []struct {
		name      string
		report    world.Position
		want      []string
		corrected world.Position // the ValidateLocation, if any
		server    world.Position
	}{
		{"100 away: closer than the speed, nothing changes", at(100, 0), nil, world.Position{}, at(0, 0)},
		{"200 away: farther than the speed, the client's position", at(200, 0), nil, world.Position{}, at(200, 0)},
		{"550 away: a correction, then the client's position", at(750, 0), []string{"ValidateLocation"}, at(200, 0), at(750, 0)},
		{"700 away: the client's position", at(1450, 0), nil, world.Position{}, at(1450, 0)},
		{"300 higher, close to the last report: the client's height", at(1450, 300), nil, world.Position{}, at(1450, 300)},
		{"1,600 higher: a correction, then the client's position", at(1450, 1900), []string{"ValidateLocation"}, at(1450, 300), at(1450, 1900)},
		{"150 lower: without geodata the server keeps its height", at(1450, 1750), nil, world.Position{}, at(1450, 1900)},
		{"a report of (0, 0) is ignored", world.Position{}, nil, world.Position{}, at(1450, 1900)},
	} {
		walkReport(e, sa, step.report)
		ps := sa.take(t)
		checkNames(t, step.name, ps, step.want...)
		if len(step.want) > 0 {
			if v := ps[0].(testclient.ValidateLocation); v.ObjectID != pa.ObjectID() || v.Position() != walkTC(step.corrected) {
				t.Errorf("%s: ValidateLocation %+v, want %v", step.name, v, step.corrected)
			}
		}
		if got := e.position(pa); got.X != step.server.X || got.Y != step.server.Y || got.Z != step.server.Z {
			t.Errorf("%s: the server has Alice at %v, want %v", step.name, got, step.server)
		}
		if ps := sb.take(t); len(ps) != 0 {
			t.Errorf("%s: Bob got %q", step.name, testclient.Names(ps))
		}
	}

	// An impossible height teleports Alice back to the last position the
	// server checked: where the "150 lower" report found her.
	walkReport(e, sa, at(1450, 30000-home.Z))
	ps := sa.take(t)
	checkNames(t, "a height of 30,000", ps, "StopMove", "DeleteObject", "TeleportToLocation")
	if tp := ps[2].(testclient.TeleportToLocation); tp.Position() != walkTC(at(1450, 1900+teleportZOffset)) {
		t.Errorf("teleported to %v, want %v", tp.Position(), at(1450, 1900+teleportZOffset))
	}
	checkNames(t, "Bob", sb.take(t), "StopMove", "DeleteObject", "TeleportToLocation")

	// While teleporting, reports are ignored.
	walkReport(e, sa, at(100, 0))
	checkNames(t, "a report while teleporting", sa.take(t))
	if got := e.position(pa); got.X != home.X+1450 {
		t.Errorf("a report while teleporting moved Alice to %v", got)
	}
}

// walkGeoTile returns L2J geodata of the one map tile that holds blocked:
// flat ground at blocked's height (cut to a multiple of 8, as geodata
// heights are), except blocked's own cell, which is closed on every side.
func walkGeoTile(t *testing.T, blocked world.Position) *geo.Geo {
	t.Helper()
	ground := int(blocked.Z) &^ 7
	return geotest.New("x").Ground(ground).Place(blocked.X, blocked.Y).Load(t)
}

// With geodata, a click on a cell closed on every side gets ActionFailed
// and starts no walk (Mobius isCompletelyBlocked); open ground is walkable.
func TestWalkRefusedOnABlockedCell(t *testing.T) {
	start := npcTestStart(t)
	blocked := world.Position{X: int32(start.X) + 300, Y: int32(start.Y), Z: int32(start.Z)}
	g := walkGeoTile(t, blocked)
	e := newEnv(t, nil, func(s *Server) { s.geo.setTerrain(g) })
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)

	walkClick(e, sa, home, blocked)
	checkNames(t, "a click on a blocked cell", sa.take(t), "ActionFailed")
	checkNames(t, "Bob", sb.take(t))
	if walking, _ := walkState(e, pa); walking {
		t.Error("walking towards a blocked cell")
	}

	walkClick(e, sa, home, walkOffset(home, 0, 300, 0))
	checkNames(t, "a click on open ground", sa.take(t), "MoveToLocation")
	checkNames(t, "Bob sees the walk", sb.take(t), "MoveToLocation")
}

// With the arrow keys the client is in charge: the walk starts at the
// client's origin, ValidatePosition puts the server where the client is
// and everyone gets ValidateLocation. A click ends that.
func TestKeyboardWalkFollowsTheClient(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	origin := walkOffset(home, 300, 0, 0)
	target := walkOffset(home, 500, 0, 0)

	e.srv.Handle(sa, clientpackets.OpMoveToLocation, walkRequest(target, origin, true))
	e.sync()
	ps := sa.take(t)
	checkNames(t, "Alice after an arrow key", ps, "MoveToLocation")
	if m := ps[0].(testclient.MoveToLocation); m.Origin() != walkTC(origin) || m.Destination() != walkTC(target) {
		t.Errorf("MoveToLocation %+v, want from the client's origin %v to %v", m, origin, target)
	}
	checkNames(t, "Bob", sb.take(t), "MoveToLocation")

	client := walkOffset(home, 320, 10, 0)
	walkReport(e, sa, client)
	ps = sa.take(t)
	checkNames(t, "Alice's report", ps, "ValidateLocation")
	if v := ps[0].(testclient.ValidateLocation); v.Position() != walkTC(client) {
		t.Errorf("ValidateLocation at %v, want the client's %v", v.Position(), client)
	}
	checkNames(t, "Bob gets Alice's report", sb.take(t), "ValidateLocation")
	if got := e.position(pa); got.X != client.X || got.Y != client.Y {
		t.Errorf("the server has Alice at %v, want %v", got, client)
	}

	// The walk goes on from the client's position.
	e.advance(world.GameTick)
	walking, here := walkState(e, pa)
	if !walking || here.X <= client.X {
		t.Errorf("after a tick walking %v at %v, want on from %v", walking, here, client)
	}

	walkClick(e, sa, here, walkOffset(home, 900, 0, 0))
	sa.take(t)
	sb.take(t)
	walkReport(e, sa, walkOffset(here, 5, 0, 0))
	checkNames(t, "a report after a click", sa.take(t))
	checkNames(t, "Bob after a click", sb.take(t))
}

// CannotMoveAnymore stops a walk where the client says; for a player that
// stands it does nothing.
func TestCannotMoveAnymore(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	walkClick(e, sa, home, walkOffset(home, 300, 0, 0))
	sa.take(t)
	sb.take(t)

	blocked := world.Position{X: home.X + 50, Y: home.Y, Z: home.Z, Heading: 1234}
	e.srv.Handle(sa, clientpackets.OpCannotMoveAnymore, walkInts(blocked.X, blocked.Y, blocked.Z, blocked.Heading))
	e.sync()
	ps := sa.take(t)
	checkNames(t, "Alice blocked", ps, "StopMove")
	if s := ps[0].(testclient.StopMove); s.Position() != walkTC(blocked) || s.Heading != 1234 {
		t.Errorf("StopMove %+v, want at %v", s, blocked)
	}
	checkNames(t, "Bob", sb.take(t), "StopMove")
	if walking, pos := walkState(e, pa); walking || pos != blocked {
		t.Errorf("walking %v at %v, want standing at %v", walking, pos, blocked)
	}

	e.srv.Handle(sa, clientpackets.OpCannotMoveAnymore, walkInts(home.X, home.Y, home.Z, 0))
	e.sync()
	checkNames(t, "CannotMoveAnymore while standing", sa.take(t))
	if got := e.position(pa); got != blocked {
		t.Errorf("a standing player moved to %v", got)
	}
}

// Turning with the keyboard is shown to everyone; the final heading sticks.
func TestRotation(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)

	e.srv.Handle(sa, clientpackets.OpStartRotating, walkInts(1000, 1))
	e.sync()
	want := testclient.StartRotation{ObjectID: pa.ObjectID(), Degree: 1000, Side: 1}
	for name, s := range map[string]*fakeSession{"Alice": sa, "Bob": sb} {
		ps := s.take(t)
		checkNames(t, name+" after StartRotating", ps, "StartRotation")
		if ps[0].(testclient.StartRotation) != want {
			t.Errorf("%s got %+v, want %+v", name, ps[0], want)
		}
	}

	e.srv.Handle(sa, clientpackets.OpFinishRotating, walkInts(2000, 0))
	e.sync()
	stop := testclient.StopRotation{ObjectID: pa.ObjectID(), Degree: 2000}
	for name, s := range map[string]*fakeSession{"Alice": sa, "Bob": sb} {
		ps := s.take(t)
		checkNames(t, name+" after FinishRotating", ps, "StopRotation")
		if ps[0].(testclient.StopRotation) != stop {
			t.Errorf("%s got %+v, want %+v", name, ps[0], stop)
		}
	}
	if got := e.position(pa); got.Heading != 2000 {
		t.Errorf("heading %d, want 2000", got.Heading)
	}
}

// MovePlayerTo walks up to a point, stopping offset − 5 short of it, and
// sends nothing itself; the arrival hooks run when the walk arrives, which
// ends with StopMove (a walk with an offset), and when the player is already
// there.
func TestMovePlayerTo(t *testing.T) {
	var arrivals []string
	e := newEnv(t, nil, func(s *Server) {
		s.OnPlayerArrived(func(p *Player) { arrivals = append(arrivals, p.Name()) })
	})
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	spot := walkOffset(home, 500, 0, 0)

	var started bool
	var dest world.Position
	e.onLoop(func() {
		started = e.srv.MovePlayerTo(pa, spot, 100)
		if pa.Move != nil {
			dest = pa.Move.Destination
		}
	})
	if !started || dest.X != home.X+405 || dest.Y != home.Y || dest.Z != home.Z {
		t.Fatalf("MovePlayerTo = %v towards %v, want a walk to x %d", started, dest, home.X+405)
	}
	checkNames(t, "Alice after MovePlayerTo", sa.take(t))
	e.onLoop(func() { e.srv.BroadcastMove(pa) })
	checkNames(t, "Alice after BroadcastMove", sa.take(t), "MoveToLocation")
	checkNames(t, "Bob after BroadcastMove", sb.take(t), "MoveToLocation")

	e.advance(5 * time.Second)
	checkNames(t, "Alice on arrival", sa.take(t), "MoveToLocation", "StopMove")
	checkNames(t, "Bob on Alice's arrival", sb.take(t), "MoveToLocation", "StopMove")
	var got []string
	e.onLoop(func() { got = append(got, arrivals...) })
	if len(got) != 1 || got[0] != "Alice" {
		t.Errorf("arrivals %q, want Alice once", got)
	}

	e.onLoop(func() { started = e.srv.MovePlayerTo(pa, walkOffset(dest, 50, 0, 0), 100) })
	if started {
		t.Error("MovePlayerTo started a walk to a point within the offset")
	}
	checkNames(t, "Alice already there", sa.take(t), "ActionFailed")
	got = nil
	e.onLoop(func() { got = append(got, arrivals...) })
	if len(got) != 2 {
		t.Errorf("arrivals %q, want a second one", got)
	}
}

// StopPlayer ends a walk where the player stands, with StopMove for
// everyone (Mobius stopMove(null)); a standing player gets StopMove too.
func TestStopPlayer(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	walkClick(e, sa, home, walkOffset(home, 300, 0, 0))
	e.advance(3 * world.GameTick)
	sa.take(t)
	sb.take(t)
	_, here := walkState(e, pa)

	e.onLoop(func() { e.srv.StopPlayer(pa) })
	ps := sa.take(t)
	checkNames(t, "Alice stopped", ps, "StopMove")
	if s := ps[0].(testclient.StopMove); s.ObjectID != pa.ObjectID() || s.Position() != walkTC(here) {
		t.Errorf("StopMove %+v, want Alice at %v", s, here)
	}
	checkNames(t, "Bob", sb.take(t), "StopMove")
	e.advance(time.Second)
	if walking, pos := walkState(e, pa); walking || pos != here {
		t.Errorf("walking %v at %v, want standing at %v", walking, pos, here)
	}
	checkNames(t, "Alice after the stop", sa.take(t))

	e.onLoop(func() { e.srv.StopPlayer(pa) })
	checkNames(t, "a standing Alice stopped", sa.take(t), "StopMove")
}

// A walk out of the world stops at the last server position (Mobius
// disposeOutOfBoundsObject): Talking Island is 3,873 units from the edge.
func TestWalkOutOfTheWorld(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	beyond := world.Position{X: home.X, Y: world.WorldYMax + 30, Z: home.Z}

	walkClick(e, sa, home, beyond)
	sa.take(t)
	sb.take(t)
	e.advance(time.Minute)
	ps := sa.take(t)
	checkNames(t, "Alice at the edge of the world", ps, "StopMove")
	if s := ps[0].(testclient.StopMove); s.Position() != walkTC(home) || s.Heading != 0 {
		t.Errorf("StopMove %+v, want at %v", s, home)
	}
	checkNames(t, "Bob", sb.take(t), "StopMove")
	if walking, pos := walkState(e, pa); walking || pos.X != home.X || pos.Y != home.Y {
		t.Errorf("walking %v at %v, want standing at %v", walking, pos, home)
	}
}

// A teleport ends the walk where the player stands, with StopMove for
// everyone before TeleportToLocation (Mobius teleToLocation's stopMove).
func TestTeleportEndsTheWalk(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	walkClick(e, sa, home, walkOffset(home, 300, 0, 0))
	e.advance(3 * world.GameTick)
	sa.take(t)
	sb.take(t)
	_, here := walkState(e, pa)

	e.onLoop(func() {
		if err := e.srv.Teleport(pa, gludio); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	ps := sa.take(t)
	checkNames(t, "Alice teleporting while walking", ps, "StopMove", "DeleteObject", "TeleportToLocation")
	if s := ps[0].(testclient.StopMove); s.ObjectID != pa.ObjectID() || s.Position() != walkTC(here) {
		t.Errorf("StopMove %+v, want Alice where the walk got to, %v", s, here)
	}
	checkNames(t, "Bob", sb.take(t), "StopMove", "DeleteObject", "TeleportToLocation")
	e.srv.Handle(sa, clientpackets.OpAppearing, nil)
	e.sync()
	e.advance(time.Second)
	checkNames(t, "Alice after Appearing", sa.take(t), "UserInfo")
	walking, pos := walkState(e, pa)
	if walking || pos.X != gludio.X || pos.Y != gludio.Y {
		t.Errorf("walking %v at %v, want standing in Gludio", walking, pos)
	}
}

// The pace of a walk follows the speed of the moment, as Mobius reads it on
// every update: walking instead of running, water (the zone feature sets
// Motion.InWater). MoveState.Speed shows the speed of the last update.
func TestWalkSpeedFollowsChanges(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, _ := twoPlayers(t, e)
	walkSettle(t, e, sa, sb)
	home := e.position(pa)
	walkClick(e, sa, home, walkOffset(home, 300, 0, 0))
	e.advance(5 * world.GameTick)

	speedNow := func() float64 {
		var v float64
		e.onLoop(func() {
			if pa.Move != nil {
				v = pa.Move.Speed
			}
		})
		return v
	}
	actionUse(e, sa, clientpackets.ActionUseWalkRun)
	e.advance(world.GameTick)
	// Walking: 80 × 1.1.
	if v := speedNow(); math.Abs(v-88) > 1e-9 {
		t.Errorf("speed after the switch to walking %v, want 88", v)
	}
	// Running, the walk would have arrived after 22 ticks.
	e.advance(16 * world.GameTick)
	if walking, _ := walkState(e, pa); !walking {
		t.Fatal("arrived at the running pace after the switch to walking")
	}

	e.onLoop(func() {
		m := pa.Stats().Motion()
		m.InWater = true
		pa.Stats().SetMotion(m)
	})
	e.advance(world.GameTick)
	// Swimming: 50 × 1.1.
	if v := speedNow(); math.Abs(v-55) > 1e-9 {
		t.Errorf("speed in water %v, want 55", v)
	}
	e.onLoop(func() { pa.Stats().SetMotion(stats.Motion{}) })
	e.advance(time.Minute)
	if walking, pos := walkState(e, pa); walking || pos.X != home.X+300 {
		t.Errorf("walking %v at %v, want arrived", walking, pos)
	}
}
