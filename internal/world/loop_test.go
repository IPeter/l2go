package world_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IPeter/l2go/internal/world"
	"github.com/IPeter/l2go/internal/world/worldtest"
)

// newFakeLoop returns an idle loop on a fake clock, driven by step.
func newFakeLoop(t *testing.T) (*world.Loop, *world.FakeClock) {
	t.Helper()
	clk := world.NewFakeClock(start)
	return world.NewLoop(world.LoopOptions{Clock: clk}), clk
}

// ms returns the loop's time as milliseconds since start.
func ms(l *world.Loop) int64 { return l.Now().Sub(start).Milliseconds() }

// runLoop starts Run in a goroutine and waits until it is running. stop
// cancels it and waits for Run to return; cleanup calls it too.
func runLoop(t *testing.T, opts world.LoopOptions) (l *world.Loop, ctx context.Context, stop func()) {
	t.Helper()
	l = world.NewLoop(opts)
	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() { errc <- l.Run(ctx) }()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cancel()
			if err := <-errc; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	t.Cleanup(stop)
	if err := l.Call(ctx, func() {}); err != nil {
		t.Fatalf("waiting for Run: %v", err)
	}
	return l, ctx, stop
}

func TestTimersRunInDeadlineOrder(t *testing.T) {
	l, clk := newFakeLoop(t)
	var got []string
	record := func(name string) func() {
		return func() { got = append(got, fmt.Sprintf("%s@%d", name, ms(l))) }
	}
	l.After(300*time.Millisecond, record("c"))
	l.After(100*time.Millisecond, record("a"))
	l.After(200*time.Millisecond, record("b1"))
	l.After(200*time.Millisecond, record("b2")) // same deadline: creation order

	step(t, l, clk, 250*time.Millisecond)
	want := []string{"a@100", "b1@200", "b2@200"}
	if !slices.Equal(got, want) {
		t.Errorf("after 250 ms: %v, want %v", got, want)
	}
	step(t, l, clk, 50*time.Millisecond)
	want = append(want, "c@300")
	if !slices.Equal(got, want) {
		t.Errorf("after 300 ms: %v, want %v", got, want)
	}
}

func TestTimersAndTicksInterleave(t *testing.T) {
	l, clk := newFakeLoop(t)
	var got []string
	l.OnTick(func(now time.Time) { got = append(got, fmt.Sprintf("tick@%d", now.Sub(start).Milliseconds())) })
	timer := func() { got = append(got, fmt.Sprintf("timer@%d", ms(l))) }
	l.After(150*time.Millisecond, timer)
	l.After(200*time.Millisecond, timer) // due with a tick: the timer first

	step(t, l, clk, 250*time.Millisecond)
	want := []string{"tick@100", "timer@150", "timer@200", "tick@200"}
	if !slices.Equal(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestEveryCatchesUp(t *testing.T) {
	l, clk := newFakeLoop(t)
	var runs []int64
	id := l.Every(100*time.Millisecond, func() { runs = append(runs, ms(l)) })

	step(t, l, clk, time.Second)
	want := []int64{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}
	if !slices.Equal(runs, want) {
		t.Errorf("runs at %v, want %v", runs, want)
	}
	if !l.Cancel(id) {
		t.Error("Cancel of a pending Every timer = false")
	}
	step(t, l, clk, time.Second)
	if len(runs) != len(want) {
		t.Errorf("runs after Cancel: %d, want %d", len(runs), len(want))
	}
	if l.Cancel(id) {
		t.Error("second Cancel = true")
	}
}

func TestEveryCancelsItself(t *testing.T) {
	l, clk := newFakeLoop(t)
	n := 0
	var id world.TimerID
	id = l.Every(100*time.Millisecond, func() {
		n++
		if n == 3 && !l.Cancel(id) {
			t.Error("self-Cancel = false")
		}
	})
	step(t, l, clk, time.Second)
	if n != 3 {
		t.Errorf("runs = %d, want 3", n)
	}
}

func TestAfterCancel(t *testing.T) {
	l, clk := newFakeLoop(t)
	ran := false
	id := l.After(100*time.Millisecond, func() { ran = true })
	if id == 0 {
		t.Fatal("After returned the zero TimerID")
	}
	if !l.Cancel(id) {
		t.Error("Cancel of a pending timer = false")
	}
	if l.Cancel(id) || l.Cancel(0) || l.Cancel(12345) {
		t.Error("Cancel of a cancelled, zero or unknown ID = true")
	}
	step(t, l, clk, time.Second)
	if ran {
		t.Error("a cancelled timer ran")
	}

	id = l.After(10*time.Millisecond, func() {})
	step(t, l, clk, 10*time.Millisecond)
	if l.Cancel(id) {
		t.Error("Cancel of a timer that already ran = true")
	}
}

func TestTimersCountFromLoopTime(t *testing.T) {
	l, clk := newFakeLoop(t)
	var chained, immediate int64 = -1, -1
	// A timer set from a timer counts from the first one's scheduled time,
	// so the chain doesn't drift even when the loop runs late.
	l.After(100*time.Millisecond, func() {
		l.After(100*time.Millisecond, func() { chained = ms(l) })
	})
	l.After(-time.Second, func() { immediate = ms(l) })

	step(t, l, clk, 250*time.Millisecond)
	if chained != 200 {
		t.Errorf("chained timer ran at %d ms, want 200", chained)
	}
	if immediate != 0 {
		t.Errorf("negative After ran at %d ms, want 0", immediate)
	}
	if got := l.Now(); !got.Equal(clk.Now()) {
		t.Errorf("Now outside events = %v, want the clock's %v", got, clk.Now())
	}
}

func TestTicksCatchUpIsBounded(t *testing.T) {
	l, clk := newFakeLoop(t)
	var ticks []int64
	l.OnTick(func(now time.Time) { ticks = append(ticks, now.Sub(start).Milliseconds()) })

	step(t, l, clk, time.Second)
	if want := []int64{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}; !slices.Equal(ticks, want) {
		t.Errorf("ticks = %v, want %v", ticks, want)
	}

	// 20 s late: only the last 100 ticks run.
	ticks = nil
	step(t, l, clk, 20*time.Second)
	if len(ticks) != 100 {
		t.Fatalf("after a 20 s gap: %d ticks, want 100", len(ticks))
	}
	if ticks[0] != 11100 || ticks[99] != 21000 {
		t.Errorf("after a 20 s gap: ticks from %d to %d ms, want 11100 to 21000", ticks[0], ticks[99])
	}
}

func TestTickHooks(t *testing.T) {
	l, clk := newFakeLoop(t)
	var got []string
	l.OnTickEnd(func(time.Time) { got = append(got, "end") })
	a := l.OnTick(func(time.Time) { got = append(got, "a") })
	l.OnTick(func(time.Time) { got = append(got, "b") })
	var once world.HookID
	once = l.OnTick(func(time.Time) {
		got = append(got, "once")
		l.RemoveHook(once)
	})

	step(t, l, clk, 100*time.Millisecond)
	if want := []string{"a", "b", "once", "end"}; !slices.Equal(got, want) {
		t.Errorf("first tick = %v, want %v", got, want)
	}
	if !l.RemoveHook(a) {
		t.Error("RemoveHook of a hook = false")
	}
	if l.RemoveHook(a) || l.RemoveHook(0) {
		t.Error("RemoveHook of a removed or zero hook = true")
	}
	got = nil
	step(t, l, clk, 100*time.Millisecond)
	if want := []string{"b", "end"}; !slices.Equal(got, want) {
		t.Errorf("second tick = %v, want %v", got, want)
	}
}

func TestPanicsAreRecovered(t *testing.T) {
	l, clk := newFakeLoop(t)
	var got []string
	var recovered any
	if !l.PostGuarded(func() { panic("boom") }, func(v any) { recovered = v }) {
		t.Fatal("PostGuarded = false")
	}
	l.Post(func() { got = append(got, "after command panic") })
	l.After(10*time.Millisecond, func() { panic("timer") })
	l.After(20*time.Millisecond, func() { got = append(got, "after timer panic") })
	n := 0
	l.Every(100*time.Millisecond, func() {
		n++
		panic("every")
	})
	l.OnTick(func(time.Time) { panic("hook") })
	l.OnTick(func(time.Time) { got = append(got, "after hook panic") })

	step(t, l, clk, 200*time.Millisecond)
	want := []string{"after timer panic", "after hook panic", "after hook panic", "after command panic"}
	if !slices.Equal(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
	if recovered != "boom" {
		t.Errorf("panic handler got %v, want boom", recovered)
	}
	if n != 2 {
		t.Errorf("a panicking Every timer ran %d times, want 2 (it keeps running)", n)
	}
}

func TestCallReturnsPanicError(t *testing.T) {
	l, ctx, _ := runLoop(t, world.LoopOptions{})
	err := l.Call(ctx, func() { panic("call boom") })
	perr, ok := errors.AsType[*world.PanicError](err)
	if !ok || perr.Value != "call boom" || len(perr.Stack) == 0 {
		t.Fatalf("Call = %v, want a *PanicError with the value and a stack", err)
	}
	if err := l.Call(ctx, func() {}); err != nil {
		t.Errorf("Call after a panic: %v", err)
	}
}

func TestPostAndCallFromManyGoroutines(t *testing.T) {
	l, ctx, _ := runLoop(t, world.LoopOptions{InboxSize: 16})
	const goroutines, each = 8, 300

	// Loop-only state: no locks, the race detector watches it.
	counter, outOfOrder := 0, 0
	last := make([]int, goroutines)
	for g := range last {
		last[g] = -1
	}
	command := func(g, i int) func() {
		return func() {
			if i <= last[g] {
				outOfOrder++
			}
			last[g] = i
			counter++
		}
	}

	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range each {
				if i%3 == 0 {
					if err := l.Call(ctx, command(g, i)); err != nil {
						t.Errorf("Call: %v", err)
						return
					}
					continue
				}
				for !l.Post(command(g, i)) { // full inbox: try again
					if ctx.Err() != nil {
						return
					}
					runtime.Gosched()
				}
			}
		})
	}
	wg.Wait()

	var gotCounter, gotOutOfOrder int
	if err := l.Call(ctx, func() { gotCounter, gotOutOfOrder = counter, outOfOrder }); err != nil {
		t.Fatal(err)
	}
	if gotCounter != goroutines*each {
		t.Errorf("commands run = %d, want %d", gotCounter, goroutines*each)
	}
	if gotOutOfOrder != 0 {
		t.Errorf("%d commands ran out of their goroutine's order", gotOutOfOrder)
	}
}

func TestStoppedLoop(t *testing.T) {
	l, _, stop := runLoop(t, world.LoopOptions{})
	stop()
	select {
	case <-l.Done():
	default:
		t.Fatal("Done is open after Run returned")
	}
	if l.Post(func() {}) {
		t.Error("Post after stop = true")
	}
	if err := l.Call(context.Background(), func() {}); !errors.Is(err, world.ErrStopped) {
		t.Errorf("Call after stop: %v, want ErrStopped", err)
	}
	if err := l.Run(context.Background()); !errors.Is(err, world.ErrStopped) {
		t.Errorf("Run after stop: %v, want ErrStopped", err)
	}
	if err := l.Step(); !errors.Is(err, world.ErrStopped) {
		t.Errorf("Step after stop: %v, want ErrStopped", err)
	}
}

func TestRunWhileRunning(t *testing.T) {
	l, _, _ := runLoop(t, world.LoopOptions{})
	if err := l.Run(context.Background()); !errors.Is(err, world.ErrRunning) {
		t.Errorf("second Run: %v, want ErrRunning", err)
	}
	if err := l.Step(); !errors.Is(err, world.ErrRunning) {
		t.Errorf("Step while running: %v, want ErrRunning", err)
	}
}

func TestPostWhenInboxFull(t *testing.T) {
	l := world.NewLoop(world.LoopOptions{InboxSize: 2})
	ran := 0
	for i := range 2 {
		if !l.Post(func() { ran++ }) {
			t.Fatalf("Post %d on a free inbox = false", i+1)
		}
	}
	if l.Post(func() { ran++ }) {
		t.Error("Post on a full inbox = true")
	}
	if err := l.Step(); err != nil {
		t.Fatal(err)
	}
	if ran != 2 {
		t.Errorf("commands run = %d, want 2", ran)
	}
}

func TestCallContextDone(t *testing.T) {
	l := world.NewLoop(world.LoopOptions{}) // never runs
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := l.Call(ctx, func() {}); !errors.Is(err, context.Canceled) {
		t.Errorf("Call with a cancelled context: %v, want context.Canceled", err)
	}
	ctx, cancel = context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if err := l.Call(ctx, func() {}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Call on a loop that never runs: %v, want context.DeadlineExceeded", err)
	}
}

func TestOnLoop(t *testing.T) {
	l, ctx, _ := runLoop(t, world.LoopOptions{})
	if l.OnLoop() {
		t.Error("OnLoop on the test goroutine = true")
	}
	var on, nested bool
	var nestedErr error
	err := l.Call(ctx, func() {
		on = l.OnLoop()
		nestedErr = l.Call(ctx, func() { nested = true }) // runs at once
	})
	if err != nil || !on || !nested || nestedErr != nil {
		t.Errorf("Call = %v, OnLoop = %v, nested Call ran %v with %v", err, on, nested, nestedErr)
	}

	idle, _ := newFakeLoop(t)
	var onStep bool
	idle.Post(func() { onStep = idle.OnLoop() })
	if err := idle.Step(); err != nil {
		t.Fatal(err)
	}
	if !onStep || idle.OnLoop() {
		t.Errorf("OnLoop inside Step = %v, after Step = %v; want true, false", onStep, idle.OnLoop())
	}
}

func TestCheckOwner(t *testing.T) {
	var mu sync.Mutex
	var misuses []string
	l := world.NewLoop(world.LoopOptions{
		CheckOwner: true,
		OnMisuse: func(op string) {
			mu.Lock()
			defer mu.Unlock()
			misuses = append(misuses, op)
		},
	})
	w := world.New(l) // before Run: any goroutine may set up
	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() { errc <- l.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-errc; err != nil {
			t.Errorf("Run: %v", err)
		}
	}()
	if err := l.Call(ctx, func() {}); err != nil {
		t.Fatal(err)
	}

	// Off the loop: reported and refused.
	if id := l.After(time.Second, func() {}); id != 0 {
		t.Errorf("After off the loop = %d, want 0", id)
	}
	if _, err := w.Add(worldtest.NewObject(1, at(100, 100, 8))); !errors.Is(err, world.ErrNotOnLoop) {
		t.Errorf("Add off the loop: %v, want ErrNotOnLoop", err)
	}
	mu.Lock()
	got := slices.Clone(misuses)
	mu.Unlock()
	if want := []string{"Loop.After", "World.Add"}; !slices.Equal(got, want) {
		t.Errorf("misuses = %v, want %v", got, want)
	}

	// On the loop: fine, and nothing reported.
	var id world.TimerID
	var addErr error
	err := l.Call(ctx, func() {
		id = l.After(time.Second, func() {})
		_, addErr = w.Add(worldtest.NewObject(1, at(100, 100, 8)))
	})
	if err != nil || id == 0 || addErr != nil {
		t.Errorf("on the loop: Call %v, After %d, Add %v", err, id, addErr)
	}
	mu.Lock()
	n := len(misuses)
	mu.Unlock()
	if n != 2 {
		t.Errorf("misuses after on-loop calls = %d, want 2", n)
	}
}

func TestLoopInSynctestBubble(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := world.NewLoop(world.LoopOptions{}) // the real clock runs on bubble time
		begin := time.Now()
		ticks := 0
		var fired time.Duration = -1
		l.OnTick(func(time.Time) { ticks++ })
		l.After(250*time.Millisecond, func() { fired = l.Now().Sub(begin) })

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan struct{})
		go func() {
			defer close(done)
			if err := l.Run(ctx); err != nil {
				t.Errorf("Run: %v", err)
			}
		}()

		time.Sleep(time.Second)
		synctest.Wait()
		var gotTicks int
		var gotFired time.Duration
		if err := l.Call(ctx, func() { gotTicks, gotFired = ticks, fired }); err != nil {
			t.Fatal(err)
		}
		if gotTicks != 10 {
			t.Errorf("ticks in 1 s = %d, want 10", gotTicks)
		}
		if gotFired != 250*time.Millisecond {
			t.Errorf("After(250 ms) ran at %v", gotFired)
		}
		cancel()
		<-done
	})
}

func TestFakeClock(t *testing.T) {
	clk := world.NewFakeClock(start)
	tm := clk.NewTimer(100 * time.Millisecond)
	clk.Advance(99 * time.Millisecond)
	select {
	case <-tm.C():
		t.Fatal("the timer fired early")
	default:
	}
	clk.Advance(time.Millisecond)
	select {
	case got := <-tm.C():
		if !got.Equal(start.Add(100 * time.Millisecond)) {
			t.Errorf("fired with %v", got)
		}
	default:
		t.Fatal("the timer did not fire")
	}

	tm.Reset(50 * time.Millisecond)
	tm.Stop()
	clk.Advance(time.Second)
	select {
	case <-tm.C():
		t.Fatal("a stopped timer fired")
	default:
	}

	tm.Reset(0)
	select {
	case <-tm.C():
	default:
		t.Fatal("Reset(0) did not fire at once")
	}

	tm.Reset(10 * time.Millisecond)
	clk.Advance(10 * time.Millisecond)
	tm.Reset(10 * time.Millisecond) // drops the value not yet received
	select {
	case <-tm.C():
		t.Fatal("a stale value survived Reset")
	default:
	}

	clk.Set(start)
	if !clk.Now().Equal(start) {
		t.Errorf("Now after Set = %v", clk.Now())
	}
}
