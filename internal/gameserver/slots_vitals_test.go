package gameserver

import (
	"errors"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/game"
	"github.com/IPeter/l2go/internal/storage"
)

// The session hands the game the slots of the list the client saw
// (game.CharacterSlots), so a selection picks the character shown there.
func TestSessionKeepsTheCharacterSlots(t *testing.T) {
	e := newEnv(t, envOptions{})
	c, list := withCharacters(t, e, "player1", "First", "Second")
	if _, err := c.SelectCharacter(testCtx(t), 1); err != nil {
		t.Fatalf("SelectCharacter: %v", err)
	}
	calls := e.game.callsTo("SelectCharacter")
	if len(calls) != 1 {
		t.Fatalf("SelectCharacter calls %+v, want one", calls)
	}
	slots, ok := calls[0].session.(game.CharacterSlots)
	if !ok {
		t.Fatal("the session does not keep the character slots")
	}
	for i, ch := range list.Characters {
		if id, ok := slots.SlotObjectID(int32(i)); !ok || id != ch.ObjectID {
			t.Errorf("slot %d: %#x %v, want %s (%#x)", i, id, ok, ch.Name, ch.ObjectID)
		}
	}
	for _, slot := range []int32{-1, int32(len(list.Characters))} {
		if id, ok := slots.SlotObjectID(slot); ok {
			t.Errorf("slot %d holds %#x, want none", slot, id)
		}
	}
}

// vitalsCall is one call of withVitals.NewCharacterVitals.
type vitalsCall struct {
	classID, level int
	items          []storage.Item
}

// withVitals is the fake game with the starting vitals of game.Server.
type withVitals struct {
	*fakeGame
	vitals game.CharacterVitals
	err    error

	mu    sync.Mutex
	calls []vitalsCall
}

func (g *withVitals) NewCharacterVitals(classID, level int, items []storage.Item) (game.CharacterVitals, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls = append(g.calls, vitalsCall{classID, level, append([]storage.Item(nil), items...)})
	return g.vitals, g.err
}

func (g *withVitals) callsSoFar() []vitalsCall {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]vitalsCall(nil), g.calls...)
}

// vitalsEnv is an environment whose game knows the starting vitals.
func vitalsEnv(t *testing.T, g *withVitals) *env {
	t.Helper()
	return newEnv(t, envOptions{server: func(s *Server) {
		g.fakeGame = s.game.(*fakeGame)
		s.game = g
	}})
}

// A new character starts with the HP, MP and CP the game computes (Mobius:
// full HP and MP with the stat bonuses, CP 0), from its class, level 1 and
// its starting items in their slots; the list shows them.
func TestCharacterCreateUsesTheGamesVitals(t *testing.T) {
	g := &withVitals{vitals: game.CharacterVitals{CurHP: 126, MaxHP: 126, CurMP: 38, MaxMP: 38, MaxCP: 50}}
	e := vitalsEnv(t, g)
	c, _ := e.login(t, "player1")
	res := wantCreated(t, c, humanFighter("Fresh"))

	ch, ok := stored(t, e, "player1", "Fresh")
	if !ok {
		t.Fatal("the character was not stored")
	}
	if ch.CurHP != 126 || ch.MaxHP != 126 || ch.CurMP != 38 || ch.MaxMP != 38 || ch.CurCP != 0 || ch.MaxCP != 50 {
		t.Errorf("stored HP %v/%v MP %v/%v CP %v/%v, want 126/126 38/38 0/50",
			ch.CurHP, ch.MaxHP, ch.CurMP, ch.MaxMP, ch.CurCP, ch.MaxCP)
	}
	info, _ := res.Characters.Find("Fresh")
	if info.CurrentHP != 126 || info.MaxHP != 126 || info.CurrentMP != 38 || info.MaxMP != 38 {
		t.Errorf("listed HP %v/%v MP %v/%v, want 126/126 38/38", info.CurrentHP, info.MaxHP, info.CurrentMP, info.MaxMP)
	}

	calls := g.callsSoFar()
	if len(calls) != 1 || calls[0].classID != 0 || calls[0].level != 1 {
		t.Fatalf("NewCharacterVitals calls %+v, want one for class 0 at level 1", calls)
	}
	worn := 0
	for _, it := range calls[0].items {
		if it.Slot.Equipped() {
			worn++
		}
	}
	// Squire's Sword, Shirt and Pants.
	if len(calls[0].items) != 5 || worn != 3 {
		t.Errorf("the game got %d items, %d worn; want 5 and 3", len(calls[0].items), worn)
	}
}

// When the game cannot compute them, the template's level 1 values stay.
func TestCharacterCreateWithoutTheGamesVitals(t *testing.T) {
	e := vitalsEnv(t, &withVitals{err: errors.New("no stats")})
	c, _ := e.login(t, "player1")
	wantCreated(t, c, humanFighter("Plain"))
	ch, _ := stored(t, e, "player1", "Plain")
	if ch.CurHP != 80 || ch.MaxHP != 80 || ch.CurMP != 30 || ch.MaxMP != 30 || ch.CurCP != 0 || ch.MaxCP != 32 {
		t.Errorf("stored HP %v/%v MP %v/%v CP %v/%v, want the template's 80, 30 and 32",
			ch.CurHP, ch.MaxHP, ch.CurMP, ch.MaxMP, ch.CurCP, ch.MaxCP)
	}
}
