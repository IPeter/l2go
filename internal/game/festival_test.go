package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/sevensigns"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// ssqParty enters n characters of the level at pos with the items of the
// leader and makes them a party, the first the leader.
func (e *testEnv) ssqParty(n int, level int32, pos world.Position, leaderItems, memberItems []storage.Item) []*Player {
	e.t.Helper()
	names := []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo", "Foxtrot"}
	var ps []*Player
	for i := range n {
		items := memberItems
		if i == 0 {
			items = leaderItems
		}
		var copied []storage.Item
		for _, it := range items {
			it.Slot, it.Mana = storage.PaperdollNone, storage.NoMana
			copied = append(copied, it)
		}
		_, p := e.enterChar(testChar{account: "acc" + names[i], name: names[i], pos: pos, level: level, items: copied})
		ps = append(ps, p)
	}
	e.onLoop(func() {
		pt := newParty(ps[0], 0)
		for _, p := range ps[1:] {
			e.srv.addPartyMember(pt, p)
		}
	})
	return ps
}

func (e *testEnv) ssqItemCount(p *Player, id int32) int64 {
	var n int64
	e.onLoop(func() {
		if p.inv.Inventory != nil {
			n = p.inv.Count(id, -1, true)
		}
	})
	return n
}

// TestFestivalCycle runs a festival: a Dawn party of five signs up with its
// seal stones, goes into the arena, gathers Blood Offerings, comes back at
// the end and its leader registers the score, which gives Dawn the
// festival's points.
func TestFestivalCycle(t *testing.T) {
	start := ssqMonday.Add(16 * time.Minute) // the competition
	e, store := newSSQEnv(t, ssqEnvOptions{start: start})
	if st := e.ssqState(); st.Period != sevensigns.PeriodCompetition {
		t.Fatalf("period %v", st.Period)
	}
	ps := e.ssqParty(5, 20, tiStart, []storage.Item{{ItemID: sevensigns.StoneBlueID, Count: 1000}}, nil)
	leader := ps[0]
	guide := e.spawnNpc(ssqTestGuide, eastOf(tiStart, 30))
	e.onLoop(func() { e.srv.bypassFestival(leader, guide, "festival 1") })
	e.onLoop(func() { e.srv.bypassFestival(leader, guide, "festival 2 6360") })
	if got := e.ssqItemCount(leader, sevensigns.StoneBlueID); got != 100 {
		t.Fatalf("blue stones left %d, want 100", got)
	}
	var bonus int
	e.onLoop(func() { bonus = e.srv.sevenSigns.festival.book.Bonus[0] })
	if bonus != 900*sevensigns.StoneBlueValue {
		t.Errorf("bonus %d", bonus)
	}

	// The manager starts 2 minutes after the server, the sign-up ends 19
	// minutes later: the party goes into the Dawn arena of festival 0.
	e.advance(2*time.Minute + 19*time.Minute)
	sd, _ := loadSSQTestData()
	arena := sd.Festival.DawnPlayer[0]
	pos := e.position(leader)
	if d := pos.Distance2D(world.Position{X: int32(arena.X), Y: int32(arena.Y)}); d > 330 {
		t.Fatalf("leader at %v, %v from the arena", pos, d)
	}
	var initialized bool
	e.onLoop(func() { initialized = e.srv.sevenSigns.festival.run.initialized })
	if !initialized {
		t.Fatal("the festival did not start")
	}

	// Offerings: what the monsters' kills give the leader.
	e.onLoop(func() {
		if _, err := e.srv.AddItem(leader, sevensigns.OfferingID, 30, 0, false, false); err != nil {
			t.Fatal(err)
		}
	})

	// 18 minutes later the festival ends and the party is back.
	e.advance(18 * time.Minute)
	if d := e.position(leader).Distance2D(tiStart); d > 1 {
		t.Errorf("leader not back: %v", e.position(leader))
	}
	e.onLoop(func() { e.srv.bypassFestival(leader, guide, "festival 3") })
	if got := e.ssqItemCount(leader, sevensigns.OfferingID); got != 0 {
		t.Errorf("offerings left %d", got)
	}
	st := e.ssqState()
	if st.DawnFestivalScore != sevensigns.FestivalLevelScores[0] {
		t.Errorf("dawn festival score %d", st.DawnFestivalScore)
	}
	var best sevensigns.FestivalScore
	e.onLoop(func() { best = e.srv.sevenSigns.festival.book.HighestScoreData(sevensigns.CabalDawn, 0) })
	if best.Score != 150 || best.Members != "Alpha,Bravo,Charlie,Delta,Echo" {
		t.Errorf("best score %+v", best)
	}
	// Saved: the festival rows and the status.
	e.stop()
	stored, err := store.LoadSevenSigns(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range stored.Festival {
		if r.Cabal == "dawn" && r.Festival == 0 && r.Score == 150 {
			found = true
		}
	}
	if !found || stored.Status.DawnFestivalScore != 60 || stored.Status.AccumulatedBonus[0] != 2700 {
		t.Errorf("stored %+v", stored)
	}
}

// TestRiftRun runs the Dimensional Rift: a party of two in the waiting room
// pays its fragments, enters a recruit room, jumps automatically and comes
// back after the last room.
func TestRiftRun(t *testing.T) {
	e, _ := newSSQEnv(t, ssqEnvOptions{start: ssqMonday.Add(time.Hour)})
	sd, _ := loadSSQTestData()
	wait, ok := sd.RiftRoom(data.RiftTypeWaiting, 0)
	if !ok {
		t.Fatal("no waiting room")
	}
	wpos := world.Position{X: int32(wait.TeleX), Y: int32(wait.TeleY), Z: int32(wait.TeleZ)}
	frag := []storage.Item{{ItemID: ssqTestFragment, Count: 20}}
	ps := e.ssqParty(2, 40, wpos, frag, frag)
	guard := e.spawnNpc(ssqTestRiftGuard, eastOf(wpos, 20))
	e.onLoop(func() { e.srv.bypassRift(ps[1], guard, "enterrift", "enterrift 1") })
	if e.ssqItemCount(ps[0], ssqTestFragment) != 20 {
		t.Fatal("a member started the rift")
	}
	e.onLoop(func() { e.srv.bypassRift(ps[0], guard, "enterrift", "enterrift 1") })
	for _, p := range ps {
		if got := e.ssqItemCount(p, ssqTestFragment); got != 2 {
			t.Errorf("%s has %d fragments, want 2", p.Name(), got)
		}
	}
	room := func() (int, int) {
		var typ, id int
		e.onLoop(func() {
			if r := e.srv.sevenSigns.rift.parties[ps[0].Party()]; r != nil {
				typ, id = r.typ, r.room
			} else {
				typ, id = -1, -1
			}
		})
		return typ, id
	}
	typ, first := room()
	if typ != 1 || first < 1 {
		t.Fatalf("rift room %d/%d", typ, first)
	}
	rd, _ := sd.RiftRoom(1, first)
	if pos := e.position(ps[0]); !rd.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
		t.Errorf("leader at %v, not in room %d", pos, first)
	}
	// The jumps: 480-600 s each (the boss room 1.5 times longer), four rooms
	// in all, then the waiting room.
	visited := map[int]bool{first: true}
	for range 3 {
		e.advance(901 * time.Second)
		if _, id := room(); id > 0 {
			visited[id] = true
		}
	}
	if len(visited) < 2 {
		t.Errorf("no jump: rooms %v", visited)
	}
	e.advance(901 * time.Second)
	if typ, _ := room(); typ != -1 {
		t.Errorf("still in the rift after the last room")
	}
	if pos := e.position(ps[0]); !wait.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
		t.Errorf("leader at %v, not in the waiting room", pos)
	}
}
