package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// An NPC walk: MoveToLocation to the players around at the start, a tenth
// of the walk speed per game tick, and on arrival the finished walk's
// MoveToLocation from the destination to itself.
func TestNpcMoveTo(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	sess, _ := statusPlayer(t, e, "alice", "Alice")
	n := aiSpawn(e, aiWolf, start, 100, 0)
	e.advance(time.Second)
	sess.take(t)

	from := npcPosition(e, n)
	dest := world.Position{X: from.X + 200, Y: from.Y, Z: from.Z}
	var started bool
	e.onLoop(func() { started = e.srv.NpcMoveTo(n, dest) })
	if !started {
		t.Fatal("NpcMoveTo did not start a walk")
	}
	moves := only[testclient.MoveToLocation](sess.take(t))
	if len(moves) != 1 || moves[0].ObjectID != int32(n.ID()) || moves[0].Origin() != walkTC(from) || moves[0].Destination() != walkTC(dest) {
		t.Fatalf("at the start: %+v; want one MoveToLocation from %v to %v", moves, from, dest)
	}

	var speed float64
	e.onLoop(func() { speed = n.Stats.MoveSpeed() })
	e.advance(time.Second)
	// Mobius's formula takes the collision radius off the distance left, so
	// a walk goes a bit faster than its speed (200 / (200 − 8) here).
	if got, lo, hi := npcPosition(e, n), from.X+int32(speed), from.X+int32(speed*1.1); got.X < lo || got.X > hi || got.Y != from.Y {
		t.Errorf("after 1 s at walk speed %v: %v, want x in [%d, %d]", speed, got, lo, hi)
	}
	e.advance(4 * time.Second)
	got := npcPosition(e, n)
	if got.X != dest.X || got.Y != dest.Y {
		t.Fatalf("after 5 s: %v, want the destination %v", got, dest)
	}
	moves = only[testclient.MoveToLocation](sess.take(t))
	if len(moves) != 1 || moves[0].Origin() != walkTC(dest) || moves[0].Destination() != walkTC(dest) {
		t.Errorf("on arrival: %+v; want the finished walk at %v", moves, dest)
	}
	var moving bool
	e.onLoop(func() { moving = n.Move != nil })
	if moving {
		t.Error("the walk did not end")
	}
}

// A player that starts to see a walking NPC gets its NpcInfo and then its
// MoveToLocation (Mobius describeStateToPlayer).
func TestNpcMoveNewViewerSeesTheWalk(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	statusPlayer(t, e, "alice", "Alice")
	n := aiSpawn(e, aiWolf, start, 100, 0)
	from := npcPosition(e, n)
	dest := world.Position{X: from.X + 250, Y: from.Y, Z: from.Z}
	e.onLoop(func() { e.srv.NpcMoveTo(n, dest) })
	e.advance(time.Second)

	e.createCharacter("bob", "Bob", classHumanFighter, 0)
	bob := newSession("bob")
	e.enter(bob, 0)
	ps := bob.take(t)
	names := aiPacketsAbout(ps, int32(n.ID()))
	if len(names) < 2 || names[0] != "NpcInfo" || names[1] != "MoveToLocation" {
		t.Fatalf("Bob got %q about the walking wolf; want NpcInfo, MoveToLocation", names)
	}
	for _, m := range only[testclient.MoveToLocation](ps) {
		if m.ObjectID == int32(n.ID()) && m.Destination() != walkTC(dest) {
			t.Errorf("Bob's MoveToLocation goes to %v, want %v", m.Destination(), dest)
		}
	}
}

// MoveToPawn goes out when an NPC walks up to a creature; the same walk up
// within a second is not started again.
func TestNpcMoveToPawnRepath(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	n := aiSpawn(e, aiWolf, start, 280, 0)
	e.advance(time.Second)
	sess.take(t)

	e.onLoop(func() { e.srv.NpcMoveToPawn(n, p, 50) })
	pawns := only[testclient.MoveToPawn](sess.take(t))
	if len(pawns) != 1 || pawns[0].TargetID != p.ObjectID() || pawns[0].Distance != 50 {
		t.Fatalf("MoveToPawn: %+v; want one to Alice at 50", pawns)
	}
	e.advance(500 * time.Millisecond)
	e.onLoop(func() { e.srv.NpcMoveToPawn(n, p, 50) })
	if pawns = only[testclient.MoveToPawn](sess.take(t)); len(pawns) != 0 {
		t.Errorf("a second MoveToPawn within 1 s: %+v", pawns)
	}
	e.advance(500 * time.Millisecond)
	e.onLoop(func() { e.srv.NpcMoveToPawn(n, p, 50) })
	if pawns = only[testclient.MoveToPawn](sess.take(t)); len(pawns) != 1 {
		t.Errorf("after 1 s: %d MoveToPawn, want 1", len(pawns))
	}
	e.advance(5 * time.Second)
	if d := npcPosition(e, n).Distance2D(e.position(p)); d > 50 {
		t.Errorf("the wolf stopped %.0f from Alice, want within 50", d)
	}
	if names := aiPacketsAbout(sess.take(t), int32(n.ID())); !aiContains(names, "StopMove") {
		t.Errorf("the walk up ended with %q; want a StopMove", names)
	}
}

// Running and walking change with ChangeMoveType and the NPC's info.
func TestSetNpcRunning(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	sess, _ := statusPlayer(t, e, "alice", "Alice")
	n := aiSpawn(e, aiWolf, start, 100, 0)
	e.advance(time.Second)
	sess.take(t)

	e.onLoop(func() { e.srv.SetNpcRunning(n, true) })
	ps := sess.take(t)
	checkNames(t, "running", ps, "ChangeMoveType", "NpcInfo")
	if c := only[testclient.ChangeMoveType](ps)[0]; !c.Running || c.ObjectID != int32(n.ID()) {
		t.Errorf("ChangeMoveType %+v, want the wolf running", c)
	}
	if info := only[testclient.NpcInfo](ps)[0]; info.RunSpd != 120 {
		t.Errorf("NpcInfo run speed %d, want 120", info.RunSpd)
	}
	e.onLoop(func() { e.srv.SetNpcRunning(n, true) })
	if ps = sess.take(t); len(ps) != 0 {
		t.Errorf("running again sent %q", testclient.Names(ps))
	}
}
