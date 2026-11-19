package world

import (
	"errors"
	"testing"
)

func TestIDAllocatorDefaultRange(t *testing.T) {
	a := NewIDAllocator()
	id, err := a.Next()
	if err != nil || id != FirstRuntimeID || !id.IsRuntime() {
		t.Fatalf("first ID = %#x, %v; want %#x", id, err, FirstRuntimeID)
	}
	if ObjectID(0x10000000).IsRuntime() {
		t.Error("a persistent ID counts as runtime")
	}
}

func TestIDAllocatorExhaustion(t *testing.T) {
	a := newIDAllocator(100, 104, 2)
	for want := ObjectID(100); want <= 104; want++ {
		if got, err := a.Next(); err != nil || got != want {
			t.Fatalf("Next = %d, %v; want %d", got, err, want)
		}
	}
	if _, err := a.Next(); !errors.Is(err, ErrNoIDs) {
		t.Fatalf("Next on a full range: %v, want ErrNoIDs", err)
	}
	// With no fresh IDs left, released ones come back oldest first, even
	// below the reuse threshold.
	for _, id := range []ObjectID{102, 100} {
		if err := a.Release(id); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []ObjectID{102, 100} {
		if got, err := a.Next(); err != nil || got != want {
			t.Fatalf("Next = %d, %v; want %d", got, err, want)
		}
	}
	if _, err := a.Next(); !errors.Is(err, ErrNoIDs) {
		t.Errorf("Next: %v, want ErrNoIDs", err)
	}
	if a.InUse() != 5 {
		t.Errorf("InUse = %d, want 5", a.InUse())
	}
}

func TestIDAllocatorReuseThreshold(t *testing.T) {
	a := newIDAllocator(100, 1000, 2)
	for range 4 { // 100..103
		if _, err := a.Next(); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []ObjectID{100, 101} {
		if err := a.Release(id); err != nil {
			t.Fatal(err)
		}
	}
	// Two waiting is not more than the threshold: a fresh ID.
	if got, _ := a.Next(); got != 104 {
		t.Errorf("Next = %d, want 104", got)
	}
	if err := a.Release(102); err != nil {
		t.Fatal(err)
	}
	// Three waiting: the oldest comes back.
	if got, _ := a.Next(); got != 100 {
		t.Errorf("Next = %d, want 100", got)
	}
	if a.InUse() != 3 {
		t.Errorf("InUse = %d, want 3", a.InUse())
	}
}

func TestIDAllocatorBadRelease(t *testing.T) {
	a := newIDAllocator(100, 1000, 2)
	id, _ := a.Next()
	for _, bad := range []ObjectID{99, 101, 5000} {
		if err := a.Release(bad); !errors.Is(err, ErrNotAllocated) {
			t.Errorf("Release(%d): %v, want ErrNotAllocated", bad, err)
		}
	}
	if err := a.Release(id); err != nil {
		t.Fatal(err)
	}
	if err := a.Release(id); !errors.Is(err, ErrNotAllocated) {
		t.Errorf("second Release: %v, want ErrNotAllocated", err)
	}
	if a.InUse() != 0 {
		t.Errorf("InUse = %d, want 0", a.InUse())
	}
}

func TestIDAllocatorCompacts(t *testing.T) {
	a := newIDAllocator(1, 100000, 0)
	for range 5000 {
		id, err := a.Next()
		if err != nil {
			t.Fatal(err)
		}
		if err := a.Release(id); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.queue) > 2048 {
		t.Errorf("queue holds %d entries after steady reuse, want it compacted", len(a.queue))
	}
}

func TestGoid(t *testing.T) {
	mine := goid()
	if mine == 0 {
		t.Fatal("goid = 0")
	}
	other := make(chan uint64)
	go func() { other <- goid() }()
	if theirs := <-other; theirs == 0 || theirs == mine {
		t.Errorf("goid in another goroutine = %d, mine %d", theirs, mine)
	}
	if goid() != mine {
		t.Error("goid is not stable")
	}
}

func TestRegionBookkeeping(t *testing.T) {
	w := New(NewLoop(LoopOptions{}))
	objs := make([]*Base, 6)
	for i := range objs {
		b := NewBase(ObjectID(i+1), Position{X: 1000, Y: 1000, Z: 0})
		objs[i] = &b
		if _, err := w.Add(objs[i]); err != nil {
			t.Fatal(err)
		}
	}
	k, _ := RegionOf(1000, 1000, 0)
	r := w.regions[k.pack()]
	// Remove from the middle, the end and the start; slots must stay right.
	for _, i := range []int{2, 5, 0} {
		w.Remove(objs[i])
		for slot, b := range r.objects {
			if int(b.slot) != slot || b.region != r {
				t.Fatalf("after removing %d: object %d has slot %d at %d", i+1, b.id, b.slot, slot)
			}
		}
	}
	if len(r.objects) != 3 || len(r.viewers) != 0 {
		t.Errorf("region holds %d objects, %d viewers; want 3, 0", len(r.objects), len(r.viewers))
	}
	if len(r.neighbors) != 1 || r.neighbors[0] != r {
		t.Errorf("a lone region's neighbours = %d, want itself only", len(r.neighbors))
	}
	// A new region next to it links both ways.
	b := NewBase(99, Position{X: 1000 + RegionSize, Y: 1000, Z: 0})
	if _, err := w.Add(&b); err != nil {
		t.Fatal(err)
	}
	if len(r.neighbors) != 2 || len(b.region.neighbors) != 2 || b.region.neighbors[0] != b.region {
		t.Errorf("neighbours after a second region: %d and %d", len(r.neighbors), len(b.region.neighbors))
	}
}
