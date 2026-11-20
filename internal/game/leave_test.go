package game

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// checkSaved fails the test unless the stored character stands at pos.
func checkSaved(t *testing.T, e *testEnv, objectID int32, pos world.Position) storage.Character {
	t.Helper()
	c, err := e.store.Get(e.ctx, objectID)
	if err != nil {
		t.Fatalf("Get(%d): %v", objectID, err)
	}
	if got := (world.Position{X: c.X, Y: c.Y, Z: c.Z, Heading: c.Heading}); got != pos {
		t.Errorf("stored at %v, want %v", got, pos)
	}
	return c
}

// checkGone fails the test if the session still has a player, or the
// player is still online or in the world.
func checkGone(t *testing.T, e *testEnv, sess *fakeSession, p *Player) {
	t.Helper()
	e.onLoop(func() {
		if e.srv.playerOf(sess) != nil {
			t.Error("the session still has a player")
		}
		if _, ok := e.srv.Player(p.ID()); ok {
			t.Error("the player is still online")
		}
		if _, ok := e.srv.PlayerByName(p.Name()); ok {
			t.Error("the player can still be found by name")
		}
		if p.InWorld() || p.InGame() {
			t.Errorf("the player is in the world %v, in the game %v", p.InWorld(), p.InGame())
		}
	})
}

func TestRestartSavesBeforeTheCharacterList(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	p := e.enter(sess, 0)
	moved := world.Position{X: c.X + 100, Y: c.Y - 50, Z: c.Z, Heading: 1234}
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, moved); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
		p.CurHP = 55
	})
	sess.take(t)

	release := e.store.holdSaves()
	e.srv.Restart(sess)
	sess.mu.Lock()
	early := sess.returned
	sess.mu.Unlock()
	if early != 0 {
		t.Error("the character list was sent before the save was written")
	}
	release()
	sess.waitFor(t, "ReturnToCharacterSelect", func(s *fakeSession) bool { return s.returned == 1 })

	saved := checkSaved(t, e, c.ObjectID, moved)
	switch {
	case saved.CurHP != 55 || saved.MaxHP != 126:
		t.Errorf("stored HP %v/%v, want 55/126", saved.CurHP, saved.MaxHP)
	case !saved.LastAccess.Equal(testStart):
		t.Errorf("stored last access %v, want %v", saved.LastAccess, testStart)
	case len(saved.Items) != len(c.Items):
		t.Errorf("stored %d items, want %d", len(saved.Items), len(c.Items))
	}
	checkGone(t, e, sess, p)
	if _, _, closed := sess.closedWith(t); closed {
		t.Error("restart closed the connection")
	}

	// Entering again puts the character where it was saved.
	sess.take(t)
	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter after restart: %v", err)
	}
	ps := sess.take(t)
	checkNames(t, "select after restart", ps, "CharSelected")
	if sel := ps[0].(testclient.CharSelected); sel.X != moved.X || sel.Y != moved.Y || sel.Z != moved.Z || sel.CurrentHP != 55 {
		t.Errorf("CharSelected at (%d, %d, %d) with HP %v, want %v and 55", sel.X, sel.Y, sel.Z, sel.CurrentHP, moved)
	}
}

func TestLogoutSavesAndClosesAfterLeaveWorld(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, _, pb := twoPlayers(t, e)
	pos := e.position(pb)

	e.srv.Logout(sb)
	final, delay, closed := sb.closedWith(t)
	if _, ok := final.(testclient.LeaveWorld); !closed || !ok || delay != time.Second {
		t.Errorf("logout closed %v with %v after %v, want LeaveWorld and 1s", closed, final, delay)
	}
	saved := e.store.waitSave(t, pb.ObjectID())
	if got := (world.Position{X: saved.X, Y: saved.Y, Z: saved.Z, Heading: saved.Heading}); got != pos {
		t.Errorf("saved at %v, want %v", got, pos)
	}
	checkGone(t, e, sb, pb)
	// Mobius deleteMe stops Bob (StopMove, also when he stood still) before
	// it takes him out of the world.
	seen := sa.take(t)
	checkNames(t, "Alice after Bob's logout", seen, "StopMove", "DeleteObject")
	want := testclient.StopMove{ObjectID: pb.ObjectID(), X: pos.X, Y: pos.Y, Z: pos.Z, Heading: pos.Heading}
	if stop := seen[0].(testclient.StopMove); stop != want {
		t.Errorf("Alice got %+v, want %+v", stop, want)
	}
	// Bob's client is detached first, as in Mobius: it gets no StopMove or
	// DeleteObject, only the LeaveWorld that closes the connection.
	if ps := sb.take(t); len(ps) != 0 {
		t.Errorf("Bob got %q when leaving, want nothing", testclient.Names(ps))
	}
}

func TestDisconnectedSavesAndRemoves(t *testing.T) {
	var left []string
	e := newEnv(t, nil, func(s *Server) {
		s.OnPlayerLeave(func(p *Player) {
			if !p.InWorld() {
				t.Errorf("OnPlayerLeave: %s is already out of the world", p.Name())
			}
			left = append(left, p.Name())
		})
	})
	sa, sb, _, pb := twoPlayers(t, e)

	e.srv.Disconnected(sb)
	e.store.waitSave(t, pb.ObjectID())
	checkGone(t, e, sb, pb)
	checkNames(t, "Alice after Bob's disconnection", sa.take(t), "StopMove", "DeleteObject")
	if _, _, closed := sb.closedWith(t); closed {
		t.Error("the game closed a connection that was already lost")
	}
	var got []string
	e.onLoop(func() { got = append(got, left...) })
	if len(got) != 1 || got[0] != "Bob" {
		t.Errorf("OnPlayerLeave ran for %q, want [Bob]", got)
	}

	// A second Disconnected, or one for a session without a player, does
	// nothing.
	e.srv.Disconnected(sb)
	e.srv.Disconnected(newSession("nobody"))
	e.sync()
	if n := e.store.saveCount(pb.ObjectID()); n != 1 {
		t.Errorf("saved %d times, want once", n)
	}
}

func TestDisconnectedBeforeEnterWorld(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	if err := e.srv.SelectCharacter(e.ctx, sess, 0); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	e.srv.Disconnected(sess)
	e.onLoop(func() {
		if e.srv.playerOf(sess) != nil || e.srv.byAccount["hero"] != nil {
			t.Error("the selected character is still registered")
		}
	})
	if n := e.store.saveCount(c.ObjectID); n != 0 {
		t.Errorf("a character that never entered was saved %d times", n)
	}
}

func TestDoubleLoginKicksTheFirstSession(t *testing.T) {
	e := newEnv(t, nil)
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	first := newSession("hero")
	p := e.enter(first, 0)
	moved := world.Position{X: c.X + 250, Y: c.Y + 250, Z: c.Z, Heading: 4000}
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, moved); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})

	second := newSession("hero")
	if err := e.srv.SelectCharacter(e.ctx, second, 0); err != nil {
		t.Fatalf("second SelectCharacter: %v", err)
	}
	final, _, closed := first.closedWith(t)
	if msg, ok := final.(testclient.SystemMessage); !closed || !ok || msg.ID != int32(sysMsgAnotherLogin) {
		t.Errorf("first session closed %v with %v, want SystemMessage %d", closed, final, sysMsgAnotherLogin)
	}
	// The character was saved before the second session loaded it.
	ps := second.take(t)
	checkNames(t, "second session", ps, "CharSelected")
	if sel := ps[0].(testclient.CharSelected); sel.X != moved.X || sel.Y != moved.Y || sel.Z != moved.Z {
		t.Errorf("the second session loaded the character at (%d, %d, %d), want %v", sel.X, sel.Y, sel.Z, moved)
	}
	if n := e.store.saveCount(c.ObjectID); n != 1 {
		t.Errorf("saved %d times, want once", n)
	}
	e.onLoop(func() {
		if e.srv.playerOf(first) != nil {
			t.Error("the kicked session still has a player")
		}
		if e.srv.playerOf(second) == nil {
			t.Error("the new session has no player")
		}
	})

	// The old connection's Disconnected does not touch the new session.
	e.srv.Disconnected(first)
	e.srv.EnterWorld(second)
	e.onLoop(func() {
		if q := e.srv.playerOf(second); q == nil || !q.InWorld() {
			t.Error("the new session could not enter the world")
		}
	})
}

func TestAutosave(t *testing.T) {
	e := newEnv(t, func(gs *config.GameServer) { gs.World.AutosaveInterval = config.Duration(time.Minute) })
	c := e.createCharacter("hero", "Hero", classHumanFighter, storage.AccessLevelPlayer)
	sess := newSession("hero")
	p := e.enter(sess, 0)
	moved := world.Position{X: c.X + 10, Y: c.Y, Z: c.Z, Heading: 77}
	e.onLoop(func() {
		if err := e.srv.MoveObject(p, moved); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})

	e.advance(time.Minute - time.Millisecond)
	if n := e.store.saveCount(c.ObjectID); n != 0 {
		t.Fatalf("saved %d times before the autosave interval", n)
	}
	e.advance(time.Millisecond)
	saved := e.store.waitSave(t, c.ObjectID)
	if got := (world.Position{X: saved.X, Y: saved.Y, Z: saved.Z, Heading: saved.Heading}); got != moved {
		t.Errorf("autosaved at %v, want %v", got, moved)
	}
	if want := testStart.Add(time.Minute); !saved.LastAccess.Equal(want) {
		t.Errorf("autosave last access %v, want %v", saved.LastAccess, want)
	}
	e.advance(time.Minute)
	e.store.waitSave(t, c.ObjectID)

	// After leaving, the autosave stops.
	e.srv.Logout(sess)
	e.store.waitSave(t, c.ObjectID)
	e.advance(3 * time.Minute)
	if n := e.store.saveCount(c.ObjectID); n != 3 {
		t.Errorf("saved %d times, want 3 (two autosaves and the logout)", n)
	}
}

func TestShutdownSavesEveryone(t *testing.T) {
	e := newEnv(t, nil)
	sa, sb, pa, pb := twoPlayers(t, e)
	e.stop()
	for _, s := range []*fakeSession{sa, sb} {
		final, _, closed := s.closedWith(t)
		if _, ok := final.(testclient.ServerClose); !closed || !ok {
			t.Errorf("%s: closed %v with %v at shutdown, want ServerClose", s.account, closed, final)
		}
	}
	for _, p := range []*Player{pa, pb} {
		if n := e.store.saveCount(p.ObjectID()); n != 1 {
			t.Errorf("%s saved %d times at shutdown, want once", p.Name(), n)
		}
	}
}

func TestSaverWritesInOrderAndKeepsTheNewest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store := newRecordingStore()
	c := storage.Character{ObjectID: storage.FirstObjectID, Account: "a", Name: "A", Level: 1}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	s := newSaver(store, slog.New(slog.DiscardHandler))
	release := store.holdSaves()
	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		s.run(ctx)
	}()

	var dones [3]chan error
	snap := func(x int32, i int) {
		dones[i] = make(chan error, 1)
		v := c
		v.X = x
		s.enqueue(v, func(err error) { dones[i] <- err })
	}
	snap(1, 0)
	// Wait until the first snapshot is being written (held by the store).
	select {
	case <-store.started:
	case <-ctx.Done():
		t.Fatal("the first snapshot was not taken up")
	}
	snap(2, 1) // replaced by the next one before it is written
	snap(3, 2)
	release()
	for i, ch := range dones {
		select {
		case err := <-ch:
			if err != nil {
				t.Errorf("save %d: %v", i, err)
			}
		case <-ctx.Done():
			t.Fatalf("save %d did not finish", i)
		}
	}
	if err := s.wait(ctx, c.ObjectID); err != nil {
		t.Fatalf("wait: %v", err)
	}
	store.mu.Lock()
	var xs []int32
	for _, v := range store.saves {
		xs = append(xs, v.X)
	}
	store.mu.Unlock()
	if len(xs) != 2 || xs[0] != 1 || xs[1] != 3 {
		t.Errorf("wrote X %v, want [1 3]: in order, the replaced snapshot skipped", xs)
	}
	got, err := store.Get(ctx, c.ObjectID)
	if err != nil || got.X != 3 {
		t.Errorf("stored X %d (%v), want 3", got.X, err)
	}

	s.close()
	select {
	case <-runDone:
	case <-ctx.Done():
		t.Fatal("run did not return after close")
	}
	late := make(chan error, 1)
	s.enqueue(c, func(err error) { late <- err })
	if err := <-late; err == nil {
		t.Error("a save after close reported success")
	}
}
