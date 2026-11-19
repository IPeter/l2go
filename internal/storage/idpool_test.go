package storage_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
)

// fakeAllocator hands out IDs from next on, one after the other. NextIDs
// fails with err if it is set, and waits for gate first if that is set.
type fakeAllocator struct {
	mu    sync.Mutex
	next  int32
	calls int
	err   error
	gate  chan struct{}
}

func (a *fakeAllocator) NextID(ctx context.Context) (int32, error) {
	ids, err := a.NextIDs(ctx, 1)
	if err != nil {
		return 0, err
	}
	return ids[0], nil
}

func (a *fakeAllocator) NextIDs(ctx context.Context, n int) ([]int32, error) {
	a.mu.Lock()
	a.calls++
	gate := a.gate
	a.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return nil, a.err
	}
	ids := make([]int32, n)
	for i := range ids {
		ids[i] = a.next
		a.next++
	}
	return ids, nil
}

func (a *fakeAllocator) set(gate chan struct{}, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.gate, a.err = gate, err
}

func (a *fakeAllocator) callCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

func newTestPool(t *testing.T, a storage.IDAllocator, opts storage.IDPoolOptions) *storage.IDPool {
	t.Helper()
	opts.Logger = slog.New(slog.DiscardHandler)
	p := storage.NewIDPool(a, opts)
	if err := p.Fill(t.Context()); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	return p
}

// take takes n IDs and fails the test if the pool runs dry.
func take(t *testing.T, p *storage.IDPool, n int) []int32 {
	t.Helper()
	ids := make([]int32, n)
	for i := range ids {
		id, ok := p.TryNext()
		if !ok {
			t.Fatalf("TryNext %d of %d: pool empty", i+1, n)
		}
		ids[i] = id
	}
	return ids
}

func TestIDPoolRefillsInTheBackground(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &fakeAllocator{next: storage.FirstObjectID}
		p := newTestPool(t, a, storage.IDPoolOptions{Block: 10, Low: 3})
		defer p.Close()
		if n := p.Len(); n != 10 {
			t.Fatalf("Len after Fill = %d, want 10", n)
		}

		got := take(t, p, 7)
		synctest.Wait()
		if n := a.callCount(); n != 1 {
			t.Errorf("%d reservations with 3 IDs left, want only Fill's", n)
		}
		got = append(got, take(t, p, 1)...) // 2 left: below Low
		synctest.Wait()
		if n, calls := p.Len(), a.callCount(); n != 12 || calls != 2 {
			t.Errorf("after the refill: %d IDs and %d reservations, want 12 and 2", n, calls)
		}
		got = append(got, take(t, p, 12)...)

		for i, id := range got {
			if want := storage.FirstObjectID + int32(i); id != want { //nolint:gosec // small test index
				t.Fatalf("ID %d = %#x, want %#x: in order, each once", i, id, want)
			}
		}
	})
}

func TestIDPoolNeverBlocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &fakeAllocator{next: storage.FirstObjectID}
		p := newTestPool(t, a, storage.IDPoolOptions{Block: 4, Low: 2})
		defer p.Close()
		gate := make(chan struct{})
		a.set(gate, nil) // the database hangs

		take(t, p, 4) // the third one starts a refill, which hangs
		if _, ok := p.TryNext(); ok {
			t.Fatal("TryNext on an empty pool = true")
		}
		synctest.Wait()
		if _, ok := p.TryNext(); ok {
			t.Fatal("TryNext on an empty pool = true")
		}
		if n := a.callCount(); n != 2 {
			t.Errorf("%d reservations, want Fill's and one refill at a time", n)
		}

		close(gate) // the database answers
		synctest.Wait()
		if id, ok := p.TryNext(); !ok || id != storage.FirstObjectID+4 {
			t.Errorf("TryNext after the refill = %#x, %v, want %#x", id, ok, storage.FirstObjectID+4)
		}
	})
}

func TestIDPoolRetriesAfterRetryDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &fakeAllocator{next: storage.FirstObjectID}
		p := newTestPool(t, a, storage.IDPoolOptions{Block: 2, Low: 1, RetryDelay: time.Second})
		defer p.Close()
		a.set(nil, errors.New("database down"))

		take(t, p, 2) // the second one starts a refill, which fails
		synctest.Wait()
		if _, ok := p.TryNext(); ok {
			t.Fatal("TryNext on an empty pool = true")
		}
		synctest.Wait()
		if n := a.callCount(); n != 2 {
			t.Errorf("%d reservations right after a failure, want no retry yet", n)
		}

		time.Sleep(time.Second)
		a.set(nil, nil)
		if _, ok := p.TryNext(); ok { // still empty, but retries now
			t.Fatal("TryNext on an empty pool = true")
		}
		synctest.Wait()
		if id, ok := p.TryNext(); !ok || id != storage.FirstObjectID+2 {
			t.Errorf("TryNext after the retry = %#x, %v, want %#x", id, ok, storage.FirstObjectID+2)
		}
	})
}

func TestIDPoolStaysBelowRuntimeRange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &fakeAllocator{next: storage.FirstRuntimeObjectID - 3}
		p := newTestPool(t, a, storage.IDPoolOptions{Block: 5, Low: 1})
		defer p.Close()
		if n := p.Len(); n != 3 {
			t.Fatalf("Len = %d, want the 3 IDs below %#x", n, storage.FirstRuntimeObjectID)
		}
		got := take(t, p, 3) // the last one starts a refill: all its IDs are too high
		if got[2] != storage.FirstRuntimeObjectID-1 {
			t.Errorf("last ID = %#x, want %#x", got[2], storage.FirstRuntimeObjectID-1)
		}
		synctest.Wait()
		if id, ok := p.TryNext(); ok {
			t.Errorf("TryNext = %#x, want none from the runtime range", id)
		}

		high := &fakeAllocator{next: storage.FirstRuntimeObjectID}
		q := storage.NewIDPool(high, storage.IDPoolOptions{Logger: slog.New(slog.DiscardHandler)})
		defer q.Close()
		if err := q.Fill(t.Context()); err == nil {
			t.Error("Fill with only IDs of the runtime range succeeded")
		}
	})
}

func TestIDPoolFillError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		down := errors.New("database down")
		a := &fakeAllocator{next: storage.FirstObjectID, err: down}
		p := storage.NewIDPool(a, storage.IDPoolOptions{Logger: slog.New(slog.DiscardHandler)})
		defer p.Close()
		if err := p.Fill(t.Context()); !errors.Is(err, down) {
			t.Errorf("Fill error = %v, want %v", err, down)
		}
	})
}

func TestIDPoolTryNextN(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &fakeAllocator{next: storage.FirstObjectID}
		p := newTestPool(t, a, storage.IDPoolOptions{Block: 5, Low: 1})
		defer p.Close()
		a.set(make(chan struct{}), nil) // no refill finishes

		ids, ok := p.TryNextN(3)
		if !ok || len(ids) != 3 || ids[0] != storage.FirstObjectID || ids[2] != storage.FirstObjectID+2 {
			t.Fatalf("TryNextN(3) = %#x, %v", ids, ok)
		}
		if ids, ok := p.TryNextN(3); ok || ids != nil {
			t.Errorf("TryNextN(3) with 2 left = %#x, %v, want none", ids, ok)
		}
		if ids, ok := p.TryNextN(0); ok || ids != nil {
			t.Errorf("TryNextN(0) = %#x, %v, want none", ids, ok)
		}
		if n := p.Len(); n != 2 {
			t.Errorf("Len = %d, want the 2 IDs a failed TryNextN left", n)
		}
		if ids, ok := p.TryNextN(2); !ok || ids[0] != storage.FirstObjectID+3 || ids[1] != storage.FirstObjectID+4 {
			t.Errorf("TryNextN(2) = %#x, %v", ids, ok)
		}
	})
}

func TestIDPoolClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &fakeAllocator{next: storage.FirstObjectID}
		p := newTestPool(t, a, storage.IDPoolOptions{Block: 2, Low: 2})
		a.set(make(chan struct{}), nil) // the database hangs

		take(t, p, 1) // starts a refill, which hangs
		synctest.Wait()
		p.Close() // ends the hanging refill and waits for it

		if id, ok := p.TryNext(); !ok || id != storage.FirstObjectID+1 {
			t.Errorf("TryNext after Close = %#x, %v, want the ID left", id, ok)
		}
		if _, ok := p.TryNext(); ok {
			t.Error("TryNext on an empty, closed pool = true")
		}
		synctest.Wait()
		if n := a.callCount(); n != 2 {
			t.Errorf("%d reservations, want none after Close", n)
		}
	})
}

// TestIDPoolOnMemstore runs the pool with its defaults on a real allocator
// for a few blocks.
func TestIDPoolOnMemstore(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newTestPool(t, memstore.NewIDs(), storage.IDPoolOptions{})
		defer p.Close()
		const n = 2500
		next := storage.FirstObjectID
		for taken := 0; taken < n; {
			id, ok := p.TryNext()
			if !ok {
				synctest.Wait() // let the refill finish
				continue
			}
			if id != next {
				t.Fatalf("ID %d = %#x, want %#x", taken, id, next)
			}
			next++
			taken++
		}
	})
}
