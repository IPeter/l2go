package world_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
	"github.com/IPeter/l2go/internal/world/worldtest"
)

func TestBroadcastOrderPerSink(t *testing.T) {
	w := newWorld(t)
	pos := at(100, 100, 8)
	a := worldtest.NewPlayer(1, pos)
	b := worldtest.NewPlayer(2, offset(pos, 100, 0, 0))
	c := worldtest.NewPlayer(3, at(101, 100, 8))   // the next region
	far := worldtest.NewPlayer(4, at(103, 100, 8)) // out of view
	npc := worldtest.NewObject(5, pos)
	add(t, w, a, b, c, far, npc)

	m1, m2, m3, m4 := []byte("m1"), []byte("m2"), []byte("m3"), []byte("m4")
	w.Broadcast(a, m1, true)
	w.Broadcast(npc, m2, true) // not a viewer: self does nothing
	w.Broadcast(b, m3, false)
	w.Broadcast(a, m4, true)

	tests := []struct {
		p    *worldtest.Player
		want []string
	}{
		{a, []string{"m1", "m2", "m3", "m4"}},
		{b, []string{"m1", "m2", "m4"}},
		{c, []string{"m1", "m2", "m3", "m4"}},
		{far, nil},
	}
	for _, tt := range tests {
		if got := tt.p.Rec.Strings(); !slices.Equal(got, tt.want) {
			t.Errorf("player %d got %v, want %v", tt.p.ID(), got, tt.want)
		}
	}
	// Encoded once: every receiver holds the sender's slice.
	if got := c.Rec.Bodies(); len(got) == 0 || &got[0][0] != &m1[0] {
		t.Error("the body was copied; a broadcast must share it")
	}
}

func TestBroadcastInRange(t *testing.T) {
	w := newWorld(t)
	pos := at(100, 100, 8)
	a := worldtest.NewPlayer(1, pos)
	near := worldtest.NewPlayer(2, offset(pos, 600, 800, 0))    // 1000 away
	beyond := worldtest.NewPlayer(3, offset(pos, 600, 800, 10)) // just over 1000
	other := worldtest.NewPlayer(4, offset(pos, 10, 0, 0))      // another instance
	tall := worldtest.NewPlayer(5, offset(pos, 0, 0, 999))      // 999 up
	if err := other.SetInstanceID(3); err != nil {
		t.Fatal(err)
	}
	add(t, w, a, near, beyond, other, tall)

	w.BroadcastInRange(a, []byte("hi"), 1000, false)
	for _, tt := range []struct {
		p    *worldtest.Player
		want int
	}{{a, 0}, {near, 1}, {beyond, 0}, {other, 0}, {tall, 1}} {
		if got := tt.p.Rec.Len(); got != tt.want {
			t.Errorf("player %d got %d packets, want %d", tt.p.ID(), got, tt.want)
		}
	}

	w.BroadcastInRange(a, []byte("self only"), -1, true)
	if got := a.Rec.Strings(); !slices.Equal(got, []string{"self only"}) {
		t.Errorf("negative range with self: sender got %v", got)
	}
	if near.Rec.Len() != 1 {
		t.Error("a negative range reached another player")
	}
}

func TestBroadcastFollowsVisibility(t *testing.T) {
	w := newWorld(t)
	pos := at(100, 100, 8)
	ghost := worldtest.NewPlayer(1, pos)
	ghost.SetInvisible(true)
	plain := worldtest.NewPlayer(2, pos)
	gm := worldtest.NewPlayer(3, pos)
	gm.GM = true
	add(t, w, ghost, plain, gm)

	w.Broadcast(ghost, []byte("move"), true)
	if ghost.Rec.Len() != 1 || plain.Rec.Len() != 0 || gm.Rec.Len() != 1 {
		t.Errorf("invisible sender reached self %d, player %d, GM %d; want 1, 0, 1",
			ghost.Rec.Len(), plain.Rec.Len(), gm.Rec.Len())
	}
}

func TestPlayersAndBroadcastAll(t *testing.T) {
	w := newWorld(t)
	a := worldtest.NewPlayer(1, at(100, 100, 8))
	b := worldtest.NewPlayer(2, at(10, 10, 8))
	teleporting := worldtest.NewPlayer(3, at(50, 50, 8)) // online, not in the world
	add(t, w, a, b)
	for _, p := range []*worldtest.Player{a, b, teleporting} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.AddPlayer(worldtest.NewPlayer(2, at(1, 1, 1))); !errors.Is(err, world.ErrDuplicateID) {
		t.Errorf("AddPlayer with a used ID: %v, want ErrDuplicateID", err)
	}
	if got := seqIDs(w.Players()); !slices.Equal(got, []world.ObjectID{1, 2, 3}) {
		t.Errorf("Players = %v", got)
	}

	w.BroadcastAll([]byte("announce"))
	for _, p := range []*worldtest.Player{a, b, teleporting} {
		if got := p.Rec.Strings(); !slices.Equal(got, []string{"announce"}) {
			t.Errorf("player %d got %v", p.ID(), got)
		}
	}

	// A stale session can't unregister the new player with its ID.
	w.RemovePlayer(worldtest.NewPlayer(2, at(1, 1, 1)))
	if v, ok := w.Player(2); !ok || v != world.Viewer(b) {
		t.Errorf("Player(2) = %v, %v; want b", v, ok)
	}
	w.RemovePlayer(b)
	if _, ok := w.Player(2); ok || w.PlayerCount() != 2 {
		t.Errorf("after RemovePlayer: Player(2) found %v, count %d", ok, w.PlayerCount())
	}
	if v, ok := w.Player(3); !ok || v != world.Viewer(teleporting) {
		t.Errorf("Player(3) after removing another = %v, %v", v, ok)
	}
	w.BroadcastAll([]byte("again"))
	if b.Rec.Len() != 1 || a.Rec.Len() != 2 {
		t.Errorf("after RemovePlayer: removed got %d, kept got %d; want 1, 2", b.Rec.Len(), a.Rec.Len())
	}
}

func TestSendWithoutSink(t *testing.T) {
	var got [][]byte
	world.SinkFunc(func(b []byte) { got = append(got, b) }).Send([]byte("x"))
	if len(got) != 1 {
		t.Fatal("SinkFunc did not forward")
	}
	world.Send(noSink{worldtest.NewPlayer(1, at(1, 1, 1))}, []byte("dropped")) // must not panic
}

// noSink is a viewer without a client.
type noSink struct{ *worldtest.Player }

func (noSink) Sink() world.Sink { return nil }

type flushed struct {
	id world.ObjectID
	u  world.Update
}

func TestUpdatesCoalescePerTick(t *testing.T) {
	w, l, clk := newFakeWorld(t)
	a := worldtest.NewPlayer(1, at(100, 100, 8))
	b := worldtest.NewObject(2, at(100, 100, 8))
	var got []flushed
	w.SetUpdateFlush(func(o world.Object, u world.Update) {
		got = append(got, flushed{o.ID(), u})
		if o.ID() == 1 && u&world.UpdateUserInfo != 0 {
			w.MarkUpdate(o, world.UpdateCharInfo) // marked while flushing: next tick
		}
	})

	w.MarkUpdate(a, world.UpdateUserInfo)
	w.MarkUpdate(b, world.UpdateCharInfo)
	w.MarkUpdate(a, world.UpdateCharInfo)
	w.MarkUpdate(b, 0)
	step(t, l, clk, 50*time.Millisecond)
	if len(got) != 0 {
		t.Fatalf("flushed before the tick: %v", got)
	}
	step(t, l, clk, 50*time.Millisecond)
	want := []flushed{{1, world.UpdateUserInfo | world.UpdateCharInfo}, {2, world.UpdateCharInfo}}
	if !slices.Equal(got, want) {
		t.Errorf("first tick flushed %v, want %v", got, want)
	}

	got = nil
	step(t, l, clk, 100*time.Millisecond)
	if want := []flushed{{1, world.UpdateCharInfo}}; !slices.Equal(got, want) {
		t.Errorf("second tick flushed %v, want %v", got, want)
	}
	got = nil
	step(t, l, clk, 100*time.Millisecond)
	if len(got) != 0 {
		t.Errorf("third tick flushed %v, want nothing", got)
	}
}

func TestUpdatesFlushAfterTickHooks(t *testing.T) {
	w, l, clk := newFakeWorld(t)
	o := worldtest.NewObject(1, at(100, 100, 8))
	var got []flushed
	w.SetUpdateFlush(func(o world.Object, u world.Update) { got = append(got, flushed{o.ID(), u}) })
	// A hook added after New still runs before the flush of the same tick.
	l.OnTick(func(time.Time) { w.MarkUpdate(o, world.UpdateCharInfo) })
	step(t, l, clk, 100*time.Millisecond)
	if want := []flushed{{1, world.UpdateCharInfo}}; !slices.Equal(got, want) {
		t.Errorf("flushed %v, want %v", got, want)
	}

	got = nil
	w.MarkUpdate(o, world.UpdateUserInfo)
	w.FlushUpdates()
	if want := []flushed{{1, world.UpdateUserInfo}}; !slices.Equal(got, want) {
		t.Errorf("FlushUpdates flushed %v, want %v", got, want)
	}
}
