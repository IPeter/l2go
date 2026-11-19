package netutil

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type closer struct {
	name   string
	closed atomic.Int32
}

func (c *closer) Close() error {
	c.closed.Add(1)
	return nil
}

// B6: removing a client used to remove whichever client matched a nil
// session ID, usually the first one. IDs must remove exactly one item.
func TestRegistryRemovesTheRightItem(t *testing.T) {
	var r Registry[*closer]
	a, b, c := &closer{name: "a"}, &closer{name: "b"}, &closer{name: "c"}
	idA, _ := r.Add(a)
	idB, _ := r.Add(b)
	idC, _ := r.Add(c)
	if idA == idB || idB == idC || idA == idC {
		t.Fatalf("IDs are not unique: %d %d %d", idA, idB, idC)
	}

	r.Remove(idB)
	if got := r.Len(); got != 2 {
		t.Fatalf("Len = %d, want 2", got)
	}
	if _, ok := r.Get(idB); ok {
		t.Error("b is still registered")
	}
	for _, want := range []struct {
		id uint64
		v  *closer
	}{{idA, a}, {idC, c}} {
		if got, ok := r.Get(want.id); !ok || got != want.v {
			t.Errorf("Get(%d) = %v, %v; want %s", want.id, got, ok, want.v.name)
		}
	}

	r.Remove(idB) // removing twice is harmless
	if got := r.Len(); got != 2 {
		t.Errorf("Len after second Remove = %d, want 2", got)
	}
}

func TestRegistryCloseAll(t *testing.T) {
	var r Registry[*closer]
	a, b := &closer{}, &closer{}
	r.Add(a)
	r.Add(b)
	r.CloseAll()
	if a.closed.Load() != 1 || b.closed.Load() != 1 {
		t.Errorf("closed counts = %d, %d; want 1, 1", a.closed.Load(), b.closed.Load())
	}

	late := &closer{}
	if _, ok := r.Add(late); ok {
		t.Error("Add after CloseAll succeeded")
	}
	if late.closed.Load() != 0 {
		t.Error("Add closed the refused item; the caller owns it")
	}
}

// B8: the client lists were changed from many goroutines without a lock.
// Run with -race.
func TestRegistryConcurrent(t *testing.T) {
	var r Registry[*closer]
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			id, ok := r.Add(&closer{})
			if !ok {
				t.Error("Add failed")
				return
			}
			_ = r.Len()
			r.Get(id)
			r.Remove(id)
		})
	}
	wg.Wait()
	if got := r.Len(); got != 0 {
		t.Errorf("Len = %d, want 0", got)
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

func waitErr(t *testing.T, errc <-chan error) error {
	t.Helper()
	select {
	case err := <-errc:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("AcceptLoop did not return")
		return nil
	}
}

func TestAcceptLoopHandlesConnections(t *testing.T) {
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var wg sync.WaitGroup
	handled := make(chan struct{}, 2)
	errc := make(chan error, 1)
	go func() {
		errc <- AcceptLoop(ctx, ln, &wg, discardLogger(), func(c net.Conn) {
			defer c.Close()
			handled <- struct{}{}
		})
	}()

	for range 2 {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		select {
		case <-handled:
		case <-time.After(5 * time.Second):
			t.Fatal("connection was not handled")
		}
	}

	cancel()
	_ = ln.Close()
	if err := waitErr(t, errc); err != nil {
		t.Errorf("AcceptLoop after cancel = %v, want nil", err)
	}
	wg.Wait()
}

// B7: a closed listener used to make the accept loop spin forever and add
// nil connections to the client list.
func TestAcceptLoopStopsOnClosedListener(t *testing.T) {
	ln := listen(t)
	_ = ln.Close()

	var wg sync.WaitGroup
	calls := 0
	err := AcceptLoop(t.Context(), ln, &wg, discardLogger(), func(c net.Conn) { calls++ })
	if !errors.Is(err, net.ErrClosed) {
		t.Errorf("AcceptLoop = %v, want net.ErrClosed", err)
	}
	wg.Wait()
	if calls != 0 {
		t.Errorf("handler ran %d times for a closed listener", calls)
	}
}

// failingListener fails Accept a few times with a non-closed error.
type failingListener struct {
	net.Listener
	failures atomic.Int32
}

func (l *failingListener) Accept() (net.Conn, error) {
	if l.failures.Add(-1) >= 0 {
		return nil, errors.New("too many open files")
	}
	return l.Listener.Accept()
}

func TestAcceptLoopRetriesTemporaryErrors(t *testing.T) {
	inner := listen(t)
	ln := &failingListener{Listener: inner}
	ln.failures.Store(3)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var wg sync.WaitGroup
	handled := make(chan struct{}, 1)
	errc := make(chan error, 1)
	go func() {
		errc <- AcceptLoop(ctx, ln, &wg, discardLogger(), func(c net.Conn) {
			_ = c.Close()
			handled <- struct{}{}
		})
	}()

	c, err := net.Dial("tcp", inner.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	select {
	case <-handled:
	case <-time.After(5 * time.Second):
		t.Fatal("connection was not handled after Accept errors")
	}

	cancel()
	_ = inner.Close()
	if err := waitErr(t, errc); err != nil {
		t.Errorf("AcceptLoop = %v, want nil", err)
	}
	wg.Wait()
}

func TestAcceptLoopCancelDuringBackoff(t *testing.T) {
	inner := listen(t)
	ln := &failingListener{Listener: inner}
	ln.failures.Store(1 << 30)

	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	errc := make(chan error, 1)
	go func() {
		errc <- AcceptLoop(ctx, ln, &wg, discardLogger(), func(c net.Conn) { _ = c.Close() })
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := waitErr(t, errc); err != nil {
		t.Errorf("AcceptLoop = %v, want nil", err)
	}
}

var _ io.Closer = (*closer)(nil)
