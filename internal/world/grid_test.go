package world_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/world"
	"github.com/IPeter/l2go/internal/world/worldtest"
)

func TestMoveDelta(t *testing.T) {
	type objects = map[world.ObjectID]world.Position
	tests := []struct {
		name        string
		from, to    world.Position
		others      objects
		wantLeft    []world.ObjectID
		wantEntered []world.ObjectID
	}{
		{
			name: "east",
			from: at(100, 100, 8), to: at(101, 100, 8),
			others: objects{
				2: at(99, 100, 8),  // leaves
				3: at(102, 100, 8), // enters
				4: at(100, 101, 8), // stays
				5: at(103, 100, 8), // never in view
				6: at(99, 101, 9),  // leaves
				7: at(102, 99, 7),  // enters, a corner
			},
			wantLeft: []world.ObjectID{2, 6}, wantEntered: []world.ObjectID{3, 7},
		},
		{
			name: "diagonal",
			from: at(100, 100, 8), to: at(101, 101, 8),
			others: objects{
				2: at(99, 101, 8),  // leaves on X
				3: at(101, 99, 8),  // leaves on Y
				4: at(102, 100, 8), // enters on X
				5: at(100, 102, 8), // enters on Y
				6: at(102, 102, 8), // enters, the far corner
				7: at(99, 99, 8),   // leaves, the near corner
				8: at(101, 100, 8), // stays
				9: at(102, 99, 8),  // never in view
			},
			wantLeft: []world.ObjectID{2, 3, 7}, wantEntered: []world.ObjectID{4, 5, 6},
		},
		{
			name: "up a Z layer",
			from: at(100, 100, 8), to: at(100, 100, 9),
			others: objects{
				2: at(100, 100, 7),  // leaves
				3: at(101, 99, 7),   // leaves
				4: at(100, 100, 10), // enters
				5: at(99, 101, 10),  // enters
				6: at(100, 100, 9),  // stays, in the new region itself
				7: at(100, 100, 11), // never in view
			},
			wantLeft: []world.ObjectID{2, 3}, wantEntered: []world.ObjectID{4, 5},
		},
		{
			name: "west and down a Z layer",
			from: at(100, 100, 8), to: at(99, 100, 7),
			others: objects{
				2: at(101, 100, 8), // leaves on X
				3: at(100, 100, 9), // leaves on Z
				4: at(98, 100, 6),  // enters
				5: at(99, 100, 8),  // stays
				6: at(98, 101, 8),  // enters
			},
			wantLeft: []world.ObjectID{2, 3}, wantEntered: []world.ObjectID{4, 6},
		},
		{
			name: "one unit over a border",
			// x 75775 is the last unit of region X 100, 75776 the first of 101.
			from: world.Position{X: 75775, Y: -56320, Z: 1000}, to: world.Position{X: 75776, Y: -56320, Z: 1000},
			others: objects{
				2: at(99, 100, 8),  // leaves
				3: at(102, 100, 8), // enters
				4: at(101, 101, 9), // stays
			},
			wantLeft: []world.ObjectID{2}, wantEntered: []world.ObjectID{3},
		},
		{
			name: "over the origin",
			from: world.Position{X: -1, Y: 100, Z: 100}, to: world.Position{X: 0, Y: 100, Z: 100},
			others: objects{
				2: at(62, 128, 8), // leaves
				3: at(65, 128, 8), // enters
				4: at(64, 127, 7), // stays
			},
			wantLeft: []world.ObjectID{2}, wantEntered: []world.ObjectID{3},
		},
		{
			name: "far jump",
			from: at(100, 100, 8), to: at(120, 120, 8),
			others: objects{
				2: at(100, 100, 8),  // leaves
				3: at(101, 101, 9),  // leaves
				4: at(120, 121, 8),  // enters
				5: at(119, 119, 7),  // enters
				6: at(110, 110, 8),  // never in view
				7: at(120, 120, 10), // never in view
			},
			wantLeft: []world.ObjectID{2, 3}, wantEntered: []world.ObjectID{4, 5},
		},
		{
			name: "inside one region",
			from: at(100, 100, 8), to: offset(at(100, 100, 8), 500, -300, 200),
			others: objects{
				2: at(99, 100, 8),
				3: at(102, 100, 8),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t)
			mover := worldtest.NewObject(1, tt.from)
			add(t, w, mover)
			for id, pos := range tt.others {
				add(t, w, worldtest.NewObject(id, pos))
			}

			d, err := w.Move(mover, tt.to)
			if err != nil {
				t.Fatalf("Move: %v", err)
			}
			if got := ids(d.Left); !slices.Equal(got, tt.wantLeft) {
				t.Errorf("Left = %v, want %v", got, tt.wantLeft)
			}
			if got := ids(d.Entered); !slices.Equal(got, tt.wantEntered) {
				t.Errorf("Entered = %v, want %v", got, tt.wantEntered)
			}
			if d.Empty() != (len(tt.wantLeft) == 0 && len(tt.wantEntered) == 0) {
				t.Errorf("Empty = %v", d.Empty())
			}
			if got := mover.Position(); got != tt.to {
				t.Errorf("Position = %v, want %v", got, tt.to)
			}
			want, _ := world.RegionOf(tt.to.X, tt.to.Y, tt.to.Z)
			if got, ok := mover.Region(); !ok || got != want {
				t.Errorf("Region = %v, %v; want %v", got, ok, want)
			}
		})
	}
}

func TestAddAndRemoveDelta(t *testing.T) {
	w := newWorld(t)
	pos := at(100, 100, 8)
	same := worldtest.NewObject(2, offset(pos, 10, 0, 0))
	other := worldtest.NewObject(3, pos)
	if err := other.SetInstanceID(7); err != nil {
		t.Fatal(err)
	}
	near := worldtest.NewObject(4, at(99, 101, 9))
	far := worldtest.NewObject(5, at(102, 100, 8))
	add(t, w, same, other, near, far)

	o := worldtest.NewObject(1, pos)
	d, err := w.Add(o)
	if err != nil {
		t.Fatal(err)
	}
	if d.Subject != world.Object(o) {
		t.Errorf("Subject = %v, want the added object", d.Subject)
	}
	if got, want := ids(d.Entered), []world.ObjectID{2, 4}; !slices.Equal(got, want) {
		t.Errorf("Add: Entered = %v, want %v (same instance, in view)", got, want)
	}
	if len(d.Left) != 0 {
		t.Errorf("Add: Left = %v, want none", ids(d.Left))
	}
	if !o.InWorld() || w.Len() != 5 {
		t.Errorf("InWorld = %v, Len = %d; want true, 5", o.InWorld(), w.Len())
	}

	d = w.Remove(o)
	// Like Mobius, Remove reports every object around, whatever its instance.
	if got, want := ids(d.Left), []world.ObjectID{2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("Remove: Left = %v, want %v", got, want)
	}
	if len(d.Entered) != 0 {
		t.Errorf("Remove: Entered = %v, want none", ids(d.Entered))
	}
	if o.InWorld() || w.Len() != 4 {
		t.Errorf("after Remove: InWorld = %v, Len = %d; want false, 4", o.InWorld(), w.Len())
	}
	if d := w.Remove(o); d.Subject != nil || !d.Empty() {
		t.Errorf("removing twice = %+v, want an empty Delta", d)
	}

	// Teleport: change position and instance while out, then add again.
	if err := o.SetPosition(at(102, 101, 8)); err != nil {
		t.Fatal(err)
	}
	if err := o.SetInstanceID(7); err != nil {
		t.Fatal(err)
	}
	d, err = w.Add(o)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(d.Entered); len(got) != 0 {
		t.Errorf("Add in instance 7: Entered = %v, want none (3 is out of view)", got)
	}
}

func TestMoveDeltaInstances(t *testing.T) {
	w := newWorld(t)
	o := worldtest.NewObject(1, at(100, 100, 8))
	leaving := worldtest.NewObject(2, at(99, 100, 8))
	entering := worldtest.NewObject(3, at(102, 100, 8))
	for _, x := range []*worldtest.Object{leaving, entering} {
		if err := x.SetInstanceID(5); err != nil {
			t.Fatal(err)
		}
	}
	add(t, w, o, leaving, entering)

	d, err := w.Move(o, at(101, 100, 8))
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(d.Entered); len(got) != 0 {
		t.Errorf("Entered = %v, want none from another instance", got)
	}
	if got, want := ids(d.Left), []world.ObjectID{2}; !slices.Equal(got, want) {
		t.Errorf("Left = %v, want %v (unfiltered, as in Mobius)", got, want)
	}
}

func TestDeltaPairs(t *testing.T) {
	w := newWorld(t)
	pos := at(100, 100, 8)
	seer := worldtest.NewPlayer(2, pos)
	npc := worldtest.NewObject(3, pos)
	gm := worldtest.NewPlayer(4, pos)
	gm.GM = true
	ghost := worldtest.NewObject(5, pos) // an invisible NPC
	ghost.SetInvisible(true)
	add(t, w, seer, npc, gm, ghost)

	p := worldtest.NewPlayer(1, pos)
	d, err := w.Add(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []pair{{1, 2}, {1, 3}, {1, 4}, {2, 1}, {4, 1}}
	if got := sortedPairs(d.Appear()); !slices.Equal(got, want) {
		t.Errorf("Add player: Appear = %v, want %v", got, want)
	}
	if got := pairs(d.Disappear()); len(got) != 0 {
		t.Errorf("Add player: Disappear = %v, want none", got)
	}

	hidden := worldtest.NewPlayer(6, pos) // an invisible player, not a GM
	hidden.SetInvisible(true)
	d, err = w.Add(hidden)
	if err != nil {
		t.Fatal(err)
	}
	// It sees the visible ones; only the GM sees it.
	want = []pair{{4, 6}, {6, 1}, {6, 2}, {6, 3}, {6, 4}}
	if got := sortedPairs(d.Appear()); !slices.Equal(got, want) {
		t.Errorf("Add invisible player: Appear = %v, want %v", got, want)
	}

	// Moving away: losses are not filtered, like Mobius's DeleteObject.
	d, err = w.Move(p, at(110, 100, 8))
	if err != nil {
		t.Fatal(err)
	}
	want = []pair{{1, 2}, {1, 3}, {1, 4}, {1, 5}, {1, 6}, {2, 1}, {4, 1}, {6, 1}}
	if got := sortedPairs(d.Disappear()); !slices.Equal(got, want) {
		t.Errorf("Move away: Disappear = %v, want %v", got, want)
	}
	if got := pairs(d.Appear()); len(got) != 0 {
		t.Errorf("Move away: Appear = %v, want none", got)
	}

	// An NPC decays: only the viewers around lose it.
	d = w.Remove(npc)
	want = []pair{{2, 3}, {4, 3}, {6, 3}}
	if got := sortedPairs(d.Disappear()); !slices.Equal(got, want) {
		t.Errorf("Remove NPC: Disappear = %v, want %v", got, want)
	}
}

func TestDeltaPairOrder(t *testing.T) {
	w := newWorld(t)
	other := worldtest.NewPlayer(2, at(100, 100, 8))
	add(t, w, other)

	p := worldtest.NewPlayer(1, at(100, 100, 8))
	d, err := w.Add(p)
	if err != nil {
		t.Fatal(err)
	}
	// Per object, the subject's own sighting comes first (Mobius order).
	if got, want := pairs(d.Appear()), []pair{{1, 2}, {2, 1}}; !slices.Equal(got, want) {
		t.Errorf("Appear = %v, want %v", got, want)
	}
	d = w.Remove(p)
	if got, want := pairs(d.Disappear()), []pair{{1, 2}, {2, 1}}; !slices.Equal(got, want) {
		t.Errorf("Disappear = %v, want %v", got, want)
	}

	// Breaking out of the iteration stops it.
	n := 0
	for range d.Disappear() {
		n++
		break
	}
	if n != 1 {
		t.Errorf("iterations after break = %d, want 1", n)
	}
}

func TestVisibilityRule(t *testing.T) {
	w := newWorld(t)
	pos := at(100, 100, 8)
	a := worldtest.NewPlayer(1, pos)
	b := worldtest.NewPlayer(2, pos)
	quest := worldtest.NewObject(3, pos)
	add(t, w, a, b)

	// Show the quest NPC to player 1 only (Mobius OnNpcCanBeSeen).
	w.SetVisibilityRule(func(v world.Viewer, o world.Object) bool {
		if o.ID() == 3 {
			return v.ID() == 1
		}
		return world.VisibleByDefault(v, o)
	})
	d, err := w.Add(quest)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sortedPairs(d.Appear()), []pair{{1, 3}}; !slices.Equal(got, want) {
		t.Errorf("Appear = %v, want %v", got, want)
	}
	if !w.CanSee(a, quest) || w.CanSee(b, quest) {
		t.Error("CanSee does not follow the rule")
	}
	w.SetVisibilityRule(nil)
	if !w.CanSee(b, quest) {
		t.Error("CanSee after restoring the default rule = false")
	}
}

func TestVisible(t *testing.T) {
	w := newWorld(t)
	center := at(100, 100, 8)
	me := worldtest.NewPlayer(1, center)
	near := worldtest.NewObject(2, offset(center, 300, 400, 0)) // 500 away
	edge := worldtest.NewPlayer(3, offset(center, 0, 0, 600))   // 600 away
	far := worldtest.NewObject(4, at(101, 101, 9))              // in view, far
	out := worldtest.NewObject(5, at(102, 100, 8))              // out of view
	elsewhere := worldtest.NewPlayer(6, center)                 // another instance
	if err := elsewhere.SetInstanceID(2); err != nil {
		t.Fatal(err)
	}
	walker := worldtest.NewPlayer(7, offset(center, 0, 1000, 0)) // 1000 away
	add(t, w, me, near, edge, far, out, elsewhere, walker)

	tests := []struct {
		name string
		got  []world.ObjectID
		want []world.ObjectID
	}{
		{"Visible", seqIDs(w.Visible(me)), []world.ObjectID{2, 3, 4, 7}},
		{"VisibleInRange 600", seqIDs(w.VisibleInRange(me, 600)), []world.ObjectID{2, 3}},
		{"VisibleInRange 599", seqIDs(w.VisibleInRange(me, 599)), []world.ObjectID{2}},
		{"VisibleInRange 0", seqIDs(w.VisibleInRange(me, 0)), nil},
		{"VisibleInRange -1", seqIDs(w.VisibleInRange(me, -1)), nil},
		{"VisiblePlayers", seqIDs(w.VisiblePlayers(me)), []world.ObjectID{3, 7}},
		{"VisiblePlayersInRange 1000", seqIDs(w.VisiblePlayersInRange(me, 1000)), []world.ObjectID{3, 7}},
		{"VisiblePlayersInRange 999", seqIDs(w.VisiblePlayersInRange(me, 999)), []world.ObjectID{3}},
		{"Visible from another instance", seqIDs(w.Visible(elsewhere)), nil},
		{"Around in instance 0", seqIDs(w.Around(center, 0)), []world.ObjectID{1, 2, 3, 4, 7}},
		{"Around in instance 2", seqIDs(w.Around(center, 2)), []world.ObjectID{6}},
		// Objects not in the world look around their position, even from a
		// region nobody created yet.
		{"Visible from outside the world", seqIDs(w.Visible(worldtest.NewObject(99, center))), []world.ObjectID{1, 2, 3, 4, 7}},
		{"Visible from an empty region", seqIDs(w.Visible(worldtest.NewObject(99, at(101, 100, 8)))), []world.ObjectID{1, 2, 3, 4, 5, 7}},
		{"Around outside the world", seqIDs(w.Around(world.Position{X: world.WorldXMax}, 0)), nil},
	}
	for _, tt := range tests {
		if !slices.Equal(tt.got, tt.want) {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}

	n := 0
	for range w.Visible(me) {
		n++
		break
	}
	if n != 1 {
		t.Errorf("iterations after break = %d, want 1", n)
	}
}

func TestWorldErrors(t *testing.T) {
	w := newWorld(t)
	o := worldtest.NewObject(1, at(100, 100, 8))
	add(t, w, o)

	if _, err := w.Add(o); !errors.Is(err, world.ErrInWorld) {
		t.Errorf("Add twice: %v, want ErrInWorld", err)
	}
	if _, err := w.Add(worldtest.NewObject(1, at(50, 50, 8))); !errors.Is(err, world.ErrDuplicateID) {
		t.Errorf("Add with a used ID: %v, want ErrDuplicateID", err)
	}
	if _, err := w.Add(worldtest.NewObject(0, at(50, 50, 8))); !errors.Is(err, world.ErrInvalidID) {
		t.Errorf("Add with ID 0: %v, want ErrInvalidID", err)
	}
	if _, err := w.Add(worldtest.NewObject(2, world.Position{X: world.WorldXMax})); !errors.Is(err, world.ErrOutOfWorld) {
		t.Errorf("Add outside: %v, want ErrOutOfWorld", err)
	}

	before := o.Position()
	if _, err := w.Move(o, world.Position{Y: world.WorldYMin - 1}); !errors.Is(err, world.ErrOutOfWorld) {
		t.Errorf("Move outside: %v, want ErrOutOfWorld", err)
	}
	if o.Position() != before {
		t.Errorf("a failed Move changed the position to %v", o.Position())
	}

	if err := o.SetPosition(at(1, 1, 1)); !errors.Is(err, world.ErrInWorld) {
		t.Errorf("SetPosition in the world: %v, want ErrInWorld", err)
	}
	if err := o.SetInstanceID(3); !errors.Is(err, world.ErrInWorld) {
		t.Errorf("SetInstanceID in the world: %v, want ErrInWorld", err)
	}
	if err := o.SetID(9); !errors.Is(err, world.ErrInWorld) {
		t.Errorf("SetID in the world: %v, want ErrInWorld", err)
	}
	o.SetHeading(1234)
	if o.Position().Heading != 1234 {
		t.Error("SetHeading had no effect")
	}

	// Moving an object that is not in the world only sets its position.
	out := worldtest.NewObject(3, at(1, 1, 1))
	d, err := w.Move(out, at(2, 2, 2))
	if err != nil || !d.Empty() || out.Position() != at(2, 2, 2) {
		t.Errorf("Move out of the world = %+v, %v; position %v", d, err, out.Position())
	}
	if out.InWorld() {
		t.Error("Move put the object into the world")
	}
}

func TestFind(t *testing.T) {
	w := newWorld(t)
	o := worldtest.NewObject(42, at(100, 100, 8))
	if _, ok := w.Find(42); ok {
		t.Error("Find before Add succeeded")
	}
	add(t, w, o)
	if got, ok := w.Find(42); !ok || got != world.Object(o) {
		t.Errorf("Find = %v, %v; want the object", got, ok)
	}
	w.Remove(o)
	if _, ok := w.Find(42); ok {
		t.Error("Find after Remove succeeded")
	}
}

// TestGridMatchesBruteForce moves objects at random and checks every Delta
// and every Visible result against a brute-force search.
func TestGridMatchesBruteForce(t *testing.T) {
	w := newWorld(t)
	rng := rand.New(rand.NewPCG(7, 11))
	const n = 120
	randomPos := func() world.Position {
		// 5×5 regions and 3 layers around region (100, 100, 8), plus a margin.
		base := at(98, 98, 7)
		return offset(base, rng.Int32N(6*world.RegionSize)-world.RegionSize/2,
			rng.Int32N(6*world.RegionSize)-world.RegionSize/2, rng.Int32N(4*world.RegionZSize)-world.RegionZSize/2)
	}
	// The test doubles' Base methods, reached through world.Object.
	type based interface {
		InWorld() bool
		InstanceID() int32
		SetInstanceID(int32) error
	}
	inWorld := func(o world.Object) bool { return o.(based).InWorld() }
	instance := func(o world.Object) int32 { return o.(based).InstanceID() }

	objs := make([]world.Object, n)
	for i := range objs {
		id := world.ObjectID(i + 1)
		if i%3 == 0 {
			objs[i] = worldtest.NewPlayer(id, randomPos())
		} else {
			objs[i] = worldtest.NewObject(id, randomPos())
		}
		if i%10 == 9 {
			if err := objs[i].(based).SetInstanceID(1); err != nil {
				t.Fatal(err)
			}
		}
		add(t, w, objs[i])
	}

	// brute returns what o sees: same instance, adjacent regions.
	brute := func(o world.Object) []world.ObjectID {
		key, _ := world.RegionOf(o.Position().X, o.Position().Y, o.Position().Z)
		var out []world.ObjectID
		for _, x := range objs {
			if x == o || !inWorld(x) || instance(x) != instance(o) {
				continue
			}
			xk, _ := world.RegionOf(x.Position().X, x.Position().Y, x.Position().Z)
			if xk.Adjacent(key) {
				out = append(out, x.ID())
			}
		}
		slices.Sort(out)
		return out
	}

	for round := range 3000 {
		o := objs[rng.IntN(n)]
		before := brute(o)
		var d world.Delta
		switch r := rng.IntN(20); {
		case !inWorld(o):
			var err error
			if d, err = w.Add(o); err != nil {
				t.Fatalf("round %d: Add: %v", round, err)
			}
			before = nil
		case r == 0:
			d = w.Remove(o)
		default:
			// Mostly short steps, sometimes a jump.
			pos := randomPos()
			if r > 2 {
				p := o.Position()
				pos = offset(p, rng.Int32N(1201)-600, rng.Int32N(1201)-600, rng.Int32N(801)-400)
			}
			var err error
			if d, err = w.Move(o, pos); err != nil {
				t.Fatalf("round %d: Move to %v: %v", round, pos, err)
			}
		}

		after := brute(o)
		if !inWorld(o) {
			after = nil
		}
		left := ids(d.Left)
		entered := ids(d.Entered)
		// after = (before - Left) + Entered, and the two lists don't overlap
		// what stays.
		want := slices.DeleteFunc(slices.Clone(before), func(id world.ObjectID) bool {
			_, found := slices.BinarySearch(left, id)
			return found
		})
		want = append(want, entered...)
		slices.Sort(want)
		if !slices.Equal(want, after) {
			t.Fatalf("round %d: object %d: before %v, Left %v, Entered %v; after %v", round, o.ID(), before, left, entered, after)
		}
		for _, id := range entered {
			if _, found := slices.BinarySearch(before, id); found {
				t.Fatalf("round %d: %d entered but was already in view", round, id)
			}
		}
		if inWorld(o) {
			if got := seqIDs(w.Visible(o)); !slices.Equal(got, after) {
				t.Fatalf("round %d: Visible(%d) = %v, want %v", round, o.ID(), got, after)
			}
		}
	}
}
