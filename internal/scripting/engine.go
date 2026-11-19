// Package scripting runs the Lua scripts of the game server.
//
// One goroutine (Run) owns the Lua state. Game code sends it events with
// Call, which waits for the result, or Dispatch, which does not. Scripts run
// in a sandbox: the base, string, table and math libraries only, a require
// that reads from the scripts folder, and a timeout per event.
//
// Reload builds a fresh state from the scripts folder and swaps it in only
// if every script loaded, so a broken script never replaces working ones.
package scripting

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Defaults for Options.
const (
	DefaultDir         = "scripts"
	DefaultTimeout     = 50 * time.Millisecond
	DefaultLoadTimeout = 2 * time.Second
)

// queueSize is the number of events that can wait for the script goroutine.
const queueSize = 1024

// Options configure an Engine.
type Options struct {
	// Dir is the scripts folder. Empty means DefaultDir.
	Dir string
	// Timeout bounds the handlers of one event. Zero means DefaultTimeout.
	Timeout time.Duration
	// LoadTimeout bounds running one script file while loading. Zero means
	// DefaultLoadTimeout.
	LoadTimeout time.Duration
	// Include, if set, picks the scripts that run, by slash-separated path
	// relative to Dir (for example "events/character_create.lua"). Files in
	// lib/ stay available to require. Tests use it to load one script.
	Include func(path string) bool
	// Logger gets engine messages and the output of the log module. Nil
	// discards them.
	Logger *slog.Logger
}

func (o Options) withDefaults() Options {
	if o.Dir == "" {
		o.Dir = DefaultDir
	}
	if o.Timeout <= 0 {
		o.Timeout = DefaultTimeout
	}
	if o.LoadTimeout <= 0 {
		o.LoadTimeout = DefaultLoadTimeout
	}
	if o.Logger == nil {
		o.Logger = slog.New(slog.DiscardHandler)
	}
	return o
}

// job runs on the script goroutine. ctx is the context of Run.
type job func(ctx context.Context)

// Engine runs the scripts of one scripts folder. Its methods are safe for
// concurrent use.
type Engine struct {
	opts     Options
	jobs     chan job
	done     chan struct{} // closed when Run returns
	started  atomic.Bool
	reloadMu sync.Mutex // one Reload at a time

	// vm belongs to New until Run starts, then to the Run goroutine.
	vm *vm
}

// New loads every script in opts.Dir. If any script fails, New returns a
// *LoadError listing all of them. Call Run to start handling events.
func New(ctx context.Context, opts Options) (*Engine, error) {
	opts = opts.withDefaults()
	v, err := load(ctx, opts)
	if err != nil {
		return nil, err
	}
	opts.Logger.Info("scripts loaded", "dir", opts.Dir, "files", v.files)
	return &Engine{
		opts: opts,
		jobs: make(chan job, queueSize),
		done: make(chan struct{}),
		vm:   v,
	}, nil
}

// Run handles events until ctx is done, then closes the Lua state. Calls
// waiting at that point return ErrStopped. Run may be called only once.
func (e *Engine) Run(ctx context.Context) error {
	if !e.started.CompareAndSwap(false, true) {
		return errors.New("scripting: Run called twice")
	}
	defer close(e.done)
	defer func() { e.vm.close() }()
	for {
		select {
		case <-ctx.Done():
			return nil
		case j := <-e.jobs:
			j(ctx)
		}
	}
}

// enqueue hands j to the script goroutine.
func (e *Engine) enqueue(ctx context.Context, j job) error {
	select {
	case <-e.done:
		return ErrStopped
	default:
	}
	select {
	case e.jobs <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return ErrStopped
	}
}

// Call runs the handlers of ev and returns what they decided. Handler
// failures (Lua errors, timeouts) are logged and reported in Result.Err;
// the error return is only for events that could not run: ctx done or
// ErrStopped. Cancelling ctx also stops a running handler.
func (e *Engine) Call(ctx context.Context, ev Event) (Result, error) {
	reply := make(chan Result, 1)
	err := e.enqueue(ctx, func(context.Context) {
		if ctx.Err() == nil {
			reply <- e.vm.fire(ctx, ev, e.opts.Timeout)
		}
	})
	if err != nil {
		return Result{}, err
	}
	select {
	case r := <-reply:
		return r, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-e.done:
		select {
		case r := <-reply:
			return r, nil
		default:
			return Result{}, ErrStopped
		}
	}
}

// Dispatch queues ev and returns at once; handler results are dropped and
// failures logged. It reports false if the queue is full or the engine
// stopped.
func (e *Engine) Dispatch(ev Event) bool {
	select {
	case <-e.done:
		return false
	default:
	}
	select {
	case e.jobs <- func(ctx context.Context) { e.vm.fire(ctx, ev, e.opts.Timeout) }:
		return true
	default:
		e.opts.Logger.Warn("script event dropped, queue full", "event", ev.Name)
		return false
	}
}

// Reload loads every script again into a fresh Lua state, off the script
// goroutine. If all of them load, the new state replaces the old one after
// the events queued before it; otherwise the old state stays and Reload
// returns a *LoadError. It returns once the new state is in use.
func (e *Engine) Reload(ctx context.Context) error {
	e.reloadMu.Lock()
	defer e.reloadMu.Unlock()

	next, err := load(ctx, e.opts)
	if err != nil {
		return err
	}
	files := next.files
	swapped := make(chan struct{})
	err = e.enqueue(ctx, func(context.Context) {
		old := e.vm
		e.vm = next
		old.close()
		close(swapped)
	})
	if err != nil {
		next.close()
		return err
	}
	select {
	case <-swapped:
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return ErrStopped
	}
	e.opts.Logger.Info("scripts reloaded", "dir", e.opts.Dir, "files", files)
	return nil
}
