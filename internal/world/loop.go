package world

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// Defaults for LoopOptions.
const (
	// DefaultTickInterval is Mobius's game tick.
	DefaultTickInterval = 100 * time.Millisecond
	// DefaultInboxSize bounds the commands waiting for the loop.
	DefaultInboxSize = 8192
)

const (
	// maxTickCatchUp bounds the missed ticks that run back to back after the
	// loop fell behind; older ones are dropped.
	maxTickCatchUp = 100
	// maxBatch bounds the commands run in a row before the loop rearms its
	// timer and looks at its context again.
	maxBatch = 256
)

// Errors of the loop.
var (
	// ErrStopped means the loop has stopped (Run returned): the command did
	// not run.
	ErrStopped = errors.New("world: loop stopped")
	// ErrRunning means Run or Step was called while the loop was running.
	ErrRunning = errors.New("world: loop already running")
	// ErrNotOnLoop means LoopOptions.CheckOwner caught a loop-only method
	// called from another goroutine; the call did nothing.
	ErrNotOnLoop = errors.New("world: loop-only method called off the loop")
)

// PanicError is what Call returns when its function panicked.
type PanicError struct {
	Value any    // the value passed to panic
	Stack []byte // the stack of the panicking goroutine
}

func (e *PanicError) Error() string { return fmt.Sprintf("world: command panicked: %v", e.Value) }

// LoopOptions configure a Loop.
type LoopOptions struct {
	// Clock is the time source. Nil means RealClock.
	Clock Clock
	// TickInterval is the time between ticks. Zero means
	// DefaultTickInterval.
	TickInterval time.Duration
	// InboxSize bounds the commands waiting for the loop. Zero means
	// DefaultInboxSize.
	InboxSize int
	// Logger gets recovered panics and warnings. Nil discards them.
	Logger *slog.Logger
	// CheckOwner makes loop-only methods check, while Run is active, that
	// they run on the loop goroutine. A call from elsewhere is reported and
	// does nothing. Each check costs about a microsecond: use it in tests
	// and debug builds.
	CheckOwner bool
	// OnMisuse is called, on the offending goroutine, when CheckOwner
	// catches a loop-only method called off the loop; op names the method.
	// Nil logs an error with the stack.
	OnMisuse func(op string)
}

const (
	loopIdle int32 = iota
	loopRunning
	loopStepping
	loopStopped
)

type command struct {
	fn      func()
	onPanic func(v any)
	done    chan error // set by Call
}

// Loop is the single owner of the world (decisions W1, W2): one goroutine
// running the commands other goroutines post, timers, and a tick every
// TickInterval with per-tick hooks. A panic in anything it runs is recovered
// and logged, and the loop goes on.
type Loop struct {
	clock    Clock
	tick     time.Duration
	log      *slog.Logger
	check    bool
	onMisuse func(op string)
	inbox    chan command
	state    atomic.Int32
	owner    atomic.Uint64 // goroutine in Run or Step; 0 when none
	done     chan struct{} // closed when Run returns

	// Loop-only.
	now       time.Time // time of the running event
	inEvent   bool
	timers    timerHeap
	timerByID map[TimerID]*timer
	lastTimer TimerID
	seq       uint64
	hooks     []tickHook
	endHooks  []tickHook
	lastHook  HookID
	nextTick  time.Time
}

// NewLoop returns a loop ready to Run. The first tick is due one interval
// after NewLoop, or after Run when Run starts later than that.
func NewLoop(opts LoopOptions) *Loop {
	if opts.Clock == nil {
		opts.Clock = RealClock()
	}
	if opts.TickInterval <= 0 {
		opts.TickInterval = DefaultTickInterval
	}
	if opts.InboxSize <= 0 {
		opts.InboxSize = DefaultInboxSize
	}
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.DiscardHandler)
	}
	l := &Loop{
		clock:     opts.Clock,
		tick:      opts.TickInterval,
		log:       opts.Logger,
		check:     opts.CheckOwner,
		onMisuse:  opts.OnMisuse,
		inbox:     make(chan command, opts.InboxSize),
		done:      make(chan struct{}),
		timerByID: make(map[TimerID]*timer),
	}
	l.now = l.clock.Now()
	l.nextTick = l.now.Add(l.tick)
	return l
}

// Run runs commands, timers and ticks until ctx is done, then returns nil.
// Commands still queued are dropped, and Calls waiting for them return
// ErrStopped. Run may be called once: later calls return ErrRunning or
// ErrStopped.
func (l *Loop) Run(ctx context.Context) error {
	if !l.state.CompareAndSwap(loopIdle, loopRunning) {
		return l.busyErr()
	}
	l.owner.Store(goid())
	defer func() {
		l.owner.Store(0)
		l.state.Store(loopStopped)
		close(l.done)
	}()

	now := l.clock.Now()
	if l.nextTick.Before(now) {
		l.nextTick = now.Add(l.tick)
	}
	wake := l.clock.NewTimer(l.nextWake().Sub(now))
	defer wake.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case c := <-l.inbox:
			l.runCommand(c)
			l.runQueued(maxBatch - 1)
		case <-wake.C():
			l.runDue(l.clock.Now())
		}
		l.rearm(wake)
	}
}

// rearm sets wake for the next timer or tick. If the clock moved past it
// while rearming (a fake clock can jump at any moment), it runs what is due
// and tries again, so the loop never sleeps through due work.
func (l *Loop) rearm(wake Timer) {
	for {
		next := l.nextWake()
		wake.Reset(next.Sub(l.clock.Now()))
		now := l.clock.Now()
		if next.After(now) {
			return
		}
		l.runDue(now)
	}
}

// Step runs on the calling goroutine whatever is due now: timers and ticks
// up to the clock's time, then the commands queued so far, each preceded by
// whatever became due. Tests use it to drive the loop by hand with a
// FakeClock. It returns ErrRunning while Run is active and ErrStopped after
// Run returned.
func (l *Loop) Step() error {
	if !l.state.CompareAndSwap(loopIdle, loopStepping) {
		return l.busyErr()
	}
	l.owner.Store(goid())
	defer func() {
		l.owner.Store(0)
		l.state.Store(loopIdle)
	}()
	l.runDue(l.clock.Now())
	l.runQueued(len(l.inbox))
	return nil
}

func (l *Loop) busyErr() error {
	if l.state.Load() == loopStopped {
		return ErrStopped
	}
	return ErrRunning
}

// Done returns a channel that is closed when Run returns.
func (l *Loop) Done() <-chan struct{} { return l.done }

// Post queues fn to run on the loop and returns at once. It reports false,
// and fn never runs, when the inbox is full or the loop has stopped.
// Commands posted by one goroutine run in the order posted. A panic in fn is
// logged and the loop goes on. Safe for concurrent use.
func (l *Loop) Post(fn func()) bool { return l.post(command{fn: fn}) }

// PostGuarded is Post with a panic handler: if fn panics, the loop logs it
// and then calls onPanic on the loop with the recovered value, for example to
// close the session whose packet caused it. Safe for concurrent use.
func (l *Loop) PostGuarded(fn func(), onPanic func(v any)) bool {
	return l.post(command{fn: fn, onPanic: onPanic})
}

func (l *Loop) post(c command) bool {
	if l.state.Load() == loopStopped {
		return false
	}
	select {
	case l.inbox <- c:
		return true
	default:
		l.log.Warn("world: inbox full, command dropped")
		return false
	}
}

// Call runs fn on the loop and waits for it. It returns a *PanicError if fn
// panicked, ErrStopped if the loop stopped before running it, and ctx.Err()
// if ctx ended first (fn may then still run later). On the loop itself Call
// runs fn at once instead of waiting for itself. Safe for concurrent use.
func (l *Loop) Call(ctx context.Context, fn func()) error {
	if l.OnLoop() {
		if perr := l.call("command", fn); perr != nil {
			return perr
		}
		return nil
	}
	if l.state.Load() == loopStopped {
		return ErrStopped
	}
	done := make(chan error, 1)
	select {
	case l.inbox <- command{fn: fn, done: done}:
	case <-ctx.Done():
		return ctx.Err()
	case <-l.done:
		return ErrStopped
	}
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-l.done:
		select {
		case err := <-done:
			return err
		default:
			return ErrStopped
		}
	}
}

// OnLoop reports whether the caller runs on the loop: in Run's goroutine or
// inside Step. It costs about a microsecond; use it in assertions. Safe for
// concurrent use.
func (l *Loop) OnLoop() bool {
	o := l.owner.Load()
	return o != 0 && o == goid()
}

// owned reports whether a loop-only method may go on. With CheckOwner, a
// call from another goroutine while the loop runs is reported and refused.
func (l *Loop) owned(op string) bool {
	if !l.check || l.owner.Load() == 0 || l.OnLoop() {
		return true
	}
	if l.onMisuse != nil {
		l.onMisuse(op)
	} else {
		l.log.Error("world: loop-only method called off the loop", "op", op, "stack", string(debug.Stack()))
	}
	return false
}

// Clock returns the loop's clock. Safe for concurrent use.
func (l *Loop) Clock() Clock { return l.clock }

// TickInterval returns the time between ticks. Safe for concurrent use.
func (l *Loop) TickInterval() time.Duration { return l.tick }

// Now returns the loop's time: the scheduled time of the running timer or
// tick, or the start of the running command; outside of them, the clock's
// time. Timers count from it, so timer chains don't drift and fake-clock
// tests stay exact. Loop-only.
func (l *Loop) Now() time.Time {
	if l.inEvent {
		return l.now
	}
	return l.clock.Now()
}

// runQueued runs up to n queued commands without waiting for more.
func (l *Loop) runQueued(n int) {
	for range n {
		select {
		case c := <-l.inbox:
			l.runCommand(c)
		default:
			return
		}
	}
}

// runCommand runs whatever is due, then c.
func (l *Loop) runCommand(c command) {
	now := l.clock.Now()
	l.runDue(now)
	l.now = now
	l.inEvent = true
	perr := l.call("command", c.fn)
	if perr != nil && c.onPanic != nil {
		l.call("panic handler", func() { c.onPanic(perr.Value) })
	}
	l.inEvent = false
	if c.done != nil {
		if perr != nil {
			c.done <- perr
		} else {
			c.done <- nil
		}
	}
}

// runDue runs the timers and ticks due at or before now, in time order. A
// timer due at the same time as a tick runs first.
func (l *Loop) runDue(now time.Time) {
	if !l.nextTick.After(now) {
		if missed := int64(now.Sub(l.nextTick)/l.tick) + 1; missed > maxTickCatchUp {
			drop := missed - maxTickCatchUp
			l.nextTick = l.nextTick.Add(time.Duration(drop) * l.tick)
			l.log.Warn("world: loop fell behind, ticks dropped", "dropped", drop)
		}
	}
	for {
		tickDue := !l.nextTick.After(now)
		var t *timer
		if len(l.timers) > 0 && !l.timers[0].when.After(now) {
			t = l.timers[0]
		}
		switch {
		case t != nil && (!tickDue || !t.when.After(l.nextTick)):
			l.runTimer(t)
		case tickDue:
			at := l.nextTick
			l.nextTick = at.Add(l.tick)
			l.runTick(at)
		default:
			return
		}
	}
}

// nextWake returns when the next timer or tick is due.
func (l *Loop) nextWake() time.Time {
	if len(l.timers) > 0 && l.timers[0].when.Before(l.nextTick) {
		return l.timers[0].when
	}
	return l.nextTick
}

// call runs fn and recovers a panic, which it logs and returns.
func (l *Loop) call(what string, fn func()) (perr *PanicError) {
	defer func() {
		if v := recover(); v != nil {
			perr = &PanicError{Value: v, Stack: debug.Stack()}
			l.log.Error("world: "+what+" panicked", "panic", v, "stack", string(perr.Stack))
		}
	}()
	fn()
	return nil
}
