package game

import (
	"errors"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// gludio is a destination far from the Talking Island start points.
var gludio = world.Position{X: -12672, Y: 122776, Z: -3116}

func TestTeleportAndAppearing(t *testing.T) {
	type move struct{ from, to world.Position }
	var moves []move
	var teleports []world.Position
	e := newEnv(t, nil, func(s *Server) {
		s.OnPlayerMoved(func(p *Player, from, to world.Position) { moves = append(moves, move{from, to}) })
		s.OnTeleport(func(p *Player, to world.Position) {
			if !p.InWorld() {
				t.Error("OnTeleport: the player already left the world")
			}
			teleports = append(teleports, to)
		})
	})
	sa, sb, pa, _ := twoPlayers(t, e)
	start := e.position(pa)
	want := gludio
	want.Z += teleportZOffset
	want.Heading = start.Heading

	e.onLoop(func() {
		// Entering ran the hook for both players; count the teleport only.
		moves = nil
		pa.Move = &MoveState{Origin: start, Destination: gludio}
		if err := e.srv.Teleport(pa, gludio); err != nil {
			t.Errorf("Teleport: %v", err)
		}
		if pa.InWorld() || !pa.Teleporting() || pa.Move != nil {
			t.Errorf("teleporting: in the world %v, teleporting %v, moving %v", pa.InWorld(), pa.Teleporting(), pa.Move != nil)
		}
	})
	ps := sa.take(t)
	checkNames(t, "the teleporting player", ps, "DeleteObject", "TeleportToLocation")
	if tp := ps[1].(testclient.TeleportToLocation); tp.ObjectID != pa.ObjectID() || tp.Position() != (testclient.Position{X: want.X, Y: want.Y, Z: want.Z}) || tp.Fade != 0 {
		t.Errorf("TeleportToLocation %+v, want %v (Z + 5)", tp, want)
	}
	// The players around the old place forget the player and see it go.
	checkNames(t, "the player left behind", sb.take(t), "DeleteObject", "TeleportToLocation")

	// Appearing puts the player back at the destination and sends UserInfo.
	e.srv.Handle(sa, clientpackets.OpAppearing, nil)
	e.sync()
	ps = sa.take(t)
	checkNames(t, "after Appearing", ps, "UserInfo")
	if ui := ps[0].(testclient.UserInfo); ui.Position() != (testclient.Position{X: want.X, Y: want.Y, Z: want.Z}) {
		t.Errorf("UserInfo after Appearing at %v, want %v", ui.Position(), want)
	}
	if got := e.position(pa); got != want || !e.inWorld(pa) {
		t.Errorf("after Appearing at %v (in the world %v), want %v", got, e.inWorld(pa), want)
	}
	// Bob is far away now and sees nothing more.
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("Bob got %q after Alice arrived far away", testclient.Names(ps))
	}
	var gotMoves []move
	var gotTeleports []world.Position
	e.onLoop(func() {
		gotMoves = append(gotMoves, moves...)
		gotTeleports = append(gotTeleports, teleports...)
	})
	if len(gotTeleports) != 1 || gotTeleports[0].X != want.X || gotTeleports[0].Z != want.Z {
		t.Errorf("OnTeleport saw %v, want one teleport to %v", gotTeleports, want)
	}
	if len(gotMoves) != 1 || gotMoves[0].from != start || gotMoves[0].to != want {
		t.Errorf("OnPlayerMoved saw %v, want %v to %v", gotMoves, start, want)
	}

	// Appearing without a teleport only sends UserInfo.
	e.srv.Handle(sa, clientpackets.OpAppearing, nil)
	e.sync()
	checkNames(t, "Appearing without a teleport", sa.take(t), "UserInfo")
}

func TestEnterRunsMovedHookAtTheSpawn(t *testing.T) {
	type move struct {
		name     string
		from, to world.Position
	}
	var moves []move
	e := newEnv(t, nil, func(s *Server) {
		s.OnPlayerMoved(func(p *Player, from, to world.Position) {
			if !p.InWorld() {
				t.Error("OnPlayerMoved on enter: not in the world yet")
			}
			moves = append(moves, move{p.Name(), from, to})
		})
	})
	c := e.createCharacter("hero", "Hero", classHumanFighter, 0)
	e.enter(newSession("hero"), 0)
	pos := world.Position{X: c.X, Y: c.Y, Z: c.Z}
	var got []move
	e.onLoop(func() { got = append(got, moves...) })
	if len(got) != 1 || got[0] != (move{"Hero", pos, pos}) {
		t.Errorf("OnPlayerMoved on enter: %+v, want Hero at %v", got, pos)
	}
}

func TestTeleportWatchdog(t *testing.T) {
	e := newEnv(t, nil)
	sa, _, pa, _ := twoPlayers(t, e)
	e.onLoop(func() {
		if err := e.srv.Teleport(pa, gludio); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	sa.take(t)

	e.advance(teleportWatchdog - time.Millisecond)
	if e.inWorld(pa) {
		t.Fatal("back in the world before the watchdog")
	}
	e.advance(time.Millisecond)
	if !e.inWorld(pa) {
		t.Fatal("not back in the world after the watchdog")
	}
	if got := e.position(pa); got.X != gludio.X || got.Y != gludio.Y || got.Z != gludio.Z+teleportZOffset {
		t.Errorf("put back at %v, want %v", got, gludio)
	}
	// A late Appearing still gets its UserInfo and changes nothing else.
	sa.take(t)
	e.srv.Handle(sa, clientpackets.OpAppearing, nil)
	e.sync()
	checkNames(t, "a late Appearing", sa.take(t), "UserInfo")
}

func TestTeleportRefusals(t *testing.T) {
	e := newEnv(t, nil)
	sa, _, pa, _ := twoPlayers(t, e)
	start := e.position(pa)
	e.onLoop(func() {
		outside := world.Position{X: world.WorldXMax + 100, Y: 0, Z: 0}
		if err := e.srv.Teleport(pa, outside); !errors.Is(err, world.ErrOutOfWorld) {
			t.Errorf("Teleport outside the world = %v, want ErrOutOfWorld", err)
		}
		if !pa.InWorld() || pa.Teleporting() || pa.Position() != start {
			t.Error("a refused teleport moved the player")
		}
	})
	// A player that left cannot teleport.
	e.srv.Disconnected(sa)
	e.onLoop(func() {
		if err := e.srv.Teleport(pa, gludio); !errors.Is(err, ErrNotInWorld) {
			t.Errorf("Teleport of a player that left = %v, want ErrNotInWorld", err)
		}
	})
}

func TestLeaveWhileTeleporting(t *testing.T) {
	e := newEnv(t, nil)
	sa, _, pa, _ := twoPlayers(t, e)
	e.onLoop(func() {
		if err := e.srv.Teleport(pa, gludio); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	e.srv.Logout(sa)
	saved := e.store.waitSave(t, pa.ObjectID())
	if saved.X != gludio.X || saved.Y != gludio.Y || saved.Z != gludio.Z+teleportZOffset {
		t.Errorf("saved at (%d, %d, %d), want the destination %v", saved.X, saved.Y, saved.Z, gludio)
	}
	// The watchdog does not bring a player that left back.
	e.advance(2 * teleportWatchdog)
	if e.inWorld(pa) {
		t.Error("the watchdog put a player that left back into the world")
	}
}
